//go:build !linux

package processmemory

func readProcessPSS(int) (uint64, bool) {
	return 0, false
}
