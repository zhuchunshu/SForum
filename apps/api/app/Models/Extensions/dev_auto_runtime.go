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
	// 上一次失败的 upgrade 会留下 terminal-failed operation，fence 使下一次
	// 升级报 requires-explicit-retry；dev 通道自动走一轮 recovery(retry)。
	LifecycleOperations(ctx context.Context, actor identity.Actor, extensionID string, limit int) ([]LifecycleOperationSummary, error)
	RecoverLifecycleOperation(ctx context.Context, actor identity.Actor, extensionID string, operationID int64, input LifecycleRecoveryInput) (LifecycleOperationDetail, error)
}

// DevAssetRepublisher 是 dev 升级后的公开资产注册表重发布边界（FrontendService）。
// Host 启动才 RestoreRoutePublications；自动 upgrade 后必须显式重发布，
// 否则浏览器拿到的仍是启动时快照的旧 digest。
type DevAssetRepublisher interface {
	RepublishPublicAssetsForDev(ctx context.Context, items []Extension) error
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
	assets     DevAssetRepublisher
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
	assets DevAssetRepublisher,
	sourceRoots []string,
	logger *slog.Logger,
) *DevAutoRuntime {
	if logger == nil {
		logger = slog.Default()
	}
	return &DevAutoRuntime{
		service: service, promoter: promoter, identities: identities,
		grants: grants, assets: assets, sourceRoots: sourceRoots, logger: logger,
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
	promotedAny := false
	for _, item := range items {
		if err := r.promoteOne(ctx, item.ID); err != nil {
			r.logger.WarnContext(ctx, "dev auto: promote failed",
				"extensionId", item.ID, "error", err)
			// 失败落台账：air 场景下 watcher 日志可能进了交互终端，
			// 台账是作者唯一确定可见的地方。
			_, _ = r.service.store.CreateEvent(ctx, EventInput{
				ExtensionID: item.ID,
				Action:      EventEnabled,
				Message:     "dev auto promote failed: " + err.Error(),
			})
			continue
		}
		promotedAny = true
	}
	if promotedAny {
		r.republishAssets(ctx, items)
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
		if !usesLifecycleV2(extension) {
			// 非 V2 包（未声明 lifecycle）没有可编程的 staged 升级路径；硬调
			// Upgrade 只会得到误导性错误。如实告知作者需走管理端 restart bridge。
			return fmt.Errorf("staged digest %s requires a manual admin restart bridge (extension does not declare lifecycle V2)",
				extension.StagedVersion.PackageDigest)
		}
		// dev 通道的组件契约版本自动晋升：组件注册表拒绝「同 contractVersion
		// 下声明/内容变化」的 upgrade（component_registry_schema.go），要求作者
		// 每次改 L2 前端都手动 bump——对 dev 迭代是纯摩擦。这里在传给
		// lifecycle 的内存副本上把每个组件的 contractVersion 改写为
		// "dev-<stagedDigest 前 8 位>"：制品文件与真实版本号不变，只是让
		// registry 把每次摘要变化视为一次新的契约发布。生产路径不受影响
		// （仅本通道改写）。
		if len(extension.Manifest.Components) > 0 {
			stagedDigest := extension.StagedVersion.PackageDigest
			if stagedDigest == "" {
				stagedDigest = extension.PackageDigest
			}
			devSuffix := "dev-" + stagedDigest
			if len(devSuffix) > 11 {
				devSuffix = devSuffix[:11]
			}
			rewritten := make([]ManifestComponent, len(extension.Manifest.Components))
			copy(rewritten, extension.Manifest.Components)
			for index := range rewritten {
				rewritten[index].ContractVersion = devSuffix
			}
			extension.Manifest.Components = rewritten
		}
		// V2 upgrade 要求 stable Idempotency-Key（HTTP 从请求头取）；dev 通道用
		// extensionID+stagedDigest 派生确定性键：同摘要重放安全，新摘要即新键。
		idempotencyKey := "dev-auto/" + extension.ID + "/" + extension.StagedVersion.PackageDigest
		_, err = r.promoter.Upgrade(ctx, actor, extension.ID, UpgradeInput{
			Extension: &extension, IdempotencyKey: idempotencyKey,
		})
		if errors.Is(err, ErrLifecycleCoordinatorRetryRequired) {
			if recoverErr := r.recoverLatestFailedUpgrade(ctx, actor, extension.ID); recoverErr != nil {
				return fmt.Errorf("auto recovery of last failed upgrade: %w", recoverErr)
			}
			_, err = r.promoter.Upgrade(ctx, actor, extension.ID, UpgradeInput{
				Extension: &extension, IdempotencyKey: idempotencyKey,
			})
		}
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

// republishAssets 在任一包升级/启用后重发布公开资产注册表：Host 只在启动时
// 重建它，不重发布则浏览器仍按启动快照的旧 digest 请求资产（404/旧内容）。
func (r *DevAutoRuntime) republishAssets(ctx context.Context, items []Extension) {
	if r.assets == nil {
		return
	}
	if err := r.assets.RepublishPublicAssetsForDev(ctx, items); err != nil {
		r.logger.WarnContext(ctx, "dev auto: public asset republish failed", "error", err)
	}
}

// recoverLatestFailedUpgrade 对最近一次 terminal-failed 的 upgrade 执行一轮
// decision=retry recovery，清除 fence 使下一次升级可以继续。
func (r *DevAutoRuntime) recoverLatestFailedUpgrade(ctx context.Context, actor identity.Actor, extensionID string) error {
	operations, err := r.promoter.LifecycleOperations(ctx, actor, extensionID, 5)
	if err != nil {
		return err
	}
	for _, operation := range operations {
		if operation.Operation == string(LifecycleMachineUpgrade) && operation.TerminalResult == LifecycleTerminalFailed {
			_, err := r.promoter.RecoverLifecycleOperation(ctx, actor, extensionID, operation.ID, LifecycleRecoveryInput{
				Decision: "retry", Reason: "dev auto activation retry after failed upgrade",
			})
			return err
		}
	}
	return fmt.Errorf("no failed upgrade operation found to recover")
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
