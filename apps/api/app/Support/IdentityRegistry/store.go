package identityregistry

import (
	"context"
	"errors"
	"sort"
	"strings"
	"time"
)

const (
	RegistryStateActive    = "active"
	RegistryStateTombstone = "tombstone"

	RoleSuggestionPending  = "pending"
	RoleSuggestionApproved = "approved"
	RoleSuggestionRejected = "rejected"

	defaultRoleSuggestionLimit = 50
	maxRoleSuggestionLimit     = 100
)

// 持久化层额外哨兵：与进程内 ErrInvalid/ErrNotFound/ErrRevisionConflict 并列，
// 供 Host 审批与重启恢复区分“声明 tip 过期”和“审批目标角色不存在”。
var (
	ErrStale          = errors.New("identity registry durable tip is stale")
	ErrTargetConflict = errors.New("identity registry role suggestion target is unavailable")
)

// DurableOwner is one permanent ownership tombstone row. It never grants authority.
type DurableOwner struct {
	IdentityKind     string    `json:"identityKind"`
	StableID         string    `json:"stableId"`
	OwnerExtensionID string    `json:"ownerExtensionId"`
	ClaimedAt        time.Time `json:"claimedAt"`
}

// DurableDeclarationTip is the latest append-only declaration event for one
// owned identity. ContractVersion is required before ownership can be restored
// into the in-process tombstone graph.
type DurableDeclarationTip struct {
	IdentityKind       string    `json:"identityKind"`
	StableID           string    `json:"stableId"`
	OwnerExtensionID   string    `json:"ownerExtensionId"`
	Revision           int64     `json:"revision"`
	RegistryState      string    `json:"registryState"`
	ExtensionVersionID int64     `json:"extensionVersionId"`
	ExtensionVersion   string    `json:"extensionVersion"`
	PackageDigest      string    `json:"packageDigest"`
	ContractVersion    string    `json:"contractVersion"`
	DeclarationDigest  string    `json:"declarationDigest"`
	ActorUserID        int64     `json:"actorUserId,omitempty"`
	AuditEventID       int64     `json:"auditEventId,omitempty"`
	CreatedAt          time.Time `json:"createdAt"`
}

// DurableState is the restart material for ownership history plus latest tips.
// Owners and Tips are always sorted deterministically by the store.
type DurableState struct {
	Owners []DurableOwner          `json:"owners"`
	Tips   []DurableDeclarationTip `json:"tips"`
}

// RoleSuggestion is a Host-reviewable permission→role recommendation. Approving
// it records review evidence only; Host role management remains the sole
// authority that may write role_permissions.
type RoleSuggestion struct {
	ID                        int64      `json:"id"`
	PermissionKey             string     `json:"permissionKey"`
	OwnerExtensionID          string     `json:"ownerExtensionId"`
	ExtensionVersionID        int64      `json:"extensionVersionId"`
	ExtensionVersion          string     `json:"extensionVersion"`
	PackageDigest             string     `json:"packageDigest"`
	PermissionContractVersion string     `json:"permissionContractVersion"`
	DeclarationDigest         string     `json:"declarationDigest"`
	RoleKey                   string     `json:"roleKey"`
	ApprovalState             string     `json:"approvalState"`
	Revision                  int64      `json:"revision"`
	DecidedByUserID           int64      `json:"decidedByUserId,omitempty"`
	DecisionAuditEventID      int64      `json:"decisionAuditEventId,omitempty"`
	DecidedAt                 *time.Time `json:"decidedAt,omitempty"`
	CreatedAt                 time.Time  `json:"createdAt"`
	UpdatedAt                 time.Time  `json:"updatedAt"`
}

// RoleSuggestionFilter selects a bounded, deterministic page of suggestions.
// Empty string fields are ignored.
type RoleSuggestionFilter struct {
	ApprovalState    string
	RoleKey          string
	PermissionKey    string
	OwnerExtensionID string
	Limit            int
}

// DecideRoleSuggestionInput is the Host CAS decision for one pending suggestion.
// The repository inserts the actor-bound audit_events row itself; callers must
// not supply a pre-allocated audit id.
type DecideRoleSuggestionInput struct {
	ID               int64
	ExpectedRevision int64
	ApprovalState    string
	ActorUserID      int64
}

// Store is the durable Identity Registry repository boundary.
type Store interface {
	LoadDurableState(ctx context.Context) (DurableState, error)
	ListRoleSuggestions(ctx context.Context, filter RoleSuggestionFilter) ([]RoleSuggestion, error)
	DecideRoleSuggestion(ctx context.Context, input DecideRoleSuggestionInput) (RoleSuggestion, error)
}

// DurableStateToTombstones converts durable ownership into in-process tombstones.
// Owner rows without a matching declaration tip/contract fail closed so restart
// restore never silently drops permanent ownership.
func DurableStateToTombstones(state DurableState) ([]Tombstone, error) {
	owners := make(map[string]DurableOwner, len(state.Owners))
	for _, raw := range state.Owners {
		owner := raw
		owner.IdentityKind = strings.ToLower(strings.TrimSpace(owner.IdentityKind))
		owner.StableID = strings.ToLower(strings.TrimSpace(owner.StableID))
		owner.OwnerExtensionID = strings.ToLower(strings.TrimSpace(owner.OwnerExtensionID))
		key := ownershipKey(owner.IdentityKind, owner.StableID)
		if _, duplicate := owners[key]; duplicate {
			return nil, ErrInvalid
		}
		owners[key] = owner
	}

	tips := make(map[string]DurableDeclarationTip, len(state.Tips))
	for _, raw := range state.Tips {
		tip := raw
		tip.IdentityKind = strings.ToLower(strings.TrimSpace(tip.IdentityKind))
		tip.StableID = strings.ToLower(strings.TrimSpace(tip.StableID))
		tip.OwnerExtensionID = strings.ToLower(strings.TrimSpace(tip.OwnerExtensionID))
		tip.RegistryState = strings.ToLower(strings.TrimSpace(tip.RegistryState))
		if tip.Revision <= 0 ||
			(tip.RegistryState != RegistryStateActive && tip.RegistryState != RegistryStateTombstone) {
			return nil, ErrInvalid
		}
		key := ownershipKey(tip.IdentityKind, tip.StableID)
		// 同一 identity 只保留最高 revision tip；Load 已确定性排序，后写覆盖前写。
		if existing, found := tips[key]; found {
			if tip.Revision == existing.Revision {
				return nil, ErrInvalid
			}
			if tip.Revision < existing.Revision {
				continue
			}
		}
		tips[key] = tip
	}
	// A declaration without its permanent owner is also an incomplete restore.
	// Ignoring it would temporarily reopen a claimed namespace after restart.
	for key, tip := range tips {
		owner, ok := owners[key]
		if !ok || tip.OwnerExtensionID != owner.OwnerExtensionID {
			return nil, ErrInvalid
		}
	}

	result := make([]Tombstone, 0, len(owners))
	for key, owner := range owners {
		tip, ok := tips[key]
		if !ok ||
			strings.TrimSpace(tip.ContractVersion) == "" ||
			tip.OwnerExtensionID != owner.OwnerExtensionID {
			return nil, ErrInvalid
		}
		tombstone, err := normalizeTombstone(Tombstone{
			Kind:             owner.IdentityKind,
			ID:               owner.StableID,
			ContractVersion:  tip.ContractVersion,
			OwnerExtensionID: owner.OwnerExtensionID,
		})
		if err != nil {
			return nil, err
		}
		result = append(result, tombstone)
	}
	sort.Slice(result, func(i, j int) bool {
		if result[i].Kind != result[j].Kind {
			return result[i].Kind < result[j].Kind
		}
		if result[i].ID != result[j].ID {
			return result[i].ID < result[j].ID
		}
		return result[i].ContractVersion < result[j].ContractVersion
	})
	return result, nil
}

func normalizeRoleSuggestionFilter(filter RoleSuggestionFilter) RoleSuggestionFilter {
	filter.ApprovalState = strings.ToLower(strings.TrimSpace(filter.ApprovalState))
	filter.RoleKey = strings.ToLower(strings.TrimSpace(filter.RoleKey))
	filter.PermissionKey = strings.ToLower(strings.TrimSpace(filter.PermissionKey))
	filter.OwnerExtensionID = strings.ToLower(strings.TrimSpace(filter.OwnerExtensionID))
	if filter.Limit <= 0 || filter.Limit > maxRoleSuggestionLimit {
		filter.Limit = defaultRoleSuggestionLimit
	}
	return filter
}

func validRoleSuggestionDecision(input DecideRoleSuggestionInput) bool {
	if input.ID <= 0 || input.ExpectedRevision <= 0 || input.ActorUserID <= 0 {
		return false
	}
	state := strings.ToLower(strings.TrimSpace(input.ApprovalState))
	return state == RoleSuggestionApproved || state == RoleSuggestionRejected
}

func roleSuggestionDecisionAction(approvalState string) string {
	switch strings.ToLower(strings.TrimSpace(approvalState)) {
	case RoleSuggestionApproved:
		return "identity.role_suggestion.approve"
	case RoleSuggestionRejected:
		return "identity.role_suggestion.reject"
	default:
		return ""
	}
}
