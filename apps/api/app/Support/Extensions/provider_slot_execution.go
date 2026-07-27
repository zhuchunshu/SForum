package extensionsruntime

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	extensions "github.com/zhuchunshu/sforum/apps/api/app/Models/Extensions"
)

type VersionedProviderInvocation struct {
	Caller          ProviderSlotCaller
	SlotID          string
	ContractVersion string
	Operation       string
	InputSchema     string
	Input           map[string]any
	Revalidate      HookDocumentRevalidator
}

type VersionedProviderInvocationResult struct {
	ProviderID        string
	ExtensionID       string
	RuntimeInstanceID string
	ResponseSchema    string
	Output            map[string]any
	Attempts          int
}

func (m *RuntimeInvoker) InvokeVersionedProvider(
	ctx context.Context,
	input VersionedProviderInvocation,
) (VersionedProviderInvocationResult, error) {
	if m == nil || ctx == nil || m.hooks == nil || strings.TrimSpace(input.Operation) != VersionedProviderOperationInvoke || input.Revalidate == nil {
		return VersionedProviderInvocationResult{}, ErrProviderSlotInvalid
	}
	var resolution ProviderSlotResolution
	var err error
	if selections := m.ProviderSlotSelections(); selections != nil {
		resolution, _, err = selections.Resolve(ctx, input.Caller, input.SlotID, input.ContractVersion)
	} else {
		resolution, err = m.hooks.providerSlots.Discover(input.Caller, input.SlotID, input.ContractVersion)
	}
	if err != nil {
		return VersionedProviderInvocationResult{}, err
	}
	if strings.TrimSpace(input.InputSchema) != resolution.Contract.RequestSchema {
		return VersionedProviderInvocationResult{}, ErrProviderSlotInputInvalid
	}
	invoker, ok := m.starter.(interface {
		InvokeVersionedProvider(context.Context, extensions.Extension, VersionedProviderRequest) (VersionedProviderResponse, error)
	})
	if !ok {
		return VersionedProviderInvocationResult{}, ErrProviderSlotNoProvider
	}
	requestPayload, err := cloneHookDocument(input.Input)
	if err != nil {
		return VersionedProviderInvocationResult{}, fmt.Errorf("clone provider input: %w", err)
	}
	var failures []error
	for index, candidate := range resolution.Candidates {
		extension, available := m.runningExtension(candidate.Artifact.ExtensionID)
		if !available || extension.Version != candidate.Artifact.ExtensionVersion || extension.PackageDigest != candidate.Artifact.PackageDigest {
			failures = append(failures, fmt.Errorf("%s: exact runtime unavailable", candidate.ID))
			if resolution.Contract.Fallback == "closed" {
				break
			}
			continue
		}
		instance, admission, acquireErr := m.AcquireActiveRuntimeCall(ctx, extension.ID, RuntimeCallProvider)
		if acquireErr != nil || instance.Identity.InstanceID != candidate.Artifact.RuntimeInstanceID {
			failures = append(failures, fmt.Errorf("%s: exact runtime admission denied", candidate.ID))
			if admission != nil {
				admission.Release()
			}
			if resolution.Contract.Fallback == "closed" {
				break
			}
			continue
		}
		callCtx := admission.Context
		callCtx, cancel := context.WithTimeout(callCtx, time.Duration(resolution.Contract.TimeoutMS)*time.Millisecond)
		release, rejected := m.resilience.tryEnter(callCtx, extension.ID)
		if rejected != "" {
			cancel()
			admission.Release()
			failures = append(failures, fmt.Errorf("%s: %s", candidate.ID, rejected))
			if resolution.Contract.Fallback == "closed" {
				break
			}
			continue
		}
		failProviderCall := func(callErr error) bool {
			cancel()
			release(false, providerFailureReason(callErr))
			admission.Release()
			failures = append(failures, fmt.Errorf("%s: %w", candidate.ID, callErr))
			return resolution.Contract.Fallback == "closed" || ctx.Err() != nil
		}
		candidateInput, cloneErr := cloneHookDocument(requestPayload)
		if cloneErr != nil {
			if cause := context.Cause(callCtx); cause != nil {
				if failProviderCall(cause) {
					break
				}
				continue
			}
			cancel()
			release(false, "extension.provider_input_invalid")
			admission.Release()
			return VersionedProviderInvocationResult{}, cloneErr
		}
		if validateErr := input.Revalidate(callCtx, resolution.Contract.RequestSchema, candidateInput); validateErr != nil {
			if cause := context.Cause(callCtx); cause != nil {
				if failProviderCall(cause) {
					break
				}
				continue
			}
			cancel()
			release(false, "extension.provider_input_invalid")
			admission.Release()
			return VersionedProviderInvocationResult{}, fmt.Errorf("%w: %v", ErrProviderSlotInputInvalid, validateErr)
		}
		if validateErr := m.hooks.providerSlots.ValidateDocument(resolution.Contract.ID, resolution.Contract.RequestSchema, candidateInput); validateErr != nil {
			if cause := context.Cause(callCtx); cause != nil {
				if failProviderCall(cause) {
					break
				}
				continue
			}
			cancel()
			release(false, "extension.provider_input_invalid")
			admission.Release()
			return VersionedProviderInvocationResult{}, fmt.Errorf("%w: %v", ErrProviderSlotInputInvalid, validateErr)
		}
		if callErr := context.Cause(callCtx); callErr != nil {
			if failProviderCall(callErr) {
				break
			}
			continue
		}
		response, callErr := invoker.InvokeVersionedProvider(callCtx, extension, VersionedProviderRequest{
			DeclarationID: candidate.ID, Slot: resolution.Contract.Slot,
			ContractVersion: resolution.Contract.ContractVersion, Operation: input.Operation,
			RequestSchema: resolution.Contract.RequestSchema, ResponseSchema: resolution.Contract.ResponseSchema,
			Timeout: time.Duration(resolution.Contract.TimeoutMS) * time.Millisecond, Input: candidateInput,
		})
		// Go cannot safely preempt an arbitrary in-process invoker. Production gRPC
		// returns on callCtx, while a non-cooperative adapter must retain its exact
		// admission and resilience slot until it really exits. Only then may a
		// fallback start; a late success still resolves to the Host timeout cause.
		if cause := context.Cause(callCtx); cause != nil {
			callErr = cause
		}
		if callErr != nil {
			if failProviderCall(callErr) {
				break
			}
			continue
		}
		output, cloneErr := cloneHookDocument(response.Output)
		if cloneErr != nil {
			if cause := context.Cause(callCtx); cause != nil {
				if failProviderCall(cause) {
					break
				}
				continue
			}
			cancel()
			release(false, "extension.provider_output_invalid")
			admission.Release()
			failures = append(failures, fmt.Errorf("%w: %s: clone provider output: %v", ErrProviderSlotOutputInvalid, candidate.ID, cloneErr))
			if resolution.Contract.Fallback == "closed" {
				break
			}
			continue
		}
		if validateErr := input.Revalidate(callCtx, resolution.Contract.ResponseSchema, output); validateErr != nil {
			if cause := context.Cause(callCtx); cause != nil {
				if failProviderCall(cause) {
					break
				}
				continue
			}
			cancel()
			release(false, "extension.provider_output_invalid")
			admission.Release()
			failures = append(failures, fmt.Errorf("%w: %s: provider output revalidation: %v", ErrProviderSlotOutputInvalid, candidate.ID, validateErr))
			if resolution.Contract.Fallback == "closed" {
				break
			}
			continue
		}
		if validateErr := m.hooks.providerSlots.ValidateDocument(resolution.Contract.ID, resolution.Contract.ResponseSchema, output); validateErr != nil {
			if cause := context.Cause(callCtx); cause != nil {
				if failProviderCall(cause) {
					break
				}
				continue
			}
			cancel()
			release(false, "extension.provider_output_invalid")
			admission.Release()
			failures = append(failures, fmt.Errorf("%w: %s: provider exact output schema: %v", ErrProviderSlotOutputInvalid, candidate.ID, validateErr))
			if resolution.Contract.Fallback == "closed" {
				break
			}
			continue
		}
		if callErr := context.Cause(callCtx); callErr != nil {
			if failProviderCall(callErr) {
				break
			}
			continue
		}
		cancel()
		release(true, "")
		admission.Release()
		return VersionedProviderInvocationResult{
			ProviderID: candidate.ID, ExtensionID: candidate.Artifact.ExtensionID,
			RuntimeInstanceID: candidate.Artifact.RuntimeInstanceID,
			ResponseSchema:    resolution.Contract.ResponseSchema,
			Output:            output, Attempts: index + 1,
		}, nil
	}
	return VersionedProviderInvocationResult{Attempts: len(failures)}, errors.Join(append([]error{ErrProviderSlotNoProvider}, failures...)...)
}

func providerFailureReason(err error) string {
	if err == nil {
		return ""
	}
	if errors.Is(err, context.DeadlineExceeded) || errors.Is(err, context.Canceled) {
		return "extension.provider_timeout"
	}
	return "extension.provider_failed"
}

// Compatibility facade: runtime logic is owned by focused collaborators.

func (m *Manager) InvokeVersionedProvider(
	ctx context.Context,
	input VersionedProviderInvocation,
) (VersionedProviderInvocationResult, error) {
	return m.invoker.InvokeVersionedProvider(ctx, input)
}
