package runtimerollout

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

// Service is the process-local Host rollout coordinator.
// Production multi-node durability reuses plugin_runtime publication journals;
// this service owns the phase machine, canary selection, health gate, and
// retention policy that those journals enforce.
type Service struct {
	mu     sync.Mutex
	plans  map[string]Plan
	seq    atomic.Uint64
	// now is injectable for tests.
	now func() time.Time
}

// New builds a rollout coordinator.
func New() *Service {
	return &Service{
		plans: make(map[string]Plan),
		now:   func() time.Time { return time.Now().UTC() },
	}
}

// CreatePlan starts a staged rollout. MigrationReady must be false until
// MarkMigrationReady is called (migration-once before activation).
func (s *Service) CreatePlan(extensionID, sourceDigest, targetDigest, actor string, canaryPercent, retain int) (Plan, error) {
	if s == nil {
		return Plan{}, ErrInvalid
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
	plan := Plan{
		SchemaVersion: SchemaVersion,
		PlanID:        fmt.Sprintf("rollout-%d", s.seq.Add(1)),
		ExtensionID:   extensionID,
		SourceDigest:  sourceDigest,
		TargetDigest:  targetDigest,
		CanaryPercent: canaryPercent,
		Phase:         PhasePending,
		RetainVersions: retain,
		Actor:         actor,
		UpdatedAt:     s.now(),
		NodeAcks:      map[string]NodeAck{},
		RetainedDigests: []string{sourceDigest},
	}
	s.mu.Lock()
	s.plans[plan.PlanID] = clonePlan(plan)
	s.mu.Unlock()
	return clonePlan(plan), nil
}

// MarkMigrationReady records that Host migration-once proof succeeded.
// Runtime activation is forbidden until this returns successfully.
func (s *Service) MarkMigrationReady(planID, actor string) (Plan, error) {
	return s.transition(planID, actor, func(plan *Plan) error {
		if plan.Phase != PhasePending && plan.Phase != PhaseMigrating {
			return ErrPhase
		}
		plan.Phase = PhaseMigrating
		plan.MigrationReady = true
		plan.Phase = PhaseStaged
		return nil
	})
}

// AckNode records a node's phase acknowledgement and health.
func (s *Service) AckNode(planID, nodeID, phase, health string, canary bool) (Plan, error) {
	nodeID = strings.TrimSpace(nodeID)
	phase = strings.ToLower(strings.TrimSpace(phase))
	health = strings.ToLower(strings.TrimSpace(health))
	if nodeID == "" || phase == "" {
		return Plan{}, ErrInvalid
	}
	if health == "" {
		health = HealthUnknown
	}
	return s.transition(planID, "node:"+nodeID, func(plan *Plan) error {
		if plan.NodeAcks == nil {
			plan.NodeAcks = map[string]NodeAck{}
		}
		plan.NodeAcks[nodeID] = NodeAck{
			NodeID: nodeID, Phase: phase, Health: health, At: s.now(), Canary: canary,
		}
		return nil
	})
}

// SelectCanary marks the canary cohort from currently known nodes.
func (s *Service) SelectCanary(planID, actor string) (Plan, error) {
	return s.transition(planID, actor, func(plan *Plan) error {
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
func (s *Service) HealthGate(planID, actor string, canaryOnly bool) (Plan, error) {
	return s.transition(planID, actor, func(plan *Plan) error {
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
func (s *Service) BeginDrain(planID, actor string) (Plan, error) {
	return s.transition(planID, actor, func(plan *Plan) error {
		if plan.Phase != PhaseCanary {
			return ErrPhase
		}
		// Require canary healthy first.
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
func (s *Service) PromoteAtomic(planID, actor string) (Plan, error) {
	return s.transition(planID, actor, func(plan *Plan) error {
		if !plan.MigrationReady {
			return ErrMigration
		}
		if plan.Phase != PhaseDraining && plan.Phase != PhaseCanary {
			return ErrPhase
		}
		// Full-cluster health required for final promote.
		for _, ack := range plan.NodeAcks {
			if ack.Health != HealthHealthy {
				plan.LastError = "node " + ack.NodeID + " not healthy for promote"
				return ErrHealthGate
			}
		}
		plan.Phase = PhasePromoting
		sum := sha256.Sum256([]byte(plan.PlanID + "\x00" + plan.TargetDigest + "\x00" + plan.UpdatedAt.Format(time.RFC3339Nano)))
		plan.SnapshotID = "snap-" + hex.EncodeToString(sum[:8])
		// Retention: keep source + prior retained, capped.
		retained := append([]string{}, plan.RetainedDigests...)
		retained = append(retained, plan.SourceDigest)
		plan.RetainedDigests = uniqueCap(retained, plan.RetainVersions)
		plan.Phase = PhaseActive
		plan.LastError = ""
		return nil
	})
}

// Rollback reverts desired pointer to source digest and keeps target retained.
func (s *Service) Rollback(planID, actor, reason string) (Plan, error) {
	return s.transition(planID, actor, func(plan *Plan) error {
		switch plan.Phase {
		case PhaseCanary, PhaseDraining, PhasePromoting, PhaseActive, PhaseFailed:
		default:
			return ErrPhase
		}
		plan.Phase = PhaseRollingBack
		plan.Reason = strings.TrimSpace(reason)
		// Target becomes retained; source is active again (caller flips desired).
		retained := append([]string{}, plan.RetainedDigests...)
		retained = append(retained, plan.TargetDigest)
		plan.RetainedDigests = uniqueCap(retained, plan.RetainVersions)
		plan.Phase = PhaseRolledBack
		plan.SnapshotID = ""
		return nil
	})
}

// Fail marks a terminal failure without promote.
func (s *Service) Fail(planID, actor, message string) (Plan, error) {
	return s.transition(planID, actor, func(plan *Plan) error {
		plan.Phase = PhaseFailed
		plan.LastError = strings.TrimSpace(message)
		return nil
	})
}

// Get returns a plan by id.
func (s *Service) Get(planID string) (Plan, error) {
	if s == nil {
		return Plan{}, ErrInvalid
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	plan, ok := s.plans[strings.TrimSpace(planID)]
	if !ok {
		return Plan{}, ErrNotFound
	}
	return clonePlan(plan), nil
}

// List returns all plans sorted by plan id.
func (s *Service) List() []Plan {
	if s == nil {
		return nil
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]Plan, 0, len(s.plans))
	for _, plan := range s.plans {
		out = append(out, clonePlan(plan))
	}
	sort.Slice(out, func(i, j int) bool { return out[i].PlanID < out[j].PlanID })
	return out
}

func (s *Service) transition(planID, actor string, fn func(*Plan) error) (Plan, error) {
	if s == nil {
		return Plan{}, ErrInvalid
	}
	planID = strings.TrimSpace(planID)
	actor = strings.TrimSpace(actor)
	if planID == "" || actor == "" {
		return Plan{}, ErrPermissionDenied
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	plan, ok := s.plans[planID]
	if !ok {
		return Plan{}, ErrNotFound
	}
	next := clonePlan(plan)
	if err := fn(&next); err != nil {
		// Persist last error on health/migration failures when set by fn.
		if next.LastError != "" {
			s.plans[planID] = next
		}
		return clonePlan(next), err
	}
	next.Actor = actor
	next.UpdatedAt = s.now()
	s.plans[planID] = next
	return clonePlan(next), nil
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
	// reverse to chronological-ish
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
