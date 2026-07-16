package mediaregistry

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"time"
)

// BackgroundOperations returns only background steps that are currently
// runnable after Host-ledger verification. Production River wiring must store a
// protected operation reference instead of raw Plan authority.
func BackgroundOperations(ctx context.Context, authority ReceiptAuthority, plan Plan, prerequisites OperationPrerequisites) ([]BackgroundOperation, error) {
	if plan.SchemaVersion != SchemaVersion || plan.Digest == "" || plan.Digest != computePlanDigest(plan) {
		return nil, ErrPlanStale
	}
	if err := verifySourceReceipt(ctx, authority, plan, prerequisites.Source); err != nil {
		return nil, err
	}
	result := []BackgroundOperation{}
	for _, step := range plan.Steps {
		if step.Processor.Execution != ExecutionBackground {
			continue
		}
		operation, err := OperationForStep(ctx, authority, plan, step.ID, 1, prerequisites)
		if errors.Is(err, ErrPredecessorRequired) {
			continue
		}
		if errors.Is(err, ErrDeletionFence) && prerequisites.Deletion == nil {
			continue
		}
		if err != nil {
			return nil, err
		}
		result = append(result, operation)
	}
	return result, nil
}

func OperationForStep(ctx context.Context, authority ReceiptAuthority, plan Plan, stepID string, attempt int, prerequisites OperationPrerequisites) (BackgroundOperation, error) {
	if plan.SchemaVersion != SchemaVersion || plan.Digest == "" || plan.Digest != computePlanDigest(plan) || attempt < 1 {
		return BackgroundOperation{}, ErrPlanStale
	}
	step, found := findPlanStep(plan, stepID)
	if !found {
		return BackgroundOperation{}, ErrNotFound
	}
	maxAttempts := effectiveMaxAttempts(step.Processor)
	if attempt > maxAttempts {
		return BackgroundOperation{}, ErrInvalid
	}
	if _, err := validateOperationPrerequisites(ctx, authority, plan, step, prerequisites); err != nil {
		return BackgroundOperation{}, err
	}
	operation := BackgroundOperation{
		SchemaVersion: SchemaVersion, StepID: step.ID, Attempt: attempt,
		Plan: clonePlan(plan), Prerequisites: cloneOperationPrerequisites(prerequisites),
	}
	operation.Key = operationKey(operation.Plan, step)
	return cloneOperation(operation), nil
}

func NextAttempt(ctx context.Context, authority ReceiptAuthority, operation BackgroundOperation) (BackgroundOperation, error) {
	return OperationForStep(ctx, authority, operation.Plan, operation.StepID, operation.Attempt+1, operation.Prerequisites)
}

func operationKey(plan Plan, step PlanStep) string {
	value := strings.Join([]string{SchemaVersion, plan.Digest, plan.Kind, plan.Purpose, plan.Source.ID, plan.Source.Digest,
		plan.Actor.ID, plan.Actor.PermissionFingerprint, step.ID, step.Processor.ContractVersion,
		step.Processor.Artifact.ExtensionID, step.Processor.Artifact.PackageDigest, step.Processor.Artifact.RuntimeInstanceID}, "\x00")
	sum := sha256.Sum256([]byte(value))
	return hex.EncodeToString(sum[:])
}

func findPlanStep(plan Plan, stepID string) (PlanStep, bool) {
	for _, step := range plan.Steps {
		if step.ID == stepID {
			return clonePlanStep(step), true
		}
	}
	return PlanStep{}, false
}

func cloneOperation(value BackgroundOperation) BackgroundOperation {
	value.Plan = clonePlan(value.Plan)
	value.Prerequisites = cloneOperationPrerequisites(value.Prerequisites)
	return value
}

type ProviderError struct {
	Class string
	Code  string
	Cause error
}

func NewProviderError(class, code string, cause error) error {
	class = strings.ToLower(strings.TrimSpace(class))
	code = strings.ToLower(strings.TrimSpace(code))
	if !validRetryClass(class) || !idPattern.MatchString(code) {
		return ErrInvalid
	}
	return &ProviderError{Class: class, Code: code, Cause: cause}
}

func (e *ProviderError) Error() string {
	if e == nil {
		return "media provider failed"
	}
	return fmt.Sprintf("media provider failed: %s", e.Code)
}
func (e *ProviderError) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.Cause
}

func ClassifyRetry(processor ProcessorContribution, err error, attempt int) RetryDecision {
	if errors.Is(err, ErrOperationBusy) {
		return retrySameAttempt(processor, attempt)
	}
	class := RetryPermanent
	var provider *ProviderError
	switch {
	case err == nil:
		class = RetryNone
	case errors.As(err, &provider) && validRetryClass(provider.Class):
		class = provider.Class
	case errors.Is(err, ErrRuntimeQuarantined):
		class = RetryPermanent
	case errors.Is(err, ErrRuntimeUnavailable), errors.Is(err, ErrExecutionTimeout), errors.Is(err, context.DeadlineExceeded):
		class = RetryTransient
	case errors.Is(err, context.Canceled), errors.Is(err, ErrMediaRejected), errors.Is(err, ErrOutputRejected), errors.Is(err, ErrPermissionDenied), errors.Is(err, ErrPlanStale), errors.Is(err, ErrDeletionFence), errors.Is(err, ErrBudgetExceeded), errors.Is(err, ErrPredecessorRequired), errors.Is(err, ErrReceiptInvalid), errors.Is(err, ErrReceiptAuthority):
		class = RetryPermanent
	}
	decision := RetryDecision{Class: class}
	if class == RetryNone || class == RetryPermanent || attempt < 1 {
		return decision
	}
	maxAttempts := effectiveMaxAttempts(processor)
	if attempt >= maxAttempts {
		return decision
	}
	base, maxDelay := processor.Retry.BaseDelaySeconds, processor.Retry.MaxDelaySeconds
	if base <= 0 || maxDelay <= 0 {
		return decision
	}
	delay := int64(base)
	for current := 1; current < attempt && delay < int64(maxDelay); current++ {
		delay *= 2
		if delay > int64(maxDelay) {
			delay = int64(maxDelay)
		}
	}
	decision.Retry = true
	decision.NextAttempt = attempt + 1
	decision.Delay = time.Duration(delay) * time.Second
	return decision
}

func retrySameAttempt(processor ProcessorContribution, attempt int) RetryDecision {
	decision := RetryDecision{Class: RetryTransient}
	if attempt < 1 {
		return decision
	}
	delay := processor.Retry.BaseDelaySeconds
	if delay <= 0 {
		delay = 1
	}
	if processor.Retry.MaxDelaySeconds > 0 && delay > processor.Retry.MaxDelaySeconds {
		delay = processor.Retry.MaxDelaySeconds
	}
	decision.Retry = true
	decision.NextAttempt = attempt
	decision.Delay = time.Duration(delay) * time.Second
	return decision
}

func effectiveMaxAttempts(processor ProcessorContribution) int {
	if processor.Retry.MaxAttempts > 0 {
		return processor.Retry.MaxAttempts
	}
	return 1
}
func validRetryClass(value string) bool {
	switch value {
	case RetryTransient, RetryRateLimited, RetryCrash, RetryPermanent:
		return true
	default:
		return false
	}
}
