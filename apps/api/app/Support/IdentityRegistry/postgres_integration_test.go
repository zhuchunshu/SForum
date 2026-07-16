package identityregistry

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"

	"github.com/zhuchunshu/sforum/apps/api/database/migrations"
)

const identityRegistryOwnershipMigrationVersion = int64(202607160028)

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

func TestPostgresStoreDecideRoleSuggestionApproveRejectAndGuards(t *testing.T) {
	fixture := newIdentityRegistryStoreFixture(t)
	fixture.seedOwner(t, "permission", "fixture.identity.profile")
	fixture.seedDeclaration(t, "permission", "fixture.identity.profile", 1, RegistryStateActive,
		"fixture.identity.profile@1", "c")

	approveID := fixture.seedSuggestion(t, "member")
	rejectID := fixture.seedSuggestion(t, "operator")
	unknownID := fixture.seedSuggestion(t, "future_role")
	rolePermsBefore := fixture.countRolePermissions(t)

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
		if listed[index-1].RoleKey > listed[index].RoleKey {
			t.Fatalf("list order not deterministic: %#v", listed)
		}
	}

	approved, err := fixture.store.DecideRoleSuggestion(fixture.ctx, DecideRoleSuggestionInput{
		ID: approveID, ExpectedRevision: 1, ApprovalState: RoleSuggestionApproved, ActorUserID: fixture.actorID,
	})
	if err != nil {
		t.Fatal(err)
	}
	if approved.ApprovalState != RoleSuggestionApproved || approved.Revision != 2 ||
		approved.DecidedByUserID != fixture.actorID || approved.DecisionAuditEventID <= 0 ||
		approved.DecidedAt == nil {
		t.Fatalf("approved = %#v", approved)
	}
	fixture.assertAudit(t, approved.DecisionAuditEventID, fixture.actorID, "identity.role_suggestion.approve")

	rejected, err := fixture.store.DecideRoleSuggestion(fixture.ctx, DecideRoleSuggestionInput{
		ID: rejectID, ExpectedRevision: 1, ApprovalState: RoleSuggestionRejected, ActorUserID: fixture.actorID,
	})
	if err != nil {
		t.Fatal(err)
	}
	if rejected.ApprovalState != RoleSuggestionRejected || rejected.Revision != 2 ||
		rejected.DecisionAuditEventID <= 0 {
		t.Fatalf("rejected = %#v", rejected)
	}
	fixture.assertAudit(t, rejected.DecisionAuditEventID, fixture.actorID, "identity.role_suggestion.reject")

	if _, err := fixture.store.DecideRoleSuggestion(fixture.ctx, DecideRoleSuggestionInput{
		ID: approveID, ExpectedRevision: 1, ApprovalState: RoleSuggestionApproved, ActorUserID: fixture.actorID,
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
	}); !errors.Is(err, ErrInvalid) {
		t.Fatalf("inactive actor error=%v", err)
	}
	if _, err := fixture.pool.Exec(fixture.ctx, `UPDATE users SET status = 'active' WHERE id = $1`, fixture.actorID); err != nil {
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
	if rolePermsAfter != rolePermsBefore {
		t.Fatalf("role_permissions changed %d -> %d", rolePermsBefore, rolePermsAfter)
	}
}

func TestPostgresStoreDecideRoleSuggestionConcurrentOneWinner(t *testing.T) {
	fixture := newIdentityRegistryStoreFixture(t)
	fixture.seedOwner(t, "permission", "fixture.identity.profile")
	fixture.seedDeclaration(t, "permission", "fixture.identity.profile", 1, RegistryStateActive,
		"fixture.identity.profile@1", "c")
	suggestionID := fixture.seedSuggestion(t, "moderator")
	rolePermsBefore := fixture.countRolePermissions(t)

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
	if winners != 1 || conflicts != workers-1 || other != 0 {
		t.Fatalf("concurrent winners=%d conflicts=%d other=%d", winners, conflicts, other)
	}

	var state string
	var auditID, actorID int64
	if err := fixture.pool.QueryRow(fixture.ctx, `
		SELECT approval_state, decided_by_user_id, decision_audit_event_id
		FROM extension_permission_role_suggestions WHERE id = $1
	`, suggestionID).Scan(&state, &actorID, &auditID); err != nil {
		t.Fatal(err)
	}
	if state != RoleSuggestionApproved || actorID != fixture.actorID || auditID <= 0 {
		t.Fatalf("winner state=%s actor=%d audit=%d", state, actorID, auditID)
	}
	fixture.assertAudit(t, auditID, fixture.actorID, "identity.role_suggestion.approve")
	if got := fixture.countRolePermissions(t); got != rolePermsBefore {
		t.Fatalf("role_permissions changed %d -> %d", rolePermsBefore, got)
	}
}

type identityRegistryStoreFixture struct {
	t       *testing.T
	ctx     context.Context
	admin   *pgxpool.Pool
	pool    *pgxpool.Pool
	store   *PostgresStore
	schema  string
	actorID int64
}

func newIdentityRegistryStoreFixture(t *testing.T) *identityRegistryStoreFixture {
	t.Helper()
	databaseURL := strings.TrimSpace(os.Getenv("SFORUM_TEST_DATABASE_URL"))
	if databaseURL == "" {
		databaseURL = strings.TrimSpace(os.Getenv("DATABASE_URL"))
	}
	if databaseURL == "" {
		t.Skip("SFORUM_TEST_DATABASE_URL or DATABASE_URL is required")
	}

	ctx := context.Background()
	admin, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		t.Fatal(err)
	}
	schema := fmt.Sprintf("identity_registry_%d", time.Now().UnixNano())
	quoted := pgx.Identifier{schema}.Sanitize()
	if _, err := admin.Exec(ctx, "CREATE SCHEMA "+quoted); err != nil {
		admin.Close()
		t.Fatal(err)
	}
	removeSchema := func() {
		_, _ = admin.Exec(context.Background(), "DROP SCHEMA IF EXISTS "+quoted+" CASCADE")
	}

	sqlConfig, err := pgx.ParseConfig(databaseURL)
	if err != nil {
		removeSchema()
		admin.Close()
		t.Fatal(err)
	}
	sqlConfig.RuntimeParams["search_path"] = schema + ",public"
	db := stdlib.OpenDB(*sqlConfig)
	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)
	if err := seedIdentityRegistryStoreBaseTables(ctx, db); err != nil {
		db.Close()
		removeSchema()
		admin.Close()
		t.Fatal(err)
	}
	if err := applyIdentityRegistryOwnershipMigration(ctx, db); err != nil {
		db.Close()
		removeSchema()
		admin.Close()
		t.Fatal(err)
	}
	db.Close()

	poolConfig, err := pgxpool.ParseConfig(databaseURL)
	if err != nil {
		removeSchema()
		admin.Close()
		t.Fatal(err)
	}
	poolConfig.ConnConfig.RuntimeParams["search_path"] = schema + ",public"
	pool, err := pgxpool.NewWithConfig(ctx, poolConfig)
	if err != nil {
		removeSchema()
		admin.Close()
		t.Fatal(err)
	}

	fixture := &identityRegistryStoreFixture{
		t: t, ctx: ctx, admin: admin, pool: pool,
		store: NewPostgresStore(pool), schema: schema,
	}
	if err := pool.QueryRow(ctx, `
		INSERT INTO users (username, username_lower, email, email_lower, display_name, status)
		VALUES ($1, $1, $2, $2, 'Identity Registry Actor', 'active')
		RETURNING id
	`, "identity_registry_"+schema, "identity_registry_"+schema+"@example.test").Scan(&fixture.actorID); err != nil {
		pool.Close()
		removeSchema()
		admin.Close()
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `
		INSERT INTO extensions (id, type, name, status) VALUES
			('fixture.identity', 'plugin', 'Identity Fixture', 'enabled')
	`); err != nil {
		pool.Close()
		removeSchema()
		admin.Close()
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `
		INSERT INTO extension_versions (
			id, extension_id, version, manifest, package_path, package_digest
		) VALUES (
			101, 'fixture.identity', '1.0.0', '{}'::jsonb, '/tmp/identity-v1', repeat('a', 64)
		)
	`); err != nil {
		pool.Close()
		removeSchema()
		admin.Close()
		t.Fatal(err)
	}

	t.Cleanup(func() {
		pool.Close()
		removeSchema()
		admin.Close()
	})
	return fixture
}

func seedIdentityRegistryStoreBaseTables(ctx context.Context, db *sql.DB) error {
	_, err := db.ExecContext(ctx, `
		CREATE TABLE users (
			id BIGSERIAL PRIMARY KEY,
			username TEXT NOT NULL,
			username_lower TEXT NOT NULL UNIQUE,
			email TEXT NOT NULL,
			email_lower TEXT NOT NULL UNIQUE,
			display_name TEXT NOT NULL,
			status TEXT NOT NULL DEFAULT 'active'
			  CHECK (status IN ('active', 'disabled', 'banned'))
		);
		CREATE TABLE roles (
			id BIGSERIAL PRIMARY KEY,
			key TEXT NOT NULL UNIQUE
		);
		CREATE TABLE permissions (
			key TEXT PRIMARY KEY,
			module TEXT NOT NULL,
			description TEXT NOT NULL DEFAULT ''
		);
		CREATE TABLE role_permissions (
			role_id BIGINT NOT NULL REFERENCES roles(id) ON DELETE CASCADE,
			permission_key TEXT NOT NULL REFERENCES permissions(key) ON DELETE CASCADE,
			PRIMARY KEY (role_id, permission_key)
		);
		CREATE TABLE user_roles (
			user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
			role_id BIGINT NOT NULL REFERENCES roles(id) ON DELETE RESTRICT,
			PRIMARY KEY (user_id, role_id)
		);
		CREATE TABLE user_permission_overrides (
			user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
			permission_key TEXT NOT NULL REFERENCES permissions(key) ON DELETE CASCADE,
			effect TEXT NOT NULL CHECK (effect IN ('allow', 'deny')),
			PRIMARY KEY (user_id, permission_key)
		);
		CREATE TABLE audit_events (
			id BIGSERIAL PRIMARY KEY,
			actor_user_id BIGINT REFERENCES users(id) ON DELETE SET NULL,
			target_user_id BIGINT REFERENCES users(id) ON DELETE SET NULL,
			action TEXT NOT NULL,
			metadata JSONB NOT NULL DEFAULT '{}'::jsonb,
			created_at TIMESTAMPTZ NOT NULL DEFAULT now()
		);
		CREATE TABLE extensions (
			id TEXT PRIMARY KEY,
			type TEXT NOT NULL CHECK (type IN ('plugin', 'theme')),
			name TEXT NOT NULL,
			status TEXT NOT NULL DEFAULT 'installed'
			  CHECK (status IN ('installed', 'enabled', 'disabled'))
		);
		CREATE TABLE extension_versions (
			id BIGINT PRIMARY KEY,
			extension_id TEXT NOT NULL REFERENCES extensions(id) ON DELETE CASCADE,
			version TEXT NOT NULL,
			manifest JSONB NOT NULL,
			package_path TEXT NOT NULL,
			package_digest TEXT NOT NULL
		);
		INSERT INTO roles (key) VALUES ('super_admin'), ('member'), ('operator'), ('moderator');
		INSERT INTO permissions (key, module, description)
		VALUES ('topic.create', 'forum', 'Create topics');
		INSERT INTO role_permissions (role_id, permission_key)
		SELECT id, 'topic.create' FROM roles WHERE key = 'member';
	`)
	return err
}

func applyIdentityRegistryOwnershipMigration(ctx context.Context, db *sql.DB) error {
	provider, err := goose.NewProvider(
		goose.DialectPostgres, db, migrations.Files(), goose.WithDisableGlobalRegistry(true),
	)
	if err != nil {
		return err
	}
	_, err = provider.ApplyVersion(ctx, identityRegistryOwnershipMigrationVersion, true)
	return err
}

func (f *identityRegistryStoreFixture) seedOwner(t *testing.T, kind, stableID string) {
	t.Helper()
	if _, err := f.pool.Exec(f.ctx, `
		INSERT INTO extension_identity_registry_owners (
			identity_kind, stable_id, owner_extension_id
		) VALUES ($1, $2, 'fixture.identity')
	`, kind, stableID); err != nil {
		t.Fatal(err)
	}
}

func (f *identityRegistryStoreFixture) seedDeclaration(
	t *testing.T,
	kind, stableID string,
	revision int64,
	state, contractVersion, declarationByte string,
) {
	t.Helper()
	if _, err := f.pool.Exec(f.ctx, `
		INSERT INTO extension_identity_registry_declarations (
			identity_kind, stable_id, owner_extension_id, revision, registry_state,
			extension_version_id, extension_version, package_digest,
			contract_version, declaration_digest
		) VALUES (
			$1, $2, 'fixture.identity', $3, $4,
			101, '1.0.0', repeat('a', 64),
			$5, repeat($6, 64)
		)
	`, kind, stableID, revision, state, contractVersion, declarationByte); err != nil {
		t.Fatal(err)
	}
}

func (f *identityRegistryStoreFixture) seedSuggestion(t *testing.T, roleKey string) int64 {
	t.Helper()
	var id int64
	if err := f.pool.QueryRow(f.ctx, `
		INSERT INTO extension_permission_role_suggestions (
			permission_key, owner_extension_id, extension_version_id,
			extension_version, package_digest, permission_contract_version,
			declaration_digest, role_key
		) VALUES (
			'fixture.identity.profile', 'fixture.identity', 101,
			'1.0.0', repeat('a', 64), 'fixture.identity.profile@1',
			repeat('c', 64), $1
		)
		RETURNING id
	`, roleKey).Scan(&id); err != nil {
		t.Fatal(err)
	}
	return id
}

func (f *identityRegistryStoreFixture) countRolePermissions(t *testing.T) int {
	t.Helper()
	var count int
	if err := f.pool.QueryRow(f.ctx, `SELECT count(*) FROM role_permissions`).Scan(&count); err != nil {
		t.Fatal(err)
	}
	return count
}

func (f *identityRegistryStoreFixture) assertAudit(t *testing.T, auditID, actorID int64, action string) {
	t.Helper()
	var gotActor int64
	var gotAction string
	if err := f.pool.QueryRow(f.ctx, `
		SELECT COALESCE(actor_user_id, 0), action FROM audit_events WHERE id = $1
	`, auditID).Scan(&gotActor, &gotAction); err != nil {
		t.Fatal(err)
	}
	if gotActor != actorID || gotAction != action {
		t.Fatalf("audit id=%d actor=%d action=%q want actor=%d action=%q",
			auditID, gotActor, gotAction, actorID, action)
	}
}
