package extensions

import (
	"context"
	"encoding/json"
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
	mu         sync.Mutex
	requests   []LifecycleCoordinatorGateRequest
	results    map[string][]LifecycleCoordinatorGateResult
	gateErrors map[string][]error
	afterStep  map[string][]func()
	failState  LifecycleMachineState
	events     *lifecycleCoordinatorTestEvents
	cancel     context.CancelFunc
}

func (h *lifecycleCoordinatorTestHost) RunLifecycleHostGate(_ context.Context, request LifecycleCoordinatorGateRequest) (LifecycleCoordinatorGateResult, error) {
	h.mu.Lock()
	h.requests = append(h.requests, request)
	result := LifecycleCoordinatorGateResult{}
	resultSet := false
	if queue := h.results[request.StepID]; len(queue) > 0 {
		result = queue[0]
		h.results[request.StepID] = queue[1:]
		resultSet = true
	}
	if !resultSet {
		result = lifecycleCoordinatorTestPlannedGateResult(request)
	}
	var gateErr error
	if queue := h.gateErrors[request.StepID]; len(queue) > 0 {
		gateErr = queue[0]
		h.gateErrors[request.StepID] = queue[1:]
	}
	var after func()
	if queue := h.afterStep[request.StepID]; len(queue) > 0 {
		after = queue[0]
		h.afterStep[request.StepID] = queue[1:]
	}
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
	if after != nil {
		after()
	}
	if fail {
		return LifecycleCoordinatorGateResult{}, fmt.Errorf("host gate %s failed", request.State)
	}
	if gateErr != nil {
		return LifecycleCoordinatorGateResult{}, gateErr
	}
	return result, nil
}

func lifecycleCoordinatorTestPlannedGateResult(request LifecycleCoordinatorGateRequest) LifecycleCoordinatorGateResult {
	if request.Position != 0 || request.State != LifecycleMachinePlanned {
		return LifecycleCoordinatorGateResult{}
	}
	result := LifecycleCoordinatorGateResult{RevalidationPolicy: LifecycleGateRevalidationRequired}
	result.SourceBinding = request.SourceBinding
	result.TargetBinding = request.TargetBinding
	if result.TargetBinding.RuntimeInstanceID == "" {
		result.TargetBinding.RuntimeInstanceID = "test-target-instance"
	}
	if !lifecycleRuntimeBindingEmpty(result.SourceBinding) && result.SourceBinding.RuntimeInstanceID == "" {
		result.SourceBinding.RuntimeInstanceID = "test-source-instance"
	}
	return result
}

func (h *lifecycleCoordinatorTestHost) requestsSnapshot() []LifecycleCoordinatorGateRequest {
	h.mu.Lock()
	defer h.mu.Unlock()
	return append([]LifecycleCoordinatorGateRequest(nil), h.requests...)
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
		Manifest: Manifest{
			ID: "demo.plugin", Version: "1.0.0", Type: TypePlugin,
			Lifecycle: &ManifestLifecycle{ContractVersion: "demo.plugin.lifecycle@1"},
		},
	}
	var source *Extension
	if operation != LifecycleMachineInstall {
		value := extension
		if operation == LifecycleMachineUpgrade || operation == LifecycleMachineRollback {
			value.Version = "0.9.0"
			value.PackageDigest = strings.Repeat("b", 64)
			value.Manifest.Version = value.Version
			value.Manifest.Lifecycle = &ManifestLifecycle{ContractVersion: "demo.plugin.lifecycle@0"}
		}
		source = &value
	}
	return LifecycleCoordinatorRunInput{
		Extension: extension, SourceExtension: source,
		Acquire: AcquireLifecycleOperationInput{
			ExtensionID: extension.ID, ExtensionVersion: extension.Version, PackageDigest: digest,
			Operation: string(operation), PlanVersion: "demo.plugin.lifecycle@1",
			IdempotencyKey: "operation-1", RequestFingerprint: strings.Repeat("f", 64),
			AuthorityType: LifecycleAuthorityBuiltin, AuthoritySnapshot: json.RawMessage(`{"schemaVersion":"test"}`),
			RequestedByUserID: 42, AuditEventID: 9001, Forced: forced,
		},
	}
}

func lifecycleCoordinatorRetry(input *LifecycleCoordinatorRunInput) {
	input.Retry = true
	input.Acquire.ExistingOnly = true
	input.RecoveryActorUserID = 84
	input.RecoveryAuditEventID = 9002
}

func lifecycleCoordinatorRecoveryReplay(input *LifecycleCoordinatorRunInput) {
	input.Retry = false
	input.Acquire.ExistingOnly = false
	input.RecoveryActorUserID = 0
	input.RecoveryAuditEventID = 0
	input.EscalateForced = false
	input.RecoveryReason = ""
	input.SkipFailedStep = false
	input.SkipReason = ""
}

func cloneLifecycleCoordinatorTestOperation(value LifecycleOperation) LifecycleOperation {
	value.ArtifactDigests = cloneLifecycleJSON(value.ArtifactDigests)
	value.AuthoritySnapshot = cloneLifecycleJSON(value.AuthoritySnapshot)
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

func countLifecycleCoordinatorString(values []string, target string) int {
	count := 0
	for _, value := range values {
		if value == target {
			count++
		}
	}
	return count
}
