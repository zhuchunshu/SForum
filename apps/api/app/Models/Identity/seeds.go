package identity

const (
	RoleSuperAdmin = "super_admin"
	RoleMember     = "member"

	PermissionAdminAccess            = "admin.access"
	PermissionRoleManage             = "role.manage"
	PermissionUserManage             = "user.manage"
	PermissionUserBan                = "user.ban"
	PermissionCategoryManage         = "category.manage"
	PermissionTopicCreate            = "topic.create"
	PermissionTopicEditAny           = "topic.edit_any"
	PermissionTopicDeleteAny         = "topic.delete_any"
	PermissionTopicLock              = "topic.lock"
	PermissionTopicPin               = "topic.pin"
	PermissionPostCreate             = "post.create"
	PermissionPostEditOwn            = "post.edit_own"
	PermissionPostEditAny            = "post.edit_any"
	PermissionPostDeleteOwn          = "post.delete_own"
	PermissionPostDeleteAny          = "post.delete_any"
	PermissionModerationReportReview = "moderation.report_review"
	PermissionSettingsManage         = "settings.manage"
)

type SeedPermission struct {
	Key         string
	Module      string
	Description string
}

var SeedPermissions = []SeedPermission{
	{Key: PermissionAdminAccess, Module: "admin", Description: "Access the admin area."},
	{Key: PermissionRoleManage, Module: "identity", Description: "Create and update roles and role permissions."},
	{Key: PermissionUserManage, Module: "identity", Description: "Manage user accounts and assignments."},
	{Key: PermissionUserBan, Module: "identity", Description: "Ban users from participating."},
	{Key: PermissionCategoryManage, Module: "forum", Description: "Create and update categories."},
	{Key: PermissionTopicCreate, Module: "forum", Description: "Create topics."},
	{Key: PermissionTopicEditAny, Module: "forum", Description: "Edit any topic."},
	{Key: PermissionTopicDeleteAny, Module: "forum", Description: "Delete any topic."},
	{Key: PermissionTopicLock, Module: "forum", Description: "Lock or unlock topics."},
	{Key: PermissionTopicPin, Module: "forum", Description: "Pin or unpin topics."},
	{Key: PermissionPostCreate, Module: "forum", Description: "Create posts."},
	{Key: PermissionPostEditOwn, Module: "forum", Description: "Edit own posts."},
	{Key: PermissionPostEditAny, Module: "forum", Description: "Edit any post."},
	{Key: PermissionPostDeleteOwn, Module: "forum", Description: "Delete own posts."},
	{Key: PermissionPostDeleteAny, Module: "forum", Description: "Delete any post."},
	{Key: PermissionModerationReportReview, Module: "moderation", Description: "Review moderation reports."},
	{Key: PermissionSettingsManage, Module: "admin", Description: "Manage system settings."},
}
