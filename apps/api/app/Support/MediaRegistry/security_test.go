package mediaregistry

import (
	"context"
	"errors"
	"testing"
)

func TestPlanRejectsTraversalMIMEConfusionAndResourceBombs(t *testing.T) {
	registry := New()
	if _, err := registry.Publish(corePublicationForTest()); err != nil {
		t.Fatal(err)
	}
	tests := []struct {
		name   string
		mutate func(*PlanRequest)
		target error
	}{
		{"filename traversal", func(r *PlanRequest) { r.Source.Filename = "../photo.png" }, ErrInvalid},
		{"windows traversal", func(r *PlanRequest) { r.Source.Filename = `C:\photo.png` }, ErrInvalid},
		{"variant source", func(r *PlanRequest) { r.Source.Kind = SourceVariant }, ErrInvalid},
		{"mutable source", func(r *PlanRequest) { r.Source.Immutable = false }, ErrInvalid},
		{"file bytes", func(r *PlanRequest) { r.Source.SizeBytes = DefaultBudget().MaxFileBytes + 1 }, ErrBudgetExceeded},
		{"file count", func(r *PlanRequest) { r.Upload.BatchFileCount = DefaultBudget().MaxFiles + 1 }, ErrBudgetExceeded},
		{"MIME candidates", func(r *PlanRequest) {
			r.Upload.DetectedMIMEs = []string{"image/png", "image/jpeg", "image/gif", "image/webp", "image/bmp"}
		}, ErrBudgetExceeded},
		{"missing sniffed MIME", func(r *PlanRequest) { r.Upload.DetectedMIMEs = []string{"image/jpeg"} }, ErrMIMEConfusion},
		{"omitted declared MIME under strict policy", func(r *PlanRequest) { r.Upload.DeclaredMIME = "" }, ErrMIMEConfusion},
		{"declared MIME confusion", func(r *PlanRequest) { r.Upload.DeclaredMIME = "text/html" }, ErrMIMEConfusion},
		{"extension confusion", func(r *PlanRequest) { r.Source.Filename = "photo.exe" }, ErrMediaRejected},
		{"SVG denied", func(r *PlanRequest) {
			r.Source.MIME = "image/svg+xml"
			r.Source.Filename = "photo.png"
			r.Upload.DeclaredMIME = "image/svg+xml"
			r.Upload.DetectedMIMEs = []string{"image/svg+xml"}
		}, ErrMediaRejected},
		{"archive unknown size", func(r *PlanRequest) { r.Upload.Archive = true }, ErrBudgetExceeded},
		{"decompressed bytes", func(r *PlanRequest) {
			r.Upload.Archive = true
			r.Upload.DecompressedBytes = DefaultBudget().MaxDecompressedBytes + 1
		}, ErrBudgetExceeded},
		{"decompression ratio", func(r *PlanRequest) {
			r.Upload.Archive = true
			r.Upload.DecompressedBytes = r.Source.SizeBytes*DefaultBudget().MaxDecompressionRatio + 1
		}, ErrBudgetExceeded},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			request := uploadRequestForTest()
			test.mutate(&request)
			_, err := registry.Plan(t.Context(), request, allowAll())
			if !errors.Is(err, test.target) {
				t.Fatalf("got %v, want %v", err, test.target)
			}
		})
	}
}

func TestPlanRequiresAuthoritativeActorPermissionRecheck(t *testing.T) {
	registry := registryWithMediaForTest()
	request := uploadRequestForTest()
	if _, err := registry.Plan(t.Context(), request, nil); !errors.Is(err, ErrInvalid) {
		t.Fatalf("nil authorizer: %v", err)
	}
	request.Actor.ID = ""
	if _, err := registry.Plan(t.Context(), request, allowAll()); !errors.Is(err, ErrInvalid) {
		t.Fatalf("missing actor: %v", err)
	}
	request = uploadRequestForTest()
	request.Kind = PlanDelete
	denied := authorizerFunc(func(_ context.Context, input AuthorizationRequest) bool {
		return input.Permission != "attachment.manage"
	})
	if _, err := registry.Plan(t.Context(), request, denied); !errors.Is(err, ErrPermissionDenied) {
		t.Fatalf("processor permission bypass: %v", err)
	}
}

func TestExplicitImmutableSourceOfTruthIsPreserved(t *testing.T) {
	registry := New()
	if _, err := registry.Publish(corePublicationForTest()); err != nil {
		t.Fatal(err)
	}
	request := uploadRequestForTest()
	request.Source.Kind = SourceSourceOfTruth
	plan, err := registry.Plan(t.Context(), request, allowAll())
	if err != nil {
		t.Fatal(err)
	}
	if plan.Source.Kind != SourceSourceOfTruth || plan.Source != plan.OriginalFallback || !plan.Source.Immutable {
		t.Fatalf("source-of-truth invariant lost: %#v", plan)
	}
}

func TestForgedCoreAndOversizedDeclarationsFailClosed(t *testing.T) {
	core := corePublicationForTest()
	core.Artifact.coreSeal = [32]byte{}
	if _, err := New().Publish(core); !errors.Is(err, ErrInvalid) {
		t.Fatalf("forged core: %v", err)
	}
	plugin := pluginPublicationForTest()
	plugin.Artifact.Core = true
	plugin.Artifact.VersionID = 0
	plugin.Artifact.RuntimeInstanceID = "core"
	if _, err := New().Publish(plugin); !errors.Is(err, ErrInvalid) {
		t.Fatalf("plugin core claim: %v", err)
	}
	plugin = pluginPublicationForTest()
	plugin.Policies[0].Budget.MaxFileBytes = MaximumBudget().MaxFileBytes + 1
	if _, err := New().Publish(plugin); !errors.Is(err, ErrInvalid) {
		t.Fatalf("oversized policy: %v", err)
	}
	plugin = pluginPublicationForTest()
	plugin.Processors[0].MIMEs = make([]string, maxPatterns+1)
	for index := range plugin.Processors[0].MIMEs {
		plugin.Processors[0].MIMEs[index] = "image/png"
	}
	if _, err := New().Publish(plugin); !errors.Is(err, ErrInvalid) {
		t.Fatalf("oversized MIME declaration: %v", err)
	}
}

func TestPlanSnapshotAndOperationAreDeepCopies(t *testing.T) {
	registry := registryWithMediaForTest()
	plan, err := registry.Plan(t.Context(), uploadRequestForTest(), allowAll())
	if err != nil {
		t.Fatal(err)
	}
	receipts := newTestReceiptAuthority()
	operations, err := BackgroundOperations(t.Context(), receipts, plan, sourcePrerequisitesForTest(t, receipts, plan))
	if err != nil {
		t.Fatal(err)
	}
	if len(operations) != 1 || operations[0].StepID != "scan:demo.media.scan" {
		t.Fatalf("background operations = %d", len(operations))
	}
	originalDigest := plan.Digest
	operations[0].Plan.Policy.AllowedMIMEs[0] = "text/html"
	operations[0].Plan.Upload.DetectedMIMEs[0] = "text/html"
	operations[0].Plan.Steps[0].Processor.MIMEs[0] = "text/html"
	if plan.Digest != originalDigest || plan.Policy.AllowedMIMEs[0] == "text/html" || plan.Upload.DetectedMIMEs[0] == "text/html" || plan.Steps[0].Processor.MIMEs[0] == "text/html" {
		t.Fatal("operation mutation escaped deep copy")
	}
	plan.Policy.AllowedMIMEs[0] = "text/html"
	if err := registry.ValidatePlan(t.Context(), plan, allowAll()); !errors.Is(err, ErrPlanStale) {
		t.Fatalf("mutated plan accepted: %v", err)
	}
	plan, err = registry.Plan(t.Context(), uploadRequestForTest(), allowAll())
	if err != nil {
		t.Fatal(err)
	}
	plan.Source.Filename = "renamed.png"
	if err := registry.ValidatePlan(t.Context(), plan, allowAll()); !errors.Is(err, ErrPlanStale) {
		t.Fatalf("private filename escaped plan digest: %v", err)
	}
	plan, err = registry.Plan(t.Context(), uploadRequestForTest(), allowAll())
	if err != nil {
		t.Fatal(err)
	}
	plan.Actor.ID = "other-user"
	if err := registry.ValidatePlan(t.Context(), plan, allowAll()); !errors.Is(err, ErrPlanStale) {
		t.Fatalf("private actor escaped plan digest: %v", err)
	}
}

func TestForgedCurrentPlanCannotOmitMandatoryScanner(t *testing.T) {
	registry := registryWithMediaForTest()
	plan, err := registry.Plan(t.Context(), uploadRequestForTest(), allowAll())
	if err != nil {
		t.Fatal(err)
	}
	filtered := plan.Steps[:0]
	for _, step := range plan.Steps {
		if step.Processor.Stage != StageScan {
			filtered = append(filtered, step)
		}
	}
	plan.Steps = filtered
	plan.Digest = computePlanDigest(plan)
	if err := registry.ValidatePlan(t.Context(), plan, allowAll()); !errors.Is(err, ErrPlanStale) {
		t.Fatalf("forged plan accepted: %v", err)
	}
	transform, _ := stepByStage(plan, StageTransform)
	receipts := newTestReceiptAuthority()
	operation := operationForStepForTest(t, receipts, plan, transform.ID, 1)
	admission := newTestAdmission(transform.Processor.Artifact)
	called := false
	_, err = NewExecutor(registry, admission, invokerFunc(func(context.Context, Invocation) (ProviderOutput, error) { called = true; return ProviderOutput{}, nil }), receipts, nil).ExecuteOperation(t.Context(), operation, allowAll())
	if !errors.Is(err, ErrPlanStale) || called {
		t.Fatalf("forged operation executed: called=%t err=%v", called, err)
	}
}
