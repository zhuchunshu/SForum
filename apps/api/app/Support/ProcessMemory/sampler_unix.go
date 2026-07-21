//go:build unix

package processmemory

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
)

// osSampler 通过 ps 采集 PID/PPID/RSS/命令行（macOS 与 Linux 均可用，无 cgo）。
type osSampler struct{}

func (osSampler) List() ([]Sample, error) {
	// rss 单位为 KiB（POSIX ps 惯例）。
	cmd := exec.Command("ps", "-axo", "pid=,ppid=,rss=,command=")
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("ps process list: %w", err)
	}
	return ParsePSList(out)
}

// ParsePSList 解析 ps -axo 输出；导出让 adminoverview 单测复用。
func ParsePSList(out []byte) ([]Sample, error) {
	lines := bytes.Split(out, []byte{'\n'})
	samples := make([]Sample, 0, len(lines))
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
		command := strings.Join(fields[3:], " ")
		samples = append(samples, Sample{
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
