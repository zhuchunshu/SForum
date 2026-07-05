package extensions

import (
	"archive/zip"
	"bytes"
	"context"
	"errors"
	"io"
	"testing"
	"time"

	identity "github.com/zhuchunshu/sforum/apps/api/app/Models/Identity"
)

func TestServiceInstallArchiveRequiresExtensionManagePermission(t *testing.T) {
	service := NewService(&fakeExtensionStore{}, t.TempDir())
	actor := identity.Actor{ID: 7, Status: identity.UserStatusActive, Permissions: map[string]bool{}}

	_, err := service.InstallArchive(context.Background(), actor, ArchiveInput{
		FileName: "sample.zip",
		Data:     extensionArchive(t, validManifest("demo.plugin", TypePlugin)),
	})
	if !errors.Is(err, identity.ErrPermissionDenied) {
		t.Fatalf("expected permission denied, got %v", err)
	}
}

func TestServiceInstallArchiveValidatesManifestAndSafeZipPaths(t *testing.T) {
	service := NewService(&fakeExtensionStore{}, t.TempDir())
	actor := extensionManager()

	_, err := service.InstallArchive(context.Background(), actor, ArchiveInput{
		FileName: "missing.zip",
		Data:     extensionArchive(t, "", zipFile{name: "README.md", body: "hello"}),
	})
	if !errors.Is(err, ErrInvalidArchive) {
		t.Fatalf("expected missing manifest to be invalid archive, got %v", err)
	}

	_, err = service.InstallArchive(context.Background(), actor, ArchiveInput{
		FileName: "unsafe.zip",
		Data: extensionArchive(t, validManifest("demo.plugin", TypePlugin),
			zipFile{name: "../escape.txt", body: "oops"},
		),
	})
	if !errors.Is(err, ErrInvalidArchive) {
		t.Fatalf("expected unsafe path to be invalid archive, got %v", err)
	}

	_, err = service.InstallArchive(context.Background(), actor, ArchiveInput{
		FileName: "bad-manifest.zip",
		Data:     extensionArchive(t, `{"id":"Bad Space","name":"Broken","version":"1.0.0","type":"plugin","sforumVersion":"^1.0.0"}`),
	})
	if !errors.Is(err, ErrInvalidManifest) {
		t.Fatalf("expected invalid manifest, got %v", err)
	}
}

func TestServiceInstallArchiveStoresManifestPackageAndEvent(t *testing.T) {
	store := &fakeExtensionStore{}
	service := NewService(store, t.TempDir())

	installed, err := service.InstallArchive(context.Background(), extensionManager(), ArchiveInput{
		FileName: "sample.zip",
		Data: extensionArchive(t, validManifest("demo.plugin", TypePlugin),
			zipFile{name: "backend/plugin", body: "#!/bin/sh\n"},
			zipFile{name: "README.md", body: "hello"},
		),
	})
	if err != nil {
		t.Fatalf("InstallArchive returned error: %v", err)
	}

	if installed.ID != "demo.plugin" || installed.Version != "1.0.0" || installed.Status != StatusInstalled {
		t.Fatalf("unexpected installed extension: %#v", installed)
	}
	if store.saved.Manifest.ID != "demo.plugin" || store.saved.Manifest.Type != TypePlugin {
		t.Fatalf("manifest was not saved: %#v", store.saved.Manifest)
	}
	if store.saved.PackagePath == "" {
		t.Fatal("expected package path to be stored")
	}
	if len(store.events) != 1 || store.events[0].Action != EventInstalled {
		t.Fatalf("expected install event, got %#v", store.events)
	}
}

func TestServiceInstallArchiveRejectsReservedDefaultThemeID(t *testing.T) {
	service := NewService(&fakeExtensionStore{}, t.TempDir())

	_, err := service.InstallArchive(context.Background(), extensionManager(), ArchiveInput{
		FileName: "default-theme.zip",
		Data:     extensionArchive(t, validManifest(DefaultThemeID, TypeTheme)),
	})
	if !errors.Is(err, ErrInvalidManifest) {
		t.Fatalf("expected reserved default theme id to be rejected, got %v", err)
	}
}

func TestServiceEnableRunsPluginPreflightBeforeStatusChange(t *testing.T) {
	expected := errors.New("rpc handshake failed")
	store := &fakeExtensionStore{items: map[string]Extension{
		"demo.plugin": installedExtension("demo.plugin", TypePlugin, ManifestBackend{Entry: "backend/plugin"}),
	}}
	service := NewServiceWithHooks(store, t.TempDir(), fakeRuntime{err: expected}, nil)

	_, err := service.Enable(context.Background(), extensionManager(), "demo.plugin")
	if !errors.Is(err, ErrPreflightFailed) {
		t.Fatalf("expected preflight failure, got %v", err)
	}
	if store.enabledID != "" {
		t.Fatalf("extension status changed despite failed preflight: %q", store.enabledID)
	}
	if last := store.events[len(store.events)-1]; last.Action != EventEnableFailed || last.Message == "" {
		t.Fatalf("expected enable failure event, got %#v", store.events)
	}

	service = NewServiceWithHooks(store, t.TempDir(), fakeRuntime{}, nil)
	enabled, err := service.Enable(context.Background(), extensionManager(), "demo.plugin")
	if err != nil {
		t.Fatalf("Enable returned error: %v", err)
	}
	if enabled.Status != StatusEnabled || store.enabledID != "demo.plugin" {
		t.Fatalf("expected enabled plugin, got enabled=%#v store=%q", enabled, store.enabledID)
	}
}

func TestServiceEnableRejectsThemesBecauseThemesUseActivation(t *testing.T) {
	store := &fakeExtensionStore{items: map[string]Extension{
		"starter.theme": installedExtension("starter.theme", TypeTheme, ManifestBackend{}),
	}}
	service := NewServiceWithHooks(store, t.TempDir(), nil, fakeThemeBuilder{})

	_, err := service.Enable(context.Background(), extensionManager(), "starter.theme")
	if !errors.Is(err, ErrThemeActivationRequired) {
		t.Fatalf("expected theme activation requirement, got %v", err)
	}
	if store.enabledID != "" {
		t.Fatalf("theme should not be enabled through plugin lifecycle, got %q", store.enabledID)
	}
}

func TestServiceVerifyExtensionChecksThemeLayerWithoutActivating(t *testing.T) {
	expected := errors.New("nuxt layer missing")
	store := &fakeExtensionStore{items: map[string]Extension{
		"starter.theme": installedExtension("starter.theme", TypeTheme, ManifestBackend{}),
	}}
	service := NewServiceWithHooks(store, t.TempDir(), nil, fakeThemeBuilder{err: expected})

	_, err := service.VerifyExtension(context.Background(), extensionManager(), "starter.theme")
	if !errors.Is(err, ErrBuildFailed) {
		t.Fatalf("expected build failure, got %v", err)
	}
	if store.enabledID != "" || store.activeThemeID != "" {
		t.Fatalf("verify should not activate theme, enabled=%q active=%q", store.enabledID, store.activeThemeID)
	}

	service = NewServiceWithHooks(store, t.TempDir(), nil, fakeThemeBuilder{})
	verified, err := service.VerifyExtension(context.Background(), extensionManager(), "starter.theme")
	if err != nil {
		t.Fatalf("VerifyExtension returned error: %v", err)
	}
	if verified.Status != StatusInstalled || store.enabledID != "" || store.activeThemeID != "" {
		t.Fatalf("verify should keep installed theme inactive, got verified=%#v enabled=%q active=%q", verified, store.enabledID, store.activeThemeID)
	}
	if last := store.events[len(store.events)-1]; last.Action != EventVerified {
		t.Fatalf("expected verify event, got %#v", store.events)
	}
}

func TestServiceActivateThemeAllowsOnlyBuiltinDefaultThemeInV1(t *testing.T) {
	store := &fakeExtensionStore{items: map[string]Extension{
		DefaultThemeID:  protectedBuiltinExtension(DefaultThemeID, TypeTheme),
		"starter.theme": uploadedExtension("starter.theme", TypeTheme),
	}}
	service := NewServiceWithHooks(store, t.TempDir(), nil, fakeThemeBuilder{})

	_, err := service.ActivateTheme(context.Background(), extensionManager(), "starter.theme")
	if !errors.Is(err, ErrThemeRuntimeUnavailable) {
		t.Fatalf("expected uploaded theme activation to be unavailable, got %v", err)
	}
	if store.activeThemeID != "" {
		t.Fatalf("uploaded theme should not become active, got %q", store.activeThemeID)
	}

	active, err := service.ActivateTheme(context.Background(), extensionManager(), DefaultThemeID)
	if err != nil {
		t.Fatalf("ActivateTheme returned error: %v", err)
	}
	if active.Status != StatusEnabled || store.activeThemeID != DefaultThemeID {
		t.Fatalf("expected default theme active, got active=%#v activeID=%q", active, store.activeThemeID)
	}
	if last := store.events[len(store.events)-1]; last.Action != EventThemeActivated {
		t.Fatalf("expected theme activated event, got %#v", store.events)
	}
}

func TestServiceEnsureDefaultThemeActiveRepairsUnsafeThemeState(t *testing.T) {
	store := &fakeExtensionStore{items: map[string]Extension{
		DefaultThemeID: protectedBuiltinExtension(DefaultThemeID, TypeTheme),
		"starter.theme": {
			ID:      "starter.theme",
			Name:    "Starter Theme",
			Version: "1.0.0",
			Type:    TypeTheme,
			Status:  StatusEnabled,
			Source:  SourceUploaded,
			Manifest: Manifest{
				ID:            "starter.theme",
				Name:          "Starter Theme",
				Version:       "1.0.0",
				Type:          TypeTheme,
				SForumVersion: "^1.0.0",
			},
			InstalledAt: time.Now(),
			UpdatedAt:   time.Now(),
		},
	}}
	store.activeThemeID = "starter.theme"
	service := NewServiceWithHooks(store, t.TempDir(), nil, fakeThemeBuilder{})

	active, err := service.EnsureDefaultThemeActive(context.Background())
	if err != nil {
		t.Fatalf("EnsureDefaultThemeActive returned error: %v", err)
	}
	if active.ID != DefaultThemeID || store.activeThemeID != DefaultThemeID {
		t.Fatalf("expected unsafe active theme repaired to default, got active=%#v activeID=%q", active, store.activeThemeID)
	}

	active, err = service.EnsureDefaultThemeActive(context.Background())
	if err != nil {
		t.Fatalf("EnsureDefaultThemeActive second call returned error: %v", err)
	}
	if active.ID != DefaultThemeID || store.activeThemeID != DefaultThemeID {
		t.Fatalf("expected idempotent default active theme, got active=%#v activeID=%q", active, store.activeThemeID)
	}
}

func TestServiceDisableRequiresPermissionAndRecordsEvent(t *testing.T) {
	store := &fakeExtensionStore{items: map[string]Extension{
		"demo.plugin": installedExtension("demo.plugin", TypePlugin, ManifestBackend{}),
	}}
	service := NewService(store, t.TempDir())

	_, err := service.Disable(context.Background(), identity.Actor{ID: 9, Status: identity.UserStatusActive}, "demo.plugin")
	if !errors.Is(err, identity.ErrPermissionDenied) {
		t.Fatalf("expected permission denied, got %v", err)
	}

	disabled, err := service.Disable(context.Background(), extensionManager(), "demo.plugin")
	if err != nil {
		t.Fatalf("Disable returned error: %v", err)
	}
	if disabled.Status != StatusDisabled || store.disabledID != "demo.plugin" {
		t.Fatalf("expected disabled extension, got disabled=%#v store=%q", disabled, store.disabledID)
	}
	if last := store.events[len(store.events)-1]; last.Action != EventDisabled {
		t.Fatalf("expected disabled event, got %#v", store.events)
	}
}

func extensionManager() identity.Actor {
	return identity.Actor{
		ID:          42,
		Status:      identity.UserStatusActive,
		Permissions: map[string]bool{identity.PermissionExtensionManage: true},
	}
}

func validManifest(id string, extensionType string) string {
	return `{
		"id": "` + id + `",
		"name": "Demo Extension",
		"version": "1.0.0",
		"type": "` + extensionType + `",
		"sforumVersion": "^1.0.0",
		"permissions": ["topic.create"],
		"settings": [{"key": "demo.enabled", "label": "Enabled", "type": "boolean"}],
		"migrations": [{"path": "migrations/001_init.sql"}],
		"backend": {"entry": "backend/plugin", "rpc": "hashicorp-go-plugin"},
		"frontend": {"layer": "frontend/layer"},
		"adminPages": [{"path": "/demo", "label": "Demo", "permission": "extension.demo.manage"}],
		"routes": [{"path": "/hello", "methods": ["GET"]}],
		"hooks": [{"name": "topic.created"}],
		"jobs": [{"name": "demo.sync"}]
	}`
}

type zipFile struct {
	name string
	body string
}

func extensionArchive(t *testing.T, manifest string, files ...zipFile) []byte {
	t.Helper()

	var buffer bytes.Buffer
	writer := zip.NewWriter(&buffer)
	if manifest != "" {
		writeZipFile(t, writer, ManifestFileName, manifest)
	}
	for _, file := range files {
		writeZipFile(t, writer, file.name, file.body)
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("close zip: %v", err)
	}
	return buffer.Bytes()
}

func writeZipFile(t *testing.T, writer *zip.Writer, name string, body string) {
	t.Helper()
	file, err := writer.Create(name)
	if err != nil {
		t.Fatalf("create zip file %s: %v", name, err)
	}
	if _, err := io.WriteString(file, body); err != nil {
		t.Fatalf("write zip file %s: %v", name, err)
	}
}

func installedExtension(id string, extensionType string, backend ManifestBackend) Extension {
	return Extension{
		ID:      id,
		Name:    "Demo Extension",
		Version: "1.0.0",
		Type:    extensionType,
		Status:  StatusInstalled,
		Manifest: Manifest{
			ID:            id,
			Name:          "Demo Extension",
			Version:       "1.0.0",
			Type:          extensionType,
			SForumVersion: "^1.0.0",
			Backend:       backend,
			Frontend:      ManifestFrontend{Layer: "frontend/layer"},
		},
		PackagePath: "/tmp/demo.zip",
		InstalledAt: time.Now(),
		UpdatedAt:   time.Now(),
	}
}

func protectedBuiltinExtension(id string, extensionType string) Extension {
	item := installedExtension(id, extensionType, ManifestBackend{})
	item.Status = StatusEnabled
	item.Source = SourceBuiltin
	item.IsSystem = true
	item.IsDeletable = false
	return item
}

func uploadedExtension(id string, extensionType string) Extension {
	item := installedExtension(id, extensionType, ManifestBackend{})
	item.Source = SourceUploaded
	item.IsSystem = false
	item.IsDeletable = true
	return item
}

type fakeRuntime struct {
	err error
}

func (r fakeRuntime) Check(context.Context, Extension) error {
	return r.err
}

type fakeThemeBuilder struct {
	err error
}

func (b fakeThemeBuilder) Build(context.Context, Extension) error {
	return b.err
}

type fakeExtensionStore struct {
	items         map[string]Extension
	saved         Extension
	enabledID     string
	disabledID    string
	activeThemeID string
	events        []ExtensionEvent
}

func (s *fakeExtensionStore) List(context.Context) ([]Extension, error) {
	items := make([]Extension, 0, len(s.items))
	for _, item := range s.items {
		items = append(items, item)
	}
	return items, nil
}

func (s *fakeExtensionStore) Get(_ context.Context, id string) (Extension, error) {
	if item, ok := s.items[id]; ok {
		return item, nil
	}
	return Extension{}, ErrExtensionNotFound
}

func (s *fakeExtensionStore) SaveInstalled(_ context.Context, input SaveInstalledInput) (Extension, error) {
	item := Extension{
		ID:          input.Manifest.ID,
		Name:        input.Manifest.Name,
		Version:     input.Manifest.Version,
		Type:        input.Manifest.Type,
		Status:      StatusInstalled,
		Source:      SourceUploaded,
		IsDeletable: true,
		Manifest:    input.Manifest,
		PackagePath: input.PackagePath,
		InstalledAt: time.Now(),
		UpdatedAt:   time.Now(),
	}
	s.saved = item
	if s.items == nil {
		s.items = map[string]Extension{}
	}
	s.items[item.ID] = item
	return item, nil
}

func (s *fakeExtensionStore) SaveBuiltin(_ context.Context, input SaveBuiltinInput) (Extension, error) {
	item := Extension{
		ID:          input.Manifest.ID,
		Name:        input.Manifest.Name,
		Version:     input.Manifest.Version,
		Type:        input.Manifest.Type,
		Status:      StatusEnabled,
		Source:      SourceBuiltin,
		IsSystem:    true,
		IsDeletable: false,
		Manifest:    input.Manifest,
		PackagePath: input.PackagePath,
		InstalledAt: time.Now(),
		UpdatedAt:   time.Now(),
	}
	if s.items == nil {
		s.items = map[string]Extension{}
	}
	s.items[item.ID] = item
	return item, nil
}

func (s *fakeExtensionStore) Enable(_ context.Context, id string, extensionType string) (Extension, error) {
	item, ok := s.items[id]
	if !ok {
		return Extension{}, ErrExtensionNotFound
	}
	item.Status = StatusEnabled
	item.UpdatedAt = time.Now()
	s.items[id] = item
	s.enabledID = id
	if extensionType == TypeTheme {
		s.activeThemeID = id
	}
	return item, nil
}

func (s *fakeExtensionStore) ActivateTheme(_ context.Context, id string) (Extension, error) {
	item, ok := s.items[id]
	if !ok {
		return Extension{}, ErrExtensionNotFound
	}
	for currentID, current := range s.items {
		if current.Type == TypeTheme && current.ID != id && current.Status == StatusEnabled {
			current.Status = StatusDisabled
			current.UpdatedAt = time.Now()
			s.items[currentID] = current
		}
	}
	item.Status = StatusEnabled
	item.UpdatedAt = time.Now()
	s.items[id] = item
	s.activeThemeID = id
	return item, nil
}

func (s *fakeExtensionStore) ActiveTheme(context.Context) (Extension, error) {
	if s.activeThemeID != "" {
		if item, ok := s.items[s.activeThemeID]; ok {
			return item, nil
		}
	}
	for _, item := range s.items {
		if item.Type == TypeTheme && item.Status == StatusEnabled {
			return item, nil
		}
	}
	return Extension{}, ErrExtensionNotFound
}

func (s *fakeExtensionStore) Disable(_ context.Context, id string) (Extension, error) {
	item, ok := s.items[id]
	if !ok {
		return Extension{}, ErrExtensionNotFound
	}
	item.Status = StatusDisabled
	item.UpdatedAt = time.Now()
	s.items[id] = item
	s.disabledID = id
	if s.activeThemeID == id {
		s.activeThemeID = ""
	}
	return item, nil
}

func (s *fakeExtensionStore) CreateEvent(_ context.Context, input EventInput) (ExtensionEvent, error) {
	event := ExtensionEvent{
		ID:          int64(len(s.events) + 1),
		ExtensionID: input.ExtensionID,
		ActorUserID: input.ActorUserID,
		Action:      input.Action,
		Message:     input.Message,
		CreatedAt:   time.Now(),
	}
	s.events = append(s.events, event)
	return event, nil
}

func (s *fakeExtensionStore) ListEvents(context.Context, string, int) ([]ExtensionEvent, error) {
	return s.events, nil
}
