package runtimerollout

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"sort"
	"strings"
	"time"
)

// Service is the Host rollout coordinator backed by a durable PlanStore.
// Production multi-node durability reuses PostgreSQL authority (same class as
// plugin_runtime publication journals): node acks, migration-once, health gate,
// drain, promote, and rollback all persist and recover after API/worker restart.
type Service struct {
	store PlanStore
	// now is injectable for tests.
	now func() time.Time
}

// New builds a coordinator with an in-memory store (tests / single-process only).
// Production must use NewWithStore(NewPostgresStore(pool)).
func New() *Service {
	return NewWithStore(NewMemoryStore())
}

// NewWithStore builds a coordinator on a durable plan store.
func NewWithStore(store PlanStore) *Service {
	if store == nil {
		store = NewMemoryStore()
	}
	return &Service{
		store: store,
		now:   func() time.Time { return time.Now().UTC() },
	}
}

// CreatePlan starts a staged rollout. MigrationReady must be false until
// MarkMigrationReady is called (migration-once before activation).
// Multi-API concurrent Create for the same extension: only one winner.
func (s *Service) CreatePlan(
	ctx context.Context,
	extensionID, sourceDigest, targetDigest, actor string,
	canaryPercent, retain int,
) (Plan, error) {
	if s == nil || s.store == nil {
		return Plan{}, ErrInvalid
	}
	if ctx == nil {
		ctx = context.Background()
	}
	extensionID = strings.ToLower(strings.TrimSpace(extensionID))
	sourceDigest = strings.ToLower(strings.TrimSpace(sourceDigest))
	targetDigest = strings.ToLower(strings.TrimSpace(targetDigest))
	actor = strings.TrimSpace(actor)
	if extensionID == "" || sourceDigest == "" || targetDigest == "" || actor == "" {
		return Plan{}, ErrInvalid
	}
	if sourceDigest == targetDigest {
		return Plan{}, ErrInvalid
	}
	if canaryPercent <= 0 {
		canaryPercent = DefaultCanaryPercent
	}
	if canaryPercent > 100 {
		canaryPercent = 100
	}
	if retain <= 0 {
		retain = DefaultRetainVersions
	}
	// PlanID 必须跨 API 节点全局唯一；进程内 seq 在并发 Create 时会撞车。
	planID, err := newPlanID()
	if err != nil {
		return Plan{}, err
	}
	plan := Plan{
		SchemaVersion:   SchemaVersion,
		PlanID:          planID,
		ExtensionID:     extensionID,
		SourceDigest:    sourceDigest,
		TargetDigest:    targetDigest,
		CanaryPercent:   canaryPercent,
		Phase:           PhasePending,
		RetainVersions:  retain,
		Actor:           actor,
		UpdatedAt:       s.now(),
		NodeAcks:        map[string]NodeAck{},
		RetainedDigests: []string{sourceDigest},
	}
	created, err := s.store.Create(ctx, plan)
	if err != nil {
		return Plan{}, err
	}
	return clonePlan(created), nil
}

// MarkMigrationReady records that Host migration-once proof succeeded.
func (s *Service) MarkMigrationReady(ctx context.Context, planID, actor string) (Plan, error) {
	return s.transition(ctx, planID, actor, func(plan *Plan) error {
		if plan.Phase != PhasePending && plan.Phase != PhaseMigrating {
			return ErrPhase
		}
		plan.Phase = PhaseMigrating
		plan.MigrationReady = true
		plan.Phase = PhaseStaged
		return nil
	})
}

// AckNode records a node's phase acknowledgement and health (durable).
// Missed Redis notifications are recovered by reloading plan state from store.
func (s *Service) AckNode(ctx context.Context, planID, nodeID, phase, health string, canary bool) (Plan, error) {
	nodeID = strings.TrimSpace(nodeID)
	phase = strings.ToLower(strings.TrimSpace(phase))
	health = strings.ToLower(strings.TrimSpace(health))
	if nodeID == "" || phase == "" {
		return Plan{}, ErrInvalid
	}
	if health == "" {
		health = HealthUnknown
	}
	return s.transition(ctx, planID, "node:"+nodeID, func(plan *Plan) error {
		if plan.NodeAcks == nil {
			plan.NodeAcks = map[string]NodeAck{}
		}
		plan.NodeAcks[nodeID] = NodeAck{
			NodeID: nodeID, Phase: phase, Health: health, At: s.now(), Canary: canary,
		}
		return nil
	})
}

// MarkNodeLost marks a node unhealthy after lease expiry / disconnect.
func (s *Service) MarkNodeLost(ctx context.Context, planID, nodeID, actor string) (Plan, error) {
	return s.transition(ctx, planID, actor, func(plan *Plan) error {
		if plan.NodeAcks == nil {
			return ErrInvalid
		}
		ack, ok := plan.NodeAcks[nodeID]
		if !ok {
			return ErrInvalid
		}
		ack.Health = HealthUnhealthy
		ack.At = s.now()
		plan.NodeAcks[nodeID] = ack
		plan.LastError = "node lost: " + nodeID
		return nil
	})
}

// SelectCanary marks the canary cohort from currently known nodes.
func (s *Service) SelectCanary(ctx context.Context, planID, actor string) (Plan, error) {
	return s.transition(ctx, planID, actor, func(plan *Plan) error {
		if !plan.MigrationReady || plan.Phase != PhaseStaged {
			return ErrPhase
		}
		nodes := nodeIDs(plan.NodeAcks)
		if len(nodes) == 0 {
			return ErrInvalid
		}
		sort.Strings(nodes)
		n := (len(nodes)*plan.CanaryPercent + 99) / 100
		if n < 1 {
			n = 1
		}
		if n > len(nodes) {
			n = len(nodes)
		}
		selected := map[string]struct{}{}
		for i := 0; i < n; i++ {
			selected[nodes[i]] = struct{}{}
		}
		for id, ack := range plan.NodeAcks {
			ack.Canary = false
			if _, ok := selected[id]; ok {
				ack.Canary = true
			}
			plan.NodeAcks[id] = ack
		}
		plan.Phase = PhaseCanary
		return nil
	})
}

// HealthGate checks canary (or all) nodes are healthy before promote.
func (s *Service) HealthGate(ctx context.Context, planID, actor string, canaryOnly bool) (Plan, error) {
	return s.transition(ctx, planID, actor, func(plan *Plan) error {
		if plan.Phase != PhaseCanary && plan.Phase != PhaseDraining && plan.Phase != PhasePromoting {
			return ErrPhase
		}
		checked := 0
		for _, ack := range plan.NodeAcks {
			if canaryOnly && !ack.Canary {
				continue
			}
			checked++
			if ack.Health != HealthHealthy {
				plan.LastError = "node " + ack.NodeID + " health=" + ack.Health
				return ErrHealthGate
			}
		}
		if checked == 0 {
			return ErrHealthGate
		}
		return nil
	})
}

// BeginDrain marks draining before atomic promote.
func (s *Service) BeginDrain(ctx context.Context, planID, actor string) (Plan, error) {
	return s.transition(ctx, planID, actor, func(plan *Plan) error {
		if plan.Phase != PhaseCanary {
			return ErrPhase
		}
		for _, ack := range plan.NodeAcks {
			if ack.Canary && ack.Health != HealthHealthy {
				return ErrHealthGate
			}
		}
		plan.Phase = PhaseDraining
		return nil
	})
}

// PromoteAtomic switches the active snapshot after drain + full health gate.
// Callers must also advance plugin/theme runtime publication to target digest.
func (s *Service) PromoteAtomic(ctx context.Context, planID, actor string) (Plan, error) {
	return s.transition(ctx, planID, actor, func(plan *Plan) error {
		if !plan.MigrationReady {
			return ErrMigration
		}
		if plan.Phase != PhaseDraining && plan.Phase != PhaseCanary {
			return ErrPhase
		}
		for _, ack := range plan.NodeAcks {
			if ack.Health != HealthHealthy {
				plan.LastError = "node " + ack.NodeID + " not healthy for promote"
				return ErrHealthGate
			}
		}
		plan.Phase = PhasePromoting
		sum := sha256.Sum256([]byte(plan.PlanID + "\x00" + plan.TargetDigest + "\x00" + plan.UpdatedAt.Format(time.RFC3339Nano)))
		plan.SnapshotID = "snap-" + hex.EncodeToString(sum[:8])
		retained := append([]string{}, plan.RetainedDigests...)
		retained = append(retained, plan.SourceDigest)
		plan.RetainedDigests = uniqueCap(retained, plan.RetainVersions)
		plan.Phase = PhaseActive
		plan.LastError = ""
		return nil
	})
}

// Rollback reverts desired pointer to source digest and keeps target retained.
func (s *Service) Rollback(ctx context.Context, planID, actor, reason string) (Plan, error) {
	return s.transition(ctx, planID, actor, func(plan *Plan) error {
		switch plan.Phase {
		case PhaseCanary, PhaseDraining, PhasePromoting, PhaseActive, PhaseFailed:
		default:
			return ErrPhase
		}
		plan.Phase = PhaseRollingBack
		plan.Reason = strings.TrimSpace(reason)
		retained := append([]string{}, plan.RetainedDigests...)
		retained = append(retained, plan.TargetDigest)
		plan.RetainedDigests = uniqueCap(retained, plan.RetainVersions)
		plan.Phase = PhaseRolledBack
		plan.SnapshotID = ""
		return nil
	})
}

// Fail marks a terminal failure without promote (e.g. migration failed).
func (s *Service) Fail(ctx context.Context, planID, actor, message string) (Plan, error) {
	return s.transition(ctx, planID, actor, func(plan *Plan) error {
		plan.Phase = PhaseFailed
		plan.LastError = strings.TrimSpace(message)
		return nil
	})
}

// Get returns a plan by id (reloadable after process restart).
func (s *Service) Get(ctx context.Context, planID string) (Plan, error) {
	if s == nil || s.store == nil {
		return Plan{}, ErrInvalid
	}
	if ctx == nil {
		ctx = context.Background()
	}
	return s.store.Get(ctx, strings.TrimSpace(planID))
}

// List returns all plans sorted by plan id.
func (s *Service) List(ctx context.Context) ([]Plan, error) {
	if s == nil || s.store == nil {
		return nil, ErrInvalid
	}
	if ctx == nil {
		ctx = context.Background()
	}
	out, err := s.store.List(ctx, "")
	if err != nil {
		return nil, err
	}
	sort.Slice(out, func(i, j int) bool { return out[i].PlanID < out[j].PlanID })
	return out, nil
}

func (s *Service) transition(ctx context.Context, planID, actor string, fn func(*Plan) error) (Plan, error) {
	if s == nil || s.store == nil {
		return Plan{}, ErrInvalid
	}
	if ctx == nil {
		ctx = context.Background()
	}
	planID = strings.TrimSpace(planID)
	actor = strings.TrimSpace(actor)
	if planID == "" || actor == "" {
		return Plan{}, ErrPermissionDenied
	}
	plan, err := s.store.Get(ctx, planID)
	if err != nil {
		return Plan{}, err
	}
	next := clonePlan(plan)
	if err := fn(&next); err != nil {
		if next.LastError != "" {
			next.UpdatedAt = s.now()
			_, _ = s.store.Save(ctx, next)
		}
		return clonePlan(next), err
	}
	next.Actor = actor
	next.UpdatedAt = s.now()
	saved, err := s.store.Save(ctx, next)
	if err != nil {
		return Plan{}, err
	}
	return clonePlan(saved), nil
}

func nodeIDs(acks map[string]NodeAck) []string {
	out := make([]string, 0, len(acks))
	for id := range acks {
		out = append(out, id)
	}
	return out
}

func uniqueCap(values []string, max int) []string {
	seen := map[string]struct{}{}
	out := make([]string, 0, len(values))
	for i := len(values) - 1; i >= 0; i-- {
		v := strings.TrimSpace(values[i])
		if v == "" {
			continue
		}
		if _, ok := seen[v]; ok {
			continue
		}
		seen[v] = struct{}{}
		out = append(out, v)
		if max > 0 && len(out) >= max {
			break
		}
	}
	for i, j := 0, len(out)-1; i < j; i, j = i+1, j-1 {
		out[i], out[j] = out[j], out[i]
	}
	return out
}

func clonePlan(plan Plan) Plan {
	out := plan
	if plan.NodeAcks != nil {
		out.NodeAcks = make(map[string]NodeAck, len(plan.NodeAcks))
		for k, v := range plan.NodeAcks {
			out.NodeAcks[k] = v
		}
	}
	if plan.RetainedDigests != nil {
		out.RetainedDigests = append([]string(nil), plan.RetainedDigests...)
	}
	return out
}

func newPlanID() (string, error) {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "", fmt.Errorf("%w: plan id: %v", ErrInvalid, err)
	}
	return "rollout-" + hex.EncodeToString(b[:]), nil
}
