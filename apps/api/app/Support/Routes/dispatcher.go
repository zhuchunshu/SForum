package routes

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"
)

var (
	ErrDispatchInvalid                = errors.New("routes: invalid dispatch request")
	ErrDispatchDenied                 = errors.New("routes: route guard denied")
	ErrDispatchSchema                 = errors.New("routes: route schema rejected")
	ErrDispatchTransport              = errors.New("routes: route transport unavailable")
	ErrDispatchAlreadyCommitted       = errors.New("routes: route writer already committed")
	ErrDispatchIdempotencyKeyInvalid  = errors.New("routes: required idempotency key is invalid")
	ErrDispatchIdempotencyInProgress  = errors.New("routes: idempotent request is in progress")
	ErrDispatchIdempotencyConflict    = errors.New("routes: idempotency key request conflict")
	ErrDispatchIdempotencyUnavailable = errors.New("routes: idempotency replay is unavailable")
)

type DispatchRequest struct {
	Method            string
	Path              string
	Query             string
	Headers           http.Header
	Body              []byte
	Params            map[string]string
	ActorID           int64
	Authenticated     bool
	CredentialSource  DispatchCredentialSource
	Permissions       map[string]bool
	ClientIP          string
	hostMutatedParams bool
}

// HostMutatedParams reports whether the Dispatcher applied an exact published
// route-params mutation operation through the Host Mutation Engine.
// The proof bit is deliberately unexported so HTTP callers cannot manufacture it.
func (r DispatchRequest) HostMutatedParams() bool { return r.hostMutatedParams }

type DispatchCredentialSource string

const (
	DispatchCredentialCookie DispatchCredentialSource = "cookie"
	DispatchCredentialBearer DispatchCredentialSource = "bearer"
)

type DispatchResponse struct {
	Status        int
	Headers       http.Header
	Body          []byte
	CanonicalPath string
}

type DispatchResult struct {
	Handled  bool
	Response DispatchResponse
}

const routeIdempotencyReplayedHeader = "Idempotency-Replayed"

type InvocationStage string

const (
	InvocationStageRequest  InvocationStage = "request"
	InvocationStageHandler  InvocationStage = "handler"
	InvocationStageResponse InvocationStage = "response"

	// InvocationStageExecute is retained until every v1 Host adapter has moved
	// to the explicit handler stage.
	// Deprecated: use InvocationStageHandler.
	InvocationStageExecute InvocationStage = InvocationStageHandler
)

// RouteInvocation is transport-neutral. Buffered HTTP is the first adapter;
// stream/SSE/WebSocket transports can consume the same exact plan and commit observer.
type RouteInvocation struct {
	PlanRevision uint64
	StepIndex    int
	Step         RouteExecutionStep
	Stage        InvocationStage
	Request      DispatchRequest
	Response     *DispatchResponse
	Commit       *RouteCommitObserver
	authority    routeInvocationAuthority
}

type RouteInvocationResult struct {
	Request           *DispatchRequest
	Response          *DispatchResponse
	RequestPatch      []RoutePatchOperation
	ResponsePatch     []RoutePatchOperation
	ResponseStarted   bool
	SideEffectStarted bool
}

type StepInvoker interface {
	SupportsMode(mode string) bool
	Invoke(context.Context, RouteInvocation) (RouteInvocationResult, error)
}

type GuardAuthorizer interface {
	Authorize(context.Context, RouteExecutionPlan, RouteExecutionStep, DispatchRequest) error
}

type SchemaValidator interface {
	ValidateRequest(context.Context, RouteExecutionStep, DispatchRequest) error
	ValidateResponse(context.Context, RouteExecutionStep, DispatchRequest, DispatchResponse) error
}

type CoreInvoker interface {
	InvokeCore(context.Context, RouteExecutionStep, DispatchRequest) (DispatchResponse, error)
}

type PlanResolver interface {
	BuildExecutionPlan(context.Context, string, string) (RouteExecutionPlan, error)
}

type DispatcherConfig struct {
	Plans          PlanResolver
	Steps          StepInvoker
	Guard          GuardAuthorizer
	Schemas        SchemaValidator
	Trace          RouteTraceSink
	Policies       RoutePolicyResolver
	Idempotency    RouteIdempotencyController
	Failures       RouteFailureSink
	StreamFailures RouteStreamFailureSink
	DefaultTimeout time.Duration
}

type Dispatcher struct {
	plans          PlanResolver
	steps          StepInvoker
	guard          GuardAuthorizer
	schemas        SchemaValidator
	trace          RouteTraceSink
	policies       RoutePolicyResolver
	idempotency    RouteIdempotencyController
	failures       RouteFailureSink
	streamFailures RouteStreamFailureSink
	defaultTimeout time.Duration
}

func NewDispatcher(config DispatcherConfig) *Dispatcher {
	timeout := config.DefaultTimeout
	if timeout <= 0 {
		timeout = 3 * time.Second
	}
	return &Dispatcher{
		plans: config.Plans, steps: config.Steps, guard: config.Guard,
		schemas: config.Schemas, trace: config.Trace, policies: config.Policies,
		idempotency: config.Idempotency, failures: config.Failures,
		streamFailures: config.StreamFailures, defaultTimeout: timeout,
	}
}

func routeRedirectLocation(step RouteExecutionStep) (string, error) {
	location := step.Destination
	if step.TargetID != "" {
		location = step.TargetPath
	}
	if location == "" || !strings.HasPrefix(location, "/") ||
		len(location) > 1 && (location[1] == '/' || location[1] == '\\') ||
		strings.ContainsAny(location, "?#\\\r\n") {
		return "", fmt.Errorf("%w: redirect target path is invalid", ErrInvalidExecutionPlan)
	}
	parsed, err := url.ParseRequestURI(location)
	if err != nil || parsed.IsAbs() || parsed.Host != "" || parsed.RawQuery != "" || parsed.Fragment != "" {
		return "", fmt.Errorf("%w: redirect target path is invalid", ErrInvalidExecutionPlan)
	}
	return parsed.EscapedPath(), nil
}

func (d *Dispatcher) committedAfterFailure(
	plan RouteExecutionPlan,
	stepIndex int,
	step RouteExecutionStep,
	stage InvocationStage,
	request DispatchRequest,
	response *DispatchResponse,
	code RouteFailureCode,
	runtimeExecutionObserved bool,
) *RouteCommittedAfterFailure {
	if d == nil || d.failures == nil || !plan.UnsafeMethod() || response == nil ||
		step.Provider.Kind != ProviderPlugin || stage != InvocationStageResponse || !responseStageAction(step.Action) {
		return nil
	}
	return &RouteCommittedAfterFailure{
		Revision: plan.Revision(), StepIndex: stepIndex, Phase: step.Phase, InvocationStage: stage, Action: step.Action,
		RouteID: step.RouteID, ContractVersion: step.ContractVersion, Method: plan.Method(),
		PathSignature: routeStepPathSignature(step), FailureCode: code,
		RuntimeExecutionObserved: runtimeExecutionObserved, ActorID: request.ActorID,
		ResponseStatus: response.Status, Artifact: step.Provider.Artifact,
	}
}

func (d *Dispatcher) authorizeReplay(
	ctx context.Context,
	plan RouteExecutionPlan,
	request DispatchRequest,
	response *DispatchResponse,
	authorization *RouteReplayAuthorization,
	binding RouteReplayBinding,
	responseContractKnown bool,
	responseContract *RouteReplayResponseContract,
) error {
	terminal := plan.Terminal()
	if response == nil || response.Status < http.StatusOK || response.Status >= http.StatusMultipleChoices ||
		!ValidTerminalResponseStatus(terminal.Mode, response.Status) {
		return ErrDispatchIdempotencyUnavailable
	}
	sequence, err := bufferedRouteInvocationSequence(plan)
	if err != nil {
		return err
	}
	chain := plan.Chain()
	if authorization != nil && !routeReplayAuthorizationMatchesPlan(authorization, binding, sequence) {
		return ErrDispatchIdempotencyUnavailable
	}
	finalResponseContract, hasFinalResponseContract, contractErr := resolveRouteReplayResponseContract(
		sequence, chain, responseContractKnown, responseContract,
	)
	if contractErr != nil {
		return ErrDispatchIdempotencyUnavailable
	}
	mutationIndex := 0
	for _, execution := range sequence {
		index, stage := execution.index, execution.stage
		step := chain[index]
		if step.Provider.Kind != ProviderPlugin {
			continue
		}
		var prior *DispatchResponse
		if stage == InvocationStageResponse && response != nil {
			value := cloneDispatchResponse(*response)
			prior = &value
		}
		authority, err := d.authorize(ctx, plan, index, step, request, prior, stage, nil)
		if err != nil {
			if ctxErr := ctx.Err(); ctxErr != nil && errors.Is(err, ctxErr) {
				return ctxErr
			}
			outcome, _, _ := classifyRouteGuardFailure(err)
			d.appendTrace(plan, index, step, stage, outcome, time.Now(), RouteCommitPristine)
			return err
		}
		if d.schemas == nil && step.RequestSchema != "" {
			return ErrDispatchIdempotencyUnavailable
		}
		if d.schemas != nil && step.RequestSchema != "" {
			if err := d.schemas.ValidateRequest(ctx, step, request); err != nil {
				return ErrDispatchIdempotencyUnavailable
			}
		}
		if stage != InvocationStageRequest || authorization == nil {
			continue
		}
		mutation := authorization.RequestMutations[mutationIndex]
		mutationIndex++
		beforeDigest, digestErr := routeReplayRequestDigest(request)
		if digestErr != nil || beforeDigest != mutation.BeforeDigest ||
			planTerminalUsesFrozenPathParams(plan) && routeRequestPatchMutatesParams(mutation.Operations) {
			return ErrDispatchIdempotencyUnavailable
		}
		value, patchErr := applyRouteRequestPatch(
			request, mutation.Operations, step.MutableRequestFields, rawMutationAuthority(authority),
		)
		if patchErr != nil {
			return ErrDispatchIdempotencyUnavailable
		}
		if routeRequestPatchMutatesParams(mutation.Operations) {
			value.hostMutatedParams = true
		}
		if d.schemas != nil && step.RequestSchema != "" {
			if err := d.schemas.ValidateRequest(ctx, step, value); err != nil {
				return ErrDispatchIdempotencyUnavailable
			}
		}
		afterDigest, digestErr := routeReplayRequestDigest(value)
		if digestErr != nil || afterDigest != mutation.AfterDigest {
			return ErrDispatchIdempotencyUnavailable
		}
		request = value
	}
	if authorization != nil && mutationIndex != len(authorization.RequestMutations) {
		return ErrDispatchIdempotencyUnavailable
	}
	if hasFinalResponseContract {
		if d.schemas == nil {
			return ErrDispatchIdempotencyUnavailable
		}
		if err := d.schemas.ValidateResponse(ctx, chain[finalResponseContract.index], request, *response); err != nil {
			return ErrDispatchIdempotencyUnavailable
		}
	}
	return nil
}

func lastRouteResponseContract(
	sequence []routeInvocationExecution,
	chain []RouteExecutionStep,
) (routeInvocationExecution, bool) {
	for index := len(sequence) - 1; index >= 0; index-- {
		execution := sequence[index]
		if execution.stage != InvocationStageRequest &&
			execution.index >= 0 && execution.index < len(chain) &&
			strings.TrimSpace(chain[execution.index].ResponseSchema) != "" {
			return execution, true
		}
	}
	return routeInvocationExecution{}, false
}

func newRouteReplayResponseContract(
	execution routeInvocationExecution,
	step RouteExecutionStep,
) *RouteReplayResponseContract {
	return &RouteReplayResponseContract{
		StepIndex: execution.index, InvocationStage: execution.stage,
		RouteID: step.RouteID, ContractVersion: step.ContractVersion,
		ResponseSchema: step.ResponseSchema,
	}
}

func resolveRouteReplayResponseContract(
	sequence []routeInvocationExecution,
	chain []RouteExecutionStep,
	known bool,
	contract *RouteReplayResponseContract,
) (routeInvocationExecution, bool, error) {
	if !known {
		if contract != nil {
			return routeInvocationExecution{}, false, ErrDispatchIdempotencyUnavailable
		}
		execution, ok := lastRouteResponseContract(sequence, chain)
		return execution, ok, nil
	}
	if contract == nil {
		return routeInvocationExecution{}, false, nil
	}
	if contract.StepIndex < 0 || contract.StepIndex >= len(chain) ||
		(contract.InvocationStage != InvocationStageHandler && contract.InvocationStage != InvocationStageResponse) {
		return routeInvocationExecution{}, false, ErrDispatchIdempotencyUnavailable
	}
	execution := routeInvocationExecution{index: contract.StepIndex, stage: contract.InvocationStage}
	found := false
	for _, candidate := range sequence {
		if candidate == execution {
			found = true
			break
		}
	}
	step := chain[contract.StepIndex]
	if !found || strings.TrimSpace(contract.ResponseSchema) == "" ||
		contract.RouteID != step.RouteID || contract.ContractVersion != step.ContractVersion ||
		contract.ResponseSchema != step.ResponseSchema {
		return routeInvocationExecution{}, false, ErrDispatchIdempotencyUnavailable
	}
	return execution, true, nil
}

func classifyRouteGuardFailure(err error) (RouteTraceOutcome, RouteFailureCode, bool) {
	var pluginFailure *PluginGuardFailure
	if errors.As(err, &pluginFailure) {
		if pluginFailure.Kind() != PluginGuardFailureDenied {
			return RouteTraceTransportFailed, RouteFailureTransportFailed, pluginFailure.RuntimeExecutionObserved()
		}
		return RouteTraceDenied, RouteFailureGuardDenied, pluginFailure.RuntimeExecutionObserved()
	}
	return RouteTraceDenied, RouteFailureGuardDenied, false
}

func (d *Dispatcher) appendTrace(
	plan RouteExecutionPlan,
	index int,
	step RouteExecutionStep,
	stage InvocationStage,
	outcome RouteTraceOutcome,
	started time.Time,
	state RouteExecutionCommitState,
) {
	if d == nil || d.trace == nil || step.Provider.Kind != ProviderPlugin {
		return
	}
	d.trace.AppendRouteTrace(RouteTraceEvent{
		Revision: plan.Revision(), StepIndex: index, Phase: step.Phase, InvocationStage: stage, Action: step.Action,
		RouteID: step.RouteID, ContractVersion: step.ContractVersion, Method: plan.Method(),
		PathSignature: routeStepPathSignature(step), Mode: step.Mode, Fallback: step.Fallback,
		Outcome: outcome, Duration: time.Since(started), CommitState: state, Provider: step.Provider,
	})
}

func dispatchPlanHasPluginStep(chain []RouteExecutionStep) bool {
	for _, step := range chain {
		if step.Provider.Kind == ProviderPlugin {
			return true
		}
	}
	return false
}

func (d *Dispatcher) authorize(
	ctx context.Context,
	plan RouteExecutionPlan,
	stepIndex int,
	step RouteExecutionStep,
	request DispatchRequest,
	response *DispatchResponse,
	stage InvocationStage,
	commit *RouteCommitObserver,
) (routeInvocationAuthority, error) {
	if step.Provider.Kind == ProviderCore {
		return routeInvocationAuthority{}, nil
	}
	if d.guard == nil {
		return routeInvocationAuthority{}, fmt.Errorf("%w: guard evaluator is unavailable", ErrDispatchDenied)
	}
	var (
		authorization RouteGuardAuthorization
		err           error
	)
	if typed, ok := d.guard.(interface {
		AuthorizeRoute(context.Context, RouteExecutionPlan, int, RouteExecutionStep, DispatchRequest) (RouteGuardAuthorization, error)
	}); ok {
		authorization, err = typed.AuthorizeRoute(ctx, plan, stepIndex, step, request)
	} else {
		err = d.guard.Authorize(ctx, plan, step, request)
		if err == nil {
			var valid bool
			authorization, valid = legacyFilteredRouteGuardAuthorization(plan, stepIndex, step, request)
			if !valid {
				err = ErrDispatchDenied
			}
		}
	}
	if err != nil {
		if ctxErr := ctx.Err(); ctxErr != nil && errors.Is(err, ctxErr) {
			return routeInvocationAuthority{}, newRouteObservedCallerCancellation(ctxErr)
		}
		var pluginFailure *PluginGuardFailure
		if errors.As(err, &pluginFailure) && pluginFailure.Kind() != PluginGuardFailureDenied {
			return routeInvocationAuthority{}, fmt.Errorf("%w: %w", ErrDispatchTransport, err)
		}
		return routeInvocationAuthority{}, fmt.Errorf("%w: %w", ErrDispatchDenied, err)
	}
	authority, valid := newRouteInvocationAuthority(plan, stepIndex, step, request, response, authorization, stage, commit)
	if !valid {
		return routeInvocationAuthority{}, fmt.Errorf("%w: invalid guard authorization proof", ErrDispatchDenied)
	}
	return authority, nil
}

func (d *Dispatcher) invokePlugin(
	ctx context.Context,
	plan RouteExecutionPlan,
	index int,
	step RouteExecutionStep,
	stage InvocationStage,
	request DispatchRequest,
	response *DispatchResponse,
	commit *RouteCommitObserver,
	authority routeInvocationAuthority,
) (RouteInvocationResult, error) {
	if d.steps == nil || !d.steps.SupportsMode(step.Mode) {
		return RouteInvocationResult{}, fmt.Errorf("%w: mode %q", ErrDispatchTransport, step.Mode)
	}
	timeout := d.defaultTimeout
	if step.TimeoutMS > 0 {
		timeout = time.Duration(step.TimeoutMS) * time.Millisecond
	}
	callCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	var current *DispatchResponse
	if response != nil {
		value := cloneDispatchResponse(*response)
		current = &value
	}
	result, err := d.steps.Invoke(callCtx, RouteInvocation{
		PlanRevision: plan.Revision(), StepIndex: index, Step: step, Stage: stage,
		Request: cloneDispatchRequest(request), Response: current, Commit: commit, authority: authority,
	})
	if err != nil {
		if stage == InvocationStageResponse && response != nil && routeCallerCancellation(ctx, err) {
			return result, newRouteObservedCallerCancellation(ctx.Err())
		}
		return result, fmt.Errorf("%w: %w", ErrDispatchTransport, err)
	}
	return result, nil
}

func (d *Dispatcher) fallback(
	ctx context.Context,
	plan RouteExecutionPlan,
	index int,
	step RouteExecutionStep,
	request DispatchRequest,
	core CoreInvoker,
	commit *RouteCommitObserver,
) (*DispatchResponse, error) {
	if !plan.AllowsFallback(index, commit.State()) {
		return nil, nil
	}
	switch step.Fallback {
	case "not_found":
		value := DispatchResponse{Status: http.StatusNotFound}
		return &value, nil
	case "readonly_core":
		if step.Phase != RoutePhaseHandler {
			return nil, nil
		}
		if core == nil {
			return nil, fmt.Errorf("%w: readonly core fallback is unavailable", ErrDispatchTransport)
		}
		value, err := invokeCoreWithCommitEvidence(ctx, step, request, core, commit)
		if err != nil {
			return nil, err
		}
		return &value, nil
	default:
		return nil, nil
	}
}

func invokeCoreWithCommitEvidence(
	ctx context.Context,
	step RouteExecutionStep,
	request DispatchRequest,
	core CoreInvoker,
	commit *RouteCommitObserver,
) (DispatchResponse, error) {
	if err := ctx.Err(); err != nil {
		return DispatchResponse{}, err
	}
	// Core 写操作一旦交付就不能证明没有发生副作用；重放租约必须保持 pending。
	commit.SideEffectStarted()
	response, err := core.InvokeCore(ctx, step, request)
	if err != nil {
		return DispatchResponse{}, err
	}
	commit.ResponseStarted()
	return response, nil
}

// RouteCommitObserver is concurrency-safe because future streaming transports
// may report response and side-effect commits from independent goroutines.
type RouteCommitObserver struct {
	mu                sync.Mutex
	state             RouteExecutionCommitState
	executionObserved bool
}

func NewRouteCommitObserver() *RouteCommitObserver {
	return &RouteCommitObserver{state: RouteCommitPristine}
}

func (o *RouteCommitObserver) State() RouteExecutionCommitState {
	if o == nil {
		return RouteCommitUnknown
	}
	o.mu.Lock()
	defer o.mu.Unlock()
	return o.state
}

func (o *RouteCommitObserver) ResponseStarted() bool {
	return o.advance(RouteCommitResponseStarted)
}

func (o *RouteCommitObserver) SideEffectStarted() bool {
	return o.advance(RouteCommitSideEffectStarted)
}

// ExecutionObserved is monotonic and survives Finalize. Required-idempotency
// leases use it to distinguish a safe pre-dispatch failure from an unknown
// remote outcome that must remain pending.
func (o *RouteCommitObserver) ExecutionObserved() bool {
	if o == nil {
		return false
	}
	o.mu.Lock()
	defer o.mu.Unlock()
	return o.executionObserved
}

func (o *RouteCommitObserver) Finalize() bool {
	if o == nil {
		return false
	}
	o.mu.Lock()
	defer o.mu.Unlock()
	if o.state == RouteCommitFinal || o.state == RouteCommitUnknown {
		return false
	}
	o.state = RouteCommitFinal
	return true
}

func (o *RouteCommitObserver) advance(next RouteExecutionCommitState) bool {
	if o == nil {
		return false
	}
	o.mu.Lock()
	defer o.mu.Unlock()
	// Evidence can race with Finalize; record it before inspecting terminal state.
	o.executionObserved = true
	if o.state == RouteCommitSideEffectStarted && next == RouteCommitResponseStarted {
		o.state = next
		return true
	}
	if o.state != RouteCommitPristine {
		return false
	}
	o.state = next
	return true
}

func cloneDispatchRequest(value DispatchRequest) DispatchRequest {
	value.Headers = cloneHTTPHeader(value.Headers)
	value.Body = append([]byte(nil), value.Body...)
	value.Params = cloneRouteExecutionParams(value.Params)
	value.Permissions = cloneDispatchPermissions(value.Permissions)
	return value
}

func cloneDispatchResponse(value DispatchResponse) DispatchResponse {
	value.Headers = cloneHTTPHeader(value.Headers)
	value.Body = append([]byte(nil), value.Body...)
	return value
}

func cloneHTTPHeader(value http.Header) http.Header {
	if value == nil {
		return http.Header{}
	}
	return value.Clone()
}

func cloneDispatchPermissions(value map[string]bool) map[string]bool {
	result := make(map[string]bool, len(value))
	for key, allowed := range value {
		result[key] = allowed
	}
	return result
}
