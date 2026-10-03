package main

import "strings"

// renderCopy carries the plugin-owned localized copy for successful renders.
// Fallback copy for runtime-unavailable states is Host-owned; this map only
// supplies labels and unavailable text that appear when the Host DID invoke
// the exact runtime and the target is simply not public.
type renderCopy struct {
	userCard           string
	topicCard          string
	commentCard        string
	categoryCard       string
	friendLinksTitle   string
	genericUnavailable string
	topicUnavailable   string
	commentUnavailable string
}

var (
	simplifiedChineseCopy = renderCopy{
		userCard:           "用户",
		topicCard:          "主题",
		commentCard:        "评论",
		categoryCard:       "分类",
		friendLinksTitle:   "友情链接",
		genericUnavailable: "该内容暂不可用",
		topicUnavailable:   "该主题不存在或暂不可见",
		commentUnavailable: "该评论不存在或暂不可见",
	}
	englishCopy = renderCopy{
		userCard:           "User",
		topicCard:          "Topic",
		commentCard:        "Comment",
		categoryCard:       "Category",
		friendLinksTitle:   "Friend links",
		genericUnavailable: "This content is currently unavailable",
		topicUnavailable:   "This topic does not exist or is not public",
		commentUnavailable: "This comment does not exist or is not public",
	}
)

// copyFor resolves one request locale with exact-locale, language-prefix, and
// safe final fallback semantics matching the Host manifest locale rules. The
// last-resort fallback never leaks a developer/internal string.
func copyFor(locale string) renderCopy {
	switch normalized := strings.ToLower(strings.ReplaceAll(strings.TrimSpace(locale), "_", "-")); normalized {
	case "zh-cn", "zh":
		return simplifiedChineseCopy
	case "en-us", "en":
		return englishCopy
	default:
		prefix := normalized
		if index := strings.IndexByte(prefix, '-'); index >= 0 {
			prefix = prefix[:index]
		}
		switch prefix {
		case "zh":
			return simplifiedChineseCopy
		case "en":
			return englishCopy
		default:
			// 站点默认 locale 为 zh-CN；未知请求语言也使用完整且安全的副本。
			return simplifiedChineseCopy
		}
	}
}
