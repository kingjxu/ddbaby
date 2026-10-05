package service

import (
	"os"
	"path/filepath"
	"syscall"
	"testing"
	"time"
)

func TestWriteImageToFileWithLockWaitsForDirectoryLock(t *testing.T) {
	tempDir := t.TempDir()
	lockPath := filepath.Join(tempDir, "images.lock")
	imagePath := filepath.Join(tempDir, "new.jpg")

	lockFile, err := os.OpenFile(lockPath, os.O_CREATE|os.O_RDWR, 0644)
	if err != nil {
		t.Fatal(err)
	}
	defer lockFile.Close()
	if err := syscall.Flock(int(lockFile.Fd()), syscall.LOCK_EX); err != nil {
		t.Fatal(err)
	}

	done := make(chan error, 1)
	go func() {
		done <- writeImageToFileWithLockAt(lockPath, []byte("new image"), imagePath)
	}()

	select {
	case err := <-done:
		t.Fatalf("writer completed before the directory lock was released: %v", err)
	case <-time.After(100 * time.Millisecond):
	}

	if err := syscall.Flock(int(lockFile.Fd()), syscall.LOCK_UN); err != nil {
		t.Fatal(err)
	}
	select {
	case err := <-done:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("writer did not continue after the directory lock was released")
	}

	data, err := os.ReadFile(imagePath)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "new image" {
		t.Fatalf("image content = %q, want %q", data, "new image")
	}
}
