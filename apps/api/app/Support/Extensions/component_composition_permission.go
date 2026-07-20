package extensionsruntime

import (
	"context"
	"errors"
	"fmt"
)

var ErrComponentCompositionPermissionDenied = errors.New("component composition permission denied")

func (e *ComponentCompositionExecutor) authorizeComponentContribution(
	ctx context.Context,
	actor ComponentActorAuthority,
	contribution ComponentContribution,
) (bool, error) {
	if contribution.Permission == "" {
		return true, nil
	}
	if ctx == nil {
		return false, ErrComponentCompositionInvalid
	}
	if !componentActorAllows(actor, contribution.Permission) || e == nil || e.permissions == nil {
		return false, nil
	}
	allowed, err := e.permissions.AuthorizeComponent(ctx, actor.UserID, contribution.Permission)
	if err != nil {
		if ctx.Err() != nil {
			return false, ctx.Err()
		}
		return false, fmt.Errorf("%w: %v", ErrComponentCompositionUnauthorized, err)
	}
	return allowed, nil
}

func (e *ComponentCompositionExecutor) authorizeComponentPlan(
	ctx context.Context,
	actor ComponentActorAuthority,
	plan ComponentResolvePlan,
) (ComponentResolvePlan, error) {
	decisions := make(map[string]bool)
	authorize := func(contribution ComponentContribution) (bool, error) {
		if contribution.Permission == "" {
			return true, nil
		}
		if allowed, found := decisions[contribution.Permission]; found {
			return allowed, nil
		}
		allowed, err := e.authorizeComponentContribution(ctx, actor, contribution)
		if err != nil {
			return false, err
		}
		decisions[contribution.Permission] = allowed
		return allowed, nil
	}

	if plan.Target.Provider != nil {
		allowed, err := authorize(*plan.Target.Provider)
		if err != nil {
			return ComponentResolvePlan{}, err
		}
		if !allowed {
			return ComponentResolvePlan{}, ErrComponentCompositionPermissionDenied
		}
	}

	filtered := make([]ComponentContribution, 0, len(plan.Contributions))
	for _, contribution := range plan.Contributions {
		allowed, err := authorize(contribution)
		if err != nil {
			return ComponentResolvePlan{}, err
		}
		if allowed {
			filtered = append(filtered, contribution)
		}
	}
	plan.Contributions = filtered
	if plan.ReplaceWinner != nil {
		allowed, err := authorize(*plan.ReplaceWinner)
		if err != nil {
			return ComponentResolvePlan{}, err
		}
		if !allowed {
			plan.ReplaceWinner = nil
		}
	}
	return plan, nil
}

func (r *componentCompositionRun) authorizeComponentContribution(
	ctx context.Context,
	contribution ComponentContribution,
) (bool, error) {
	if r == nil || r.executor == nil {
		return false, ErrComponentCompositionInvalid
	}
	return r.executor.authorizeComponentContribution(ctx, r.actor, contribution)
}

func cloneComponentActorAuthority(value ComponentActorAuthority) ComponentActorAuthority {
	if value.Permissions == nil {
		return value
	}
	permissions := make(map[string]bool, len(value.Permissions))
	for permission, allowed := range value.Permissions {
		permissions[permission] = allowed
	}
	value.Permissions = permissions
	return value
}

func componentActorAllows(actor ComponentActorAuthority, permission string) bool {
	return actor.UserID > 0 && (actor.SuperAdmin || actor.Permissions[permission])
}

func (r *componentCompositionRun) validateComponentPermissions(ctx context.Context) error {
	seen := make(map[string]struct{})
	for _, held := range r.admissions {
		permission := held.contribution.Permission
		if permission == "" {
			continue
		}
		if _, duplicate := seen[permission]; duplicate {
			continue
		}
		seen[permission] = struct{}{}
		allowed, err := r.authorizeComponentContribution(ctx, held.contribution)
		if err != nil {
			return err
		}
		if !allowed {
			return ErrComponentCompositionPermissionDenied
		}
	}
	return nil
}
