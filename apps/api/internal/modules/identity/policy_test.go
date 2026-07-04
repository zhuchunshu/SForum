package identity

import "testing"

func TestSuperAdminCanUseEveryPermission(t *testing.T) {
	actor := Actor{
		ID:       1,
		Status:   UserStatusActive,
		RoleKeys: []string{RoleSuperAdmin},
	}

	if !actor.Can("made.up.permission") {
		t.Fatal("expected super admin to pass any permission")
	}
}

func TestInactiveActorCannotUsePermissions(t *testing.T) {
	actor := Actor{
		ID:          2,
		Status:      UserStatusDisabled,
		Permissions: map[string]bool{PermissionAdminAccess: true},
	}

	if actor.Can(PermissionAdminAccess) {
		t.Fatal("expected disabled actor to fail permission check")
	}
}

func TestMemberCanEditOwnPost(t *testing.T) {
	actor := Actor{
		ID:          3,
		Status:      UserStatusActive,
		Permissions: map[string]bool{PermissionPostEditOwn: true},
	}
	post := PostSummary{ID: 10, AuthorUserID: 3}

	if !CanEditPost(actor, post) {
		t.Fatal("expected actor to edit own post")
	}
}

func TestMemberCannotEditOtherPostWithoutAnyPermission(t *testing.T) {
	actor := Actor{
		ID:          3,
		Status:      UserStatusActive,
		Permissions: map[string]bool{PermissionPostEditOwn: true},
	}
	post := PostSummary{ID: 10, AuthorUserID: 4}

	if CanEditPost(actor, post) {
		t.Fatal("expected actor not to edit someone else's post")
	}
}
