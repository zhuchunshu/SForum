package identityregistry

import (
	"context"
	"encoding/json"
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
	ErrUnauthorized   = errors.New("identity registry role suggestion actor is unauthorized")
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

// DurableRootPublicationTip is the append-only exact publication graph for one
// plugin. It complements leaf ownership rows so session policies, risk hooks,
// and root-only Identity declarations remain provable after restart.
// PublicationJSON is canonical declarative material and excludes the ephemeral
// RuntimeInstanceID; startup validates the current runtime binding separately.
type DurableRootPublicationTip struct {
	OwnerExtensionID   string          `json:"ownerExtensionId"`
	Revision           int64           `json:"revision"`
	RegistryState      string          `json:"registryState"`
	ExtensionVersionID int64           `json:"extensionVersionId"`
	ExtensionVersion   string          `json:"extensionVersion"`
	PackageDigest      string          `json:"packageDigest"`
	SchemaVersion      string          `json:"schemaVersion"`
	PublicationDigest  string          `json:"publicationDigest"`
	PublicationJSON    json.RawMessage `json:"publicationJson"`
	ActorUserID        int64           `json:"actorUserId"`
	AuditEventID       int64           `json:"auditEventId"`
	CreatedAt          time.Time       `json:"createdAt"`
}

// DurableState is the restart material for ownership history plus latest tips.
// Owners and Tips are always sorted deterministically by the store.
type DurableState struct {
	Owners   []DurableOwner              `json:"owners"`
	Tips     []DurableDeclarationTip     `json:"tips"`
	RootTips []DurableRootPublicationTip `json:"rootTips"`
}

// RoleSuggestion is a Host-reviewable permission-to-role recommendation.
// Install and enable only create pending rows. An active Host role manager may
// explicitly approve one, which adds this one mapping without replacing any
// existing role permissions. ApprovalState records Host review; Applied records
// whether immutable grant evidence exists. Legacy 028 approved rows stay
// Applied=false until an explicit apply with expected revision 2.
type RoleSuggestion struct {
	ID                        int64  `json:"id"`
	PermissionKey             string `json:"permissionKey"`
	OwnerExtensionID          string `json:"ownerExtensionId"`
	ExtensionVersionID        int64  `json:"extensionVersionId"`
	ExtensionVersion          string `json:"extensionVersion"`
	PackageDigest             string `json:"packageDigest"`
	PermissionContractVersion string `json:"permissionContractVersion"`
	DeclarationDigest         string `json:"declarationDigest"`
	RoleKey                   string `json:"roleKey"`
	ApprovalState             string `json:"approvalState"`
	// Applied is true only when extension_permission_role_grants has evidence.
	// Approved without Applied means review-only (legacy) and must not be shown
	// as an authority grant.
	Applied              bool       `json:"applied"`
	Revision             int64      `json:"revision"`
	DecidedByUserID      int64      `json:"decidedByUserId,omitempty"`
	DecisionAuditEventID int64      `json:"decisionAuditEventId,omitempty"`
	DecidedAt            *time.Time `json:"decidedAt,omitempty"`
	AppliedByUserID      int64      `json:"appliedByUserId,omitempty"`
	AppliedAuditEventID  int64      `json:"appliedAuditEventId,omitempty"`
	AppliedAt            *time.Time `json:"appliedAt,omitempty"`
	CreatedAt            time.Time  `json:"createdAt"`
	UpdatedAt            time.Time  `json:"updatedAt"`
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

// RoleSuggestionPageInput requests one stable keyset page. Cursor is opaque and
// bound to the normalized filters; callers cannot reuse it under another view.
type RoleSuggestionPageInput struct {
	Filter RoleSuggestionFilter
	Cursor string
}

// RoleSuggestionPage carries a high-water-bounded page. Rows inserted after the
// first page are intentionally deferred to a fresh traversal.
type RoleSuggestionPage struct {
	Items      []RoleSuggestion `json:"items"`
	NextCursor string           `json:"nextCursor,omitempty"`
}

const roleSuggestionCursorVersion = 1

type roleSuggestionCursor struct {
	Version          int    `json:"version"`
	AfterID          int64  `json:"afterId"`
	HighWaterID      int64  `json:"highWaterId"`
	ApprovalState    string `json:"approvalState,omitempty"`
	RoleKey          string `json:"roleKey,omitempty"`
	PermissionKey    string `json:"permissionKey,omitempty"`
	OwnerExtensionID string `json:"ownerExtensionId,omitempty"`
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

// ReconcilePublicationInput binds one durable lifecycle transition to the
// exact source/target artifacts admitted by the Host coordinator. Desired is
// nil when the transition retires every active declaration for the extension.
// Startup restoration is read-only and uses LoadDurableState instead.
type ReconcilePublicationInput struct {
	ExtensionID   string
	AllowedSource *Artifact
	AllowedTarget *Artifact
	Desired       *Publication
	ActorUserID   int64
	AuditEventID  int64
}

type roleSuggestionAuditMetadata struct {
	SuggestionID                int64  `json:"suggestionId"`
	PermissionKey               string `json:"permissionKey"`
	OwnerExtensionID            string `json:"ownerExtensionId"`
	ExtensionVersionID          int64  `json:"extensionVersionId"`
	ExtensionVersion            string `json:"extensionVersion"`
	PackageDigest               string `json:"packageDigest"`
	PermissionContractVersion   string `json:"permissionContractVersion"`
	DeclarationDigest           string `json:"declarationDigest"`
	RoleKey                     string `json:"roleKey"`
	ExpectedRevision            int64  `json:"expectedRevision"`
	ApprovalState               string `json:"approvalState"`
	PermissionCatalogRegistered bool   `json:"permissionCatalogRegistered"`
	RolePermissionAdded         bool   `json:"rolePermissionAdded"`
	RoleGrantApplied            bool   `json:"roleGrantApplied"`
}

type roleSuggestionAuditEvidence struct {
	SuggestionID                int64  `json:"suggestionId"`
	PermissionKey               string `json:"permissionKey"`
	OwnerExtensionID            string `json:"ownerExtensionId"`
	ExtensionVersionID          int64  `json:"extensionVersionId"`
	ExtensionVersion            string `json:"extensionVersion"`
	PackageDigest               string `json:"packageDigest"`
	PermissionContractVersion   string `json:"permissionContractVersion"`
	DeclarationDigest           string `json:"declarationDigest"`
	RoleKey                     string `json:"roleKey"`
	ExpectedRevision            int64  `json:"expectedRevision"`
	ApprovalState               string `json:"approvalState"`
	PermissionCatalogRegistered *bool  `json:"permissionCatalogRegistered"`
	RolePermissionAdded         *bool  `json:"rolePermissionAdded"`
	RoleGrantApplied            *bool  `json:"roleGrantApplied"`
}

// Store is the durable Identity Registry repository boundary.
type Store interface {
	LoadDurableState(ctx context.Context) (DurableState, error)
	ListRoleSuggestionPage(ctx context.Context, input RoleSuggestionPageInput) (RoleSuggestionPage, error)
	ListRoleSuggestions(ctx context.Context, filter RoleSuggestionFilter) ([]RoleSuggestion, error)
	DecideRoleSuggestion(ctx context.Context, input DecideRoleSuggestionInput) (RoleSuggestion, error)
}

// PublicationStore is deliberately separate from Store. Lifecycle publication
// writes durable declaration history, while Store remains the Host review
// boundary used by existing role-suggestion services and their test doubles.
type PublicationStore interface {
	Reconcile(ctx context.Context, input ReconcilePublicationInput) (DurableState, error)
	LoadDurableState(ctx context.Context) (DurableState, error)
}

// LegacyPublicationAdopter is an optional, narrowly scoped capability for a
// one-time startup adoption of pre-feature enabled plugins that never wrote
// durable Identity Registry history. Normal ValidateDurablePublication remains
// strict: empty durable history is never generally acceptable.
//
// Implementations must adopt the full missing batch in one all-or-none
// transaction and fail closed unless every evidence lock proves each enabled
// exact artifact still has live trust-grant + matching trust audit + full
// TrustImpact integrity. Stores without this capability stay ErrNotFound.
type LegacyPublicationAdopter interface {
	AdoptLegacyPublications(ctx context.Context, publications []Publication) (DurableState, error)
}

// DurableStateToTombstones converts durable ownership into in-process tombstones.
// Owner rows without a matching declaration tip/contract fail closed so restart
// restore never silently drops permanent ownership.
func DurableStateToTombstones(state DurableState) ([]Tombstone, error) {
	if _, err := durableRootPublications(state); err != nil {
		return nil, err
	}
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
