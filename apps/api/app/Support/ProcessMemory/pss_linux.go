//go:build linux

package processmemory

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"
)

func readProcessPSS(pid int32) (uint64, bool) {
	file, err := os.Open(fmt.Sprintf("/proc/%d/smaps_rollup", pid))
	if err != nil {
		return 0, false
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if !strings.HasPrefix(line, "Pss:") {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) < 2 {
			return 0, false
		}
		pssKiB, parseErr := strconv.ParseUint(fields[1], 10, 64)
		return pssKiB * 1024, parseErr == nil && pssKiB > 0
	}
	return 0, false
}
