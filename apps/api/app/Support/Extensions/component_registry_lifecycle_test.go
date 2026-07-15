package extensionsruntime

import (
	"errors"
	"fmt"
	"strings"
	"sync"
	"testing"

	extensions "github.com/zhuchunshu/sforum/apps/api/app/Models/Extensions"
	extensionmanifest "github.com/zhuchunshu/sforum/apps/api/app/Support/ExtensionManifest"
)

func TestComponentRegistryFreezesSchemasRejectsDriftAndRollsBackAtomically(t *testing.T) {
	id := "component.schema"
	declaration := componentTestContribution(
		id, "replace-home", extensionmanifest.ComponentActionReplace, 10,
		componentTestCoreTarget, componentTestCoreContract,
	)
	original := componentTestExtension(t, id, extensions.TypePlugin, declaration)
	registry := NewComponentRegistry()
	if err := registry.ReplaceRuntime(original, "runtime-original"); err != nil {
		t.Fatal(err)
	}
	runtimePlan, err := registry.resolveRuntimePlan(componentTestCoreTarget, componentTestCoreContract)
	if err != nil || runtimePlan.ReplaceWinner == nil {
		t.Fatalf("runtime plan = %#v, %v", runtimePlan, err)
	}
	frozen := *runtimePlan.ReplaceWinner
	plan, err := registry.ResolvePlan(componentTestCoreTarget, componentTestCoreContract)
	if err != nil || plan.ReplaceWinner == nil || plan.ReplaceWinner.PropsSchemaDigest == "" ||
		plan.ReplaceWinner.ResultSchemaDigest == "" ||
		plan.ReplaceWinner.PropsSchemaDigest == plan.ReplaceWinner.ResultSchemaDigest {
		t.Fatalf("frozen schema plan = %#v, %v", plan, err)
	}
	if err := registry.ValidateProps(*plan.ReplaceWinner, map[string]any{"scope": "home"}); err != nil {
		t.Fatal(err)
	}
	if err := registry.ValidateProps(*plan.ReplaceWinner, map[string]any{"scope": 42}); !errors.Is(err, ErrComponentRegistryInvalid) {
		t.Fatalf("invalid props = %v", err)
	}
	if err := registry.ValidateResult(*plan.ReplaceWinner, map[string]any{"html": "<p>ready</p>"}); err != nil {
		t.Fatal(err)
	}
	if err := registry.ValidateResult(*plan.ReplaceWinner, map[string]any{"scope": "wrong"}); !errors.Is(err, ErrComponentRegistryInvalid) {
		t.Fatalf("invalid result = %v", err)
	}

	drift := componentTestExtensionWithSchemas(
		t, id, extensions.TypePlugin,
		`{"type":"object","required":["scope","page"],"properties":{"scope":{"type":"string"},"page":{"type":"integer"}},"additionalProperties":false}`,
		componentTestResultSchema,
		declaration,
	)
	drift.Version, drift.Manifest.Version = "1.1.0", "1.1.0"
	drift.PackageDigest = strings.Repeat("b", 64)
	if err := registry.ReplaceRuntime(drift, "runtime-drift"); !errors.Is(err, ErrComponentRegistryConflict) {
		t.Fatalf("same-contract schema drift = %v", err)
	}
	templateDrift := componentTestExtension(t, id, extensions.TypePlugin, declaration)
	templateDrift.Version, templateDrift.Manifest.Version = "1.1.0", "1.1.0"
	templateDrift.PackageDigest = strings.Repeat("c", 64)
	templateDrift.Manifest.Templates[0].Digest = strings.Repeat("2", 64)
	if err := registry.ReplaceRuntime(templateDrift, "runtime-template-drift"); !errors.Is(err, ErrComponentRegistryConflict) {
		t.Fatalf("same-contract template drift = %v", err)
	}
	declarationDrift := componentTestExtension(t, id, extensions.TypePlugin, declaration)
	declarationDrift.Version, declarationDrift.Manifest.Version = "1.1.0", "1.1.0"
	declarationDrift.PackageDigest = strings.Repeat("d", 64)
	declarationDrift.Manifest.Components[0].Priority++
	if err := registry.ReplaceRuntime(declarationDrift, "runtime-declaration-drift"); !errors.Is(err, ErrComponentRegistryConflict) {
		t.Fatalf("same-contract declaration drift = %v", err)
	}
	unchanged, err := registry.ResolvePlan(componentTestCoreTarget, componentTestCoreContract)
	if err != nil || unchanged.Revision != 1 || unchanged.ReplaceWinner == nil ||
		unchanged.ReplaceWinner.Artifact.RuntimeInstanceID != "runtime-original" {
		t.Fatalf("failed replacement changed snapshot = %#v, %v", unchanged, err)
	}

	bumpedDeclaration := declaration
	bumpedDeclaration.ContractVersion = id + ".component.replace-home@2"
	bumped := componentTestExtensionWithSchemas(
		t, id, extensions.TypePlugin,
		`{"type":"object","required":["scope","page"],"properties":{"scope":{"type":"string"},"page":{"type":"integer"}},"additionalProperties":false}`,
		componentTestResultSchema,
		bumpedDeclaration,
	)
	bumped.Version, bumped.Manifest.Version = "1.1.0", "1.1.0"
	bumped.PackageDigest = strings.Repeat("e", 64)
	if err := registry.ReplaceRuntime(bumped, "runtime-bumped"); err != nil {
		t.Fatalf("versioned schema change = %v", err)
	}
	if err := registry.ValidateProps(*plan.ReplaceWinner, map[string]any{"scope": "home"}); !errors.Is(err, ErrComponentRegistryTargetNotFound) {
		t.Fatalf("detached stale plan = %v", err)
	}
	if err := registry.ValidateProps(frozen, map[string]any{"scope": "home"}); err != nil {
		t.Fatalf("admitted frozen validator = %v", err)
	}
	if err := registry.ValidateProps(frozen, map[string]any{"scope": "home", "page": 1}); !errors.Is(err, ErrComponentRegistryInvalid) {
		t.Fatalf("old validator silently adopted new schema = %v", err)
	}
	if err := registry.ReplaceRuntime(original, "runtime-rollback"); err != nil {
		t.Fatalf("exact rollback = %v", err)
	}
	rolledBack, err := registry.ResolvePlan(componentTestCoreTarget, componentTestCoreContract)
	if err != nil || rolledBack.ReplaceWinner == nil ||
		rolledBack.ReplaceWinner.ContractVersion != declaration.ContractVersion ||
		rolledBack.ReplaceWinner.Artifact.RuntimeInstanceID != "runtime-rollback" {
		t.Fatalf("rollback plan = %#v, %v", rolledBack, err)
	}
	if removed, err := registry.RemoveRuntime(id, "runtime-original"); removed || !errors.Is(err, ErrComponentRegistryConflict) {
		t.Fatalf("stale runtime removal = %t, %v", removed, err)
	}
	if removed, err := registry.RemoveRuntime(id, "runtime-rollback"); err != nil || !removed {
		t.Fatalf("exact runtime removal = %t, %v", removed, err)
	}
}

func TestComponentRegistryDropsExactSelectionOnProviderUpgrade(t *testing.T) {
	id := "component.selection"
	declaration := componentTestContribution(
		id, "replace-home", extensionmanifest.ComponentActionReplace, 10,
		componentTestCoreTarget, componentTestCoreContract,
	)
	extension := componentTestExtension(t, id, extensions.TypePlugin, declaration)
	backupID := "component.selection-backup"
	backup := componentTestExtension(t, backupID, extensions.TypePlugin,
		componentTestContribution(
			backupID, "replace-home", extensionmanifest.ComponentActionReplace, 20,
			componentTestCoreTarget, componentTestCoreContract,
		),
	)
	registry := NewComponentRegistry()
	if err := registry.ReplaceRuntime(extension, "runtime-selected"); err != nil {
		t.Fatal(err)
	}
	if err := registry.ReplaceRuntime(backup, "runtime-backup"); err != nil {
		t.Fatal(err)
	}
	plan, _ := registry.ResolvePlan(componentTestCoreTarget, componentTestCoreContract)
	if _, err := registry.SelectReplaceProvider(SelectComponentProviderRequest{
		TargetID: componentTestCoreTarget, TargetContractVersion: componentTestCoreContract,
		ContributionID: declaration.ID, ExpectedRevision: plan.Revision,
	}); err != nil {
		t.Fatal(err)
	}
	upgrade := extension
	upgrade.Version, upgrade.Manifest.Version = "1.1.0", "1.1.0"
	upgrade.PackageDigest = strings.Repeat("f", 64)
	if err := registry.ReplaceRuntime(upgrade, "runtime-upgraded"); err != nil {
		t.Fatal(err)
	}
	upgraded, err := registry.ResolvePlan(componentTestCoreTarget, componentTestCoreContract)
	if err != nil || upgraded.Selection != nil || upgraded.ReplaceWinner == nil ||
		upgraded.ReplaceWinner.Artifact.ExtensionID != backupID ||
		upgraded.Conflict == nil || upgraded.Conflict.ExplicitSelection {
		t.Fatalf("upgraded exact selection = %#v, %v", upgraded, err)
	}
}

func TestComponentRegistryConcurrentReadersObserveWholeSnapshots(t *testing.T) {
	id := "component.concurrent"
	declaration := componentTestContribution(
		id, "replace-home", extensionmanifest.ComponentActionReplace, 10,
		componentTestCoreTarget, componentTestCoreContract,
	)
	extension := componentTestExtension(t, id, extensions.TypePlugin, declaration)
	registry := NewComponentRegistry()
	if err := registry.ReplaceRuntime(extension, "runtime-0"); err != nil {
		t.Fatal(err)
	}

	const readers = 8
	const readsPerReader = 200
	errorsCh := make(chan error, readers+1)
	var wait sync.WaitGroup
	for reader := 0; reader < readers; reader++ {
		wait.Add(1)
		go func() {
			defer wait.Done()
			var previous uint64
			for range readsPerReader {
				snapshot := registry.Snapshot()
				if snapshot.Revision < previous {
					errorsCh <- fmt.Errorf("revision regressed from %d to %d", previous, snapshot.Revision)
					return
				}
				previous = snapshot.Revision
				plan, err := registry.ResolvePlan(componentTestCoreTarget, componentTestCoreContract)
				if err != nil || len(plan.ReplaceCandidates) != 1 || plan.ReplaceWinner == nil ||
					plan.ReplaceWinner.ID != declaration.ID || plan.ReplaceWinner.Artifact.RuntimeInstanceID == "" {
					errorsCh <- fmt.Errorf("partial resolve plan: %#v, %v", plan, err)
					return
				}
				err = registry.ValidateProps(*plan.ReplaceWinner, map[string]any{"scope": "home"})
				if err != nil && !errors.Is(err, ErrComponentRegistryTargetNotFound) {
					errorsCh <- fmt.Errorf("validate concurrent plan: %w", err)
					return
				}
			}
		}()
	}
	wait.Add(1)
	go func() {
		defer wait.Done()
		for version := 1; version <= 50; version++ {
			next := extension
			next.Version = fmt.Sprintf("1.0.%d", version)
			next.Manifest.Version = next.Version
			next.PackageDigest = fmt.Sprintf("%064x", version+1)
			if err := registry.ReplaceRuntime(next, fmt.Sprintf("runtime-%d", version)); err != nil {
				errorsCh <- fmt.Errorf("replace runtime %d: %w", version, err)
				return
			}
		}
	}()
	wait.Wait()
	close(errorsCh)
	for err := range errorsCh {
		t.Fatal(err)
	}
	final, err := registry.ResolvePlan(componentTestCoreTarget, componentTestCoreContract)
	if err != nil || final.Revision != 51 || final.ReplaceWinner == nil ||
		final.ReplaceWinner.Artifact.RuntimeInstanceID != "runtime-50" {
		t.Fatalf("final concurrent plan = %#v, %v", final, err)
	}
}
