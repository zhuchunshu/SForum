// Package marketplace is the Host signed marketplace index for V3 P12.
// Indexes are verified offline; install still uses existing direct-upload
// and staged rollout paths.
package marketplace

import (
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
)

var (
	ErrInvalid      = errors.New("marketplace input is invalid")
	ErrNotFound     = errors.New("marketplace entry is not found")
	ErrSignature    = errors.New("marketplace signature is invalid")
	ErrStale        = errors.New("marketplace index is stale")
	ErrIncompatible = errors.New("marketplace package is incompatible")
	ErrWithdrawn    = errors.New("marketplace release is withdrawn")
	ErrPolicy       = errors.New("marketplace operator policy denied")
)

// Index is a signed catalog snapshot.
type Index struct {
	SchemaVersion string    `json:"schemaVersion"`
	GeneratedAt   time.Time `json:"generatedAt"`
	// ExpiresAt marks staleness for offline operators.
	ExpiresAt time.Time `json:"expiresAt"`
	// Entries are package releases.
	Entries []Entry `json:"entries"`
	// Signature is hex-encoded HMAC/Ed25519 over canonical body (Host verifies).
	Signature string `json:"signature"`
	// SignerID identifies the key (not the secret).
	SignerID string `json:"signerId"`
}

// Entry is one marketplace package release.
type Entry struct {
	ExtensionID      string   `json:"extensionId"`
	Version          string   `json:"version"`
	PackageDigest    string   `json:"packageDigest"`
	Channel          string   `json:"channel"`
	MinSForumVersion string   `json:"minSForumVersion,omitempty"`
	Dependencies     []string `json:"dependencies,omitempty"`
	SBOMDigest       string   `json:"sbomDigest,omitempty"`
	Provenance       string   `json:"provenance,omitempty"`
	// Withdrawn hides the release from install while retaining audit.
	Withdrawn bool     `json:"withdrawn,omitempty"`
	Notices   []Notice `json:"notices,omitempty"`
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
	AllowedChannels      []string `json:"allowedChannels"`
	// MaxVulnerabilitySeverity blocks installs above this (empty = allow all).
	MaxVulnerabilitySeverity string `json:"maxVulnerabilitySeverity,omitempty"`
	// AllowUnsigned is false in production.
	AllowUnsigned bool `json:"allowUnsigned,omitempty"`
	// DirectUploadFallback keeps offline install available.
	DirectUploadFallback bool `json:"directUploadFallback"`
}

// ResolveResult is a dependency-resolved install plan (no network download).
type ResolveResult struct {
	ExtensionID   string   `json:"extensionId"`
	Version       string   `json:"version"`
	PackageDigest string   `json:"packageDigest"`
	Channel       string   `json:"channel"`
	// Order is install order (dependencies first).
	Order []string `json:"order"`
}
