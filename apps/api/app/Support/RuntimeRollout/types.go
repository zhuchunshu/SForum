// Package runtimerollout is the Host multi-node staged/canary activation
// coordinator for V3 P12. It gates promote on migration-once proofs, health,
// drain, and atomic snapshot switches with rollback and artifact retention.
package runtimerollout

import (
	"errors"
	"time"
)

// SchemaVersion is the Host rollout contract identity.
const SchemaVersion = "sforum.runtime-rollout@1"

const (
	PhasePending    = "pending"
	PhaseMigrating  = "migrating"
	PhaseStaged     = "staged"
	PhaseCanary     = "canary"
	PhaseDraining   = "draining"
	PhasePromoting  = "promoting"
	PhaseActive     = "active"
	PhaseRollingBack = "rolling_back"
	PhaseFailed     = "failed"
	PhaseRolledBack = "rolled_back"

	HealthUnknown = "unknown"
	HealthHealthy = "healthy"
	HealthUnhealthy = "unhealthy"

	DefaultCanaryPercent = 10
	DefaultRetainVersions = 3
)

var (
	ErrInvalid          = errors.New("runtime rollout input is invalid")
	ErrNotFound         = errors.New("runtime rollout plan is not found")
	ErrPermissionDenied = errors.New("runtime rollout permission denied")
	ErrPhase            = errors.New("runtime rollout phase transition denied")
	ErrHealthGate       = errors.New("runtime rollout health gate failed")
	ErrMigration        = errors.New("runtime rollout migration not ready")
	ErrCanary           = errors.New("runtime rollout canary cohort failed")
	// ErrConflict is returned when multi-API concurrent Create races; one winner.
	ErrConflict = errors.New("runtime rollout plan conflict")
)

// Plan is one extension rollout across the cluster.
type Plan struct {
	SchemaVersion string `json:"schemaVersion"`
	PlanID        string `json:"planId"`
	ExtensionID   string `json:"extensionId"`
	// SourceDigest is the currently active package digest.
	SourceDigest string `json:"sourceDigest"`
	// TargetDigest is the staged package digest to promote.
	TargetDigest string `json:"targetDigest"`
	// MigrationReady is set only after Host migration-once proof succeeds.
	MigrationReady bool `json:"migrationReady"`
	// CanaryPercent is 1-100 of admitted nodes in the first promote wave.
	CanaryPercent int `json:"canaryPercent"`
	Phase         string `json:"phase"`
	// SnapshotID is the atomic multi-registry snapshot after promote.
	SnapshotID string `json:"snapshotId,omitempty"`
	// RetainVersions is how many old package digests to keep for rollback.
	RetainVersions int       `json:"retainVersions"`
	Actor          string    `json:"actor,omitempty"`
	Reason         string    `json:"reason,omitempty"`
	UpdatedAt      time.Time `json:"updatedAt"`
	// NodeAcks maps nodeID -> ack phase for that node.
	NodeAcks map[string]NodeAck `json:"nodeAcks,omitempty"`
	// RetainedDigests lists old artifacts kept for rollback/GC.
	RetainedDigests []string `json:"retainedDigests,omitempty"`
	// LastError is a redacted operator message (never secrets).
	LastError string `json:"lastError,omitempty"`
}

// NodeAck is one node's acknowledgement of a rollout phase.
type NodeAck struct {
	NodeID    string    `json:"nodeId"`
	Phase     string    `json:"phase"`
	Health    string    `json:"health"`
	At        time.Time `json:"at"`
	// Canary marks membership in the canary cohort.
	Canary bool `json:"canary,omitempty"`
}

// HealthReport is submitted by a node for the health gate.
type HealthReport struct {
	NodeID string
	Health string
	// Detail is optional operator-safe text.
	Detail string
}
