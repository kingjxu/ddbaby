package service

import (
	"fmt"
	"os"
	"syscall"

	"github.com/kingjxu/ddbaby/util"
)

const imageDirectoryLockFile = "/run/lock/ddbaby-images.lock"

func writeImageToFileWithLock(data []byte, filePath string) error {
	return writeImageToFileWithLockAt(imageDirectoryLockFile, data, filePath)
}

func writeImageToFileWithLockAt(lockPath string, data []byte, filePath string) error {
	lockFile, err := os.OpenFile(lockPath, os.O_CREATE|os.O_RDWR, 0644)
	if err != nil {
		return fmt.Errorf("open image directory lock: %w", err)
	}
	defer lockFile.Close()

	if err := syscall.Flock(int(lockFile.Fd()), syscall.LOCK_EX); err != nil {
		return fmt.Errorf("lock image directory: %w", err)
	}
	defer syscall.Flock(int(lockFile.Fd()), syscall.LOCK_UN)

	if err := util.WriteImageToFile(data, filePath); err != nil {
		return fmt.Errorf("write image: %w", err)
	}
	return nil
}
