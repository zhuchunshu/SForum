package extensions

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"sort"
	"strconv"
	"strings"
	"time"
)

type PluginRuntimePublicationReason string

const (
	PluginRuntimePublicationEnable           PluginRuntimePublicationReason = "enable"
	PluginRuntimePublicationDisable          PluginRuntimePublicationReason = "disable"
	PluginRuntimePublicationUpgrade          PluginRuntimePublicationReason = "upgrade"
	PluginRuntimePublicationRollback         PluginRuntimePublicationReason = "rollback"
	PluginRuntimePublicationUninstall        PluginRuntimePublicationReason = "uninstall"
	PluginRuntimePublicationStartupReconcile PluginRuntimePublicationReason = "startup_reconcile"
	PluginRuntimePublicationRecovery         PluginRuntimePublicationReason = "recovery"
)

var (
	ErrPluginRuntimePublicationNotFound = errors.New("extensions: plugin runtime publication not found")
	ErrPluginRuntimePublicationConflict = errors.New("extensions: plugin runtime publication conflict")
	// ErrPluginRuntimePublicationSuperseded 表示请求 revision 已被 durable 或
	// process-local revision 超越。Coordinator 必须重读 durable latest，且仅在
	// 确有更新 revision 时立即追赶；process-ahead 情况走正常轮询退避。
	ErrPluginRuntimePublicationSuperseded = errors.New("extensions: plugin runtime publication superseded")
)

type PluginRuntimeMember struct {
	ExtensionID        string `json:"extensionId"`
	ExtensionVersionID int64  `json:"extensionVersionId"`
	ExtensionVersion   string `json:"extensionVersion"`
	PackageDigest      string `json:"packageDigest"`
}

type PluginRuntimePublication struct {
	Revision      int64                          `json:"revision"`
	MemberCount   int                            `json:"memberCount"`
	MembersDigest string                         `json:"membersDigest"`
	Members       []PluginRuntimeMember          `json:"members"`
	Reason        PluginRuntimePublicationReason `json:"reason"`
	ActorUserID   int64                          `json:"actorUserId,omitempty"`
	CreatedAt     time.Time                      `json:"createdAt"`
}

type PluginRuntimePublicationRepository interface {
	LatestPluginRuntimePublication(context.Context) (PluginRuntimePublication, error)
	PluginRuntimePublicationByRevision(context.Context, int64) (PluginRuntimePublication, error)
}

type PluginRuntimePublicationNotificationSource interface {
	WatchPluginRuntimePublications(context.Context, func())
}

// PluginRuntimeMembersDigest uses the exact byte-length-prefixed representation
// enforced by migration 202607160027. Runtime instance ids are deliberately not
// part of desired artifact identity.
func PluginRuntimeMembersDigest(members []PluginRuntimeMember) (string, error) {
	_, digest, err := canonicalPluginRuntimeMembers(members)
	return digest, err
}

func canonicalPluginRuntimeMembers(members []PluginRuntimeMember) ([]PluginRuntimeMember, string, error) {
	canonical := append([]PluginRuntimeMember(nil), members...)
	for _, member := range canonical {
		if !validPluginRuntimeMember(member) {
			return nil, "", ErrPluginRuntimePublicationConflict
		}
	}
	sort.Slice(canonical, func(i, j int) bool {
		return canonical[i].ExtensionID < canonical[j].ExtensionID
	})
	for index := 1; index < len(canonical); index++ {
		if canonical[index-1].ExtensionID == canonical[index].ExtensionID {
			return nil, "", ErrPluginRuntimePublicationConflict
		}
	}

	var input strings.Builder
	for _, member := range canonical {
		for _, field := range []string{
			member.ExtensionID,
			strconv.FormatInt(member.ExtensionVersionID, 10),
			member.ExtensionVersion,
			member.PackageDigest,
		} {
			input.WriteString(strconv.Itoa(len([]byte(field))))
			input.WriteByte(':')
			input.WriteString(field)
		}
	}
	sum := sha256.Sum256([]byte(input.String()))
	return canonical, hex.EncodeToString(sum[:]), nil
}

func validPluginRuntimeMember(member PluginRuntimeMember) bool {
	return member.ExtensionID != "" && member.ExtensionID == strings.TrimSpace(member.ExtensionID) &&
		member.ExtensionVersionID > 0 &&
		member.ExtensionVersion != "" && member.ExtensionVersion == strings.TrimSpace(member.ExtensionVersion) &&
		validPackageDigest(member.PackageDigest)
}

func validPluginRuntimePublicationReason(reason PluginRuntimePublicationReason) bool {
	switch reason {
	case PluginRuntimePublicationEnable,
		PluginRuntimePublicationDisable,
		PluginRuntimePublicationUpgrade,
		PluginRuntimePublicationRollback,
		PluginRuntimePublicationUninstall,
		PluginRuntimePublicationStartupReconcile,
		PluginRuntimePublicationRecovery:
		return true
	default:
		return false
	}
}

func normalizedPluginRuntimePublication(publication PluginRuntimePublication) (PluginRuntimePublication, error) {
	if publication.Revision <= 0 || publication.ActorUserID < 0 || publication.CreatedAt.IsZero() ||
		!validPluginRuntimePublicationReason(publication.Reason) {
		return PluginRuntimePublication{}, ErrPluginRuntimePublicationConflict
	}
	members, digest, err := canonicalPluginRuntimeMembers(publication.Members)
	if err != nil || publication.MemberCount != len(members) || publication.MembersDigest != digest {
		return PluginRuntimePublication{}, ErrPluginRuntimePublicationConflict
	}
	publication.Members = members
	return publication, nil
}

func samePluginRuntimePublication(left, right PluginRuntimePublication) bool {
	left, leftErr := normalizedPluginRuntimePublication(left)
	right, rightErr := normalizedPluginRuntimePublication(right)
	if leftErr != nil || rightErr != nil || left.Revision != right.Revision ||
		left.MemberCount != right.MemberCount || left.MembersDigest != right.MembersDigest ||
		left.Reason != right.Reason || left.ActorUserID != right.ActorUserID ||
		!left.CreatedAt.Equal(right.CreatedAt) || len(left.Members) != len(right.Members) {
		return false
	}
	for index := range left.Members {
		if left.Members[index] != right.Members[index] {
			return false
		}
	}
	return true
}
