import fcntl
import os
import tempfile
import threading
import time
import unittest
from pathlib import Path
from unittest.mock import patch

from cleanup_images import cleanup_images, delete_if_unchanged


class CleanupImagesTest(unittest.TestCase):
    def setUp(self):
        self.temp_dir = tempfile.TemporaryDirectory()
        self.image_dir = Path(self.temp_dir.name)
        self.image_directory_lock_file = self.image_dir / "images.lock"
        self.now = time.time()

    def tearDown(self):
        self.temp_dir.cleanup()

    def create_file(self, name, age_minutes, content=b"image"):
        path = self.image_dir / name
        path.write_bytes(content)
        timestamp = self.now - age_minutes * 60
        os.utime(str(path), (timestamp, timestamp))
        return path

    def test_deletes_only_expired_ordinary_jpegs(self):
        expired = self.create_file("2026-10-01 10:00:00.jpg", 361, b"old")
        recent = self.create_file("2026-10-05 10:00:00.jpg", 10)
        hero = self.create_file("2026-10-01 10:00:00-hero.jpg", 1000)
        decided = self.create_file("2026-10-01 10:00:00-decided.jpg", 1000)
        text_file = self.create_file("notes.txt", 1000)
        target = self.create_file("target.jpg", 10)
        symlink = self.image_dir / "linked.jpg"
        symlink.symlink_to(target)

        result = cleanup_images(
            self.image_dir,
            360,
            now=self.now,
            image_directory_lock_file=self.image_directory_lock_file,
        )

        self.assertEqual(result, (1, 3, 1, 3, 0, []))
        self.assertFalse(expired.exists())
        for path in (recent, hero, decided, text_file, target, symlink):
            self.assertTrue(path.exists())

    def test_dry_run_reports_without_deleting(self):
        expired = self.create_file("old.jpg", 361, b"12345")

        result = cleanup_images(
            self.image_dir,
            360,
            dry_run=True,
            now=self.now,
            image_directory_lock_file=self.image_directory_lock_file,
        )

        self.assertEqual(result, (1, 5, 0, 0, 0, []))
        self.assertTrue(expired.exists())

    def test_skips_file_replaced_after_scan(self):
        path = self.create_file("old.jpg", 361, b"old")
        observed_stat = path.lstat()
        path.unlink()
        path.write_bytes(b"replacement")
        os.utime(str(path), (self.now, self.now))

        deleted = delete_if_unchanged(
            path, observed_stat, self.now - 360 * 60
        )

        self.assertFalse(deleted)
        self.assertEqual(path.read_bytes(), b"replacement")

    def test_skips_file_whose_mtime_changed_after_scan(self):
        path = self.create_file("old.jpg", 361, b"old")
        observed_stat = path.lstat()
        os.utime(str(path), (self.now, self.now))

        deleted = delete_if_unchanged(
            path, observed_stat, self.now - 360 * 60
        )

        self.assertFalse(deleted)
        self.assertTrue(path.exists())

    def test_cleanup_counts_file_changed_before_delete(self):
        path = self.create_file("old.jpg", 361, b"old")

        with patch("cleanup_images.delete_if_unchanged", return_value=False):
            result = cleanup_images(
                self.image_dir,
                360,
                now=self.now,
                image_directory_lock_file=self.image_directory_lock_file,
            )

        self.assertEqual(result, (1, 3, 0, 0, 1, []))
        self.assertTrue(path.exists())

    def test_rejects_non_positive_retention(self):
        with self.assertRaises(ValueError):
            cleanup_images(self.image_dir, 0, now=self.now)

    def test_writer_waits_between_final_check_and_unlink(self):
        path = self.create_file("old.jpg", 361, b"old")
        checked = threading.Event()
        writer_attempted = threading.Event()
        writer_finished = threading.Event()
        writer_errors = []

        def writer():
            if not checked.wait(2):
                writer_errors.append("cleanup never reached the final identity check")
                return
            with self.image_directory_lock_file.open("a") as lock_handle:
                writer_attempted.set()
                fcntl.flock(lock_handle, fcntl.LOCK_EX)
                path.write_bytes(b"replacement")
                os.utime(str(path), (self.now, self.now))
                fcntl.flock(lock_handle, fcntl.LOCK_UN)
            writer_finished.set()

        writer_thread = threading.Thread(target=writer)
        writer_thread.start()

        def before_unlink():
            checked.set()
            self.assertTrue(writer_attempted.wait(2))
            self.assertFalse(writer_finished.is_set())

        result = cleanup_images(
            self.image_dir,
            360,
            now=self.now,
            image_directory_lock_file=self.image_directory_lock_file,
            before_unlink=before_unlink,
        )
        writer_thread.join(2)

        self.assertFalse(writer_thread.is_alive())
        self.assertEqual(writer_errors, [])
        self.assertEqual(result, (1, 3, 1, 3, 0, []))
        self.assertEqual(path.read_bytes(), b"replacement")


if __name__ == "__main__":
    unittest.main()
