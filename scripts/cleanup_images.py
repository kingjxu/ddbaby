#!/usr/bin/env python3
"""Delete expired ordinary screenshots while retaining evidence images."""

import argparse
import fcntl
import os
import stat
import sys
import time
from pathlib import Path


DEFAULT_IMAGE_DIR = Path("/usr/local/webserver/images")
DEFAULT_RETENTION_MINUTES = 360
DEFAULT_LOCK_FILE = Path("/run/lock/ddbaby-image-cleanup.lock")
RETAINED_SUFFIXES = ("-hero.jpg", "-decided.jpg")


def file_identity(file_stat):
    return (
        file_stat.st_dev,
        file_stat.st_ino,
        file_stat.st_size,
        file_stat.st_mtime_ns,
    )


def delete_if_unchanged(path, observed_stat, cutoff):
    """Delete path only when it is still the expired regular file we inspected."""
    current_stat = path.lstat()
    if not stat.S_ISREG(current_stat.st_mode):
        return False
    if current_stat.st_mtime >= cutoff:
        return False
    if file_identity(current_stat) != file_identity(observed_stat):
        return False
    path.unlink()
    return True


def cleanup_images(image_dir, retention_minutes, dry_run=False, now=None):
    """Return the number and bytes of expired ordinary screenshots matched."""
    if retention_minutes <= 0:
        raise ValueError("retention_minutes must be positive")

    image_dir = Path(image_dir).resolve(strict=True)
    if not image_dir.is_dir():
        raise ValueError("image_dir must be a directory")

    cutoff = (time.time() if now is None else now) - retention_minutes * 60
    matched_count = 0
    matched_bytes = 0
    deleted_count = 0
    deleted_bytes = 0
    changed_count = 0
    errors = []

    for path in image_dir.iterdir():
        name = path.name
        if not name.endswith(".jpg") or name.endswith(RETAINED_SUFFIXES):
            continue

        try:
            file_stat = path.lstat()
            if not stat.S_ISREG(file_stat.st_mode) or file_stat.st_mtime >= cutoff:
                continue

            matched_count += 1
            matched_bytes += file_stat.st_size
            if dry_run:
                continue
            if delete_if_unchanged(path, file_stat, cutoff):
                deleted_count += 1
                deleted_bytes += file_stat.st_size
            else:
                changed_count += 1
        except FileNotFoundError:
            changed_count += 1
        except OSError as error:
            errors.append("{}: {}".format(name, error))

    return (
        matched_count,
        matched_bytes,
        deleted_count,
        deleted_bytes,
        changed_count,
        errors,
    )


def parse_args(argv=None):
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument(
        "--directory",
        type=Path,
        default=DEFAULT_IMAGE_DIR,
        help="screenshot directory (default: %(default)s)",
    )
    parser.add_argument(
        "--retention-minutes",
        type=int,
        default=DEFAULT_RETENTION_MINUTES,
        help="retain ordinary screenshots newer than this age (default: %(default)s)",
    )
    parser.add_argument("--dry-run", action="store_true", help="report without deleting")
    parser.add_argument(
        "--lock-file",
        type=Path,
        default=DEFAULT_LOCK_FILE,
        help=argparse.SUPPRESS,
    )
    return parser.parse_args(argv)


def main(argv=None):
    args = parse_args(argv)
    if args.retention_minutes <= 0:
        print("retention-minutes must be positive", file=sys.stderr)
        return 2

    args.lock_file.parent.mkdir(parents=True, exist_ok=True)
    with args.lock_file.open("a") as lock_handle:
        try:
            fcntl.flock(lock_handle, fcntl.LOCK_EX | fcntl.LOCK_NB)
        except BlockingIOError:
            print("cleanup already running; skipped")
            return 0

        try:
            (
                matched_count,
                matched_bytes,
                deleted_count,
                deleted_bytes,
                changed_count,
                errors,
            ) = cleanup_images(
                args.directory,
                args.retention_minutes,
                dry_run=args.dry_run,
            )
        except (OSError, ValueError) as error:
            print("cleanup failed: {}".format(error), file=sys.stderr)
            return 1

    print(
        "matched_files={} matched_bytes={} deleted_files={} deleted_bytes={} "
        "changed_files={} dry_run={} retention_minutes={} directory={}".format(
            matched_count,
            matched_bytes,
            deleted_count,
            deleted_bytes,
            changed_count,
            str(args.dry_run).lower(),
            args.retention_minutes,
            args.directory.resolve(),
        )
    )
    for error in errors:
        print("delete failed: {}".format(error), file=sys.stderr)
    return 1 if errors else 0


if __name__ == "__main__":
    sys.exit(main())
