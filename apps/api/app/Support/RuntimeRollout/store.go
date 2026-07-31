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
	created := clonePlan(plan)
	if created.Revision < 1 {
		created.Revision = 1
	}
	m.plans[plan.PlanID] = created
	return clonePlan(created), nil
}

// Save implements PlanStore with revision CAS.
func (m *MemoryStore) Save(_ context.Context, plan Plan) (Plan, error) {
	if m == nil {
		return Plan{}, ErrNotFound
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.plans == nil {
		return Plan{}, ErrNotFound
	}
	current, ok := m.plans[plan.PlanID]
	if !ok {
		return Plan{}, ErrNotFound
	}
	// expectedRevision 0 表示不校验（兼容旧调用）；正值必须匹配。
	if plan.Revision > 0 && current.Revision != plan.Revision {
		return Plan{}, ErrConflict
	}
	next := clonePlan(plan)
	next.Revision = current.Revision + 1
	m.plans[plan.PlanID] = next
	return clonePlan(next), nil
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

// ActiveForExtension implements PlanStore. Active plans remain addressable so
// a rollback after promotion can use the same durable plan record.
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
		if plan.ExtensionID == extensionID && plan.Phase != PhaseFailed && plan.Phase != PhaseRolledBack {
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
