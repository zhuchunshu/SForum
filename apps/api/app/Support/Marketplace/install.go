package marketplace

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"
	"sync"
)

// MemoryInstaller is a test/dev installer that records stage/activate/rollback.
// Production should wire Host staged upload + RuntimeRollout instead.
type MemoryInstaller struct {
	mu      sync.Mutex
	staged  map[string]StageResult
	active  map[string]string // extensionID -> digest
	history []string
}

// NewMemoryInstaller builds an in-memory install binding for tests.
func NewMemoryInstaller() *MemoryInstaller {
	return &MemoryInstaller{
		staged: make(map[string]StageResult),
		active: make(map[string]string),
	}
}

// Preflight verifies plan digests and non-empty order.
func (m *MemoryInstaller) Preflight(_ context.Context, plan InstallPlan) error {
	if plan.ExtensionID == "" || plan.PackageDigest == "" || len(plan.Order) == 0 {
		return fmt.Errorf("%w: empty plan", ErrInstall)
	}
	if !validSHA256Digest(plan.PackageDigest) {
		return ErrDigest
	}
	for _, step := range plan.Order {
		if !validExtensionID(step.ExtensionID) || !validSHA256Digest(step.PackageDigest) {
			return ErrDigest
		}
	}
	return nil
}

// Stage records package bytes and verifies digest matches plan.
func (m *MemoryInstaller) Stage(_ context.Context, plan InstallPlan, packageBytes []byte) (StageResult, error) {
	if err := m.Preflight(context.Background(), plan); err != nil {
		return StageResult{}, err
	}
	if len(packageBytes) == 0 {
		return StageResult{}, fmt.Errorf("%w: empty package", ErrInstall)
	}
	sum := sha256.Sum256(packageBytes)
	digest := hex.EncodeToString(sum[:])
	if !strings.EqualFold(digest, plan.PackageDigest) {
		return StageResult{}, fmt.Errorf("%w: package digest mismatch", ErrDigest)
	}
	orderIDs := make([]string, 0, len(plan.Order))
	for _, step := range plan.Order {
		orderIDs = append(orderIDs, step.ExtensionID)
	}
	result := StageResult{
		ExtensionID: plan.ExtensionID, StagedDigest: digest,
		StagedVersion: plan.Version, DependencyOrder: orderIDs,
		RolloutPlanID: "mem-rollout-" + plan.ExtensionID, PreflightOK: true,
	}
	m.mu.Lock()
	m.staged[plan.ExtensionID] = result
	m.history = append(m.history, "stage:"+plan.ExtensionID)
	m.mu.Unlock()
	return result, nil
}

// Activate promotes staged digest to active.
func (m *MemoryInstaller) Activate(_ context.Context, plan InstallPlan, staged StageResult) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	current, ok := m.staged[plan.ExtensionID]
	if !ok || current.StagedDigest != staged.StagedDigest {
		return fmt.Errorf("%w: staged candidate missing", ErrInstall)
	}
	m.active[plan.ExtensionID] = staged.StagedDigest
	m.history = append(m.history, "activate:"+plan.ExtensionID)
	return nil
}

// Rollback reverts active pointer to source digest.
func (m *MemoryInstaller) Rollback(_ context.Context, plan InstallPlan, reason string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if plan.SourceDigest == "" {
		return fmt.Errorf("%w: source digest required for rollback", ErrInstall)
	}
	m.active[plan.ExtensionID] = plan.SourceDigest
	m.history = append(m.history, "rollback:"+plan.ExtensionID+":"+strings.TrimSpace(reason))
	return nil
}

// ActiveDigest returns the active package digest for tests.
func (m *MemoryInstaller) ActiveDigest(extensionID string) string {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.active[strings.ToLower(strings.TrimSpace(extensionID))]
}

// History returns install binding actions for tests.
func (m *MemoryInstaller) History() []string {
	m.mu.Lock()
	defer m.mu.Unlock()
	return append([]string(nil), m.history...)
}

// HostInstaller binds marketplace to Host staged install + RuntimeRollout.
// Callers supply function hooks so Support/Marketplace does not import Models.
type HostInstaller struct {
	// PreflightFn validates the plan against Host state.
	PreflightFn func(ctx context.Context, plan InstallPlan) error
	// StageFn stages package bytes and returns staged identity.
	StageFn func(ctx context.Context, plan InstallPlan, packageBytes []byte) (StageResult, error)
	// ActivateFn promotes staged candidate (creates/drives rollout).
	ActivateFn func(ctx context.Context, plan InstallPlan, staged StageResult) error
	// RollbackFn one-click rollback to source digest.
	RollbackFn func(ctx context.Context, plan InstallPlan, reason string) error
}

// Preflight implements Installer.
func (h *HostInstaller) Preflight(ctx context.Context, plan InstallPlan) error {
	if h == nil || h.PreflightFn == nil {
		return fmt.Errorf("%w: host preflight not bound", ErrInstall)
	}
	return h.PreflightFn(ctx, plan)
}

// Stage implements Installer.
func (h *HostInstaller) Stage(ctx context.Context, plan InstallPlan, packageBytes []byte) (StageResult, error) {
	if h == nil || h.StageFn == nil {
		return StageResult{}, fmt.Errorf("%w: host stage not bound", ErrInstall)
	}
	return h.StageFn(ctx, plan, packageBytes)
}

// Activate implements Installer.
func (h *HostInstaller) Activate(ctx context.Context, plan InstallPlan, staged StageResult) error {
	if h == nil || h.ActivateFn == nil {
		return fmt.Errorf("%w: host activate not bound", ErrInstall)
	}
	return h.ActivateFn(ctx, plan, staged)
}

// Rollback implements Installer.
func (h *HostInstaller) Rollback(ctx context.Context, plan InstallPlan, reason string) error {
	if h == nil || h.RollbackFn == nil {
		return fmt.Errorf("%w: host rollback not bound", ErrInstall)
	}
	return h.RollbackFn(ctx, plan, reason)
}
