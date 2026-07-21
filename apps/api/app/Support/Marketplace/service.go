package marketplace

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"
)

// Service verifies and queries a Host marketplace index.
// After LoadIndex the internal snapshot is fully deep-copied; List/Resolve
// return isolated copies so callers cannot mutate nested slices.
type Service struct {
	mu       sync.Mutex
	index    *Index
	policy   OperatorPolicy
	verifier Verifier
	// now injectable for stale/window tests.
	now func() time.Time
	// installer optional production binding.
	installer Installer
}

// Options configures a marketplace service.
type Options struct {
	Policy    OperatorPolicy
	Verifier  Verifier
	Installer Installer
	Now       func() time.Time
}

// New builds a marketplace service. Prefer NewWithOptions for Ed25519.
func New(policy OperatorPolicy) *Service {
	return NewWithOptions(Options{Policy: policy})
}

// NewWithOptions builds a service with an Ed25519 verifier and optional installer.
func NewWithOptions(opts Options) *Service {
	policy := opts.Policy
	if len(policy.AllowedChannels) == 0 {
		policy.AllowedChannels = []string{ChannelStable}
	}
	// 推荐默认：离线直传始终可用。
	policy.DirectUploadFallback = true
	now := opts.Now
	if now == nil {
		now = func() time.Time { return time.Now().UTC() }
	}
	return &Service{
		policy:    policy,
		verifier:  opts.Verifier,
		installer: opts.Installer,
		now:       now,
	}
}

// BindInstaller late-binds Host staged install + RuntimeRollout after bootstrap.
func (s *Service) BindInstaller(installer Installer) {
	if s == nil {
		return
	}
	s.mu.Lock()
	s.installer = installer
	s.mu.Unlock()
}

// Installer returns the bound installer (may be nil until production bind).
func (s *Service) Installer() Installer {
	if s == nil {
		return nil
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.installer
}

// LoadIndex verifies signature (unless AllowUnsigned) and replaces the active index
// with a full deep copy of Entries/Dependencies/Notices.
func (s *Service) LoadIndex(index Index) error {
	if s == nil {
		return ErrInvalid
	}
	index.SchemaVersion = strings.TrimSpace(index.SchemaVersion)
	if index.SchemaVersion == "" {
		index.SchemaVersion = SchemaVersion
	}
	if index.SchemaVersion != SchemaVersion {
		return ErrInvalid
	}
	// Normalize and validate every entry before accepting the snapshot.
	for i := range index.Entries {
		if err := normalizeEntry(&index.Entries[i]); err != nil {
			return err
		}
	}
	if !s.policy.AllowUnsigned {
		if s.verifier == nil {
			return ErrSignature
		}
		if strings.TrimSpace(index.Signature) == "" {
			return ErrSignature
		}
		kind := strings.ToLower(strings.TrimSpace(index.SignerKind))
		if kind == "" {
			kind = SignerKindEd25519
		}
		if kind != SignerKindEd25519 {
			return ErrSignature
		}
		if id := s.verifier.PublicKeyID(); id != "" && strings.TrimSpace(index.SignerID) != "" &&
			!strings.EqualFold(id, index.SignerID) {
			return ErrSignature
		}
		body, err := canonicalIndexBytes(index)
		if err != nil {
			return err
		}
		if err := s.verifier.Verify(body, index.Signature); err != nil {
			return ErrSignature
		}
	}
	// 时间窗：NotBefore 之后才接受索引。
	now := s.now()
	if !index.NotBefore.IsZero() && now.Before(index.NotBefore) {
		return ErrStale
	}
	if !index.ExpiresAt.IsZero() && now.After(index.ExpiresAt) {
		// 仍可 Load 以便运维查看，但 Resolve 会拒绝；这里选择拒绝 Load 更安全。
		// 与旧行为一致：Load 成功、Resolve 时检查 stale。允许 Load。
	}

	s.mu.Lock()
	// 验签后完整深拷贝；调用方后续改嵌套切片不得影响内部快照。
	cloned := cloneIndex(index)
	s.index = &cloned
	s.mu.Unlock()
	return nil
}

// Resolve finds an installable release with recursive dependency order.
func (s *Service) Resolve(extensionID, channel string) (ResolveResult, error) {
	if s == nil {
		return ResolveResult{}, ErrInvalid
	}
	extensionID = strings.ToLower(strings.TrimSpace(extensionID))
	channel = strings.ToLower(strings.TrimSpace(channel))
	if channel == "" {
		channel = ChannelStable
	}
	if !validExtensionID(extensionID) {
		return ResolveResult{}, ErrInvalid
	}
	if !channelAllowed(s.policy.AllowedChannels, channel) {
		return ResolveResult{}, ErrPolicy
	}

	s.mu.Lock()
	if s.index == nil {
		s.mu.Unlock()
		return ResolveResult{}, ErrNotFound
	}
	// 使用内部快照的深拷贝做解析，持锁时间尽量短。
	snapshot := cloneIndex(*s.index)
	policy := s.policy
	now := s.now()
	s.mu.Unlock()

	if !snapshot.ExpiresAt.IsZero() && now.After(snapshot.ExpiresAt) {
		return ResolveResult{}, ErrStale
	}
	if !snapshot.NotBefore.IsZero() && now.Before(snapshot.NotBefore) {
		return ResolveResult{}, ErrStale
	}

	result, err := resolveRecursive(snapshot.Entries, extensionID, channel, policy, now)
	if err != nil {
		return ResolveResult{}, err
	}
	return cloneResolveResult(result), nil
}

// List returns non-withdrawn entries (or all if includeWithdrawn) as deep copies.
func (s *Service) List(includeWithdrawn bool) []Entry {
	if s == nil {
		return nil
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.index == nil {
		return nil
	}
	out := make([]Entry, 0, len(s.index.Entries))
	for _, entry := range s.index.Entries {
		if entry.Withdrawn && !includeWithdrawn {
			continue
		}
		out = append(out, cloneEntry(entry))
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].ExtensionID != out[j].ExtensionID {
			return out[i].ExtensionID < out[j].ExtensionID
		}
		return out[i].Version < out[j].Version
	})
	return out
}

// Policy returns a copy of operator policy.
func (s *Service) Policy() OperatorPolicy {
	if s == nil {
		return OperatorPolicy{}
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.policy
}

// DirectUploadAvailable reports offline install fallback.
func (s *Service) DirectUploadAvailable() bool {
	if s == nil {
		return false
	}
	return s.policy.DirectUploadFallback
}

// SetInstaller wires the production install/stage/rollback binding.
func (s *Service) SetInstaller(installer Installer) {
	if s == nil {
		return
	}
	s.mu.Lock()
	s.installer = installer
	s.mu.Unlock()
}

// InstallPreflight runs the bound installer preflight for a resolved plan.
func (s *Service) InstallPreflight(ctx context.Context, plan InstallPlan) error {
	if s == nil {
		return ErrInvalid
	}
	s.mu.Lock()
	installer := s.installer
	s.mu.Unlock()
	if installer == nil {
		return fmt.Errorf("%w: installer not bound", ErrInstall)
	}
	return installer.Preflight(ctx, cloneInstallPlan(plan))
}

// StageInstall stages package bytes after resolve (real Host path).
func (s *Service) StageInstall(ctx context.Context, plan InstallPlan, packageBytes []byte) (StageResult, error) {
	if s == nil {
		return StageResult{}, ErrInvalid
	}
	s.mu.Lock()
	installer := s.installer
	s.mu.Unlock()
	if installer == nil {
		return StageResult{}, fmt.Errorf("%w: installer not bound", ErrInstall)
	}
	if err := installer.Preflight(ctx, cloneInstallPlan(plan)); err != nil {
		return StageResult{}, err
	}
	return installer.Stage(ctx, cloneInstallPlan(plan), packageBytes)
}

// ActivateStaged promotes a staged marketplace install via rollout authority.
func (s *Service) ActivateStaged(ctx context.Context, plan InstallPlan, staged StageResult) error {
	if s == nil {
		return ErrInvalid
	}
	s.mu.Lock()
	installer := s.installer
	s.mu.Unlock()
	if installer == nil {
		return fmt.Errorf("%w: installer not bound", ErrInstall)
	}
	return installer.Activate(ctx, cloneInstallPlan(plan), staged)
}

// RollbackInstall one-click reverts to source digest via installer.
func (s *Service) RollbackInstall(ctx context.Context, plan InstallPlan, reason string) error {
	if s == nil {
		return ErrInvalid
	}
	s.mu.Lock()
	installer := s.installer
	s.mu.Unlock()
	if installer == nil {
		return fmt.Errorf("%w: installer not bound", ErrInstall)
	}
	return installer.Rollback(ctx, cloneInstallPlan(plan), reason)
}

// FormatNotice is a helper for tests/audit messages.
func FormatNotice(n Notice) string {
	return fmt.Sprintf("%s:%s", n.Kind, n.Summary)
}

func channelAllowed(allowed []string, channel string) bool {
	for _, item := range allowed {
		if strings.EqualFold(item, channel) {
			return true
		}
	}
	return false
}

func blockedByNotice(policy OperatorPolicy, notices []Notice) bool {
	max := strings.ToLower(strings.TrimSpace(policy.MaxVulnerabilitySeverity))
	if max == "" {
		for _, n := range notices {
			if n.Kind == NoticeRevocation || n.Kind == NoticeWithdrawn {
				return true
			}
		}
		return false
	}
	rank := severityRank(max)
	for _, n := range notices {
		if n.Kind == NoticeRevocation || n.Kind == NoticeWithdrawn {
			return true
		}
		if n.Kind == NoticeVulnerability && severityRank(n.Severity) > rank {
			return true
		}
	}
	return false
}

func severityRank(value string) int {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "low":
		return 1
	case "medium":
		return 2
	case "high":
		return 3
	case "critical":
		return 4
	default:
		return 0
	}
}
