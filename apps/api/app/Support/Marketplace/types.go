// Package marketplace is the Host signed marketplace index for V3 P12.
// Indexes are verified offline; install uses staged preflight + rollout
// bindings rather than returning string lists alone.
package marketplace

import (
	"context"
	"errors"
	"time"
)

// SchemaVersion is the marketplace index contract.
const SchemaVersion = "sforum.marketplace-index@1"

const (
	ChannelStable = "stable"
	ChannelBeta   = "beta"
	ChannelDev    = "dev"

	NoticeVulnerability = "vulnerability"
	NoticeRevocation    = "revocation"
	NoticeWithdrawn     = "withdrawn"

	// SignerKindEd25519 is the production signing algorithm (raw 64-byte sig, hex).
	SignerKindEd25519 = "ed25519"
)

var (
	ErrInvalid      = errors.New("marketplace input is invalid")
	ErrNotFound     = errors.New("marketplace entry is not found")
	ErrSignature    = errors.New("marketplace signature is invalid")
	ErrStale        = errors.New("marketplace index is stale")
	ErrIncompatible = errors.New("marketplace package is incompatible")
	ErrWithdrawn    = errors.New("marketplace release is withdrawn")
	ErrPolicy       = errors.New("marketplace operator policy denied")
	ErrCycle        = errors.New("marketplace dependency cycle")
	ErrConflict     = errors.New("marketplace dependency conflict")
	ErrDigest       = errors.New("marketplace digest is invalid")
	ErrInstall      = errors.New("marketplace install binding failed")
)

// Index is a signed catalog snapshot.
type Index struct {
	SchemaVersion string    `json:"schemaVersion"`
	GeneratedAt   time.Time `json:"generatedAt"`
	// ExpiresAt marks staleness for offline operators.
	ExpiresAt time.Time `json:"expiresAt"`
	// NotBefore is the earliest acceptance instant (time window).
	NotBefore time.Time `json:"notBefore,omitempty"`
	// Entries are package releases.
	Entries []Entry `json:"entries"`
	// Signature is hex-encoded Ed25519 signature over canonical body.
	Signature string `json:"signature"`
	// SignerID identifies the public key (not the private material).
	SignerID string `json:"signerId"`
	// SignerKind is ed25519 (required in production).
	SignerKind string `json:"signerKind,omitempty"`
}

// DependencyConstraint is one required/optional package dependency with SemVer range.
type DependencyConstraint struct {
	// ExtensionID is the required package id (manifest id pattern).
	ExtensionID string `json:"extensionId"`
	// Version is a Masterminds/semver constraint (e.g. ^1.2.0).
	Version string `json:"version"`
	// Optional when true does not fail resolve if missing.
	Optional bool `json:"optional,omitempty"`
}

// Entry is one marketplace package release.
type Entry struct {
	ExtensionID string `json:"extensionId"`
	Version     string `json:"version"`
	// PackageDigest is lowercase SHA-256 hex (64 chars).
	PackageDigest string `json:"packageDigest"`
	Channel       string `json:"channel"`
	// MinSForumVersion is a SemVer constraint against the Host SForum version.
	MinSForumVersion string `json:"minSForumVersion,omitempty"`
	// MaxSForumVersion optional upper bound constraint.
	MaxSForumVersion string `json:"maxSForumVersion,omitempty"`
	// Dependencies uses structured constraints (preferred).
	Dependencies []DependencyConstraint `json:"dependencies,omitempty"`
	// SBOMDigest is optional lowercase SHA-256 of the SBOM artifact.
	SBOMDigest string `json:"sbomDigest,omitempty"`
	Provenance string `json:"provenance,omitempty"`
	// Withdrawn hides the release from install while retaining audit.
	Withdrawn bool     `json:"withdrawn,omitempty"`
	Notices   []Notice `json:"notices,omitempty"`
	// AvailableFrom/Until optional release time window.
	AvailableFrom  time.Time `json:"availableFrom,omitempty"`
	AvailableUntil time.Time `json:"availableUntil,omitempty"`
}

// Notice is a security or operator notice bound to a release.
type Notice struct {
	Kind    string `json:"kind"`
	Summary string `json:"summary"`
	// Severity is low|medium|high|critical for vulnerabilities.
	Severity string `json:"severity,omitempty"`
}

// OperatorPolicy controls which channels and severities are installable.
type OperatorPolicy struct {
	AllowedChannels []string `json:"allowedChannels"`
	// MaxVulnerabilitySeverity blocks installs above this (empty = allow all except revocation).
	MaxVulnerabilitySeverity string `json:"maxVulnerabilitySeverity,omitempty"`
	// AllowUnsigned is false in production.
	AllowUnsigned bool `json:"allowUnsigned,omitempty"`
	// DirectUploadFallback keeps offline install available.
	DirectUploadFallback bool `json:"directUploadFallback"`
	// HostSForumVersion is the running Host version for compatibility checks.
	HostSForumVersion string `json:"hostSForumVersion,omitempty"`
}

// ResolveResult is a dependency-resolved install plan with digests (not string ids only).
type ResolveResult struct {
	ExtensionID   string `json:"extensionId"`
	Version       string `json:"version"`
	PackageDigest string `json:"packageDigest"`
	Channel       string `json:"channel"`
	SBOMDigest    string `json:"sbomDigest,omitempty"`
	// Order is install order (dependencies first) as exact PlanSteps.
	Order []PlanStep `json:"order"`
	// Report is a human/operator compatibility summary.
	Report CompatibilityReport `json:"report"`
}

// PlanStep is one exact artifact in the install plan.
type PlanStep struct {
	ExtensionID   string `json:"extensionId"`
	Version       string `json:"version"`
	PackageDigest string `json:"packageDigest"`
	Channel       string `json:"channel"`
	SBOMDigest    string `json:"sbomDigest,omitempty"`
}

// CompatibilityReport summarizes preflight checks for operators.
type CompatibilityReport struct {
	Compatible bool     `json:"compatible"`
	Warnings   []string `json:"warnings,omitempty"`
	BlockedBy  []string `json:"blockedBy,omitempty"`
}

// InstallPlan is the durable-facing plan after Resolve (download/stage/activate/rollback).
type InstallPlan struct {
	ResolveResult
	// Action is stage|activate|rollback.
	Action string `json:"action"`
	// SourceDigest is the currently active digest when upgrading (for rollback).
	SourceDigest string `json:"sourceDigest,omitempty"`
	Actor        string `json:"actor,omitempty"`
}

// StageResult is returned after preflight + staged package registration.
type StageResult struct {
	ExtensionID     string   `json:"extensionId"`
	StagedDigest    string   `json:"stagedDigest"`
	StagedVersion   string   `json:"stagedVersion"`
	DependencyOrder []string `json:"dependencyOrder"`
	// RolloutPlanID is set when a RuntimeRollout plan was created.
	RolloutPlanID string `json:"rolloutPlanId,omitempty"`
	// PreflightOK is always true on success.
	PreflightOK bool `json:"preflightOk"`
}

// Installer binds marketplace resolve to real Host install paths.
// Production wires staged upload preflight + RuntimeRollout; tests use fakes.
type Installer interface {
	// Preflight validates digests/SBOM/policy against the Host before download.
	Preflight(ctx context.Context, plan InstallPlan) error
	// Stage registers the exact package as a staged candidate (no runtime switch).
	Stage(ctx context.Context, plan InstallPlan, packageBytes []byte) (StageResult, error)
	// Activate promotes a staged candidate via rollout authority.
	Activate(ctx context.Context, plan InstallPlan, staged StageResult) error
	// Rollback reverts desired pointer to source digest (one-click).
	Rollback(ctx context.Context, plan InstallPlan, reason string) error
}

// Verifier verifies index signatures with a public key.
type Verifier interface {
	// PublicKeyID returns the SignerID this key is known as.
	PublicKeyID() string
	// Verify returns nil when signature matches canonical body.
	Verify(canonicalBody []byte, signatureHex string) error
}
