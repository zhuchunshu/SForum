package identity

import (
	"slices"
	"testing"
)

func TestListActiveUserIDsWithPermissionTxHonorsEffectiveRBAC(t *testing.T) {
	fixture := newIdentityPersistencePGFixture(t)
	const permission = PermissionModerationReview
	if _, err := fixture.pool.Exec(fixture.ctx, `INSERT INTO permissions (key,module,description) VALUES ($1,'moderation','review') ON CONFLICT DO NOTHING`, permission); err != nil {
		t.Fatal(err)
	}
	if _, err := fixture.pool.Exec(fixture.ctx, `
		INSERT INTO role_permissions (role_id,permission_key)
		SELECT id,$1 FROM roles WHERE key='identity_reviewer'
		ON CONFLICT DO NOTHING;
		INSERT INTO user_permission_overrides (user_id,permission_key,effect)
		VALUES ($2,$1,'allow'),($3,$1,'deny')
		ON CONFLICT (user_id,permission_key) DO UPDATE SET effect=EXCLUDED.effect
	`, permission, fixture.targetUserID, fixture.adminUserID); err != nil {
		t.Fatal(err)
	}
	tx, err := fixture.pool.Begin(fixture.ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer tx.Rollback(fixture.ctx)
	got, err := NewPostgresStore(fixture.pool).ListActiveUserIDsWithPermissionTx(fixture.ctx, tx, permission)
	if err != nil {
		t.Fatal(err)
	}
	want := []int64{fixture.adminUserID, fixture.actorUserID, fixture.targetUserID}
	slices.Sort(want)
	if !slices.Equal(got, want) {
		t.Fatalf("permission recipients=%v want=%v", got, want)
	}

	if _, err := tx.Exec(fixture.ctx, `
		INSERT INTO user_permission_overrides (user_id,permission_key,effect)
		VALUES ($1,$3,'deny')
		ON CONFLICT (user_id,permission_key) DO UPDATE SET effect='deny';
		UPDATE users SET status='disabled' WHERE id=$2
	`, fixture.actorUserID, fixture.targetUserID, permission); err != nil {
		t.Fatal(err)
	}
	got, err = NewPostgresStore(fixture.pool).ListActiveUserIDsWithPermissionTx(fixture.ctx, tx, permission)
	if err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(got, []int64{fixture.adminUserID}) {
		t.Fatalf("denied/inactive recipients=%v want super admin %d", got, fixture.adminUserID)
	}
}
