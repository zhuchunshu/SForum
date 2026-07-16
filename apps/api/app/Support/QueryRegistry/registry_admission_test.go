package queryregistry

import (
	"context"
	"errors"
	"testing"
)

func TestPluginPlanningAndReleaseRequireExactRuntimeAdmission(t *testing.T) {
	plugin := publication("plugin.admission", false, 'a')
	plugin.Queries = []QueryDeclaration{
		query("plugin.admission.items", "plugin.admission.item", PaginationNone, "public"),
	}
	registry := newPlanningRegistry()
	if _, err := registry.Publish(plugin); err != nil {
		t.Fatal(err)
	}
	request := PlanRequest{QueryID: "plugin.admission.items", Permission: PermissionInput{}}
	if _, err := registry.Plan(context.Background(), request); !errors.Is(err, ErrArtifactUnavailable) {
		t.Fatalf("plugin plan without Host admission = %v", err)
	}

	available := true
	expected := plugin.Artifact
	seen := Artifact{}
	registry.WithPluginAdmission(func(artifact Artifact) bool {
		seen = artifact
		return available && artifact == expected
	})
	plan, err := registry.Plan(context.Background(), request)
	if err != nil {
		t.Fatal(err)
	}
	if seen != expected || plan.Query.Artifact != expected {
		t.Fatalf("admission did not receive exact artifact: seen=%#v plan=%#v", seen, plan.Query.Artifact)
	}

	available = false
	if err := registry.RecheckBeforeRelease(context.Background(), plan, PermissionInput{}); !errors.Is(err, ErrArtifactUnavailable) {
		t.Fatalf("drained runtime released planned results = %v", err)
	}
}

func TestPlanRechecksRuntimeAdmissionAcrossCallbacks(t *testing.T) {
	t.Run("cost", func(t *testing.T) {
		plugin := publication("plugin.cost", false, 'a')
		plugin.Queries = []QueryDeclaration{
			query("plugin.cost.items", "plugin.cost.item", PaginationNone, "public"),
		}
		available := true
		var registry *Registry
		registry = New(WithCostPolicy(CostPolicyFunc(func(QueryCostInput) (QueryCost, error) {
			available = false
			return QueryCost{Units: 1, Maximum: 100}, nil
		}))).WithPluginAdmission(func(artifact Artifact) bool {
			return available && artifact == plugin.Artifact
		})
		if _, err := registry.Publish(plugin); err != nil {
			t.Fatal(err)
		}
		if _, err := registry.Plan(context.Background(), PlanRequest{
			QueryID: plugin.Queries[0].ID, Permission: PermissionInput{},
		}); !errors.Is(err, ErrArtifactUnavailable) {
			t.Fatalf("plan survived runtime drain inside cost callback = %v", err)
		}
	})

	t.Run("permission", func(t *testing.T) {
		plugin := publication("plugin.permission", false, 'b')
		plugin.Queries = []QueryDeclaration{
			query("plugin.permission.items", "plugin.permission.item", PaginationNone, "plugin.permission.read"),
		}
		available := true
		registry := newPlanningRegistry().WithPluginAdmission(func(artifact Artifact) bool {
			return available && artifact == plugin.Artifact
		})
		if _, err := registry.Publish(plugin); err != nil {
			t.Fatal(err)
		}
		if _, err := registry.Plan(context.Background(), PlanRequest{
			QueryID: plugin.Queries[0].ID,
			Permission: PermissionInput{
				Authenticated: true, ActorFingerprint: "actor", PolicyFingerprint: "policy",
				Recheck: PermissionRecheckFunc(func(context.Context, PermissionClaim) error {
					available = false
					return nil
				}),
			},
		}); !errors.Is(err, ErrArtifactUnavailable) {
			t.Fatalf("plan survived runtime drain inside permission callback = %v", err)
		}
	})
}

func TestReleaseRechecksRuntimeAdmissionAcrossCallbacks(t *testing.T) {
	t.Run("cost", func(t *testing.T) {
		plugin := publication("plugin.release.cost", false, 'a')
		plugin.Queries = []QueryDeclaration{
			query("plugin.release.cost.items", "plugin.release.cost.item", PaginationNone, "public"),
		}
		available := true
		drainDuringCost := false
		registry := New(WithCostPolicy(CostPolicyFunc(func(QueryCostInput) (QueryCost, error) {
			if drainDuringCost {
				available = false
			}
			return QueryCost{Units: 1, Maximum: 100}, nil
		}))).WithPluginAdmission(func(artifact Artifact) bool {
			return available && artifact == plugin.Artifact
		})
		if _, err := registry.Publish(plugin); err != nil {
			t.Fatal(err)
		}
		plan, err := registry.Plan(context.Background(), PlanRequest{
			QueryID: plugin.Queries[0].ID, Permission: PermissionInput{},
		})
		if err != nil {
			t.Fatal(err)
		}
		drainDuringCost = true
		if err := registry.RecheckBeforeRelease(context.Background(), plan, PermissionInput{}); !errors.Is(err, ErrArtifactUnavailable) {
			t.Fatalf("release survived runtime drain inside cost callback = %v", err)
		}
	})

	t.Run("permission", func(t *testing.T) {
		plugin := publication("plugin.release.permission", false, 'b')
		plugin.Queries = []QueryDeclaration{
			query("plugin.release.permission.items", "plugin.release.permission.item", PaginationNone, "plugin.release.permission.read"),
		}
		available := true
		registry := newPlanningRegistry().WithPluginAdmission(func(artifact Artifact) bool {
			return available && artifact == plugin.Artifact
		})
		if _, err := registry.Publish(plugin); err != nil {
			t.Fatal(err)
		}
		permission := PermissionInput{
			Authenticated: true, ActorFingerprint: "actor", PolicyFingerprint: "policy", Recheck: allowAll(),
		}
		plan, err := registry.Plan(context.Background(), PlanRequest{
			QueryID: plugin.Queries[0].ID, Permission: permission,
		})
		if err != nil {
			t.Fatal(err)
		}
		permission.Recheck = PermissionRecheckFunc(func(context.Context, PermissionClaim) error {
			available = false
			return nil
		})
		if err := registry.RecheckBeforeRelease(context.Background(), plan, permission); !errors.Is(err, ErrArtifactUnavailable) {
			t.Fatalf("release survived runtime drain inside permission callback = %v", err)
		}
	})
}
