package extensionsruntime

import (
	"context"
	"errors"
	"testing"
)

type fixedComponentPermissionAuthorizer struct {
	allowed map[string]bool
	err     error
}

func (a fixedComponentPermissionAuthorizer) AuthorizeComponent(
	_ context.Context,
	_ int64,
	permission string,
) (bool, error) {
	if a.err != nil {
		return false, a.err
	}
	return a.allowed[permission], nil
}

func TestAuthorizeComponentPlanFiltersAndDeniesProvider(t *testing.T) {
	t.Parallel()

	executor := &ComponentCompositionExecutor{
		permissions: fixedComponentPermissionAuthorizer{
			allowed: map[string]bool{"plugin.view": true},
		},
	}
	plan := ComponentResolvePlan{
		Revision: 1,
		Target: ComponentTarget{
			ID: "core.panel",
			Provider: &ComponentContribution{
				ID: "plugin.provider", Permission: "plugin.admin",
			},
		},
		Contributions: []ComponentContribution{
			{ID: "plugin.allowed", Permission: "plugin.view"},
			{ID: "plugin.hidden", Permission: "plugin.admin"},
			{ID: "plugin.public"},
		},
	}
	actor := ComponentActorAuthority{
		UserID: 9,
		Permissions: map[string]bool{
			"plugin.view":  true,
			"plugin.admin": true, // credential ceiling; live authorizer still denies admin
		},
	}

	// Provider denied → whole plan denied.
	if _, err := executor.authorizeComponentPlan(t.Context(), actor, plan); !errors.Is(err, ErrComponentCompositionPermissionDenied) {
		t.Fatalf("provider deny: %v", err)
	}

	plan.Target.Provider = nil
	filtered, err := executor.authorizeComponentPlan(t.Context(), actor, plan)
	if err != nil {
		t.Fatal(err)
	}
	if len(filtered.Contributions) != 2 {
		t.Fatalf("contributions=%#v", filtered.Contributions)
	}
	if filtered.Contributions[0].ID != "plugin.allowed" || filtered.Contributions[1].ID != "plugin.public" {
		t.Fatalf("filtered=%#v", filtered.Contributions)
	}
}

func TestComponentActorCeilingDeniesWithoutLiveCall(t *testing.T) {
	t.Parallel()

	var called bool
	executor := &ComponentCompositionExecutor{
		permissions: ComponentPermissionAuthorizerFunc(func(context.Context, int64, string) (bool, error) {
			called = true
			return true, nil
		}),
	}
	allowed, err := executor.authorizeComponentContribution(
		t.Context(),
		ComponentActorAuthority{UserID: 4, Permissions: map[string]bool{}},
		ComponentContribution{ID: "x", Permission: "plugin.view"},
	)
	if err != nil || allowed || called {
		t.Fatalf("allowed=%v err=%v called=%v", allowed, err, called)
	}
}
