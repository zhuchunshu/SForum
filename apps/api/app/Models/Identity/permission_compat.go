package identity

import "slices"

// 旧版“大包”权限 → 细粒度子权限。
// 持有父权限时：Can(子权限) 通过，且会话权限列表会展开出子权限（供前端菜单使用）。
// 不反向：仅有子权限时 Can(父权限) 仍为 false，避免邮件运营误得全站设置权。
var legacyPermissionChildren = map[string][]string{
	PermissionSettingsManage: {
		PermissionSettingsSiteManage,
		PermissionSettingsMailManage,
		PermissionSettingsAvatarManage,
		PermissionSettingsAppearanceManage,
		PermissionForumSettingsManage,
	},
	PermissionExtensionManage: {
		PermissionExtensionView,
		PermissionExtensionPluginManage,
		PermissionExtensionThemeManage,
		PermissionExtensionReleaseManage,
	},
	// user.manage 仅展开只读视图，不展开 user.permission_override：
	// 个人权限例外是高危能力，必须显式单独授予，不能由运营父权限继承。
	PermissionUserManage: {
		PermissionUserView,
	},
}

// ExpandEffectivePermissions 在角色/覆盖结果上展开父权限对应的子权限。
// 用于写入会话与后台展示的有效权限列表。
func ExpandEffectivePermissions(keys []string) []string {
	set := make(map[string]bool, len(keys)+16)
	for _, key := range keys {
		if key == "" {
			continue
		}
		set[key] = true
	}
	for parent, children := range legacyPermissionChildren {
		if !set[parent] {
			continue
		}
		for _, child := range children {
			set[child] = true
		}
	}
	out := make([]string, 0, len(set))
	for key := range set {
		out = append(out, key)
	}
	slices.Sort(out)
	return out
}

// hasExpandedPermission 判断 actor 是否直接持有 permission，或通过父权限继承。
func hasExpandedPermission(perms map[string]bool, permission string) bool {
	if perms[permission] {
		return true
	}
	for parent, children := range legacyPermissionChildren {
		if !perms[parent] {
			continue
		}
		for _, child := range children {
			if child == permission {
				return true
			}
		}
	}
	return false
}
