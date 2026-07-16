package mediaregistry

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"testing"
)

type authorizerFunc func(context.Context, AuthorizationRequest) bool

func (fn authorizerFunc) Authorize(ctx context.Context, request AuthorizationRequest) bool {
	return fn(ctx, request)
}

func allowAll() Authorizer {
	return authorizerFunc(func(context.Context, AuthorizationRequest) bool { return true })
}

type testReceiptAuthority struct {
	mu      sync.Mutex
	next    int
	records map[string]struct {
		claim    ReceiptClaim
		evidence ReceiptEvidence
	}
	rejectRecord               bool
	recordCalls                int
	beforeCommit               func()
	panicReceiptLeaseContext   bool
	panicOperationLeaseContext bool
	active                     map[*testReceiptLease]map[string]struct{}
	operations                 map[string]*testOperationState
}

func newTestReceiptAuthority() *testReceiptAuthority {
	return &testReceiptAuthority{records: make(map[string]struct {
		claim    ReceiptClaim
		evidence ReceiptEvidence
	}), active: make(map[*testReceiptLease]map[string]struct{}), operations: make(map[string]*testOperationState)}
}

type testOperationState struct {
	claim      OperationClaim
	active     *testOperationLease
	completion *OperationCompletion
}

type testOperationLease struct {
	ctx       context.Context
	cancel    context.CancelFunc
	authority *testReceiptAuthority
	key       string
	once      sync.Once
}

type panickingOperationLease struct{ *testOperationLease }

func (*panickingOperationLease) Context() context.Context {
	panic("test operation lease context panic")
}

func (lease *testOperationLease) Context() context.Context { return lease.ctx }
func (lease *testOperationLease) Release() {
	lease.once.Do(func() {
		lease.authority.mu.Lock()
		state := lease.authority.operations[lease.key]
		if state != nil && state.active == lease {
			state.active = nil
		}
		lease.authority.mu.Unlock()
		lease.cancel()
	})
}

type testReceiptLease struct {
	ctx       context.Context
	cancel    context.CancelFunc
	authority *testReceiptAuthority
	once      sync.Once
}

type panickingReceiptLease struct{ *testReceiptLease }

func (*panickingReceiptLease) Context() context.Context { panic("test receipt lease context panic") }

func (lease *testReceiptLease) Context() context.Context { return lease.ctx }
func (lease *testReceiptLease) Release() {
	lease.once.Do(func() {
		lease.authority.mu.Lock()
		delete(lease.authority.active, lease)
		lease.authority.mu.Unlock()
		lease.cancel()
	})
}

func (authority *testReceiptAuthority) RecordMediaReceipt(ctx context.Context, hostClaim HostReceiptClaim) (ReceiptEvidence, error) {
	if err := ctx.Err(); err != nil {
		return ReceiptEvidence{}, err
	}
	claim := hostClaim.Claim()
	if claim.Kind == ReceiptKindOperation {
		return ReceiptEvidence{}, errors.New("test ledger requires operation CAS")
	}
	authority.mu.Lock()
	authority.recordCalls++
	evidence, err := authority.recordLocked(claim)
	if err != nil {
		authority.mu.Unlock()
		return ReceiptEvidence{}, err
	}
	authority.mu.Unlock()
	return evidence, nil
}

func (authority *testReceiptAuthority) directRecordCalls() int {
	authority.mu.Lock()
	defer authority.mu.Unlock()
	return authority.recordCalls
}

func (authority *testReceiptAuthority) cancelOperation(key string) {
	authority.mu.Lock()
	defer authority.mu.Unlock()
	if state := authority.operations[key]; state != nil && state.active != nil {
		state.active.cancel()
	}
}

func (authority *testReceiptAuthority) seedOperationReceipt(claim ReceiptClaim) (ReceiptEvidence, error) {
	authority.mu.Lock()
	defer authority.mu.Unlock()
	return authority.recordLocked(claim)
}

func (authority *testReceiptAuthority) recordLocked(claim ReceiptClaim) (ReceiptEvidence, error) {
	if authority.rejectRecord {
		return ReceiptEvidence{}, errors.New("test ledger rejected record")
	}
	for _, existing := range authority.records {
		if existing.claim == claim {
			return existing.evidence, nil
		}
	}
	authority.next++
	evidence := ReceiptEvidence{
		ID:   fmt.Sprintf("media-evidence-%d", authority.next),
		Seal: fmt.Sprintf("test-seal-%d-%s", authority.next, claim.Digest),
	}
	authority.records[evidence.ID] = struct {
		claim    ReceiptClaim
		evidence ReceiptEvidence
	}{claim: claim, evidence: evidence}
	return evidence, nil
}

func (authority *testReceiptAuthority) VerifyMediaReceipt(ctx context.Context, claim ReceiptClaim, evidence ReceiptEvidence) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	authority.mu.Lock()
	defer authority.mu.Unlock()
	record, found := authority.records[evidence.ID]
	if !found || record.claim != claim || record.evidence != evidence {
		return errors.New("test ledger evidence mismatch")
	}
	return nil
}

func (authority *testReceiptAuthority) AcquireMediaReceipts(ctx context.Context, bindings []ReceiptBinding) (ReceiptLease, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	authority.mu.Lock()
	defer authority.mu.Unlock()
	ids := make(map[string]struct{}, len(bindings))
	for _, binding := range bindings {
		record, found := authority.records[binding.Evidence.ID]
		if !found || record.claim != binding.Claim || record.evidence != binding.Evidence {
			return nil, errors.New("test ledger lease mismatch")
		}
		ids[binding.Evidence.ID] = struct{}{}
	}
	leaseCtx, cancel := context.WithCancel(context.Background())
	lease := &testReceiptLease{ctx: leaseCtx, cancel: cancel, authority: authority}
	authority.active[lease] = ids
	if authority.panicReceiptLeaseContext {
		return &panickingReceiptLease{testReceiptLease: lease}, nil
	}
	return lease, nil
}

func (authority *testReceiptAuthority) AcquireMediaOperation(ctx context.Context, claim OperationClaim) (OperationAcquisition, error) {
	if err := ctx.Err(); err != nil {
		return OperationAcquisition{}, err
	}
	authority.mu.Lock()
	defer authority.mu.Unlock()
	state := authority.operations[claim.OperationKey]
	if state != nil && !sameOperationTarget(state.claim, claim) {
		return OperationAcquisition{}, errors.New("test operation target mismatch")
	}
	if state != nil && state.completion != nil {
		completion := cloneOperationCompletion(*state.completion)
		return OperationAcquisition{Replay: &completion}, nil
	}
	if state != nil && state.active != nil && state.active.Context().Err() == nil {
		return OperationAcquisition{}, ErrOperationBusy
	}
	if state == nil {
		state = &testOperationState{claim: claim}
		authority.operations[claim.OperationKey] = state
	} else {
		state.claim = claim
	}
	leaseCtx, cancel := context.WithCancel(context.Background())
	lease := &testOperationLease{ctx: leaseCtx, cancel: cancel, authority: authority, key: claim.OperationKey}
	state.active = lease
	if authority.panicOperationLeaseContext {
		return OperationAcquisition{Lease: &panickingOperationLease{testOperationLease: lease}}, nil
	}
	return OperationAcquisition{Lease: lease}, nil
}

func (authority *testReceiptAuthority) CommitMediaOperation(ctx context.Context, opaque OperationLease, prerequisiteOpaque ReceiptLease, completion OperationCompletion) (ReceiptEvidence, error) {
	if err := ctx.Err(); err != nil {
		return ReceiptEvidence{}, err
	}
	lease, ok := opaque.(*testOperationLease)
	prerequisites, prerequisitesOK := prerequisiteOpaque.(*testReceiptLease)
	if !ok || lease == nil || lease.authority != authority || lease.Context().Err() != nil ||
		!prerequisitesOK || prerequisites == nil || prerequisites.authority != authority || prerequisites.Context().Err() != nil ||
		!validOperationCompletionShape(completion) {
		return ReceiptEvidence{}, errors.New("test operation lease mismatch")
	}
	if authority.beforeCommit != nil {
		authority.beforeCommit()
	}
	authority.mu.Lock()
	state := authority.operations[lease.key]
	claim := operationReceiptClaim(completion.Receipt)
	_, prerequisitesActive := authority.active[prerequisites]
	if state == nil || state.active != lease || !prerequisitesActive || prerequisites.Context().Err() != nil ||
		!sameOperationTarget(state.claim, operationClaimFromReceipt(completion.Receipt)) ||
		completion.Receipt.Evidence != (ReceiptEvidence{}) || completion.Receipt.Digest != receiptIntegrityDigest(claim) {
		authority.mu.Unlock()
		return ReceiptEvidence{}, errors.New("test operation completion mismatch")
	}
	evidence, err := authority.recordLocked(claim)
	if err != nil {
		authority.mu.Unlock()
		return ReceiptEvidence{}, err
	}
	completion.Receipt.Evidence = evidence
	stored := cloneOperationCompletion(completion)
	state.completion = &stored
	authority.mu.Unlock()
	return evidence, nil
}

func (authority *testReceiptAuthority) forget(evidence ReceiptEvidence) {
	authority.mu.Lock()
	delete(authority.records, evidence.ID)
	for lease, ids := range authority.active {
		if _, found := ids[evidence.ID]; found {
			lease.cancel()
		}
	}
	authority.mu.Unlock()
}

type invokerFunc func(context.Context, Invocation) (ProviderOutput, error)

func (fn invokerFunc) Invoke(ctx context.Context, invocation Invocation) (ProviderOutput, error) {
	return fn(ctx, invocation)
}

type testLease struct {
	ctx           context.Context
	artifact      Artifact
	release       func()
	once          sync.Once
	panicContext  bool
	panicArtifact bool
}

func (lease *testLease) Context() context.Context {
	if lease.panicContext {
		panic("test runtime lease context panic")
	}
	return lease.ctx
}
func (lease *testLease) Artifact() Artifact {
	if lease.panicArtifact {
		panic("test runtime lease artifact panic")
	}
	return lease.artifact
}
func (lease *testLease) Release() {
	lease.once.Do(func() {
		if lease.release != nil {
			lease.release()
		}
	})
}

type testAdmission struct {
	mu                 sync.Mutex
	available          map[Artifact]bool
	leaseAs            Artifact
	leaseCtx           context.Context
	active             int
	released           int
	onRelease          func()
	panicRelease       bool
	panicLeaseContext  bool
	panicLeaseArtifact bool
}

func newTestAdmission(artifacts ...Artifact) *testAdmission {
	result := &testAdmission{available: map[Artifact]bool{}}
	for _, artifact := range artifacts {
		result.available[artifact] = true
	}
	return result
}

func (admission *testAdmission) Available(artifact Artifact) bool {
	admission.mu.Lock()
	defer admission.mu.Unlock()
	return admission.available[artifact]
}

func (admission *testAdmission) Acquire(ctx context.Context, artifact Artifact) (RuntimeLease, error) {
	admission.mu.Lock()
	if !admission.available[artifact] {
		admission.mu.Unlock()
		return nil, ErrRuntimeUnavailable
	}
	leasedArtifact := artifact
	if admission.leaseAs.ExtensionID != "" {
		leasedArtifact = admission.leaseAs
	}
	leaseContext := admission.leaseCtx
	if leaseContext == nil {
		leaseContext = context.WithoutCancel(ctx)
	}
	admission.active++
	admission.mu.Unlock()
	return &testLease{ctx: leaseContext, artifact: leasedArtifact, panicContext: admission.panicLeaseContext, panicArtifact: admission.panicLeaseArtifact, release: func() {
		admission.mu.Lock()
		admission.active--
		admission.released++
		onRelease := admission.onRelease
		panicRelease := admission.panicRelease
		admission.mu.Unlock()
		if onRelease != nil {
			onRelease()
		}
		if panicRelease {
			panic("test runtime release panic")
		}
	}}, nil
}

func (admission *testAdmission) counts() (int, int) {
	admission.mu.Lock()
	defer admission.mu.Unlock()
	return admission.active, admission.released
}

func (authority *testReceiptAuthority) operationCompletion(key string) *OperationCompletion {
	authority.mu.Lock()
	defer authority.mu.Unlock()
	state := authority.operations[key]
	if state == nil || state.completion == nil {
		return nil
	}
	completion := cloneOperationCompletion(*state.completion)
	return &completion
}

func (authority *testReceiptAuthority) activeReceiptLeases() int {
	authority.mu.Lock()
	defer authority.mu.Unlock()
	return len(authority.active)
}

func coreArtifactForTest() Artifact {
	artifact, err := NewCoreArtifact("core.media", "1.0.0", strings.Repeat("1", 64), strings.Repeat("2", 64))
	if err != nil {
		panic(err)
	}
	return artifact
}

func pluginArtifactForTest(extensionID string, marker byte) Artifact {
	return Artifact{ExtensionID: extensionID, ExtensionVersion: "1.0.0", PackageDigest: strings.Repeat(string(marker), 64), ImpactDigest: strings.Repeat(string(marker+1), 64), VersionID: 1, RuntimeInstanceID: extensionID + "-runtime"}
}

func corePublicationForTest() Publication {
	return Publication{Artifact: coreArtifactForTest(), Policies: []MIMEPolicyDeclaration{{
		ID: "core.media.policy", ContractVersion: "core.media.policy@1", Purpose: "general", RequiredPermission: "attachment.upload",
		AllowedMIMEs: []string{"image/*"}, DeniedMIMEs: []string{"image/svg+xml"}, AllowedExtensions: []string{"jpg", "png"}, StrictDeclaredMIME: true, Budget: DefaultBudget(),
	}}}
}

func pluginPublicationForTest() Publication {
	artifact := pluginArtifactForTest("demo.media", '3')
	retry := RetryPolicy{MaxAttempts: 3, BaseDelaySeconds: 2, MaxDelaySeconds: 30}
	return Publication{Artifact: artifact,
		Policies: []MIMEPolicyDeclaration{{ID: "demo.media.policy", ContractVersion: "demo.media.policy@1", Purpose: "general", Priority: 20, RequiredPermission: "attachment.upload", AllowedMIMEs: []string{"image/png"}, AllowedExtensions: []string{"png"}, StrictDeclaredMIME: true, Budget: DefaultBudget()}},
		Processors: []ProcessorDeclaration{
			{ID: "demo.media.scan", ContractVersion: "demo.media.scan@1", Stage: StageScan, Purpose: "general", MIMEs: []string{"image/*"}, Handler: "scan", Priority: 10, Mode: ProcessorCompose, Execution: ExecutionBackground, FailureMode: FailureFailClosed, RequiredPermission: "attachment.upload", Retry: retry},
			{ID: "demo.media.metadata", ContractVersion: "demo.media.metadata@1", Stage: StageMetadata, Purpose: "general", MIMEs: []string{"image/*"}, Handler: "metadata", Mode: ProcessorCompose, Execution: ExecutionSync, FailureMode: FailureSkip, RequiredPermission: "attachment.upload"},
			{ID: "demo.media.transform", ContractVersion: "demo.media.transform@1", Stage: StageTransform, Purpose: "general", MIMEs: []string{"image/*"}, Handler: "transform", Priority: 10, Mode: ProcessorCompose, Execution: ExecutionBackground, FailureMode: FailureFallbackOriginal, RequiredPermission: "attachment.upload", Retry: retry},
			{ID: "demo.media.cdn", ContractVersion: "demo.media.cdn@1", Stage: StageCDN, Purpose: "general", MIMEs: []string{"image/*"}, Handler: "cdn", Priority: 10, Mode: ProcessorExclusive, Slot: "primary.cdn", Execution: ExecutionSync, FailureMode: FailureFallbackOriginal, RequiredPermission: "attachment.upload"},
			{ID: "demo.media.retention", ContractVersion: "demo.media.retention@1", Stage: StageRetention, Purpose: "general", MIMEs: []string{"image/*"}, Handler: "retention", Mode: ProcessorCompose, Execution: ExecutionSync, FailureMode: FailureSkip, RequiredPermission: "attachment.upload"},
			{ID: "demo.media.before.delete", ContractVersion: "demo.media.before.delete@1", Stage: StageBeforeDelete, Purpose: "general", MIMEs: []string{"image/*"}, Handler: "before_delete", Mode: ProcessorCompose, Execution: ExecutionBackground, FailureMode: FailureFailClosed, RequiredPermission: "attachment.manage", Retry: retry},
			{ID: "demo.media.after.delete", ContractVersion: "demo.media.after.delete@1", Stage: StageAfterDelete, Purpose: "general", MIMEs: []string{"image/*"}, Handler: "after_delete", Mode: ProcessorCompose, Execution: ExecutionBackground, FailureMode: FailureSkip, RequiredPermission: "attachment.manage", Retry: retry},
		},
		Variants: []VariantDeclaration{{
			ID: "demo.media.thumbnail", ContractVersion: "demo.media.thumbnail@1", Purpose: "general", Name: "thumbnail",
			ProcessorID: "demo.media.transform", ProcessorContractVersion: "demo.media.transform@1",
			ProcessorOwnerExtensionID: "demo.media", ProcessorPackageDigest: artifact.PackageDigest,
			OutputMIME: "image/webp", Priority: 10,
		}},
	}
}

func competingPublicationForTest() Publication {
	artifact := pluginArtifactForTest("alternate.media", '5')
	return Publication{Artifact: artifact,
		Processors: []ProcessorDeclaration{
			{ID: "alternate.media.transform", ContractVersion: "alternate.media.transform@1", Stage: StageTransform, Purpose: "general", MIMEs: []string{"image/*"}, Handler: "transform", Priority: 30, Mode: ProcessorCompose, Execution: ExecutionSync, FailureMode: FailureFallbackOriginal, RequiredPermission: "attachment.upload"},
			{ID: "alternate.media.cdn", ContractVersion: "alternate.media.cdn@1", Stage: StageCDN, Purpose: "general", MIMEs: []string{"image/*"}, Handler: "cdn", Priority: 30, Mode: ProcessorExclusive, Slot: "primary.cdn", Execution: ExecutionSync, FailureMode: FailureFallbackOriginal, RequiredPermission: "attachment.upload"},
		},
		Variants: []VariantDeclaration{{
			ID: "alternate.media.thumbnail", ContractVersion: "alternate.media.thumbnail@1", Purpose: "general", Name: "thumbnail",
			ProcessorID: "alternate.media.transform", ProcessorContractVersion: "alternate.media.transform@1",
			ProcessorOwnerExtensionID: "alternate.media", ProcessorPackageDigest: artifact.PackageDigest,
			OutputMIME: "image/avif", Priority: 30,
		}},
	}
}

func sourceForTest() SourceAsset {
	return SourceAsset{ID: "attachment-42", Digest: strings.Repeat("a", 64), Kind: SourceOriginal, MIME: "image/png", Filename: "photo.png", SizeBytes: 1024, Immutable: true}
}

func uploadRequestForTest() PlanRequest {
	return PlanRequest{Kind: PlanUpload, Purpose: "general", Permission: "attachment.upload", Actor: Actor{ID: "user-7", PermissionFingerprint: "permissions-v1"}, Source: sourceForTest(), Upload: UploadFacts{BatchFileCount: 1, DeclaredMIME: "image/png", DetectedMIMEs: []string{"image/png"}}}
}

func registryWithMediaForTest() *Registry {
	registry := New()
	if _, err := registry.ReplaceAll([]Publication{corePublicationForTest(), pluginPublicationForTest()}, false); err != nil {
		panic(err)
	}
	return registry
}

func planWithMediaForTest() Plan {
	plan, err := registryWithMediaForTest().Plan(context.Background(), uploadRequestForTest(), allowAll())
	if err != nil {
		panic(err)
	}
	return plan
}

func stepByStage(plan Plan, stage string) (PlanStep, bool) {
	for _, step := range plan.Steps {
		if step.Processor.Stage == stage {
			return step, true
		}
	}
	return PlanStep{}, false
}

func sourcePrerequisitesForTest(t testing.TB, authority ReceiptAuthority, plan Plan) OperationPrerequisites {
	t.Helper()
	receipt, err := RecordSourceReceipt(context.Background(), authority, plan)
	if err != nil {
		t.Fatal(err)
	}
	return OperationPrerequisites{Source: receipt}
}

func operationForStepForTest(t testing.TB, authority ReceiptAuthority, plan Plan, stepID string, attempt int) BackgroundOperation {
	t.Helper()
	prerequisites := prerequisitesBeforeStepForTest(t, authority, plan, stepID)
	operation, err := OperationForStep(context.Background(), authority, plan, stepID, attempt, prerequisites)
	if err != nil {
		t.Fatal(err)
	}
	return operation
}

func prerequisitesBeforeStepForTest(t testing.TB, authority ReceiptAuthority, plan Plan, stepID string) OperationPrerequisites {
	t.Helper()
	target := planStepIndex(plan, stepID)
	if target < 0 {
		t.Fatalf("step %s is absent", stepID)
	}
	prerequisites := sourcePrerequisitesForTest(t, authority, plan)
	prerequisites.Steps = make([]OperationReceipt, 0, target)
	for index := 0; index < target; index++ {
		step := plan.Steps[index]
		operation, err := OperationForStep(context.Background(), authority, plan, step.ID, 1, prerequisites)
		if err != nil {
			t.Fatal(err)
		}
		receipt, err := recordOperationReceiptForTest(context.Background(), authority, operation, step, ProviderOutput{}, TraceSucceeded)
		if err != nil {
			t.Fatal(err)
		}
		prerequisites.Steps = append(prerequisites.Steps, receipt)
	}
	return prerequisites
}

func recordOperationReceiptForTest(ctx context.Context, authority ReceiptAuthority, operation BackgroundOperation, step PlanStep, output ProviderOutput, outcome string) (OperationReceipt, error) {
	receipt, err := operationReceiptTemplate(ctx, authority, operation, step, output, outcome)
	if err != nil {
		return OperationReceipt{}, err
	}
	seeder, ok := authority.(interface {
		seedOperationReceipt(ReceiptClaim) (ReceiptEvidence, error)
	})
	if !ok {
		return OperationReceipt{}, ErrReceiptAuthority
	}
	claim := operationReceiptClaim(receipt)
	evidence, err := seeder.seedOperationReceipt(claim)
	if err != nil {
		return OperationReceipt{}, err
	}
	if err := verifyReceiptEvidence(ctx, authority, claim, evidence); err != nil {
		return OperationReceipt{}, err
	}
	receipt.Evidence = evidence
	return receipt, nil
}
