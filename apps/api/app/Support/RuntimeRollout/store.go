package runtimerollout

import (
	"context"
	"sync"
)

// PlanStore is the durable authority for multi-node rollout plans.
// Process-local maps are not sufficient for multi-API coordination.
type PlanStore interface {
	// Create inserts a pending plan. Concurrent creates for the same extension
	// must yield exactly one winner (unique active plan).
	Create(ctx context.Context, plan Plan) (Plan, error)
	// Save persists a full plan snapshot (phase machine transitions).
	Save(ctx context.Context, plan Plan) (Plan, error)
	// Get loads one plan by id.
	Get(ctx context.Context, planID string) (Plan, error)
	// List returns all plans (or filtered by extensionID when non-empty).
	List(ctx context.Context, extensionID string) ([]Plan, error)
	// ActiveForExtension returns the non-terminal plan for an extension if any.
	ActiveForExtension(ctx context.Context, extensionID string) (Plan, bool, error)
}

// MemoryStore is tests-only durable stand-in (still survives Service rebuild
// when the same store instance is reused). Concurrent-safe for race tests.
type MemoryStore struct {
	mu    sync.Mutex
	plans map[string]Plan
}

// NewMemoryStore builds an empty memory plan store.
func NewMemoryStore() *MemoryStore {
	return &MemoryStore{plans: make(map[string]Plan)}
}

// Create implements PlanStore.
func (m *MemoryStore) Create(_ context.Context, plan Plan) (Plan, error) {
	if m == nil {
		return Plan{}, ErrInvalid
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.plans == nil {
		m.plans = make(map[string]Plan)
	}
	for _, existing := range m.plans {
		if existing.ExtensionID == plan.ExtensionID && !isTerminalPhase(existing.Phase) {
			return Plan{}, ErrConflict
		}
	}
	m.plans[plan.PlanID] = clonePlan(plan)
	return clonePlan(plan), nil
}

// Save implements PlanStore.
func (m *MemoryStore) Save(_ context.Context, plan Plan) (Plan, error) {
	if m == nil {
		return Plan{}, ErrNotFound
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.plans == nil {
		return Plan{}, ErrNotFound
	}
	if _, ok := m.plans[plan.PlanID]; !ok {
		return Plan{}, ErrNotFound
	}
	m.plans[plan.PlanID] = clonePlan(plan)
	return clonePlan(plan), nil
}

// Get implements PlanStore.
func (m *MemoryStore) Get(_ context.Context, planID string) (Plan, error) {
	if m == nil {
		return Plan{}, ErrNotFound
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.plans == nil {
		return Plan{}, ErrNotFound
	}
	plan, ok := m.plans[planID]
	if !ok {
		return Plan{}, ErrNotFound
	}
	return clonePlan(plan), nil
}

// List implements PlanStore.
func (m *MemoryStore) List(_ context.Context, extensionID string) ([]Plan, error) {
	if m == nil {
		return nil, nil
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.plans == nil {
		return nil, nil
	}
	out := make([]Plan, 0, len(m.plans))
	for _, plan := range m.plans {
		if extensionID != "" && plan.ExtensionID != extensionID {
			continue
		}
		out = append(out, clonePlan(plan))
	}
	return out, nil
}

// ActiveForExtension implements PlanStore.
func (m *MemoryStore) ActiveForExtension(_ context.Context, extensionID string) (Plan, bool, error) {
	if m == nil {
		return Plan{}, false, nil
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.plans == nil {
		return Plan{}, false, nil
	}
	for _, plan := range m.plans {
		if plan.ExtensionID == extensionID && !isTerminalPhase(plan.Phase) {
			return clonePlan(plan), true, nil
		}
	}
	return Plan{}, false, nil
}

func isTerminalPhase(phase string) bool {
	switch phase {
	case PhaseActive, PhaseFailed, PhaseRolledBack:
		return true
	default:
		return false
	}
}
