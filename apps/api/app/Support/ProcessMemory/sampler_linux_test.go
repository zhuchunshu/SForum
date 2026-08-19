//go:build linux

package processmemory

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

func TestProcSamplerWorksWithoutPSAndCalculatesCPUFromFrames(t *testing.T) {
	root := t.TempDir()
	selfPID := os.Getpid()
	pluginPID := selfPID + 1
	writeProcCPUStat(t, root, 1000)
	writeProcProcess(t, root, selfPID, 1, 10, 5, 25, "/usr/local/bin/sforum-api")
	writeProcProcess(t, root, pluginPID, selfPID, 4, 1, 3, "/app/storage/extensions/sforum.smtp/1.1.1/digest/backend/plugin")

	sampler := &procSampler{procRoot: root, numCPU: 4}
	first, err := sampler.List()
	if err != nil {
		t.Fatal(err)
	}
	if len(first) != 2 || first[0].CPUPercent != 0 || first[1].CPUPercent != 0 {
		t.Fatalf("unexpected first proc frame: %#v", first)
	}
	byPID := samplesByPID(first)
	if byPID[selfPID].RSSBytes != 25*uint64(os.Getpagesize()) {
		t.Fatalf("self RSS=%d", byPID[selfPID].RSSBytes)
	}
	if byPID[pluginPID].PPID != selfPID || !IsBackendPluginCommand(byPID[pluginPID].Command) {
		t.Fatalf("plugin sample=%#v", byPID[pluginPID])
	}

	writeProcCPUStat(t, root, 1100)
	writeProcProcess(t, root, selfPID, 1, 17, 8, 25, "/usr/local/bin/sforum-api")
	writeProcProcess(t, root, pluginPID, selfPID, 6, 2, 3, "/app/storage/extensions/sforum.smtp/1.1.1/digest/backend/plugin")
	second, err := sampler.List()
	if err != nil {
		t.Fatal(err)
	}
	byPID = samplesByPID(second)
	if byPID[selfPID].CPUPercent != 40 {
		t.Fatalf("self CPU=%.2f want 40", byPID[selfPID].CPUPercent)
	}
	if byPID[pluginPID].CPUPercent != 12 {
		t.Fatalf("plugin CPU=%.2f want 12", byPID[pluginPID].CPUPercent)
	}
	usage := AggregateRuntimeUsage(selfPID, second)
	if usage.APIMemoryBytes == 0 || usage.PluginMemoryBytes == 0 || usage.TotalCPUPercent != 52 {
		t.Fatalf("unexpected proc usage: %#v", usage)
	}
}

func TestReadProcProcessHandlesSpacesAndParenthesesInComm(t *testing.T) {
	root := t.TempDir()
	pid := 42
	dir := filepath.Join(root, "42")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	stat := "42 (plugin worker (v2)) S 7 0 0 0 0 0 0 0 0 0 9 3 0 0 0 0 0 0 0 0 0\n"
	if err := os.WriteFile(filepath.Join(dir, "stat"), []byte(stat), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "statm"), []byte("20 4\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	sample, ticks, err := readProcProcess(root, pid)
	if err != nil {
		t.Fatal(err)
	}
	if sample.PPID != 7 || sample.Command != "plugin worker (v2)" || ticks != 12 {
		t.Fatalf("sample=%#v ticks=%d", sample, ticks)
	}
}

func TestReadProcessMemoryDetailsReadsPSSAndAnonHugePages(t *testing.T) {
	root := t.TempDir()
	dir := filepath.Join(root, "42")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	body := "Rss: 4096 kB\nPss: 3072 kB\nAnonHugePages: 2048 kB\n"
	if err := os.WriteFile(filepath.Join(dir, "smaps_rollup"), []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	pss, anonHugePages, ok := readProcessMemoryDetailsAt(root, 42)
	if !ok || pss != 3072*1024 || anonHugePages != 2048*1024 {
		t.Fatalf("pss=%d anonHugePages=%d ok=%t", pss, anonHugePages, ok)
	}
}

func writeProcCPUStat(t *testing.T, root string, total uint64) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(root, "stat"), []byte(fmt.Sprintf("cpu %d 0 0 0 0 0 0 0 0 0\n", total)), 0o644); err != nil {
		t.Fatal(err)
	}
}

func writeProcProcess(t *testing.T, root string, pid, ppid int, userTicks, systemTicks, residentPages uint64, command string) {
	t.Helper()
	dir := filepath.Join(root, fmt.Sprint(pid))
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	stat := fmt.Sprintf("%d (sforum process) S %d 0 0 0 0 0 0 0 0 0 %d %d 0 0 0 0 0 0 0 0 0\n", pid, ppid, userTicks, systemTicks)
	if err := os.WriteFile(filepath.Join(dir, "stat"), []byte(stat), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "statm"), []byte(fmt.Sprintf("100 %d\n", residentPages)), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "cmdline"), append([]byte(command), 0), 0o644); err != nil {
		t.Fatal(err)
	}
}

func samplesByPID(samples []Sample) map[int]Sample {
	out := make(map[int]Sample, len(samples))
	for _, sample := range samples {
		out[sample.PID] = sample
	}
	return out
}
