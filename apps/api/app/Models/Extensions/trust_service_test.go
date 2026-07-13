package extensions

import (
	"bytes"
	"context"
	"errors"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	extensionpackage "github.com/zhuchunshu/sforum/apps/api/app/Support/ExtensionPackage"
)

func TestExecutableTrustChallengeBindsExactImpactAndIsOneUse(t *testing.T) {
	extension := exactTrustExtension(t, "demo.trusted")
	extensions := &fakeExtensionStore{items: map[string]Extension{extension.ID: extension}}
	now := time.Date(2026, 7, 13, 12, 0, 0, 0, time.UTC)
	trustStore := &memoryExecutableTrustStore{now: func() time.Time { return now }}
	service := NewExecutableTrustService(extensions, trustStore)
	service.now = func() time.Time { return now }
	service.random = bytes.NewReader(bytes.Repeat([]byte{0x2a}, 32))

	challenge, err := service.Challenge(context.Background(), extensionManager(), extension.ID)
	if err != nil {
		t.Fatal(err)
	}
	if challenge.Token == "" || challenge.Impact.Digest == "" || challenge.Impact.PackageDigest != extension.PackageDigest {
		t.Fatalf("incomplete challenge: %#v", challenge)
	}
	if challenge.Impact.ArtifactDigests["backend"] == "" || challenge.Impact.ArtifactDigests["migration:migrations/001.sql"] == "" {
		t.Fatalf("executable digests missing: %#v", challenge.Impact.ArtifactDigests)
	}
	if trustStore.challenge.TokenHash == "" || trustStore.challenge.TokenHash == challenge.Token {
		t.Fatal("store must persist only the token hash")
	}
	if err := service.ConfirmEnable(context.Background(), extensionManager(), extension, ""); !errors.Is(err, ErrTrustChallengeRequired) {
		t.Fatalf("missing token: %v", err)
	}

	otherSuperAdmin := extensionManager()
	otherSuperAdmin.ID = 99
	if err := service.ConfirmEnable(context.Background(), otherSuperAdmin, extension, challenge.Token); !errors.Is(err, ErrTrustChallengeInvalid) {
		t.Fatalf("wrong actor: %v", err)
	}
	if err := service.ConfirmEnable(context.Background(), extensionManager(), extension, challenge.Token); err != nil {
		t.Fatal(err)
	}
	if err := trustStore.RevokeAll(context.Background(), extension.ID, extensionManager().ID, "test"); err != nil {
		t.Fatal(err)
	}
	if err := service.ConfirmEnable(context.Background(), extensionManager(), extension, challenge.Token); !errors.Is(err, ErrTrustChallengeReplayed) {
		t.Fatalf("replayed token: %v", err)
	}
}

func TestExecutableTrustChallengeRejectsExpiredAndChangedImpact(t *testing.T) {
	extension := exactTrustExtension(t, "demo.expiring")
	extensions := &fakeExtensionStore{items: map[string]Extension{extension.ID: extension}}
	now := time.Date(2026, 7, 13, 12, 0, 0, 0, time.UTC)
	trustStore := &memoryExecutableTrustStore{now: func() time.Time { return now }}
	service := NewExecutableTrustService(extensions, trustStore)
	service.now = func() time.Time { return now }

	challenge, err := service.Challenge(context.Background(), extensionManager(), extension.ID)
	if err != nil {
		t.Fatal(err)
	}
	now = now.Add(DefaultTrustChallengeTTL + time.Second)
	if err := service.ConfirmEnable(context.Background(), extensionManager(), extension, challenge.Token); !errors.Is(err, ErrTrustChallengeExpired) {
		t.Fatalf("expired token: %v", err)
	}

	now = now.Add(time.Second)
	challenge, err = service.Challenge(context.Background(), extensionManager(), extension.ID)
	if err != nil {
		t.Fatal(err)
	}
	changed := extension
	changed.Manifest.Routes = append(changed.Manifest.Routes, ManifestRoute{Path: "/changed", Methods: []string{"POST"}, Access: RouteAccessLogin})
	if err := service.ConfirmEnable(context.Background(), extensionManager(), changed, challenge.Token); !errors.Is(err, ErrTrustChallengeStale) {
		t.Fatalf("changed impact: %v", err)
	}
}

func TestExecutableTrustChallengeRejectsEveryExecutableDigestChange(t *testing.T) {
	tests := []struct {
		name    string
		prepare func(*testing.T, *Extension)
		change  func(*testing.T, *Extension)
	}{
		{
			name: "backend bytes and package digest",
			change: func(t *testing.T, extension *Extension) {
				writeTrustFile(t, extension, "backend/plugin", "changed-plugin-binary", 0o755)
				refreshTrustPackageIdentity(t, extension)
			},
		},
		{
			name: "migration bytes",
			change: func(t *testing.T, extension *Extension) {
				writeTrustFile(t, extension, "migrations/001.sql", "SELECT 2;", 0o600)
				refreshTrustPackageIdentity(t, extension)
			},
		},
		{
			name: "migration declaration",
			change: func(t *testing.T, extension *Extension) {
				writeTrustFile(t, extension, "migrations/002.sql", "SELECT 2;", 0o600)
				extension.Manifest.Migrations = append(extension.Manifest.Migrations, ManifestMigration{Path: "migrations/002.sql"})
				refreshTrustPackageIdentity(t, extension)
			},
		},
		{
			name: "admin frontend bytes",
			prepare: func(t *testing.T, extension *Extension) {
				addTrustAdminFrontend(t, extension)
			},
			change: func(t *testing.T, extension *Extension) {
				writeTrustFile(t, extension, "frontend/admin/dist/settings.mjs", "export function mount() { return () => {} }\n", 0o600)
				refreshTrustPackageIdentity(t, extension)
			},
		},
		{
			name: "admin frontend contract",
			prepare: func(t *testing.T, extension *Extension) {
				addTrustAdminFrontend(t, extension)
			},
			change: func(t *testing.T, extension *Extension) {
				extension.Manifest.SettingsDocument.UI.Component.APIVersion = 2
				refreshTrustPackageIdentity(t, extension)
			},
		},
		{
			name: "requested network authority",
			change: func(t *testing.T, extension *Extension) {
				extension.Manifest.Capabilities = nil
				refreshTrustPackageIdentity(t, extension)
			},
		},
		{
			name: "permission declaration",
			change: func(t *testing.T, extension *Extension) {
				extension.Manifest.Permissions = append(extension.Manifest.Permissions, "demo.use")
				refreshTrustPackageIdentity(t, extension)
			},
		},
		{
			name: "required feature declaration",
			change: func(t *testing.T, extension *Extension) {
				extension.Manifest.RequiresFeatures = append(extension.Manifest.RequiresFeatures, "features.demo")
				refreshTrustPackageIdentity(t, extension)
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			extension := exactTrustExtension(t, "demo.changed")
			if test.prepare != nil {
				test.prepare(t, &extension)
			}
			store := &fakeExtensionStore{items: map[string]Extension{extension.ID: extension}}
			trustStore := &memoryExecutableTrustStore{}
			service := NewExecutableTrustService(store, trustStore)
			challenge, err := service.Challenge(context.Background(), extensionManager(), extension.ID)
			if err != nil {
				t.Fatal(err)
			}

			test.change(t, &extension)
			if err := service.ConfirmEnable(context.Background(), extensionManager(), extension, challenge.Token); !errors.Is(err, ErrTrustChallengeStale) {
				t.Fatalf("changed exact artifact must be stale, got %v", err)
			}
		})
	}
}

func TestCanonicalTrustImpactDigestBindsAuthorityContractsAndDependencies(t *testing.T) {
	base, err := buildTrustImpact(exactTrustExtension(t, "demo.contracts"), TrustActionEnable)
	if err != nil {
		t.Fatal(err)
	}
	tests := []struct {
		name   string
		change func(*TrustImpact)
	}{
		{name: "raw request authority", change: func(impact *TrustImpact) { impact.RequestedAuthority.RawRequest = true }},
		{name: "raw core database authority", change: func(impact *TrustImpact) { impact.RequestedAuthority.RawCoreDatabase = true }},
		{name: "host contract", change: func(impact *TrustImpact) { impact.Contracts.HostAPI = "sforum.host/v2" }},
		{name: "frontend contract", change: func(impact *TrustImpact) { impact.Contracts.FrontendAPI = "sforum.component/v2" }},
		{name: "dependency", change: func(impact *TrustImpact) {
			impact.Dependencies = []Dependency{{Name: "demo.parent", Version: "^2.0.0", Integrity: "sha256:demo"}}
		}},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			changed := base
			test.change(&changed)
			digest, err := canonicalTrustImpactDigest(changed)
			if err != nil {
				t.Fatal(err)
			}
			if digest == base.Digest {
				t.Fatalf("%s change did not invalidate impact digest", test.name)
			}
		})
	}
}

func TestServiceV3TrustAllowsAlreadyGrantedDelegatedEnable(t *testing.T) {
	extension := exactTrustExtension(t, "demo.enable")
	store := &fakeExtensionStore{items: map[string]Extension{extension.ID: extension}}
	trustStore := &memoryExecutableTrustStore{now: func() time.Time { return time.Now().UTC() }}
	trust := NewExecutableTrustService(store, trustStore)
	service := NewServiceWithOptions(store, t.TempDir(), "", &fakeRuntimeManager{}, WithExecutableTrust(trust, true))

	if _, err := service.Enable(context.Background(), extensionManager(), extension.ID, EnableInput{}); !errors.Is(err, ErrTrustChallengeRequired) {
		t.Fatalf("enable without challenge: %v", err)
	}
	challenge, err := service.IssueExecutableTrustChallenge(context.Background(), extensionManager(), extension.ID)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := service.Enable(context.Background(), extensionManager(), extension.ID, EnableInput{ConfirmationToken: challenge.Token}); err != nil {
		t.Fatal(err)
	}
	if _, err := service.Disable(context.Background(), techAdminPluginManager(), extension.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := service.Enable(context.Background(), techAdminPluginManager(), extension.ID, EnableInput{}); err != nil {
		t.Fatalf("delegated manager should operate an already trusted digest: %v", err)
	}
}

func TestV3StaticInstallByDelegatedManagerDoesNotExecutePackage(t *testing.T) {
	store := &fakeExtensionStore{items: map[string]Extension{}}
	runtime := &countingRuntimeManager{}
	service := NewServiceWithOptions(store, t.TempDir(), "", runtime, WithExecutableTrust(NewExecutableTrustService(store, &memoryExecutableTrustStore{}), true))
	archive := extensionArchive(t, validManifest("delegated.backend", TypePlugin),
		zipFile{name: "backend/plugin", body: "binary", mode: 0o755},
		zipFile{name: "migrations/001_init.sql", body: "SELECT 1", mode: 0o644},
	)

	result, err := service.InstallOrUpgradeArchive(context.Background(), techAdminPluginManager(), ArchiveInput{FileName: "plugin.zip", Data: archive})
	if err != nil {
		t.Fatal(err)
	}
	if result.Extension.Status != StatusInstalled || runtime.checks != 0 || runtime.starts != 0 {
		t.Fatalf("static install executed package code: result=%#v checks=%d starts=%d", result, runtime.checks, runtime.starts)
	}
}

func exactTrustExtension(t *testing.T, id string) Extension {
	t.Helper()
	item := installedExtension(id, TypePlugin, ManifestBackend{Entry: "backend/plugin", RPC: "hashicorp-go-plugin", ProtocolVersion: 1})
	item.Source = SourceUploaded
	item.IsDeletable = true
	item.Manifest.Routes = []ManifestRoute{{Path: "/run", Methods: []string{"POST"}, Access: RouteAccessPermission, Permission: "topic.create"}}
	item.Manifest.Migrations = []ManifestMigration{{Path: "migrations/001.sql"}}
	item.Manifest.Capabilities = []string{"net.outbound"}
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "backend"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(root, "migrations"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "backend/plugin"), []byte("plugin-binary"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "migrations/001.sql"), []byte("SELECT 1;"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := writeManifest(root, item.Manifest); err != nil {
		t.Fatal(err)
	}
	digest, err := extensionpackage.DigestTree(root)
	if err != nil {
		t.Fatal(err)
	}
	item.PackagePath = root
	item.PackageDigest = digest
	return item
}

func addTrustAdminFrontend(t *testing.T, extension *Extension) {
	t.Helper()
	component := &SettingsComponent{ID: "settings", APIVersion: 1, Entry: "frontend/admin/dist/settings.mjs"}
	extension.Manifest.Settings = []ManifestSetting{{Key: "title", Type: "text", Label: LocalizedText{Default: "Title"}}}
	extension.Manifest.SettingsDocument = SettingsDocument{
		SchemaVersion: 1,
		Explicit:      true,
		UI:            SettingsUI{Mode: "component", Layout: "form", Component: component},
		Fields:        extension.Manifest.Settings,
	}
	writeTrustFile(t, extension, component.Entry, "export function mount() {}\n", 0o600)
	refreshTrustPackageIdentity(t, extension)
}

func writeTrustFile(t *testing.T, extension *Extension, relative, body string, mode os.FileMode) {
	t.Helper()
	target := filepath.Join(extension.PackagePath, filepath.FromSlash(relative))
	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(target, []byte(body), mode); err != nil {
		t.Fatal(err)
	}
}

func refreshTrustPackageIdentity(t *testing.T, extension *Extension) {
	t.Helper()
	if err := writeManifest(extension.PackagePath, extension.Manifest); err != nil {
		t.Fatal(err)
	}
	if component := extension.Manifest.SettingsDocument.UI.Component; component != nil && component.Entry != "" {
		digest, err := ComputeAdminFrontendDigest(extension.Manifest, extension.PackagePath)
		if err != nil {
			t.Fatal(err)
		}
		extension.AdminFrontendDigest = digest
	}
	digest, err := extensionpackage.DigestTree(extension.PackagePath)
	if err != nil {
		t.Fatal(err)
	}
	extension.PackageDigest = digest
}

type memoryExecutableTrustStore struct {
	mu        sync.Mutex
	now       func() time.Time
	challenge TrustChallengeRecord
	consumed  bool
	grants    map[TrustIdentity]bool
}

func (s *memoryExecutableTrustStore) CreateChallenge(_ context.Context, input TrustChallengeRecord) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.challenge = input
	s.consumed = false
	return nil
}

func (s *memoryExecutableTrustStore) HasLiveGrant(_ context.Context, identity TrustIdentity) (bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.grants[identity], nil
}

func (s *memoryExecutableTrustStore) ConsumeChallenge(_ context.Context, input TrustConsumeInput) (TrustGrant, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if input.TokenHash == "" || input.TokenHash != s.challenge.TokenHash || input.ActorUserID != s.challenge.ActorUserID {
		return TrustGrant{}, ErrTrustChallengeInvalid
	}
	if s.consumed {
		return TrustGrant{}, ErrTrustChallengeReplayed
	}
	now := time.Now().UTC()
	if s.now != nil {
		now = s.now()
	}
	if !now.Before(s.challenge.ExpiresAt) {
		return TrustGrant{}, ErrTrustChallengeExpired
	}
	if input.Identity != s.challenge.Identity {
		return TrustGrant{}, ErrTrustChallengeStale
	}
	s.consumed = true
	if s.grants == nil {
		s.grants = map[TrustIdentity]bool{}
	}
	s.grants[input.Identity] = true
	return TrustGrant{ExtensionID: input.Identity.ExtensionID, ImpactDigest: input.Identity.ImpactDigest}, nil
}

func (s *memoryExecutableTrustStore) RevokeAll(_ context.Context, extensionID string, _ int64, _ string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	for identity := range s.grants {
		if identity.ExtensionID == extensionID {
			delete(s.grants, identity)
		}
	}
	return nil
}

type countingRuntimeManager struct {
	checks int
	starts int
}

func (m *countingRuntimeManager) Check(context.Context, Extension) error {
	m.checks++
	return nil
}

func (m *countingRuntimeManager) Start(context.Context, Extension) error {
	m.starts++
	return nil
}

func (*countingRuntimeManager) Stop(context.Context, Extension) error { return nil }
func (*countingRuntimeManager) Status(context.Context, Extension) RuntimeStatus {
	return RuntimeStatus{State: RuntimeStopped}
}
func (*countingRuntimeManager) EmitHook(context.Context, string, map[string]any) {}
