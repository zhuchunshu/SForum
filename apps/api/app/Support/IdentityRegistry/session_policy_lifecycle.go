package identityregistry

import (
	"context"

	"github.com/jackc/pgx/v5"
)

// IdentitySessionPolicySelectionLockKey is shared by operator selection and
// lifecycle invalidation so both paths serialize the singleton consistently.
const IdentitySessionPolicySelectionLockKey = "sforum:identity-session-policy:selection@1"

// SessionPolicyLifecycleTransition describes the exact provider association
// that may survive one durable Registry reconciliation. PreservedProvider is
// nil unless the complete active publication is an exact replay.
type SessionPolicyLifecycleTransition struct {
	OwnerExtensionID      string
	ActorUserID           int64
	LifecycleAuditEventID int64
	PreservedProvider     *DurableDeclarationTip
}

// SessionPolicyLifecycleInvalidator is the narrow transaction-bound bridge
// from Registry lifecycle publication to the Host-owned session selection.
// Implementations must not begin, commit, or roll back a transaction.
type SessionPolicyLifecycleInvalidator interface {
	InvalidateSessionPolicySelectionTx(
		context.Context,
		pgx.Tx,
		SessionPolicyLifecycleTransition,
	) error
}
