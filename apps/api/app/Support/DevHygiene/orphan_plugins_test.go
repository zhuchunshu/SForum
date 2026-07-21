package devhygiene

import (
	"reflect"
	"testing"
)

func TestSelectOrphanExtensionPluginPIDsAllowReparented(t *testing.T) {
	rows := []ProcessRow{
		{PID: 100, PPID: 10, Command: "/path/sforum-api"},
		// orphan reparented to init
		{PID: 201, PPID: 1, Command: "../../storage/extensions/sforum.smtp/1.1.0/old/backend/plugin"},
		{PID: 202, PPID: 1, Command: "/app/storage/extensions/sforum.storage-fs/1.0.0/x/backend/plugin"},
	}
	got := SelectOrphanExtensionPluginPIDs(rows)
	want := []int{201, 202}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %v want %v", got, want)
	}
}

func TestSelectOrphanExtensionPluginPIDsDenyOwnedByLiveAPI(t *testing.T) {
	rows := []ProcessRow{
		{PID: 100, PPID: 10, Command: "/Users/x/apps/api/tmp/sforum-api"},
		// still owned by live API — must not select
		{PID: 301, PPID: 100, Command: "storage/extensions/sforum.smtp/1.0.0/a/backend/plugin"},
		// orphan — select
		{PID: 302, PPID: 1, Command: "storage/extensions/sforum.smtp/1.0.0/b/backend/plugin"},
	}
	got := SelectOrphanExtensionPluginPIDs(rows)
	if len(got) != 1 || got[0] != 302 {
		t.Fatalf("got %v want [302]", got)
	}
}

func TestSelectOrphanExtensionPluginPIDsDenyNonMatching(t *testing.T) {
	rows := []ProcessRow{
		{PID: 1, PPID: 0, Command: "/sbin/launchd"},
		{PID: 50, PPID: 1, Command: "/usr/bin/plugin"},
		{PID: 51, PPID: 1, Command: "node server.js"},
		{PID: 52, PPID: 1, Command: "backend/plugin"}, // missing extensions path
		{PID: 100, PPID: 1, Command: "sforum-api"},    // API itself
	}
	got := SelectOrphanExtensionPluginPIDs(rows)
	if len(got) != 0 {
		t.Fatalf("expected no selection, got %v", got)
	}
}

func TestSelectOrphanExtensionPluginPIDsAllowMissingParent(t *testing.T) {
	// 父进程已退出且未出现在表中：热重载竞态后的孤儿。
	rows := []ProcessRow{
		{PID: 400, PPID: 99999, Command: "storage/extensions/sforum.content-policy/1.0.0/z/backend/plugin"},
	}
	got := SelectOrphanExtensionPluginPIDs(rows)
	if len(got) != 1 || got[0] != 400 {
		t.Fatalf("got %v want [400]", got)
	}
}

func TestSelectOrphanExtensionPluginPIDsDenyParentedByNonAPI(t *testing.T) {
	rows := []ProcessRow{
		{PID: 20, PPID: 1, Command: "bash debug-plugin.sh"},
		{PID: 21, PPID: 20, Command: "storage/extensions/sforum.smtp/1.0.0/a/backend/plugin"},
	}
	got := SelectOrphanExtensionPluginPIDs(rows)
	if len(got) != 0 {
		t.Fatalf("must not kill plugins still parented by non-API helpers, got %v", got)
	}
}

func TestIsExtensionBackendPluginCommand(t *testing.T) {
	if !IsExtensionBackendPluginCommand("/x/storage/extensions/a/1/b/backend/plugin") {
		t.Fatal("expected allow")
	}
	if IsExtensionBackendPluginCommand("sforum-api") {
		t.Fatal("expected deny")
	}
}

// air 热重载时序：旧 API 仍存活时 pre_cmd 不能杀其子进程；旧 API 死后插件被 init 收养才可选。
func TestAirHotReloadSelectionBeforeAndAfterParentDeath(t *testing.T) {
	pluginCmd := "storage/extensions/sforum.smtp/1.0.0/gen1/backend/plugin"
	before := []ProcessRow{
		{PID: 100, PPID: 10, Command: "/tmp/sforum-api"}, // 即将被 air stopBin 杀掉的旧 API
		{PID: 200, PPID: 100, Command: pluginCmd},
		{PID: 110, PPID: 10, Command: "/tmp/sforum-api"}, // 新 API 已启动
	}
	if got := SelectOrphanExtensionPluginPIDs(before); len(got) != 0 {
		t.Fatalf("pre-kill selection must be empty, got %v", got)
	}

	// 旧 API 100 已死，插件被 reparent 到 1；新 API 110 仍在。
	after := []ProcessRow{
		{PID: 110, PPID: 10, Command: "/tmp/sforum-api"},
		{PID: 200, PPID: 1, Command: pluginCmd},
	}
	got := SelectOrphanExtensionPluginPIDs(after)
	if len(got) != 1 || got[0] != 200 {
		t.Fatalf("post-kill orphan must be selected, got %v", got)
	}
	// 新 API 自己的子进程仍受保护
	afterOwned := []ProcessRow{
		{PID: 110, PPID: 10, Command: "/tmp/sforum-api"},
		{PID: 200, PPID: 1, Command: pluginCmd},
		{PID: 210, PPID: 110, Command: "storage/extensions/sforum.smtp/1.0.0/gen2/backend/plugin"},
	}
	got = SelectOrphanExtensionPluginPIDs(afterOwned)
	if len(got) != 1 || got[0] != 200 {
		t.Fatalf("must select only reparented orphan, got %v", got)
	}
}
