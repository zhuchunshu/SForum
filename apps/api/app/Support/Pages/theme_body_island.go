package pages

import "strings"

// RequiredThemeBodyIslandTag 返回可替换公开页在 L1 模板中应嵌入的宿主 body 岛标签。
// 不可替换页返回空串。
func RequiredThemeBodyIslandTag(pageID string) string {
	switch strings.TrimSpace(pageID) {
	case "forum.home":
		return "sf-home-page"
	case "forum.category.index":
		return "sf-category-index-page"
	case "forum.category.show":
		return "sf-category-show-page"
	case "forum.tag.index":
		return "sf-tag-index-page"
	case "forum.tag.show":
		return "sf-tag-show-page"
	case "forum.topic.show":
		return "sf-topic-show-page"
	case "forum.topic.create":
		return "sf-topic-composer"
	case "forum.topic.reply":
		return "sf-topic-reply"
	case "forum.topic.edit":
		return "sf-topic-editor"
	case "forum.profile.show":
		return "sf-profile-page"
	case "forum.settings.profile":
		return "sf-profile-settings"
	case "forum.settings.security":
		return "sf-security-settings"
	case "forum.settings.notifications":
		return "sf-notification-settings"
	case "forum.notifications":
		return "sf-notifications-page"
	case "auth.login":
		return "sf-login-form"
	case "auth.register":
		return "sf-register-form"
	case "auth.forgot_password":
		return "sf-recovery-request"
	case "auth.reset_password":
		return "sf-recovery-confirm"
	case "site.terms":
		return "sf-terms-page"
	case "site.privacy":
		return "sf-privacy-page"
	case "site.guidelines":
		return "sf-guidelines-page"
	case "system.forbidden", "system.not_found", "system.rate_limited", "system.server_error":
		return "sf-error-details"
	default:
		return ""
	}
}
