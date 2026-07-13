package extensions

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

type fakeTrustRevoker struct {
	calls []string
}

func (f *fakeTrustRevoker) RevokeAllForExtension(_ context.Context, extensionID string, _ int64) error {
	f.calls = append(f.calls, extensionID)
	return nil
}

func newLifecycleService(store Store, root string, runtime RuntimeManager, opts ...ServiceOption) *Service {
	return NewServiceWithOptions(store, root, "", runtime, opts...)
}

func TestInstallOrUpgradeArchiveStagesUpgradeWithoutTouchingActiveRuntime(t *testing.T) {
	store := &fakeExtensionStore{}
	runtime := &fakeRuntimeManager{}
	revoker := &fakeTrustRevoker{}
	service := newLifecycleService(store, t.TempDir(), runtime, WithTrustRevoker(revoker))
	actor := extensionManager()

	manifestV1 := validManifest("upgrade.plugin", TypePlugin)
	first, err := service.InstallOrUpgradeArchive(context.Background(), actor, ArchiveInput{
		FileName: "v1.zip",
		Data: extensionArchive(t, manifestV1,
			zipFile{name: "README.md", body: "v1"},
			zipFile{name: "migrations/001_init.sql", body: "SELECT 1;"},
		),
	})
	if err != nil {
		t.Fatalf("first install: %v", err)
	}
	if first.Upgraded {
		t.Fatalf("first install should not be upgrade: %#v", first)
	}
	if len(store.events) != 1 || store.events[0].Action != EventInstalled {
		t.Fatalf("expected installed event, got %#v", store.events)
	}

	// 启用后再上传升级包：只保存候选，活动制品继续服务直到受信任升级事务提交。
	enabled := store.items[first.Extension.ID]
	enabled.Status = StatusEnabled
	store.items[enabled.ID] = enabled
	runtime.statuses = map[string]RuntimeStatus{enabled.ID: {State: RuntimeRunning}}

	manifestV2 := strings.Replace(manifestV1, `"version": "1.0.0"`, `"version": "1.1.0"`, 1)
	second, err := service.InstallOrUpgradeArchive(context.Background(), actor, ArchiveInput{
		FileName: "v2.zip",
		Data: extensionArchive(t, manifestV2,
			zipFile{name: "README.md", body: "v2"},
			zipFile{name: "migrations/001_init.sql", body: "SELECT 1;"},
		),
	})
	if err != nil {
		t.Fatalf("upgrade install: %v", err)
	}
	if !second.Upgraded || second.PreviousVersion != "1.0.0" {
		t.Fatalf("expected upgrade metadata: %#v", second)
	}
	if second.Extension.Status != StatusEnabled {
		t.Fatalf("static upgrade changed active status: %s", second.Extension.Status)
	}
	if second.RequiredReEnable || !second.ActivationPending {
		t.Fatalf("static upgrade activation metadata = %#v", second)
	}
	if second.PreviousDigest == "" || second.PreviousDigest != second.Extension.PackageDigest {
		t.Fatalf("active digest changed during static upload: %#v", second)
	}
	if second.Extension.StagedVersion == nil || second.Extension.StagedVersion.Version != "1.1.0" ||
		second.Extension.StagedVersion.PackageDigest == second.PreviousDigest {
		t.Fatalf("upgrade candidate was not staged: %#v", second.Extension.StagedVersion)
	}
	if second.TrustRevoked {
		t.Fatalf("backend/settings-only package change must not revoke frontend trust")
	}
	if len(revoker.calls) != 0 {
		t.Fatalf("unexpected trust revoker call: %#v", revoker.calls)
	}
	if len(runtime.stopped) != 0 || len(runtime.hooks) != 0 {
		t.Fatalf("static upgrade touched active runtime: stopped=%#v hooks=%#v", runtime.stopped, runtime.hooks)
	}
	foundUpgraded := false
	for _, ev := range store.events {
		if ev.Action == EventUpgraded {
			foundUpgraded = true
			break
		}
	}
	if !foundUpgraded {
		t.Fatalf("expected upgraded event, got %#v", store.events)
	}
	ledger, err := store.ListMigrationLedger(context.Background(), "upgrade.plugin")
	if err != nil {
		t.Fatal(err)
	}
	if len(ledger) != 0 {
		t.Fatalf("static upload wrote migration execution ledger: %#v", ledger)
	}
}

func TestUninstallRequiresDisableAndDeletesPackage(t *testing.T) {
	store := &fakeExtensionStore{}
	runtime := &fakeRuntimeManager{}
	service := newLifecycleService(store, t.TempDir(), runtime)
	actor := extensionManager()

	installed, err := service.InstallArchive(context.Background(), actor, ArchiveInput{
		FileName: "u.zip",
		Data: extensionArchive(t, validManifest("gone.plugin", TypePlugin),
			zipFile{name: "README.md", body: "x"},
		),
	})
	if err != nil {
		t.Fatal(err)
	}
	packagePath := installed.PackagePath

	item := store.items[installed.ID]
	item.Status = StatusEnabled
	store.items[item.ID] = item
	err = service.Uninstall(context.Background(), actor, installed.ID, UninstallInput{})
	if !errors.Is(err, ErrMustDisableFirst) {
		t.Fatalf("expected must disable first, got %v", err)
	}

	item.Status = StatusDisabled
	store.items[item.ID] = item
	if err := service.Uninstall(context.Background(), actor, installed.ID, UninstallInput{}); err != nil {
		t.Fatalf("uninstall: %v", err)
	}
	if _, err := store.Get(context.Background(), installed.ID); !errors.Is(err, ErrExtensionNotFound) {
		t.Fatalf("expected deleted from store, got %v", err)
	}
	if _, err := os.Stat(packagePath); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("expected package dir removed, err=%v", err)
	}
	if len(runtime.stopped) == 0 {
		t.Fatalf("uninstall should drain runtime")
	}
}

func TestUninstallRejectsBuiltin(t *testing.T) {
	store := &fakeExtensionStore{items: map[string]Extension{
		"builtin.plugin": {
			ID: "builtin.plugin", Type: TypePlugin, Status: StatusDisabled,
			Source: SourceBuiltin, IsSystem: true, IsDeletable: false,
		},
	}}
	service := NewService(store, t.TempDir())
	err := service.Uninstall(context.Background(), extensionManager(), "builtin.plugin", UninstallInput{})
	if !errors.Is(err, ErrNotDeletable) {
		t.Fatalf("expected not deletable, got %v", err)
	}
}

func TestUninstallRetainPackageKeepsDirectory(t *testing.T) {
	store := &fakeExtensionStore{}
	service := NewService(store, t.TempDir())
	actor := extensionManager()
	installed, err := service.InstallArchive(context.Background(), actor, ArchiveInput{
		FileName: "keep.zip",
		Data:     extensionArchive(t, validManifest("keep.plugin", TypePlugin)),
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := service.Uninstall(context.Background(), actor, installed.ID, UninstallInput{RetainPackage: true}); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(installed.PackagePath); err != nil {
		t.Fatalf("package should remain: %v", err)
	}
}

func TestDisableDrainsRuntimeBeforeStatusChange(t *testing.T) {
	store := &fakeExtensionStore{items: map[string]Extension{
		"drain.plugin": {
			ID: "drain.plugin", Name: "Drain", Version: "1.0.0",
			Type: TypePlugin, Status: StatusEnabled, Source: SourceUploaded, IsDeletable: true,
		},
	}}
	runtime := &fakeRuntimeManager{
		statuses: map[string]RuntimeStatus{"drain.plugin": {State: RuntimeRunning}},
	}
	service := NewServiceWithRuntime(store, t.TempDir(), runtime)
	disabled, err := service.Disable(context.Background(), extensionManager(), "drain.plugin")
	if err != nil {
		t.Fatal(err)
	}
	if disabled.Status != StatusDisabled {
		t.Fatalf("status: %s", disabled.Status)
	}
	if len(runtime.stopped) != 1 || runtime.stopped[0] != "drain.plugin" {
		t.Fatalf("expected stop before disable, stopped=%#v", runtime.stopped)
	}
	if len(runtime.hooks) == 0 {
		t.Fatalf("expected disabled lifecycle hook")
	}
}

func TestApplyDeclaredMigrationsIdempotent(t *testing.T) {
	root := t.TempDir()
	pkg := filepath.Join(root, "pkg")
	if err := os.MkdirAll(filepath.Join(pkg, "migrations"), 0o755); err != nil {
		t.Fatal(err)
	}
	migBody := []byte("-- plugin migration\n")
	if err := os.WriteFile(filepath.Join(pkg, "migrations", "001_init.sql"), migBody, 0o644); err != nil {
		t.Fatal(err)
	}
	// PackageDigest 非空时路径解析以 PackagePath 为根（内容寻址快照）。
	store := &fakeExtensionStore{items: map[string]Extension{
		"mig.plugin": {
			ID: "mig.plugin", Type: TypePlugin, Status: StatusInstalled,
			Source: SourceUploaded, IsDeletable: true, PackagePath: pkg, PackageDigest: "abc",
			Manifest: Manifest{
				ID: "mig.plugin", Type: TypePlugin, Version: "1.0.0",
				Migrations: []ManifestMigration{{Path: "migrations/001_init.sql"}},
			},
			InstalledAt: time.Now(), UpdatedAt: time.Now(),
		},
	}}
	service := NewService(store, root)
	first, err := service.ApplyDeclaredMigrations(context.Background(), extensionManager(), "mig.plugin")
	if err != nil {
		t.Fatal(err)
	}
	if len(first) != 1 || first[0].Checksum == "" || first[0].Status != "recorded" {
		t.Fatalf("first apply: %#v", first)
	}
	second, err := service.ApplyDeclaredMigrations(context.Background(), extensionManager(), "mig.plugin")
	if err != nil {
		t.Fatal(err)
	}
	if len(second) != 1 {
		t.Fatalf("second apply should not duplicate: %#v", second)
	}
}

func TestInstallOrUpgradeRejectsBuiltinOverwrite(t *testing.T) {
	store := &fakeExtensionStore{items: map[string]Extension{
		"sforum.smtp": {
			ID: "sforum.smtp", Type: TypePlugin, Status: StatusEnabled,
			Source: SourceBuiltin, IsSystem: true, IsDeletable: false, Version: "1.0.0",
		},
	}}
	service := NewService(store, t.TempDir())
	_, err := service.InstallOrUpgradeArchive(context.Background(), extensionManager(), ArchiveInput{
		FileName: "x.zip",
		Data:     extensionArchive(t, validManifest("sforum.smtp", TypePlugin)),
	})
	if !errors.Is(err, ErrNotDeletable) {
		t.Fatalf("expected not deletable for builtin upgrade, got %v", err)
	}
}
