package extensions

import (
	"context"
	"errors"
	"sync"
	"testing"
)

// recordingExtensionSessionInvalidator 记录每一次 InvalidateExtension 的扩展 id，
// 可设置为返回错误或 panic，用来证明擦除是提交后的 best-effort。
type recordingExtensionSessionInvalidator struct {
	mu       sync.Mutex
	ids      []string
	failWith error
	panicNow bool
}

func (r *recordingExtensionSessionInvalidator) InvalidateExtension(_ context.Context, extensionID string) (int, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.panicNow {
		panic("invalidator exploded")
	}
	r.ids = append(r.ids, extensionID)
	if r.failWith != nil {
		return 0, r.failWith
	}
	return 1, nil
}

func (r *recordingExtensionSessionInvalidator) recorded() []string {
	r.mu.Lock()
	defer r.mu.Unlock()
	return append([]string(nil), r.ids...)
}

func (r *recordingExtensionSessionInvalidator) reset() {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.ids = nil
}

func enabledUploadedPlugin(id string) (Extension, *fakeExtensionStore) {
	item := installedExtension(id, TypePlugin, ManifestBackend{})
	item.Status = StatusEnabled
	item.Source = SourceUploaded
	item.IsDeletable = true
	store := &fakeExtensionStore{items: map[string]Extension{id: item}}
	return item, store
}

// TestDisableErasesExactlyTheDisabledExtension 证明禁用擦除的正是被禁用那个扩展 id。
func TestDisableErasesExactlyTheDisabledExtension(t *testing.T) {
	item, store := enabledUploadedPlugin("session.erase.disable")
	runtime := &fakeRuntimeManager{statuses: map[string]RuntimeStatus{item.ID: {State: RuntimeRunning}}}
	fake := &recordingExtensionSessionInvalidator{}
	service := newLifecycleService(store, t.TempDir(), runtime, WithPluginSessionStateInvalidator(fake))

	if _, err := service.Disable(context.Background(), extensionManager(), item.ID); err != nil {
		t.Fatalf("disable: %v", err)
	}
	if got := fake.recorded(); len(got) != 1 || got[0] != item.ID {
		t.Fatalf("erased extension ids = %#v, want exactly [%s]", got, item.ID)
	}
}

// TestUninstallErasesExactlyTheUninstalledExtension 证明卸载（disable 之后）再次擦除
// 该扩展 id：同 id 重装不得读到上一份会话状态字节。
func TestUninstallErasesExactlyTheUninstalledExtension(t *testing.T) {
	runtime := &fakeRuntimeManager{}
	fake := &recordingExtensionSessionInvalidator{}
	service := newLifecycleService(&fakeExtensionStore{}, t.TempDir(), runtime, WithPluginSessionStateInvalidator(fake))
	actor := extensionManager()

	installed, err := service.InstallArchive(context.Background(), actor, ArchiveInput{
		FileName: "erase.zip",
		Data:     extensionArchive(t, validManifest("session.erase.uninstall", TypePlugin)),
	})
	if err != nil {
		t.Fatalf("install: %v", err)
	}
	if _, err := service.Disable(context.Background(), actor, installed.ID); err != nil {
		t.Fatalf("disable: %v", err)
	}
	fake.reset()
	if err := service.Uninstall(context.Background(), actor, installed.ID, UninstallInput{}); err != nil {
		t.Fatalf("uninstall: %v", err)
	}
	if got := fake.recorded(); len(got) != 1 || got[0] != installed.ID {
		t.Fatalf("erased extension ids on uninstall = %#v, want exactly [%s]", got, installed.ID)
	}
}

// TestDisableSurvivesSessionErasureFailure 证明擦除失败（错误或 panic）不影响禁用本身。
func TestDisableSurvivesSessionErasureFailure(t *testing.T) {
	item, store := enabledUploadedPlugin("session.erase.resilient")
	runtime := &fakeRuntimeManager{statuses: map[string]RuntimeStatus{item.ID: {State: RuntimeRunning}}}
	fake := &recordingExtensionSessionInvalidator{failWith: errors.New("redis down")}
	service := newLifecycleService(store, t.TempDir(), runtime, WithPluginSessionStateInvalidator(fake))

	disabled, err := service.Disable(context.Background(), extensionManager(), item.ID)
	if err != nil {
		t.Fatalf("disable must not fail when erasure fails: %v", err)
	}
	if disabled.Status != StatusDisabled {
		t.Fatalf("status = %s, want disabled", disabled.Status)
	}
	if got := fake.recorded(); len(got) != 1 || got[0] != item.ID {
		t.Fatalf("erasure was attempted: %#v", got)
	}

	panicker := &recordingExtensionSessionInvalidator{panicNow: true}
	item2, store2 := enabledUploadedPlugin("session.erase.panic")
	runtime2 := &fakeRuntimeManager{statuses: map[string]RuntimeStatus{item2.ID: {State: RuntimeRunning}}}
	service2 := newLifecycleService(store2, t.TempDir(), runtime2, WithPluginSessionStateInvalidator(panicker))
	disabled2, err := service2.Disable(context.Background(), extensionManager(), item2.ID)
	if err != nil {
		t.Fatalf("disable must not fail when erasure panics: %v", err)
	}
	if disabled2.Status != StatusDisabled {
		t.Fatalf("status = %s, want disabled", disabled2.Status)
	}
}

// TestLifecycleWorksWithoutSessionInvalidator 证明未注入 invalidator 时
// 禁用/卸载照常（未接存储的部署不被拖挂）。
func TestLifecycleWorksWithoutSessionInvalidator(t *testing.T) {
	item, store := enabledUploadedPlugin("session.erase.unwired")
	runtime := &fakeRuntimeManager{statuses: map[string]RuntimeStatus{item.ID: {State: RuntimeRunning}}}
	service := newLifecycleService(store, t.TempDir(), runtime)

	disabled, err := service.Disable(context.Background(), extensionManager(), item.ID)
	if err != nil {
		t.Fatalf("disable without invalidator: %v", err)
	}
	if disabled.Status != StatusDisabled {
		t.Fatalf("status = %s, want disabled", disabled.Status)
	}
	if err := service.Uninstall(context.Background(), extensionManager(), item.ID, UninstallInput{}); err != nil {
		t.Fatalf("uninstall without invalidator: %v", err)
	}
}
