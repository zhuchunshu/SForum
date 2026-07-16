package mediaregistry

import (
	"context"
	"errors"
)

const OperationClaimSchemaVersion = "sforum.media-operation-claim@1"

// OperationClaim identifies one exact logical step. Attempt is deliberately
// outside the stable operation key: a terminal completion suppresses every
// redelivery, while an aborted non-terminal attempt may be retried.
type OperationClaim struct {
	SchemaVersion     string   `json:"schemaVersion"`
	OperationKey      string   `json:"operationKey"`
	PlanDigest        string   `json:"planDigest"`
	RegistryDigest    string   `json:"registryDigest"`
	PlanKind          string   `json:"planKind"`
	SourceDigest      string   `json:"sourceDigest"`
	StepID            string   `json:"stepId"`
	Stage             string   `json:"stage"`
	Artifact          Artifact `json:"artifact"`
	PredecessorDigest string   `json:"predecessorDigest,omitempty"`
	Attempt           int      `json:"attempt"`
}

// OperationLease is an exclusive Host-owned claim for OperationKey. Context
// is an independent ownership-loss signal and must remain valid after Acquire
// returns until Release or Host revocation. An authority must not reassign a
// revoked/expired live claim until the old callback is fenced or stable-key
// replay safety is guaranteed. Release aborts only a non-terminal claim; a
// committed terminal completion remains replayable.
type OperationLease interface {
	Context() context.Context
	Release()
}

// OperationCompletion is stored atomically with its durable receipt evidence.
// Persisting the normalized output lets duplicate delivery return the exact
// result without invoking a provider again.
type OperationCompletion struct {
	Receipt          OperationReceipt `json:"receipt"`
	Output           ProviderOutput   `json:"output"`
	FallbackOriginal bool             `json:"fallbackOriginal,omitempty"`
	Skipped          bool             `json:"skipped,omitempty"`
}

// OperationAcquisition has exactly one branch: Lease for the sole executor or
// Replay for an already terminal operation. A live claim owned elsewhere must
// return ErrOperationBusy instead of either branch.
type OperationAcquisition struct {
	Lease  OperationLease
	Replay *OperationCompletion
}

func operationClaim(operation BackgroundOperation, step PlanStep) OperationClaim {
	return OperationClaim{
		SchemaVersion: OperationClaimSchemaVersion,
		OperationKey:  operation.Key, PlanDigest: operation.Plan.Digest,
		RegistryDigest: operation.Plan.RegistryDigest, PlanKind: operation.Plan.Kind,
		SourceDigest: operation.Plan.Source.Digest, StepID: step.ID,
		Stage: step.Processor.Stage, Artifact: receiptArtifact(step.Processor.Artifact),
		PredecessorDigest: predecessorReceiptDigest(operation.Prerequisites.Steps),
		Attempt:           operation.Attempt,
	}
}

func sameOperationTarget(left, right OperationClaim) bool {
	left.Attempt = 0
	right.Attempt = 0
	return left == right
}

func operationClaimFromReceipt(receipt OperationReceipt) OperationClaim {
	return OperationClaim{
		SchemaVersion: OperationClaimSchemaVersion,
		OperationKey:  receipt.OperationKey, PlanDigest: receipt.PlanDigest,
		RegistryDigest: receipt.RegistryDigest, PlanKind: receipt.PlanKind,
		SourceDigest: receipt.SourceDigest, StepID: receipt.StepID,
		Stage: receipt.Stage, Artifact: receiptArtifact(receipt.Artifact),
		PredecessorDigest: receipt.PredecessorDigest, Attempt: receipt.Attempt,
	}
}

func validOperationCompletionShape(value OperationCompletion) bool {
	if value.Receipt.OutputDigest != providerOutputDigest(value.Output) {
		return false
	}
	switch value.Receipt.Outcome {
	case TraceSucceeded:
		return !value.FallbackOriginal && !value.Skipped
	case TraceFallback:
		return value.FallbackOriginal && !value.Skipped && emptyProviderOutput(value.Output)
	case TraceSkipped:
		return !value.FallbackOriginal && value.Skipped && emptyProviderOutput(value.Output)
	default:
		return false
	}
}

func emptyProviderOutput(value ProviderOutput) bool {
	return value.Decision == "" && value.ReasonCode == "" && len(value.Metadata) == 0 &&
		len(value.Variants) == 0 && value.CDNURL == "" && value.RetainUntil.IsZero()
}

func cloneOperationCompletion(value OperationCompletion) OperationCompletion {
	value.Output = cloneProviderOutput(value.Output)
	return value
}

func acquireOperationClaim(ctx context.Context, authority ReceiptAuthority, claim OperationClaim) (OperationAcquisition, error) {
	if ctx == nil || authority == nil {
		return OperationAcquisition{}, ErrReceiptAuthority
	}
	if err := ctx.Err(); err != nil {
		return OperationAcquisition{}, executionContextError(ctx)
	}
	acquisition, err := authority.AcquireMediaOperation(ctx, claim)
	if err != nil {
		if ctx.Err() != nil {
			return OperationAcquisition{}, executionContextError(ctx)
		}
		if errors.Is(err, ErrOperationBusy) {
			return OperationAcquisition{}, ErrOperationBusy
		}
		return OperationAcquisition{}, ErrReceiptInvalid
	}
	if acquisition.Replay != nil {
		if acquisition.Lease != nil {
			releaseOperationLease(acquisition.Lease)
			return OperationAcquisition{}, ErrReceiptInvalid
		}
		completion := cloneOperationCompletion(*acquisition.Replay)
		acquisition.Replay = &completion
		return acquisition, nil
	}
	if !operationLeaseCurrent(acquisition.Lease) {
		if acquisition.Lease != nil {
			releaseOperationLease(acquisition.Lease)
		}
		if ctx.Err() != nil {
			return OperationAcquisition{}, executionContextError(ctx)
		}
		return OperationAcquisition{}, ErrReceiptInvalid
	}
	return acquisition, nil
}

func safeOperationLeaseContext(lease OperationLease) (ctx context.Context, ok bool) {
	if lease == nil {
		return nil, false
	}
	defer func() {
		if recover() != nil {
			ctx, ok = nil, false
		}
	}()
	ctx = lease.Context()
	return ctx, ctx != nil
}

func operationLeaseCurrent(lease OperationLease) bool {
	ctx, ok := safeOperationLeaseContext(lease)
	return ok && ctx.Err() == nil
}

func releaseOperationLease(lease OperationLease) (panicked bool) {
	if lease == nil {
		return false
	}
	defer func() { panicked = recover() != nil }()
	lease.Release()
	return false
}

func commitOperationCompletion(
	ctx context.Context,
	authority ReceiptAuthority,
	lease OperationLease,
	prerequisites ReceiptLease,
	operation BackgroundOperation,
	step PlanStep,
	output ProviderOutput,
	outcome string,
	fallback, skipped bool,
) (OperationCompletion, error) {
	receipt, err := operationReceiptTemplate(ctx, authority, operation, step, output, outcome)
	if err != nil {
		return OperationCompletion{}, err
	}
	completion := OperationCompletion{
		Receipt: receipt, Output: cloneProviderOutput(output),
		FallbackOriginal: fallback, Skipped: skipped,
	}
	if !operationLeaseCurrent(lease) || !receiptLeaseCurrent(prerequisites) || !validOperationCompletionShape(completion) {
		return OperationCompletion{}, ErrReceiptInvalid
	}
	evidence, err := authority.CommitMediaOperation(ctx, lease, prerequisites, cloneOperationCompletion(completion))
	if err != nil {
		if ctx.Err() != nil {
			return OperationCompletion{}, executionContextError(ctx)
		}
		return OperationCompletion{}, ErrReceiptInvalid
	}
	completion.Receipt.Evidence = evidence
	// Commit is the terminal linearization point. The Host authority has already
	// atomically validated both leases and the exact claim; no cancellable I/O may
	// run after it and turn a committed terminal into an apparent failure.
	return completion, nil
}

func replayOperationCompletion(
	ctx context.Context,
	authority ReceiptAuthority,
	operation BackgroundOperation,
	step PlanStep,
	usage OperationBudgetUsage,
	remaining Budget,
	completion OperationCompletion,
) (ExecutionResult, error) {
	result := ExecutionResult{OperationKey: operation.Key, StepID: operation.StepID, Replayed: true}
	if !validOperationCompletionShape(completion) ||
		!sameOperationTarget(operationClaim(operation, step), operationClaimFromReceipt(completion.Receipt)) {
		return result, ErrReceiptInvalid
	}
	if err := verifyOperationReceipt(ctx, authority, operation.Plan, step,
		predecessorReceiptDigest(operation.Prerequisites.Steps), usage, completion.Receipt); err != nil {
		return result, err
	}
	output := cloneProviderOutput(completion.Output)
	if completion.Receipt.Outcome == TraceSucceeded {
		normalized, err := validateReplayedProviderOutput(output, operation.Plan, step, remaining)
		if err != nil || providerOutputDigest(normalized) != completion.Receipt.OutputDigest ||
			providerOutputBudgetUsage(normalized) != completion.Receipt.DeltaUsage {
			return result, ErrReceiptInvalid
		}
		output = normalized
	} else if completion.Receipt.DeltaUsage != (OperationBudgetUsage{}) {
		return result, ErrReceiptInvalid
	}
	result.Output = output
	result.Receipt = completion.Receipt
	result.FallbackOriginal = completion.FallbackOriginal
	result.Skipped = completion.Skipped
	result.Retry = RetryDecision{Class: RetryNone}
	return result, nil
}
