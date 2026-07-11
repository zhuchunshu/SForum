package identity

const (
	RoleSuperAdmin = "super_admin"
	RoleMember     = "member"

	PermissionAdminAccess = "admin.access"
	PermissionRoleManage  = "role.manage"
	// PermissionUserView 只读用户列表/详情；不含改组与权限例外。
	PermissionUserView = "user.view"
	// PermissionUserManage 管理用户状态与用户组分配（不含个人权限例外）。
	PermissionUserManage = "user.manage"
	// PermissionUserPermissionOverride 编辑用户直接权限例外（高危，单独授予）。
	PermissionUserPermissionOverride = "user.permission_override"
	PermissionUserBan                = "user.ban"
	PermissionCategoryManage         = "category.manage"
	PermissionTagManage              = "tag.manage"
	PermissionTopicCreate            = "topic.create"
	// PermissionTopicEditOwn 作者编辑自己的主题（与回复的 post.edit_own 分离）。
	PermissionTopicEditOwn   = "topic.edit_own"
	PermissionTopicEditAny   = "topic.edit_any"
	PermissionTopicDeleteOwn = "topic.delete_own"
	PermissionTopicDeleteAny = "topic.delete_any"
	PermissionTopicLock      = "topic.lock"
	PermissionTopicPin       = "topic.pin"
	PermissionPostCreate     = "post.create"
	PermissionPostEditOwn    = "post.edit_own"
	PermissionPostEditAny    = "post.edit_any"
	PermissionPostDeleteOwn  = "post.delete_own"
	PermissionPostDeleteAny  = "post.delete_any"
	PermissionModerationManage = "moderation.manage"
	PermissionModerationReview = "moderation.review"
	// PermissionModerationReportReview 保留为源码兼容别名；数据库权限已迁移为 moderation.review。
	PermissionModerationReportReview = PermissionModerationReview
	// PermissionSettingsManage 为兼容父权限；细粒度见 settings.* / forum.settings.manage。
	PermissionSettingsManage           = "settings.manage"
	PermissionSettingsSiteManage       = "settings.site.manage"
	PermissionSettingsMailManage       = "settings.mail.manage"
	PermissionSettingsAvatarManage     = "settings.avatar.manage"
	PermissionSettingsAppearanceManage = "settings.appearance.manage"
	PermissionForumSettingsManage      = "forum.settings.manage"
	PermissionSEOManage                = "seo.manage"
	PermissionAttachmentUpload         = "attachment.upload"
	PermissionAttachmentManage         = "attachment.manage"
	PermissionAttachmentSettings       = "attachment.settings.manage"
	// PermissionExtensionManage 为兼容父权限；细粒度见 extension.view/plugin/theme/release。
	PermissionExtensionManage        = "extension.manage"
	PermissionExtensionView          = "extension.view"
	PermissionExtensionPluginManage  = "extension.plugin.manage"
	PermissionExtensionThemeManage   = "extension.theme.manage"
	PermissionExtensionReleaseManage = "extension.release.manage"
	PermissionDatabaseManage         = "database.manage"
	PermissionSearchManage           = "search.manage"
	PermissionJobsView               = "jobs.view"
	PermissionJobsManage             = "jobs.manage"
)

type SeedPermission struct {
	Key         string
	Module      string
	Description string
}

var SeedPermissions = []SeedPermission{
	{Key: PermissionAdminAccess, Module: "admin", Description: "Access the admin area."},
	{Key: PermissionRoleManage, Module: "identity", Description: "Create and update roles and role permissions."},
	{Key: PermissionUserView, Module: "identity", Description: "View user accounts without changing assignments."},
	{Key: PermissionUserManage, Module: "identity", Description: "Manage user accounts and role assignments."},
	{Key: PermissionUserPermissionOverride, Module: "identity", Description: "Edit per-user permission allow and deny overrides."},
	{Key: PermissionUserBan, Module: "identity", Description: "Ban users from participating."},
	{Key: PermissionCategoryManage, Module: "forum", Description: "Create and update categories."},
	{Key: PermissionTagManage, Module: "forum", Description: "Create, approve, disable, and manage tags."},
	{Key: PermissionTopicCreate, Module: "forum", Description: "Create topics."},
	{Key: PermissionTopicEditOwn, Module: "forum", Description: "Edit own topics."},
	{Key: PermissionTopicEditAny, Module: "forum", Description: "Edit any topic."},
	{Key: PermissionTopicDeleteOwn, Module: "forum", Description: "Delete own topics."},
	{Key: PermissionTopicDeleteAny, Module: "forum", Description: "Delete any topic."},
	{Key: PermissionTopicLock, Module: "forum", Description: "Lock or unlock topics."},
	{Key: PermissionTopicPin, Module: "forum", Description: "Pin or unpin topics."},
	{Key: PermissionPostCreate, Module: "forum", Description: "Create posts."},
	{Key: PermissionPostEditOwn, Module: "forum", Description: "Edit own replies."},
	{Key: PermissionPostEditAny, Module: "forum", Description: "Edit any post."},
	{Key: PermissionPostDeleteOwn, Module: "forum", Description: "Delete own replies."},
	{Key: PermissionPostDeleteAny, Module: "forum", Description: "Delete any post."},
	{Key: PermissionModerationManage, Module: "moderation", Description: "Manage moderation settings and audit history."},
	{Key: PermissionModerationReview, Module: "moderation", Description: "Review pending content and moderation reports."},
	{Key: PermissionSettingsManage, Module: "admin", Description: "Legacy parent: manage all non-SEO site settings groups."},
	{Key: PermissionSettingsSiteManage, Module: "admin", Description: "Manage core site identity, locale, verification, and security settings."},
	{Key: PermissionSettingsMailManage, Module: "admin", Description: "Manage mail providers, notification policy, and delivery tests."},
	{Key: PermissionSettingsAvatarManage, Module: "admin", Description: "Manage avatar upload and default avatar settings."},
	{Key: PermissionSettingsAppearanceManage, Module: "admin", Description: "Manage appearance theme and public chrome personalization."},
	{Key: PermissionForumSettingsManage, Module: "forum", Description: "Manage forum runtime limits, reading, and behavior settings."},
	{Key: PermissionSEOManage, Module: "admin", Description: "Manage search engine optimization settings."},
	{Key: PermissionAttachmentUpload, Module: "attachment", Description: "Upload attachments."},
	{Key: PermissionAttachmentManage, Module: "attachment", Description: "Manage uploaded attachments."},
	{Key: PermissionAttachmentSettings, Module: "attachment", Description: "Manage attachment storage and upload settings."},
	{Key: PermissionExtensionManage, Module: "extension", Description: "Legacy parent: manage all extension capabilities."},
	{Key: PermissionExtensionView, Module: "extension", Description: "View installed extensions, events, and contributions."},
	{Key: PermissionExtensionPluginManage, Module: "extension", Description: "Enable, disable, and configure plugins."},
	{Key: PermissionExtensionThemeManage, Module: "extension", Description: "Activate and manage themes."},
	{Key: PermissionExtensionReleaseManage, Module: "extension", Description: "Build and activate trusted admin web releases."},
	{Key: PermissionDatabaseManage, Module: "admin", Description: "Browse database tables and rows."},
	{Key: PermissionSearchManage, Module: "search", Description: "Rebuild and manage the search index."},
	{Key: PermissionJobsView, Module: "jobs", Description: "View background jobs, queues, failures, and worker activity."},
	{Key: PermissionJobsManage, Module: "jobs", Description: "Retry, cancel, pause, and resume background job processing."},
}
