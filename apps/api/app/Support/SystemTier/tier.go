// Package systemtier models the optional operator-managed system extension tier
// for early auth/cache/storage providers (V3 P12). Safe Mode always bypasses it
// before any system extension code is loaded. CLI recovery can disable members
// without starting API/Nuxt/plugin runtimes.
package systemtier

import (
	"context"
	"errors"
	"sort"
	"strings"
	"sync"
	"time"
)

// SchemaVersion is the system tier contract.
const SchemaVersion = "sforum.system-tier@1"

// Roles for early infrastructure providers.
const (
	RoleAuth    = "auth"
	RoleCache   = "cache"
	RoleStorage = "storage"
	RoleInfra   = "infra"
)

var (
	ErrInvalid  = errors.New("system tier input is invalid")
	ErrNotFound = errors.New("system tier member is not found")
)

// Member is one system-tier extension assignment.
type Member struct {
	ExtensionID string    `json:"extensionId"`
	Role        string    `json:"role"`
	// Priority orders load within the tier (lower first).
	Priority  int       `json:"priority"`
	Enabled   bool      `json:"enabled"`
	UpdatedAt time.Time `json:"updatedAt,omitempty"`
	UpdatedBy string    `json:"updatedBy,omitempty"`
}

// Snapshot is the operator view of the system tier.
type Snapshot struct {
	SchemaVersion string `json:"schemaVersion"`
	// SafeModeBypass is always true: Safe Mode never loads this tier.
	SafeModeBypass bool     `json:"safeModeBypass"`
	Members        []Member `json:"members"`
}

// Store is the durable system tier membership authority.
type Store interface {
	Upsert(ctx context.Context, member Member) error
	Disable(ctx context.Context, extensionID, actor string) error
	// Get returns one member.
	Get(ctx context.Context, extensionID string) (Member, error)
	// List returns all members including disabled.
	List(ctx context.Context) ([]Member, error)
}

// Registry is the Host system tier facade over a durable Store.
// CLI recovery and Safe Mode paths must not load extension code.
type Registry struct {
	store Store
}

// New builds a process-local memory registry (tests). Prefer NewWithStore.
func New() *Registry {
	return NewWithStore(NewMemoryStore())
}

// NewWithStore builds a registry on durable membership storage.
func NewWithStore(store Store) *Registry {
	if store == nil {
		store = NewMemoryStore()
	}
	return &Registry{store: store}
}

// Upsert adds or updates a member. Does not load package code.
func (r *Registry) Upsert(ctx context.Context, member Member) error {
	if r == nil || r.store == nil {
		return ErrInvalid
	}
	if ctx == nil {
		ctx = context.Background()
	}
	member.ExtensionID = strings.ToLower(strings.TrimSpace(member.ExtensionID))
	member.Role = strings.ToLower(strings.TrimSpace(member.Role))
	member.UpdatedBy = strings.TrimSpace(member.UpdatedBy)
	if member.ExtensionID == "" || !validRole(member.Role) {
		return ErrInvalid
	}
	if member.UpdatedAt.IsZero() {
		member.UpdatedAt = time.Now().UTC()
	}
	return r.store.Upsert(ctx, member)
}

// Disable marks a member disabled without loading code (CLI recovery path).
func (r *Registry) Disable(ctx context.Context, extensionID, actor string) error {
	if r == nil || r.store == nil {
		return ErrInvalid
	}
	if ctx == nil {
		ctx = context.Background()
	}
	return r.store.Disable(ctx, strings.ToLower(strings.TrimSpace(extensionID)), strings.TrimSpace(actor))
}

// LoadOrder returns enabled members sorted by priority then id.
// When safeMode is true, returns nil BEFORE any system extension code would load.
func (r *Registry) LoadOrder(ctx context.Context, safeMode bool) ([]Member, error) {
	if r == nil || r.store == nil || safeMode {
		// Safe Mode 在加载任何 system extension 代码前绕过整个 tier。
		return nil, nil
	}
	if ctx == nil {
		ctx = context.Background()
	}
	all, err := r.store.List(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]Member, 0, len(all))
	for _, member := range all {
		if member.Enabled {
			out = append(out, member)
		}
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Priority != out[j].Priority {
			return out[i].Priority < out[j].Priority
		}
		return out[i].ExtensionID < out[j].ExtensionID
	})
	return out, nil
}

// Snapshot returns full membership including disabled.
func (r *Registry) Snapshot(ctx context.Context) (Snapshot, error) {
	if r == nil || r.store == nil {
		return Snapshot{SchemaVersion: SchemaVersion, SafeModeBypass: true}, nil
	}
	if ctx == nil {
		ctx = context.Background()
	}
	members, err := r.store.List(ctx)
	if err != nil {
		return Snapshot{}, err
	}
	sort.Slice(members, func(i, j int) bool { return members[i].ExtensionID < members[j].ExtensionID })
	return Snapshot{SchemaVersion: SchemaVersion, SafeModeBypass: true, Members: members}, nil
}

// MemoryStore is process-local membership (tests; production uses PostgresStore).
type MemoryStore struct {
	mu      sync.Mutex
	members map[string]Member
}

// NewMemoryStore builds an empty memory store.
func NewMemoryStore() *MemoryStore {
	return &MemoryStore{members: make(map[string]Member)}
}

// Upsert implements Store.
func (m *MemoryStore) Upsert(_ context.Context, member Member) error {
	if m == nil {
		return ErrInvalid
	}
	m.mu.Lock()
	if m.members == nil {
		m.members = make(map[string]Member)
	}
	m.members[member.ExtensionID] = member
	m.mu.Unlock()
	return nil
}

// Disable implements Store.
func (m *MemoryStore) Disable(_ context.Context, extensionID, actor string) error {
	if m == nil {
		return ErrInvalid
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	member, ok := m.members[extensionID]
	if !ok {
		return ErrNotFound
	}
	member.Enabled = false
	member.UpdatedBy = actor
	member.UpdatedAt = time.Now().UTC()
	m.members[extensionID] = member
	return nil
}

// Get implements Store.
func (m *MemoryStore) Get(_ context.Context, extensionID string) (Member, error) {
	if m == nil {
		return Member{}, ErrNotFound
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	member, ok := m.members[extensionID]
	if !ok {
		return Member{}, ErrNotFound
	}
	return member, nil
}

// List implements Store.
func (m *MemoryStore) List(_ context.Context) ([]Member, error) {
	if m == nil {
		return nil, nil
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	out := make([]Member, 0, len(m.members))
	for _, member := range m.members {
		out = append(out, member)
	}
	return out, nil
}

func validRole(role string) bool {
	switch role {
	case RoleAuth, RoleCache, RoleStorage, RoleInfra:
		return true
	default:
		return false
	}
}

var (
	_ Store = (*MemoryStore)(nil)
)
