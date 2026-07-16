package contentregistry

import (
	"fmt"
	"sort"
)

type plannedBinding struct {
	binding      ExecutionBinding
	contribution Contribution
}

type executionPlan struct {
	state     *registryState
	target    Contribution
	base      plannedBinding
	terminal  *plannedBinding
	before    []plannedBinding
	after     []plannedBinding
	wrap      []plannedBinding
	filters   []plannedBinding
	conflicts []plannedBinding
	stale     []ExecutionBinding
	hidden    bool
}

func (e *Executor) buildPlan(request ExecutionRequest) (executionPlan, error) {
	return e.buildPlanFromState(e.registry.load(), request)
}

func (e *Executor) buildPlanFromState(state *registryState, request ExecutionRequest) (executionPlan, error) {
	target, found := state.content[request.TargetID]
	if !found {
		return executionPlan{}, ErrNotFound
	}
	if target.ContractVersion != request.ContractVersion {
		return executionPlan{}, ErrContractStale
	}
	plan := executionPlan{state: state, target: cloneContribution(target)}
	var bases, terminals []plannedBinding
	for _, binding := range e.bindings {
		if binding.TargetID != request.TargetID {
			continue
		}
		if binding.TargetContractVersion != target.ContractVersion {
			plan.stale = append(plan.stale, binding)
			continue
		}
		contribution, active := state.content[binding.DeclarationID]
		if !active || contribution.ContractVersion != binding.ContractVersion || contribution.Artifact != binding.Artifact {
			plan.stale = append(plan.stale, binding)
			continue
		}
		if err := validateBindingContribution(binding, contribution); err != nil {
			return executionPlan{}, err
		}
		step := plannedBinding{binding: binding, contribution: cloneContribution(contribution)}
		switch binding.Action {
		case ActionAdd:
			bases = append(bases, step)
		case ActionBefore:
			plan.before = append(plan.before, step)
		case ActionAfter:
			plan.after = append(plan.after, step)
		case ActionWrap:
			plan.wrap = append(plan.wrap, step)
		case ActionFilter:
			plan.filters = append(plan.filters, step)
		case ActionReplace, ActionHide:
			terminals = append(terminals, step)
		}
	}
	if len(bases) != 1 || bases[0].binding.DeclarationID != request.TargetID ||
		bases[0].contribution.Artifact != target.Artifact || bases[0].contribution.ContractVersion != target.ContractVersion {
		return executionPlan{}, fmt.Errorf("%w: target %s requires one exact add binding", ErrCompositionInvalid, request.TargetID)
	}
	plan.base = bases[0]
	sortPlannedBindings(plan.before)
	sortPlannedBindings(plan.after)
	sortPlannedBindings(plan.wrap)
	sortPlannedBindings(plan.filters)
	sortPlannedBindings(terminals)
	if len(terminals) > 0 {
		selected := terminals[0]
		plan.terminal = &selected
		plan.hidden = selected.binding.Action == ActionHide
		plan.conflicts = append(plan.conflicts, terminals[1:]...)
	}
	return plan, nil
}

func validateBindingContribution(binding ExecutionBinding, contribution Contribution) error {
	if binding.ContractVersion != contribution.ContractVersion || binding.Artifact != contribution.Artifact {
		return ErrContractStale
	}
	switch binding.Action {
	case ActionFilter:
		if contribution.Kind != KindRenderFilter && contribution.Kind != KindSanitizer {
			return fmt.Errorf("%w: %s cannot execute as a filter", ErrCompositionInvalid, contribution.ID)
		}
	default:
		if contribution.Kind == KindRenderFilter || contribution.Kind == KindSanitizer {
			return fmt.Errorf("%w: %s cannot execute as a renderer", ErrCompositionInvalid, contribution.ID)
		}
	}
	if binding.Action != ActionHide && contribution.Handler == "" {
		// ProviderSet is an executable backend callback. Renderer-only/static
		// declarations require a different Host renderer and cannot be smuggled
		// through this in-process execution interface.
		return fmt.Errorf("%w: %s has no executable handler", ErrContractInsufficient, contribution.ID)
	}
	// The existing content contract carries no target/action fields. Only the
	// Host binding may map a declaration to another target.
	if binding.Action == ActionAdd && binding.DeclarationID != binding.TargetID {
		return fmt.Errorf("%w: add binding must own its target", ErrCompositionInvalid)
	}
	return nil
}

func sortPlannedBindings(values []plannedBinding) {
	sort.Slice(values, func(i, j int) bool {
		return executionBindingBefore(values[i].binding, values[j].binding)
	})
}

func (p executionPlan) selectedSteps() []plannedBinding {
	result := make([]plannedBinding, 0, len(p.before)+len(p.wrap)+len(p.after)+len(p.filters)+1)
	result = append(result, p.before...)
	if p.terminal != nil {
		result = append(result, *p.terminal)
	} else {
		result = append(result, p.base)
	}
	result = append(result, p.wrap...)
	result = append(result, p.after...)
	result = append(result, p.filters...)
	return result
}
