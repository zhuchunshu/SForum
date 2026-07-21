//go:build unix

package adminoverview

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
)

// osProcessSampler 通过 ps 采集 PID/PPID/RSS/命令行（macOS 与 Linux 均可用，无 cgo）。
// admin overview 为低频路径，可接受一次短 ps。
type osProcessSampler struct{}

func (osProcessSampler) List() ([]ProcessSample, error) {
	// rss 单位为 KiB（POSIX ps 惯例）。
	cmd := exec.Command("ps", "-axo", "pid=,ppid=,rss=,command=")
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("ps process list: %w", err)
	}
	return parsePSList(out)
}

func parsePSList(out []byte) ([]ProcessSample, error) {
	lines := bytes.Split(out, []byte{'\n'})
	samples := make([]ProcessSample, 0, len(lines))
	for _, line := range lines {
		line = bytes.TrimSpace(line)
		if len(line) == 0 {
			continue
		}
		// pid ppid rss command... — 前三个字段固定，其余为命令。
		fields := strings.Fields(string(line))
		if len(fields) < 4 {
			continue
		}
		pid, err := strconv.Atoi(fields[0])
		if err != nil || pid <= 0 {
			continue
		}
		ppid, err := strconv.Atoi(fields[1])
		if err != nil || ppid < 0 {
			continue
		}
		rssKiB, err := strconv.ParseUint(fields[2], 10, 64)
		if err != nil {
			continue
		}
		command := strings.Join(fields[3:], " ")
		samples = append(samples, ProcessSample{
			PID:      pid,
			PPID:     ppid,
			RSSBytes: rssKiB * 1024,
			Command:  command,
		})
	}
	if len(samples) == 0 {
		return nil, fmt.Errorf("ps process list: empty")
	}
	return samples, nil
}

// readSelfRSSFallback 在 ps 不可用时尝试读自身 RSS（Linux /proc）。
func readSelfRSSFallback() (uint64, bool) {
	data, err := os.ReadFile("/proc/self/statm")
	if err != nil {
		return 0, false
	}
	fields := strings.Fields(string(data))
	if len(fields) < 2 {
		return 0, false
	}
	pages, err := strconv.ParseUint(fields[1], 10, 64)
	if err != nil {
		return 0, false
	}
	pageSize := uint64(os.Getpagesize())
	return pages * pageSize, true
}
