package extensions

import (
	"context"
	"errors"
	"strings"
	"testing"
)

// 卸载是扩展附件引用的唯一释放点：插件被删掉后若引用悬空，附件永远不会被
// 孤儿回收。这里断言释放发生在物理 purge 之前，并且失败会让卸载失败（可重试）。
func TestUninstallReleasesExtensionAttachmentReferences(t *testing.T) {
	item := lifecycleV2ServiceArtifact(t, "service.uninstall.attachment-release", "1.0.0", SourceUploaded)
	item.Status = StatusEnabled
	store := newFakeExtensionStore(map[string]Extension{item.ID: item})
	authority := lifecycleAuthorityTestGrant(t, item)
	order := []string{}
	runner := &lifecycleV2RecordingRunner{operationID: 402, beforeRun: func(LifecycleCoordinatorRunInput) {
		order = append(order, "coordinator")
		stored := store.items[item.ID]
		stored.Status = StatusDisabled
		store.items[item.ID] = stored
	}}
	reaper := &recordingAttachmentReferenceReaper{onRelease: func(extensionID string) {
		order = append(order, "reaper:"+extensionID)
	}}
	service := NewServiceWithOptions(store, t.TempDir(), "", LocalRuntimeManager{},
		WithAuditor(&lifecycleV2AuditWriter{nextID: 501}),
		WithLifecycleCoordinator(
			runner,
			func(_ context.Context, operation LifecycleMachineOperation, source *Extension, target Extension) error {
				order = append(order, "preflight")
				return nil
			},
			lifecycleV2AuthorityStore{authority: authority},
		),
		WithExtensionAttachmentReferenceReaper(reaper),
		WithLifecycleCleanupFinalizer(func(_ context.Context, operationID int64) (LifecycleCleanupFinalization, error) {
			order = append(order, "finalizer")
			delete(store.items, item.ID)
			return LifecycleCleanupFinalization{OperationID: operationID, Status: "finalized", PhysicalPurgeComplete: true}, nil
		}),
	)
	actor := techAdminPluginManager()
	actor.ID = 77

	result, err := service.UninstallWithResult(t.Context(), actor, item.ID, UninstallInput{
		RemovalMode: LifecycleRemovalPreserve, IdempotencyKey: "uninstall-attachment-release",
	})
	if err != nil {
		t.Fatal(err)
	}
	if !result.Uninstalled {
		t.Fatalf("uninstall result = %#v", result)
	}
	if reaper.calls != 1 || reaper.extensionID != item.ID {
		t.Fatalf("reaper calls = %d extensionID = %q", reaper.calls, reaper.extensionID)
	}
	// 释放必须在物理 purge 之前：失败时精确制品与身份还在，卸载可以幂等重试。
	if got := strings.Join(order, ","); !strings.Contains(got, "reaper:"+item.ID+",finalizer") {
		t.Fatalf("uninstall order = %q", got)
	}
}

func TestUninstallFailsWhenAttachmentReleaseFails(t *testing.T) {
	item := lifecycleV2ServiceArtifact(t, "service.uninstall.attachment-release-fail", "1.0.0", SourceUploaded)
	item.Status = StatusEnabled
	store := newFakeExtensionStore(map[string]Extension{item.ID: item})
	authority := lifecycleAuthorityTestGrant(t, item)
	finalized := false
	service := NewServiceWithOptions(store, t.TempDir(), "", LocalRuntimeManager{},
		WithAuditor(&lifecycleV2AuditWriter{nextID: 502}),
		WithLifecycleCoordinator(
			&lifecycleV2RecordingRunner{operationID: 403},
			func(context.Context, LifecycleMachineOperation, *Extension, Extension) error { return nil },
			lifecycleV2AuthorityStore{authority: authority},
		),
		WithExtensionAttachmentReferenceReaper(&recordingAttachmentReferenceReaper{
			onRelease: func(string) {},
			err:       errors.New("attachment reference store unavailable"),
		}),
		WithLifecycleCleanupFinalizer(func(_ context.Context, operationID int64) (LifecycleCleanupFinalization, error) {
			finalized = true
			return LifecycleCleanupFinalization{OperationID: operationID, Status: "finalized", PhysicalPurgeComplete: true}, nil
		}),
	)
	actor := techAdminPluginManager()
	actor.ID = 78

	if _, err := service.UninstallWithResult(t.Context(), actor, item.ID, UninstallInput{
		RemovalMode: LifecycleRemovalPreserve, IdempotencyKey: "uninstall-attachment-release-fail",
	}); !errors.Is(err, ErrLifecycleCleanupFinalization) {
		t.Fatalf("err = %v", err)
	}
	if finalized {
		t.Fatal("physical purge must not run when attachment reference release failed")
	}
}

// 禁用不释放：禁用期间图标保持可读是预期行为。
func TestDisableDoesNotReleaseExtensionAttachmentReferences(t *testing.T) {
	item := lifecycleV2ServiceArtifact(t, "service.disable.attachment-release", "1.0.0", SourceUploaded)
	item.Status = StatusEnabled
	store := newFakeExtensionStore(map[string]Extension{item.ID: item})
	reaper := &recordingAttachmentReferenceReaper{onRelease: func(string) {}}
	service := NewServiceWithOptions(store, t.TempDir(), "", LocalRuntimeManager{},
		WithAuditor(&lifecycleV2AuditWriter{nextID: 503}),
		WithLifecycleCoordinator(
			&lifecycleV2RecordingRunner{operationID: 404, beforeRun: func(LifecycleCoordinatorRunInput) {
				stored := store.items[item.ID]
				stored.Status = StatusDisabled
				store.items[item.ID] = stored
			}},
			func(context.Context, LifecycleMachineOperation, *Extension, Extension) error { return nil },
			lifecycleV2AuthorityStore{authority: lifecycleAuthorityTestGrant(t, item)},
		),
		WithExtensionAttachmentReferenceReaper(reaper),
	)
	actor := techAdminPluginManager()
	actor.ID = 79

	if _, err := service.DisableWithInput(t.Context(), actor, item.ID, LifecycleRequestInput{
		IdempotencyKey: "disable-attachment-release",
	}); err != nil {
		t.Fatal(err)
	}
	if reaper.calls != 0 {
		t.Fatalf("disable must not release attachment references, calls = %d", reaper.calls)
	}
}

type recordingAttachmentReferenceReaper struct {
	calls       int
	extensionID string
	err         error
	onRelease   func(extensionID string)
}

func (r *recordingAttachmentReferenceReaper) ReleaseExtensionAttachmentReferences(
	_ context.Context,
	extensionID string,
) (int, error) {
	r.calls++
	r.extensionID = extensionID
	if r.onRelease != nil {
		r.onRelease(extensionID)
	}
	if r.err != nil {
		return 0, r.err
	}
	return 1, nil
}

var _ ExtensionAttachmentReferenceReaper = (*recordingAttachmentReferenceReaper)(nil)
