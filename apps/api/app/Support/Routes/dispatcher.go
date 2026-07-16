package routes

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	extensionmanifest "github.com/zhuchunshu/sforum/apps/api/app/Support/ExtensionManifest"
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
	Method           string
	Path             string
	Query            string
	Headers          http.Header
	Body             []byte
	Params           map[string]string
	ActorID          int64
	Authenticated    bool
	CredentialSource DispatchCredentialSource
	Permissions      map[string]bool
	ClientIP         string
}

type DispatchCredentialSource string

const (
	DispatchCredentialCookie DispatchCredentialSource = "cookie"
	DispatchCredentialBearer DispatchCredentialSource = "bearer"
)

type DispatchResponse struct {
	Status  int
	Headers http.Header
	Body    []byte
}

type DispatchResult struct {
	Handled  bool
	Response DispatchResponse
}

type InvocationStage string

const (
	InvocationStageExecute InvocationStage = "execute"
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
		idempotency: config.Idempotency, failures: config.Failures, defaultTimeout: timeout,
	}
}

func (d *Dispatcher) Dispatch(ctx context.Context, request DispatchRequest, core CoreInvoker) (DispatchResult, error) {
	if d == nil || d.plans == nil || ctx == nil {
		return DispatchResult{}, ErrDispatchInvalid
	}
	request.Method = strings.ToUpper(strings.TrimSpace(request.Method))
	if request.Method == "" || strings.TrimSpace(request.Path) == "" {
		return DispatchResult{}, ErrDispatchInvalid
	}
	plan, err := d.plans.BuildExecutionPlan(ctx, request.Method, request.Path)
	if err != nil {
		if errors.Is(err, ErrRouteNotFound) {
			return DispatchResult{}, nil
		}
		return DispatchResult{}, err
	}
	if !plan.Valid() {
		return DispatchResult{}, ErrInvalidExecutionPlan
	}
	chain := plan.Chain()
	if !dispatchPlanHasPluginStep(chain) {
		// Core-only requests stay entirely on Fiber's existing path. Capturing them
		// would silently turn downloads, streams, and protocol upgrades into buffers.
		return DispatchResult{}, nil
	}
	request.Params = plan.Params()
	request.Headers = cloneHTTPHeader(request.Headers)
	request.Body = append([]byte(nil), request.Body...)
	request.Permissions = cloneDispatchPermissions(request.Permissions)

	commit := NewRouteCommitObserver()
	var idempotencyLease RouteIdempotencyLease
	preservePending := false
	terminal := plan.Terminal()
	if terminal.Provider.Kind == ProviderPlugin && d.policies != nil {
		policy, policyErr := d.policies.ResolveRouteExecutionPolicy(terminal)
		if policyErr != nil && !errors.Is(policyErr, ErrRoutePolicyNotFound) {
			return DispatchResult{}, fmt.Errorf("%w: %v", ErrDispatchIdempotencyUnavailable, policyErr)
		}
		if policyErr == nil && policy.IdempotencyRequired {
			if terminal.Mode != extensionmanifest.RouteModeHTTP || d.idempotency == nil {
				return DispatchResult{}, ErrDispatchIdempotencyUnavailable
			}
			var replay *DispatchResponse
			idempotencyLease, replay, err = d.idempotency.Begin(ctx, plan, terminal, policy, request)
			if err != nil {
				return DispatchResult{}, err
			}
			if replay != nil {
				if err := d.authorizeReplay(ctx, plan, chain, request); err != nil {
					return DispatchResult{}, err
				}
				return DispatchResult{Handled: true, Response: cloneDispatchResponse(*replay)}, nil
			}
			if idempotencyLease == nil {
				return DispatchResult{}, ErrDispatchIdempotencyUnavailable
			}
			defer func() {
				if !preservePending && !commit.ExecutionObserved() {
					_ = idempotencyLease.Abort(ctx)
				}
			}()
		}
	}

	var response *DispatchResponse
	committingStep := -1
	var committingStarted time.Time
	var committedAfterFailure *RouteCommittedAfterFailure
	for index, step := range chain {
		started := time.Now()
		authority, err := d.authorize(ctx, plan, index, step, request, response, InvocationStageExecute, commit)
		if err != nil {
			d.appendTrace(plan, index, step, RouteTraceDenied, started, commit.State())
			if event := d.committedAfterFailure(plan, index, step, request, response, RouteFailureGuardDenied); event != nil {
				committedAfterFailure, committingStep, committingStarted = event, index, started
				break
			}
			return DispatchResult{}, err
		}
		if d.schemas == nil && step.RequestSchema != "" {
			d.appendTrace(plan, index, step, RouteTraceSchemaRejected, started, commit.State())
			if event := d.committedAfterFailure(plan, index, step, request, response, RouteFailureRequestSchemaRejected); event != nil {
				committedAfterFailure, committingStep, committingStarted = event, index, started
				break
			}
			return DispatchResult{}, fmt.Errorf("%w: request validator is unavailable", ErrDispatchSchema)
		}
		if d.schemas != nil && step.RequestSchema != "" {
			if err := d.schemas.ValidateRequest(ctx, step, request); err != nil {
				d.appendTrace(plan, index, step, RouteTraceSchemaRejected, started, commit.State())
				if event := d.committedAfterFailure(plan, index, step, request, response, RouteFailureRequestSchemaRejected); event != nil {
					committedAfterFailure, committingStep, committingStarted = event, index, started
					break
				}
				return DispatchResult{}, fmt.Errorf("%w: %v", ErrDispatchSchema, err)
			}
		}

		var invocation RouteInvocationResult
		if step.Phase == RoutePhaseHandler && step.Provider.Kind == ProviderCore ||
			step.Phase == RoutePhaseHandler && (step.Action == extensionmanifest.RouteActionAlias || step.Action == extensionmanifest.RouteActionRewrite) {
			if core == nil {
				d.appendTrace(plan, index, step, RouteTraceTransportFailed, started, commit.State())
				return DispatchResult{}, fmt.Errorf("%w: core handler is unavailable", ErrDispatchTransport)
			}
			value, callErr := core.InvokeCore(ctx, step, request)
			if callErr != nil {
				d.appendTrace(plan, index, step, RouteTraceTransportFailed, started, commit.State())
				return DispatchResult{}, callErr
			}
			invocation.Response = &value
		} else if step.Phase == RoutePhaseHandler && step.Action == extensionmanifest.RouteActionRedirect {
			invocation.Response = &DispatchResponse{Status: http.StatusPermanentRedirect, Headers: http.Header{"Location": []string{step.Destination}}}
		} else {
			invocation, err = d.invokePlugin(ctx, plan, index, step, request, response, commit, authority)
			if err != nil {
				// Transport errors may arrive after a remote side effect or response began.
				// Advance the fence before evaluating any safe-method fallback.
				if invocation.SideEffectStarted {
					commit.SideEffectStarted()
				}
				if invocation.ResponseStarted {
					commit.ResponseStarted()
				}
				d.appendTrace(plan, index, step, RouteTraceTransportFailed, started, commit.State())
				if event := d.committedAfterFailure(plan, index, step, request, response, RouteFailureTransportFailed); event != nil {
					committedAfterFailure, committingStep, committingStarted = event, index, started
					break
				}
				fallback, fallbackErr := d.fallback(ctx, plan, index, step, request, core, commit)
				if fallbackErr != nil {
					return DispatchResult{}, fallbackErr
				}
				if fallback != nil {
					response = fallback
					d.appendTrace(plan, index, step, RouteTraceFallbackUsed, started, commit.State())
					committingStep = index
					committingStarted = started
					break
				}
				if plan.AllowsFallback(index, commit.State()) && step.Phase != RoutePhaseHandler {
					d.appendTrace(plan, index, step, RouteTraceFallbackUsed, started, commit.State())
					committingStep = index
					committingStarted = started
					continue
				}
				return DispatchResult{}, err
			}
		}

		if invocation.SideEffectStarted {
			commit.SideEffectStarted()
		}
		if invocation.ResponseStarted {
			commit.ResponseStarted()
		}
		if invocation.Request != nil {
			request = cloneDispatchRequest(*invocation.Request)
		}
		if invocation.Response != nil {
			value := cloneDispatchResponse(*invocation.Response)
			if d.schemas == nil && step.ResponseSchema != "" {
				d.appendTrace(plan, index, step, RouteTraceSchemaRejected, started, commit.State())
				if event := d.committedAfterFailure(plan, index, step, request, response, RouteFailureResponseSchemaRejected); event != nil {
					committedAfterFailure, committingStep, committingStarted = event, index, started
					break
				}
				return DispatchResult{}, fmt.Errorf("%w: response validator is unavailable", ErrDispatchSchema)
			}
			if d.schemas != nil && step.ResponseSchema != "" {
				if err := d.schemas.ValidateResponse(ctx, step, request, value); err != nil {
					d.appendTrace(plan, index, step, RouteTraceSchemaRejected, started, commit.State())
					if event := d.committedAfterFailure(plan, index, step, request, response, RouteFailureResponseSchemaRejected); event != nil {
						committedAfterFailure, committingStep, committingStarted = event, index, started
						break
					}
					return DispatchResult{}, fmt.Errorf("%w: %v", ErrDispatchSchema, err)
				}
			}
			response = &value
		}
		if step.Provider.Kind == ProviderPlugin {
			d.appendTrace(plan, index, step, RouteTraceSucceeded, started, commit.State())
			committingStep = index
			committingStarted = started
		}
	}
	if response == nil {
		return DispatchResult{}, fmt.Errorf("%w: chain produced no response", ErrDispatchTransport)
	}
	if !commit.Finalize() {
		return DispatchResult{}, ErrDispatchAlreadyCommitted
	}
	if committingStep >= 0 {
		d.appendTrace(plan, committingStep, chain[committingStep], RouteTraceCommitted, committingStarted, commit.State())
	}
	if committedAfterFailure != nil {
		committedAfterFailure.CommitState = commit.State()
		d.failures.RecordCommittedAfterFailure(ctx, *committedAfterFailure)
	}
	if idempotencyLease != nil && response.Status >= http.StatusOK && response.Status < http.StatusMultipleChoices {
		// Complete 失败时保留 pending；客户端只能得到 fail-closed unavailable，
		// 不能在未知持久化结果后再次执行插件副作用。
		preservePending = true
		if err := idempotencyLease.Complete(ctx, *response); err != nil {
			return DispatchResult{}, fmt.Errorf("%w: %v", ErrDispatchIdempotencyUnavailable, err)
		}
	}
	return DispatchResult{Handled: true, Response: cloneDispatchResponse(*response)}, nil
}

func (d *Dispatcher) committedAfterFailure(
	plan RouteExecutionPlan,
	stepIndex int,
	step RouteExecutionStep,
	request DispatchRequest,
	response *DispatchResponse,
	code RouteFailureCode,
) *RouteCommittedAfterFailure {
	if d == nil || d.failures == nil || !plan.UnsafeMethod() || response == nil ||
		step.Provider.Kind != ProviderPlugin || step.Phase != RoutePhaseAfter {
		return nil
	}
	return &RouteCommittedAfterFailure{
		Revision: plan.Revision(), StepIndex: stepIndex, Phase: step.Phase, Action: step.Action,
		RouteID: step.RouteID, ContractVersion: step.ContractVersion, Method: plan.Method(),
		PathSignature: routeStepPathSignature(step), FailureCode: code, ActorID: request.ActorID,
		ResponseStatus: response.Status, Artifact: step.Provider.Artifact,
	}
}

func (d *Dispatcher) authorizeReplay(
	ctx context.Context,
	plan RouteExecutionPlan,
	chain []RouteExecutionStep,
	request DispatchRequest,
) error {
	for index, step := range chain {
		if step.Provider.Kind != ProviderPlugin {
			continue
		}
		if _, err := d.authorize(ctx, plan, index, step, request, nil, InvocationStageExecute, nil); err != nil {
			d.appendTrace(plan, index, step, RouteTraceDenied, time.Now(), RouteCommitPristine)
			return err
		}
	}
	return nil
}

func (d *Dispatcher) appendTrace(
	plan RouteExecutionPlan,
	index int,
	step RouteExecutionStep,
	outcome RouteTraceOutcome,
	started time.Time,
	state RouteExecutionCommitState,
) {
	if d == nil || d.trace == nil || step.Provider.Kind != ProviderPlugin {
		return
	}
	d.trace.AppendRouteTrace(RouteTraceEvent{
		Revision: plan.Revision(), StepIndex: index, Phase: step.Phase, Action: step.Action,
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
		PlanRevision: plan.Revision(), StepIndex: index, Step: step, Stage: InvocationStageExecute,
		Request: cloneDispatchRequest(request), Response: current, Commit: commit, authority: authority,
	})
	if err != nil {
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
		value, err := core.InvokeCore(ctx, step, request)
		if err != nil {
			return nil, err
		}
		return &value, nil
	default:
		return nil, nil
	}
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
