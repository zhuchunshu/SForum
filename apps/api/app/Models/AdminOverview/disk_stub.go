//go:build !unix

package adminoverview

func sampleDiskUsage(string) (DiskRuntimeStats, bool) {
	return DiskRuntimeStats{}, false
}
