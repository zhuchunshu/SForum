//go:build unix && !linux

package processmemory

import (
	"fmt"
	"os"
	"os/exec"
)

// psSampler is the portable Unix implementation used outside Linux.
// Linux production images use procSampler and do not require procps.
type psSampler struct{}

func newOSSampler() Sampler {
	return psSampler{}
}

func (psSampler) List() ([]Sample, error) {
	cmd := exec.Command("ps", "-axo", "pid=,ppid=,rss=,%cpu=,command=")
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("ps process list: %w", err)
	}
	samples, err := ParsePSList(out)
	if err != nil {
		return nil, err
	}
	selfPID := os.Getpid()
	runtimeSamples := make([]Sample, 0, 16)
	for _, sample := range samples {
		if sample.PID != selfPID && !IsSforumWorkerCommand(sample.Command) && !IsBackendPluginCommand(sample.Command) {
			continue
		}
		sample.PSSBytes, _ = readProcessPSS(sample.PID)
		runtimeSamples = append(runtimeSamples, sample)
	}
	return runtimeSamples, nil
}
