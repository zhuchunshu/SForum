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
	if plan.UnsafeMethod() && routeChainHasResponseModifiers(chain) && d.failures == nil {
		// Unsafe response modifiers may fail only after the handler has written.
		// Without the Host-owned audit/quarantine sink, executing the writer would
		// leave retries able to create a second writer with no durable incident.
		return DispatchResult{}, fmt.Errorf("%w: unsafe response modifiers require a failure recorder", ErrDispatchTransport)
	}
	request.Params = plan.Params()
	request.hostMutatedParams = false
	request.Headers = cloneHTTPHeader(request.Headers)
	request.Body = append([]byte(nil), request.Body...)
	request.Permissions = cloneDispatchPermissions(request.Permissions)

	commit := NewRouteCommitObserver()
	var idempotencyLease RouteIdempotencyLease
	preservePending := false
	mutableReplay := routeChainHasMutableRequestFields(chain)
	var replayBinding RouteReplayBinding
	var replayMutations []RouteReplayRequestMutation
	terminal := plan.Terminal()
	if terminal.Provider.Kind == ProviderPlugin {
		policy, policyExists, policyErr := resolvePlanRouteExecutionPolicy(plan, terminal, d.policies)
		if policyErr != nil {
			return DispatchResult{}, fmt.Errorf("%w: %v", ErrDispatchIdempotencyUnavailable, policyErr)
		}
		if policyExists && policy.IdempotencyRequired {
			if terminal.Mode != extensionmanifest.RouteModeHTTP || d.idempotency == nil {
				return DispatchResult{}, ErrDispatchIdempotencyUnavailable
			}
			if routeChainHasCredentialMutableRequestFields(chain) {
				// Credential patches cannot be persisted and replaying the modifier would
				// violate the no-second-execution contract.
				return DispatchResult{}, ErrDispatchIdempotencyUnavailable
			}
			if mutableReplay {
				capability, ok := d.idempotency.(RouteMutationReplayCapability)
				if !ok || !capability.MutationReplayAvailable() {
					return DispatchResult{}, ErrDispatchIdempotencyUnavailable
				}
			}
			replayBinding, err = BuildRouteReplayBinding(plan, request)
			if err != nil {
				return DispatchResult{}, ErrDispatchIdempotencyKeyInvalid
			}
			var replay *RouteIdempotencyReplay
			idempotencyLease, replay, err = d.idempotency.Begin(ctx, plan, terminal, policy, request)
			if err != nil {
				return DispatchResult{}, err
			}
			if replay != nil {
				if mutableReplay && replay.Authorization == nil || !mutableReplay && replay.Authorization != nil {
					return DispatchResult{}, ErrDispatchIdempotencyUnavailable
				}
				validationResponse := cloneDispatchResponse(replay.Response)
				// 这是 Host 在验证完成后返回给客户端的传输证据，不能参与
				// 当前 guard 或响应 Schema 的授权输入。
				validationResponse.Headers.Del(routeIdempotencyReplayedHeader)
				if err := d.authorizeReplay(
					ctx, plan, request, &validationResponse, replay.Authorization, replayBinding,
					replay.ResponseContractKnown, replay.ResponseContract,
				); err != nil {
					return DispatchResult{}, err
				}
				return DispatchResult{Handled: true, Response: cloneDispatchResponse(replay.Response)}, nil
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
	var committingStage InvocationStage
	var committedAfterFailure *RouteCommittedAfterFailure
	responseEligible := make([]bool, len(chain))
	sequence, err := bufferedRouteInvocationSequence(plan)
	if err != nil {
		return DispatchResult{}, err
	}
	var finalResponseContract routeInvocationExecution
	hasFinalResponseContract := false
	var finalResponseCheckpoint *DispatchResponse
dispatchSequence:
	for _, execution := range sequence {
		index, stage := execution.index, execution.stage
		step := chain[index]
		if stage == InvocationStageResponse && pairedResponseStageAction(step.Action) && !responseEligible[index] {
			continue
		}
		started := time.Now()
		mutationBeforeDigest := ""
		authority, err := d.authorize(ctx, plan, index, step, request, response, stage, commit)
		if err != nil {
			if ctxErr := ctx.Err(); ctxErr != nil && errors.Is(err, ctxErr) {
				return DispatchResult{}, ctxErr
			}
			outcome, code, observed := classifyRouteGuardFailure(err)
			d.appendTrace(plan, index, step, stage, outcome, started, commit.State())
			if event := d.committedAfterFailure(plan, index, step, stage, request, response, code, observed); event != nil {
				committedAfterFailure, committingStep, committingStarted = event, index, started
				break dispatchSequence
			}
			return DispatchResult{}, err
		}
		if d.schemas == nil && step.RequestSchema != "" {
			d.appendTrace(plan, index, step, stage, RouteTraceSchemaRejected, started, commit.State())
			if event := d.committedAfterFailure(plan, index, step, stage, request, response, RouteFailureRequestSchemaRejected, false); event != nil {
				committedAfterFailure, committingStep, committingStarted = event, index, started
				break dispatchSequence
			}
			return DispatchResult{}, fmt.Errorf("%w: request validator is unavailable", ErrDispatchSchema)
		}
		if d.schemas != nil && step.RequestSchema != "" {
			if err := d.schemas.ValidateRequest(ctx, step, request); err != nil {
				d.appendTrace(plan, index, step, stage, RouteTraceSchemaRejected, started, commit.State())
				if event := d.committedAfterFailure(plan, index, step, stage, request, response, RouteFailureRequestSchemaRejected, false); event != nil {
					committedAfterFailure, committingStep, committingStarted = event, index, started
					break dispatchSequence
				}
				return DispatchResult{}, fmt.Errorf("%w: %v", ErrDispatchSchema, err)
			}
		}
		if idempotencyLease != nil && mutableReplay && stage == InvocationStageRequest {
			mutationBeforeDigest, err = routeReplayRequestDigest(request)
			if err != nil {
				return DispatchResult{}, ErrDispatchIdempotencyUnavailable
			}
		}
		if stage == InvocationStageResponse {
			if response == nil {
				return DispatchResult{}, fmt.Errorf("%w: response stage has no prior response", ErrDispatchTransport)
			}
			if d.schemas == nil && step.ResponseSchema != "" {
				d.appendTrace(plan, index, step, stage, RouteTraceSchemaRejected, started, commit.State())
				if event := d.committedAfterFailure(plan, index, step, stage, request, response, RouteFailureResponseSchemaRejected, false); event != nil {
					committedAfterFailure, committingStep, committingStarted = event, index, started
					break dispatchSequence
				}
				return DispatchResult{}, fmt.Errorf("%w: response validator is unavailable", ErrDispatchSchema)
			}
			if d.schemas != nil && step.ResponseSchema != "" {
				if err := d.schemas.ValidateResponse(ctx, step, request, *response); err != nil {
					d.appendTrace(plan, index, step, stage, RouteTraceSchemaRejected, started, commit.State())
					if event := d.committedAfterFailure(plan, index, step, stage, request, response, RouteFailureResponseSchemaRejected, false); event != nil {
						committedAfterFailure, committingStep, committingStarted = event, index, started
						break dispatchSequence
					}
					return DispatchResult{}, fmt.Errorf("%w: %v", ErrDispatchSchema, err)
				}
				if hasFinalResponseContract && execution == finalResponseContract {
					value := cloneDispatchResponse(*response)
					finalResponseCheckpoint = &value
				}
			}
		}

		var invocation RouteInvocationResult
		if stage == InvocationStageHandler && step.Provider.Kind == ProviderCore ||
			stage == InvocationStageHandler && (step.Action == extensionmanifest.RouteActionAlias || step.Action == extensionmanifest.RouteActionRewrite) {
			if core == nil {
				d.appendTrace(plan, index, step, stage, RouteTraceTransportFailed, started, commit.State())
				return DispatchResult{}, fmt.Errorf("%w: core handler is unavailable", ErrDispatchTransport)
			}
			value, callErr := core.InvokeCore(ctx, step, request)
			if callErr != nil {
				d.appendTrace(plan, index, step, stage, RouteTraceTransportFailed, started, commit.State())
				return DispatchResult{}, callErr
			}
			invocation.Response = &value
		} else if stage == InvocationStageHandler && step.Action == extensionmanifest.RouteActionRedirect {
			location, locationErr := routeRedirectLocation(step)
			if locationErr != nil {
				return DispatchResult{}, locationErr
			}
			invocation.Response = &DispatchResponse{Status: step.StatusCode, Headers: http.Header{"Location": []string{location}}}
		} else {
			invocation, err = d.invokePlugin(ctx, plan, index, step, stage, request, response, commit, authority)
			if err != nil {
				// Transport errors may arrive after a remote side effect or response began.
				// Advance the fence before evaluating any safe-method fallback.
				if invocation.SideEffectStarted {
					commit.SideEffectStarted()
				}
				if invocation.ResponseStarted {
					commit.ResponseStarted()
				}
				d.appendTrace(plan, index, step, stage, RouteTraceTransportFailed, started, commit.State())
				observed := invocation.SideEffectStarted || invocation.ResponseStarted
				if event := d.committedAfterFailure(plan, index, step, stage, request, response, RouteFailureTransportFailed, observed); event != nil {
					committedAfterFailure, committingStep, committingStarted = event, index, started
					break dispatchSequence
				}
				var fallback *DispatchResponse
				var fallbackErr error
				if stage != InvocationStageResponse {
					fallback, fallbackErr = d.fallback(ctx, plan, index, step, request, core, commit)
				}
				if fallbackErr != nil {
					return DispatchResult{}, fallbackErr
				}
				if fallback != nil {
					response = fallback
					// The fallback owns this response; a plugin contract that never
					// produced it is not an applicable final-response contract.
					hasFinalResponseContract = false
					finalResponseCheckpoint = nil
					if idempotencyLease != nil && mutableReplay && stage == InvocationStageRequest {
						replayMutations, err = appendRouteReplayRequestMutation(replayBinding, replayMutations, RouteReplayRequestMutation{
							StepIndex: index, BeforeDigest: mutationBeforeDigest, AfterDigest: mutationBeforeDigest,
						})
						if err != nil {
							return DispatchResult{}, ErrDispatchIdempotencyUnavailable
						}
					}
					d.appendTrace(plan, index, step, stage, RouteTraceFallbackUsed, started, commit.State())
					committingStep = index
					committingStarted = started
					committingStage = stage
					break dispatchSequence
				}
				if stage == InvocationStageRequest && plan.AllowsFallback(index, commit.State()) {
					if idempotencyLease != nil && mutableReplay {
						replayMutations, err = appendRouteReplayRequestMutation(replayBinding, replayMutations, RouteReplayRequestMutation{
							StepIndex: index, BeforeDigest: mutationBeforeDigest, AfterDigest: mutationBeforeDigest,
						})
						if err != nil {
							return DispatchResult{}, ErrDispatchIdempotencyUnavailable
						}
					}
					d.appendTrace(plan, index, step, stage, RouteTraceFallbackUsed, started, commit.State())
					committingStep = index
					committingStarted = started
					committingStage = stage
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
		if err := validateRouteInvocationResult(stage, invocation); err != nil {
			d.appendTrace(plan, index, step, stage, RouteTraceSchemaRejected, started, commit.State())
			if event := d.committedAfterFailure(plan, index, step, stage, request, response, RouteFailureResponseSchemaRejected, true); event != nil {
				committedAfterFailure, committingStep, committingStarted = event, index, started
				break dispatchSequence
			}
			return DispatchResult{}, err
		}
		switch stage {
		case InvocationStageRequest:
			// Required replay records every request stage, including an empty patch.
			// Run the same post-schema boundary on first execution and replay.
			if len(invocation.RequestPatch) != 0 || idempotencyLease != nil && mutableReplay {
				if planTerminalUsesFrozenPathParams(plan) && routeRequestPatchMutatesParams(invocation.RequestPatch) {
					d.appendTrace(plan, index, step, stage, RouteTraceSchemaRejected, started, commit.State())
					return DispatchResult{}, fmt.Errorf("%w: core-bound route params cannot be mutated after route selection", ErrDispatchSchema)
				}
				value, patchErr := applyRouteRequestPatch(
					request, invocation.RequestPatch, step.MutableRequestFields, rawMutationAuthority(authority),
				)
				if patchErr != nil {
					d.appendTrace(plan, index, step, stage, RouteTraceSchemaRejected, started, commit.State())
					return DispatchResult{}, fmt.Errorf("%w: %v", ErrDispatchSchema, patchErr)
				}
				if routeRequestPatchMutatesParams(invocation.RequestPatch) {
					value.hostMutatedParams = true
				}
				if d.schemas == nil && step.RequestSchema != "" {
					return DispatchResult{}, fmt.Errorf("%w: request validator is unavailable", ErrDispatchSchema)
				}
				if d.schemas != nil && step.RequestSchema != "" {
					if err := d.schemas.ValidateRequest(ctx, step, value); err != nil {
						d.appendTrace(plan, index, step, stage, RouteTraceSchemaRejected, started, commit.State())
						return DispatchResult{}, fmt.Errorf("%w: %v", ErrDispatchSchema, err)
					}
				}
				request = value
			}
			if idempotencyLease != nil && mutableReplay {
				afterDigest, digestErr := routeReplayRequestDigest(request)
				if digestErr != nil {
					return DispatchResult{}, ErrDispatchIdempotencyUnavailable
				}
				replayMutations, err = appendRouteReplayRequestMutation(replayBinding, replayMutations, RouteReplayRequestMutation{
					StepIndex: index, BeforeDigest: mutationBeforeDigest, AfterDigest: afterDigest,
					Operations: cloneRoutePatchOperations(invocation.RequestPatch),
				})
				if err != nil {
					return DispatchResult{}, ErrDispatchIdempotencyUnavailable
				}
			}
		case InvocationStageHandler:
			value := cloneDispatchResponse(*invocation.Response)
			if d.schemas == nil && step.ResponseSchema != "" {
				d.appendTrace(plan, index, step, stage, RouteTraceSchemaRejected, started, commit.State())
				if event := d.committedAfterFailure(plan, index, step, stage, request, response, RouteFailureResponseSchemaRejected, true); event != nil {
					committedAfterFailure, committingStep, committingStarted = event, index, started
					break dispatchSequence
				}
				return DispatchResult{}, fmt.Errorf("%w: response validator is unavailable", ErrDispatchSchema)
			}
			if d.schemas != nil && step.ResponseSchema != "" {
				if err := d.schemas.ValidateResponse(ctx, step, request, value); err != nil {
					d.appendTrace(plan, index, step, stage, RouteTraceSchemaRejected, started, commit.State())
					if event := d.committedAfterFailure(plan, index, step, stage, request, response, RouteFailureResponseSchemaRejected, true); event != nil {
						committedAfterFailure, committingStep, committingStarted = event, index, started
						break dispatchSequence
					}
					return DispatchResult{}, fmt.Errorf("%w: %v", ErrDispatchSchema, err)
				}
			}
			response = &value
		case InvocationStageResponse:
			if len(invocation.ResponsePatch) == 0 {
				break
			}
			value, patchErr := applyRouteResponsePatch(*response, invocation.ResponsePatch, step.MutableResponseFields)
			if patchErr != nil {
				d.appendTrace(plan, index, step, stage, RouteTraceSchemaRejected, started, commit.State())
				if event := d.committedAfterFailure(plan, index, step, stage, request, response, RouteFailureResponseSchemaRejected, true); event != nil {
					committedAfterFailure, committingStep, committingStarted = event, index, started
					break dispatchSequence
				}
				return DispatchResult{}, fmt.Errorf("%w: %v", ErrDispatchSchema, patchErr)
			}
			if d.schemas == nil && step.ResponseSchema != "" {
				return DispatchResult{}, fmt.Errorf("%w: response validator is unavailable", ErrDispatchSchema)
			}
			if d.schemas != nil && step.ResponseSchema != "" {
				if err := d.schemas.ValidateResponse(ctx, step, request, value); err != nil {
					d.appendTrace(plan, index, step, stage, RouteTraceSchemaRejected, started, commit.State())
					if event := d.committedAfterFailure(plan, index, step, stage, request, response, RouteFailureResponseSchemaRejected, true); event != nil {
						committedAfterFailure, committingStep, committingStarted = event, index, started
						break dispatchSequence
					}
					return DispatchResult{}, fmt.Errorf("%w: %v", ErrDispatchSchema, err)
				}
			}
			response = &value
		}
		if stage != InvocationStageRequest && strings.TrimSpace(step.ResponseSchema) != "" && response != nil {
			finalResponseContract = execution
			hasFinalResponseContract = true
			value := cloneDispatchResponse(*response)
			finalResponseCheckpoint = &value
		}
		if stage == InvocationStageRequest && pairedResponseStageAction(step.Action) {
			responseEligible[index] = true
		}
		if step.Provider.Kind == ProviderPlugin {
			d.appendTrace(plan, index, step, stage, RouteTraceSucceeded, started, commit.State())
			committingStep = index
			committingStarted = started
			committingStage = stage
		}
	}
	if response == nil {
		return DispatchResult{}, fmt.Errorf("%w: chain produced no response", ErrDispatchTransport)
	}
	if hasFinalResponseContract && committedAfterFailure == nil {
		if d.schemas == nil {
			return DispatchResult{}, fmt.Errorf("%w: response validator is unavailable", ErrDispatchSchema)
		}
		contractStep := chain[finalResponseContract.index]
		if err := d.schemas.ValidateResponse(ctx, contractStep, request, *response); err != nil {
			// A schema-less response modifier may only preserve the most recent
			// declared response contract. Unsafe routes retain the last response
			// that passed that contract and record the exact failing modifier.
			if finalResponseCheckpoint != nil && committingStep >= 0 {
				checkpoint := cloneDispatchResponse(*finalResponseCheckpoint)
				d.appendTrace(plan, committingStep, chain[committingStep], committingStage, RouteTraceSchemaRejected, committingStarted, commit.State())
				if event := d.committedAfterFailure(
					plan, committingStep, chain[committingStep], committingStage, request,
					&checkpoint, RouteFailureResponseSchemaRejected, true,
				); event != nil {
					committedAfterFailure = event
					response = &checkpoint
				} else {
					return DispatchResult{}, fmt.Errorf("%w: %v", ErrDispatchSchema, err)
				}
			} else {
				return DispatchResult{}, fmt.Errorf("%w: %v", ErrDispatchSchema, err)
			}
		}
	}
	switch terminal.Action {
	case extensionmanifest.RouteActionAlias:
		response.CanonicalPath = terminal.TargetPath
	case extensionmanifest.RouteActionRedirect:
		location, locationErr := routeRedirectLocation(terminal)
		if locationErr != nil {
			return DispatchResult{}, locationErr
		}
		response.Status = terminal.StatusCode
		if response.Headers == nil {
			response.Headers = make(http.Header)
		}
		response.Headers.Set("Location", location)
	case extensionmanifest.RouteActionRewrite:
		response.CanonicalPath = plan.Path()
	}
	if !commit.Finalize() {
		return DispatchResult{}, ErrDispatchAlreadyCommitted
	}
	if committingStep >= 0 {
		if committedAfterFailure != nil {
			committingStage = committedAfterFailure.InvocationStage
		}
		d.appendTrace(plan, committingStep, chain[committingStep], committingStage, RouteTraceCommitted, committingStarted, commit.State())
	}
	if committedAfterFailure != nil {
		committedAfterFailure.CommitState = commit.State()
		d.failures.RecordCommittedAfterFailure(ctx, *committedAfterFailure)
	}
	if idempotencyLease != nil && response.Status >= http.StatusOK && response.Status < http.StatusMultipleChoices {
		// Complete 失败时保留 pending；客户端只能得到 fail-closed unavailable，
		// 不能在未知持久化结果后再次执行插件副作用。
		preservePending = true
		completion := RouteIdempotencyCompletion{
			Response: cloneDispatchResponse(*response), ResponseContractKnown: true,
		}
		if hasFinalResponseContract {
			completion.ResponseContract = newRouteReplayResponseContract(
				finalResponseContract, chain[finalResponseContract.index],
			)
		}
		if mutableReplay {
			completion.Authorization, err = newRouteReplayAuthorization(replayBinding, replayMutations)
			if err != nil || completion.Authorization == nil ||
				!routeReplayAuthorizationMatchesPlan(completion.Authorization, replayBinding, sequence) {
				return DispatchResult{}, ErrDispatchIdempotencyUnavailable
			}
		}
		if err := idempotencyLease.Complete(ctx, completion); err != nil {
			return DispatchResult{}, fmt.Errorf("%w: %v", ErrDispatchIdempotencyUnavailable, err)
		}
	}
	return DispatchResult{Handled: true, Response: cloneDispatchResponse(*response)}, nil
}

func routeRedirectLocation(step RouteExecutionStep) (string, error) {
	location := step.Destination
	if step.TargetID != "" {
		location = step.TargetPath
	}
	if location == "" || !strings.HasPrefix(location, "/") || strings.HasPrefix(location, "//") || strings.ContainsAny(location, "?#\r\n") {
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
			return routeInvocationAuthority{}, ctxErr
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
