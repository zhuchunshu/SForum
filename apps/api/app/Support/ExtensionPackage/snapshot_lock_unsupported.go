//go:build !darwin && !dragonfly && !freebsd && !illumos && !linux && !netbsd && !openbsd && !windows

package extensionpackage

import "os"

func lockUploadedSnapshotFile(_ *os.File) error {
	return ErrSnapshotLockUnsupported
}

func unlockUploadedSnapshotFile(_ *os.File) error {
	return ErrSnapshotLockUnsupported
}

func tryLockUploadedSnapshotFile(_ *os.File) (bool, error) {
	return false, ErrSnapshotLockUnsupported
}
