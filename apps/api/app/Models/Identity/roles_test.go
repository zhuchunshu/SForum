package identity

import (
	"context"
	"slices"
	"testing"
)

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

func TestCreateRoleRejectsBlankRequiredFields(t *testing.T) {
	service, _ := newTestService(t)
	admin := Actor{ID: 1, Status: UserStatusActive, RoleKeys: []string{RoleSuperAdmin}}

	for _, input := range []RoleInput{
		{Key: "", Alias: "版主"},
		{Key: "moderator", Alias: ""},
		{Key: "  ", Alias: "  "},
	} {
		if _, err := service.CreateRole(testContext(t), admin, input); err == nil {
			t.Fatalf("expected CreateRole to reject blank fields for %#v", input)
		}
	}
}

func TestCreateRoleNormalizesInput(t *testing.T) {
	service, _ := newTestService(t)
	admin := Actor{ID: 1, Status: UserStatusActive, RoleKeys: []string{RoleSuperAdmin}}

	role, err := service.CreateRole(testContext(t), admin, RoleInput{
		Key:         " moderator ",
		Alias:       " 版主 ",
		Description: " 管理内容 ",
	})
	if err != nil {
		t.Fatalf("CreateRole returned error: %v", err)
	}
	if role.Key != "moderator" || role.Alias != "版主" || role.Description != "管理内容" {
		t.Fatalf("expected normalized role fields, got %#v", role)
	}
}

func TestDirectAllowAddsUserPermission(t *testing.T) {
	service, _ := newTestService(t)
	ctx := testContext(t)
	admin := Actor{ID: 1, Status: UserStatusActive, RoleKeys: []string{RoleSuperAdmin}}
	member := registerMemberForPermissionTest(t, service, ctx)

	detail, err := service.ReplaceUserPermissionOverrides(ctx, admin, member.ID, PermissionOverrides{
		Allow: []string{PermissionAdminAccess},
	})
	if err != nil {
		t.Fatalf("ReplaceUserPermissionOverrides returned error: %v", err)
	}
	if !slices.Contains(detail.Permissions, PermissionAdminAccess) {
		t.Fatalf("expected direct allow to add admin access, got %v", detail.Permissions)
	}
}

func TestDirectDenyOverridesRolePermission(t *testing.T) {
	service, _ := newTestService(t)
	ctx := testContext(t)
	admin := Actor{ID: 1, Status: UserStatusActive, RoleKeys: []string{RoleSuperAdmin}}
	member := registerMemberForPermissionTest(t, service, ctx)

	detail, err := service.ReplaceUserPermissionOverrides(ctx, admin, member.ID, PermissionOverrides{
		Deny: []string{PermissionTopicCreate},
	})
	if err != nil {
		t.Fatalf("ReplaceUserPermissionOverrides returned error: %v", err)
	}
	if slices.Contains(detail.Permissions, PermissionTopicCreate) {
		t.Fatalf("expected direct deny to remove topic create, got %v", detail.Permissions)
	}
}

func TestNonAdminCannotManageUserPermissions(t *testing.T) {
	service, _ := newTestService(t)
	member := Actor{ID: 2, Status: UserStatusActive, Permissions: map[string]bool{}}

	_, err := service.ReplaceUserPermissionOverrides(testContext(t), member, 2, PermissionOverrides{
		Allow: []string{PermissionAdminAccess},
	})
	if err != ErrPermissionDenied {
		t.Fatalf("expected permission denied, got %v", err)
	}
}

func TestInitialSuperAdminCannotLoseSuperAdminRole(t *testing.T) {
	service, _ := newTestService(t)
	ctx := testContext(t)
	adminUser, err := service.Register(ctx, RegisterInput{
		Username: "admin",
		Email:    "admin@example.com",
		Password: "correct horse battery staple",
	})
	if err != nil {
		t.Fatalf("Register returned error: %v", err)
	}
	// 用另一个 super_admin（不同 ID）操作 initial super_admin target，避免触发 self-change 检查。
	otherSuperAdmin := Actor{ID: adminUser.ID + 100, Status: UserStatusActive, RoleKeys: []string{RoleSuperAdmin}}

	_, err = service.ReplaceUserRoles(ctx, otherSuperAdmin, adminUser.ID, []string{RoleMember})
	if err != ErrInitialSuperAdminLocked {
		t.Fatalf("expected initial super admin lock, got %v", err)
	}
}

func TestUnknownPermissionIsRejected(t *testing.T) {
	service, _ := newTestService(t)
	ctx := testContext(t)
	admin := Actor{ID: 1, Status: UserStatusActive, RoleKeys: []string{RoleSuperAdmin}}
	member := registerMemberForPermissionTest(t, service, ctx)

	_, err := service.ReplaceUserPermissionOverrides(ctx, admin, member.ID, PermissionOverrides{
		Allow: []string{"unknown.permission"},
	})
	if err != ErrInvalidPermission {
		t.Fatalf("expected invalid permission, got %v", err)
	}
}

func TestUserPermissionOverrideConflictIsRejected(t *testing.T) {
	service, _ := newTestService(t)
	ctx := testContext(t)
	admin := Actor{ID: 1, Status: UserStatusActive, RoleKeys: []string{RoleSuperAdmin}}
	member := registerMemberForPermissionTest(t, service, ctx)

	_, err := service.ReplaceUserPermissionOverrides(ctx, admin, member.ID, PermissionOverrides{
		Allow: []string{PermissionPostCreate},
		Deny:  []string{PermissionPostCreate},
	})
	if err != ErrPermissionOverrideConflict {
		t.Fatalf("expected permission override conflict, got %v", err)
	}
}

func TestSuperAdminPermissionOverridesAreLocked(t *testing.T) {
	service, _ := newTestService(t)
	ctx := testContext(t)
	adminUser, err := service.Register(ctx, RegisterInput{
		Username: "admin",
		Email:    "admin@example.com",
		Password: "correct horse battery staple",
	})
	if err != nil {
		t.Fatalf("Register returned error: %v", err)
	}
	// 用另一个 super_admin（不同 ID）操作 target，避免 self-change 检查。
	otherSuperAdmin := Actor{ID: adminUser.ID + 100, Status: UserStatusActive, RoleKeys: []string{RoleSuperAdmin}}

	_, err = service.ReplaceUserPermissionOverrides(ctx, otherSuperAdmin, adminUser.ID, PermissionOverrides{
		Deny: []string{PermissionAdminAccess},
	})
	if err != ErrSuperAdminOverridesLocked {
		t.Fatalf("expected super admin overrides lock, got %v", err)
	}
}

func registerMemberForPermissionTest(t *testing.T, service *Service, ctx context.Context) CurrentUser {
	t.Helper()
	if _, err := service.Register(ctx, RegisterInput{
		Username: "admin",
		Email:    "admin@example.com",
		Password: "correct horse battery staple",
	}); err != nil {
		t.Fatalf("first Register returned error: %v", err)
	}
	member, err := service.Register(ctx, RegisterInput{
		Username: "member",
		Email:    "member@example.com",
		Password: "correct horse battery staple",
	})
	if err != nil {
		t.Fatalf("second Register returned error: %v", err)
	}
	return member
}

// TestUserManagerCannotChangeOwnRoles 验证 H1：持有 user.manage 的非 super_admin
// 不能修改自己的角色，防止自我提权到 super_admin。
func TestUserManagerCannotChangeOwnRoles(t *testing.T) {
	service, _ := newTestService(t)
	ctx := testContext(t)
	// manager 有 user.manage 权限但不是 super_admin。
	manager := Actor{ID: 50, Status: UserStatusActive, Permissions: map[string]bool{PermissionUserManage: true}}

	_, err := service.ReplaceUserRoles(ctx, manager, 50, []string{RoleSuperAdmin})
	if err != ErrSelfRoleChange {
		t.Fatalf("expected ErrSelfRoleChange, got %v", err)
	}
}

// TestUserManagerCannotChangeOwnOverrides 验证 H1：禁止改自己的权限覆盖。
func TestUserManagerCannotChangeOwnOverrides(t *testing.T) {
	service, _ := newTestService(t)
	ctx := testContext(t)
	manager := Actor{ID: 50, Status: UserStatusActive, Permissions: map[string]bool{PermissionUserManage: true}}

	_, err := service.ReplaceUserPermissionOverrides(ctx, manager, 50, PermissionOverrides{
		Allow: []string{PermissionAdminAccess},
	})
	if err != ErrSelfRoleChange {
		t.Fatalf("expected ErrSelfRoleChange, got %v", err)
	}
}

// TestNonSuperAdminCannotGrantSuperAdminRole 验证 H1：非 super_admin 的 user.manage
// 持有者不能把 super_admin 角色授予他人（即使 target 不是自己）。
func TestNonSuperAdminCannotGrantSuperAdminRole(t *testing.T) {
	service, _ := newTestService(t)
	ctx := testContext(t)
	// 先注册一个首 super_admin（让 store 里有 super_admin 角色），再注册一个普通 member 作为 target。
	_, _ = service.Register(ctx, RegisterInput{
		Username: "firstadmin",
		Email:    "firstadmin@example.com",
		Password: "correct horse battery staple",
	})
	target, err := service.Register(ctx, RegisterInput{
		Username: "target",
		Email:    "target@example.com",
		Password: "correct horse battery staple",
	})
	if err != nil {
		t.Fatalf("Register returned error: %v", err)
	}
	// manager 有 user.manage 但非 super_admin，尝试给 target 加 super_admin。
	manager := Actor{ID: 60, Status: UserStatusActive, Permissions: map[string]bool{PermissionUserManage: true}}

	_, err = service.ReplaceUserRoles(ctx, manager, target.ID, []string{RoleSuperAdmin})
	if err != ErrSuperAdminGrantRestricted {
		t.Fatalf("expected ErrSuperAdminGrantRestricted, got %v", err)
	}
}

// TestSuperAdminCanGrantSuperAdminRole 验证 H1：super_admin 可以把 super_admin 授予他人。
func TestSuperAdminCanGrantSuperAdminRole(t *testing.T) {
	service, _ := newTestService(t)
	ctx := testContext(t)
	_, _ = service.Register(ctx, RegisterInput{
		Username: "firstadmin",
		Email:    "firstadmin@example.com",
		Password: "correct horse battery staple",
	})
	target, err := service.Register(ctx, RegisterInput{
		Username: "target2",
		Email:    "target2@example.com",
		Password: "correct horse battery staple",
	})
	if err != nil {
		t.Fatalf("Register returned error: %v", err)
	}
	// 另一个 super_admin（ID 与 target 不同）操作 target。
	superAdmin := Actor{ID: 70, Status: UserStatusActive, RoleKeys: []string{RoleSuperAdmin}}

	_, err = service.ReplaceUserRoles(ctx, superAdmin, target.ID, []string{RoleSuperAdmin})
	if err != nil {
		t.Fatalf("expected super_admin to grant super_admin role, got %v", err)
	}
}
