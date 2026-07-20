// Package extensionobservability attributes Host latency, errors, queries,
// cache, memory, and fallbacks to exact extension artifacts for V3 P12.
package extensionobservability

import (
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

// SchemaVersion is the attribution snapshot contract.
const SchemaVersion = "sforum.extension-observability@1"

// Event is one attributed observation (never includes secrets or bodies).
type Event struct {
	ExtensionID   string        `json:"extensionId"`
	PackageDigest string        `json:"packageDigest,omitempty"`
	Surface       string        `json:"surface"` // route|hook|query|cache|job|rpc|media|seo
	Name          string        `json:"name,omitempty"`
	Duration      time.Duration `json:"duration,omitempty"`
	ErrorClass    string        `json:"errorClass,omitempty"`
	Fallback      bool          `json:"fallback,omitempty"`
	At            time.Time     `json:"at"`
}

// Aggregate is per-extension rolled-up metrics.
type Aggregate struct {
	ExtensionID   string        `json:"extensionId"`
	PackageDigest string        `json:"packageDigest,omitempty"`
	Events        uint64        `json:"events"`
	Errors        uint64        `json:"errors"`
	Fallbacks     uint64        `json:"fallbacks"`
	TotalLatency  time.Duration `json:"totalLatency"`
	AvgLatency    time.Duration `json:"avgLatency"`
	BySurface     map[string]uint64 `json:"bySurface,omitempty"`
}

// Snapshot is the operator-facing attribution view.
type Snapshot struct {
	SchemaVersion string      `json:"schemaVersion"`
	Aggregates    []Aggregate `json:"aggregates"`
	Recent        []Event     `json:"recent,omitempty"`
}

// Recorder is a process-local attribution ring.
type Recorder struct {
	mu        sync.Mutex
	events    []Event
	maxRecent int
	// aggregates keyed by extensionID\x00digest
	agg map[string]*aggState
}

type aggState struct {
	extensionID   string
	packageDigest string
	events        atomic.Uint64
	errors        atomic.Uint64
	fallbacks     atomic.Uint64
	totalNanos    atomic.Uint64
	bySurface     map[string]*atomic.Uint64
	surfaceMu     sync.Mutex
}

// New builds a recorder with a bounded recent ring.
func New(maxRecent int) *Recorder {
	if maxRecent <= 0 {
		maxRecent = 256
	}
	return &Recorder{maxRecent: maxRecent, agg: make(map[string]*aggState)}
}

// Observe records an attributed event.
func (r *Recorder) Observe(event Event) {
	if r == nil {
		return
	}
	event.ExtensionID = strings.ToLower(strings.TrimSpace(event.ExtensionID))
	event.PackageDigest = strings.ToLower(strings.TrimSpace(event.PackageDigest))
	event.Surface = strings.ToLower(strings.TrimSpace(event.Surface))
	if event.ExtensionID == "" || event.Surface == "" {
		return
	}
	if event.At.IsZero() {
		event.At = time.Now().UTC()
	}
	key := event.ExtensionID + "\x00" + event.PackageDigest
	r.mu.Lock()
	state, ok := r.agg[key]
	if !ok {
		state = &aggState{
			extensionID: event.ExtensionID, packageDigest: event.PackageDigest,
			bySurface: make(map[string]*atomic.Uint64),
		}
		r.agg[key] = state
	}
	r.events = append(r.events, event)
	if len(r.events) > r.maxRecent {
		r.events = append([]Event(nil), r.events[len(r.events)-r.maxRecent:]...)
	}
	r.mu.Unlock()

	state.events.Add(1)
	if event.ErrorClass != "" {
		state.errors.Add(1)
	}
	if event.Fallback {
		state.fallbacks.Add(1)
	}
	if event.Duration > 0 {
		state.totalNanos.Add(uint64(event.Duration.Nanoseconds()))
	}
	state.surfaceMu.Lock()
	counter, ok := state.bySurface[event.Surface]
	if !ok {
		counter = &atomic.Uint64{}
		state.bySurface[event.Surface] = counter
	}
	counter.Add(1)
	state.surfaceMu.Unlock()
}

// Snapshot returns aggregates and recent events.
func (r *Recorder) Snapshot() Snapshot {
	if r == nil {
		return Snapshot{SchemaVersion: SchemaVersion}
	}
	r.mu.Lock()
	recent := append([]Event(nil), r.events...)
	states := make([]*aggState, 0, len(r.agg))
	for _, state := range r.agg {
		states = append(states, state)
	}
	r.mu.Unlock()

	out := make([]Aggregate, 0, len(states))
	for _, state := range states {
		events := state.events.Load()
		total := time.Duration(state.totalNanos.Load())
		var avg time.Duration
		if events > 0 {
			avg = total / time.Duration(events)
		}
		bySurface := map[string]uint64{}
		state.surfaceMu.Lock()
		for surface, counter := range state.bySurface {
			bySurface[surface] = counter.Load()
		}
		state.surfaceMu.Unlock()
		out = append(out, Aggregate{
			ExtensionID: state.extensionID, PackageDigest: state.packageDigest,
			Events: events, Errors: state.errors.Load(), Fallbacks: state.fallbacks.Load(),
			TotalLatency: total, AvgLatency: avg, BySurface: bySurface,
		})
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].ExtensionID != out[j].ExtensionID {
			return out[i].ExtensionID < out[j].ExtensionID
		}
		return out[i].PackageDigest < out[j].PackageDigest
	})
	return Snapshot{SchemaVersion: SchemaVersion, Aggregates: out, Recent: recent}
}
