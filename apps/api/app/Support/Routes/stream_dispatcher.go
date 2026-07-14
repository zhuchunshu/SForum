package routes

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	extensionmanifest "github.com/zhuchunshu/sforum/apps/api/app/Support/ExtensionManifest"
)

// RouteStreamDispatch is the Host-owned lifetime fence for one non-buffered
// terminal plugin route. Transport adapters report only observed milestones;
// they cannot reset commit state or manufacture fallback eligibility.
type RouteStreamDispatch struct {
	dispatcher *Dispatcher
	plan       RouteExecutionPlan
	index      int
	step       RouteExecutionStep
	request    DispatchRequest
	commit     *RouteCommitObserver
	started    time.Time

	mu              sync.Mutex
	requestStarted  bool
	responseStarted bool
	finished        bool
}

type RouteStreamPreparation struct {
	Handled  bool
	Dispatch *RouteStreamDispatch
}

// PrepareStream selects and authorizes a non-buffered plan without reading the
// request body. Composed stream middleware remains closed until its mutation
// and after-response failure semantics are frozen and implemented.
func (d *Dispatcher) PrepareStream(ctx context.Context, request DispatchRequest) (RouteStreamPreparation, error) {
	if d == nil || d.plans == nil || ctx == nil {
		return RouteStreamPreparation{}, ErrDispatchInvalid
	}
	request.Method = strings.ToUpper(strings.TrimSpace(request.Method))
	if request.Method == "" || strings.TrimSpace(request.Path) == "" {
		return RouteStreamPreparation{}, ErrDispatchInvalid
	}
	plan, err := d.plans.BuildExecutionPlan(ctx, request.Method, request.Path)
	if err != nil {
		if errors.Is(err, ErrRouteNotFound) {
			return RouteStreamPreparation{}, nil
		}
		return RouteStreamPreparation{}, err
	}
	if !plan.Valid() {
		return RouteStreamPreparation{}, ErrInvalidExecutionPlan
	}
	chain := plan.Chain()
	if !dispatchPlanHasPluginStep(chain) {
		return RouteStreamPreparation{}, nil
	}
	terminal := chain[len(chain)-1]
	if terminal.Mode == extensionmanifest.RouteModeHTTP {
		return RouteStreamPreparation{}, nil
	}
	if terminal.Provider.Kind != ProviderPlugin || terminal.Phase != RoutePhaseHandler {
		return RouteStreamPreparation{}, fmt.Errorf("%w: stream terminal is not an exact plugin handler", ErrDispatchTransport)
	}
	if len(chain) != 1 {
		return RouteStreamPreparation{}, fmt.Errorf("%w: composed stream chains are not available", ErrDispatchTransport)
	}
	request.Params = plan.Params()
	request.Headers = cloneHTTPHeader(request.Headers)
	request.Body = nil
	request.Permissions = cloneDispatchPermissions(request.Permissions)
	if err := d.authorize(ctx, plan, terminal, request); err != nil {
		d.appendTrace(plan, 0, terminal, RouteTraceDenied, time.Now(), RouteCommitPristine)
		return RouteStreamPreparation{}, err
	}
	return RouteStreamPreparation{Handled: true, Dispatch: &RouteStreamDispatch{
		dispatcher: d, plan: plan, step: terminal, request: request,
		commit: NewRouteCommitObserver(), started: time.Now(),
	}}, nil
}

func (d *RouteStreamDispatch) Step() RouteExecutionStep {
	if d == nil {
		return RouteExecutionStep{}
	}
	return d.step
}

func (d *RouteStreamDispatch) Request() DispatchRequest {
	if d == nil {
		return DispatchRequest{}
	}
	return cloneDispatchRequest(d.request)
}

func (d *RouteStreamDispatch) RequestStarted() {
	if d == nil {
		return
	}
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.finished || d.requestStarted {
		return
	}
	d.requestStarted = true
	d.commit.SideEffectStarted()
}

func (d *RouteStreamDispatch) ResponseStarted() {
	if d == nil {
		return
	}
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.finished || d.responseStarted {
		return
	}
	d.responseStarted = true
	d.commit.ResponseStarted()
	d.dispatcher.appendTrace(d.plan, d.index, d.step, RouteTraceSucceeded, d.started, d.commit.State())
}

func (d *RouteStreamDispatch) Fail() {
	if d == nil {
		return
	}
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.finished {
		return
	}
	d.finished = true
	d.dispatcher.appendTrace(d.plan, d.index, d.step, RouteTraceTransportFailed, d.started, d.commit.State())
}

func (d *RouteStreamDispatch) Complete() error {
	if d == nil {
		return ErrDispatchInvalid
	}
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.finished || !d.responseStarted || !d.commit.Finalize() {
		return ErrDispatchAlreadyCommitted
	}
	d.finished = true
	d.dispatcher.appendTrace(d.plan, d.index, d.step, RouteTraceCommitted, d.started, d.commit.State())
	return nil
}
