//go:build linux

package processmemory

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"runtime"
	"strconv"
	"strings"
	"sync"
)

// procSampler reads Linux procfs directly. Release containers intentionally do
// not ship procps, so invoking `ps` would make every resource card unavailable.
type procSampler struct {
	procRoot string
	numCPU   int

	mu               sync.Mutex
	lastSystemTicks  uint64
	lastProcessTicks map[int]uint64
}

func newOSSampler() Sampler {
	return &procSampler{procRoot: "/proc", numCPU: runtime.NumCPU()}
}

func (s *procSampler) List() ([]Sample, error) {
	if s == nil {
		return nil, errors.New("proc process list: nil sampler")
	}
	root := strings.TrimSpace(s.procRoot)
	if root == "" {
		root = "/proc"
	}
	entries, err := os.ReadDir(root)
	if err != nil {
		return nil, fmt.Errorf("proc process list: %w", err)
	}

	currentSystemTicks, err := readSystemCPUTicks(root)
	if err != nil {
		return nil, err
	}
	selfPID := os.Getpid()
	samples := make([]Sample, 0, 16)
	processTicks := make(map[int]uint64)
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		pid, parseErr := strconv.Atoi(entry.Name())
		if parseErr != nil || pid <= 0 {
			continue
		}
		sample, ticks, readErr := readProcProcess(root, pid)
		if readErr != nil {
			// Processes may exit between ReadDir and file reads.
			continue
		}
		if pid != selfPID && !IsSforumWorkerCommand(sample.Command) && !IsBackendPluginCommand(sample.Command) {
			continue
		}
		processTicks[pid] = ticks
		sample.PSSBytes, _ = readProcessPSSAt(root, pid)
		samples = append(samples, sample)
	}
	if len(samples) == 0 {
		return nil, errors.New("proc process list: no SForum processes")
	}

	s.mu.Lock()
	previousSystemTicks := s.lastSystemTicks
	previousProcessTicks := s.lastProcessTicks
	s.lastSystemTicks = currentSystemTicks
	s.lastProcessTicks = processTicks
	s.mu.Unlock()

	if previousSystemTicks > 0 && currentSystemTicks > previousSystemTicks {
		systemDelta := currentSystemTicks - previousSystemTicks
		cpuCount := s.numCPU
		if cpuCount <= 0 {
			cpuCount = 1
		}
		for index := range samples {
			currentTicks := processTicks[samples[index].PID]
			previousTicks, found := previousProcessTicks[samples[index].PID]
			if !found || currentTicks < previousTicks {
				continue
			}
			samples[index].CPUPercent = float64(currentTicks-previousTicks) * float64(cpuCount) * 100 / float64(systemDelta)
		}
	}
	return samples, nil
}

func readSystemCPUTicks(procRoot string) (uint64, error) {
	data, err := os.ReadFile(procRoot + "/stat")
	if err != nil {
		return 0, fmt.Errorf("proc cpu stat: %w", err)
	}
	line, _, _ := bytes.Cut(data, []byte{'\n'})
	fields := strings.Fields(string(line))
	if len(fields) < 2 || fields[0] != "cpu" {
		return 0, errors.New("proc cpu stat: malformed aggregate row")
	}
	var total uint64
	for _, field := range fields[1:] {
		value, parseErr := strconv.ParseUint(field, 10, 64)
		if parseErr != nil {
			return 0, fmt.Errorf("proc cpu stat: %w", parseErr)
		}
		total += value
	}
	return total, nil
}

func readProcProcess(procRoot string, pid int) (Sample, uint64, error) {
	stat, err := os.ReadFile(fmt.Sprintf("%s/%d/stat", procRoot, pid))
	if err != nil {
		return Sample{}, 0, err
	}
	closeParen := bytes.LastIndexByte(stat, ')')
	openParen := bytes.IndexByte(stat, '(')
	if openParen < 0 || closeParen <= openParen || closeParen+2 >= len(stat) {
		return Sample{}, 0, errors.New("malformed process stat")
	}
	rest := strings.Fields(string(stat[closeParen+2:]))
	// rest starts at field 3 (state): PPID=4, utime=14, stime=15.
	if len(rest) < 13 {
		return Sample{}, 0, errors.New("short process stat")
	}
	ppid, err := strconv.Atoi(rest[1])
	if err != nil || ppid < 0 {
		return Sample{}, 0, errors.New("invalid process parent")
	}
	userTicks, err := strconv.ParseUint(rest[11], 10, 64)
	if err != nil {
		return Sample{}, 0, errors.New("invalid process user ticks")
	}
	systemTicks, err := strconv.ParseUint(rest[12], 10, 64)
	if err != nil {
		return Sample{}, 0, errors.New("invalid process system ticks")
	}
	rssBytes, err := readProcRSS(procRoot, pid)
	if err != nil {
		return Sample{}, 0, err
	}
	command := readProcCommand(procRoot, pid)
	if command == "" {
		command = string(stat[openParen+1 : closeParen])
	}
	return Sample{PID: pid, PPID: ppid, RSSBytes: rssBytes, Command: command}, userTicks + systemTicks, nil
}

func readProcRSS(procRoot string, pid int) (uint64, error) {
	data, err := os.ReadFile(fmt.Sprintf("%s/%d/statm", procRoot, pid))
	if err != nil {
		return 0, err
	}
	fields := strings.Fields(string(data))
	if len(fields) < 2 {
		return 0, errors.New("short process statm")
	}
	pages, err := strconv.ParseUint(fields[1], 10, 64)
	if err != nil {
		return 0, err
	}
	return pages * uint64(os.Getpagesize()), nil
}

func readProcCommand(procRoot string, pid int) string {
	data, err := os.ReadFile(fmt.Sprintf("%s/%d/cmdline", procRoot, pid))
	if err != nil {
		return ""
	}
	fields := bytes.FieldsFunc(data, func(r rune) bool { return r == 0 })
	parts := make([]string, 0, len(fields))
	for _, field := range fields {
		parts = append(parts, string(field))
	}
	return strings.Join(parts, " ")
}
