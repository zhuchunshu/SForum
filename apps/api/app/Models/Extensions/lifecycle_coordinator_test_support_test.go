package extensions

import (
	"context"
	"fmt"
	"strings"
	"sync"
)

type lifecycleCoordinatorTestBehavior struct {
	progress       []LifecycleCoordinatorActionProgress
	result         LifecycleCoordinatorActionResult
	err            error
	after          func()
	beforeProgress func()
	cancel         context.CancelFunc
	waitForContext bool
}

type lifecycleCoordinatorTestRuntime struct {
	mu        sync.Mutex
	requests  []LifecycleCoordinatorActionRequest
	behaviors map[LifecycleMachineAction][]lifecycleCoordinatorTestBehavior
	events    *lifecycleCoordinatorTestEvents
}

func (r *lifecycleCoordinatorTestRuntime) RunLifecycleAction(
	ctx context.Context,
	request LifecycleCoordinatorActionRequest,
	onProgress func(LifecycleCoordinatorActionProgress) error,
) (LifecycleCoordinatorActionResult, error) {
	r.mu.Lock()
	r.requests = append(r.requests, request)
	behavior := lifecycleCoordinatorTestBehavior{result: LifecycleCoordinatorActionResult{Status: LifecycleStepSucceeded}}
	if queue := r.behaviors[request.Action]; len(queue) > 0 {
		behavior = queue[0]
		r.behaviors[request.Action] = queue[1:]
		if behavior.result.Status == "" && behavior.err == nil {
			behavior.result.Status = LifecycleStepSucceeded
		}
	}
	r.mu.Unlock()
	if r.events != nil {
		r.events.add("action:" + string(request.Action))
	}
	if behavior.beforeProgress != nil {
		behavior.beforeProgress()
	}
	for _, progress := range behavior.progress {
		if err := onProgress(progress); err != nil {
			return LifecycleCoordinatorActionResult{}, err
		}
	}
	if behavior.cancel != nil {
		behavior.cancel()
	}
	if behavior.waitForContext {
		<-ctx.Done()
		behavior.err = ctx.Err()
	}
	if behavior.after != nil {
		behavior.after()
	}
	return behavior.result, behavior.err
}

func (r *lifecycleCoordinatorTestRuntime) actionNames() []LifecycleMachineAction {
	r.mu.Lock()
	defer r.mu.Unlock()
	result := make([]LifecycleMachineAction, 0, len(r.requests))
	for _, request := range r.requests {
		result = append(result, request.Action)
	}
	return result
}

func (r *lifecycleCoordinatorTestRuntime) requestsSnapshot() []LifecycleCoordinatorActionRequest {
	r.mu.Lock()
	defer r.mu.Unlock()
	return append([]LifecycleCoordinatorActionRequest(nil), r.requests...)
}

type lifecycleCoordinatorTestHost struct {
	mu        sync.Mutex
	requests  []LifecycleCoordinatorGateRequest
	failState LifecycleMachineState
	events    *lifecycleCoordinatorTestEvents
	cancel    context.CancelFunc
}

func (h *lifecycleCoordinatorTestHost) RunLifecycleHostGate(_ context.Context, request LifecycleCoordinatorGateRequest) error {
	h.mu.Lock()
	h.requests = append(h.requests, request)
	fail := h.failState == request.State
	if fail {
		h.failState = ""
	}
	cancel := h.cancel
	h.cancel = nil
	h.mu.Unlock()
	if h.events != nil {
		h.events.add("host:" + string(request.State))
	}
	if cancel != nil {
		cancel()
	}
	if fail {
		return fmt.Errorf("host gate %s failed", request.State)
	}
	return nil
}

func (h *lifecycleCoordinatorTestHost) gateIDs() []string {
	h.mu.Lock()
	defer h.mu.Unlock()
	result := make([]string, 0, len(h.requests))
	for _, request := range h.requests {
		result = append(result, request.StepID)
	}
	return result
}

type lifecycleCoordinatorTestEvents struct {
	mu     sync.Mutex
	values []string
}

func (e *lifecycleCoordinatorTestEvents) add(value string) {
	e.mu.Lock()
	e.values = append(e.values, value)
	e.mu.Unlock()
}

func (e *lifecycleCoordinatorTestEvents) snapshot() []string {
	e.mu.Lock()
	defer e.mu.Unlock()
	return append([]string(nil), e.values...)
}

func lifecycleCoordinatorTestInput(operation LifecycleMachineOperation, forced bool) LifecycleCoordinatorRunInput {
	digest := strings.Repeat("a", 64)
	extension := Extension{
		ID: "demo.plugin", Version: "1.0.0", Type: TypePlugin,
		Status: StatusEnabled, PackageDigest: digest,
	}
	return LifecycleCoordinatorRunInput{
		Extension: extension,
		Acquire: AcquireLifecycleOperationInput{
			ExtensionID: extension.ID, ExtensionVersion: extension.Version, PackageDigest: digest,
			Operation: string(operation), PlanVersion: "demo.plugin.lifecycle@1",
			IdempotencyKey: "operation-1", RequestFingerprint: strings.Repeat("f", 64),
			AuthorityType: LifecycleAuthorityBuiltin, Forced: forced,
		},
	}
}

func cloneLifecycleCoordinatorTestOperation(value LifecycleOperation) LifecycleOperation {
	value.Progress = cloneLifecycleJSON(value.Progress)
	value.Checkpoint = cloneLifecycleJSON(value.Checkpoint)
	value.ResultDocument = cloneLifecycleJSON(value.ResultDocument)
	return value
}

func countLifecycleCoordinatorAction(values []LifecycleMachineAction, action LifecycleMachineAction) int {
	count := 0
	for _, value := range values {
		if value == action {
			count++
		}
	}
	return count
}
