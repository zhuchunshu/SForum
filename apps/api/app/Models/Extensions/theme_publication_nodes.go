package extensions

import (
	"context"
	"errors"
	"strings"
	"time"
	"unicode/utf8"
)

const (
	ThemeRuntimeAckApplying = "applying"
	ThemeRuntimeAckApplied  = "applied"
	ThemeRuntimeAckFailed   = "failed"
)

var (
	ErrThemeRuntimeNodeInvalid   = errors.New("extensions: invalid theme runtime node identity")
	ErrThemeRuntimeNodeLeaseLost = errors.New("extensions: theme runtime node lease lost")
	ErrThemeRuntimeAckConflict   = errors.New("extensions: theme runtime acknowledgement conflict")
)

type ThemeRuntimeNodeIdentity struct {
	NodeID string `json:"nodeId"`
	BootID string `json:"bootId"`
}

type ThemeRuntimeNode struct {
	ThemeRuntimeNodeIdentity
	LastAppliedRevision int64     `json:"lastAppliedRevision"`
	FirstSeenAt         time.Time `json:"firstSeenAt"`
	LastSeenAt          time.Time `json:"lastSeenAt"`
	LeaseExpiresAt      time.Time `json:"leaseExpiresAt"`
}

type ThemeRuntimePublicationAck struct {
	PublicationRevision int64 `json:"publicationRevision"`
	ThemeRuntimeNodeIdentity
	Status               string                       `json:"status"`
	AppliedState         ThemeRuntimePublicationState `json:"appliedState,omitempty"`
	AppliedThemeID       string                       `json:"appliedThemeId,omitempty"`
	AppliedThemeVersion  string                       `json:"appliedThemeVersion,omitempty"`
	AppliedPackageDigest string                       `json:"appliedPackageDigest,omitempty"`
	ErrorReason          string                       `json:"errorReason,omitempty"`
	AttemptCount         int                          `json:"attemptCount"`
	Revision             int64                        `json:"revision"`
	StartedAt            time.Time                    `json:"startedAt"`
	UpdatedAt            time.Time                    `json:"updatedAt"`
	AppliedAt            *time.Time                   `json:"appliedAt,omitempty"`
}

type ThemeRuntimeNodeRepository interface {
	RegisterThemeRuntimeNode(context.Context, ThemeRuntimeNodeIdentity, time.Duration) (ThemeRuntimeNode, error)
	HeartbeatThemeRuntimeNode(context.Context, ThemeRuntimeNodeIdentity, time.Duration) (ThemeRuntimeNode, error)
	GetThemeRuntimeNode(context.Context, ThemeRuntimeNodeIdentity) (ThemeRuntimeNode, error)
	BeginThemeRuntimePublicationApply(context.Context, ThemeRuntimeNodeIdentity, int64) (ThemeRuntimePublicationAck, error)
	CompleteThemeRuntimePublicationApply(context.Context, ThemeRuntimeNodeIdentity, ThemeRuntimePublication, int64) (ThemeRuntimePublicationAck, error)
	FailThemeRuntimePublicationApply(context.Context, ThemeRuntimeNodeIdentity, int64, int64, string) (ThemeRuntimePublicationAck, error)
}

func validThemeRuntimeNodeIdentity(identity ThemeRuntimeNodeIdentity) bool {
	return validThemeRuntimeNodeIdentityPart(identity.NodeID) && validThemeRuntimeNodeIdentityPart(identity.BootID)
}

func validThemeRuntimeNodeIdentityPart(value string) bool {
	return value != "" && value == strings.TrimSpace(value) && utf8.ValidString(value) && len([]byte(value)) <= 128
}
