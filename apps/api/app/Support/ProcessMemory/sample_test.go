package processmemory

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
		"node backend/plugin.js",
	}
	for _, cmd := range deny {
		if IsBackendPluginCommand(cmd) {
			t.Fatalf("expected plugin command denied: %q", cmd)
		}
	}
}

func TestAggregateFamilyIncludesOwnedPluginsOnly(t *testing.T) {
	selfPID := 100
	samples := []Sample{
		{PID: selfPID, PPID: 10, RSSBytes: 160 * 1024 * 1024, Command: "/tmp/sforum-api"},
		{PID: 201, PPID: selfPID, RSSBytes: 18 * 1024 * 1024, Command: "storage/extensions/sforum.smtp/1.0.0/aaa/backend/plugin"},
		{PID: 202, PPID: selfPID, RSSBytes: 17 * 1024 * 1024, Command: "storage/extensions/sforum.storage-fs/1.0.0/bbb/backend/plugin"},
		{PID: 301, PPID: 1, RSSBytes: 16 * 1024 * 1024, Command: "storage/extensions/sforum.smtp/1.0.0/old/backend/plugin"},
		{PID: 401, PPID: 999, RSSBytes: 20 * 1024 * 1024, Command: "storage/extensions/sforum.smtp/1.0.0/ccc/backend/plugin"},
		{PID: 999, PPID: 1, RSSBytes: 100 * 1024 * 1024, Command: "/other/sforum-api"},
		{PID: 203, PPID: selfPID, RSSBytes: 5 * 1024 * 1024, Command: "some-helper"},
	}

	selfRSS, familyRSS, children, found := AggregateFamily(selfPID, samples)
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
}

func TestAggregateRuntimeUsageSplitsAPIWorkerAndPlugins(t *testing.T) {
	selfPID := 100
	samples := []Sample{
		{PID: selfPID, RSSBytes: 100, CPUPercent: 1.25, Command: "sforum-api"},
		{PID: 200, RSSBytes: 50, CPUPercent: 2.5, Command: "sforum-worker"},
		{PID: 201, PPID: selfPID, RSSBytes: 10, CPUPercent: 0.5, Command: "storage/extensions/demo/1.0.0/a/backend/plugin"},
		{PID: 202, PPID: 200, RSSBytes: 20, CPUPercent: 1.5, Command: "storage/extensions/demo/1.0.0/a/backend/plugin"},
		{PID: 203, PPID: 1, RSSBytes: 99, CPUPercent: 9, Command: "storage/extensions/demo/1.0.0/orphan/backend/plugin"},
	}

	usage := AggregateRuntimeUsage(selfPID, samples)
	if usage.APIMemoryBytes != 100 || usage.WorkerMemoryBytes != 50 || usage.PluginMemoryBytes != 30 {
		t.Fatalf("unexpected memory split: %#v", usage)
	}
	if usage.TotalMemoryBytes != 180 || usage.TotalCPUPercent != 5.75 {
		t.Fatalf("unexpected totals: %#v", usage)
	}
	if usage.PluginChildCount != 2 || !usage.WorkerFound {
		t.Fatalf("unexpected process metadata: %#v", usage)
	}
	if usage.APIOwnedPluginMemoryBytes != 10 || usage.APIOwnedPluginCount != 1 {
		t.Fatalf("unexpected API-owned plugin metadata: %#v", usage)
	}
	if len(usage.Plugins) != 1 || usage.Plugins[0].ExtensionID != "demo" || usage.Plugins[0].ProcessCount != 2 {
		t.Fatalf("unexpected plugin details: %#v", usage.Plugins)
	}
	if usage.Plugins[0].APIOwnedProcessCount != 1 || usage.Plugins[0].WorkerOwnedProcessCount != 1 || usage.PluginOverlapCount != 0 {
		t.Fatalf("unexpected plugin ownership: %#v", usage.Plugins[0])
	}
}

func TestAggregateRuntimeUsageExposesCompletePSSAndOverlap(t *testing.T) {
	selfPID := 100
	samples := []Sample{
		{PID: selfPID, RSSBytes: 100, PSSBytes: 80, Command: "sforum-api"},
		{PID: 201, PPID: selfPID, RSSBytes: 20, PSSBytes: 15, Command: "storage/extensions/demo.plugin/1.0.0/a/backend/plugin"},
		{PID: 202, PPID: selfPID, RSSBytes: 22, PSSBytes: 16, Command: "storage/extensions/demo.plugin/1.0.0/b/backend/plugin"},
	}

	usage := AggregateRuntimeUsage(selfPID, samples)
	if usage.TotalPSSBytes != 111 || usage.PluginPSSBytes != 31 {
		t.Fatalf("unexpected PSS totals: %#v", usage)
	}
	if usage.PluginOverlapCount != 1 || len(usage.Plugins) != 1 || usage.Plugins[0].PSSBytes != 31 {
		t.Fatalf("unexpected overlap details: %#v", usage)
	}

	samples[2].PSSBytes = 0
	usage = AggregateRuntimeUsage(selfPID, samples)
	if usage.TotalPSSBytes != 0 || usage.APIPSSBytes != 0 || usage.Plugins[0].PSSBytes != 0 {
		t.Fatalf("partial PSS must be omitted: %#v", usage)
	}
}

func TestExtensionIDFromPluginCommand(t *testing.T) {
	cases := []struct {
		cmd  string
		want string
		ok   bool
	}{
		{"../../storage/extensions/sforum.smtp/1.1.0/abc/backend/plugin", "sforum.smtp", true},
		{"/app/storage/extensions/sforum.storage-fs/1.0.0/dead/backend/plugin", "sforum.storage-fs", true},
		{"/var/lib/sforum/extensions/sforum.auth-github/1.0.2/dead/backend/plugin", "sforum.auth-github", true},
		{"/app/extensions/builtin/plugins/sforum-smtp/backend/plugin", "sforum-smtp", true},
		{"/usr/bin/plugin", "", false},
		{"", "", false},
	}
	for _, tc := range cases {
		got, ok := ExtensionIDFromPluginCommand(tc.cmd)
		if ok != tc.ok || got != tc.want {
			t.Fatalf("cmd=%q got=(%q,%v) want=(%q,%v)", tc.cmd, got, ok, tc.want, tc.ok)
		}
	}
}

func TestOwnedPluginRSSByExtensionID(t *testing.T) {
	selfPID := 100
	samples := []Sample{
		{PID: selfPID, PPID: 1, RSSBytes: 160 * 1024 * 1024, Command: "sforum-api"},
		{PID: 201, PPID: selfPID, RSSBytes: 18 * 1024 * 1024, Command: "storage/extensions/sforum.smtp/1.0.0/aaa/backend/plugin"},
		{PID: 202, PPID: selfPID, RSSBytes: 17 * 1024 * 1024, Command: "storage/extensions/sforum.storage-fs/1.0.0/bbb/backend/plugin"},
		{PID: 203, PPID: selfPID, RSSBytes: 2 * 1024 * 1024, Command: "storage/extensions/sforum.smtp/1.0.0/ccc/backend/plugin"},
		{PID: 301, PPID: 1, RSSBytes: 16 * 1024 * 1024, Command: "storage/extensions/sforum.smtp/1.0.0/old/backend/plugin"},
		{PID: 401, PPID: 999, RSSBytes: 20 * 1024 * 1024, Command: "storage/extensions/sforum.smtp/1.0.0/other/backend/plugin"},
	}
	byID := OwnedPluginRSSByExtensionID(selfPID, samples)
	if byID["sforum.smtp"] != 20*1024*1024 {
		t.Fatalf("sforum.smtp RSS=%d want 20MiB", byID["sforum.smtp"])
	}
	if byID["sforum.storage-fs"] != 17*1024*1024 {
		t.Fatalf("storage-fs RSS=%d", byID["sforum.storage-fs"])
	}
}

type fixedSampler struct {
	samples []Sample
	err     error
}

func (f fixedSampler) List() ([]Sample, error) {
	if f.err != nil {
		return nil, f.err
	}
	return f.samples, nil
}

func TestSampleOwnedPluginRSSWith(t *testing.T) {
	selfPID := os.Getpid()
	sampler := fixedSampler{samples: []Sample{
		{PID: selfPID, PPID: 1, RSSBytes: 100 * 1024 * 1024, Command: "sforum-api"},
		{PID: selfPID + 1, PPID: selfPID, RSSBytes: 12 * 1024 * 1024, Command: "storage/extensions/demo.plugin/1.0.0/x/backend/plugin"},
	}}
	byID := SampleOwnedPluginRSSWith(sampler)
	if byID["demo.plugin"] != 12*1024*1024 {
		t.Fatalf("got %#v", byID)
	}
	if len(SampleOwnedPluginRSSWith(fixedSampler{err: errors.New("boom")})) != 0 {
		t.Fatal("expected empty map on sampler error")
	}
}

func TestParsePSList(t *testing.T) {
	raw := []byte("  51406  42704 164360  1.25 /Users/x/tmp/sforum-api\n  51418  51406  18056  0.50 ../../storage/extensions/sforum.smtp/1.0.0/a/backend/plugin\n  51419  51406  12000 node backend/plugin.js\n")
	samples, err := ParsePSList(raw)
	if err != nil {
		t.Fatal(err)
	}
	if len(samples) != 3 {
		t.Fatalf("len=%d", len(samples))
	}
	if samples[0].PID != 51406 || samples[0].PPID != 42704 || samples[0].RSSBytes != 164360*1024 || samples[0].CPUPercent != 1.25 {
		t.Fatalf("row0=%#v", samples[0])
	}
	if !IsBackendPluginCommand(samples[1].Command) {
		t.Fatalf("plugin cmd=%q", samples[1].Command)
	}
	if samples[2].Command != "node backend/plugin.js" {
		t.Fatalf("legacy command parsing lost words: %q", samples[2].Command)
	}
}

type countingSampler struct {
	calls   int
	samples []Sample
}

func (s *countingSampler) List() ([]Sample, error) {
	s.calls++
	return append([]Sample(nil), s.samples...), nil
}

func TestCachedSamplerSharesFramesAndReturnsCopies(t *testing.T) {
	inner := &countingSampler{samples: []Sample{{PID: 1, RSSBytes: 10}}}
	cached := NewCachedSampler(inner, 5*time.Second).(*cachedSampler)
	now := time.Date(2026, 7, 31, 12, 0, 0, 0, time.UTC)
	cached.now = func() time.Time { return now }

	first, err := cached.List()
	if err != nil {
		t.Fatal(err)
	}
	first[0].RSSBytes = 999
	second, err := cached.List()
	if err != nil {
		t.Fatal(err)
	}
	if inner.calls != 1 || second[0].RSSBytes != 10 || second[0].CapturedAt != now {
		t.Fatalf("cached frame calls=%d samples=%#v", inner.calls, second)
	}

	now = now.Add(5 * time.Second)
	if _, err := cached.List(); err != nil {
		t.Fatal(err)
	}
	if inner.calls != 2 {
		t.Fatalf("expired cache calls=%d", inner.calls)
	}
}

func TestUsageWindowUsesUniqueFramesAndRollingMedian(t *testing.T) {
	window := NewUsageWindow(time.Minute)
	base := time.Date(2026, 7, 31, 12, 0, 0, 0, time.UTC)
	values := []uint64{100, 300, 200}
	var observed RuntimeUsage
	for index, value := range values {
		at := base.Add(time.Duration(index) * 5 * time.Second)
		observed = window.Observe(RuntimeUsage{
			APIMemoryBytes: value, PluginMemoryBytes: value / 2,
			TotalMemoryBytes: value + value/2, SampledAt: &at,
		})
	}
	if observed.APIMemoryMedianBytes != 200 || observed.PluginMemoryMedianBytes != 100 || observed.MemorySampleCount != 3 || observed.MemoryWindowSeconds != 60 {
		t.Fatalf("unexpected median window: %#v", observed)
	}

	duplicate := window.Observe(observed)
	if duplicate.MemorySampleCount != 3 {
		t.Fatalf("cached frame counted twice: %#v", duplicate)
	}

	later := base.Add(70 * time.Second)
	pruned := window.Observe(RuntimeUsage{APIMemoryBytes: 50, TotalMemoryBytes: 50, SampledAt: &later})
	if pruned.MemorySampleCount != 1 || pruned.APIMemoryMedianBytes != 50 {
		t.Fatalf("old samples were not pruned: %#v", pruned)
	}
}
