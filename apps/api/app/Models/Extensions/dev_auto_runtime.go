package extensions

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"sort"

	identity "github.com/zhuchunshu/sforum/apps/api/app/Models/Identity"
)

// DevAutoRuntime 是「彻底 0 命令」dev 模式的 host 侧自动激活服务。
//
// 语义（操作者 2026-10-17 拍板，见 knowledge/decisions/2026-10-17-dev-zero-command.md）：
// APP_ENV=development 且非 Safe Mode 时，dev 来源根（EXTERNAL_EXTENSION_ROOTS ∪
// extensions/dev 脚手架树）下的插件由 host 自己完成 安装→启用→逐摘要升级，
// 不需要任何登录身份、信任会话或人工确认。生产装配不注入本服务，全部路径
// fail closed（super_admin 门原样）。
//
// 合成身份：dev-auto 系统账号（无密码凭据，不可登录）。lifecycle/authority/audit
// 链要求合法 UserID（V2 authority 多处 ActorUserID>0 校验），因此不用 ID=0。
// 该账号在管理端可见，是「dev 自动机」的显式审计归因；在管理端禁用/删除它
// 即等于关闭本通道。
const (
	DevAutoUsername    = "dev-auto"
	DevAutoEmail       = "dev-auto@sforum.invalid"
	DevAutoDisplayName = "SForum dev auto"
)

// DevIdentityStore 是本服务需要的最小身份写入边界（identity.Store 的窄投影）。
type DevIdentityStore interface {
	identity.ActorStore
	GetCurrentUserByEmail(ctx context.Context, email string) (identity.CurrentUser, error)
	WithBootstrapTx(ctx context.Context, fn func(context.Context, identity.TxStore) error) error
}

// DevAutoPromoter 是升级/启用的窄边界（*Service 自身实现，包内直连避免环）。
type DevAutoPromoter interface {
	Enable(ctx context.Context, actor identity.Actor, id string, input EnableInput) (Extension, error)
	Upgrade(ctx context.Context, actor identity.Actor, id string, input UpgradeInput) (Extension, error)
}

// DevTrustGrantWriter 是 exact live grant 的写入边界（*ExecutableTrustService）。
type DevTrustGrantWriter interface {
	EnsureLiveGrantForDevAuto(ctx context.Context, actor identity.Actor, extension Extension) error
}

type DevAutoRuntime struct {
	service    *Service
	promoter   DevAutoPromoter
	identities DevIdentityStore
	grants     DevTrustGrantWriter
	// sourceRoots 与 CatalogService.externalRoots 同源；用于日志与防御性过滤。
	sourceRoots []string
	logger      *slog.Logger
	// operatorUser 缓存 dev-auto 账号 id（进程内；不存在则惰性创建）。
	operatorUserID int64
}

func NewDevAutoRuntime(
	service *Service,
	promoter DevAutoPromoter,
	identities DevIdentityStore,
	grants DevTrustGrantWriter,
	sourceRoots []string,
	logger *slog.Logger,
) *DevAutoRuntime {
	if logger == nil {
		logger = slog.Default()
	}
	return &DevAutoRuntime{
		service: service, promoter: promoter, identities: identities,
		grants: grants, sourceRoots: sourceRoots, logger: logger,
	}
}

// PromoteAll 是 watcher 的唯一入口：重扫外部源 → 对每个 dev 来源包确保
// grant + 启用/升级。单包失败只记日志，不中断其余包（dev 容错语义）。
func (r *DevAutoRuntime) PromoteAll(ctx context.Context) {
	if r == nil || r.service == nil {
		return
	}
	result, err := r.service.SyncExternalSources(ctx)
	if err != nil {
		r.logger.WarnContext(ctx, "dev auto: external source resync failed", "error", err)
		return
	}
	for _, diagnostic := range result.Diagnostics {
		r.logger.WarnContext(ctx, "dev auto: external source skipped",
			"code", diagnostic.Code, "packagePath", diagnostic.PackagePath,
			"extensionId", diagnostic.ExtensionID, "reason", diagnostic.Message)
	}
	items := append([]Extension(nil), result.Items...)
	sort.Slice(items, func(i, j int) bool { return items[i].ID < items[j].ID })
	for _, item := range items {
		if err := r.promoteOne(ctx, item.ID); err != nil {
			r.logger.WarnContext(ctx, "dev auto: promote failed",
				"extensionId", item.ID, "error", err)
		}
	}
}

func (r *DevAutoRuntime) promoteOne(ctx context.Context, extensionID string) error {
	actor, err := r.ensureDevAutoActor(ctx)
	if err != nil {
		return fmt.Errorf("ensure dev-auto actor: %w", err)
	}
	extension, err := r.service.store.Get(ctx, normalizeID(extensionID))
	if err != nil {
		return err
	}
	if extension.Source == SourceBuiltin || extension.IsSystem {
		return nil
	}
	// 操作者显式禁用的包尊重其状态：不 grant、不升级（重新启用即恢复自动化）。
	if extension.Status == StatusDisabled {
		return nil
	}
	// 精确制品 grant：dev 直通的信任语义仍然是「逐摘要」，只是执行者是 dev 自动机。
	if RequiresExecutableTrust(extension) && r.grants != nil {
		target := extension
		if staged, ok := extension.StagedArtifact(); ok && extension.Status != StatusDisabled {
			target = staged
		}
		if err := r.grants.EnsureLiveGrantForDevAuto(ctx, actor, target); err != nil {
			return fmt.Errorf("ensure dev grant: %w", err)
		}
	}
	switch extension.Status {
	case StatusInstalled:
		_, err = r.promoter.Enable(ctx, actor, extension.ID, EnableInput{ConfirmCapabilities: true})
	case StatusEnabled:
		if extension.StagedVersion == nil {
			return nil
		}
		_, err = r.promoter.Upgrade(ctx, actor, extension.ID, UpgradeInput{})
	default:
		return nil
	}
	if err != nil {
		return err
	}
	r.logger.InfoContext(ctx, "dev auto: extension promoted",
		"extensionId", extension.ID, "digest", func() string {
			if extension.StagedVersion != nil {
				return extension.StagedVersion.PackageDigest
			}
			return extension.PackageDigest
		}())
	return nil
}

// ensureDevAutoActor 返回（必要时创建）dev-auto 系统账号的 actor。
// 账号无密码凭据（不调用 CreateCredential），不可登录；幂等。
func (r *DevAutoRuntime) ensureDevAutoActor(ctx context.Context) (identity.Actor, error) {
	if r.operatorUserID > 0 {
		return r.identities.LoadActor(ctx, r.operatorUserID)
	}
	if current, err := r.identities.GetCurrentUserByEmail(ctx, DevAutoEmail); err == nil {
		r.operatorUserID = current.ID
		return r.identities.LoadActor(ctx, current.ID)
	} else if !errors.Is(err, identity.ErrUserNotFound) {
		return identity.Actor{}, err
	}
	var userID int64
	txErr := r.identities.WithBootstrapTx(ctx, func(ctx context.Context, tx identity.TxStore) error {
		current, createErr := tx.CreateUser(ctx, identity.CreateUserInput{
			Username:      DevAutoUsername,
			Email:         DevAutoEmail,
			DisplayName:   DevAutoDisplayName,
			EmailVerified: true,
		})
		if createErr != nil {
			// 并发创建（双实例 dev）时唯一索引冲突：按已存在处理。
			existing, getErr := r.identities.GetCurrentUserByEmail(ctx, DevAutoEmail)
			if getErr != nil {
				return createErr
			}
			userID = existing.ID
			return nil
		}
		superAdmin, roleErr := tx.GetRole(ctx, identity.RoleSuperAdmin)
		if roleErr != nil {
			return roleErr
		}
		if err := tx.AssignRole(ctx, current.ID, superAdmin.ID); err != nil {
			return err
		}
		userID = current.ID
		return nil
	})
	if txErr != nil {
		return identity.Actor{}, txErr
	}
	r.operatorUserID = userID
	// 不调用 CreateCredential：无密码凭据 → GetCredentialByLogin 找不到 → 不可登录。
	return r.identities.LoadActor(ctx, userID)
}

// withStoreForTests 返回共享同一 store 的副本（测试专用；生产 store 随装配持有）。
func (s *Service) withStoreForTests(store Store) *Service {
	clone := *s
	clone.store = store
	return &clone
}
