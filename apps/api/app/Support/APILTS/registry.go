// Package apilts publishes Host/Frontend API LTS windows, deprecation periods,
// and shim usage telemetry for V3 P12.
package apilts

import (
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

// SchemaVersion is the LTS registry contract.
const SchemaVersion = "sforum.api-lts@1"

// DefaultMinDeprecation is the minimum deprecation period before removal.
const DefaultMinDeprecation = 180 * 24 * time.Hour // 180 days

// Contract describes one versioned Host or Frontend surface.
type Contract struct {
	// ID e.g. sforum.host.v2.CacheService
	ID string `json:"id"`
	// Kind is host|frontend|protocol|manifest
	Kind string `json:"kind"`
	// Status is current|deprecated|removed
	Status string `json:"status"`
	// Introduced is when the contract became supported.
	Introduced time.Time `json:"introduced,omitempty"`
	// DeprecatedAt starts the minimum deprecation clock.
	DeprecatedAt time.Time `json:"deprecatedAt,omitempty"`
	// RemoveAfter is the earliest allowed removal instant.
	RemoveAfter time.Time `json:"removeAfter,omitempty"`
	// Replacement is the successor contract id.
	Replacement string `json:"replacement,omitempty"`
	// ShimEnabled records whether a compatibility shim is active.
	ShimEnabled bool `json:"shimEnabled,omitempty"`
}

// Snapshot is the operator/developer LTS view.
type Snapshot struct {
	SchemaVersion     string     `json:"schemaVersion"`
	MinDeprecation    string     `json:"minDeprecation"`
	Contracts         []Contract `json:"contracts"`
	ShimUsage         []ShimStat `json:"shimUsage,omitempty"`
}

// ShimStat is deprecation telemetry for a shimmed contract.
type ShimStat struct {
	ContractID string `json:"contractId"`
	Calls      uint64 `json:"calls"`
}

// Registry is the process-local LTS + telemetry store.
type Registry struct {
	mu        sync.Mutex
	contracts map[string]Contract
	shimCalls map[string]*atomic.Uint64
}

// New builds a registry with recommended Host contracts pre-seeded.
func New() *Registry {
	r := &Registry{
		contracts: make(map[string]Contract),
		shimCalls: make(map[string]*atomic.Uint64),
	}
	// Seed current Host/protocol surfaces (stable; not exhaustive).
	now := time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC)
	for _, c := range []Contract{
		{ID: "sforum.host.v2", Kind: "host", Status: "current", Introduced: now},
		{ID: "sforum.protocol.v2", Kind: "protocol", Status: "current", Introduced: now},
		{ID: "sforum.manifest.v3", Kind: "manifest", Status: "current", Introduced: now},
		{
			ID: "sforum.protocol.v1", Kind: "protocol", Status: "deprecated",
			Introduced: now.Add(-365 * 24 * time.Hour),
			DeprecatedAt: now.Add(-30 * 24 * time.Hour),
			RemoveAfter:  now.Add(-30*24*time.Hour + DefaultMinDeprecation),
			Replacement:  "sforum.protocol.v2", ShimEnabled: true,
		},
	} {
		_ = r.Register(c)
	}
	return r
}

// Register upserts a contract. Removal before RemoveAfter is rejected.
func (r *Registry) Register(c Contract) error {
	if r == nil {
		return errInvalid
	}
	c.ID = strings.TrimSpace(c.ID)
	c.Kind = strings.ToLower(strings.TrimSpace(c.Kind))
	c.Status = strings.ToLower(strings.TrimSpace(c.Status))
	if c.ID == "" || c.Kind == "" || c.Status == "" {
		return errInvalid
	}
	if c.Status == "deprecated" && c.DeprecatedAt.IsZero() {
		c.DeprecatedAt = time.Now().UTC()
	}
	if c.Status == "deprecated" && c.RemoveAfter.IsZero() {
		c.RemoveAfter = c.DeprecatedAt.Add(DefaultMinDeprecation)
	}
	if c.Status == "removed" {
		if c.RemoveAfter.IsZero() || time.Now().UTC().Before(c.RemoveAfter) {
			return errTooEarly
		}
	}
	r.mu.Lock()
	r.contracts[c.ID] = c
	if c.ShimEnabled {
		if _, ok := r.shimCalls[c.ID]; !ok {
			r.shimCalls[c.ID] = &atomic.Uint64{}
		}
	}
	r.mu.Unlock()
	return nil
}

// RecordShimCall increments deprecation telemetry for a shimmed contract.
func (r *Registry) RecordShimCall(contractID string) {
	if r == nil {
		return
	}
	contractID = strings.TrimSpace(contractID)
	r.mu.Lock()
	counter, ok := r.shimCalls[contractID]
	if !ok {
		counter = &atomic.Uint64{}
		r.shimCalls[contractID] = counter
	}
	r.mu.Unlock()
	counter.Add(1)
}

// Snapshot returns contracts and shim usage.
func (r *Registry) Snapshot() Snapshot {
	if r == nil {
		return Snapshot{SchemaVersion: SchemaVersion, MinDeprecation: DefaultMinDeprecation.String()}
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	contracts := make([]Contract, 0, len(r.contracts))
	for _, c := range r.contracts {
		contracts = append(contracts, c)
	}
	sort.Slice(contracts, func(i, j int) bool { return contracts[i].ID < contracts[j].ID })
	usage := make([]ShimStat, 0, len(r.shimCalls))
	for id, counter := range r.shimCalls {
		usage = append(usage, ShimStat{ContractID: id, Calls: counter.Load()})
	}
	sort.Slice(usage, func(i, j int) bool { return usage[i].ContractID < usage[j].ContractID })
	return Snapshot{
		SchemaVersion: SchemaVersion, MinDeprecation: DefaultMinDeprecation.String(),
		Contracts: contracts, ShimUsage: usage,
	}
}

// CanRemove reports whether a deprecated contract may be removed now.
func (r *Registry) CanRemove(contractID string, now time.Time) bool {
	if r == nil {
		return false
	}
	if now.IsZero() {
		now = time.Now().UTC()
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	c, ok := r.contracts[strings.TrimSpace(contractID)]
	if !ok || c.Status != "deprecated" {
		return false
	}
	return !c.RemoveAfter.IsZero() && !now.Before(c.RemoveAfter)
}

var (
	errInvalid  = errString("api lts input is invalid")
	errTooEarly = errString("api lts removal before minimum deprecation period")
)

type errString string

func (e errString) Error() string { return string(e) }
