package processmemory

import (
	"errors"
	"os"
	"testing"
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

func TestExtensionIDFromPluginCommand(t *testing.T) {
	cases := []struct {
		cmd  string
		want string
		ok   bool
	}{
		{"../../storage/extensions/sforum.smtp/1.1.0/abc/backend/plugin", "sforum.smtp", true},
		{"/app/storage/extensions/sforum.storage-fs/1.0.0/dead/backend/plugin", "sforum.storage-fs", true},
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
	raw := []byte("  51406  42704 164360 /Users/x/tmp/sforum-api\n  51418  51406  18056 ../../storage/extensions/sforum.smtp/1.0.0/a/backend/plugin\n")
	samples, err := ParsePSList(raw)
	if err != nil {
		t.Fatal(err)
	}
	if len(samples) != 2 {
		t.Fatalf("len=%d", len(samples))
	}
	if samples[0].PID != 51406 || samples[0].PPID != 42704 || samples[0].RSSBytes != 164360*1024 {
		t.Fatalf("row0=%#v", samples[0])
	}
	if !IsBackendPluginCommand(samples[1].Command) {
		t.Fatalf("plugin cmd=%q", samples[1].Command)
	}
}
