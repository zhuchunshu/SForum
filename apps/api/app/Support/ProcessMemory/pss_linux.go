//go:build linux

package processmemory

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"
)

func readProcessPSS(pid int) (uint64, bool) {
	return readProcessPSSAt("/proc", pid)
}

func readProcessPSSAt(procRoot string, pid int) (uint64, bool) {
	pssBytes, _, ok := readProcessMemoryDetailsAt(procRoot, pid)
	return pssBytes, ok
}

func readProcessMemoryDetailsAt(procRoot string, pid int) (pssBytes, anonHugePagesBytes uint64, ok bool) {
	if pid <= 0 {
		return 0, 0, false
	}
	file, err := os.Open(fmt.Sprintf("%s/%d/smaps_rollup", strings.TrimRight(procRoot, "/"), pid))
	if err != nil {
		return 0, 0, false
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}
		valueKiB, parseErr := strconv.ParseUint(fields[1], 10, 64)
		if parseErr != nil {
			continue
		}
		switch fields[0] {
		case "Pss:":
			pssBytes = valueKiB * 1024
		case "AnonHugePages:":
			anonHugePagesBytes = valueKiB * 1024
		}
	}
	return pssBytes, anonHugePagesBytes, scanner.Err() == nil && pssBytes > 0
}
