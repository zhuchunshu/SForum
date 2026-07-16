package mediaregistry

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"strings"
	"testing"
)

func TestRegistryPublishesDeterministicImmutableGraphAndConflicts(t *testing.T) {
	core, plugin := corePublicationForTest(), pluginPublicationForTest()
	first := New()
	if _, err := first.Publish(core); err != nil {
		t.Fatal(err)
	}
	if _, err := first.Publish(plugin); err != nil {
		t.Fatal(err)
	}
	second := New()
	if _, err := second.ReplaceAll([]Publication{plugin, core}, false); err != nil {
		t.Fatal(err)
	}
	left, right := first.Snapshot(), second.Snapshot()
	if left.Digest != right.Digest || !reflect.DeepEqual(left.Publications, right.Publications) {
		t.Fatalf("publication order changed graph: %s != %s", left.Digest, right.Digest)
	}
	var policyConflict ProviderConflict
	for _, conflict := range left.Conflicts {
		if conflict.Family == ConflictMIMEPolicy && conflict.Key == "general" {
			policyConflict = conflict
		}
	}
	if policyConflict.Winner.ContributionID != "demo.media.policy" || len(policyConflict.Candidates) != 2 {
		t.Fatalf("unexpected deterministic conflict: %#v", policyConflict)
	}

	left.Publications[0].Policies[0].AllowedMIMEs[0] = "text/html"
	left.Policies[0].MIMEAliases = append(left.Policies[0].MIMEAliases, MIMEAlias{Declared: "x/y", Detected: "z/y"})
	left.VariantBindings[0].Processor.ContributionID = "mutated"
	left.Conflicts[0].Candidates[0].ContributionID = "mutated"
	fresh := first.Snapshot()
	if fresh.Publications[0].Policies[0].AllowedMIMEs[0] == "text/html" || len(fresh.Policies[0].MIMEAliases) != 0 ||
		fresh.VariantBindings[0].Processor.ContributionID == "mutated" || fresh.Conflicts[0].Candidates[0].ContributionID == "mutated" {
		t.Fatal("snapshot mutation escaped deep copy")
	}

	mutated := plugin
	mutated.Policies[0].AllowedMIMEs = []string{"image/jpeg"}
	if _, err := first.Publish(mutated); !errors.Is(err, ErrArtifactConflict) {
		t.Fatalf("exact artifact replay mutation: %v", err)
	}
}

func TestProcessorAndVariantConflictsAreDeterministicAndSelectable(t *testing.T) {
	publications := []Publication{corePublicationForTest(), pluginPublicationForTest(), competingPublicationForTest()}
	first := New()
	if _, err := first.ReplaceAll(publications, false); err != nil {
		t.Fatal(err)
	}
	second := New()
	if _, err := second.ReplaceAll([]Publication{publications[2], publications[0], publications[1]}, false); err != nil {
		t.Fatal(err)
	}
	if first.Snapshot().Digest != second.Snapshot().Digest {
		t.Fatal("input order changed conflict digest")
	}

	snapshot := first.Snapshot()
	conflicts := map[string]ProviderConflict{}
	for _, conflict := range snapshot.Conflicts {
		conflicts[conflict.Family+":"+conflict.Key] = conflict
	}
	cdnKey := ConflictProcessor + ":cdn/general/primary.cdn"
	variantKey := ConflictVariant + ":general/thumbnail"
	if conflicts[cdnKey].Winner.ContributionID != "alternate.media.cdn" || conflicts[variantKey].Winner.ContributionID != "alternate.media.thumbnail" {
		t.Fatalf("unexpected winners: %#v", snapshot.Conflicts)
	}
	plan, err := first.Plan(t.Context(), uploadRequestForTest(), allowAll())
	if err != nil {
		t.Fatal(err)
	}
	transform, _ := stepByStage(plan, StageTransform)
	if transform.Processor.ID != "alternate.media.transform" || len(transform.Variants) != 1 || transform.Variants[0].OutputMIME != "image/avif" {
		t.Fatalf("unexpected transform plan: %#v", transform)
	}

	var demoVariant ProviderRef
	for _, candidate := range conflicts[variantKey].Candidates {
		if candidate.ContributionID == "demo.media.thumbnail" {
			demoVariant = candidate
		}
	}
	if _, err := first.SelectProvider(first.Revision(), ProviderSelection{Family: ConflictVariant, Key: "general/thumbnail", Provider: demoVariant}); err != nil {
		t.Fatal(err)
	}
	plan, err = first.Plan(t.Context(), uploadRequestForTest(), allowAll())
	if err != nil {
		t.Fatal(err)
	}
	transform, _ = stepByStage(plan, StageTransform)
	if transform.Processor.ID != "demo.media.transform" || transform.Variants[0].OutputMIME != "image/webp" {
		t.Fatalf("variant selection did not bind transform: %#v", transform)
	}
	for index := 1; index < len(plan.Steps); index++ {
		if stageRank(plan.Steps[index-1].Processor.Stage) > stageRank(plan.Steps[index].Processor.Stage) {
			t.Fatalf("steps are not ordered: %#v", plan.Steps)
		}
	}
}

func TestSerializedCoreSelectionAndPrivatePlanSummary(t *testing.T) {
	registry := registryWithMediaForTest()
	var conflict ProviderConflict
	for _, value := range registry.Snapshot().Conflicts {
		if value.Family == ConflictMIMEPolicy {
			conflict = value
		}
	}
	var selection ProviderSelection
	for _, candidate := range conflict.Candidates {
		if candidate.Artifact.Core {
			selection = ProviderSelection{Family: ConflictMIMEPolicy, Key: conflict.Key, Provider: candidate}
		}
	}
	encoded, err := json.Marshal(selection)
	if err != nil {
		t.Fatal(err)
	}
	var decoded ProviderSelection
	if err := json.Unmarshal(encoded, &decoded); err != nil {
		t.Fatal(err)
	}
	if decoded.Provider.Artifact.coreSeal != ([32]byte{}) {
		t.Fatal("serialized selection unexpectedly retained core seal")
	}
	if _, err := registry.SelectProvider(registry.Revision(), decoded); err != nil {
		t.Fatalf("serialized exact selection: %v", err)
	}

	plan, err := registry.Plan(t.Context(), uploadRequestForTest(), allowAll())
	if err != nil {
		t.Fatal(err)
	}
	if !plan.Policy.Artifact.Core {
		t.Fatal("core policy was not selected")
	}
	step, _ := stepByStage(plan, StageTransform)
	receipts := newTestReceiptAuthority()
	operation := operationForStepForTest(t, receipts, plan, step.ID, 1)
	if _, err := json.Marshal(plan); !errors.Is(err, ErrPrivatePlan) {
		t.Fatalf("private plan serialization: %v", err)
	}
	if _, err := json.Marshal(operation); !errors.Is(err, ErrPrivatePlan) {
		t.Fatalf("private operation serialization: %v", err)
	}
	// Even reflection through a local alias cannot recover the two raw fields
	// most likely to appear in accidental queue/log serialization.
	type planAlias Plan
	aliased, err := json.Marshal(planAlias(plan))
	if err != nil {
		t.Fatal(err)
	}
	for _, secret := range []string{plan.Actor.ID, plan.Actor.PermissionFingerprint, plan.Source.Filename} {
		if strings.Contains(string(aliased), secret) {
			t.Fatalf("aliased plan leaked %q: %s", secret, aliased)
		}
	}
	summary, err := SummarizePlan(plan)
	if err != nil {
		t.Fatal(err)
	}
	encoded, err = json.Marshal(summary)
	if err != nil {
		t.Fatal(err)
	}
	for _, secret := range []string{plan.Actor.ID, plan.Actor.PermissionFingerprint, plan.Source.ID, plan.Source.Digest, plan.Source.Filename} {
		if strings.Contains(string(encoded), secret) || strings.Contains(fmt.Sprintf("%#v", plan), secret) || strings.Contains(fmt.Sprintf("%#v", operation), secret) {
			t.Fatalf("plan projection leaked %q: json=%s plan=%#v operation=%#v", secret, encoded, plan, operation)
		}
	}
	admission := newTestAdmission(step.Processor.Artifact)
	invoker := invokerFunc(func(context.Context, Invocation) (ProviderOutput, error) {
		return ProviderOutput{Variants: []VariantOutput{{Name: "thumbnail", Handle: "variant/serialized", Digest: strings.Repeat("b", 64), SourceDigest: plan.Source.Digest, MIME: "image/webp", SizeBytes: 100}}}, nil
	})
	if _, err := NewExecutor(registry, admission, invoker, receipts, nil).ExecuteOperation(t.Context(), operation, allowAll()); err != nil {
		t.Fatalf("private in-memory operation: %v", err)
	}
}

func TestRegistryExactSelectionCASUpgradeAndSafeMode(t *testing.T) {
	core, plugin := corePublicationForTest(), pluginPublicationForTest()
	registry := New()
	revision, err := registry.ReplaceAll([]Publication{plugin, core}, false)
	if err != nil {
		t.Fatal(err)
	}
	conflict := registry.Snapshot().Conflicts[0]
	var coreRef ProviderRef
	for _, candidate := range conflict.Candidates {
		if candidate.Artifact.Core {
			coreRef = candidate
		}
	}
	revision, err = registry.SelectProvider(revision, ProviderSelection{Family: ConflictMIMEPolicy, Key: "general", Provider: coreRef})
	if err != nil {
		t.Fatal(err)
	}
	plan, err := registry.Plan(t.Context(), uploadRequestForTest(), allowAll())
	if err != nil {
		t.Fatal(err)
	}
	if plan.Policy.Artifact != core.Artifact {
		t.Fatalf("selected policy ignored: %#v", plan.Policy.Artifact)
	}
	if _, err := registry.ResetProvider(revision-1, ConflictMIMEPolicy, "general"); !errors.Is(err, ErrRevisionConflict) {
		t.Fatalf("stale reset: %v", err)
	}

	upgrade := plugin
	upgrade.Artifact.ExtensionVersion = "1.1.0"
	upgrade.Artifact.PackageDigest = strings.Repeat("7", 64)
	upgrade.Artifact.ImpactDigest = strings.Repeat("8", 64)
	upgrade.Artifact.VersionID = 2
	upgrade.Artifact.RuntimeInstanceID = "demo.media-runtime-v2"
	if _, err := registry.PublishIfArtifact(plugin.Artifact, upgrade); err != nil {
		t.Fatal(err)
	}
	if _, _, err := registry.Remove(plugin.Artifact); !errors.Is(err, ErrArtifactConflict) {
		t.Fatalf("stale artifact removed upgrade: %v", err)
	}

	broken := plugin
	broken.Artifact = Artifact{ExtensionID: "broken"}
	if _, err := registry.ReplaceAll([]Publication{broken, core}, true); err != nil {
		t.Fatalf("safe mode parsed broken plugin: %v", err)
	}
	snapshot := registry.Snapshot()
	if !snapshot.SafeMode || len(snapshot.Publications) != 1 || snapshot.Publications[0].Artifact != core.Artifact {
		t.Fatalf("safe mode snapshot: %#v", snapshot)
	}
	if _, err := registry.Publish(plugin); !errors.Is(err, ErrSafeMode) {
		t.Fatalf("safe mode publication: %v", err)
	}
}

func TestRegistryImmutablePackageHistorySurvivesRemoveAndSafeMode(t *testing.T) {
	plugin := pluginPublicationForTest()
	registry := New()
	if _, err := registry.Publish(plugin); err != nil {
		t.Fatal(err)
	}
	if _, removed, err := registry.Remove(plugin.Artifact); err != nil || !removed {
		t.Fatalf("remove exact publication: removed=%t err=%v", removed, err)
	}

	drift := plugin
	drift.Artifact.RuntimeInstanceID = "demo.media-runtime-restarted"
	drift.Processors = append([]ProcessorDeclaration(nil), plugin.Processors...)
	drift.Processors[0].Handler = "media.processor.reinterpreted"
	if _, err := registry.Publish(drift); !errors.Is(err, ErrArtifactConflict) {
		t.Fatalf("removed immutable package changed declarations: %v", err)
	}
	restarted := plugin
	restarted.Artifact.RuntimeInstanceID = "demo.media-runtime-restarted"
	if _, err := registry.Publish(restarted); err != nil {
		t.Fatalf("same immutable declarations rejected after runtime rotation: %v", err)
	}

	snapshot := registry.Snapshot()
	if _, err := registry.ReplaceAllIfRevision(snapshot.Revision, nil, true); err != nil {
		t.Fatal(err)
	}
	snapshot = registry.Snapshot()
	if _, err := registry.ReplaceAllIfRevision(snapshot.Revision, []Publication{drift}, false); !errors.Is(err, ErrArtifactConflict) {
		t.Fatalf("Safe Mode erased immutable declaration history: %v", err)
	}

	upgrade := drift
	upgrade.Artifact.ExtensionVersion = "1.1.0"
	upgrade.Artifact.PackageDigest = strings.Repeat("7", 64)
	upgrade.Artifact.ImpactDigest = strings.Repeat("8", 64)
	upgrade.Artifact.VersionID = 2
	if _, err := registry.ReplaceAllIfRevision(snapshot.Revision, []Publication{upgrade}, false); err != nil {
		t.Fatalf("new immutable version could not declare a new contract: %v", err)
	}
}

func TestRemovePreservesOriginalFallbackAndStalesOldPlan(t *testing.T) {
	registry := registryWithMediaForTest()
	request := uploadRequestForTest()
	before, err := registry.Plan(t.Context(), request, allowAll())
	if err != nil {
		t.Fatal(err)
	}
	if _, found := stepByStage(before, StageTransform); !found {
		t.Fatal("plugin transform missing")
	}
	plugin := pluginPublicationForTest()
	if _, removed, err := registry.Remove(plugin.Artifact); err != nil || !removed {
		t.Fatalf("remove: removed=%t err=%v", removed, err)
	}
	if err := registry.ValidatePlan(t.Context(), before, allowAll()); !errors.Is(err, ErrPlanStale) {
		t.Fatalf("old plan survived disable: %v", err)
	}
	after, err := registry.Plan(t.Context(), request, allowAll())
	if err != nil {
		t.Fatal(err)
	}
	if after.Source != before.Source || after.OriginalFallback != before.Source || !after.OriginalFallback.Immutable {
		t.Fatalf("original fallback changed: %#v", after.OriginalFallback)
	}
	if len(after.Steps) != 0 || after.Policy.Artifact.ExtensionID != "core.media" {
		t.Fatalf("plugin declarations survived remove: %#v", after.Steps)
	}
}

func TestRemoveIgnoresCrossPublicationProcessorDependencies(t *testing.T) {
	provider := Publication{
		Artifact: pluginArtifactForTest("provider.media", '3'),
		Processors: []ProcessorDeclaration{{
			ID: "provider.media.transform", ContractVersion: "provider.media.transform@1",
			Stage: StageTransform, Purpose: "general", MIMEs: []string{"image/*"}, Handler: "transform",
			Mode: ProcessorCompose, Execution: ExecutionSync, FailureMode: FailureFallbackOriginal,
			RequiredPermission: "attachment.upload",
		}},
	}
	dependent := Publication{
		Artifact: pluginArtifactForTest("dependent.media", '5'),
		Variants: []VariantDeclaration{{
			ID: "dependent.media.thumbnail", ContractVersion: "dependent.media.thumbnail@1",
			Purpose: "general", Name: "thumbnail", ProcessorID: "provider.media.transform",
			ProcessorContractVersion: "provider.media.transform@1", ProcessorOwnerExtensionID: "provider.media",
			ProcessorPackageDigest: provider.Artifact.PackageDigest, OutputMIME: "image/webp",
		}},
	}
	registry := New()
	if _, err := registry.ReplaceAll([]Publication{corePublicationForTest(), provider, dependent}, false); err != nil {
		t.Fatal(err)
	}
	active := registry.Snapshot()
	activeDigest := active.Digest
	if len(active.VariantBindings) != 1 || active.VariantBindings[0].Status != VariantBindingActive ||
		active.VariantBindings[0].Processor == nil || active.VariantBindings[0].Processor.Artifact != provider.Artifact {
		t.Fatalf("variant did not bind exact processor artifact: %#v", active.VariantBindings)
	}
	if _, removed, err := registry.Remove(provider.Artifact); err != nil || !removed {
		t.Fatalf("cross-publication dependency blocked remove: removed=%t err=%v", removed, err)
	}
	snapshot := registry.Snapshot()
	if snapshot.Digest == activeDigest {
		t.Fatal("processor removal did not change graph digest")
	}
	for _, variant := range snapshot.Variants {
		if variant.ID == "dependent.media.thumbnail" || variant.ProcessorID == "provider.media.transform" {
			t.Fatalf("orphan variant remained active: %#v", variant)
		}
	}
	if len(snapshot.VariantBindings) != 1 || snapshot.VariantBindings[0].Status != VariantBindingPending ||
		snapshot.VariantBindings[0].Reason != VariantPendingProcessorMissing || snapshot.VariantBindings[0].Processor != nil {
		t.Fatalf("orphan variant status is not inspectable: %#v", snapshot.VariantBindings)
	}
	foundDependent := false
	for _, publication := range snapshot.Publications {
		if publication.Artifact.ExtensionID == "dependent.media" {
			foundDependent = true
			if len(publication.Variants) != 1 || publication.Variants[0] != dependent.Variants[0] {
				t.Fatalf("inactive orphan declaration disappeared: %#v", publication.Variants)
			}
		}
		if publication.Artifact.ExtensionID == "provider.media" {
			t.Fatal("provider publication survived exact remove")
		}
	}
	if !foundDependent {
		t.Fatal("dependent publication was removed with its processor dependency")
	}
	request := uploadRequestForTest()
	plan, err := registry.Plan(t.Context(), request, allowAll())
	if err != nil {
		t.Fatalf("source-of-truth plan after remove: %v", err)
	}
	if plan.Source != request.Source || plan.OriginalFallback != request.Source || !plan.OriginalFallback.Immutable {
		t.Fatalf("original fallback lost after cross-publication remove: %#v", plan)
	}
	if _, found := stepByStage(plan, StageTransform); found {
		t.Fatalf("removed transform still planned: %#v", plan.Steps)
	}
	if publication, found := registry.SnapshotPublication("dependent.media"); !found || len(publication.Variants) != 1 {
		t.Fatalf("single-publication inspector hid orphan declaration: found=%t publication=%#v", found, publication)
	}

	// 相同 ID、stage 和 purpose 不足以恢复依赖：owner 不匹配时必须保持 pending。
	impostor := provider
	impostor.Artifact = pluginArtifactForTest("impostor.media", '6')
	if _, err := registry.Publish(impostor); err != nil {
		t.Fatalf("publish same-ID impostor: %v", err)
	}
	impostorSnapshot := registry.Snapshot()
	if len(impostorSnapshot.VariantBindings) != 1 ||
		impostorSnapshot.VariantBindings[0].Reason != VariantPendingProcessorIdentityMismatch ||
		len(impostorSnapshot.Variants) != 0 {
		t.Fatalf("same-ID impostor activated variant: %#v", impostorSnapshot.VariantBindings)
	}
	if _, removed, err := registry.Remove(impostor.Artifact); err != nil || !removed {
		t.Fatalf("remove impostor: removed=%t err=%v", removed, err)
	}

	// 相同 owner/processor/contract 但 package digest 不同仍不能自动重连。
	upgraded := provider
	upgraded.Artifact = pluginArtifactForTest("provider.media", '7')
	if _, err := registry.Publish(upgraded); err != nil {
		t.Fatalf("publish mismatched exact package: %v", err)
	}
	upgradedSnapshot := registry.Snapshot()
	if len(upgradedSnapshot.VariantBindings) != 1 ||
		upgradedSnapshot.VariantBindings[0].Reason != VariantPendingProcessorIdentityMismatch ||
		len(upgradedSnapshot.Variants) != 0 {
		t.Fatalf("mismatched exact package activated variant: %#v", upgradedSnapshot.VariantBindings)
	}
	if _, removed, err := registry.Remove(upgraded.Artifact); err != nil || !removed {
		t.Fatalf("remove upgraded provider: removed=%t err=%v", removed, err)
	}

	// 正确 package/owner 但错误 contract 同样不能激活。
	mismatched := provider
	// 使用新的 immutable version row 隔离 contract 字段；保留 variant 所
	// 绑定的 package digest，避免把本断言退化成上一段 package mismatch。
	mismatched.Artifact.ExtensionVersion = "1.1.0"
	mismatched.Artifact.VersionID = 2
	mismatched.Artifact.RuntimeInstanceID = "provider.media-runtime-contract-v2"
	mismatched.Processors = append([]ProcessorDeclaration(nil), provider.Processors...)
	mismatched.Processors[0].ContractVersion = "provider.media.transform@2"
	if _, err := registry.Publish(mismatched); err != nil {
		t.Fatalf("publish mismatched contract: %v", err)
	}
	mismatchedSnapshot := registry.Snapshot()
	if len(mismatchedSnapshot.VariantBindings) != 1 ||
		mismatchedSnapshot.VariantBindings[0].Reason != VariantPendingProcessorIdentityMismatch ||
		len(mismatchedSnapshot.Variants) != 0 {
		t.Fatalf("mismatched contract activated variant: %#v", mismatchedSnapshot.VariantBindings)
	}
	if _, removed, err := registry.Remove(mismatched.Artifact); err != nil || !removed {
		t.Fatalf("remove mismatched provider: removed=%t err=%v", removed, err)
	}
	if _, err := registry.Publish(provider); err != nil {
		t.Fatalf("restore provider: %v", err)
	}
	restored := registry.Snapshot()
	foundVariant := false
	for _, variant := range restored.Variants {
		foundVariant = foundVariant || variant.ID == "dependent.media.thumbnail"
	}
	if !foundVariant {
		t.Fatal("retained declaration did not reactivate when its exact processor returned")
	}
	if len(restored.VariantBindings) != 1 || restored.VariantBindings[0].Status != VariantBindingActive ||
		restored.VariantBindings[0].Processor == nil || restored.VariantBindings[0].Processor.Artifact != provider.Artifact {
		t.Fatalf("restored variant did not bind exact artifact: %#v", restored.VariantBindings)
	}
}

func TestPendingVariantDependencyEntersSnapshotAndDigest(t *testing.T) {
	publication := Publication{
		Artifact: pluginArtifactForTest("dependent.media", '5'),
		Variants: []VariantDeclaration{{
			ID: "dependent.media.thumbnail", ContractVersion: "dependent.media.thumbnail@1",
			Purpose: "general", Name: "thumbnail", ProcessorID: "provider.media.transform",
			ProcessorContractVersion: "provider.media.transform@1", ProcessorOwnerExtensionID: "provider.media",
			ProcessorPackageDigest: strings.Repeat("3", 64), OutputMIME: "image/webp",
		}},
	}
	first := New()
	if _, err := first.ReplaceAll([]Publication{corePublicationForTest(), publication}, false); err != nil {
		t.Fatal(err)
	}
	firstSnapshot := first.Snapshot()
	if len(firstSnapshot.VariantBindings) != 1 || firstSnapshot.VariantBindings[0].Status != VariantBindingPending ||
		firstSnapshot.VariantBindings[0].Variant.ProcessorContractVersion != "provider.media.transform@1" ||
		firstSnapshot.VariantBindings[0].Variant.ProcessorPackageDigest != strings.Repeat("3", 64) {
		t.Fatalf("pending dependency missing from snapshot: %#v", firstSnapshot.VariantBindings)
	}

	changed := publication
	changed.Variants = append([]VariantDeclaration(nil), publication.Variants...)
	changed.Variants[0].ProcessorPackageDigest = strings.Repeat("9", 64)
	second := New()
	if _, err := second.ReplaceAll([]Publication{corePublicationForTest(), changed}, false); err != nil {
		t.Fatal(err)
	}
	if firstSnapshot.Digest == second.Snapshot().Digest {
		t.Fatal("dormant dependency declaration was omitted from graph digest")
	}

	// Snapshot 返回值必须与 immutable registry state 隔离。
	firstSnapshot.VariantBindings[0].Variant.ProcessorContractVersion = "mutated@9"
	firstSnapshot.Publications[1].Variants[0].ProcessorOwnerExtensionID = "mutated.owner"
	fresh := first.Snapshot()
	if fresh.VariantBindings[0].Variant.ProcessorContractVersion != "provider.media.transform@1" {
		t.Fatalf("snapshot mutation reached registry state: %#v", fresh.VariantBindings[0])
	}
}

func TestCrossPublicationVariantRequiresExplicitProcessorIdentity(t *testing.T) {
	publication := Publication{
		Artifact: pluginArtifactForTest("dependent.media", '5'),
		Variants: []VariantDeclaration{{
			ID: "dependent.media.thumbnail", ContractVersion: "dependent.media.thumbnail@1",
			Purpose: "general", Name: "thumbnail", ProcessorID: "provider.media.transform", OutputMIME: "image/webp",
		}},
	}
	registry := New()
	if _, err := registry.ReplaceAll([]Publication{corePublicationForTest(), publication}, false); !errors.Is(err, ErrInvalid) {
		t.Fatalf("implicit cross-publication dependency accepted: %v", err)
	}
	publication.Variants[0].ProcessorOwnerExtensionID = "provider.media"
	if _, err := registry.ReplaceAll([]Publication{corePublicationForTest(), publication}, false); !errors.Is(err, ErrInvalid) {
		t.Fatalf("partial dependency identity accepted: %v", err)
	}
	publication.Variants[0].ProcessorContractVersion = "provider.media.transform@1"
	if _, err := registry.ReplaceAll([]Publication{corePublicationForTest(), publication}, false); !errors.Is(err, ErrInvalid) {
		t.Fatalf("dependency without exact package accepted: %v", err)
	}
	publication.Variants[0].ProcessorPackageDigest = strings.Repeat("3", 64)
	if _, err := registry.ReplaceAll([]Publication{corePublicationForTest(), publication}, false); err != nil {
		t.Fatalf("complete orphan dependency identity rejected: %v", err)
	}
}

func TestPlanConflictsArePurposeAndStepRelevant(t *testing.T) {
	avatarHigh := Publication{Artifact: pluginArtifactForTest("avatar.high", 'a'), Policies: []MIMEPolicyDeclaration{{
		ID: "avatar.high.policy", ContractVersion: "avatar.high.policy@1", Purpose: "avatar", Priority: 20,
		RequiredPermission: "attachment.upload", AllowedMIMEs: []string{"image/png"}, StrictDeclaredMIME: true, Budget: DefaultBudget(),
	}}}
	avatarLow := Publication{Artifact: pluginArtifactForTest("avatar.low", 'c'), Policies: []MIMEPolicyDeclaration{{
		ID: "avatar.low.policy", ContractVersion: "avatar.low.policy@1", Purpose: "avatar", Priority: 10,
		RequiredPermission: "attachment.upload", AllowedMIMEs: []string{"image/png"}, StrictDeclaredMIME: true, Budget: DefaultBudget(),
	}}}
	wildcardHigh := Publication{Artifact: pluginArtifactForTest("wildcard.high", 'e'), Policies: []MIMEPolicyDeclaration{{
		ID: "wildcard.high.policy", ContractVersion: "wildcard.high.policy@1", Purpose: "*", Priority: 20,
		RequiredPermission: "attachment.upload", AllowedMIMEs: []string{"image/png"}, StrictDeclaredMIME: true, Budget: DefaultBudget(),
	}}}
	wildcardLow := Publication{Artifact: pluginArtifactForTest("wildcard.low", '7'), Policies: []MIMEPolicyDeclaration{{
		ID: "wildcard.low.policy", ContractVersion: "wildcard.low.policy@1", Purpose: "*", Priority: 10,
		RequiredPermission: "attachment.upload", AllowedMIMEs: []string{"image/png"}, StrictDeclaredMIME: true, Budget: DefaultBudget(),
	}}}
	registry := New()
	if _, err := registry.ReplaceAll([]Publication{
		corePublicationForTest(), pluginPublicationForTest(), competingPublicationForTest(), avatarHigh, avatarLow, wildcardHigh, wildcardLow,
	}, false); err != nil {
		t.Fatal(err)
	}
	snapshot := registry.Snapshot()
	plan, err := registry.Plan(t.Context(), uploadRequestForTest(), allowAll())
	if err != nil {
		t.Fatal(err)
	}
	if len(plan.Conflicts) == 0 || len(plan.Conflicts) >= len(snapshot.Conflicts) {
		t.Fatalf("plan conflicts not filtered: plan=%d snapshot=%d", len(plan.Conflicts), len(snapshot.Conflicts))
	}
	seen := map[string]struct{}{}
	for _, conflict := range plan.Conflicts {
		key := conflict.Family + ":" + conflict.Key
		if _, dup := seen[key]; dup {
			t.Fatalf("duplicate conflict key: %s", key)
		}
		seen[key] = struct{}{}
		switch conflict.Family {
		case ConflictMIMEPolicy:
			if conflict.Key != "general" {
				t.Fatalf("irrelevant MIME policy conflict: %#v", conflict)
			}
		case ConflictProcessor:
			matched := false
			for _, step := range plan.Steps {
				if step.Processor.Mode == ProcessorExclusive && processorConflictKey(step.Processor) == conflict.Key {
					matched = true
					break
				}
			}
			if !matched {
				t.Fatalf("processor conflict not tied to plan steps: %#v", conflict)
			}
		case ConflictVariant:
			matched := false
			for _, step := range plan.Steps {
				for _, variant := range step.Variants {
					if variantConflictKey(variant.Purpose, variant.Name) == conflict.Key {
						matched = true
					}
				}
			}
			if !matched {
				t.Fatalf("variant conflict not tied to plan steps: %#v", conflict)
			}
		default:
			t.Fatalf("unknown conflict family: %#v", conflict)
		}
	}
	// 同一 plan 重复计算必须稳定有界。
	again, err := registry.Plan(t.Context(), uploadRequestForTest(), allowAll())
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(plan.Conflicts, again.Conflicts) {
		t.Fatalf("relevant conflicts not deterministic: %#v vs %#v", plan.Conflicts, again.Conflicts)
	}
}
