package identityregistry

import (
	"bytes"
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

// SessionPolicyLifecycleMutationGate is an additive process-local coordination
// boundary. PostgreSQL remains authoritative across processes; this gate keeps
// same-process lifecycle waiters from borrowing the Host effect's main pool
// before they block on the durable selection lock. An allow path must invoke
// the callback synchronously and exactly once, must propagate its error, and
// must not retain the callback after returning.
type SessionPolicyLifecycleMutationGate interface {
	RunSessionPolicyMutation(context.Context, func() error) error
}

func exactReplaySessionPolicyProvider(
	currentRoot *DurableRootPublicationTip,
	desiredRoot *durableDesiredRootPublication,
	current map[string]DurableDeclarationTip,
	desiredByKey map[string]durableDesiredDeclaration,
	desired *Publication,
) *DurableDeclarationTip {
	if currentRoot == nil || desiredRoot == nil || desired == nil ||
		currentRoot.RegistryState != RegistryStateActive ||
		durableRootTipArtifactIdentity(*currentRoot) != durableRootTipArtifactIdentity(desiredRoot.tip) ||
		currentRoot.SchemaVersion != desiredRoot.tip.SchemaVersion ||
		currentRoot.PublicationDigest != desiredRoot.tip.PublicationDigest ||
		!bytes.Equal(currentRoot.PublicationJSON, desiredRoot.tip.PublicationJSON) ||
		desired.Identity == nil || desired.Identity.SessionPolicy == "" ||
		desired.Identity.SessionPolicy == "core.session.default" {
		return nil
	}

	activeCount := 0
	for key, tip := range current {
		if tip.RegistryState != RegistryStateActive {
			continue
		}
		activeCount++
		declaration, found := desiredByKey[key]
		if !found || durableTipArtifactIdentity(tip) != durableArtifactIdentityOf(declaration.artifact) ||
			tip.ContractVersion != declaration.contractVersion || tip.DeclarationDigest != declaration.digest {
			return nil
		}
	}
	if activeCount != len(desiredByKey) {
		return nil
	}

	key := ownershipKey(TombstoneKindProvider, desired.Identity.SessionPolicy)
	tip, found := current[key]
	if !found || tip.RegistryState != RegistryStateActive {
		return nil
	}
	copy := tip
	return &copy
}
