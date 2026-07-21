package bootstrap

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"

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
}

// bindProductionP12Ops wires RuntimeRollout / SystemTier / Marketplace / Privacy
// onto PostgreSQL authority so multi-node and CLI recovery share one store.
// Safe Mode does not load system-tier members (checked by callers via LoadOrder).
func bindProductionP12Ops(
	cfg config.Config,
	pool *pgxpool.Pool,
	logger *slog.Logger,
) (*productionP12Ops, error) {
	if pool == nil {
		return nil, fmt.Errorf("bootstrap: P12 ops requires PostgreSQL")
	}
	_ = logger

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

	// Marketplace: production requires Ed25519 verifier when configured;
	// without a public key, only AllowUnsigned (non-production) may load indexes.
	marketPolicy := marketplace.OperatorPolicy{
		AllowedChannels:      []string{marketplace.ChannelStable, marketplace.ChannelBeta},
		DirectUploadFallback: true,
		AllowUnsigned:        !isProdLike(cfg),
	}
	market := marketplace.NewWithOptions(marketplace.Options{Policy: marketPolicy})

	privacyReg := privacy.New()
	// 默认权限：非空 actor；生产可替换为 Host RBAC。
	privacyReg.SetPermissionCheck(func(_ context.Context, actor, userID, operation string) error {
		_ = userID
		_ = operation
		if strings.TrimSpace(actor) == "" {
			return privacy.ErrPermissionDenied
		}
		return nil
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
		Rollout: rollout, SystemTier: tier, Marketplace: market, Privacy: privacyReg,
	}, nil
}

func isProdLike(cfg config.Config) bool {
	return strings.EqualFold(cfg.AppEnv, "production") || strings.EqualFold(cfg.AppEnv, "staging")
}
