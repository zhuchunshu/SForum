//go:build unix

package processmemory

import (
	"bytes"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

// ParsePSList 解析 ps -axo 输出；导出让 adminoverview 单测复用。
func ParsePSList(out []byte) ([]Sample, error) {
	lines := bytes.Split(out, []byte{'\n'})
	samples := make([]Sample, 0, len(lines))
	capturedAt := time.Now().UTC()
	for _, line := range lines {
		line = bytes.TrimSpace(line)
		if len(line) == 0 {
			continue
		}
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
		commandStart := 3
		cpuPercent := float64(0)
		if len(fields) >= 5 {
			parsedCPU, cpuErr := strconv.ParseFloat(fields[3], 64)
			if cpuErr == nil && parsedCPU >= 0 {
				cpuPercent = parsedCPU
				commandStart = 4
			}
		}
		command := strings.Join(fields[commandStart:], " ")
		samples = append(samples, Sample{
			PID:        pid,
			PPID:       ppid,
			RSSBytes:   rssKiB * 1024,
			CPUPercent: cpuPercent,
			Command:    command,
			CapturedAt: capturedAt,
		})
	}
	if len(samples) == 0 {
		return nil, fmt.Errorf("ps process list: empty")
	}
	return samples, nil
}

// ReadSelfRSSFallback 在 ps 不可用时尝试读自身 RSS（Linux /proc）。
func ReadSelfRSSFallback() (uint64, bool) {
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
