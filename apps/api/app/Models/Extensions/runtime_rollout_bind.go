package extensions

import (
	"context"
	"fmt"
	"strings"

	identity "github.com/zhuchunshu/sforum/apps/api/app/Models/Identity"
	runtimerollout "github.com/zhuchunshu/sforum/apps/api/app/Support/RuntimeRollout"
)

// RuntimeRolloutCoordinator is the minimal RuntimeRollout surface used by upgrades.
type RuntimeRolloutCoordinator interface {
	CreatePlan(ctx context.Context, extensionID, sourceDigest, targetDigest, actor string, canaryPercent, retain int) (runtimerollout.Plan, error)
	MarkMigrationReady(ctx context.Context, planID, actor string) (runtimerollout.Plan, error)
	AckNode(ctx context.Context, planID, nodeID, phase, health string, canary bool) (runtimerollout.Plan, error)
	SelectCanary(ctx context.Context, planID, actor string) (runtimerollout.Plan, error)
	BeginDrain(ctx context.Context, planID, actor string) (runtimerollout.Plan, error)
	PromoteAtomic(ctx context.Context, planID, actor string) (runtimerollout.Plan, error)
	Rollback(ctx context.Context, planID, actor, reason string) (runtimerollout.Plan, error)
	Fail(ctx context.Context, planID, actor, message string) (runtimerollout.Plan, error)
	ActivePlan(ctx context.Context, extensionID string) (runtimerollout.Plan, bool, error)
}

// BindRuntimeRollout late-binds multi-node staged rollout after P12 bootstrap.
func (s *Service) BindRuntimeRollout(coord RuntimeRolloutCoordinator) *Service {
	if s == nil {
		return nil
	}
	s.assetPublicationMu.Lock()
	s.runtimeRollout = coord
	s.assetPublicationMu.Unlock()
	return s
}

// DriveRuntimeRolloutForStagedUpgrade creates/drives a rollout plan for staged upgrade.
// Migration failure marks the plan failed without promoting.
func (s *LifecycleService) DriveRuntimeRolloutForStagedUpgrade(
	ctx context.Context,
	actor identity.Actor,
	source, target Extension,
	migrationOK bool,
	migrationErr error,
) (runtimerollout.Plan, error) {
	if s == nil || s.runtimeRollout == nil {
		return runtimerollout.Plan{}, nil
	}
	sourceDigest := strings.ToLower(strings.TrimSpace(source.PackageDigest))
	targetDigest := strings.ToLower(strings.TrimSpace(target.PackageDigest))
	if sourceDigest == "" || targetDigest == "" || sourceDigest == targetDigest {
		return runtimerollout.Plan{}, nil
	}
	actorID := settingsActorID(actor)
	plan, err := s.runtimeRollout.CreatePlan(ctx, source.ID, sourceDigest, targetDigest, actorID, 10, 3)
	if err != nil {
		if err == runtimerollout.ErrConflict {
			// 复用已有活动 plan。
			existing, ok, getErr := s.runtimeRollout.ActivePlan(ctx, source.ID)
			if getErr != nil {
				return runtimerollout.Plan{}, getErr
			}
			if ok {
				plan = existing
			} else {
				return runtimerollout.Plan{}, err
			}
		} else {
			return runtimerollout.Plan{}, err
		}
	}
	if !migrationOK {
		msg := "migration failed"
		if migrationErr != nil {
			msg = migrationErr.Error()
		}
		return s.runtimeRollout.Fail(ctx, plan.PlanID, actorID, msg)
	}
	if _, err := s.runtimeRollout.MarkMigrationReady(ctx, plan.PlanID, actorID); err != nil {
		return plan, err
	}
	// Node health is an external, node-bound proof. A process-local upgrade
	// helper must never manufacture an AckNode or promote without one.
	return plan, fmt.Errorf("runtime rollout staged; waiting for external node health acknowledgements")
}
