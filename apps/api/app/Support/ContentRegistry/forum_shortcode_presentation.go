package contentregistry

import "strings"

// WrapForumShortcodeHTML adds the Host-owned presentation hook around one
// shortcode occurrence. The plugin HTML remains semantic and is sanitized
// before this wrapper is added. Only the fixed registry IDs below influence
// the class list; callers cannot inject arbitrary class names.
func WrapForumShortcodeHTML(id, value string, fallback bool) string {
	if strings.TrimSpace(value) == "" {
		return ""
	}

	classes := "sf-shortcode"
	switch id {
	case ShortcodeUserID:
		classes += " sf-shortcode--reference sf-shortcode--user"
	case ShortcodeTopicID:
		classes += " sf-shortcode--reference sf-shortcode--topic"
	case ShortcodeCommentID:
		classes += " sf-shortcode--reference sf-shortcode--comment"
	case ShortcodeCategoryID:
		classes += " sf-shortcode--reference sf-shortcode--category"
	case ShortcodeFriendLinksID:
		classes += " sf-shortcode--reference sf-shortcode--friend-links"
	case ShortcodeLoginID:
		classes += " sf-shortcode--block sf-shortcode--protected sf-shortcode--login"
	case ShortcodeReplyID:
		classes += " sf-shortcode--block sf-shortcode--protected sf-shortcode--reply"
	case ShortcodeOnlyAuthorID:
		classes += " sf-shortcode--block sf-shortcode--protected sf-shortcode--only-author"
	default:
		classes += " sf-shortcode--reference"
	}
	if fallback {
		classes += " sf-shortcode--fallback"
	}
	return `<div class="` + classes + `">` + value + `</div>`
}
