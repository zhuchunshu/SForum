package bootstrap

import (
	"context"
	"crypto/ed25519"
	"encoding/hex"
	"fmt"
	"log/slog"
	"strconv"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"

	extensions "github.com/zhuchunshu/sforum/apps/api/app/Models/Extensions"
	identity "github.com/zhuchunshu/sforum/apps/api/app/Models/Identity"
	marketplace "github.com/zhuchunshu/sforum/apps/api/app/Support/Marketplace"
	privacy "github.com/zhuchunshu/sforum/apps/api/app/Support/Privacy"
	runtimerollout "github.com/zhuchunshu/sforum/apps/api/app/Support/RuntimeRollout"
	systemtier "github.com/zhuchunshu/sforum/apps/api/app/Support/SystemTier"
	"github.com/zhuchunshu/sforum/apps/api/config"
)

// productionP12Ops holds durable P12 operations packages bound at API boot.
type productionP12Ops struct {
	Rollout     *runtimerollout.Service
	SystemTier  *systemtier.Registry
	Marketplace *marketplace.Service
	Privacy     *privacy.Registry
	Installer   marketplace.Installer
}

// bindProductionP12Ops wires RuntimeRollout / SystemTier / Marketplace / Privacy
// onto PostgreSQL authority so multi-node and CLI recovery share one store.
// Safe Mode does not load system-tier members (checked by callers via LoadOrder).
//
// extensionService / identityStore bind real HostInstaller and RBAC — not stubs.
func bindProductionP12Ops(
	cfg config.Config,
	pool *pgxpool.Pool,
	logger *slog.Logger,
	extensionService *extensions.Service,
	identityStore *identity.PostgresStore,
) (*productionP12Ops, error) {
	if pool == nil {
		return nil, fmt.Errorf("bootstrap: P12 ops requires PostgreSQL")
	}
	if logger == nil {
		logger = slog.Default()
	}

	rolloutStore, err := runtimerollout.NewPostgresStore(pool)
	if err != nil {
		return nil, fmt.Errorf("create runtime rollout store: %w", err)
	}
	rollout := runtimerollout.NewWithStore(rolloutStore)

	tierStore, err := systemtier.NewPostgresStore(pool)
	if err != nil {
		return nil, fmt.Errorf("create system tier store: %w", err)
	}
	tier := systemtier.NewWithStore(tierStore)

	// Marketplace：生产从配置加载 Ed25519 公钥；缺公钥时生产 fail closed。
	market, err := newProductionMarketplace(cfg, logger)
	if err != nil {
		return nil, err
	}

	// HostInstaller：绑定真实 InstallArchive / lifecycle / RuntimeRollout。
	installer := newHostMarketplaceInstaller(extensionService, rollout, identityStore)
	market.BindInstaller(installer)

	// Privacy：真实 RBAC + PostgreSQL audit，禁止「actor 非空即允许」。
	privacyReg := privacy.New()
	auditStore, err := privacy.NewPostgresAuditor(pool)
	if err != nil {
		return nil, fmt.Errorf("create privacy auditor: %w", err)
	}
	privacyReg.SetAuditor(auditStore)
	privacyReg.SetPermissionCheck(func(ctx context.Context, actor, userID, operation string) error {
		return checkPrivacyPermission(ctx, identityStore, actor, userID, operation)
	})

	// Safe Mode 不在此加载 system extension 代码；仅暴露 registry 供调用方查询。
	if cfg.SafeMode {
		if order, err := tier.LoadOrder(context.Background(), true); err != nil {
			return nil, err
		} else if order != nil {
			return nil, fmt.Errorf("bootstrap: safe mode must bypass system tier")
		}
	}

	return &productionP12Ops{
		Rollout: rollout, SystemTier: tier, Marketplace: market,
		Privacy: privacyReg, Installer: installer,
	}, nil
}

func newProductionMarketplace(cfg config.Config, logger *slog.Logger) (*marketplace.Service, error) {
	prodLike := isProdLike(cfg)
	policy := marketplace.OperatorPolicy{
		AllowedChannels:      []string{marketplace.ChannelStable, marketplace.ChannelBeta},
		DirectUploadFallback: true,
		AllowUnsigned:        !prodLike,
	}
	opts := marketplace.Options{Policy: policy}

	keyHex := strings.TrimSpace(cfg.MarketplaceEd25519PublicKeyHex)
	keyID := strings.TrimSpace(cfg.MarketplaceEd25519KeyID)
	if keyHex != "" {
		raw, err := hex.DecodeString(keyHex)
		if err != nil || len(raw) != ed25519.PublicKeySize {
			return nil, fmt.Errorf("bootstrap: MARKETPLACE_ED25519_PUBLIC_KEY_HEX must be 32-byte hex")
		}
		if keyID == "" {
			keyID = "marketplace-primary"
		}
		verifier, err := marketplace.NewEd25519Verifier(keyID, ed25519.PublicKey(raw))
		if err != nil {
			return nil, fmt.Errorf("bootstrap: marketplace Ed25519 verifier: %w", err)
		}
		opts.Verifier = verifier
		logger.Info("marketplace Ed25519 public key loaded", "keyId", keyID)
	} else if prodLike {
		return nil, fmt.Errorf("bootstrap: production/staging requires MARKETPLACE_ED25519_PUBLIC_KEY_HEX")
	} else {
		logger.Warn("marketplace running without Ed25519 public key (non-production AllowUnsigned)")
	}
	return marketplace.NewWithOptions(opts), nil
}

// newHostMarketplaceInstaller binds staged install to real Extensions Service + RuntimeRollout.
func newHostMarketplaceInstaller(
	extensionService *extensions.Service,
	rollout *runtimerollout.Service,
	identityStore *identity.PostgresStore,
) *marketplace.HostInstaller {
	loadActor := func(ctx context.Context, ref string) (identity.Actor, error) {
		ref = strings.TrimSpace(ref)
		if identityStore == nil || !strings.HasPrefix(ref, "user:") {
			return identity.Actor{}, fmt.Errorf("%w: marketplace actor must be a bound user:<id>", marketplace.ErrInstall)
		}
		id, err := strconv.ParseInt(strings.TrimPrefix(ref, "user:"), 10, 64)
		if err != nil || id <= 0 {
			return identity.Actor{}, fmt.Errorf("%w: invalid marketplace actor", marketplace.ErrInstall)
		}
		actor, err := identityStore.LoadActor(ctx, id)
		if err != nil || !actor.IsSuperAdmin() {
			return identity.Actor{}, fmt.Errorf("%w: marketplace install requires an active super_admin actor", marketplace.ErrInstall)
		}
		return actor, nil
	}
	return &marketplace.HostInstaller{
		PreflightFn: func(ctx context.Context, plan marketplace.InstallPlan) error {
			if extensionService == nil {
				return fmt.Errorf("%w: extension service unavailable", marketplace.ErrInstall)
			}
			if strings.TrimSpace(plan.ExtensionID) == "" || strings.TrimSpace(plan.PackageDigest) == "" {
				return fmt.Errorf("%w: empty plan", marketplace.ErrInstall)
			}
			if _, err := loadActor(ctx, plan.Actor); err != nil {
				return err
			}
			return nil
		},
		StageFn: func(ctx context.Context, plan marketplace.InstallPlan, packageBytes []byte) (marketplace.StageResult, error) {
			if extensionService == nil {
				return marketplace.StageResult{}, fmt.Errorf("%w: extension service unavailable", marketplace.ErrInstall)
			}
			actor, err := loadActor(ctx, plan.Actor)
			if err != nil {
				return marketplace.StageResult{}, err
			}
			result, err := extensionService.InstallOrUpgradeArchive(ctx, actor, extensions.ArchiveInput{
				FileName: plan.ExtensionID + ".sforum.zip",
				Data:     packageBytes,
			})
			if err != nil {
				return marketplace.StageResult{}, err
			}
			stagedDigest := result.Extension.PackageDigest
			stagedVersion := result.Extension.Version
			if result.Extension.StagedVersion != nil {
				stagedDigest = result.Extension.StagedVersion.PackageDigest
				stagedVersion = result.Extension.StagedVersion.Version
			}
			orderIDs := make([]string, 0, len(plan.Order))
			for _, step := range plan.Order {
				orderIDs = append(orderIDs, step.ExtensionID)
			}
			// 创建 RuntimeRollout plan（真实 staged version + 节点确认入口）。
			rolloutPlanID := ""
			if rollout != nil && result.Upgraded {
				source := plan.SourceDigest
				if source == "" {
					source = result.PreviousDigest
				}
				if source != "" && stagedDigest != "" && !strings.EqualFold(source, stagedDigest) {
					rp, rpErr := rollout.CreatePlan(ctx, plan.ExtensionID, source, stagedDigest, plan.Actor, 10, 3)
					if rpErr != nil && rpErr != runtimerollout.ErrConflict {
						return marketplace.StageResult{}, rpErr
					}
					if rpErr == nil {
						rolloutPlanID = rp.PlanID
					}
				}
			}
			return marketplace.StageResult{
				ExtensionID: plan.ExtensionID, StagedDigest: stagedDigest,
				StagedVersion: stagedVersion, DependencyOrder: orderIDs,
				RolloutPlanID: rolloutPlanID, PreflightOK: true,
			}, nil
		},
		ActivateFn: func(ctx context.Context, plan marketplace.InstallPlan, staged marketplace.StageResult) error {
			if rollout == nil || staged.RolloutPlanID == "" {
				// 无 rollout plan 时仍视为 stage 完成；晋升由运营 Upgrade 驱动。
				return nil
			}
			// migration-once is safe to record here. Canary selection and promote
			// require node-bound health acknowledgements from the runtime nodes.
			if _, err := rollout.MarkMigrationReady(ctx, staged.RolloutPlanID, plan.Actor); err != nil {
				return err
			}
			return fmt.Errorf("marketplace install staged; waiting for external node health acknowledgements before promote")
		},
		RollbackFn: func(ctx context.Context, plan marketplace.InstallPlan, reason string) error {
			if rollout == nil {
				return fmt.Errorf("%w: rollout unavailable", marketplace.ErrInstall)
			}
			if _, err := loadActor(ctx, plan.Actor); err != nil {
				return err
			}
			active, ok, err := rollout.ActivePlan(ctx, plan.ExtensionID)
			if err != nil {
				return err
			}
			if !ok {
				return fmt.Errorf("%w: no active rollout plan for %s", marketplace.ErrInstall, plan.ExtensionID)
			}
			_, err = rollout.Rollback(ctx, active.PlanID, plan.Actor, reason)
			return err
		},
	}
}

// checkPrivacyPermission 使用真实 RBAC：export/erase 需要 user.manage（或 super_admin）。
// 禁止「actor 字符串非空即允许」。
func checkPrivacyPermission(
	ctx context.Context,
	store *identity.PostgresStore,
	actorRef, userID, operation string,
) error {
	_ = userID
	_ = operation
	actorRef = strings.TrimSpace(actorRef)
	if actorRef == "" || store == nil {
		return privacy.ErrPermissionDenied
	}
	// actor 形如 user:123
	var userIDInt int64
	if !strings.HasPrefix(actorRef, "user:") {
		return privacy.ErrPermissionDenied
	}
	if _, err := fmt.Sscanf(actorRef, "user:%d", &userIDInt); err != nil || userIDInt <= 0 {
		return privacy.ErrPermissionDenied
	}
	actor, err := store.LoadActor(ctx, userIDInt)
	if err != nil {
		return privacy.ErrPermissionDenied
	}
	// super_admin 或 user.manage 可执行隐私导出/擦除。
	if actor.IsSuperAdmin() || actor.Can(identity.PermissionUserManage) {
		return nil
	}
	return privacy.ErrPermissionDenied
}

func isProdLike(cfg config.Config) bool {
	return strings.EqualFold(cfg.AppEnv, "production") || strings.EqualFold(cfg.AppEnv, "staging")
}
