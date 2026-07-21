package devhygiene

import (
	"fmt"
	"reflect"
	"syscall"
	"testing"
	"time"
)

func TestParseProcessList(t *testing.T) {
	raw := []byte("  100  10 /tmp/sforum-api\n  201   1 ../../storage/extensions/sforum.smtp/1.0.0/a/backend/plugin\n")
	rows, err := ParseProcessList(raw)
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 2 || rows[0].PID != 100 || rows[1].PPID != 1 {
		t.Fatalf("rows=%#v", rows)
	}
}

func TestCleanupOrphanExtensionPluginsDryRunSelectsOnlyOrphans(t *testing.T) {
	rows := []ProcessRow{
		{PID: 100, PPID: 10, Command: "/path/sforum-api"},
		{PID: 201, PPID: 100, Command: "storage/extensions/sforum.smtp/1.0.0/a/backend/plugin"},
		{PID: 202, PPID: 1, Command: "storage/extensions/sforum.smtp/1.0.0/b/backend/plugin"},
	}
	var signaled []int
	result, err := CleanupOrphanExtensionPlugins(CleanupOptions{
		DryRun: true,
		List:   func() ([]ProcessRow, error) { return rows, nil },
		Signal: func(pid int, sig syscall.Signal) error {
			signaled = append(signaled, pid)
			return nil
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(result.Selected, []int{202}) {
		t.Fatalf("selected=%v", result.Selected)
	}
	if len(signaled) != 0 {
		t.Fatalf("dry-run must not signal, got %v", signaled)
	}
}

func TestCleanupOrphanExtensionPluginsSignalsSelected(t *testing.T) {
	rows := []ProcessRow{
		{PID: 100, PPID: 10, Command: "/path/sforum-api"},
		{PID: 301, PPID: 1, Command: "storage/extensions/sforum.storage-fs/1.0.0/x/backend/plugin"},
	}
	var events []string
	result, err := CleanupOrphanExtensionPlugins(CleanupOptions{
		List:  func() ([]ProcessRow, error) { return rows, nil },
		Grace: time.Millisecond,
		Signal: func(pid int, sig syscall.Signal) error {
			events = append(events, sig.String()+":"+itoa(pid))
			if sig == 0 {
				// 模拟 TERM 后仍存活，触发 KILL。
				return nil
			}
			return nil
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(result.Selected, []int{301}) {
		t.Fatalf("selected=%v", result.Selected)
	}
	if len(result.Signaled) != 1 || result.Signaled[0] != 301 {
		t.Fatalf("signaled=%v", result.Signaled)
	}
	// TERM then existence probe (signal 0) then KILL
	if len(events) < 2 {
		t.Fatalf("expected term+probe/kill events, got %v", events)
	}
}

func TestCleanupDeniesLiveAPIChildren(t *testing.T) {
	rows := []ProcessRow{
		{PID: 50, PPID: 1, Command: "./tmp/sforum-api"},
		{PID: 51, PPID: 50, Command: "storage/extensions/sforum.smtp/1.0.0/a/backend/plugin"},
	}
	result, err := CleanupOrphanExtensionPlugins(CleanupOptions{
		DryRun: true,
		List:   func() ([]ProcessRow, error) { return rows, nil },
		Signal: func(int, syscall.Signal) error {
			t.Fatal("must not signal live API children")
			return nil
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Selected) != 0 {
		t.Fatalf("selected=%v", result.Selected)
	}
}

func itoa(v int) string {
	return fmt.Sprintf("%d", v)
}
