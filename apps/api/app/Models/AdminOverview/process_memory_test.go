package adminoverview

import (
	"errors"
	"os"
	"testing"
	"time"
)

func TestIsBackendPluginCommand(t *testing.T) {
	allow := []string{
		"../../storage/extensions/sforum.smtp/1.1.0/abc/backend/plugin",
		"/Users/x/Code/SForum/storage/extensions/sforum.storage-fs/1.0.0/deadbeef/backend/plugin",
		"/app/extensions/builtin/plugins/sforum-smtp/backend/plugin",
	}
	for _, cmd := range allow {
		if !IsBackendPluginCommand(cmd) {
			t.Fatalf("expected plugin command allowed: %q", cmd)
		}
	}
	deny := []string{
		"/usr/bin/plugin",
		"sforum-api",
		"backend/other",
		"",
		"node backend/plugin.js", // 无 extensions 路径
	}
	for _, cmd := range deny {
		if IsBackendPluginCommand(cmd) {
			t.Fatalf("expected plugin command denied: %q", cmd)
		}
	}
}

func TestAggregateProcessMemoryIncludesOwnedPluginsOnly(t *testing.T) {
	selfPID := 100
	samples := []ProcessSample{
		{PID: selfPID, PPID: 10, RSSBytes: 160 * 1024 * 1024, Command: "/tmp/sforum-api"},
		// owned plugin
		{PID: 201, PPID: selfPID, RSSBytes: 18 * 1024 * 1024, Command: "storage/extensions/sforum.smtp/1.0.0/aaa/backend/plugin"},
		{PID: 202, PPID: selfPID, RSSBytes: 17 * 1024 * 1024, Command: "storage/extensions/sforum.storage-fs/1.0.0/bbb/backend/plugin"},
		// orphan PPID=1 — must NOT enter family
		{PID: 301, PPID: 1, RSSBytes: 16 * 1024 * 1024, Command: "storage/extensions/sforum.smtp/1.0.0/old/backend/plugin"},
		// other API's child
		{PID: 401, PPID: 999, RSSBytes: 20 * 1024 * 1024, Command: "storage/extensions/sforum.smtp/1.0.0/ccc/backend/plugin"},
		{PID: 999, PPID: 1, RSSBytes: 100 * 1024 * 1024, Command: "/other/sforum-api"},
		// non-plugin child
		{PID: 203, PPID: selfPID, RSSBytes: 5 * 1024 * 1024, Command: "some-helper"},
	}

	selfRSS, familyRSS, children, found := AggregateProcessMemory(selfPID, samples)
	if !found {
		t.Fatal("expected self found")
	}
	if selfRSS != 160*1024*1024 {
		t.Fatalf("self RSS=%d", selfRSS)
	}
	if children != 2 {
		t.Fatalf("plugin children=%d want 2", children)
	}
	wantFamily := uint64((160 + 18 + 17) * 1024 * 1024)
	if familyRSS != wantFamily {
		t.Fatalf("family RSS=%d want %d", familyRSS, wantFamily)
	}
	if familyRSS <= selfRSS {
		t.Fatal("family must be strictly greater than self when owned plugins exist")
	}
}

func TestAggregateProcessMemoryEqualsSelfWhenNoPlugins(t *testing.T) {
	selfPID := 50
	samples := []ProcessSample{
		{PID: selfPID, PPID: 1, RSSBytes: 90 * 1024 * 1024, Command: "./tmp/sforum-api"},
	}
	selfRSS, familyRSS, children, found := AggregateProcessMemory(selfPID, samples)
	if !found || children != 0 || selfRSS != familyRSS || selfRSS != 90*1024*1024 {
		t.Fatalf("self=%d family=%d children=%d found=%v", selfRSS, familyRSS, children, found)
	}
}

type fixedSampler struct {
	samples []ProcessSample
	err     error
}

func (f fixedSampler) List() ([]ProcessSample, error) {
	if f.err != nil {
		return nil, f.err
	}
	return f.samples, nil
}

func TestRuntimeCollectorPrimaryMemoryIsRSSNotSys(t *testing.T) {
	selfPID := os.Getpid()
	rss := uint64(123 * 1024 * 1024)
	pluginRSS := uint64(20 * 1024 * 1024)
	sampler := fixedSampler{samples: []ProcessSample{
		{PID: selfPID, PPID: 1, RSSBytes: rss, Command: "sforum-api"},
		{PID: selfPID + 1, PPID: selfPID, RSSBytes: pluginRSS, Command: "storage/extensions/sforum.smtp/1.0.0/x/backend/plugin"},
	}}

	collector := NewRuntimeCollector(time.Now().Add(-time.Minute), nil).WithProcessSampler(sampler)
	stats := collector.Snapshot()

	if stats.MemoryBytes != rss {
		t.Fatalf("memoryBytes (primary) want RSS %d, got %d", rss, stats.MemoryBytes)
	}
	if stats.SysBytes == 0 {
		t.Fatal("expected SysBytes from MemStats to remain available")
	}
	// Sys 通常 ≠ 我们注入的 RSS；主字段不得等于「仅 Sys」语义回退时才可能相等。
	if stats.MemoryBytes == stats.SysBytes && stats.SysBytes != rss {
		t.Fatalf("primary memory should not silently be Sys when RSS is available: memory=%d sys=%d", stats.MemoryBytes, stats.SysBytes)
	}
	if stats.HeapAllocBytes == 0 {
		t.Fatal("expected non-zero heap alloc")
	}
	if stats.FamilyMemoryBytes == nil {
		t.Fatal("expected family memory")
	}
	wantFamily := rss + pluginRSS
	if *stats.FamilyMemoryBytes != wantFamily {
		t.Fatalf("family=%d want %d", *stats.FamilyMemoryBytes, wantFamily)
	}
	if *stats.FamilyMemoryBytes < stats.MemoryBytes {
		t.Fatal("family must be >= parent RSS")
	}
	if stats.PluginChildCount != 1 {
		t.Fatalf("pluginChildCount=%d", stats.PluginChildCount)
	}
}

func TestRuntimeCollectorExposesResourceAndDiskSnapshots(t *testing.T) {
	selfPID := os.Getpid()
	sampler := fixedSampler{samples: []ProcessSample{
		{PID: selfPID, RSSBytes: 100, CPUPercent: 1.25, Command: "sforum-api"},
		{PID: selfPID + 1, RSSBytes: 20, CPUPercent: 0.5, Command: "sforum-worker"},
		{PID: selfPID + 2, PPID: selfPID, RSSBytes: 10, CPUPercent: 0.25, Command: "storage/extensions/demo/1.0.0/x/backend/plugin"},
		{PID: selfPID + 3, PPID: selfPID + 1, RSSBytes: 5, CPUPercent: 0.75, Command: "storage/extensions/demo/1.0.0/x/backend/plugin"},
	}}
	collector := NewRuntimeCollector(time.Now(), nil).
		WithProcessSampler(sampler).
		WithDiskSampler(func(string) (DiskRuntimeStats, bool) {
			return DiskRuntimeStats{TotalBytes: 1000, UsedBytes: 250, FreeBytes: 750, UsedPercent: 25}, true
		}).
		WithLoadSampler(func() (SystemLoadAverage, bool) {
			return SystemLoadAverage{OneMinute: 0.25, FiveMinutes: 0.5, FifteenMinutes: 0.75}, true
		})

	stats := collector.Snapshot()
	if stats.Resources == nil {
		t.Fatal("expected resource snapshot")
	}
	if stats.Resources.TotalMemoryBytes != 135 || stats.Resources.TotalCPUPercent != 2.75 {
		t.Fatalf("unexpected resources: %#v", stats.Resources)
	}
	if stats.Resources.PluginMemoryBytes != 15 || !stats.Resources.WorkerFound {
		t.Fatalf("unexpected plugin/worker resources: %#v", stats.Resources)
	}
	if stats.Disk == nil || stats.Disk.UsedPercent != 25 {
		t.Fatalf("unexpected disk stats: %#v", stats.Disk)
	}
	if stats.LoadAverage == nil || stats.LoadAverage.OneMinute != 0.25 || stats.LoadAverage.FifteenMinutes != 0.75 {
		t.Fatalf("unexpected load average: %#v", stats.LoadAverage)
	}
}

func TestRuntimeCollectorOmitsFamilyWhenSamplerFails(t *testing.T) {
	collector := NewRuntimeCollector(time.Now(), nil).WithProcessSampler(fixedSampler{err: errors.New("boom")})
	stats := collector.Snapshot()
	if stats.FamilyMemoryBytes != nil {
		t.Fatalf("family should be omitted on sampler failure, got %v", *stats.FamilyMemoryBytes)
	}
	if stats.SysBytes == 0 {
		t.Fatal("sysBytes must still be present")
	}
	// 回退路径：Linux /proc 或 Sys；主字段仍应非 0。
	if stats.MemoryBytes == 0 {
		t.Fatal("expected non-zero fallback memoryBytes")
	}
}
