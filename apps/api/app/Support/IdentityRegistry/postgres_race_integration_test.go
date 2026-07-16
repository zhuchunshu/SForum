package identityregistry

import (
	"context"
	"encoding/json"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

func TestPostgresStoreConcurrentRoleReplacementAndApproval(t *testing.T) {
	for _, test := range []struct {
		name             string
		replacement      []string
		wantFinalMapping int
	}{
		{
			name:             "full set includes approved mapping",
			replacement:      []string{"topic.create", "fixture.identity.profile"},
			wantFinalMapping: 1,
		},
		{
			name:             "full set excludes approved mapping",
			replacement:      []string{"topic.create"},
			wantFinalMapping: 0,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			fixture := newIdentityRegistryStoreFixture(t)
			fixture.seedOwner(t, "permission", "fixture.identity.profile")
			fixture.seedDeclaration(t, "permission", "fixture.identity.profile", 1, RegistryStateActive,
				"fixture.identity.profile@1", "c")
			fixture.seedPermissionCatalog(t, "fixture.identity.profile", "fixture.identity.profile@1", "c", 1)
			suggestionID := fixture.seedSuggestion(t, "member")

			ctx, cancel := context.WithTimeout(fixture.ctx, 10*time.Second)
			defer cancel()
			start := make(chan struct{})
			var wait sync.WaitGroup
			var approved RoleSuggestion
			var approveErr, replaceErr error
			wait.Add(2)
			go func() {
				defer wait.Done()
				<-start
				approved, approveErr = fixture.store.DecideRoleSuggestion(ctx, DecideRoleSuggestionInput{
					ID: suggestionID, ExpectedRevision: 1,
					ApprovalState: RoleSuggestionApproved, ActorUserID: fixture.actorID,
				})
			}()
			go func() {
				defer wait.Done()
				<-start
				replaceErr = replaceRolePermissionsForTest(
					ctx, fixture.pool, fixture.actorID, "member", test.replacement,
				)
			}()
			close(start)
			wait.Wait()
			if ctx.Err() != nil {
				t.Fatalf("approval/replacement did not finish within deadline: %v", ctx.Err())
			}
			if approveErr != nil || replaceErr != nil {
				t.Fatalf("approval error=%v replacement error=%v", approveErr, replaceErr)
			}

			// A racing additive approval and full-set replacement are serialized, but
			// either may commit last. Apply the same full set once more to assert the
			// public replacement contract deterministically: include retains it;
			// exclude revokes it without deleting immutable approval evidence.
			if err := replaceRolePermissionsForTest(
				ctx, fixture.pool, fixture.actorID, "member", test.replacement,
			); err != nil {
				t.Fatal(err)
			}
			if got := fixture.countRolePermissionMapping(t, "member", "fixture.identity.profile"); got != test.wantFinalMapping {
				t.Fatalf("final approved mapping=%d want %d", got, test.wantFinalMapping)
			}
			if fixture.countGrants(t) != 1 {
				t.Fatalf("grant evidence count=%d", fixture.countGrants(t))
			}
			// Which writer commits first is intentionally nondeterministic here. The
			// controlled new-mapping and pre-existing-mapping tests above pin true
			// and false respectively; this race only requires a typed provenance bit.
			fixture.assertAudit(t, approved.DecisionAuditEventID, approved,
				"identity.role_suggestion.approve", nil, true)

			var approvalAudits, replacementAudits int
			if err := fixture.pool.QueryRow(fixture.ctx, `
				SELECT
				  count(*) FILTER (WHERE action = 'identity.role_suggestion.approve'),
				  count(*) FILTER (WHERE action = 'role.permissions.replace')
				FROM audit_events
			`).Scan(&approvalAudits, &replacementAudits); err != nil {
				t.Fatal(err)
			}
			if approvalAudits != 1 || replacementAudits != 2 {
				t.Fatalf("approval audits=%d replacement audits=%d", approvalAudits, replacementAudits)
			}

			auditsBeforeReplay := fixture.countAuditEvents(t)
			replay, err := fixture.store.DecideRoleSuggestion(ctx, DecideRoleSuggestionInput{
				ID: suggestionID, ExpectedRevision: 1,
				ApprovalState: RoleSuggestionApproved, ActorUserID: fixture.actorID,
			})
			if err != nil || !replay.Applied {
				t.Fatalf("idempotent replay=%#v error=%v", replay, err)
			}
			if fixture.countAuditEvents(t) != auditsBeforeReplay || fixture.countGrants(t) != 1 {
				t.Fatal("idempotent replay duplicated audit or grant evidence")
			}
			if got := fixture.countRolePermissionMapping(t, "member", "fixture.identity.profile"); got != test.wantFinalMapping {
				t.Fatalf("idempotent replay changed replacement result=%d want %d", got, test.wantFinalMapping)
			}
		})
	}
}

func replaceRolePermissionsForTest(
	ctx context.Context,
	pool *pgxpool.Pool,
	actorUserID int64,
	roleKey string,
	permissions []string,
) error {
	tx, err := pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(context.Background()) }()

	var roleID int64
	if err := tx.QueryRow(ctx, `SELECT id FROM roles WHERE key = $1`, roleKey).Scan(&roleID); err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, `DELETE FROM role_permissions WHERE role_id = $1`, roleID); err != nil {
		return err
	}
	for _, permissionKey := range permissions {
		if _, err := tx.Exec(ctx, `
			INSERT INTO role_permissions (role_id, permission_key) VALUES ($1, $2)
		`, roleID, permissionKey); err != nil {
			return err
		}
	}
	metadata, err := json.Marshal(map[string]any{
		"roleKey": roleKey, "permissions": permissions,
	})
	if err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, `
		INSERT INTO audit_events (actor_user_id, action, metadata)
		VALUES ($1, 'role.permissions.replace', $2::jsonb)
	`, actorUserID, metadata); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func TestPostgresStoreExactArtifactDisableRace(t *testing.T) {
	fixture := newIdentityRegistryStoreFixture(t)
	fixture.seedOwner(t, "permission", "fixture.identity.profile")
	fixture.seedDeclaration(t, "permission", "fixture.identity.profile", 1, RegistryStateActive,
		"fixture.identity.profile@1", "c")
	fixture.seedPermissionCatalog(t, "fixture.identity.profile", "fixture.identity.profile@1", "c", 1)
	suggestionID := fixture.seedSuggestion(t, "member")

	start := make(chan struct{})
	var wait sync.WaitGroup
	var decideErr, disableErr error
	wait.Add(2)
	go func() {
		defer wait.Done()
		<-start
		_, decideErr = fixture.store.DecideRoleSuggestion(fixture.ctx, DecideRoleSuggestionInput{
			ID: suggestionID, ExpectedRevision: 1,
			ApprovalState: RoleSuggestionApproved, ActorUserID: fixture.actorID,
		})
	}()
	go func() {
		defer wait.Done()
		<-start
		// Disable the exact artifact after a brief delay to race publication.
		time.Sleep(5 * time.Millisecond)
		_, disableErr = fixture.pool.Exec(fixture.ctx, `
			UPDATE extensions SET status = 'disabled', active_version_id = NULL
			WHERE id = 'fixture.identity'
		`)
	}()
	close(start)
	wait.Wait()
	if disableErr != nil {
		t.Fatalf("disable race: %v", disableErr)
	}
	if decideErr != nil && !errors.Is(decideErr, ErrStale) && !errors.Is(decideErr, ErrRevisionConflict) {
		t.Fatalf("unexpected decide error during disable race: %v", decideErr)
	}
	var state string
	var grants int
	if err := fixture.pool.QueryRow(fixture.ctx, `
		SELECT approval_state,
		       (SELECT count(*) FROM extension_permission_role_grants WHERE suggestion_id = $1)
		FROM extension_permission_role_suggestions WHERE id = $1
	`, suggestionID).Scan(&state, &grants); err != nil {
		t.Fatal(err)
	}
	if state == RoleSuggestionApproved && grants != 1 {
		t.Fatalf("approved without grant evidence after disable race: state=%s grants=%d", state, grants)
	}
	if state == RoleSuggestionPending && grants != 0 {
		t.Fatalf("pending suggestion has grant after failed race: grants=%d", grants)
	}
}

func TestPostgresStoreActivePermissionRevokeSerialization(t *testing.T) {
	fixture := newIdentityRegistryStoreFixture(t)
	fixture.seedOwner(t, "permission", "fixture.identity.profile")
	fixture.seedDeclaration(t, "permission", "fixture.identity.profile", 1, RegistryStateActive,
		"fixture.identity.profile@1", "c")
	fixture.seedPermissionCatalog(t, "fixture.identity.profile", "fixture.identity.profile@1", "c", 1)
	suggestionID := fixture.seedSuggestion(t, "member")

	// Approve once so the mapping exists, then race revoke against a second suggestion.
	if _, err := fixture.store.DecideRoleSuggestion(fixture.ctx, DecideRoleSuggestionInput{
		ID: suggestionID, ExpectedRevision: 1,
		ApprovalState: RoleSuggestionApproved, ActorUserID: fixture.actorID,
	}); err != nil {
		t.Fatal(err)
	}
	secondID := fixture.seedSuggestion(t, "operator")

	start := make(chan struct{})
	var wait sync.WaitGroup
	var approveErr, revokeErr error
	wait.Add(2)
	go func() {
		defer wait.Done()
		<-start
		_, approveErr = fixture.store.DecideRoleSuggestion(fixture.ctx, DecideRoleSuggestionInput{
			ID: secondID, ExpectedRevision: 1,
			ApprovalState: RoleSuggestionApproved, ActorUserID: fixture.actorID,
		})
	}()
	go func() {
		defer wait.Done()
		<-start
		_, revokeErr = fixture.pool.Exec(fixture.ctx, `
			DELETE FROM role_permissions
			WHERE permission_key = 'role.manage'
			  AND role_id = (SELECT id FROM roles WHERE key = 'identity_reviewer')
		`)
	}()
	close(start)
	wait.Wait()
	if revokeErr != nil {
		t.Fatalf("revoke race: %v", revokeErr)
	}
	// After revoke, actor may still have won the race with authority, or fail unauthorized.
	if approveErr != nil &&
		!errors.Is(approveErr, ErrUnauthorized) &&
		!errors.Is(approveErr, ErrRevisionConflict) {
		t.Fatalf("unexpected approve error during revoke race: %v", approveErr)
	}
	var grants int
	if err := fixture.pool.QueryRow(fixture.ctx, `
		SELECT count(*) FROM extension_permission_role_grants WHERE suggestion_id = $1
	`, secondID).Scan(&grants); err != nil {
		t.Fatal(err)
	}
	if approveErr == nil && grants != 1 {
		t.Fatalf("approved second suggestion without grant evidence")
	}
	if errors.Is(approveErr, ErrUnauthorized) && grants != 0 {
		t.Fatalf("unauthorized approve left grant evidence")
	}
}
