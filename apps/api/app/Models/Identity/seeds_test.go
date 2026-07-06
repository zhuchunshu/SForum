package identity

import "testing"

func TestSystemRoles(t *testing.T) {
	if RoleSuperAdmin != "super_admin" {
		t.Fatalf("unexpected super admin role key: %q", RoleSuperAdmin)
	}
	if RoleMember != "member" {
		t.Fatalf("unexpected member role key: %q", RoleMember)
	}
}

func TestDefaultPermissionsContainAdminAccess(t *testing.T) {
	found := false
	for _, permission := range SeedPermissions {
		if permission.Key == PermissionAdminAccess {
			found = true
		}
		if permission.Key == "" || permission.Module == "" {
			t.Fatalf("permission must have key and module: %#v", permission)
		}
	}
	if !found {
		t.Fatal("expected admin.access seed permission")
	}
}

func TestDefaultPermissionsContainTagManage(t *testing.T) {
	found := false
	for _, permission := range SeedPermissions {
		if permission.Key == PermissionTagManage {
			found = true
			if permission.Module != "forum" {
				t.Fatalf("expected tag.manage to belong to forum module, got %q", permission.Module)
			}
		}
	}
	if !found {
		t.Fatal("expected tag.manage seed permission")
	}
}
