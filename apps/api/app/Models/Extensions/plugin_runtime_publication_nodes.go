package extensions

import (
	"context"
	"errors"
	"sort"
	"strings"
	"time"
	"unicode/utf8"
)

type PluginRuntimeProcessRole string

const (
	PluginRuntimeProcessAPI    PluginRuntimeProcessRole = "api"
	PluginRuntimeProcessWorker PluginRuntimeProcessRole = "worker"

	PluginRuntimeAckApplying = "applying"
	PluginRuntimeAckApplied  = "applied"
	PluginRuntimeAckFailed   = "failed"

	maxPluginRuntimeNodeLease = 24 * time.Hour
)

var (
	ErrPluginRuntimeNodeInvalid   = errors.New("extensions: invalid plugin runtime node identity")
	ErrPluginRuntimeNodeLeaseLost = errors.New("extensions: plugin runtime node lease lost")
	ErrPluginRuntimeAckConflict   = errors.New("extensions: plugin runtime acknowledgement conflict")
)

type PluginRuntimeNodeIdentity struct {
	NodeID      string                   `json:"nodeId"`
	ProcessRole PluginRuntimeProcessRole `json:"processRole"`
	BootID      string                   `json:"bootId"`
}

type PluginRuntimeNode struct {
	PluginRuntimeNodeIdentity
	LastAppliedRevision int64     `json:"lastAppliedRevision"`
	FirstSeenAt         time.Time `json:"firstSeenAt"`
	LastSeenAt          time.Time `json:"lastSeenAt"`
	LeaseExpiresAt      time.Time `json:"leaseExpiresAt"`
}

type PluginRuntimeAppliedMember struct {
	PluginRuntimeMember
	RuntimeInstanceID string `json:"runtimeInstanceId"`
}

type PluginRuntimePublicationAck struct {
	PublicationRevision int64 `json:"publicationRevision"`
	PluginRuntimeNodeIdentity
	Status               string     `json:"status"`
	AppliedMemberCount   *int       `json:"appliedMemberCount,omitempty"`
	AppliedMembersDigest string     `json:"appliedMembersDigest,omitempty"`
	ErrorReason          string     `json:"errorReason,omitempty"`
	AttemptCount         int        `json:"attemptCount"`
	Revision             int64      `json:"revision"`
	StartedAt            time.Time  `json:"startedAt"`
	UpdatedAt            time.Time  `json:"updatedAt"`
	AppliedAt            *time.Time `json:"appliedAt,omitempty"`
}

type PluginRuntimeNodeRepository interface {
	RegisterPluginRuntimeNode(context.Context, PluginRuntimeNodeIdentity, time.Duration) (PluginRuntimeNode, error)
	HeartbeatPluginRuntimeNode(context.Context, PluginRuntimeNodeIdentity, time.Duration) (PluginRuntimeNode, error)
	GetPluginRuntimeNode(context.Context, PluginRuntimeNodeIdentity) (PluginRuntimeNode, error)
	BeginPluginRuntimePublicationApply(context.Context, PluginRuntimeNodeIdentity, int64) (PluginRuntimePublicationAck, error)
	CompletePluginRuntimePublicationApply(
		context.Context,
		PluginRuntimeNodeIdentity,
		PluginRuntimePublication,
		int64,
		[]PluginRuntimeAppliedMember,
	) (PluginRuntimePublicationAck, error)
	FailPluginRuntimePublicationApply(
		context.Context,
		PluginRuntimeNodeIdentity,
		int64,
		int64,
		string,
	) (PluginRuntimePublicationAck, error)
}

func validPluginRuntimeNodeIdentity(identity PluginRuntimeNodeIdentity) bool {
	return validPluginRuntimeNodeIdentityPart(identity.NodeID) &&
		validPluginRuntimeNodeIdentityPart(identity.BootID) &&
		(identity.ProcessRole == PluginRuntimeProcessAPI || identity.ProcessRole == PluginRuntimeProcessWorker)
}

func validPluginRuntimeNodeIdentityPart(value string) bool {
	return value != "" && value == strings.TrimSpace(value) && utf8.ValidString(value) && len([]byte(value)) <= 128
}

func canonicalPluginRuntimeAppliedMembers(
	desired []PluginRuntimeMember,
	applied []PluginRuntimeAppliedMember,
) ([]PluginRuntimeAppliedMember, string, error) {
	canonicalDesired, digest, err := canonicalPluginRuntimeMembers(desired)
	if err != nil || len(canonicalDesired) != len(applied) {
		return nil, "", ErrPluginRuntimeAckConflict
	}
	canonicalApplied := append([]PluginRuntimeAppliedMember(nil), applied...)
	for _, member := range canonicalApplied {
		if !validPluginRuntimeMember(member.PluginRuntimeMember) ||
			member.RuntimeInstanceID == "" || member.RuntimeInstanceID != strings.TrimSpace(member.RuntimeInstanceID) ||
			!utf8.ValidString(member.RuntimeInstanceID) || len([]byte(member.RuntimeInstanceID)) > 512 {
			return nil, "", ErrPluginRuntimeAckConflict
		}
	}
	sortPluginRuntimeAppliedMembers(canonicalApplied)
	for index := range canonicalDesired {
		if canonicalApplied[index].PluginRuntimeMember != canonicalDesired[index] {
			return nil, "", ErrPluginRuntimeAckConflict
		}
	}
	return canonicalApplied, digest, nil
}

func sortPluginRuntimeAppliedMembers(members []PluginRuntimeAppliedMember) {
	sort.Slice(members, func(i, j int) bool {
		return members[i].ExtensionID < members[j].ExtensionID
	})
}

func validPluginRuntimeNodeLease(lease time.Duration) bool {
	return lease > 0 && lease <= maxPluginRuntimeNodeLease && lease.Milliseconds() > 0
}
