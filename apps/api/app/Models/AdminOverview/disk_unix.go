//go:build unix

package adminoverview

import "golang.org/x/sys/unix"

func sampleDiskUsage(path string) (DiskRuntimeStats, bool) {
	if path == "" {
		return DiskRuntimeStats{}, false
	}

	var stat unix.Statfs_t
	if err := unix.Statfs(path, &stat); err != nil {
		return DiskRuntimeStats{}, false
	}

	blockSize := uint64(stat.Bsize)
	totalBytes := uint64(stat.Blocks) * blockSize
	freeBytes := uint64(stat.Bavail) * blockSize
	if totalBytes == 0 || freeBytes > totalBytes {
		return DiskRuntimeStats{}, false
	}

	usedBytes := totalBytes - freeBytes
	return DiskRuntimeStats{
		TotalBytes:  totalBytes,
		UsedBytes:   usedBytes,
		FreeBytes:   freeBytes,
		UsedPercent: float64(usedBytes) / float64(totalBytes) * 100,
	}, true
}
