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
	ErrDispatchInvalid          = errors.New("routes: invalid dispatch request")
	ErrDispatchDenied           = errors.New("routes: route guard denied")
	ErrDispatchSchema           = errors.New("routes: route schema rejected")
	ErrDispatchTransport        = errors.New("routes: route transport unavailable")
	ErrDispatchAlreadyCommitted = errors.New("routes: route writer already committed")
)

type DispatchRequest struct {
	Method        string
	Path          string
	Query         string
	Headers       http.Header
	Body          []byte
	Params        map[string]string
	ActorID       int64
	Authenticated bool
	Permissions   map[string]bool
}

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
	ValidateResponse(context.Context, RouteExecutionStep, DispatchResponse) error
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
	DefaultTimeout time.Duration
}

type Dispatcher struct {
	plans          PlanResolver
	steps          StepInvoker
	guard          GuardAuthorizer
	schemas        SchemaValidator
	defaultTimeout time.Duration
}

func NewDispatcher(config DispatcherConfig) *Dispatcher {
	timeout := config.DefaultTimeout
	if timeout <= 0 {
		timeout = 3 * time.Second
	}
	return &Dispatcher{
		plans: config.Plans, steps: config.Steps, guard: config.Guard,
		schemas: config.Schemas, defaultTimeout: timeout,
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
	request.Params = plan.Params()
	request.Headers = cloneHTTPHeader(request.Headers)
	request.Body = append([]byte(nil), request.Body...)
	request.Permissions = cloneDispatchPermissions(request.Permissions)

	commit := NewRouteCommitObserver()
	chain := plan.Chain()
	var response *DispatchResponse
	for index, step := range chain {
		if err := d.authorize(ctx, plan, step, request); err != nil {
			return DispatchResult{}, err
		}
		if d.schemas == nil && step.RequestSchema != "" {
			return DispatchResult{}, fmt.Errorf("%w: request validator is unavailable", ErrDispatchSchema)
		}
		if d.schemas != nil && step.RequestSchema != "" {
			if err := d.schemas.ValidateRequest(ctx, step, request); err != nil {
				return DispatchResult{}, fmt.Errorf("%w: %v", ErrDispatchSchema, err)
			}
		}

		var invocation RouteInvocationResult
		if step.Phase == RoutePhaseHandler && step.Provider.Kind == ProviderCore ||
			step.Phase == RoutePhaseHandler && (step.Action == extensionmanifest.RouteActionAlias || step.Action == extensionmanifest.RouteActionRewrite) {
			if core == nil {
				return DispatchResult{}, fmt.Errorf("%w: core handler is unavailable", ErrDispatchTransport)
			}
			value, callErr := core.InvokeCore(ctx, step, request)
			if callErr != nil {
				return DispatchResult{}, callErr
			}
			invocation.Response = &value
		} else if step.Phase == RoutePhaseHandler && step.Action == extensionmanifest.RouteActionRedirect {
			invocation.Response = &DispatchResponse{Status: http.StatusPermanentRedirect, Headers: http.Header{"Location": []string{step.Destination}}}
		} else {
			invocation, err = d.invokePlugin(ctx, plan, index, step, request, response, commit)
			if err != nil {
				// Transport errors may arrive after a remote side effect or response began.
				// Advance the fence before evaluating any safe-method fallback.
				if invocation.SideEffectStarted {
					commit.SideEffectStarted()
				}
				if invocation.ResponseStarted {
					commit.ResponseStarted()
				}
				fallback, fallbackErr := d.fallback(ctx, plan, index, step, request, core, commit)
				if fallbackErr != nil {
					return DispatchResult{}, fallbackErr
				}
				if fallback != nil {
					response = fallback
					break
				}
				if plan.AllowsFallback(index, commit.State()) && step.Phase != RoutePhaseHandler {
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
				return DispatchResult{}, fmt.Errorf("%w: response validator is unavailable", ErrDispatchSchema)
			}
			if d.schemas != nil && step.ResponseSchema != "" {
				if err := d.schemas.ValidateResponse(ctx, step, value); err != nil {
					return DispatchResult{}, fmt.Errorf("%w: %v", ErrDispatchSchema, err)
				}
			}
			response = &value
		}
	}
	if response == nil {
		return DispatchResult{}, fmt.Errorf("%w: chain produced no response", ErrDispatchTransport)
	}
	if !commit.Finalize() {
		return DispatchResult{}, ErrDispatchAlreadyCommitted
	}
	return DispatchResult{Handled: true, Response: cloneDispatchResponse(*response)}, nil
}

func (d *Dispatcher) authorize(ctx context.Context, plan RouteExecutionPlan, step RouteExecutionStep, request DispatchRequest) error {
	if step.Provider.Kind == ProviderCore {
		return nil
	}
	if d.guard == nil {
		return fmt.Errorf("%w: guard evaluator is unavailable", ErrDispatchDenied)
	}
	if err := d.guard.Authorize(ctx, plan, step, request); err != nil {
		return fmt.Errorf("%w: %v", ErrDispatchDenied, err)
	}
	return nil
}

func (d *Dispatcher) invokePlugin(
	ctx context.Context,
	plan RouteExecutionPlan,
	index int,
	step RouteExecutionStep,
	request DispatchRequest,
	response *DispatchResponse,
	commit *RouteCommitObserver,
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
		Request: cloneDispatchRequest(request), Response: current, Commit: commit,
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
	mu    sync.Mutex
	state RouteExecutionCommitState
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
