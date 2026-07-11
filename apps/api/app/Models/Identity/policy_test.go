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


func TestLegacySettingsManageImpliesMailManage(t *testing.T) {
	actor := Actor{
		ID:          4,
		Status:      UserStatusActive,
		Permissions: map[string]bool{PermissionSettingsManage: true},
	}
	if !actor.Can(PermissionSettingsMailManage) {
		t.Fatal("expected settings.manage parent to satisfy settings.mail.manage")
	}
	if actor.Can(PermissionSEOManage) {
		t.Fatal("settings.manage must not imply unrelated permissions")
	}
}

func TestExpandEffectivePermissionsAddsChildren(t *testing.T) {
	got := ExpandEffectivePermissions([]string{PermissionUserManage, PermissionPostCreate})
	// user.manage 只展开 user.view，不得展开 user.permission_override。
	wantChildren := []string{PermissionUserView, PermissionUserManage, PermissionPostCreate}
	for _, key := range wantChildren {
		found := false
		for _, item := range got {
			if item == key {
				found = true
				break
			}
		}
		if !found {
			t.Fatalf("expected expanded list to contain %s, got %v", key, got)
		}
	}
	for _, item := range got {
		if item == PermissionUserPermissionOverride {
			t.Fatalf("user.manage must not expand to %s, got %v", PermissionUserPermissionOverride, got)
		}
	}
}

// TestUserManageDoesNotGrantPermissionOverride 回归：仅有 user.manage 时
// Can(user.permission_override) 必须为 false，防止运营通过兼容展开提权。
func TestUserManageDoesNotGrantPermissionOverride(t *testing.T) {
	actor := Actor{
		ID:     1,
		Status: UserStatusActive,
		Permissions: map[string]bool{
			PermissionUserManage: true,
		},
	}
	if !actor.Can(PermissionUserManage) {
		t.Fatal("expected Can(user.manage)")
	}
	if !actor.Can(PermissionUserView) {
		t.Fatal("expected user.manage to expand to user.view")
	}
	if actor.Can(PermissionUserPermissionOverride) {
		t.Fatal("user.manage must not imply user.permission_override")
	}
}
