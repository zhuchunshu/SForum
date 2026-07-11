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

func TestModerationPermissionsRemainIndependent(t *testing.T) {
	found := map[string]bool{}
	for _, permission := range SeedPermissions {
		if permission.Module == "moderation" {
			found[permission.Key] = true
		}
	}

	if !found[PermissionModerationManage] {
		t.Fatal("expected moderation.manage seed permission")
	}
	if !found[PermissionModerationReview] {
		t.Fatal("expected moderation.review seed permission")
	}
	if PermissionModerationManage == PermissionModerationReview {
		t.Fatal("moderation permissions must remain independent")
	}
}

func TestSeedPermissionsCoverCurrentAdminAndForumSurfaces(t *testing.T) {
	required := []string{
		PermissionAdminAccess,
		PermissionRoleManage,
		PermissionUserView,
		PermissionUserManage,
		PermissionUserPermissionOverride,
		PermissionUserBan,
		PermissionCategoryManage,
		PermissionTagManage,
		PermissionTopicCreate,
		PermissionTopicEditOwn,
		PermissionTopicEditAny,
		PermissionTopicDeleteOwn,
		PermissionTopicDeleteAny,
		PermissionTopicLock,
		PermissionTopicPin,
		PermissionPostCreate,
		PermissionPostEditOwn,
		PermissionPostEditAny,
		PermissionPostDeleteOwn,
		PermissionPostDeleteAny,
		PermissionModerationManage,
		PermissionModerationReview,
		PermissionSettingsManage,
		PermissionSettingsSiteManage,
		PermissionSettingsMailManage,
		PermissionSettingsAvatarManage,
		PermissionSettingsAppearanceManage,
		PermissionForumSettingsManage,
		PermissionSEOManage,
		PermissionAttachmentUpload,
		PermissionAttachmentManage,
		PermissionAttachmentSettings,
		PermissionExtensionManage,
		PermissionExtensionView,
		PermissionExtensionPluginManage,
		PermissionExtensionThemeManage,
		PermissionExtensionReleaseManage,
		PermissionDatabaseManage,
		PermissionSearchManage,
		PermissionJobsView,
		PermissionJobsManage,
	}

	found := map[string]SeedPermission{}
	for _, permission := range SeedPermissions {
		if _, exists := found[permission.Key]; exists {
			t.Fatalf("duplicate seed permission key: %s", permission.Key)
		}
		found[permission.Key] = permission
	}

	for _, key := range required {
		permission, ok := found[key]
		if !ok {
			t.Fatalf("expected seed permission %s", key)
		}
		if permission.Module == "" || permission.Description == "" {
			t.Fatalf("permission %s must declare module and description", key)
		}
	}

	if len(found) != len(required) {
		t.Fatalf("seed catalog size drifted: got %d want %d", len(found), len(required))
	}
}
