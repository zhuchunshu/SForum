package identity

import "testing"

func TestMemberAliasCanChangeButRoleCannotBeDeleted(t *testing.T) {
	service, store := newTestService(t)
	admin := Actor{ID: 1, Status: UserStatusActive, RoleKeys: []string{RoleSuperAdmin}}

	role, err := service.UpdateRole(testContext(t), admin, RoleMember, RoleInput{
		Alias:       "注册用户",
		Description: "所有开放注册用户",
	})
	if err != nil {
		t.Fatalf("UpdateRole returned error: %v", err)
	}
	if role.Key != RoleMember || role.Alias != "注册用户" {
		t.Fatalf("unexpected role after update: %#v", role)
	}

	err = service.DeleteRole(testContext(t), admin, RoleMember)
	if err != ErrDefaultRoleLocked {
		t.Fatalf("expected default role lock, got %v; store=%#v", err, store)
	}
}

func TestNonAdminCannotManageRoles(t *testing.T) {
	service, _ := newTestService(t)
	member := Actor{ID: 2, Status: UserStatusActive, Permissions: map[string]bool{}}

	_, err := service.CreateRole(testContext(t), member, RoleInput{
		Key:         "moderator",
		Alias:       "版主",
		Description: "管理内容",
	})
	if err != ErrPermissionDenied {
		t.Fatalf("expected permission denied, got %v", err)
	}
}
