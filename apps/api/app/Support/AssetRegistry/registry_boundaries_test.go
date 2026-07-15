package assetregistry

import (
	"errors"
	"fmt"
	"strings"
	"testing"
)

func TestRegistryEnforcesPerAssetLimits(t *testing.T) {
	accepted := limitPublication(
		"limit.accepted", 1, maxDependenciesPerAsset, maxScopesPerAsset, maxCSPPerAsset,
	)
	if _, err := normalizePublication(accepted); err != nil {
		t.Fatalf("accepted boundary: %v", err)
	}

	tests := []struct {
		name        string
		publication Publication
	}{
		{"dependencies", limitPublication("limit.dependencies", 1, maxDependenciesPerAsset+1, 0, 0)},
		{"scopes", limitPublication("limit.scopes", 1, 0, maxScopesPerAsset+1, 0)},
		{"csp", limitPublication("limit.csp", 1, 0, 0, maxCSPPerAsset+1)},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if _, err := normalizePublication(test.publication); !errors.Is(err, ErrInvalid) {
				t.Fatalf("expected invalid, got %v", err)
			}
		})
	}
}

func TestRegistryEnforcesPerPublicationAggregateLimits(t *testing.T) {
	accepted := limitPublication(
		"publication.accepted", 64,
		maxDependenciesPerPublication, maxScopesPerPublication, maxCSPPerPublication,
	)
	if _, err := normalizePublication(accepted); err != nil {
		t.Fatalf("accepted aggregate boundary: %v", err)
	}

	tests := []struct {
		name        string
		publication Publication
	}{
		{"assets", limitPublication("publication.assets", maxAssetsPerPublication+1, 0, 0, 0)},
		{"dependencies", limitPublication("publication.dependencies", 65, maxDependenciesPerPublication+1, 0, 0)},
		{"scopes", limitPublication("publication.scopes", 65, 0, maxScopesPerPublication+1, 0)},
		{"csp", limitPublication("publication.csp", 65, 0, 0, maxCSPPerPublication+1)},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if _, err := normalizePublication(test.publication); !errors.Is(err, ErrInvalid) {
				t.Fatalf("expected invalid, got %v", err)
			}
		})
	}
}

func TestRegistryEnforcesGlobalGraphLimits(t *testing.T) {
	owners := make([]Publication, 0, maxRegistryOwners+1)
	for index := 0; index < maxRegistryOwners+1; index++ {
		owners = append(owners, fixturePublication(fmt.Sprintf("global.owner.%03d", index), digestA, nil))
	}
	assertGlobalLimit(t, "owners", owners[:maxRegistryOwners], owners)

	tests := []struct {
		name           string
		kind           string
		maximum        int
		perPublication int
		perAsset       int
	}{
		{"assets", "assets", maxRegistryAssets, maxAssetsPerPublication, 1},
		{"dependencies", "dependencies", maxRegistryDependencies, maxDependenciesPerPublication, maxDependenciesPerAsset},
		{"scopes", "scopes", maxRegistryScopes, maxScopesPerPublication, maxScopesPerAsset},
		{"csp", "csp", maxRegistryCSP, maxCSPPerPublication, maxCSPPerAsset},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			exact := limitPublications("global."+test.name, test.kind, test.maximum, test.perPublication, test.perAsset)
			overflow := limitPublications("global."+test.name, test.kind, test.maximum+1, test.perPublication, test.perAsset)
			assertGlobalLimit(t, test.name, exact, overflow)
		})
	}
}

func TestRegistryBoundsPlanInputs(t *testing.T) {
	registry := New()
	if _, err := registry.Plan(PlanRequest{Handles: make([]string, maxPlanHandles+1)}); !errors.Is(err, ErrInvalid) {
		t.Fatalf("handle bound: %v", err)
	}
	if _, err := registry.Plan(PlanRequest{Scopes: make([]string, maxPlanScopes+1)}); !errors.Is(err, ErrInvalid) {
		t.Fatalf("scope bound: %v", err)
	}
}

func TestRegistryStrictlyValidatesPlanIdentities(t *testing.T) {
	registry := New()
	if _, err := registry.Publish(fixturePublication("plan.assets", digestA, []Declaration{
		{
			Handle: "plan.assets.topic", ContractVersion: "plan.assets.topic@1", Type: "script",
			Path: "topic.mjs", Digest: digestA, Scope: []string{"forum.component.topic"},
		},
		{
			Handle: "plan.assets.global", ContractVersion: "plan.assets.global@1", Type: "style",
			Path: "global.css", Digest: digestB,
		},
	})); err != nil {
		t.Fatal(err)
	}

	invalid := []struct {
		name    string
		request PlanRequest
	}{
		{"empty handle", PlanRequest{Handles: []string{""}}},
		{"trimmed handle", PlanRequest{Handles: []string{" plan.assets.topic "}}},
		{"case-folded handle", PlanRequest{Handles: []string{"PLAN.ASSETS.TOPIC"}}},
		{"malformed handle", PlanRequest{Handles: []string{"../plan.assets.topic"}}},
		{"long handle", PlanRequest{Handles: []string{"plan.assets." + strings.Repeat("a", 122)}}},
		{"duplicate handle", PlanRequest{Handles: []string{"plan.assets.topic", "plan.assets.topic"}}},
		{"empty scope", PlanRequest{Scopes: []string{""}}},
		{"trimmed scope", PlanRequest{Scopes: []string{" forum.component.topic "}}},
		{"case-folded scope", PlanRequest{Scopes: []string{"FORUM.COMPONENT.TOPIC"}}},
		{"malformed scope", PlanRequest{Scopes: []string{"forum/component/topic"}}},
		{"long scope", PlanRequest{Scopes: []string{"forum.component." + strings.Repeat("a", 122)}}},
		{"duplicate scope", PlanRequest{Scopes: []string{"forum.component.topic", "forum.component.topic"}}},
		{"scope bypass", PlanRequest{Handles: []string{"plan.assets.topic"}, Scopes: []string{"forum.component.other"}}},
	}
	for _, test := range invalid {
		t.Run(test.name, func(t *testing.T) {
			if _, err := registry.Plan(test.request); !errors.Is(err, ErrInvalid) {
				t.Fatalf("expected invalid, got %v", err)
			}
		})
	}

	if _, err := registry.Plan(PlanRequest{Handles: []string{"plan.assets.missing"}}); !errors.Is(err, ErrNotFound) {
		t.Fatalf("unknown handle: %v", err)
	}
	plan, err := registry.Plan(PlanRequest{
		Handles: []string{"plan.assets.topic"}, Scopes: []string{"forum.component.topic"},
	})
	if err != nil || len(plan) != 1 || plan[0].Handle != "plan.assets.topic" {
		t.Fatalf("valid scoped handle: plan=%#v err=%v", plan, err)
	}
	plan, err = registry.Plan(PlanRequest{Scopes: []string{"forum.component.unknown"}})
	if err != nil || len(plan) != 0 {
		t.Fatalf("unused valid scope should return an empty plan: plan=%#v err=%v", plan, err)
	}
}

func assertGlobalLimit(t *testing.T, name string, accepted, rejected []Publication) {
	t.Helper()
	if _, err := normalizePublications(accepted); err != nil {
		t.Fatalf("%s accepted boundary: %v", name, err)
	}
	if _, err := normalizePublications(rejected); !errors.Is(err, ErrInvalid) {
		t.Fatalf("%s overflow: %v", name, err)
	}
}

func limitPublications(prefix, kind string, total, perPublication, perAsset int) []Publication {
	result := make([]Publication, 0, total/perPublication+1)
	remaining := total
	for index := 0; remaining > 0; index++ {
		count := min(remaining, perPublication)
		assetCount := count
		dependencies, scopes, csp := 0, 0, 0
		switch kind {
		case "dependencies":
			assetCount = (count + perAsset - 1) / perAsset
			dependencies = count
		case "scopes":
			assetCount = (count + perAsset - 1) / perAsset
			scopes = count
		case "csp":
			assetCount = (count + perAsset - 1) / perAsset
			csp = count
		}
		result = append(result, limitPublication(
			fmt.Sprintf("%s.owner.%03d", prefix, index), assetCount, dependencies, scopes, csp,
		))
		remaining -= count
	}
	return result
}

func limitPublication(extensionID string, assetCount, dependencyCount, scopeCount, cspCount int) Publication {
	assets := make([]Declaration, assetCount)
	dependenciesLeft, scopesLeft, cspLeft := dependencyCount, scopeCount, cspCount
	for assetIndex := range assets {
		handle := fmt.Sprintf("%s.asset.%04d", extensionID, assetIndex)
		assets[assetIndex] = Declaration{
			Handle: handle, ContractVersion: handle + "@1", Type: "script",
			Path: fmt.Sprintf("asset-%04d.mjs", assetIndex), Digest: digestB,
			Dependencies: limitIDs("core.asset.dependency", min(dependenciesLeft, maxDependenciesPerAsset)),
			Scope:        limitIDs("scope.asset", min(scopesLeft, maxScopesPerAsset)),
			CSP:          limitCSP(min(cspLeft, maxCSPPerAsset)),
		}
		dependenciesLeft -= len(assets[assetIndex].Dependencies)
		scopesLeft -= len(assets[assetIndex].Scope)
		cspLeft -= len(assets[assetIndex].CSP)
	}
	// Deliberately retain overflow on the last asset so per-asset limit tests
	// exercise normalization instead of silently truncating their inputs.
	if assetCount > 0 {
		last := &assets[assetCount-1]
		last.Dependencies = append(last.Dependencies, limitIDs("core.asset.overflow", dependenciesLeft)...)
		last.Scope = append(last.Scope, limitIDs("scope.overflow", scopesLeft)...)
		last.CSP = append(last.CSP, limitCSP(cspLeft)...)
	}
	return fixturePublication(extensionID, digestA, assets)
}

func limitIDs(prefix string, count int) []string {
	result := make([]string, count)
	for index := range result {
		result[index] = fmt.Sprintf("%s.%04d", prefix, index)
	}
	return result
}

func limitCSP(count int) []string {
	result := make([]string, count)
	for index := range result {
		result[index] = fmt.Sprintf("connect-src https://source-%04d.invalid", index)
	}
	return result
}
