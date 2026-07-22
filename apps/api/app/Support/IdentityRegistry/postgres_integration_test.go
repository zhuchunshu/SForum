package identityregistry

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"testing"

	"github.com/jackc/pgx/v5/pgconn"
)

const identityRegistryOwnershipMigrationVersion = int64(202607160028)
const identityRoleApprovalsMigrationVersion = int64(202607160029)
const identityRootPublicationsMigrationVersion = int64(202607160033)
const identityOrphanTombstoneMigrationVersion = int64(202607210044)
const extensionPermissionLocalizationMigrationVersion = int64(202607231001)

func TestDurableStateToTombstonesRejectsIncompleteOrDuplicateState(t *testing.T) {
	tip := DurableDeclarationTip{
		IdentityKind: TombstoneKindPermission, StableID: "fixture.identity.profile",
		OwnerExtensionID: "fixture.identity", Revision: 1, RegistryState: RegistryStateActive,
		ContractVersion: "fixture.identity.profile@1",
	}
	owner := DurableOwner{
		IdentityKind: TombstoneKindPermission, StableID: "fixture.identity.profile",
		OwnerExtensionID: "fixture.identity",
	}
	for name, state := range map[string]DurableState{
		"tip without owner":      {Tips: []DurableDeclarationTip{tip}},
		"owner without tip":      {Owners: []DurableOwner{owner}},
		"duplicate owner":        {Owners: []DurableOwner{owner, owner}, Tips: []DurableDeclarationTip{tip}},
		"duplicate tip revision": {Owners: []DurableOwner{owner}, Tips: []DurableDeclarationTip{tip, tip}},
	} {
		t.Run(name, func(t *testing.T) {
			if _, err := DurableStateToTombstones(state); !errors.Is(err, ErrInvalid) {
				t.Fatalf("incomplete durable state error=%v", err)
			}
		})
	}
}

func TestMapStoreErrorPreservesContextAndClassifiesPostgresConflicts(t *testing.T) {
	if err := mapStoreError(fmt.Errorf("wrapped: %w", context.Canceled)); !errors.Is(err, context.Canceled) {
		t.Fatalf("cancel mapping=%v", err)
	}
	if err := mapStoreError(fmt.Errorf("wrapped: %w", context.DeadlineExceeded)); !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("deadline mapping=%v", err)
	}
	if err := mapStoreError(&pgconn.PgError{Code: "40001"}); !errors.Is(err, errRetryableIdentityRegistryTransaction) {
		t.Fatalf("serialization mapping=%v", err)
	}
	if err := mapStoreError(&pgconn.PgError{Code: "23505"}); !errors.Is(err, ErrRevisionConflict) {
		t.Fatalf("unique mapping=%v", err)
	}
	if err := mapStoreError(&pgconn.PgError{Code: "23503"}); !errors.Is(err, ErrInvalid) {
		t.Fatalf("actor foreign-key mapping=%v", err)
	}
	if err := mapStoreError(&pgconn.PgError{Code: "P0001", Message: "permission role suggestion decision actor lacks role.manage"}); !errors.Is(err, ErrUnauthorized) {
		t.Fatalf("authorization mapping=%v", err)
	}
}

func TestPostgresStoreDurableRestoreAndOrphanFailClosed(t *testing.T) {
	fixture := newIdentityRegistryStoreFixture(t)
	fixture.seedOwner(t, "permission", "fixture.identity.profile")
	fixture.seedOwner(t, "provider", "fixture.identity.provider.risk")
	fixture.seedOwner(t, "user_field", "fixture.identity.field.code")
	// 故意乱序插入，验证 LoadDurableState 输出确定性排序。
	fixture.seedDeclaration(t, "user_field", "fixture.identity.field.code", 1, RegistryStateActive,
		"fixture.identity.field.code@1", "d")
	fixture.seedDeclaration(t, "permission", "fixture.identity.profile", 1, RegistryStateActive,
		"fixture.identity.profile@1", "c")
	fixture.seedDeclaration(t, "provider", "fixture.identity.provider.risk", 1, RegistryStateActive,
		"fixture.identity.provider.risk@1", "e")

	state, err := fixture.store.LoadDurableState(fixture.ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(state.Owners) != 3 || len(state.Tips) != 3 {
		t.Fatalf("durable state owners=%d tips=%d", len(state.Owners), len(state.Tips))
	}
	for index := 1; index < len(state.Owners); index++ {
		prev, next := state.Owners[index-1], state.Owners[index]
		if prev.IdentityKind > next.IdentityKind ||
			(prev.IdentityKind == next.IdentityKind && prev.StableID > next.StableID) {
			t.Fatalf("owners not sorted: %#v then %#v", prev, next)
		}
	}
	for index := 1; index < len(state.Tips); index++ {
		prev, next := state.Tips[index-1], state.Tips[index]
		if prev.IdentityKind > next.IdentityKind ||
			(prev.IdentityKind == next.IdentityKind && prev.StableID > next.StableID) {
			t.Fatalf("tips not sorted: %#v then %#v", prev, next)
		}
	}

	tombstones, err := DurableStateToTombstones(state)
	if err != nil {
		t.Fatal(err)
	}
	if len(tombstones) != 3 {
		t.Fatalf("tombstones=%d", len(tombstones))
	}
	if tombstones[0].Kind != TombstoneKindPermission ||
		tombstones[0].ID != "fixture.identity.profile" ||
		tombstones[0].ContractVersion != "fixture.identity.profile@1" {
		t.Fatalf("first tombstone = %#v", tombstones[0])
	}
	fixture.seedDeclaration(t, "permission", "fixture.identity.profile", 2, RegistryStateTombstone,
		"fixture.identity.profile@1", "c")
	tombstonedState, err := fixture.store.LoadDurableState(fixture.ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(tombstonedState.Tips) != 3 || tombstonedState.Tips[0].Revision != 2 ||
		tombstonedState.Tips[0].RegistryState != RegistryStateTombstone {
		t.Fatalf("latest tombstone tip = %#v", tombstonedState.Tips)
	}
	if restored, err := DurableStateToTombstones(tombstonedState); err != nil || len(restored) != 3 {
		t.Fatalf("restore tombstoned ownership: count=%d err=%v", len(restored), err)
	}

	fixture.seedOwner(t, "permission", "fixture.identity.orphan")
	orphanState, err := fixture.store.LoadDurableState(fixture.ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(orphanState.Owners) != 4 || len(orphanState.Tips) != 3 {
		t.Fatalf("orphan durable state owners=%d tips=%d", len(orphanState.Owners), len(orphanState.Tips))
	}
	if _, err := DurableStateToTombstones(orphanState); !errors.Is(err, ErrInvalid) {
		t.Fatalf("orphan conversion error=%v", err)
	}
}

func TestPostgresStoreRoleSuggestionKeysetPaginationExceedsLimitAndFreezesInserts(t *testing.T) {
	fixture := newIdentityRegistryStoreFixture(t)
	fixture.seedOwner(t, "permission", "fixture.identity.profile")
	fixture.seedDeclaration(t, "permission", "fixture.identity.profile", 1, RegistryStateActive,
		"fixture.identity.profile@1", "c")
	fixture.seedPermissionCatalog(t, "fixture.identity.profile", "fixture.identity.profile@1", "c", 1)

	wantIDs := make(map[int64]bool, 205)
	for index := 0; index < 205; index++ {
		wantIDs[fixture.seedSuggestion(t, fmt.Sprintf("role_%03d", index))] = true
	}
	input := RoleSuggestionPageInput{Filter: RoleSuggestionFilter{
		ApprovalState: RoleSuggestionPending,
		Limit:         37,
	}}
	page, err := fixture.store.ListRoleSuggestionPage(fixture.ctx, input)
	if err != nil {
		t.Fatal(err)
	}
	if len(page.Items) != 37 || page.NextCursor == "" {
		t.Fatalf("first page items=%d cursor=%q", len(page.Items), page.NextCursor)
	}

	// This insert commits while the client is between page requests. Offset
	// pagination would shift; the cursor's first-page high water excludes it.
	lateID := fixture.seedSuggestion(t, "role_late")
	if wantIDs[lateID] {
		t.Fatalf("late id %d overlaps initial traversal", lateID)
	}

	seen := make(map[int64]bool, len(wantIDs))
	for {
		for _, suggestion := range page.Items {
			if seen[suggestion.ID] {
				t.Fatalf("duplicate keyset row %d", suggestion.ID)
			}
			if !wantIDs[suggestion.ID] {
				t.Fatalf("post-high-water suggestion %d leaked into traversal", suggestion.ID)
			}
			if suggestion.Applied {
				t.Fatalf("pending suggestion claimed applied: %#v", suggestion)
			}
			seen[suggestion.ID] = true
		}
		if page.NextCursor == "" {
			break
		}
		input.Cursor = page.NextCursor
		page, err = fixture.store.ListRoleSuggestionPage(fixture.ctx, input)
		if err != nil {
			t.Fatal(err)
		}
	}
	if len(seen) != len(wantIDs) {
		t.Fatalf("keyset traversal saw %d of %d initial suggestions", len(seen), len(wantIDs))
	}

	input.Cursor = ""
	legacy, err := fixture.store.ListRoleSuggestions(fixture.ctx, RoleSuggestionFilter{
		ApprovalState: RoleSuggestionPending,
		Limit:         100,
	})
	if err != nil || len(legacy) != 100 {
		t.Fatalf("legacy first-page wrapper count=%d error=%v", len(legacy), err)
	}
	if _, err := fixture.store.ListRoleSuggestionPage(fixture.ctx, RoleSuggestionPageInput{
		Filter: RoleSuggestionFilter{ApprovalState: RoleSuggestionApproved, Limit: 37},
		Cursor: pageCursorFromFirstPage(t, fixture, RoleSuggestionFilter{
			ApprovalState: RoleSuggestionPending, Limit: 37,
		}),
	}); !errors.Is(err, ErrInvalid) {
		t.Fatalf("cursor reused with another filter error=%v", err)
	}
}

func pageCursorFromFirstPage(
	t *testing.T,
	fixture *identityRegistryStoreFixture,
	filter RoleSuggestionFilter,
) string {
	t.Helper()
	page, err := fixture.store.ListRoleSuggestionPage(fixture.ctx, RoleSuggestionPageInput{Filter: filter})
	if err != nil {
		t.Fatal(err)
	}
	if page.NextCursor == "" {
		t.Fatal("expected a next cursor")
	}
	return page.NextCursor
}

func TestPostgresStoreDecideRoleSuggestionApproveRejectAndGuards(t *testing.T) {
	fixture := newIdentityRegistryStoreFixture(t)
	fixture.seedOwner(t, "permission", "fixture.identity.profile")
	fixture.seedDeclaration(t, "permission", "fixture.identity.profile", 1, RegistryStateActive,
		"fixture.identity.profile@1", "c")
	fixture.seedPermissionCatalog(t, "fixture.identity.profile", "fixture.identity.profile@1", "c", 1)

	approveID := fixture.seedSuggestion(t, "member")
	rejectID := fixture.seedSuggestion(t, "operator")
	unknownID := fixture.seedSuggestion(t, "future_role")
	rolePermsBefore := fixture.countRolePermissions(t)
	auditBefore := fixture.countAuditEvents(t)

	listed, err := fixture.store.ListRoleSuggestions(fixture.ctx, RoleSuggestionFilter{
		ApprovalState: RoleSuggestionPending, Limit: 10,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(listed) != 3 {
		t.Fatalf("pending suggestions=%d", len(listed))
	}
	for index := 1; index < len(listed); index++ {
		if listed[index-1].ID >= listed[index].ID {
			t.Fatalf("list keyset order not deterministic: %#v", listed)
		}
	}

	approved, err := fixture.store.DecideRoleSuggestion(fixture.ctx, DecideRoleSuggestionInput{
		ID: approveID, ExpectedRevision: 1, ApprovalState: RoleSuggestionApproved, ActorUserID: fixture.actorID,
	})
	if err != nil {
		t.Fatal(err)
	}
	if approved.ApprovalState != RoleSuggestionApproved || approved.Revision != 2 ||
		!approved.Applied || approved.DecidedByUserID != fixture.actorID ||
		approved.DecisionAuditEventID <= 0 || approved.AppliedAuditEventID <= 0 ||
		approved.DecidedAt == nil || approved.AppliedAt == nil {
		t.Fatalf("approved = %#v", approved)
	}
	if approved.DecisionAuditEventID != approved.AppliedAuditEventID {
		t.Fatalf("pending approval must reuse one decision/grant audit: decision=%d applied=%d",
			approved.DecisionAuditEventID, approved.AppliedAuditEventID)
	}
	fixture.assertAudit(t, approved.DecisionAuditEventID, approved, "identity.role_suggestion.approve", boolExpectation(true), true)

	// Exact replay of terminal decision/grant returns durable result, no new audit.
	replay, err := fixture.store.DecideRoleSuggestion(fixture.ctx, DecideRoleSuggestionInput{
		ID: approveID, ExpectedRevision: 1, ApprovalState: RoleSuggestionApproved, ActorUserID: fixture.actorID,
	})
	if err != nil {
		t.Fatal(err)
	}
	if !replay.Applied || replay.DecisionAuditEventID != approved.DecisionAuditEventID {
		t.Fatalf("idempotent replay = %#v", replay)
	}
	if got := fixture.countAuditEvents(t); got != auditBefore+1 {
		t.Fatalf("idempotent replay duplicated audit: before=%d after=%d", auditBefore, got)
	}
	if got := fixture.countGrants(t); got != 1 {
		t.Fatalf("idempotent replay duplicated grants=%d", got)
	}

	rejected, err := fixture.store.DecideRoleSuggestion(fixture.ctx, DecideRoleSuggestionInput{
		ID: rejectID, ExpectedRevision: 1, ApprovalState: RoleSuggestionRejected, ActorUserID: fixture.actorID,
	})
	if err != nil {
		t.Fatal(err)
	}
	if rejected.ApprovalState != RoleSuggestionRejected || rejected.Revision != 2 ||
		rejected.Applied || rejected.DecisionAuditEventID <= 0 {
		t.Fatalf("rejected = %#v", rejected)
	}
	fixture.assertAudit(t, rejected.DecisionAuditEventID, rejected, "identity.role_suggestion.reject", boolExpectation(false), false)

	if _, err := fixture.store.DecideRoleSuggestion(fixture.ctx, DecideRoleSuggestionInput{
		ID: approveID, ExpectedRevision: 99, ApprovalState: RoleSuggestionApproved, ActorUserID: fixture.actorID,
	}); !errors.Is(err, ErrRevisionConflict) {
		t.Fatalf("wrong revision error=%v", err)
	}
	if _, err := fixture.store.DecideRoleSuggestion(fixture.ctx, DecideRoleSuggestionInput{
		ID: 9_999_999_999, ExpectedRevision: 1, ApprovalState: RoleSuggestionApproved, ActorUserID: fixture.actorID,
	}); !errors.Is(err, ErrNotFound) {
		t.Fatalf("missing id error=%v", err)
	}
	if _, err := fixture.store.DecideRoleSuggestion(fixture.ctx, DecideRoleSuggestionInput{
		ID: unknownID, ExpectedRevision: 1, ApprovalState: RoleSuggestionApproved, ActorUserID: fixture.actorID,
	}); !errors.Is(err, ErrTargetConflict) {
		t.Fatalf("unknown role error=%v", err)
	}
	var unknownState string
	if err := fixture.pool.QueryRow(fixture.ctx, `
		SELECT approval_state FROM extension_permission_role_suggestions WHERE id = $1
	`, unknownID).Scan(&unknownState); err != nil {
		t.Fatal(err)
	}
	if unknownState != RoleSuggestionPending {
		t.Fatalf("unknown role state=%q", unknownState)
	}

	inactiveID := fixture.seedSuggestion(t, "moderator")
	if _, err := fixture.pool.Exec(fixture.ctx, `UPDATE users SET status = 'disabled' WHERE id = $1`, fixture.actorID); err != nil {
		t.Fatal(err)
	}
	if _, err := fixture.store.DecideRoleSuggestion(fixture.ctx, DecideRoleSuggestionInput{
		ID: inactiveID, ExpectedRevision: 1, ApprovalState: RoleSuggestionApproved, ActorUserID: fixture.actorID,
	}); !errors.Is(err, ErrUnauthorized) {
		t.Fatalf("inactive actor error=%v", err)
	}
	if _, err := fixture.pool.Exec(fixture.ctx, `UPDATE users SET status = 'active' WHERE id = $1`, fixture.actorID); err != nil {
		t.Fatal(err)
	}
	deniedID := fixture.seedSuggestion(t, "denied_role")
	if _, err := fixture.pool.Exec(fixture.ctx, `
		INSERT INTO user_permission_overrides (user_id, permission_key, effect)
		VALUES ($1, 'role.manage', 'deny')
	`, fixture.actorID); err != nil {
		t.Fatal(err)
	}
	if _, err := fixture.store.DecideRoleSuggestion(fixture.ctx, DecideRoleSuggestionInput{
		ID: deniedID, ExpectedRevision: 1, ApprovalState: RoleSuggestionApproved, ActorUserID: fixture.actorID,
	}); !errors.Is(err, ErrUnauthorized) {
		t.Fatalf("database-denied actor error=%v", err)
	}
	if _, err := fixture.pool.Exec(fixture.ctx, `
		DELETE FROM user_permission_overrides WHERE user_id = $1 AND permission_key = 'role.manage'
	`, fixture.actorID); err != nil {
		t.Fatal(err)
	}

	staleID := fixture.seedSuggestion(t, "stale_role")
	fixture.seedDeclaration(t, "permission", "fixture.identity.profile", 2, RegistryStateTombstone,
		"fixture.identity.profile@1", "c")
	if _, err := fixture.store.DecideRoleSuggestion(fixture.ctx, DecideRoleSuggestionInput{
		ID: staleID, ExpectedRevision: 1, ApprovalState: RoleSuggestionApproved, ActorUserID: fixture.actorID,
	}); !errors.Is(err, ErrStale) {
		t.Fatalf("stale exact-artifact decision error=%v", err)
	}

	rolePermsAfter := fixture.countRolePermissions(t)
	if rolePermsAfter != rolePermsBefore+1 {
		t.Fatalf("approved suggestion did not add exactly one role permission: %d -> %d", rolePermsBefore, rolePermsAfter)
	}
	var memberGrant, catalogRows, grantRows int
	if err := fixture.pool.QueryRow(fixture.ctx, `
		SELECT
		  (SELECT count(*)
		   FROM role_permissions
		   JOIN roles ON roles.id = role_permissions.role_id
		   WHERE roles.key = 'member' AND permission_key = 'fixture.identity.profile'),
		  (SELECT count(*)
		   FROM extension_permission_catalog
		   WHERE permission_key = 'fixture.identity.profile'
		     AND owner_extension_id = 'fixture.identity'),
		  (SELECT count(*)
		   FROM extension_permission_role_grants
		   WHERE permission_key = 'fixture.identity.profile')
	`).Scan(&memberGrant, &catalogRows, &grantRows); err != nil {
		t.Fatal(err)
	}
	if memberGrant != 1 || catalogRows != 1 || grantRows != 1 {
		t.Fatalf("member grant=%d catalog=%d grants=%d", memberGrant, catalogRows, grantRows)
	}
}

func TestPostgresStoreTerminalReplayRequiresOriginalAuthorizedActor(t *testing.T) {
	fixture := newIdentityRegistryStoreFixture(t)
	fixture.seedOwner(t, "permission", "fixture.identity.profile")
	fixture.seedDeclaration(t, "permission", "fixture.identity.profile", 1, RegistryStateActive,
		"fixture.identity.profile@1", "c")
	fixture.seedPermissionCatalog(t, "fixture.identity.profile", "fixture.identity.profile@1", "c", 1)

	approvedID := fixture.seedSuggestion(t, "member")
	rejectedID := fixture.seedSuggestion(t, "operator")
	for id, state := range map[int64]string{
		approvedID: RoleSuggestionApproved,
		rejectedID: RoleSuggestionRejected,
	} {
		if _, err := fixture.store.DecideRoleSuggestion(fixture.ctx, DecideRoleSuggestionInput{
			ID: id, ExpectedRevision: 1, ApprovalState: state, ActorUserID: fixture.actorID,
		}); err != nil {
			t.Fatal(err)
		}
	}

	otherActorID := fixture.seedRoleManager(t, "active")
	assertReplaysUnauthorized := func(t *testing.T, actorUserID int64) {
		t.Helper()
		for id, state := range map[int64]string{
			approvedID: RoleSuggestionApproved,
			rejectedID: RoleSuggestionRejected,
		} {
			if _, err := fixture.store.DecideRoleSuggestion(fixture.ctx, DecideRoleSuggestionInput{
				ID: id, ExpectedRevision: 1, ApprovalState: state, ActorUserID: actorUserID,
			}); !errors.Is(err, ErrUnauthorized) {
				t.Fatalf("terminal replay id=%d state=%s actor=%d error=%v", id, state, actorUserID, err)
			}
		}
	}

	// Current authority alone is insufficient: replay remains bound to the
	// immutable actor recorded by the original decision and audit event.
	assertReplaysUnauthorized(t, otherActorID)
	canceledCtx, cancel := context.WithCancel(fixture.ctx)
	cancel()
	for id, state := range map[int64]string{
		approvedID: RoleSuggestionApproved,
		rejectedID: RoleSuggestionRejected,
	} {
		readback, err := fixture.store.readbackRoleSuggestionDecision(canceledCtx, DecideRoleSuggestionInput{
			ID: id, ExpectedRevision: 1, ApprovalState: state, ActorUserID: fixture.actorID,
		}, state, errors.New("ambiguous commit"))
		if err != nil || readback.ID != id {
			t.Fatalf("bounded terminal readback id=%d state=%s result=%#v error=%v", id, state, readback, err)
		}
		if _, err := fixture.store.readbackRoleSuggestionDecision(canceledCtx, DecideRoleSuggestionInput{
			ID: id, ExpectedRevision: 1, ApprovalState: state, ActorUserID: otherActorID,
		}, state, errors.New("ambiguous commit")); !errors.Is(err, ErrUnauthorized) {
			t.Fatalf("terminal readback id=%d state=%s accepted different actor: %v", id, state, err)
		}
	}

	if _, err := fixture.pool.Exec(fixture.ctx, `UPDATE users SET status = 'disabled' WHERE id = $1`, fixture.actorID); err != nil {
		t.Fatal(err)
	}
	assertReplaysUnauthorized(t, fixture.actorID)
	if _, err := fixture.pool.Exec(fixture.ctx, `UPDATE users SET status = 'active' WHERE id = $1`, fixture.actorID); err != nil {
		t.Fatal(err)
	}
	if _, err := fixture.pool.Exec(fixture.ctx, `
		INSERT INTO user_permission_overrides (user_id, permission_key, effect)
		VALUES ($1, 'role.manage', 'deny')
	`, fixture.actorID); err != nil {
		t.Fatal(err)
	}
	assertReplaysUnauthorized(t, fixture.actorID)
	if _, err := fixture.pool.Exec(fixture.ctx, `
		DELETE FROM user_permission_overrides
		WHERE user_id = $1 AND permission_key = 'role.manage'
	`, fixture.actorID); err != nil {
		t.Fatal(err)
	}
	if _, err := fixture.pool.Exec(fixture.ctx, `DELETE FROM users WHERE id = $1`, fixture.actorID); err != nil {
		t.Fatal(err)
	}
	assertReplaysUnauthorized(t, fixture.actorID)
}

func TestPostgresStoreApprovalRecordsExistingMappingTruthfully(t *testing.T) {
	fixture := newIdentityRegistryStoreFixture(t)
	fixture.seedOwner(t, "permission", "fixture.identity.profile")
	fixture.seedDeclaration(t, "permission", "fixture.identity.profile", 1, RegistryStateActive,
		"fixture.identity.profile@1", "c")
	fixture.seedPermissionCatalog(t, "fixture.identity.profile", "fixture.identity.profile@1", "c", 1)
	suggestionID := fixture.seedSuggestion(t, "member")

	if _, err := fixture.pool.Exec(fixture.ctx, `
		INSERT INTO role_permissions (role_id, permission_key)
		SELECT id, 'fixture.identity.profile' FROM roles WHERE key = 'member'
	`); err != nil {
		t.Fatal(err)
	}
	rolePermissionsBefore := fixture.countRolePermissions(t)
	approved, err := fixture.store.DecideRoleSuggestion(fixture.ctx, DecideRoleSuggestionInput{
		ID: suggestionID, ExpectedRevision: 1,
		ApprovalState: RoleSuggestionApproved, ActorUserID: fixture.actorID,
	})
	if err != nil {
		t.Fatal(err)
	}
	if !approved.Applied || fixture.countRolePermissions(t) != rolePermissionsBefore {
		t.Fatalf("existing mapping approval=%#v role_permissions=%d want %d",
			approved, fixture.countRolePermissions(t), rolePermissionsBefore)
	}
	fixture.assertAudit(t, approved.DecisionAuditEventID, approved,
		"identity.role_suggestion.approve", boolExpectation(false), true)
	if fixture.countGrants(t) != 1 {
		t.Fatalf("existing mapping grant evidence=%d", fixture.countGrants(t))
	}
}

func TestInsertRoleSuggestionGrantConflictRequiresExactEvidence(t *testing.T) {
	fixture := newIdentityRegistryStoreFixture(t)
	fixture.seedOwner(t, "permission", "fixture.identity.profile")
	fixture.seedDeclaration(t, "permission", "fixture.identity.profile", 1, RegistryStateActive,
		"fixture.identity.profile@1", "c")
	fixture.seedPermissionCatalog(t, "fixture.identity.profile", "fixture.identity.profile@1", "c", 1)
	suggestionID := fixture.seedSuggestion(t, "member")
	approved, err := fixture.store.DecideRoleSuggestion(fixture.ctx, DecideRoleSuggestionInput{
		ID: suggestionID, ExpectedRevision: 1,
		ApprovalState: RoleSuggestionApproved, ActorUserID: fixture.actorID,
	})
	if err != nil {
		t.Fatal(err)
	}
	otherActorID := fixture.seedRoleManager(t, "active")

	tx, err := fixture.pool.Begin(fixture.ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = tx.Rollback(context.Background()) }()
	var roleID int64
	if err := tx.QueryRow(fixture.ctx, `SELECT id FROM roles WHERE key = 'member'`).Scan(&roleID); err != nil {
		t.Fatal(err)
	}
	if err := insertRoleSuggestionGrant(
		fixture.ctx, tx, approved, roleID, approved.AppliedByUserID, approved.AppliedAuditEventID,
	); err != nil {
		t.Fatalf("exact grant replay error=%v", err)
	}
	foreignActorAudit, err := insertRoleSuggestionAuditEvent(
		fixture.ctx, tx, "identity.role_suggestion.approve", otherActorID, approved,
		RoleSuggestionApproved, 1, false, true,
	)
	if err != nil {
		t.Fatal(err)
	}
	foreignAudit, err := insertRoleSuggestionAuditEvent(
		fixture.ctx, tx, "identity.role_suggestion.approve", fixture.actorID, approved,
		RoleSuggestionApproved, 1, false, true,
	)
	if err != nil {
		t.Fatal(err)
	}
	for _, test := range []struct {
		name    string
		actorID int64
		auditID int64
	}{
		{name: "foreign actor", actorID: otherActorID, auditID: foreignActorAudit},
		{name: "foreign audit", actorID: fixture.actorID, auditID: foreignAudit},
	} {
		t.Run(test.name, func(t *testing.T) {
			if err := insertRoleSuggestionGrant(
				fixture.ctx, tx, approved, roleID, test.actorID, test.auditID,
			); !errors.Is(err, ErrRevisionConflict) {
				t.Fatalf("foreign grant evidence error=%v", err)
			}
		})
	}
}

func TestPostgresStoreLegacyApprovedApplyAndPrivacy(t *testing.T) {
	fixture := newIdentityRegistryStoreFixture(t)
	fixture.seedOwner(t, "permission", "fixture.identity.profile")
	fixture.seedDeclaration(t, "permission", "fixture.identity.profile", 1, RegistryStateActive,
		"fixture.identity.profile@1", "c")
	fixture.seedPermissionCatalog(t, "fixture.identity.profile", "fixture.identity.profile@1", "c", 1)

	// Simulate a migrated 028 review-only approved row: approved, revision 2, no grant.
	suggestionID := fixture.seedSuggestion(t, "member")
	legacyAudit := fixture.insertLegacyReviewAudit(t, suggestionID, "member")
	if _, err := fixture.pool.Exec(fixture.ctx, `
		ALTER TABLE extension_permission_role_suggestions
		  DISABLE TRIGGER extension_permission_role_suggestion_update_valid
	`); err != nil {
		t.Fatal(err)
	}
	if _, err := fixture.pool.Exec(fixture.ctx, `
		UPDATE extension_permission_role_suggestions
		SET approval_state = 'approved',
		    revision = 2,
		    decided_by_user_id = $2,
		    decision_audit_event_id = $3,
		    decided_at = statement_timestamp(),
		    updated_at = statement_timestamp()
		WHERE id = $1
	`, suggestionID, fixture.actorID, legacyAudit); err != nil {
		t.Fatal(err)
	}
	if _, err := fixture.pool.Exec(fixture.ctx, `
		ALTER TABLE extension_permission_role_suggestions
		  ENABLE TRIGGER extension_permission_role_suggestion_update_valid
	`); err != nil {
		t.Fatal(err)
	}

	listed, err := fixture.store.ListRoleSuggestions(fixture.ctx, RoleSuggestionFilter{
		ApprovalState: RoleSuggestionApproved, Limit: 10,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(listed) != 1 || listed[0].Applied || listed[0].ApprovalState != RoleSuggestionApproved {
		t.Fatalf("legacy approved listing must expose Applied=false: %#v", listed)
	}

	auditBefore := fixture.countAuditEvents(t)
	if _, err := fixture.pool.Exec(fixture.ctx, `
		INSERT INTO role_permissions (role_id, permission_key)
		SELECT id, 'fixture.identity.profile' FROM roles WHERE key = 'member'
	`); err != nil {
		t.Fatal(err)
	}
	rolePermsBefore := fixture.countRolePermissions(t)
	applied, err := fixture.store.DecideRoleSuggestion(fixture.ctx, DecideRoleSuggestionInput{
		ID: suggestionID, ExpectedRevision: 2,
		ApprovalState: RoleSuggestionApproved, ActorUserID: fixture.actorID,
	})
	if err != nil {
		t.Fatal(err)
	}
	if !applied.Applied || applied.Revision != 2 ||
		applied.DecisionAuditEventID != legacyAudit ||
		applied.AppliedAuditEventID == legacyAudit ||
		applied.AppliedByUserID != fixture.actorID {
		t.Fatalf("legacy apply = %#v", applied)
	}
	if fixture.countRolePermissions(t) != rolePermsBefore {
		t.Fatal("legacy apply changed an existing mapping")
	}
	if fixture.countAuditEvents(t) != auditBefore+1 {
		t.Fatal("legacy apply did not write exactly one new audit")
	}
	fixture.assertAudit(t, applied.AppliedAuditEventID, applied,
		"identity.role_suggestion.approve", boolExpectation(false), true)

	otherActorID := fixture.seedRoleManager(t, "active")
	if _, err := fixture.store.DecideRoleSuggestion(fixture.ctx, DecideRoleSuggestionInput{
		ID: suggestionID, ExpectedRevision: 2,
		ApprovalState: RoleSuggestionApproved, ActorUserID: otherActorID,
	}); !errors.Is(err, ErrUnauthorized) {
		t.Fatalf("legacy apply replay by different actor error=%v", err)
	}
	replay, err := fixture.store.DecideRoleSuggestion(fixture.ctx, DecideRoleSuggestionInput{
		ID: suggestionID, ExpectedRevision: 2,
		ApprovalState: RoleSuggestionApproved, ActorUserID: fixture.actorID,
	})
	if err != nil {
		t.Fatal(err)
	}
	if !replay.Applied || replay.AppliedAuditEventID != applied.AppliedAuditEventID {
		t.Fatalf("legacy apply replay = %#v", replay)
	}
	if fixture.countAuditEvents(t) != auditBefore+1 || fixture.countGrants(t) != 1 {
		t.Fatal("legacy apply replay duplicated audit or grant")
	}

	canceledCtx, cancel := context.WithCancel(fixture.ctx)
	cancel()
	readback, err := fixture.store.readbackRoleSuggestionDecision(canceledCtx, DecideRoleSuggestionInput{
		ID: suggestionID, ExpectedRevision: 2,
		ApprovalState: RoleSuggestionApproved, ActorUserID: fixture.actorID,
	}, RoleSuggestionApproved, errors.New("ambiguous commit"))
	if err != nil || readback.AppliedAuditEventID != applied.AppliedAuditEventID {
		t.Fatalf("bounded ambiguous commit readback=%#v error=%v", readback, err)
	}
	if _, err := fixture.store.readbackRoleSuggestionDecision(canceledCtx, DecideRoleSuggestionInput{
		ID: suggestionID, ExpectedRevision: 2,
		ApprovalState: RoleSuggestionApproved, ActorUserID: otherActorID,
	}, RoleSuggestionApproved, errors.New("ambiguous commit")); !errors.Is(err, ErrUnauthorized) {
		t.Fatalf("ambiguous readback accepted different actor: %v", err)
	}

	if _, err := fixture.pool.Exec(fixture.ctx, `UPDATE users SET status = 'disabled' WHERE id = $1`, fixture.actorID); err != nil {
		t.Fatal(err)
	}
	if _, err := fixture.store.DecideRoleSuggestion(fixture.ctx, DecideRoleSuggestionInput{
		ID: suggestionID, ExpectedRevision: 2,
		ApprovalState: RoleSuggestionApproved, ActorUserID: fixture.actorID,
	}); !errors.Is(err, ErrUnauthorized) {
		t.Fatalf("disabled legacy apply actor replay error=%v", err)
	}
	if _, err := fixture.pool.Exec(fixture.ctx, `UPDATE users SET status = 'active' WHERE id = $1`, fixture.actorID); err != nil {
		t.Fatal(err)
	}
	if _, err := fixture.pool.Exec(fixture.ctx, `
		INSERT INTO user_permission_overrides (user_id, permission_key, effect)
		VALUES ($1, 'role.manage', 'deny')
	`, fixture.actorID); err != nil {
		t.Fatal(err)
	}
	if _, err := fixture.store.DecideRoleSuggestion(fixture.ctx, DecideRoleSuggestionInput{
		ID: suggestionID, ExpectedRevision: 2,
		ApprovalState: RoleSuggestionApproved, ActorUserID: fixture.actorID,
	}); !errors.Is(err, ErrUnauthorized) {
		t.Fatalf("revoked legacy apply actor replay error=%v", err)
	}
	if _, err := fixture.pool.Exec(fixture.ctx, `
		DELETE FROM user_permission_overrides
		WHERE user_id = $1 AND permission_key = 'role.manage'
	`, fixture.actorID); err != nil {
		t.Fatal(err)
	}

	// Privacy: erase actor; authority tables keep numeric ids; audit actor nulls.
	if _, err := fixture.pool.Exec(fixture.ctx, `DELETE FROM users WHERE id = $1`, fixture.actorID); err != nil {
		t.Fatalf("privacy actor deletion: %v", err)
	}
	var nulled, retainedDecider, retainedApplier int
	if err := fixture.pool.QueryRow(fixture.ctx, `
		SELECT
		  (SELECT count(*) FROM audit_events
		   WHERE id IN ($1, $2) AND actor_user_id IS NULL),
		  (SELECT count(*) FROM extension_permission_role_suggestions
		   WHERE id = $3 AND decided_by_user_id = $4),
		  (SELECT count(*) FROM extension_permission_role_grants
		   WHERE suggestion_id = $3 AND applied_by_user_id = $4)
	`, legacyAudit, applied.AppliedAuditEventID, suggestionID, fixture.actorID).Scan(
		&nulled, &retainedDecider, &retainedApplier,
	); err != nil {
		t.Fatal(err)
	}
	if nulled != 2 || retainedDecider != 1 || retainedApplier != 1 {
		t.Fatalf("privacy evidence nulled=%d decider=%d applier=%d", nulled, retainedDecider, retainedApplier)
	}
	if _, err := fixture.store.DecideRoleSuggestion(fixture.ctx, DecideRoleSuggestionInput{
		ID: suggestionID, ExpectedRevision: 2,
		ApprovalState: RoleSuggestionApproved, ActorUserID: fixture.actorID,
	}); !errors.Is(err, ErrUnauthorized) {
		t.Fatalf("deleted legacy apply actor replay error=%v", err)
	}
}

func TestPostgresStoreDecideRoleSuggestionConcurrentOneWinner(t *testing.T) {
	fixture := newIdentityRegistryStoreFixture(t)
	fixture.seedOwner(t, "permission", "fixture.identity.profile")
	fixture.seedDeclaration(t, "permission", "fixture.identity.profile", 1, RegistryStateActive,
		"fixture.identity.profile@1", "c")
	fixture.seedPermissionCatalog(t, "fixture.identity.profile", "fixture.identity.profile@1", "c", 1)
	suggestionID := fixture.seedSuggestion(t, "moderator")
	rolePermsBefore := fixture.countRolePermissions(t)
	auditBefore := fixture.countAuditEvents(t)

	const workers = 8
	start := make(chan struct{})
	results := make(chan error, workers)
	var wait sync.WaitGroup
	for range workers {
		wait.Add(1)
		go func() {
			defer wait.Done()
			<-start
			_, err := fixture.store.DecideRoleSuggestion(fixture.ctx, DecideRoleSuggestionInput{
				ID: suggestionID, ExpectedRevision: 1,
				ApprovalState: RoleSuggestionApproved, ActorUserID: fixture.actorID,
			})
			results <- err
		}()
	}
	close(start)
	wait.Wait()
	close(results)

	var winners, conflicts, other int
	for err := range results {
		switch {
		case err == nil:
			winners++
		case errors.Is(err, ErrRevisionConflict):
			conflicts++
		default:
			other++
			t.Errorf("unexpected concurrent error=%v", err)
		}
	}
	// Idempotent replay may let more than one worker observe success after the
	// winner commits; durable side effects must still be exactly-once.
	if winners < 1 || other != 0 {
		t.Fatalf("concurrent winners=%d conflicts=%d other=%d", winners, conflicts, other)
	}
	if fixture.countRolePermissions(t) != rolePermsBefore+1 {
		t.Fatalf("winning approval did not add exactly one role permission")
	}
	if fixture.countGrants(t) != 1 {
		t.Fatalf("winning approval grant count=%d", fixture.countGrants(t))
	}
	if fixture.countAuditEvents(t) != auditBefore+1 {
		t.Fatalf("winning approval audit count before=%d after=%d", auditBefore, fixture.countAuditEvents(t))
	}

	approved, err := scanRoleSuggestion(fixture.pool.QueryRow(fixture.ctx, roleSuggestionSelectSQL+` WHERE suggestion.id = $1`, suggestionID))
	if err != nil {
		t.Fatal(err)
	}
	if !approved.Applied || approved.ApprovalState != RoleSuggestionApproved {
		t.Fatalf("winner state=%#v", approved)
	}
	fixture.assertAudit(t, approved.DecisionAuditEventID, approved, "identity.role_suggestion.approve", boolExpectation(true), true)
}

func TestPostgresStoreApprovalRejectsUntrackedHostPermission(t *testing.T) {
	fixture := newIdentityRegistryStoreFixture(t)
	fixture.seedOwner(t, "permission", "fixture.identity.untracked")
	fixture.seedDeclaration(t, "permission", "fixture.identity.untracked", 1, RegistryStateActive,
		"fixture.identity.untracked@1", "b")
	// Host permission exists without catalog ownership — fail closed.
	if _, err := fixture.pool.Exec(fixture.ctx, `
		INSERT INTO permissions (key, module, description)
		VALUES ('fixture.identity.untracked', 'legacy', 'Untracked legacy permission')
	`); err != nil {
		t.Fatal(err)
	}
	suggestionID := fixture.seedSuggestionFor(
		t, "fixture.identity.untracked", "fixture.identity.untracked@1", "b", "member",
	)

	if _, err := fixture.store.DecideRoleSuggestion(fixture.ctx, DecideRoleSuggestionInput{
		ID: suggestionID, ExpectedRevision: 1,
		ApprovalState: RoleSuggestionApproved, ActorUserID: fixture.actorID,
	}); !errors.Is(err, ErrTargetConflict) {
		t.Fatalf("untracked Host permission approval error=%v", err)
	}
	var pending, catalogRows, grantRows int
	if err := fixture.pool.QueryRow(fixture.ctx, `
		SELECT
		  (SELECT count(*) FROM extension_permission_role_suggestions
		   WHERE id = $1 AND approval_state = 'pending'),
		  (SELECT count(*) FROM extension_permission_catalog
		   WHERE permission_key = 'fixture.identity.untracked'),
		  (SELECT count(*) FROM role_permissions
		   WHERE permission_key = 'fixture.identity.untracked')
	`, suggestionID).Scan(&pending, &catalogRows, &grantRows); err != nil {
		t.Fatal(err)
	}
	if pending != 1 || catalogRows != 0 || grantRows != 0 {
		t.Fatalf("pending=%d catalog=%d grants=%d", pending, catalogRows, grantRows)
	}
}

func TestPostgresStoreApprovalRequiresExistingCatalog(t *testing.T) {
	fixture := newIdentityRegistryStoreFixture(t)
	fixture.seedOwner(t, "permission", "fixture.identity.profile")
	fixture.seedDeclaration(t, "permission", "fixture.identity.profile", 1, RegistryStateActive,
		"fixture.identity.profile@1", "c")
	// No catalog registration — approval must not create it.
	suggestionID := fixture.seedSuggestion(t, "member")
	if _, err := fixture.store.DecideRoleSuggestion(fixture.ctx, DecideRoleSuggestionInput{
		ID: suggestionID, ExpectedRevision: 1,
		ApprovalState: RoleSuggestionApproved, ActorUserID: fixture.actorID,
	}); !errors.Is(err, ErrTargetConflict) {
		t.Fatalf("missing catalog approval error=%v", err)
	}
	var catalogRows, permissionRows, grantRows int
	if err := fixture.pool.QueryRow(fixture.ctx, `
		SELECT
		  (SELECT count(*) FROM extension_permission_catalog
		   WHERE permission_key = 'fixture.identity.profile'),
		  (SELECT count(*) FROM permissions
		   WHERE key = 'fixture.identity.profile'),
		  (SELECT count(*) FROM extension_permission_role_grants)
	`).Scan(&catalogRows, &permissionRows, &grantRows); err != nil {
		t.Fatal(err)
	}
	if catalogRows != 0 || permissionRows != 0 || grantRows != 0 {
		t.Fatalf("approval created catalog/permission/grant: catalog=%d perms=%d grants=%d",
			catalogRows, permissionRows, grantRows)
	}
}

func TestPostgresStoreSuperAdminDenyAndDisabledSuperAdmin(t *testing.T) {
	fixture := newIdentityRegistryStoreFixture(t)
	fixture.seedOwner(t, "permission", "fixture.identity.profile")
	fixture.seedDeclaration(t, "permission", "fixture.identity.profile", 1, RegistryStateActive,
		"fixture.identity.profile@1", "c")
	fixture.seedPermissionCatalog(t, "fixture.identity.profile", "fixture.identity.profile@1", "c", 1)

	// Active super_admin remains non-deniable even with a direct deny override.
	var superAdminID int64
	if err := fixture.pool.QueryRow(fixture.ctx, `
		INSERT INTO users (username, username_lower, email, email_lower, display_name, status)
		VALUES ('super_actor', 'super_actor', 'super@example.test', 'super@example.test', 'Super', 'active')
		RETURNING id
	`).Scan(&superAdminID); err != nil {
		t.Fatal(err)
	}
	if _, err := fixture.pool.Exec(fixture.ctx, `
		INSERT INTO user_roles (user_id, role_id)
		SELECT $1, id FROM roles WHERE key = 'super_admin'
	`, superAdminID); err != nil {
		t.Fatal(err)
	}
	if _, err := fixture.pool.Exec(fixture.ctx, `
		INSERT INTO user_permission_overrides (user_id, permission_key, effect)
		VALUES ($1, 'role.manage', 'deny')
	`, superAdminID); err != nil {
		t.Fatal(err)
	}
	approveID := fixture.seedSuggestion(t, "member")
	if _, err := fixture.store.DecideRoleSuggestion(fixture.ctx, DecideRoleSuggestionInput{
		ID: approveID, ExpectedRevision: 1,
		ApprovalState: RoleSuggestionApproved, ActorUserID: superAdminID,
	}); err != nil {
		t.Fatalf("active super_admin with deny must still manage roles: %v", err)
	}

	// Disabled super_admin role does not authorize.
	var disabledSuperID int64
	if err := fixture.pool.QueryRow(fixture.ctx, `
		INSERT INTO users (username, username_lower, email, email_lower, display_name, status)
		VALUES ('disabled_super', 'disabled_super', 'ds@example.test', 'ds@example.test', 'DS', 'active')
		RETURNING id
	`).Scan(&disabledSuperID); err != nil {
		t.Fatal(err)
	}
	if _, err := fixture.pool.Exec(fixture.ctx, `
		INSERT INTO roles (key, is_enabled) VALUES ('super_admin_disabled_probe', TRUE)
		ON CONFLICT DO NOTHING
	`); err != nil {
		t.Fatal(err)
	}
	// Attach the real super_admin role then disable it.
	if _, err := fixture.pool.Exec(fixture.ctx, `
		INSERT INTO user_roles (user_id, role_id)
		SELECT $1, id FROM roles WHERE key = 'super_admin'
	`, disabledSuperID); err != nil {
		t.Fatal(err)
	}
	if _, err := fixture.pool.Exec(fixture.ctx, `
		UPDATE roles SET is_enabled = FALSE WHERE key = 'super_admin'
	`); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_, _ = fixture.pool.Exec(fixture.ctx, `UPDATE roles SET is_enabled = TRUE WHERE key = 'super_admin'`)
	})
	deniedID := fixture.seedSuggestion(t, "operator")
	if _, err := fixture.store.DecideRoleSuggestion(fixture.ctx, DecideRoleSuggestionInput{
		ID: deniedID, ExpectedRevision: 1,
		ApprovalState: RoleSuggestionApproved, ActorUserID: disabledSuperID,
	}); !errors.Is(err, ErrUnauthorized) {
		t.Fatalf("disabled super_admin role error=%v", err)
	}
}
