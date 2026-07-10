package identity

const (
	RoleSuperAdmin = "super_admin"
	RoleMember     = "member"

	PermissionAdminAccess      = "admin.access"
	PermissionRoleManage       = "role.manage"
	PermissionUserManage       = "user.manage"
	PermissionUserBan          = "user.ban"
	PermissionCategoryManage   = "category.manage"
	PermissionTagManage        = "tag.manage"
	PermissionTopicCreate      = "topic.create"
	PermissionTopicEditAny     = "topic.edit_any"
	PermissionTopicDeleteAny   = "topic.delete_any"
	PermissionTopicLock        = "topic.lock"
	PermissionTopicPin         = "topic.pin"
	PermissionPostCreate       = "post.create"
	PermissionPostEditOwn      = "post.edit_own"
	PermissionPostEditAny      = "post.edit_any"
	PermissionPostDeleteOwn    = "post.delete_own"
	PermissionPostDeleteAny    = "post.delete_any"
	PermissionModerationManage = "moderation.manage"
	PermissionModerationReview = "moderation.review"
	// PermissionModerationReportReview 保留为源码兼容别名；数据库权限已迁移为 moderation.review。
	PermissionModerationReportReview = PermissionModerationReview
	PermissionSettingsManage         = "settings.manage"
	PermissionSEOManage              = "seo.manage"
	PermissionAttachmentUpload       = "attachment.upload"
	PermissionAttachmentManage       = "attachment.manage"
	PermissionAttachmentSettings     = "attachment.settings.manage"
	PermissionExtensionManage        = "extension.manage"
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
	{Key: PermissionUserManage, Module: "identity", Description: "Manage user accounts and assignments."},
	{Key: PermissionUserBan, Module: "identity", Description: "Ban users from participating."},
	{Key: PermissionCategoryManage, Module: "forum", Description: "Create and update categories."},
	{Key: PermissionTagManage, Module: "forum", Description: "Create, approve, disable, and manage tags."},
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
	{Key: PermissionModerationManage, Module: "moderation", Description: "Manage moderation settings and audit history."},
	{Key: PermissionModerationReview, Module: "moderation", Description: "Review pending content and moderation reports."},
	{Key: PermissionSettingsManage, Module: "admin", Description: "Manage system settings."},
	{Key: PermissionSEOManage, Module: "admin", Description: "Manage search engine optimization settings."},
	{Key: PermissionAttachmentUpload, Module: "attachment", Description: "Upload attachments."},
	{Key: PermissionAttachmentManage, Module: "attachment", Description: "Manage uploaded attachments."},
	{Key: PermissionAttachmentSettings, Module: "attachment", Description: "Manage attachment storage and upload settings."},
	{Key: PermissionExtensionManage, Module: "extension", Description: "Install and manage extensions and themes."},
	{Key: PermissionDatabaseManage, Module: "admin", Description: "Browse database tables and rows."},
	{Key: PermissionSearchManage, Module: "search", Description: "Rebuild and manage the search index."},
	{Key: PermissionJobsView, Module: "jobs", Description: "View background jobs, queues, failures, and worker activity."},
	{Key: PermissionJobsManage, Module: "jobs", Description: "Retry, cancel, pause, and resume background job processing."},
}
