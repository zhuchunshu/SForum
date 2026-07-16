package contentregistry

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"
)

type ExecutorOption interface {
	applyContentExecutor(*executorOptions)
}

type executorOptionFunc func(*executorOptions)

func (f executorOptionFunc) applyContentExecutor(options *executorOptions) { f(options) }

type executorOptions struct {
	trace ContentTraceSink
}

func WithContentTraceSink(sink ContentTraceSink) ExecutorOption {
	return executorOptionFunc(func(options *executorOptions) { options.trace = sink })
}

// Executor consumes immutable Registry declarations through Host-authored
// bindings. It contains no lifecycle publication or Protocol transport logic;
// NewExecutor alone therefore does not attest callback provenance. Do not wire
// it to production until Manager/Protocol dispatches the exact handler carried
// by contentAdmissionRequest through the acquired runtime lease.
type Executor struct {
	registry      *Registry
	admission     RuntimeAdmission
	schemas       SchemaValidator
	limits        ExecutionLimits
	bindings      []ExecutionBinding
	bindingDigest string
	trace         ContentTraceSink
	traceDispatch *contentTraceDispatcher
	runtimeSlots  chan struct{}
	hostSlots     chan struct{}
	quarantine    runtimeQuarantine
}

func NewExecutor(
	registry *Registry,
	bindings []ExecutionBinding,
	admission RuntimeAdmission,
	schemas SchemaValidator,
	limits ExecutionLimits,
	options ...ExecutorOption,
) (*Executor, error) {
	if registry == nil || admission == nil || schemas == nil {
		return nil, ErrExecutionInvalid
	}
	normalizedLimits, err := normalizeExecutionLimits(limits)
	if err != nil {
		return nil, err
	}
	normalizedBindings, digest, err := normalizeExecutionBindings(bindings, normalizedLimits)
	if err != nil {
		return nil, err
	}
	configuration := executorOptions{}
	for _, option := range options {
		if option != nil {
			option.applyContentExecutor(&configuration)
		}
	}
	executor := &Executor{
		registry: registry, admission: admission, schemas: schemas, limits: normalizedLimits,
		bindings: normalizedBindings, bindingDigest: digest, trace: configuration.trace,
		runtimeSlots: make(chan struct{}, normalizedLimits.MaxConcurrentCalls),
		hostSlots:    make(chan struct{}, normalizedLimits.MaxConcurrentCalls),
	}
	if configuration.trace != nil {
		if _, synchronous := configuration.trace.(*ContentTraceRing); !synchronous {
			executor.traceDispatch = newContentTraceDispatcher(configuration.trace)
		}
	}
	return executor, nil
}

// Close stops the optional bounded trace worker. It is idempotent. A custom
// sink that ignores its own return contract may delay Close, but can never
// delay Execute or create one goroutine per event.
func (e *Executor) Close() {
	if e != nil && e.traceDispatch != nil {
		e.traceDispatch.Close()
	}
}

func (e *Executor) Execute(ctx context.Context, raw ExecutionRequest) (ExecutionResult, error) {
	if e == nil || ctx == nil {
		return ExecutionResult{}, ErrExecutionInvalid
	}
	request, err := normalizeExecutionRequest(raw, e.limits)
	if err != nil {
		return ExecutionResult{}, err
	}
	executionCtx, cancelExecution := context.WithTimeout(ctx, e.limits.CallTimeout)
	defer cancelExecution()
	plan, err := e.buildPlan(request)
	if err != nil {
		return ExecutionResult{}, err
	}

	var document EditorDocument
	var serialized SerializedContent
	var rendered RenderSegments
	var used []plannedBinding
	fallbackUsed := false
	sourcePreserved := false
	if plan.hidden {
		step := *plan.terminal
		if err := e.checkNonCallStep(executionCtx, plan, step, request, OperationHide); err != nil {
			return ExecutionResult{}, err
		}
		rendered = hiddenRender(plan.target)
		used = append(used, step)
	} else {
		started := time.Now()
		if err := e.authorize(
			executionCtx, request.Permission,
			e.permissionClaim(plan, plan.base, request, OperationSource),
		); err != nil {
			e.recordTrace(plan, plan.base, OperationSource, traceOutcome(err), time.Since(started))
			return ExecutionResult{}, err
		}
		var value any
		document, value, err = normalizeEditorDocument(request.Document, plan.target, e.limits)
		if err != nil {
			return ExecutionResult{}, err
		}
		if err := e.validateSchema(executionCtx, plan.target, SchemaPhaseInput, value); err != nil {
			return ExecutionResult{}, err
		}
		terminal := plan.base
		if plan.terminal != nil {
			terminal = *plan.terminal
		}
		document, serialized, rendered, err = e.executeTerminal(executionCtx, plan, terminal, request, document)
		if err != nil {
			document, serialized, rendered, used, fallbackUsed, sourcePreserved, err = e.terminalFallback(
				executionCtx, plan, terminal, request, request.Document, err,
			)
			if err != nil {
				return ExecutionResult{}, err
			}
		} else {
			used = append(used, terminal)
		}

		before, beforeUsed, beforeFallback, err := e.executeSideRenderers(
			executionCtx, plan, plan.before, request, document, serialized, RenderSegments{},
		)
		if err != nil {
			return ExecutionResult{}, err
		}
		used = append(used, beforeUsed...)
		fallbackUsed = fallbackUsed || beforeFallback
		after, afterUsed, afterFallback, err := e.executeSideRenderers(
			executionCtx, plan, plan.after, request, document, serialized, RenderSegments{},
		)
		if err != nil {
			return ExecutionResult{}, err
		}
		used = append(used, afterUsed...)
		fallbackUsed = fallbackUsed || afterFallback

		wrapped := rendered
		for _, step := range plan.wrap {
			next, callErr := e.callRenderer(executionCtx, plan, step, request, document, serialized, wrapped)
			if callErr != nil {
				if !modifierCanFallback(step.binding.Fallback) || !declaredFallbackAllowed(callErr) {
					return ExecutionResult{}, callErr
				}
				e.recordFallback(plan, step, OperationRenderer)
				fallbackUsed = true
				continue
			}
			wrapped = next
			used = append(used, step)
		}
		rendered, err = mergeRenderSegments(plan.target, e.limits, before, wrapped, after)
		if err != nil {
			return ExecutionResult{}, err
		}

		for _, step := range plan.filters {
			next, callErr := e.callFilter(executionCtx, plan, step, request, document, serialized, rendered)
			if callErr != nil {
				if !modifierCanFallback(step.binding.Fallback) || !declaredFallbackAllowed(callErr) {
					return ExecutionResult{}, callErr
				}
				e.recordFallback(plan, step, OperationFilter)
				fallbackUsed = true
				continue
			}
			// Every filter result is structurally normalized and sanitized before
			// another filter can observe it. ManifestContent has no separate
			// render-result schema, so the editor schema is not misapplied here.
			rendered = next
			used = append(used, step)
		}
	}
	rendered, err = finalizeRenderSegmentsForSSR(rendered, plan.target, e.limits)
	if err != nil {
		return ExecutionResult{}, fmt.Errorf("finalize SSR render: %w", err)
	}

	result := ExecutionResult{
		SchemaVersion: ExecutionSchemaVersion, Revision: plan.state.revision, Digest: plan.state.digest,
		Render:       rendered,
		FallbackUsed: fallbackUsed, SourcePreserved: sourcePreserved, Hidden: plan.hidden,
	}
	result.Attribution = executionAttribution(used)
	result.CacheTags, err = e.executionCacheTags(request, used)
	if err != nil {
		return ExecutionResult{}, err
	}
	result.CacheKey = e.executionCacheKey(plan, request, result, serialized)
	if err := preflightExecutionResultJSON(result, e.limits); err != nil {
		return ExecutionResult{}, err
	}
	if encoded, marshalErr := json.Marshal(result); marshalErr != nil || len(encoded) > e.limits.MaxOutputBytes {
		return ExecutionResult{}, ErrExecutionLimit
	}
	if err := e.releaseFence(executionCtx, plan, request, result, used, document, serialized); err != nil {
		return ExecutionResult{}, fmt.Errorf("release content result: %w", err)
	}
	return cloneExecutionResult(result), nil
}

func (e *Executor) executeTerminal(
	ctx context.Context,
	plan executionPlan,
	step plannedBinding,
	request ExecutionRequest,
	document EditorDocument,
) (EditorDocument, SerializedContent, RenderSegments, error) {
	var err error
	if step.binding.Providers.Editor != nil {
		document, err = invokeProvider(e, ctx, plan, step, request, OperationEditor, func(callCtx context.Context) (EditorDocument, error) {
			input := EditorProviderRequest{
				Target: cloneContribution(plan.target), Provider: cloneContribution(step.contribution),
				Action: step.binding.Action, Document: cloneEditorDocument(document),
				ResourceID: request.ResourceID, Locale: request.Locale, Scope: request.Scope,
			}
			return step.binding.Providers.Editor.PrepareEditorDocument(callCtx, input)
		}, func(hostCtx context.Context, candidate EditorDocument) (EditorDocument, error) {
			normalized, value, normalizeErr := normalizeEditorDocument(candidate, plan.target, e.limits)
			if normalizeErr != nil {
				return EditorDocument{}, normalizeErr
			}
			if schemaErr := e.validateSchema(hostCtx, plan.target, SchemaPhaseOutput, value); schemaErr != nil {
				return EditorDocument{}, schemaErr
			}
			return normalized, nil
		})
		if err != nil {
			return EditorDocument{}, SerializedContent{}, RenderSegments{}, err
		}
	}
	if step.binding.Providers.Validator != nil {
		_, err = invokeProvider(e, ctx, plan, step, request, OperationValidator, func(callCtx context.Context) (struct{}, error) {
			input := ValidatorProviderRequest{
				Target: cloneContribution(plan.target), Provider: cloneContribution(step.contribution),
				Action: step.binding.Action, Document: cloneEditorDocument(document),
				ResourceID: request.ResourceID, Locale: request.Locale, Scope: request.Scope,
			}
			return struct{}{}, step.binding.Providers.Validator.ValidateEditorDocument(callCtx, input)
		}, nil)
		if err != nil {
			return EditorDocument{}, SerializedContent{}, RenderSegments{}, err
		}
	}
	value, err := decodeBoundedJSON(document.Value, e.limits)
	if err != nil {
		return EditorDocument{}, SerializedContent{}, RenderSegments{}, err
	}
	if err = e.validateSchema(ctx, plan.target, SchemaPhaseInput, value); err != nil {
		return EditorDocument{}, SerializedContent{}, RenderSegments{}, err
	}

	var serialized SerializedContent
	if step.binding.Providers.Serializer != nil {
		serialized, err = invokeProvider(e, ctx, plan, step, request, OperationSerializer, func(callCtx context.Context) (SerializedContent, error) {
			input := SerializerProviderRequest{
				Target: cloneContribution(plan.target), Provider: cloneContribution(step.contribution),
				Action: step.binding.Action, Document: cloneEditorDocument(document),
				ResourceID: request.ResourceID, Locale: request.Locale, Scope: request.Scope,
			}
			return step.binding.Providers.Serializer.SerializeEditorDocument(callCtx, input)
		}, func(hostCtx context.Context, candidate SerializedContent) (SerializedContent, error) {
			normalized, normalizeErr := e.normalizeSerialized(candidate, document)
			if normalizeErr != nil {
				return SerializedContent{}, normalizeErr
			}
			value, decodeErr := decodeBoundedJSON(normalized.Data, e.limits)
			if decodeErr != nil {
				return SerializedContent{}, decodeErr
			}
			if schemaErr := e.validateSchema(hostCtx, plan.target, SchemaPhaseOutput, value); schemaErr != nil {
				return SerializedContent{}, schemaErr
			}
			return normalized, nil
		})
		if err != nil {
			return EditorDocument{}, SerializedContent{}, RenderSegments{}, err
		}
	} else {
		serialized, err = e.defaultSerialize(document)
		if err != nil {
			return EditorDocument{}, SerializedContent{}, RenderSegments{}, err
		}
	}
	rendered, err := e.callRenderer(ctx, plan, step, request, document, serialized, RenderSegments{})
	if err != nil {
		return EditorDocument{}, SerializedContent{}, RenderSegments{}, err
	}
	return document, serialized, rendered, nil
}

func (e *Executor) terminalFallback(
	ctx context.Context,
	plan executionPlan,
	failed plannedBinding,
	request ExecutionRequest,
	rawDocument EditorDocument,
	cause error,
) (EditorDocument, SerializedContent, RenderSegments, []plannedBinding, bool, bool, error) {
	if !declaredFallbackAllowed(cause) {
		return EditorDocument{}, SerializedContent{}, RenderSegments{}, nil, false, false, cause
	}
	switch failed.binding.Fallback {
	case FallbackBase:
		if failed.binding.Action != ActionReplace {
			return EditorDocument{}, SerializedContent{}, RenderSegments{}, nil, false, false, cause
		}
		e.recordFallback(plan, failed, OperationRenderer)
		document, _, err := normalizeEditorDocument(rawDocument, plan.target, e.limits)
		if err != nil {
			return EditorDocument{}, SerializedContent{}, RenderSegments{}, nil, false, false, err
		}
		document, serialized, rendered, err := e.executeTerminal(ctx, plan, plan.base, request, document)
		if err != nil {
			if plan.base.binding.Fallback != FallbackPreserveSource || !declaredFallbackAllowed(err) {
				return EditorDocument{}, SerializedContent{}, RenderSegments{}, nil, false, false, err
			}
			return e.preserveSourceFallback(ctx, plan, request, rawDocument, plan.base)
		}
		return document, serialized, rendered, []plannedBinding{plan.base}, true, false, nil
	case FallbackPreserveSource:
		e.recordFallback(plan, failed, OperationRenderer)
		return e.preserveSourceFallback(ctx, plan, request, rawDocument, failed)
	default:
		return EditorDocument{}, SerializedContent{}, RenderSegments{}, nil, false, false, cause
	}
}

func (e *Executor) preserveSourceFallback(
	ctx context.Context,
	plan executionPlan,
	request ExecutionRequest,
	rawDocument EditorDocument,
	step plannedBinding,
) (EditorDocument, SerializedContent, RenderSegments, []plannedBinding, bool, bool, error) {
	document, value, err := normalizeEditorDocument(rawDocument, plan.target, e.limits)
	if err != nil {
		return EditorDocument{}, SerializedContent{}, RenderSegments{}, nil, false, false, err
	}
	if err := e.validateSchema(ctx, plan.target, SchemaPhaseInput, value); err != nil {
		return EditorDocument{}, SerializedContent{}, RenderSegments{}, nil, false, false, err
	}
	serialized, err := e.defaultSerialize(document)
	if err != nil {
		return EditorDocument{}, SerializedContent{}, RenderSegments{}, nil, false, false, err
	}
	rendered, err := preservedSourceFallback(plan.target, e.limits)
	if err != nil {
		return EditorDocument{}, SerializedContent{}, RenderSegments{}, nil, false, false, err
	}
	_ = request
	return document, serialized, rendered, nil, true, true, nil
}

func (e *Executor) executeSideRenderers(
	ctx context.Context,
	plan executionPlan,
	steps []plannedBinding,
	request ExecutionRequest,
	document EditorDocument,
	serialized SerializedContent,
	inner RenderSegments,
) (RenderSegments, []plannedBinding, bool, error) {
	merged := RenderSegments{
		SchemaVersion: RenderSegmentsSchemaVersion,
		ContentID:     plan.target.ID, ContractVersion: plan.target.ContractVersion,
		Segments: []RenderSegment{},
	}
	used := make([]plannedBinding, 0, len(steps))
	fallback := false
	for _, step := range steps {
		if ctx.Err() != nil {
			return RenderSegments{}, nil, false, executionContextError(ctx)
		}
		rendered, err := e.callRenderer(ctx, plan, step, request, document, serialized, inner)
		if err != nil {
			if !modifierCanFallback(step.binding.Fallback) || !declaredFallbackAllowed(err) {
				return RenderSegments{}, nil, false, err
			}
			e.recordFallback(plan, step, OperationRenderer)
			fallback = true
			continue
		}
		if err := preflightMergedRenderSegments(plan.target, e.limits, merged, rendered); err != nil {
			return RenderSegments{}, nil, false, err
		}
		merged.Segments = append(merged.Segments, rendered.Segments...)
		used = append(used, step)
	}
	merged, err := normalizeRenderSegments(merged, plan.target, e.limits)
	return merged, used, fallback, err
}

func (e *Executor) callRenderer(
	ctx context.Context,
	plan executionPlan,
	step plannedBinding,
	request ExecutionRequest,
	document EditorDocument,
	serialized SerializedContent,
	inner RenderSegments,
) (RenderSegments, error) {
	result, err := invokeProvider(e, ctx, plan, step, request, OperationRenderer, func(callCtx context.Context) (RenderSegments, error) {
		input := RendererProviderRequest{
			Target: cloneContribution(plan.target), Provider: cloneContribution(step.contribution),
			Action: step.binding.Action, Document: cloneEditorDocument(document),
			Serialized: cloneSerializedContent(serialized), Inner: cloneRenderSegments(inner),
			ResourceID: request.ResourceID, Locale: request.Locale, Scope: request.Scope,
		}
		return step.binding.Providers.Renderer.RenderContent(callCtx, input)
	}, func(_ context.Context, candidate RenderSegments) (RenderSegments, error) {
		return normalizeRenderSegments(candidate, plan.target, e.limits)
	})
	return result, err
}

func (e *Executor) callFilter(
	ctx context.Context,
	plan executionPlan,
	step plannedBinding,
	request ExecutionRequest,
	document EditorDocument,
	serialized SerializedContent,
	rendered RenderSegments,
) (RenderSegments, error) {
	result, err := invokeProvider(e, ctx, plan, step, request, OperationFilter, func(callCtx context.Context) (RenderSegments, error) {
		input := FilterProviderRequest{
			Target: cloneContribution(plan.target), Provider: cloneContribution(step.contribution),
			Action: step.binding.Action, Document: cloneEditorDocument(document),
			Serialized: cloneSerializedContent(serialized), Render: cloneRenderSegments(rendered),
			ResourceID: request.ResourceID, Locale: request.Locale, Scope: request.Scope,
		}
		return step.binding.Providers.Filter.FilterRenderedContent(callCtx, input)
	}, func(_ context.Context, candidate RenderSegments) (RenderSegments, error) {
		return normalizeRenderSegments(candidate, plan.target, e.limits)
	})
	return result, err
}

func (e *Executor) normalizeSerialized(input SerializedContent, document EditorDocument) (SerializedContent, error) {
	if input.SchemaVersion == "" {
		input.SchemaVersion = SerializedSchemaVersion
	}
	input.ContentID = strings.ToLower(strings.TrimSpace(input.ContentID))
	input.ContractVersion = strings.TrimSpace(input.ContractVersion)
	input.StorageVersion = strings.TrimSpace(input.StorageVersion)
	input.MediaType = strings.ToLower(strings.TrimSpace(input.MediaType))
	if input.SchemaVersion != SerializedSchemaVersion || input.ContentID != document.ContentID ||
		input.ContractVersion != document.ContractVersion || input.StorageVersion != document.StorageVersion ||
		input.MediaType != "application/json" && input.MediaType != "application/vnd.sforum.editor+json" ||
		len(input.Data) == 0 || len(input.Data) > e.limits.MaxOutputBytes {
		return SerializedContent{}, ErrContractStale
	}
	if _, err := decodeBoundedJSON(input.Data, e.limits); err != nil {
		return SerializedContent{}, err
	}
	return cloneSerializedContent(input), nil
}

func (e *Executor) defaultSerialize(document EditorDocument) (SerializedContent, error) {
	result := SerializedContent{
		SchemaVersion: SerializedSchemaVersion, ContentID: document.ContentID,
		ContractVersion: document.ContractVersion, StorageVersion: document.StorageVersion,
		MediaType: "application/vnd.sforum.editor+json", Data: append([]byte(nil), document.Value...),
	}
	return e.normalizeSerialized(result, document)
}

func (e *Executor) validateSchema(
	ctx context.Context,
	contribution Contribution,
	phase SchemaValidationPhase,
	value any,
) error {
	if err := ctx.Err(); err != nil {
		return executionContextError(ctx)
	}
	err := invokeHostCallback(e, ctx, func(callbackCtx context.Context) error {
		return validateContentSchemaFailClosed(e.schemas, callbackCtx, SchemaValidationRequest{
			SchemaRef: contribution.Schema, ContentID: contribution.ID,
			ContractVersion: contribution.ContractVersion, Phase: phase, Value: value,
		})
	})
	if err != nil {
		if ctx.Err() != nil {
			return executionContextError(ctx)
		}
		if errors.Is(err, ErrExecutionTimeout) || errors.Is(err, context.Canceled) {
			return err
		}
		if errors.Is(err, ErrSchemaRejected) {
			return err
		}
		return ErrSchemaRejected
	}
	if ctx.Err() != nil {
		return executionContextError(ctx)
	}
	return nil
}

func (e *Executor) authorize(ctx context.Context, input PermissionInput, claim PermissionClaim) error {
	if input.Recheck == nil || input.PolicyFingerprint == "" {
		return ErrExecutionDenied
	}
	if ctx == nil {
		return ErrExecutionInvalid
	}
	if ctx.Err() != nil {
		return executionContextError(ctx)
	}
	err := invokeHostCallback(e, ctx, func(callbackCtx context.Context) error {
		return authorizeContentFailClosed(input.Recheck, callbackCtx, claim)
	})
	if err != nil {
		if ctx.Err() != nil {
			return executionContextError(ctx)
		}
		if errors.Is(err, ErrExecutionTimeout) || errors.Is(err, context.Canceled) {
			return err
		}
		if errors.Is(err, ErrExecutionDenied) {
			return err
		}
		return ErrExecutionDenied
	}
	if ctx.Err() != nil {
		return executionContextError(ctx)
	}
	return nil
}

func validateContentSchemaFailClosed(
	validator SchemaValidator,
	ctx context.Context,
	request SchemaValidationRequest,
) (err error) {
	defer func() {
		if recover() != nil {
			err = ErrSchemaRejected
		}
	}()
	return validator.ValidateContentSchema(ctx, request)
}

func authorizeContentFailClosed(
	recheck PermissionRecheck,
	ctx context.Context,
	claim PermissionClaim,
) (err error) {
	defer func() {
		if recover() != nil {
			err = ErrExecutionDenied
		}
	}()
	return recheck.AuthorizeContent(ctx, claim)
}

func (e *Executor) permissionClaim(
	plan executionPlan,
	step plannedBinding,
	request ExecutionRequest,
	operation string,
) PermissionClaim {
	return PermissionClaim{
		TargetID: plan.target.ID, TargetContractVersion: plan.target.ContractVersion,
		TargetSchema: plan.target.Schema, TargetArtifact: plan.target.Artifact,
		ContentID:       step.contribution.ID,
		ContractVersion: step.contribution.ContractVersion, Schema: step.contribution.Schema,
		Action: step.binding.Action, Operation: operation, Artifact: step.contribution.Artifact,
		ResourceID: request.ResourceID, Locale: request.Locale, Scope: request.Scope,
	}
}

func contentAdmissionRequest(plan executionPlan, step plannedBinding, operation string) AdmissionRequest {
	return AdmissionRequest{
		TargetID: plan.target.ID, TargetContractVersion: plan.target.ContractVersion,
		TargetSchema: plan.target.Schema, TargetArtifact: plan.target.Artifact,
		ContentID: step.contribution.ID, ContractVersion: step.contribution.ContractVersion,
		HandlerReference: step.contribution.Handler, RendererReference: step.contribution.Renderer,
		Action: step.binding.Action, Operation: operation, Artifact: step.contribution.Artifact,
	}
}

func (e *Executor) releaseFence(
	ctx context.Context,
	plan executionPlan,
	request ExecutionRequest,
	result ExecutionResult,
	used []plannedBinding,
	document EditorDocument,
	serialized SerializedContent,
) (returnErr error) {
	if e.registry.load() != plan.state {
		return ErrContractStale
	}
	activeTarget, targetFound := plan.state.content[plan.target.ID]
	if !targetFound || activeTarget.Artifact != plan.target.Artifact ||
		activeTarget.ContractVersion != plan.target.ContractVersion || activeTarget.Schema != plan.target.Schema {
		return ErrContractStale
	}
	if err := e.authorize(
		ctx, request.Permission, e.permissionClaim(plan, plan.base, request, OperationRelease),
	); err != nil {
		return err
	}
	steps := uniquePlannedBindings(used)
	if len(steps) > cap(e.runtimeSlots) {
		return ErrExecutionLimit
	}
	leases := make([]*ownedAdmissionLease, 0, len(steps))
	defer func() {
		for index := len(leases) - 1; index >= 0; index-- {
			if err := e.releaseOwnedAdmission(ctx, leases[index]); err != nil {
				if returnErr == nil {
					returnErr = err
				} else {
					returnErr = errors.Join(returnErr, err)
				}
			}
		}
	}()
	for _, step := range steps {
		active, found := plan.state.content[step.contribution.ID]
		if !found || active.Artifact != step.contribution.Artifact ||
			active.ContractVersion != step.contribution.ContractVersion || active.Schema != step.contribution.Schema {
			return ErrContractStale
		}
		if err := e.authorize(ctx, request.Permission, e.permissionClaim(plan, step, request, OperationRelease)); err != nil {
			return err
		}
		if err := e.runtimeQuarantineError(step.contribution.Artifact); err != nil {
			return err
		}
		lease, err := e.acquireOwnedAdmission(
			ctx, contentAdmissionRequest(plan, step, OperationRelease), step.contribution.Artifact,
		)
		if err != nil {
			return fmt.Errorf("result release: %w", err)
		}
		leases = append(leases, lease)
	}
	if !result.Hidden {
		value, err := decodeBoundedJSON(document.Value, e.limits)
		if err != nil {
			return fmt.Errorf("release document: %w", err)
		}
		if err := e.validateSchema(ctx, plan.target, SchemaPhaseOutput, value); err != nil {
			return err
		}
		if _, err := e.normalizeSerialized(serialized, document); err != nil {
			return fmt.Errorf("release serialized content: %w", err)
		}
	}
	if err := validateSSRRenderSegments(result.Render, plan.target, e.limits); err != nil {
		return fmt.Errorf("release SSR render: %w", err)
	}
	if err := e.authorize(
		ctx, request.Permission, e.permissionClaim(plan, plan.base, request, OperationRelease),
	); err != nil {
		return err
	}
	for _, step := range steps {
		if err := e.authorize(ctx, request.Permission, e.permissionClaim(plan, step, request, OperationRelease)); err != nil {
			return err
		}
	}
	if e.registry.load() != plan.state {
		return ErrContractStale
	}
	for index := range leases {
		if err := e.runtimeQuarantineError(steps[index].contribution.Artifact); err != nil {
			return err
		}
		leaseErr, panicked := contentContextErrSafely(leases[index].leaseCtx)
		if panicked {
			e.quarantine.mark(steps[index].contribution.Artifact)
			return errors.Join(ErrRuntimeUnavailable, ErrRuntimeQuarantined)
		}
		if leaseErr != nil {
			if ctx.Err() != nil {
				return executionContextError(ctx)
			}
			return ErrRuntimeUnavailable
		}
	}
	return nil
}

func (e *Executor) executionCacheTags(request ExecutionRequest, used []plannedBinding) ([]string, error) {
	tags := append([]string(nil), request.CacheTags...)
	tags = append(tags, "content:"+request.TargetID)
	for _, step := range uniquePlannedBindings(used) {
		tags = append(tags, step.binding.CacheTags...)
	}
	return normalizeCacheTags(tags, e.limits.MaxCacheTags)
}

func (e *Executor) executionCacheKey(
	plan executionPlan,
	request ExecutionRequest,
	result ExecutionResult,
	serialized SerializedContent,
) string {
	digest := sha256.New()
	for _, value := range []string{
		ExecutionSchemaVersion, plan.state.digest, e.bindingDigest,
		plan.target.ID, plan.target.ContractVersion,
		request.Permission.ActorFingerprint, request.Permission.PolicyFingerprint,
		request.ResourceID, request.Locale, request.Scope,
		serialized.SchemaVersion, serialized.ContentID, serialized.ContractVersion,
		serialized.StorageVersion, serialized.MediaType,
		result.Render.SchemaVersion, result.Render.ContentID, result.Render.ContractVersion,
		result.Render.TextEncoding, result.Render.PlainText,
	} {
		writeExecutionDigestString(digest, value)
	}
	serializedDigest := sha256.Sum256(serialized.Data)
	_, _ = digest.Write(serializedDigest[:])
	writeExecutionDigestUint64(digest, uint64(len(result.Render.Segments)))
	for _, segment := range result.Render.Segments {
		writeExecutionDigestString(digest, segment.Kind)
		writeExecutionDigestString(digest, segment.HTML)
		writeExecutionDigestString(digest, segment.Text)
	}
	// Tags participate in identity because a cache entry written without a
	// later request's additional invalidation tag would otherwise survive that
	// tag's invalidation despite sharing the same key.
	writeExecutionDigestUint64(digest, uint64(len(result.CacheTags)))
	for _, tag := range result.CacheTags {
		writeExecutionDigestString(digest, tag)
	}
	writeExecutionDigestUint64(digest, uint64(len(result.Attribution)))
	for _, item := range result.Attribution {
		writeExecutionDigestString(digest, item.ContentID)
		writeExecutionDigestString(digest, item.ContractVersion)
		writeExecutionDigestString(digest, item.Action)
		writeExecutionDigestArtifact(digest, item.Artifact)
		writeExecutionDigestUint64(digest, uint64(int64(item.Priority)))
	}
	flags := byte(0)
	if result.FallbackUsed {
		flags |= 1 << 0
	}
	if result.SourcePreserved {
		flags |= 1 << 1
	}
	if result.Hidden {
		flags |= 1 << 2
	}
	_, _ = digest.Write([]byte{flags})
	return "content:" + hex.EncodeToString(digest.Sum(nil))
}

func executionAttribution(used []plannedBinding) []Attribution {
	steps := uniquePlannedBindings(used)
	result := make([]Attribution, 0, len(steps))
	for _, step := range steps {
		result = append(result, Attribution{
			ContentID: step.contribution.ID, ContractVersion: step.contribution.ContractVersion,
			Action: step.binding.Action, Priority: step.binding.Priority, Artifact: step.contribution.Artifact,
		})
	}
	return result
}

func uniquePlannedBindings(input []plannedBinding) []plannedBinding {
	seen := make(map[string]struct{}, len(input))
	result := make([]plannedBinding, 0, len(input))
	for _, step := range input {
		key := step.contribution.ID + "\x00" + step.binding.Action + "\x00" + step.contribution.Artifact.PackageDigest
		if _, duplicate := seen[key]; duplicate {
			continue
		}
		seen[key] = struct{}{}
		result = append(result, step)
	}
	return result
}

func modifierCanFallback(fallback string) bool {
	return fallback == FallbackOmit || fallback == FallbackBase
}

func declaredFallbackAllowed(err error) bool {
	return errors.Is(err, ErrProviderFailed)
}

func releaseAdmissionLease(lease AdmissionLease) (panicked bool) {
	if lease == nil {
		return false
	}
	defer func() { panicked = recover() != nil }()
	lease.Release()
	return false
}

func executionContextError(ctx context.Context) error {
	if errors.Is(ctx.Err(), context.DeadlineExceeded) {
		return ErrExecutionTimeout
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	return ErrExecutionTimeout
}

func deadlineReached(ctx context.Context) bool {
	deadline, ok := ctx.Deadline()
	return ok && !time.Now().Before(deadline)
}

func traceOutcome(err error) TraceOutcome {
	switch {
	case err == nil:
		return TraceSucceeded
	case errors.Is(err, ErrExecutionDenied):
		return TraceDenied
	case errors.Is(err, ErrRuntimeUnavailable), errors.Is(err, ErrContractStale):
		return TraceStale
	case errors.Is(err, ErrSchemaRejected):
		return TraceSchemaRejected
	case errors.Is(err, ErrExecutionTimeout), errors.Is(err, context.DeadlineExceeded):
		return TraceTimedOut
	case errors.Is(err, ErrProviderPanic):
		return TracePanicked
	default:
		return TraceFailed
	}
}

func (e *Executor) recordTrace(plan executionPlan, step plannedBinding, operation string, outcome TraceOutcome, duration time.Duration) {
	if e.trace == nil {
		return
	}
	e.appendTrace(ContentTraceEvent{
		Revision: plan.state.revision, TargetID: plan.target.ID,
		ContentID: step.contribution.ID, ContractVersion: step.contribution.ContractVersion,
		Action: step.binding.Action, Operation: operation, Artifact: step.contribution.Artifact,
		Outcome: outcome, Duration: duration,
	})
}

func (e *Executor) recordFallback(plan executionPlan, step plannedBinding, operation string) {
	if e.trace == nil {
		return
	}
	e.appendTrace(ContentTraceEvent{
		Revision: plan.state.revision, TargetID: plan.target.ID,
		ContentID: step.contribution.ID, ContractVersion: step.contribution.ContractVersion,
		Action: step.binding.Action, Operation: operation, Artifact: step.contribution.Artifact,
		Outcome: TraceFallback, Fallback: step.binding.Fallback,
	})
}
