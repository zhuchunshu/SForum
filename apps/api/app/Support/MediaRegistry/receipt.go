package mediaregistry

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
)

const (
	OperationReceiptSchemaVersion = "sforum.media-operation-receipt@1"
	SourceReceiptSchemaVersion    = "sforum.media-source-receipt@1"
	DeletionReceiptSchemaVersion  = "sforum.media-deletion-receipt@1"
	ReceiptClaimSchemaVersion     = "sforum.media-receipt-claim@1"

	ReceiptKindSourceAdmission  = "source_admission"
	ReceiptKindOperation        = "operation_completion"
	ReceiptKindDeletionComplete = "deletion_complete"

	receiptOutcomeSourceCommitted = "source_committed"
	receiptOutcomeDeleted         = "deleted"
)

var (
	ErrPredecessorRequired = errors.New("media registry predecessor receipt is required")
	ErrReceiptInvalid      = errors.New("media registry predecessor receipt is invalid")
)

// ReceiptEvidence is opaque Host-owned durable-ledger evidence. MediaRegistry
// validates only its shape; authority implementations own lookup, sealing,
// revocation, and exact claim comparison.
type ReceiptEvidence struct {
	ID   string `json:"id"`
	Seal string `json:"seal"`
}

// ReceiptClaim is the canonical, privacy-bounded value a Host ledger records.
// Implementations must compare the entire claim, not only Digest or Evidence.
type ReceiptClaim struct {
	SchemaVersion     string               `json:"schemaVersion"`
	ReceiptSchema     string               `json:"receiptSchema"`
	Kind              string               `json:"kind"`
	PlanDigest        string               `json:"planDigest"`
	RegistryDigest    string               `json:"registryDigest"`
	PlanKind          string               `json:"planKind"`
	SourceDigest      string               `json:"sourceDigest"`
	Artifact          Artifact             `json:"artifact"`
	OperationKey      string               `json:"operationKey,omitempty"`
	StepID            string               `json:"stepId,omitempty"`
	Stage             string               `json:"stage,omitempty"`
	Attempt           int                  `json:"attempt,omitempty"`
	Outcome           string               `json:"outcome"`
	PredecessorDigest string               `json:"predecessorDigest,omitempty"`
	DeltaUsage        OperationBudgetUsage `json:"deltaUsage"`
	CumulativeUsage   OperationBudgetUsage `json:"cumulativeUsage"`
	OutputDigest      string               `json:"outputDigest,omitempty"`
	Digest            string               `json:"digest"`
}

// HostReceiptClaim is minted only by the source/deletion helpers in this
// package. The generic durable writer therefore cannot be called with an
// operation-completion claim; those terminals have exactly one production
// path through CommitMediaOperation and its two live CAS leases.
//
// Host implementations may inspect the canonical value through Claim. The
// zero value is invalid and must never be persisted.
type HostReceiptClaim struct {
	claim ReceiptClaim
}

func (value HostReceiptClaim) Claim() ReceiptClaim { return value.claim }

type ReceiptBinding struct {
	Claim    ReceiptClaim
	Evidence ReceiptEvidence
}

// ReceiptLease keeps every prerequisite evidence valid while a provider may
// perform side effects. Revocation must cancel Context before it can complete;
// Release ends the Host-owned read/lease fence.
type ReceiptLease interface {
	Context() context.Context
	Release()
}

// ReceiptAuthority is implemented by Host-owned protected storage.
// RecordMediaReceipt accepts an unforgeable source/deletion claim, makes the
// exact claim durable, and returns the same evidence for exact replay.
// Operation claims can be written only by CommitMediaOperation. Verify fails when
// evidence is missing, revoked, unknown, forged, or bound to any other claim.
// AcquireMediaReceipts atomically verifies prerequisite bindings and holds a
// revocation-aware lease. AcquireMediaOperation returns either the sole live
// exact-operation claim, an exact terminal replay, or ErrOperationBusy.
// CommitMediaOperation atomically compares both the target-operation and
// prerequisite leases, records receipt evidence, and stores the replayable
// completion. Success is the final linearization point and must return valid
// exact evidence. An error guarantees no terminal; implementations must resolve
// ambiguous storage outcomes internally. Invalid terminals never reopen work.
type ReceiptAuthority interface {
	RecordMediaReceipt(context.Context, HostReceiptClaim) (ReceiptEvidence, error)
	VerifyMediaReceipt(context.Context, ReceiptClaim, ReceiptEvidence) error
	AcquireMediaReceipts(context.Context, []ReceiptBinding) (ReceiptLease, error)
	AcquireMediaOperation(context.Context, OperationClaim) (OperationAcquisition, error)
	CommitMediaOperation(context.Context, OperationLease, ReceiptLease, OperationCompletion) (ReceiptEvidence, error)
}

type SourceReceipt struct {
	SchemaVersion  string          `json:"schemaVersion"`
	PlanDigest     string          `json:"planDigest"`
	RegistryDigest string          `json:"registryDigest"`
	PlanKind       string          `json:"planKind"`
	SourceDigest   string          `json:"sourceDigest"`
	PolicyArtifact Artifact        `json:"policyArtifact"`
	Digest         string          `json:"digest"`
	Evidence       ReceiptEvidence `json:"evidence"`
}

type DeletionReceipt struct {
	SchemaVersion     string          `json:"schemaVersion"`
	PlanDigest        string          `json:"planDigest"`
	RegistryDigest    string          `json:"registryDigest"`
	PlanKind          string          `json:"planKind"`
	SourceDigest      string          `json:"sourceDigest"`
	PolicyArtifact    Artifact        `json:"policyArtifact"`
	PredecessorDigest string          `json:"predecessorDigest"`
	Digest            string          `json:"digest"`
	Evidence          ReceiptEvidence `json:"evidence"`
}

// OperationPrerequisites travel only inside a Host-protected operation
// envelope. Source is mandatory for every operation; Deletion is additionally
// mandatory for after_delete.
type OperationPrerequisites struct {
	Source   SourceReceipt      `json:"source"`
	Steps    []OperationReceipt `json:"steps,omitempty"`
	Deletion *DeletionReceipt   `json:"deletion,omitempty"`
}

// OperationBudgetUsage is cumulative across one exact plan. A later operation
// cannot reset policy ceilings by switching processors or execution modes.
type OperationBudgetUsage struct {
	MetadataBytes int `json:"metadataBytes,omitempty"`
	Variants      int `json:"variants,omitempty"`
}

// OperationReceipt is durable completion evidence for one exact plan step.
// Digest is only a corruption checksum. Evidence plus ReceiptAuthority is the
// authority boundary and must survive queue/restart round trips.
type OperationReceipt struct {
	SchemaVersion     string               `json:"schemaVersion"`
	PlanDigest        string               `json:"planDigest"`
	RegistryDigest    string               `json:"registryDigest"`
	PlanKind          string               `json:"planKind"`
	SourceDigest      string               `json:"sourceDigest"`
	OperationKey      string               `json:"operationKey"`
	StepID            string               `json:"stepId"`
	Stage             string               `json:"stage"`
	Artifact          Artifact             `json:"artifact"`
	Attempt           int                  `json:"attempt"`
	Outcome           string               `json:"outcome"`
	PredecessorDigest string               `json:"predecessorDigest,omitempty"`
	DeltaUsage        OperationBudgetUsage `json:"deltaUsage"`
	CumulativeUsage   OperationBudgetUsage `json:"cumulativeUsage"`
	OutputDigest      string               `json:"outputDigest"`
	Digest            string               `json:"digest"`
	Evidence          ReceiptEvidence      `json:"evidence"`
}

// RecordSourceReceipt must be called only after the Host has durably committed
// the immutable source/admission state. The authority record, not this helper,
// is the durable proof.
func RecordSourceReceipt(ctx context.Context, authority ReceiptAuthority, plan Plan) (SourceReceipt, error) {
	receipt, err := sourceReceiptTemplate(plan)
	if err != nil {
		return SourceReceipt{}, err
	}
	evidence, err := recordReceiptEvidence(ctx, authority, sourceReceiptClaim(receipt))
	if err != nil {
		return SourceReceipt{}, err
	}
	receipt.Evidence = evidence
	return receipt, nil
}

// RecordDeletionReceipt must be called only after Host storage has deleted the
// exact source and after every pre-delete step receipt is durable.
func RecordDeletionReceipt(ctx context.Context, authority ReceiptAuthority, plan Plan, prerequisites OperationPrerequisites) (DeletionReceipt, error) {
	boundary := firstAfterDeleteStep(plan)
	if boundary < 0 {
		return DeletionReceipt{}, ErrNotFound
	}
	if prerequisites.Deletion != nil {
		return DeletionReceipt{}, ErrReceiptInvalid
	}
	if _, err := validateSourceAndStepReceipts(ctx, authority, plan, plan.Steps[boundary], prerequisites); err != nil {
		return DeletionReceipt{}, err
	}
	receipt := deletionReceiptTemplate(plan, prerequisites.Steps[:boundary])
	evidence, err := recordReceiptEvidence(ctx, authority, deletionReceiptClaim(receipt))
	if err != nil {
		return DeletionReceipt{}, err
	}
	receipt.Evidence = evidence
	return receipt, nil
}

func validateOperationPrerequisites(ctx context.Context, authority ReceiptAuthority, plan Plan, target PlanStep, prerequisites OperationPrerequisites) (OperationBudgetUsage, error) {
	usage, err := validateSourceAndStepReceipts(ctx, authority, plan, target, prerequisites)
	if err != nil {
		return OperationBudgetUsage{}, err
	}
	if target.Processor.Stage == StageAfterDelete {
		if prerequisites.Deletion == nil {
			return OperationBudgetUsage{}, ErrDeletionFence
		}
		if err := verifyDeletionReceipt(ctx, authority, plan, prerequisites.Steps, *prerequisites.Deletion); err != nil {
			return OperationBudgetUsage{}, err
		}
	} else if prerequisites.Deletion != nil {
		return OperationBudgetUsage{}, ErrReceiptInvalid
	}
	return usage, nil
}

func validateSourceAndStepReceipts(ctx context.Context, authority ReceiptAuthority, plan Plan, target PlanStep, prerequisites OperationPrerequisites) (OperationBudgetUsage, error) {
	index := planStepIndex(plan, target.ID)
	if index < 0 {
		return OperationBudgetUsage{}, ErrPlanStale
	}
	if err := verifySourceReceipt(ctx, authority, plan, prerequisites.Source); err != nil {
		return OperationBudgetUsage{}, err
	}
	if len(prerequisites.Steps) != index {
		return OperationBudgetUsage{}, ErrPredecessorRequired
	}
	usage := OperationBudgetUsage{}
	predecessorDigest := ""
	for receiptIndex, receipt := range prerequisites.Steps {
		expected := plan.Steps[receiptIndex]
		if err := verifyOperationReceipt(ctx, authority, plan, expected, predecessorDigest, usage, receipt); err != nil {
			return OperationBudgetUsage{}, fmt.Errorf("%w: step %s", err, expected.ID)
		}
		usage = receipt.CumulativeUsage
		predecessorDigest = extendPredecessorDigest(predecessorDigest, receipt)
	}
	return usage, nil
}

func operationReceiptTemplate(ctx context.Context, authority ReceiptAuthority, operation BackgroundOperation, step PlanStep, output ProviderOutput, outcome string) (OperationReceipt, error) {
	usage, err := validateOperationPrerequisites(ctx, authority, operation.Plan, step, operation.Prerequisites)
	if err != nil {
		return OperationReceipt{}, err
	}
	if !validReceiptOutcomeForStep(outcome, step) {
		return OperationReceipt{}, ErrReceiptInvalid
	}
	delta := providerOutputBudgetUsage(output)
	cumulative := OperationBudgetUsage{
		MetadataBytes: usage.MetadataBytes + delta.MetadataBytes,
		Variants:      usage.Variants + delta.Variants,
	}
	if !usageWithinBudget(cumulative, operation.Plan.Policy.Budget) {
		return OperationReceipt{}, ErrBudgetExceeded
	}
	receipt := OperationReceipt{
		SchemaVersion: OperationReceiptSchemaVersion, PlanDigest: operation.Plan.Digest,
		RegistryDigest: operation.Plan.RegistryDigest, PlanKind: operation.Plan.Kind,
		SourceDigest: operation.Plan.Source.Digest, OperationKey: operation.Key,
		StepID: step.ID, Stage: step.Processor.Stage, Artifact: receiptArtifact(step.Processor.Artifact),
		Attempt: operation.Attempt, Outcome: outcome,
		PredecessorDigest: predecessorReceiptDigest(operation.Prerequisites.Steps),
		DeltaUsage:        delta, CumulativeUsage: cumulative, OutputDigest: providerOutputDigest(output),
	}
	receipt.Digest = receiptIntegrityDigest(operationReceiptClaim(receipt))
	return receipt, nil
}

func verifySourceReceipt(ctx context.Context, authority ReceiptAuthority, plan Plan, receipt SourceReceipt) error {
	expected, err := sourceReceiptTemplate(plan)
	if err != nil {
		return err
	}
	claim := sourceReceiptClaim(receipt)
	if claim != sourceReceiptClaim(expected) {
		return ErrReceiptInvalid
	}
	return verifyReceiptEvidence(ctx, authority, claim, receipt.Evidence)
}

func verifyOperationReceipt(ctx context.Context, authority ReceiptAuthority, plan Plan, expected PlanStep, predecessorDigest string, previousUsage OperationBudgetUsage, receipt OperationReceipt) error {
	if receipt.SchemaVersion != OperationReceiptSchemaVersion || receipt.PlanDigest != plan.Digest ||
		receipt.RegistryDigest != plan.RegistryDigest || receipt.PlanKind != plan.Kind ||
		receipt.SourceDigest != plan.Source.Digest || receipt.OperationKey != operationKey(plan, expected) ||
		receipt.StepID != expected.ID || receipt.Stage != expected.Processor.Stage ||
		!artifactIdentityEqual(receipt.Artifact, expected.Processor.Artifact) ||
		receipt.Attempt < 1 || receipt.Attempt > effectiveMaxAttempts(expected.Processor) ||
		!validReceiptOutcomeForStep(receipt.Outcome, expected) ||
		receipt.PredecessorDigest != predecessorDigest ||
		!digestPattern.MatchString(receipt.OutputDigest) ||
		receipt.DeltaUsage.MetadataBytes < 0 || receipt.DeltaUsage.Variants < 0 {
		return ErrReceiptInvalid
	}
	cumulative := OperationBudgetUsage{
		MetadataBytes: previousUsage.MetadataBytes + receipt.DeltaUsage.MetadataBytes,
		Variants:      previousUsage.Variants + receipt.DeltaUsage.Variants,
	}
	if receipt.CumulativeUsage != cumulative || !usageWithinBudget(cumulative, plan.Policy.Budget) {
		return ErrReceiptInvalid
	}
	claim := operationReceiptClaim(receipt)
	if receipt.Digest == "" || receipt.Digest != receiptIntegrityDigest(claim) {
		return ErrReceiptInvalid
	}
	return verifyReceiptEvidence(ctx, authority, claim, receipt.Evidence)
}

func verifyDeletionReceipt(ctx context.Context, authority ReceiptAuthority, plan Plan, stepReceipts []OperationReceipt, receipt DeletionReceipt) error {
	boundary := firstAfterDeleteStep(plan)
	if boundary < 0 || len(stepReceipts) < boundary {
		return ErrDeletionFence
	}
	expected := deletionReceiptTemplate(plan, stepReceipts[:boundary])
	claim := deletionReceiptClaim(receipt)
	if claim != deletionReceiptClaim(expected) {
		return ErrDeletionFence
	}
	if err := verifyReceiptEvidence(ctx, authority, claim, receipt.Evidence); err != nil {
		if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) || errors.Is(err, ErrReceiptAuthority) {
			return err
		}
		return errors.Join(ErrDeletionFence, ErrReceiptInvalid)
	}
	return nil
}

func sourceReceiptTemplate(plan Plan) (SourceReceipt, error) {
	if !validReceiptPlan(plan) {
		return SourceReceipt{}, ErrPlanStale
	}
	receipt := SourceReceipt{
		SchemaVersion: SourceReceiptSchemaVersion, PlanDigest: plan.Digest,
		RegistryDigest: plan.RegistryDigest, PlanKind: plan.Kind, SourceDigest: plan.Source.Digest,
		PolicyArtifact: receiptArtifact(plan.Policy.Artifact),
	}
	receipt.Digest = receiptIntegrityDigest(sourceReceiptClaim(receipt))
	return receipt, nil
}

func deletionReceiptTemplate(plan Plan, predecessors []OperationReceipt) DeletionReceipt {
	receipt := DeletionReceipt{
		SchemaVersion: DeletionReceiptSchemaVersion, PlanDigest: plan.Digest,
		RegistryDigest: plan.RegistryDigest, PlanKind: plan.Kind, SourceDigest: plan.Source.Digest,
		PolicyArtifact:    receiptArtifact(plan.Policy.Artifact),
		PredecessorDigest: predecessorReceiptDigest(predecessors),
	}
	receipt.Digest = receiptIntegrityDigest(deletionReceiptClaim(receipt))
	return receipt
}

func sourceReceiptClaim(receipt SourceReceipt) ReceiptClaim {
	return ReceiptClaim{
		SchemaVersion: ReceiptClaimSchemaVersion, ReceiptSchema: receipt.SchemaVersion, Kind: ReceiptKindSourceAdmission,
		PlanDigest: receipt.PlanDigest, RegistryDigest: receipt.RegistryDigest, PlanKind: receipt.PlanKind,
		SourceDigest: receipt.SourceDigest, Artifact: receiptArtifact(receipt.PolicyArtifact),
		Outcome: receiptOutcomeSourceCommitted, Digest: receipt.Digest,
	}
}

func deletionReceiptClaim(receipt DeletionReceipt) ReceiptClaim {
	return ReceiptClaim{
		SchemaVersion: ReceiptClaimSchemaVersion, ReceiptSchema: receipt.SchemaVersion, Kind: ReceiptKindDeletionComplete,
		PlanDigest: receipt.PlanDigest, RegistryDigest: receipt.RegistryDigest, PlanKind: receipt.PlanKind,
		SourceDigest: receipt.SourceDigest, Artifact: receiptArtifact(receipt.PolicyArtifact),
		Outcome: receiptOutcomeDeleted, PredecessorDigest: receipt.PredecessorDigest, Digest: receipt.Digest,
	}
}

func operationReceiptClaim(receipt OperationReceipt) ReceiptClaim {
	return ReceiptClaim{
		SchemaVersion: ReceiptClaimSchemaVersion, ReceiptSchema: receipt.SchemaVersion, Kind: ReceiptKindOperation,
		PlanDigest: receipt.PlanDigest, RegistryDigest: receipt.RegistryDigest, PlanKind: receipt.PlanKind,
		SourceDigest: receipt.SourceDigest, Artifact: receiptArtifact(receipt.Artifact),
		OperationKey: receipt.OperationKey, StepID: receipt.StepID, Stage: receipt.Stage,
		Attempt: receipt.Attempt, Outcome: receipt.Outcome, PredecessorDigest: receipt.PredecessorDigest,
		DeltaUsage: receipt.DeltaUsage, CumulativeUsage: receipt.CumulativeUsage,
		OutputDigest: receipt.OutputDigest, Digest: receipt.Digest,
	}
}

func recordReceiptEvidence(ctx context.Context, authority ReceiptAuthority, claim ReceiptClaim) (ReceiptEvidence, error) {
	if ctx == nil || authority == nil {
		return ReceiptEvidence{}, ErrReceiptAuthority
	}
	if claim.Kind != ReceiptKindSourceAdmission && claim.Kind != ReceiptKindDeletionComplete {
		return ReceiptEvidence{}, ErrReceiptInvalid
	}
	if err := ctx.Err(); err != nil {
		return ReceiptEvidence{}, err
	}
	evidence, err := authority.RecordMediaReceipt(ctx, HostReceiptClaim{claim: claim})
	if err != nil {
		if ctx.Err() != nil {
			return ReceiptEvidence{}, ctx.Err()
		}
		return ReceiptEvidence{}, ErrReceiptInvalid
	}
	if err := verifyReceiptEvidence(ctx, authority, claim, evidence); err != nil {
		return ReceiptEvidence{}, err
	}
	return evidence, nil
}

func verifyReceiptEvidence(ctx context.Context, authority ReceiptAuthority, claim ReceiptClaim, evidence ReceiptEvidence) error {
	if ctx == nil || authority == nil {
		return ErrReceiptAuthority
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	if !validReceiptEvidence(evidence) || claim.Digest == "" || claim.Digest != receiptIntegrityDigest(claim) {
		return ErrReceiptInvalid
	}
	if err := authority.VerifyMediaReceipt(ctx, claim, evidence); err != nil {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		return ErrReceiptInvalid
	}
	return nil
}

func validReceiptEvidence(evidence ReceiptEvidence) bool {
	return evidence.ID == strings.TrimSpace(evidence.ID) && evidence.Seal == strings.TrimSpace(evidence.Seal) &&
		evidence.ID != "" && evidence.Seal != "" && validPlainString(evidence.ID, maxStringBytes) && validPlainString(evidence.Seal, maxStringBytes)
}

func receiptIntegrityDigest(claim ReceiptClaim) string {
	claim.Digest = ""
	body, _ := json.Marshal(claim)
	digest := sha256.Sum256(body)
	return hex.EncodeToString(digest[:])
}

func predecessorReceiptDigest(receipts []OperationReceipt) string {
	digest := ""
	for index := range receipts {
		digest = extendPredecessorDigest(digest, receipts[index])
	}
	return digest
}

func extendPredecessorDigest(previous string, receipt OperationReceipt) string {
	type receiptRef struct {
		Digest     string `json:"digest"`
		EvidenceID string `json:"evidenceId"`
		Seal       string `json:"seal"`
	}
	body, _ := json.Marshal(struct {
		SchemaVersion string     `json:"schemaVersion"`
		Previous      string     `json:"previous,omitempty"`
		Receipt       receiptRef `json:"receipt"`
	}{
		SchemaVersion: OperationReceiptSchemaVersion,
		Previous:      previous,
		Receipt: receiptRef{
			Digest: receipt.Digest, EvidenceID: receipt.Evidence.ID, Seal: receipt.Evidence.Seal,
		},
	})
	sum := sha256.Sum256(body)
	return hex.EncodeToString(sum[:])
}

func providerOutputBudgetUsage(output ProviderOutput) OperationBudgetUsage {
	usage := OperationBudgetUsage{Variants: len(output.Variants)}
	for key, value := range output.Metadata {
		usage.MetadataBytes += len(key) + len(value)
	}
	return usage
}

func providerOutputDigest(output ProviderOutput) string {
	body, _ := json.Marshal(cloneProviderOutput(output))
	digest := sha256.Sum256(body)
	return hex.EncodeToString(digest[:])
}

func usageWithinBudget(usage OperationBudgetUsage, budget Budget) bool {
	return usage.MetadataBytes >= 0 && usage.MetadataBytes <= budget.MaxMetadataBytes &&
		usage.Variants >= 0 && usage.Variants <= budget.MaxVariants
}

func planStepIndex(plan Plan, stepID string) int {
	for index := range plan.Steps {
		if plan.Steps[index].ID == stepID {
			return index
		}
	}
	return -1
}

func firstAfterDeleteStep(plan Plan) int {
	for index := range plan.Steps {
		if plan.Steps[index].Processor.Stage == StageAfterDelete {
			return index
		}
	}
	return -1
}

func validReceiptOutcomeForStep(value string, step PlanStep) bool {
	switch value {
	case TraceSucceeded:
		return true
	case TraceFallback:
		return step.Processor.FailureMode == FailureFallbackOriginal
	case TraceSkipped:
		return step.Processor.FailureMode == FailureSkip
	default:
		return false
	}
}

func validReceiptPlan(plan Plan) bool {
	return plan.SchemaVersion == SchemaVersion && plan.Digest != "" && plan.Digest == computePlanDigest(plan) &&
		plan.Source == plan.OriginalFallback && digestPattern.MatchString(plan.RegistryDigest) && digestPattern.MatchString(plan.Source.Digest)
}

func receiptArtifact(artifact Artifact) Artifact {
	artifact.coreSeal = [32]byte{}
	return artifact
}

func cloneOperationPrerequisites(value OperationPrerequisites) OperationPrerequisites {
	value.Steps = cloneOperationReceipts(value.Steps)
	if value.Deletion != nil {
		deletion := *value.Deletion
		value.Deletion = &deletion
	}
	return value
}

func cloneOperationReceipts(values []OperationReceipt) []OperationReceipt {
	return append([]OperationReceipt(nil), values...)
}

func operationReceiptBindings(target PlanStep, prerequisites OperationPrerequisites) []ReceiptBinding {
	bindings := make([]ReceiptBinding, 0, len(prerequisites.Steps)+2)
	bindings = append(bindings, ReceiptBinding{
		Claim: sourceReceiptClaim(prerequisites.Source), Evidence: prerequisites.Source.Evidence,
	})
	for _, receipt := range prerequisites.Steps {
		bindings = append(bindings, ReceiptBinding{Claim: operationReceiptClaim(receipt), Evidence: receipt.Evidence})
	}
	if target.Processor.Stage == StageAfterDelete && prerequisites.Deletion != nil {
		bindings = append(bindings, ReceiptBinding{
			Claim: deletionReceiptClaim(*prerequisites.Deletion), Evidence: prerequisites.Deletion.Evidence,
		})
	}
	return bindings
}
