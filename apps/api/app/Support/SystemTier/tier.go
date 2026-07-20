// Package systemtier models the optional operator-managed system extension tier
// for early auth/cache/storage providers (V3 P12). Safe Mode always bypasses it.
package systemtier

import (
	"sort"
	"strings"
	"sync"
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

// Member is one system-tier extension assignment.
type Member struct {
	ExtensionID string `json:"extensionId"`
	Role        string `json:"role"`
	// Priority orders load within the tier (lower first).
	Priority int `json:"priority"`
	Enabled  bool `json:"enabled"`
}

// Snapshot is the operator view of the system tier.
type Snapshot struct {
	SchemaVersion string   `json:"schemaVersion"`
	// SafeModeBypass is always true: Safe Mode never loads this tier.
	SafeModeBypass bool     `json:"safeModeBypass"`
	Members        []Member `json:"members"`
}

// Registry is the process-local system tier membership store.
// CLI recovery can disable members without loading extension code.
type Registry struct {
	mu      sync.Mutex
	members map[string]Member
}

// New builds an empty system tier registry.
func New() *Registry {
	return &Registry{members: make(map[string]Member)}
}

// Upsert adds or updates a member. Does not load package code.
func (r *Registry) Upsert(member Member) error {
	if r == nil {
		return errInvalid
	}
	member.ExtensionID = strings.ToLower(strings.TrimSpace(member.ExtensionID))
	member.Role = strings.ToLower(strings.TrimSpace(member.Role))
	if member.ExtensionID == "" || !validRole(member.Role) {
		return errInvalid
	}
	r.mu.Lock()
	r.members[member.ExtensionID] = member
	r.mu.Unlock()
	return nil
}

// Disable marks a member disabled without loading code (CLI recovery path).
func (r *Registry) Disable(extensionID string) error {
	if r == nil {
		return errInvalid
	}
	extensionID = strings.ToLower(strings.TrimSpace(extensionID))
	r.mu.Lock()
	defer r.mu.Unlock()
	member, ok := r.members[extensionID]
	if !ok {
		return errNotFound
	}
	member.Enabled = false
	r.members[extensionID] = member
	return nil
}

// LoadOrder returns enabled members sorted by priority then id.
// When safeMode is true, returns nil (tier always bypassed).
func (r *Registry) LoadOrder(safeMode bool) []Member {
	if r == nil || safeMode {
		return nil
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	out := make([]Member, 0, len(r.members))
	for _, member := range r.members {
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
	return out
}

// Snapshot returns full membership including disabled.
func (r *Registry) Snapshot() Snapshot {
	if r == nil {
		return Snapshot{SchemaVersion: SchemaVersion, SafeModeBypass: true}
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	out := make([]Member, 0, len(r.members))
	for _, member := range r.members {
		out = append(out, member)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ExtensionID < out[j].ExtensionID })
	return Snapshot{SchemaVersion: SchemaVersion, SafeModeBypass: true, Members: out}
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
	errInvalid  = errString("system tier input is invalid")
	errNotFound = errString("system tier member is not found")
)

type errString string

func (e errString) Error() string { return string(e) }
