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
	admin := Actor{ID: adminUser.ID, Status: UserStatusActive, RoleKeys: []string{RoleSuperAdmin}}

	_, err = service.ReplaceUserRoles(ctx, admin, adminUser.ID, []string{RoleMember})
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
	admin := Actor{ID: adminUser.ID, Status: UserStatusActive, RoleKeys: []string{RoleSuperAdmin}}

	_, err = service.ReplaceUserPermissionOverrides(ctx, admin, adminUser.ID, PermissionOverrides{
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
