package consentbridge

import (
	"context"
	"errors"
	"time"
)

var (
	ErrInvalid          = errors.New("consent bridge: invalid input")
	ErrExpired          = errors.New("consent bridge: transaction expired")
	ErrReplay           = errors.New("consent bridge: transaction already used")
	ErrActorMismatch    = errors.New("consent bridge: actor mismatch")
	ErrSessionMismatch  = errors.New("consent bridge: session mismatch")
	ErrArtifactMismatch = errors.New("consent bridge: artifact mismatch")
	ErrRecentAuth       = errors.New("consent bridge: recent authentication required")
	ErrCSRF             = errors.New("consent bridge: csrf mismatch")
)

const TransactionTTL = 10 * time.Minute

type ArtifactBinding struct {
	ExtensionID      string
	ExtensionVersion string
	PackageDigest    string
	ImpactDigest     string
}

type ClientDescriptor struct {
	ClientID string
	Name     string
	LogoURL  string
}

type BeginInput struct {
	ActorUserID        int64
	SessionFingerprint string
	Binding            ArtifactBinding
	Client             ClientDescriptor
	RedirectURI        string
	Scopes             []string
	RequireRecentAuth  bool
	RecentAuth         bool
}

// ConsentFrame is Host UI data. It deliberately contains neither the actor
// ID nor the session fingerprint.
type ConsentFrame struct {
	TransactionID string           `json:"transactionId"`
	CSRFToken     string           `json:"csrfToken"`
	Client        ClientDescriptor `json:"client"`
	RedirectURI   string           `json:"redirectUri"`
	Scopes        []string         `json:"scopes"`
	ExpiresAt     time.Time        `json:"expiresAt"`
}

type TransactionRecord struct {
	Frame              ConsentFrame
	ActorUserID        int64
	SessionFingerprint string
	Binding            ArtifactBinding
	RequireRecentAuth  bool
	UsedAt             *time.Time
}

type Decision struct {
	Approved bool
	Scopes   []string
}

type Store interface {
	Put(context.Context, TransactionRecord) error
	Get(context.Context, string) (TransactionRecord, error)
	Consume(context.Context, string, time.Time) error
}

type Clock interface{ Now() time.Time }
