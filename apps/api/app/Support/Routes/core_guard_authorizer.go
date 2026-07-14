package routes

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"

	extensionmanifest "github.com/zhuchunshu/sforum/apps/api/app/Support/ExtensionManifest"
)

const CoreGuardEvaluatorContractV1 = "sforum.route.guard_evaluator@1"

var (
	ErrCoreGuardRegistryInvalid      = errors.New("routes: core guard evaluator registry is invalid")
	ErrCoreGuardEvaluatorUnavailable = errors.New("routes: core guard evaluator is unavailable")
	ErrCoreGuardLoginRequired        = errors.New("routes: core guard login required")
	ErrCoreGuardGuestRequired        = errors.New("routes: core guard guest required")
	ErrCoreGuardPermissionDenied     = errors.New("routes: core guard permission denied")
)

type CoreGuardEvaluation struct {
	Descriptor    CoreGuardDescriptor
	PlanRevision  uint64
	RequestMethod string
	RequestPath   string
	Step          RouteExecutionStep
	Request       DispatchRequest
}

type CoreGuardEvaluator interface {
	EvaluateCoreGuard(context.Context, CoreGuardEvaluation) error
}

type CoreGuardEvaluatorFunc func(context.Context, CoreGuardEvaluation) error

func (f CoreGuardEvaluatorFunc) EvaluateCoreGuard(ctx context.Context, evaluation CoreGuardEvaluation) error {
	if f == nil {
		return ErrCoreGuardEvaluatorUnavailable
	}
	return f(ctx, evaluation)
}

type CoreGuardEvaluatorRegistration struct {
	EvaluatorID     string
	ContractVersion string
	Evaluator       CoreGuardEvaluator
}

type CoreGuardEvaluatorBinding struct {
	EvaluatorID     string
	ContractVersion string
}

type coreGuardEvaluatorEntry struct {
	contractVersion string
	evaluator       CoreGuardEvaluator
}

// CoreGuardEvaluatorRegistry is immutable after construction. Contextual Host
// policy is executable only when its exact evaluator id is registered here.
type CoreGuardEvaluatorRegistry struct {
	entries map[string]coreGuardEvaluatorEntry
}

func NewCoreGuardEvaluatorRegistry(registrations []CoreGuardEvaluatorRegistration) (*CoreGuardEvaluatorRegistry, error) {
	registry := &CoreGuardEvaluatorRegistry{entries: make(map[string]coreGuardEvaluatorEntry, len(registrations))}
	for _, registration := range registrations {
		if !strings.HasPrefix(registration.EvaluatorID, "core.guard.") ||
			!routeIDPattern.MatchString(registration.EvaluatorID) ||
			registration.ContractVersion != CoreGuardEvaluatorContractV1 ||
			registration.Evaluator == nil {
			return nil, fmt.Errorf("%w: invalid evaluator registration", ErrCoreGuardRegistryInvalid)
		}
		if _, duplicate := registry.entries[registration.EvaluatorID]; duplicate {
			return nil, fmt.Errorf("%w: duplicate evaluator %q", ErrCoreGuardRegistryInvalid, registration.EvaluatorID)
		}
		registry.entries[registration.EvaluatorID] = coreGuardEvaluatorEntry{
			contractVersion: registration.ContractVersion,
			evaluator:       registration.Evaluator,
		}
	}
	return registry, nil
}

func MustNewCoreGuardEvaluatorRegistry(registrations []CoreGuardEvaluatorRegistration) *CoreGuardEvaluatorRegistry {
	registry, err := NewCoreGuardEvaluatorRegistry(registrations)
	if err != nil {
		panic(err)
	}
	return registry
}

func (r *CoreGuardEvaluatorRegistry) Bindings() []CoreGuardEvaluatorBinding {
	if r == nil {
		return nil
	}
	bindings := make([]CoreGuardEvaluatorBinding, 0, len(r.entries))
	for id, entry := range r.entries {
		bindings = append(bindings, CoreGuardEvaluatorBinding{EvaluatorID: id, ContractVersion: entry.contractVersion})
	}
	sort.Slice(bindings, func(i, j int) bool { return bindings[i].EvaluatorID < bindings[j].EvaluatorID })
	return bindings
}

func (r *CoreGuardEvaluatorRegistry) evaluate(ctx context.Context, evaluation CoreGuardEvaluation) error {
	if r == nil || ctx == nil {
		return ErrCoreGuardEvaluatorUnavailable
	}
	entry, exists := r.entries[evaluation.Descriptor.EvaluatorID]
	if !exists || entry.contractVersion != CoreGuardEvaluatorContractV1 || entry.evaluator == nil {
		return ErrCoreGuardEvaluatorUnavailable
	}
	evaluation.Descriptor = cloneCoreGuardDescriptor(evaluation.Descriptor)
	evaluation.Step.CoreGuard = cloneCoreGuardDescriptor(evaluation.Step.CoreGuard)
	evaluation.Request = cloneDispatchRequest(evaluation.Request)
	return entry.evaluator.EvaluateCoreGuard(ctx, evaluation)
}

type CoreGuardAuthorizer struct {
	Evaluators *CoreGuardEvaluatorRegistry
}

func (a CoreGuardAuthorizer) Authorize(
	ctx context.Context,
	plan RouteExecutionPlan,
	step RouteExecutionStep,
	request DispatchRequest,
) error {
	if ctx == nil {
		return ErrCoreGuardEvaluatorUnavailable
	}
	if !validPluginCoreGuardEvaluation(plan, step, request) {
		return ErrCoreGuardEvaluatorUnavailable
	}
	switch step.Guard {
	case extensionmanifest.GuardCorePublic:
		return nil
	case extensionmanifest.GuardCoreLogin:
		return requireCoreGuardLogin(request)
	case extensionmanifest.GuardCoreGuest:
		if request.Authenticated || request.ActorID > 0 {
			return ErrCoreGuardGuestRequired
		}
		return nil
	case extensionmanifest.GuardCorePermission:
		return requireAnyCoreGuardPermission(request, []string{step.Permission})
	case extensionmanifest.GuardCoreInherit:
		return a.authorizeInherited(ctx, plan, step, request)
	default:
		// Custom and raw-request guards require their separately confirmed runtime.
		return ErrCoreGuardEvaluatorUnavailable
	}
}

func (a CoreGuardAuthorizer) authorizeInherited(
	ctx context.Context,
	plan RouteExecutionPlan,
	step RouteExecutionStep,
	request DispatchRequest,
) error {
	descriptor := step.CoreGuard
	if !validInheritedCoreGuardEvaluation(plan, step, request, descriptor) {
		return ErrCoreGuardEvaluatorUnavailable
	}
	switch descriptor.Kind {
	case CoreGuardPublic:
		return nil
	case CoreGuardLogin:
		return requireCoreGuardLogin(request)
	case CoreGuardGuest:
		if request.Authenticated || request.ActorID > 0 {
			return ErrCoreGuardGuestRequired
		}
		return nil
	case CoreGuardSuperAdmin:
		if err := requireCoreGuardLogin(request); err != nil {
			return err
		}
		if request.Permissions["*"] {
			return nil
		}
		return ErrCoreGuardPermissionDenied
	case CoreGuardPermissionAny:
		return requireAnyCoreGuardPermission(request, descriptor.Permissions)
	case CoreGuardContextual:
		return a.Evaluators.evaluate(ctx, CoreGuardEvaluation{
			Descriptor: descriptor, PlanRevision: plan.Revision(), RequestMethod: plan.Method(),
			RequestPath: plan.Path(), Step: step, Request: request,
		})
	default:
		return ErrCoreGuardEvaluatorUnavailable
	}
}

func validInheritedCoreGuardEvaluation(
	plan RouteExecutionPlan,
	step RouteExecutionStep,
	request DispatchRequest,
	descriptor CoreGuardDescriptor,
) bool {
	if step.TargetID == "" || step.TargetID != descriptor.RouteID {
		return false
	}
	if err := validateCoreGuardDescriptor(CoreRoute{
		ID: descriptor.RouteID, ContractVersion: descriptor.ContractVersion,
		Method: descriptor.Method, Guard: descriptor,
	}); err != nil {
		return false
	}
	return routeMethodMatches(Route{Method: descriptor.Method, Mode: extensionmanifest.RouteModeHTTP}, plan.Method())
}

func validPluginCoreGuardEvaluation(plan RouteExecutionPlan, step RouteExecutionStep, request DispatchRequest) bool {
	return plan.Valid() && step.Provider.Kind == ProviderPlugin && request.Method == plan.Method() &&
		request.Path == plan.Path() && planContainsCoreGuardStep(plan, step)
}

func planContainsCoreGuardStep(plan RouteExecutionPlan, wanted RouteExecutionStep) bool {
	for _, step := range plan.Chain() {
		if equalCoreGuardExecutionStep(step, wanted) {
			return true
		}
	}
	return false
}

func equalCoreGuardExecutionStep(left, right RouteExecutionStep) bool {
	return left.Phase == right.Phase && left.Action == right.Action && left.RouteID == right.RouteID &&
		left.ContractVersion == right.ContractVersion && left.TargetID == right.TargetID && left.Path == right.Path &&
		left.Method == right.Method && left.Provider == right.Provider && left.Guard == right.Guard &&
		left.Access == right.Access && left.Permission == right.Permission && left.Mode == right.Mode &&
		left.Destination == right.Destination && left.Handler == right.Handler &&
		left.RequestSchema == right.RequestSchema && left.ResponseSchema == right.ResponseSchema &&
		left.TimeoutMS == right.TimeoutMS && left.Fallback == right.Fallback && left.Priority == right.Priority &&
		equalCoreGuardDescriptor(left.CoreGuard, right.CoreGuard) &&
		equalPluginGuardBinding(left.PluginGuard, right.PluginGuard)
}

func requireCoreGuardLogin(request DispatchRequest) error {
	if request.Authenticated && request.ActorID > 0 {
		return nil
	}
	return ErrCoreGuardLoginRequired
}

func requireAnyCoreGuardPermission(request DispatchRequest, permissions []string) error {
	if err := requireCoreGuardLogin(request); err != nil {
		return err
	}
	if request.Permissions["*"] {
		return nil
	}
	for _, permission := range permissions {
		if permission != "" && request.Permissions[permission] {
			return nil
		}
	}
	return ErrCoreGuardPermissionDenied
}
