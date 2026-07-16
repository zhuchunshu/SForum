package queryregistry

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

func TestExecutionResultFiltersShareCumulativeByteBudget(t *testing.T) {
	artifact := publication("plugin.filter-budget", false, 'b').Artifact
	var calls atomic.Int32
	filters := executionBudgetFilters(artifact, 4, ResultFilterFailOpen, func(_ context.Context, request ResultFilterRequest, _ int) (ResultFilterResult, error) {
		calls.Add(1)
		return ResultFilterResult{Rows: request.Rows}, nil
	})
	provider := ExecutableProviderFunc(func(context.Context, ProviderExecutionRequest) (ProviderExecutionResult, error) {
		return ProviderExecutionResult{Rows: []QueryRow{{"id": "1", "title": strings.Repeat("x", 256)}}}, nil
	})
	runtime, _ := executionTestRuntime(t, PaginationNone, PermissionPolicyPublic, provider, filters, func(config *ExecutionConfig) {
		config.MaxResultBytes = 1024
		config.Registry.WithPluginAdmission(func(candidate Artifact) bool { return candidate == artifact })
	})

	_, err := runtime.Execute(t.Context(), PlanRequest{QueryID: "core.execute.items"})
	if !errors.Is(err, ErrResultTooLarge) || calls.Load() < 2 {
		t.Fatalf("cumulative filter bytes: err=%v calls=%d", err, calls.Load())
	}
}

func TestExecutionResultFiltersShareCumulativeNodeBudget(t *testing.T) {
	artifact := publication("plugin.filter-node-budget", false, 'b').Artifact
	var calls atomic.Int32
	filters := executionBudgetFilters(artifact, 46, ResultFilterFailOpen, func(_ context.Context, request ResultFilterRequest, _ int) (ResultFilterResult, error) {
		calls.Add(1)
		request.Rows[0]["title"] = nestedResultPointer("", 25)
		return ResultFilterResult{Rows: request.Rows}, nil
	})
	provider := ExecutableProviderFunc(func(context.Context, ProviderExecutionRequest) (ProviderExecutionResult, error) {
		return ProviderExecutionResult{Rows: []QueryRow{{"id": "", "title": ""}}}, nil
	})
	runtime, _ := executionTestRuntime(t, PaginationNone, PermissionPolicyPublic, provider, filters, func(config *ExecutionConfig) {
		config.MaxResultBytes = 1024
		config.Registry.WithPluginAdmission(func(candidate Artifact) bool { return candidate == artifact })
	})

	_, err := runtime.Execute(t.Context(), PlanRequest{QueryID: "core.execute.items"})
	if !errors.Is(err, ErrResultTooLarge) || !strings.Contains(err.Error(), "node count") || calls.Load() < 2 {
		t.Fatalf("cumulative filter nodes: err=%v calls=%d", err, calls.Load())
	}
}

func TestExecutionResultFiltersShareWholeExecutionDeadline(t *testing.T) {
	artifact := publication("plugin.filter-time-budget", false, 'c').Artifact
	var calls atomic.Int32
	filters := executionBudgetFilters(artifact, 2, ResultFilterFailOpen, func(ctx context.Context, request ResultFilterRequest, _ int) (ResultFilterResult, error) {
		calls.Add(1)
		timer := time.NewTimer(60 * time.Millisecond)
		defer timer.Stop()
		select {
		case <-ctx.Done():
			return ResultFilterResult{}, ctx.Err()
		case <-timer.C:
			return ResultFilterResult{Rows: request.Rows}, nil
		}
	})
	for index := range filters {
		filters[index].Timeout = 500 * time.Millisecond
	}
	provider := ExecutableProviderFunc(func(context.Context, ProviderExecutionRequest) (ProviderExecutionResult, error) {
		return ProviderExecutionResult{Rows: []QueryRow{{"id": "1", "title": "base"}}}, nil
	})
	runtime, _ := executionTestRuntime(t, PaginationNone, PermissionPolicyPublic, provider, filters, func(config *ExecutionConfig) {
		config.Timeout = 100 * time.Millisecond
		config.Registry.WithPluginAdmission(func(candidate Artifact) bool { return candidate == artifact })
	})

	started := time.Now()
	_, err := runtime.Execute(t.Context(), PlanRequest{QueryID: "core.execute.items"})
	duration := time.Since(started)
	if !errors.Is(err, context.DeadlineExceeded) || calls.Load() != 2 || duration >= 180*time.Millisecond {
		t.Fatalf("cumulative filter time: err=%v calls=%d duration=%s", err, calls.Load(), duration)
	}
}

func TestExecutionResultFilterCallbackAlwaysRunsHostFences(t *testing.T) {
	provider := ExecutableProviderFunc(func(context.Context, ProviderExecutionRequest) (ProviderExecutionResult, error) {
		return ProviderExecutionResult{Rows: []QueryRow{{"id": "1", "title": "private"}}}, nil
	})

	t.Run("permission after failed fail-open callback", func(t *testing.T) {
		artifact := publication("plugin.filter-permission-fence", false, 'd').Artifact
		var denied atomic.Bool
		filter := executionBudgetFilters(artifact, 1, ResultFilterFailOpen, func(_ context.Context, _ ResultFilterRequest, _ int) (ResultFilterResult, error) {
			denied.Store(true)
			return ResultFilterResult{}, errors.New("ordinary plugin failure")
		})
		permission := PermissionInput{
			Authenticated: true, ActorFingerprint: "actor-1", PolicyFingerprint: "role:reader",
			Recheck: PermissionRecheckFunc(func(context.Context, PermissionClaim) error {
				if denied.Load() {
					return ErrDenied
				}
				return nil
			}),
		}
		runtime, _ := executionTestRuntime(t, PaginationNone, "core.execute.read", provider, filter, func(config *ExecutionConfig) {
			config.Registry.WithPluginAdmission(func(candidate Artifact) bool { return candidate == artifact })
		})
		if _, err := runtime.Execute(t.Context(), PlanRequest{
			QueryID: "core.execute.items", Permission: permission,
		}); !errors.Is(err, ErrDenied) {
			t.Fatalf("permission fence after failed callback=%v", err)
		}
	})

	t.Run("artifact after failed fail-open callback", func(t *testing.T) {
		artifact := publication("plugin.filter-artifact-fence", false, 'a').Artifact
		var admitted atomic.Bool
		admitted.Store(true)
		filter := executionBudgetFilters(artifact, 1, ResultFilterFailOpen, func(_ context.Context, _ ResultFilterRequest, _ int) (ResultFilterResult, error) {
			admitted.Store(false)
			return ResultFilterResult{}, errors.New("ordinary plugin failure")
		})
		runtime, _ := executionTestRuntime(t, PaginationNone, PermissionPolicyPublic, provider, filter, func(config *ExecutionConfig) {
			config.Registry.WithPluginAdmission(func(candidate Artifact) bool {
				return candidate == artifact && admitted.Load()
			})
		})
		if _, err := runtime.Execute(t.Context(), PlanRequest{QueryID: "core.execute.items"}); !errors.Is(err, ErrDependencyDenied) {
			t.Fatalf("artifact fence after failed callback=%v", err)
		}
	})

	t.Run("plan cost after failed fail-open callback", func(t *testing.T) {
		artifact := publication("plugin.filter-cost-fence", false, 'c').Artifact
		var drifted atomic.Bool
		filter := executionBudgetFilters(artifact, 1, ResultFilterFailOpen, func(_ context.Context, _ ResultFilterRequest, _ int) (ResultFilterResult, error) {
			drifted.Store(true)
			return ResultFilterResult{}, errors.New("ordinary plugin failure")
		})
		runtime, _ := executionTestRuntime(t, PaginationNone, PermissionPolicyPublic, provider, filter, func(config *ExecutionConfig) {
			config.Registry.WithPluginAdmission(func(candidate Artifact) bool { return candidate == artifact })
			config.Registry.costPolicy = CostPolicyFunc(func(QueryCostInput) (QueryCost, error) {
				if drifted.Load() {
					return QueryCost{Units: 2, Maximum: 1}, nil
				}
				return QueryCost{Units: 1, Maximum: 100}, nil
			})
		})
		if _, err := runtime.Execute(t.Context(), PlanRequest{QueryID: "core.execute.items"}); !errors.Is(err, ErrArtifactConflict) {
			t.Fatalf("cost fence after failed callback=%v", err)
		}
	})
}

func executionBudgetFilters(
	artifact Artifact,
	count int,
	failurePolicy string,
	callback func(context.Context, ResultFilterRequest, int) (ResultFilterResult, error),
) []ResultFilterRegistration {
	filters := make([]ResultFilterRegistration, 0, count)
	for index := 0; index < count; index++ {
		filterIndex := index
		id := fmt.Sprintf("%s.filter-%02d", artifact.ExtensionID, index)
		filter := executionTestFilter(artifact, id, count-index, failurePolicy, func(rows []QueryRow) []QueryRow { return rows })
		filter.Filter = ResultFilterFunc(func(ctx context.Context, request ResultFilterRequest) (ResultFilterResult, error) {
			return callback(ctx, request, filterIndex)
		})
		filters = append(filters, filter)
	}
	return filters
}

func nestedResultPointer(value any, layers int) any {
	result := value
	for index := 0; index < layers; index++ {
		current := result
		result = &current
	}
	return result
}
