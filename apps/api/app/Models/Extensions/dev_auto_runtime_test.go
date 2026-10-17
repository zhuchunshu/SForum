package extensions

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"strings"
	"testing"

	identity "github.com/zhuchunshu/sforum/apps/api/app/Models/Identity"
)

// DevAutoRuntime 的行为测试：彻底 0 命令 dev 自动激活的三个核心语义——
// 首次安装自动启用、已启用包热更升级、操作者显式禁用的包不被翻回。

type devAutoFakeIdentity struct {
	byEmail    map[string]identity.CurrentUser
	nextID     int64
	loadCalled int
}

func newDevAutoFakeIdentity() *devAutoFakeIdentity {
	return &devAutoFakeIdentity{byEmail: map[string]identity.CurrentUser{}, nextID: 100}
}

func (s *devAutoFakeIdentity) LoadActor(_ context.Context, userID int64) (identity.Actor, error) {
	s.loadCalled++
	for _, user := range s.byEmail {
		if user.ID == userID {
			return identity.Actor{
				ID: userID, Status: identity.UserStatusActive,
				RoleKeys: []string{identity.RoleSuperAdmin},
			}, nil
		}
	}
	return identity.Actor{}, identity.ErrUserNotFound
}

func (s *devAutoFakeIdentity) GetCurrentUserByEmail(_ context.Context, email string) (identity.CurrentUser, error) {
	if user, ok := s.byEmail[email]; ok {
		return user, nil
	}
	return identity.CurrentUser{}, identity.ErrUserNotFound
}

func (s *devAutoFakeIdentity) WithBootstrapTx(
	ctx context.Context, fn func(context.Context, identity.TxStore) error,
) error {
	// 简化：真实事务语义由 identity store 测试覆盖；这里直接执行（回调用 TxStore，
	// fake 里用同一自身实现 Cast，不真正走 identity.TxStore——为避免依赖其全接口，
	// 该测试只验证 Runtime 行为；创建用户经由注入的创建函数模拟）。
	return fn(ctx, devAutoTxStoreAdapter{s})
}

type devAutoTxStoreAdapter struct{ inner *devAutoFakeIdentity }

func (a devAutoTxStoreAdapter) CreateUser(_ context.Context, input identity.CreateUserInput) (identity.CurrentUser, error) {
	if _, exists := a.inner.byEmail[input.Email]; exists {
		return identity.CurrentUser{}, errors.New("duplicate email")
	}
	a.inner.nextID++
	user := identity.CurrentUser{ID: a.inner.nextID, Username: input.Username, Status: identity.UserStatusActive}
	a.inner.byEmail[input.Email] = user
	return user, nil
}

func (a devAutoTxStoreAdapter) GetRole(_ context.Context, roleKey string) (identity.Role, error) {
	return identity.Role{ID: 1, Key: roleKey}, nil
}

func (a devAutoTxStoreAdapter) AssignRole(_ context.Context, userID int64, _ int64) error {
	// 角色信息在 fake LoadActor 中按 ID 硬编码 super_admin，无需记录。
	return nil
}

type devAutoFakePromoter struct {
	enabledIDs  []string
	upgradedIDs []string
}

func (p *devAutoFakePromoter) Enable(_ context.Context, _ identity.Actor, id string, _ EnableInput) (Extension, error) {
	p.enabledIDs = append(p.enabledIDs, id)
	return Extension{ID: id, Status: StatusEnabled}, nil
}

func (p *devAutoFakePromoter) Upgrade(_ context.Context, _ identity.Actor, id string, _ UpgradeInput) (Extension, error) {
	p.upgradedIDs = append(p.upgradedIDs, id)
	return Extension{ID: id, Status: StatusEnabled}, nil
}

type devAutoFakeGrants struct{ ensured []string }

func (g *devAutoFakeGrants) EnsureLiveGrantForDevAuto(_ context.Context, _ identity.Actor, extension Extension) error {
	g.ensured = append(g.ensured, extension.ID+"@"+extension.PackageDigest)
	return nil
}

func newDevAutoTestRuntime(t *testing.T, items map[string]Extension) (
	*DevAutoRuntime, *devAutoFakePromoter, *devAutoFakeGrants, *devAutoFakeIdentity,
) {
	t.Helper()
	promoter := &devAutoFakePromoter{}
	grants := &devAutoFakeGrants{}
	identities := newDevAutoFakeIdentity()
	base := NewService(&fakeExtensionStore{items: items}, t.TempDir())
	runtime := &DevAutoRuntime{
		service:    base,
		promoter:   promoter,
		identities: identities,
		grants:     grants,
		logger:     discardLogger(),
	}
	return runtime, promoter, grants, identities
}

func TestDevAutoRuntimeEnablesFreshlyInstalledExtension(t *testing.T) {
	extension := devTrustTestExtension(t)
	extension.Status = StatusInstalled
	runtime, promoter, grants, _ := newDevAutoTestRuntime(t, map[string]Extension{extension.ID: extension})
	runtime.promoteOneForTest(t, extension.ID)

	if len(promoter.enabledIDs) != 1 || promoter.enabledIDs[0] != extension.ID {
		t.Fatalf("enabled = %#v; want [%s]", promoter.enabledIDs, extension.ID)
	}
	if len(grants.ensured) != 1 || !strings.HasPrefix(grants.ensured[0], extension.ID+"@") {
		t.Fatalf("grants = %#v", grants.ensured)
	}
}

func TestDevAutoRuntimeUpgradesEnabledExtensionWithStagedDigest(t *testing.T) {
	extension := devTrustTestExtension(t)
	extension.Status = StatusEnabled
	staged := devTrustTestExtension(t)
	extension.StagedVersion = &ExtensionVersion{
		ID: 77, Version: "2.0.0", PackageDigest: staged.PackageDigest,
		PackagePath: staged.PackagePath, Manifest: staged.Manifest,
	}
	runtime, promoter, grants, _ := newDevAutoTestRuntime(t, map[string]Extension{extension.ID: extension})
	runtime.promoteOneForTest(t, extension.ID)

	if len(promoter.upgradedIDs) != 1 || promoter.upgradedIDs[0] != extension.ID {
		t.Fatalf("upgraded = %#v", promoter.upgradedIDs)
	}
	if len(promoter.enabledIDs) != 0 {
		t.Fatalf("enabled unexpectedly: %#v", promoter.enabledIDs)
	}
	if len(grants.ensured) != 1 || grants.ensured[0] != extension.ID+"@"+staged.PackageDigest {
		t.Fatalf("grant must target staged digest: %#v", grants.ensured)
	}
}

func TestDevAutoRuntimeRespectsDisabledState(t *testing.T) {
	extension := devTrustTestExtension(t)
	extension.Status = StatusDisabled
	extension.StagedVersion = &ExtensionVersion{ID: 77, Version: "2.0.0", PackageDigest: "abc"}
	runtime, promoter, grants, _ := newDevAutoTestRuntime(t, map[string]Extension{extension.ID: extension})
	runtime.promoteOneForTest(t, extension.ID)

	if len(promoter.enabledIDs)+len(promoter.upgradedIDs) != 0 {
		t.Fatalf("disabled extension must not be touched: %#v", promoter)
	}
	if len(grants.ensured) != 0 {
		t.Fatalf("disabled extension must not be granted: %#v", grants.ensured)
	}
}

func TestDevAutoRuntimeCreatesOperatorOnceAndReusesIt(t *testing.T) {
	extension := devTrustTestExtension(t)
	extension.Status = StatusInstalled
	runtime, _, _, identities := newDevAutoTestRuntime(t, map[string]Extension{extension.ID: extension})
	runtime.promoteOneForTest(t, extension.ID)
	runtime.promoteOneForTest(t, extension.ID)

	// 两次 promote 共享同一 dev-auto 用户（第二此后 LoadActor 缓存命中）。
	if len(identities.byEmail) != 1 {
		t.Fatalf("identity users = %#v; want exactly one dev-auto account", identities.byEmail)
	}
	if _, ok := identities.byEmail[DevAutoEmail]; !ok {
		t.Fatalf("dev-auto account missing: %#v", identities.byEmail)
	}
}

func discardLogger() *slog.Logger { return slog.New(slog.NewTextHandler(io.Discard, nil)) }

// promoteOneForTest 直接驱动单包激活（测试入口；生产入口是 PromoteAll）。
func (r *DevAutoRuntime) promoteOneForTest(t *testing.T, extensionID string) {
	t.Helper()
	_ = r.promoteOne(context.Background(), extensionID)
}

func (a devAutoTxStoreAdapter) AnyUserExists(context.Context) (bool, error) { return false, nil }
func (a devAutoTxStoreAdapter) FindRegistrationConflicts(context.Context, string, string) (identity.RegistrationConflicts, error) {
	return identity.RegistrationConflicts{}, nil
}
func (a devAutoTxStoreAdapter) CreateCredential(context.Context, int64, string) error { return nil }
func (a devAutoTxStoreAdapter) GetDefaultRole(context.Context) (identity.Role, error) {
	return identity.Role{ID: 2, Key: "member"}, nil
}
func (a devAutoTxStoreAdapter) LoadCurrentUserAccess(context.Context, *identity.CurrentUser) error {
	return nil
}
