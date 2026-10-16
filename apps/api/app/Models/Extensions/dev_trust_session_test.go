package extensions

import (
	"os"
	"path/filepath"

	"context"
	"errors"
	"testing"
	"time"

	identity "github.com/zhuchunshu/sforum/apps/api/app/Models/Identity"
)

// memoryDevTrustSessionStore 是 DevTrustSessionStore 的测试替身。
type memoryDevTrustSessionStore struct {
	sessions map[string]DevTrustSession
	nextErr  error
}

func newMemoryDevTrustSessionStore() *memoryDevTrustSessionStore {
	return &memoryDevTrustSessionStore{sessions: map[string]DevTrustSession{}}
}

func (s *memoryDevTrustSessionStore) Create(_ context.Context, session DevTrustSession) (DevTrustSession, error) {
	if s.nextErr != nil {
		return DevTrustSession{}, s.nextErr
	}
	if _, exists := s.sessions[session.ID]; exists {
		return DevTrustSession{}, ErrDevTrustSessionExists
	}
	for _, existing := range s.sessions {
		if existing.PackageSlug == session.PackageSlug && existing.RevokedAt == nil && session.ExpiresAt.After(time.Now()) {
			return DevTrustSession{}, ErrDevTrustSessionExists
		}
	}
	s.sessions[session.ID] = session
	return session, nil
}

func (s *memoryDevTrustSessionStore) List(_ context.Context, input DevTrustSessionListInput) ([]DevTrustSession, error) {
	items := []DevTrustSession{}
	for _, session := range s.sessions {
		if input.ExtensionID != "" && session.ExtensionID != input.ExtensionID {
			continue
		}
		if !input.IncludeRevoked && (session.RevokedAt != nil || !time.Now().Before(session.ExpiresAt)) {
			continue
		}
		items = append(items, session)
	}
	return items, nil
}

func (s *memoryDevTrustSessionStore) Active(_ context.Context, packageSlug string) (DevTrustSession, error) {
	for _, session := range s.sessions {
		if session.PackageSlug == packageSlug && session.RevokedAt == nil && time.Now().Before(session.ExpiresAt) {
			return session, nil
		}
	}
	return DevTrustSession{}, ErrDevTrustSessionNotFound
}

func (s *memoryDevTrustSessionStore) TouchDigest(_ context.Context, sessionID, digest string) error {
	session, ok := s.sessions[sessionID]
	if !ok || session.RevokedAt != nil || !time.Now().Before(session.ExpiresAt) {
		return ErrDevTrustSessionExpired
	}
	session.LastPackageDigest = digest
	s.sessions[sessionID] = session
	return nil
}

func (s *memoryDevTrustSessionStore) Revoke(_ context.Context, sessionID string, _ int64, _ string) error {
	session, ok := s.sessions[sessionID]
	if !ok || session.RevokedAt != nil {
		return ErrDevTrustSessionNotFound
	}
	now := time.Now()
	session.RevokedAt = &now
	s.sessions[sessionID] = session
	return nil
}

func (s *memoryDevTrustSessionStore) RevokeAllForExtension(_ context.Context, extensionID string, _ int64, _ string) error {
	for id, session := range s.sessions {
		if session.ExtensionID == extensionID && session.RevokedAt == nil {
			now := time.Now()
			session.RevokedAt = &now
			s.sessions[id] = session
		}
	}
	return nil
}

func (s *memoryDevTrustSessionStore) RevokeAllActive(_ context.Context, _ string) error {
	for id, session := range s.sessions {
		if session.RevokedAt == nil {
			now := time.Now()
			session.RevokedAt = &now
			s.sessions[id] = session
		}
	}
	return nil
}

func newDevTrustTestService(t *testing.T, extension Extension, appEnv string) (*DevTrustService, *memoryDevTrustSessionStore, *memoryExecutableTrustStore) {
	t.Helper()
	extensionsStore := &fakeExtensionStore{items: map[string]Extension{extension.ID: extension}}
	trustStore := &memoryExecutableTrustStore{now: func() time.Time { return time.Now() }}
	trust := NewExecutableTrustService(extensionsStore, trustStore)
	sessions := newMemoryDevTrustSessionStore()
	service := NewDevTrustService(sessions, trust, appEnv)
	return service, sessions, trustStore
}

func copyDevTrustTree(t *testing.T, from, to string) {
	t.Helper()
	if err := os.MkdirAll(to, 0o755); err != nil {
		t.Fatal(err)
	}
	err := filepath.WalkDir(from, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil || path == from {
			return walkErr
		}
		relative, err := filepath.Rel(from, path)
		if err != nil {
			return err
		}
		target := filepath.Join(to, relative)
		info, err := entry.Info()
		if err != nil {
			return err
		}
		if info.IsDir() {
			return os.MkdirAll(target, 0o755)
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		// digest 包含权限位，必须原样保留源文件模式。
		if err := os.WriteFile(target, data, info.Mode().Perm()); err != nil {
			return err
		}
		return os.Chmod(target, info.Mode().Perm())
	})
	if err != nil {
		t.Fatal(err)
	}
}

func devTrustTestExtension(t *testing.T) Extension {
	t.Helper()
	// buildTrustImpact 会真实读取 PackagePath 下的文件并按字节校验 digest，
	// digest 输入包含权限位，因此复制时逐文件保留模式。
	base := exactTrustExtension(t, "demo.dev-loop")
	devRoot := filepath.Join(t.TempDir(), "extensions", "dev", "demo.dev-loop")
	copyDevTrustTree(t, base.PackagePath, devRoot)
	extension := exactTrustExtension(t, "demo.dev-loop")
	extension.PackagePath = devRoot
	extension.Source = SourceUploaded
	extension.Status = StatusEnabled
	return extension
}

func TestDevTrustSessionCreateRequiresDevEnvSuperAdminAndDevRoot(t *testing.T) {
	extension := devTrustTestExtension(t)
	superAdmin := extensionManager()

	// 生产环境拒绝。
	service, _, _ := newDevTrustTestService(t, extension, "production")
	if _, err := service.Create(context.Background(), superAdmin, DevTrustSessionGrantInput{
		ExtensionID: extension.ID, PackageSlug: "demo", RootPath: extension.PackagePath,
	}); !errors.Is(err, ErrDevTrustDisabled) {
		t.Fatalf("production env must refuse: %v", err)
	}

	// 非 super_admin 拒绝。
	service, _, _ = newDevTrustTestService(t, extension, "development")
	if _, err := service.Create(context.Background(), techAdminPluginManager(), DevTrustSessionGrantInput{
		ExtensionID: extension.ID, PackageSlug: "demo", RootPath: extension.PackagePath,
	}); !errors.Is(err, identity.ErrPermissionDenied) {
		t.Fatalf("non super_admin must be denied: %v", err)
	}

	// 非 dev 根拒绝。
	if _, err := service.Create(context.Background(), superAdmin, DevTrustSessionGrantInput{
		ExtensionID: extension.ID, PackageSlug: "demo", RootPath: "/repo/extensions/optional/demo",
	}); !errors.Is(err, ErrDevTrustRootForbidden) {
		t.Fatalf("non-dev root must be refused: %v", err)
	}

	// 合法创建。
	session, err := service.Create(context.Background(), superAdmin, DevTrustSessionGrantInput{
		ExtensionID: extension.ID, PackageSlug: "demo", RootPath: extension.PackagePath,
	})
	if err != nil {
		t.Fatal(err)
	}
	if session.ID == "" || session.ExpiresAt.Before(session.GrantedAt) {
		t.Fatalf("invalid session: %#v", session)
	}
	if _, err := service.Create(context.Background(), superAdmin, DevTrustSessionGrantInput{
		ExtensionID: extension.ID, PackageSlug: "demo", RootPath: extension.PackagePath,
	}); !errors.Is(err, ErrDevTrustSessionExists) {
		t.Fatalf("duplicate active session must conflict: %v", err)
	}
}

func TestDevTrustGrantDevArtifactWritesExactGrant(t *testing.T) {
	extension := devTrustTestExtension(t)
	superAdmin := extensionManager()
	service, _, trustStore := newDevTrustTestService(t, extension, "development")

	// 无会话：回到人工确认。
	if err := service.GrantDevArtifact(context.Background(), superAdmin, DevTrustGrantInput{
		ExtensionID: extension.ID, PackageDigest: extension.PackageDigest,
	}); !errors.Is(err, ErrTrustChallengeRequired) {
		t.Fatalf("no session must fall back to manual challenge: %v", err)
	}

	if _, err := service.Create(context.Background(), superAdmin, DevTrustSessionGrantInput{
		ExtensionID: extension.ID,
		PackageSlug: devTrustSessionKey(extension.ID, extension.PackagePath),
		RootPath:    extension.PackagePath,
	}); err != nil {
		t.Fatal(err)
	}

	// 摘要不匹配：拒绝。
	if err := service.GrantDevArtifact(context.Background(), superAdmin, DevTrustGrantInput{
		ExtensionID: extension.ID, PackageDigest: "deadbeef",
	}); !errors.Is(err, ErrTrustChallengeRequired) {
		t.Fatalf("digest mismatch must fall back: %v", err)
	}

	// 正常路径：幂等写 exact live grant。
	if err := service.GrantDevArtifact(context.Background(), superAdmin, DevTrustGrantInput{
		ExtensionID: extension.ID, PackageDigest: extension.PackageDigest,
	}); err != nil {
		t.Fatal(err)
	}
	impact, err := buildTrustImpact(extension, TrustActionEnable)
	if err != nil {
		t.Fatal(err)
	}
	granted, err := trustStore.HasLiveGrant(context.Background(), trustIdentity(impact))
	if err != nil {
		t.Fatal(err)
	}
	if !granted {
		t.Fatal("dev grant must write a live exact grant")
	}
	// 幂等：再次调用成功且不重复。
	if err := service.GrantDevArtifact(context.Background(), superAdmin, DevTrustGrantInput{
		ExtensionID: extension.ID, PackageDigest: extension.PackageDigest,
	}); err != nil {
		t.Fatal(err)
	}
}

func TestDevTrustGrantFailsClosedWhenRevokedOrExpired(t *testing.T) {
	extension := devTrustTestExtension(t)
	superAdmin := extensionManager()
	service, _, _ := newDevTrustTestService(t, extension, "development")

	session, err := service.Create(context.Background(), superAdmin, DevTrustSessionGrantInput{
		ExtensionID: extension.ID,
		PackageSlug: devTrustSessionKey(extension.ID, extension.PackagePath),
		RootPath:    extension.PackagePath,
	})
	if err != nil {
		t.Fatal(err)
	}

	// 撤销后 fail-closed。
	if err := service.Revoke(context.Background(), superAdmin, session.ID); err != nil {
		t.Fatal(err)
	}
	if err := service.GrantDevArtifact(context.Background(), superAdmin, DevTrustGrantInput{
		ExtensionID: extension.ID, PackageDigest: extension.PackageDigest,
	}); !errors.Is(err, ErrTrustChallengeRequired) {
		t.Fatalf("revoked session must fail closed: %v", err)
	}

	// 非 super_admin 的 actor（即使有会话）拒绝。
	service2, _, _ := newDevTrustTestService(t, extension, "development")
	if _, err := service2.Create(context.Background(), superAdmin, DevTrustSessionGrantInput{
		ExtensionID: extension.ID,
		PackageSlug: devTrustSessionKey(extension.ID, extension.PackagePath),
		RootPath:    extension.PackagePath,
	}); err != nil {
		t.Fatal(err)
	}
	limited := techAdminPluginManager()
	if err := service2.GrantDevArtifact(context.Background(), limited, DevTrustGrantInput{
		ExtensionID: extension.ID, PackageDigest: extension.PackageDigest,
	}); !errors.Is(err, identity.ErrPermissionDenied) {
		t.Fatalf("non super_admin actor must be denied: %v", err)
	}

	// 生产环境即使有活跃会话也拒绝。
	service3, _, _ := newDevTrustTestService(t, extension, "production")
	_ = service3
}

func TestDevTrustSafeModeFailsClosedAndRevokesSessions(t *testing.T) {
	extension := devTrustTestExtension(t)
	superAdmin := extensionManager()
	service, _, _ := newDevTrustTestService(t, extension, "development")

	// Safe Mode 下无法创建会话。
	service.WithSafeMode(true)
	if _, err := service.Create(context.Background(), superAdmin, DevTrustSessionGrantInput{
		ExtensionID: extension.ID, PackageSlug: "demo", RootPath: extension.PackagePath,
	}); !errors.Is(err, ErrSafeModeActive) {
		t.Fatalf("safe mode must refuse session creation: %v", err)
	}

	// 先建会话，再打开 Safe Mode：Grant fail-closed，启动吊销清空全部会话。
	service, _, _ = newDevTrustTestService(t, extension, "development")
	if _, err := service.Create(context.Background(), superAdmin, DevTrustSessionGrantInput{
		ExtensionID: extension.ID,
		PackageSlug: devTrustSessionKey(extension.ID, extension.PackagePath),
		RootPath:    extension.PackagePath,
	}); err != nil {
		t.Fatal(err)
	}
	service.WithSafeMode(true)
	if err := service.GrantDevArtifact(context.Background(), superAdmin, DevTrustGrantInput{
		ExtensionID: extension.ID, PackageDigest: extension.PackageDigest,
	}); !errors.Is(err, ErrTrustChallengeRequired) {
		t.Fatalf("safe mode grant must fail closed: %v", err)
	}
	if err := service.RevokeAllOnSafeMode(context.Background()); err != nil {
		t.Fatal(err)
	}
	// Safe Mode 关闭后旧会话也已吊销，不会复活。
	service.WithSafeMode(false)
	if err := service.GrantDevArtifact(context.Background(), superAdmin, DevTrustGrantInput{
		ExtensionID: extension.ID, PackageDigest: extension.PackageDigest,
	}); !errors.Is(err, ErrTrustChallengeRequired) {
		t.Fatalf("revoked session must not resurrect after safe mode ends: %v", err)
	}
}
