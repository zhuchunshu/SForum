//go:build !linux

package processmemory

func readProcessPSS(int32) (uint64, bool) {
	return 0, false
}
