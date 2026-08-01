package identitydelegation

import (
	"context"
	"errors"
	"time"
)

var (
	ErrInvalid          = errors.New("identity delegation: invalid input")
	ErrActorUnavailable = errors.New("identity delegation: actor unavailable")
	ErrSubjectNotFound  = errors.New("identity delegation: subject not found")
	ErrArtifactStale    = errors.New("identity delegation: artifact is stale")
)

const (
	ActorStatusActive = "active"
	SubjectTTL        = 15 * time.Minute
)

// ArtifactBinding is the minimum exact-artifact identity needed to bind a
// delegated subject. It intentionally contains no runtime capability token.
type ArtifactBinding struct {
	ExtensionID      string
	ExtensionVersion string
	PackageDigest    string
	ImpactDigest     string
}

// ActorProjectionInput is Host-side data. UserID is accepted only by the
// Host store and is never copied into DelegatedIdentity or plugin responses.
type ActorProjectionInput struct {
	UserID        int64
	Status        string
	Username      string
	DisplayName   string
	Locale        string
	Email         string
	EmailVerified bool
}

// DelegatedIdentity is the only identity projection a plugin may receive.
// Claims are deliberately fixed and do not contain roles, permissions, or
// Core identifiers beyond the opaque subject.
type DelegatedIdentity struct {
	Subject       string
	Username      string
	DisplayName   string
	Locale        string
	Email         string
	EmailVerified bool
	AuthTime      time.Time
	ExpiresAt     time.Time
}

type SubjectRecord struct {
	Binding     ArtifactBinding
	Subject     string
	ActorUserID int64
	Identity    DelegatedIdentity
	CreatedAt   time.Time
	ExpiresAt   time.Time
	RevokedAt   *time.Time
}

type SubjectStore interface {
	Put(context.Context, SubjectRecord) error
	Get(context.Context, ArtifactBinding, string) (SubjectRecord, error)
}

type Clock interface {
	Now() time.Time
}
