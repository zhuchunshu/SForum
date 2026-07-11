package localization

import (
	"sort"
	"strconv"
	"strings"
)

var messages = map[string]map[string]string{
	"zh-CN": {
		"ok":                                  "OK",
		"auth.required":                       "请先登录。",
		"auth.invalid_credentials":            "登录失败：请检查用户名/邮箱和密码；如果还没有账号，请先注册。",
		"auth.register_invalid":               "注册失败：请按标出的提示修改后再提交。",
		"auth.register_disabled":              "当前站点已关闭开放注册，请联系管理员获取账号。",
		"auth.session_unavailable":            "账号已创建，但自动登录失败，请直接登录。",
		"auth.username_required":              "请填写用户名。",
		"auth.email_required":                 "请填写邮箱地址。",
		"auth.email_invalid":                  "邮箱格式不正确，请填写可接收邮件的地址。",
		"auth.password_min_length":            "密码长度低于当前站点要求。",
		"auth.password_max_length":            "密码长度超过当前站点要求。",
		"auth.password_lowercase":             "密码需要包含小写字母。",
		"auth.password_uppercase":             "密码需要包含大写字母。",
		"auth.password_number":                "密码需要包含数字。",
		"auth.password_symbol":                "密码需要包含符号。",
		"auth.username_taken":                 "这个用户名已被使用，请换一个。",
		"auth.email_taken":                    "这个邮箱已经注册过，请直接登录或换一个邮箱。",
		"permission.denied":                   "没有权限执行此操作。",
		"permission.invalid":                  "权限标识不存在，请检查后重试。",
		"permission.override_conflict":        "同一权限不能同时设置为允许和拒绝。",
		"role.invalid":                        "用户组不存在，请检查后重试。",
		"role.invalid_input":                  "请填写用户组标识和别名；标识只能使用小写字母、数字、点号、短横线或下划线。",
		"options.invalid":                     "站点设置不正确，请检查后重试。",
		"attachment.invalid":                  "附件不正确，请检查文件类型、大小或配置后重试。",
		"attachment.upload_disabled":          "当前站点未启用附件上传。",
		"attachment.referenced":               "附件仍被内容引用，不能物理删除。",
		"attachment.storage_unavailable":      "附件存储暂时不可用，请检查存储配置。",
		"forum.content_invalid":               "内容不正确，请检查正文和编辑器格式后重试。",
		"forum.topic_invalid":                 "帖子不正确，请检查标题、版块和正文后重试。",
		"forum.topic_not_found":               "帖子不存在或暂不可见。",
		"forum.comment_not_found":             "评论不存在或暂不可见。",
		"forum.topic_closed":                  "该帖子已关闭，暂不能继续评论。",
		"forum.tag_invalid":                   "标签不正确，请选择已允许的标签或检查标签设置。",
		"forum.tag_not_found":                 "标签不存在或暂不可见。",
		"forum.settings_invalid":              "论坛设置不正确，请检查默认版块和标签策略。",
		"forum.title_too_short":               "标题字数不足，请补充后再发布。",
		"forum.title_too_long":                "标题过长，请缩短后再发布。",
		"forum.content_too_short":             "正文内容过短，请补充后再发布。",
		"forum.content_too_long":              "正文内容过长，请精简后再发布。",
		"forum.comment_too_short":             "评论内容过短，请补充后再发送。",
		"forum.comment_too_long":              "评论内容过长，请精简后再发送。",
		"forum.comment_nesting_too_deep":      "回复层级过深，请回复更高层的评论。",
		"forum.edit_window_expired":           "已超过可编辑时间窗口，无法继续修改。",
		"forum.topic_cooldown":                "发帖过于频繁，请稍后再试。",
		"forum.comment_cooldown":              "评论过于频繁，请稍后再试。",
		"forum.daily_topic_limit":             "今日发帖数量已达上限。",
		"forum.daily_comment_limit":           "今日评论数量已达上限。",
		"forum.tag_min_required":              "请至少选择要求数量的标签。",
		"extension.archive_invalid":           "扩展包无效，请上传符合 SForum manifest 规范的 ZIP 文件。",
		"extension.manifest_invalid":          "扩展 manifest 无效，请检查扩展标识、类型、版本和路径。",
		"extension.not_found":                 "扩展不存在，请刷新后重试。",
		"extension.disabled":                  "扩展已禁用，请先启用后再使用其设置与功能。",
		"extension.preflight_failed":          "插件预检失败，请检查后端入口文件。",
		"extension.build_failed":              "主题验证失败，请检查 Nuxt Layer 路径。",
		"extension.theme_activation_required": "主题不能使用插件启用/禁用流程，请使用主题激活操作。",
		"extension.theme_activation_queued":   "主题构建已排队，完成后会自动切换。",
		"extension.theme_runtime_unavailable": "当前版本暂不支持应用上传主题。",
		"validation.invalid":                  "请求参数不正确。",
		"human_verification.required":         "请先完成人机验证。",
		"human_verification.invalid":          "人机验证失败，请重新验证。",
		"human_verification.expired":          "人机验证已过期，请重新验证。",
		"human_verification.replayed":         "本次人机验证已使用，请重新验证。",
		"rate_limit.exceeded":                 "操作过于频繁，请稍后再试。",
		"role.system_role_locked":             "系统角色不能执行此操作。",
		"role.default_role_locked":            "默认角色不能执行此操作。",
		"user.initial_super_admin_locked":     "初始超级管理员不能执行此操作。",
		"user.super_admin_permissions_locked": "超级管理员的权限例外不能直接编辑。",
		"auth.password_policy":                "密码不符合当前站点密码策略。",
		"not_found":                           "请求的资源不存在。",
		"method_not_allowed":                  "不支持当前请求方法。",
		"internal_error":                      "服务器暂时不可用，请稍后再试。",
	},
	"en-US": {
		"ok":                                  "OK",
		"auth.required":                       "Please sign in first.",
		"auth.invalid_credentials":            "Login failed: check your username/email and password, or register first if you do not have an account.",
		"auth.register_invalid":               "Registration failed: fix the highlighted fields and submit again.",
		"auth.register_disabled":              "Open registration is disabled on this site. Contact an administrator for an account.",
		"auth.session_unavailable":            "Your account was created, but automatic sign-in failed. Please sign in directly.",
		"auth.username_required":              "Enter a username.",
		"auth.email_required":                 "Enter an email address.",
		"auth.email_invalid":                  "Enter a valid email address that can receive mail.",
		"auth.password_min_length":            "The password is shorter than the current site policy allows.",
		"auth.password_max_length":            "The password is longer than the current site policy allows.",
		"auth.password_lowercase":             "Include a lowercase letter in your password.",
		"auth.password_uppercase":             "Include an uppercase letter in your password.",
		"auth.password_number":                "Include a number in your password.",
		"auth.password_symbol":                "Include a symbol in your password.",
		"auth.username_taken":                 "This username is already taken. Try another one.",
		"auth.email_taken":                    "This email is already registered. Sign in or use another email.",
		"permission.denied":                   "You do not have permission to perform this action.",
		"permission.invalid":                  "The permission key does not exist. Check it and try again.",
		"permission.override_conflict":        "The same permission cannot be both allowed and denied.",
		"role.invalid":                        "The user group does not exist. Check it and try again.",
		"role.invalid_input":                  "Enter a user group key and alias. The key may contain lowercase letters, numbers, dots, hyphens, or underscores.",
		"options.invalid":                     "The site setting is invalid. Check it and try again.",
		"attachment.invalid":                  "The attachment is invalid. Check its type, size, or storage settings and try again.",
		"attachment.upload_disabled":          "Attachment uploads are disabled for this site.",
		"attachment.referenced":               "The attachment is still referenced by content and cannot be physically deleted.",
		"attachment.storage_unavailable":      "Attachment storage is temporarily unavailable. Check the storage settings.",
		"forum.content_invalid":               "The content is invalid. Check the body and editor format, then try again.",
		"forum.topic_invalid":                 "The topic is invalid. Check the title, category, and body, then try again.",
		"forum.topic_not_found":               "The topic does not exist or is not currently visible.",
		"forum.comment_not_found":             "The comment does not exist or is not currently visible.",
		"forum.topic_closed":                  "This topic is closed and cannot receive new comments.",
		"forum.tag_invalid":                   "The tag is invalid. Choose an allowed tag or check the tag settings.",
		"forum.tag_not_found":                 "The tag does not exist or is not currently visible.",
		"forum.settings_invalid":              "The forum settings are invalid. Check the default category and tag policy.",
		"forum.title_too_short":               "The title is too short. Add more characters before publishing.",
		"forum.title_too_long":                "The title is too long. Shorten it before publishing.",
		"forum.content_too_short":             "The body is too short. Add more content before publishing.",
		"forum.content_too_long":              "The body is too long. Shorten it before publishing.",
		"forum.comment_too_short":             "The comment is too short. Add more content before posting.",
		"forum.comment_too_long":              "The comment is too long. Shorten it before posting.",
		"forum.comment_nesting_too_deep":      "This reply is nested too deeply. Reply to a higher-level comment.",
		"forum.edit_window_expired":           "The edit window has expired. This content can no longer be changed.",
		"forum.topic_cooldown":                "You are creating topics too quickly. Please wait and try again.",
		"forum.comment_cooldown":              "You are posting comments too quickly. Please wait and try again.",
		"forum.daily_topic_limit":             "You have reached today's topic creation limit.",
		"forum.daily_comment_limit":           "You have reached today's comment limit.",
		"forum.tag_min_required":              "Select at least the required number of tags.",
		"extension.archive_invalid":           "The extension package is invalid. Upload a ZIP file with a valid SForum manifest.",
		"extension.manifest_invalid":          "The extension manifest is invalid. Check the extension id, type, version, and paths.",
		"extension.not_found":                 "The extension does not exist. Refresh and try again.",
		"extension.disabled":                  "The extension is disabled. Enable it before using its settings or features.",
		"extension.preflight_failed":          "Plugin preflight failed. Check the backend entry file.",
		"extension.build_failed":              "Theme verification failed. Check the Nuxt Layer path.",
		"extension.theme_activation_required": "Themes cannot use the plugin enable/disable flow. Use theme activation instead.",
		"extension.theme_activation_queued":   "Theme build has been queued and will switch automatically when it completes.",
		"extension.theme_runtime_unavailable": "Applying uploaded themes is not supported in this release.",
		"validation.invalid":                  "The request parameters are invalid.",
		"human_verification.required":         "Please complete human verification first.",
		"human_verification.invalid":          "Human verification failed. Please try again.",
		"human_verification.expired":          "Human verification expired. Please verify again.",
		"human_verification.replayed":         "This human verification has already been used. Please verify again.",
		"rate_limit.exceeded":                 "Too many attempts. Please try again later.",
		"role.system_role_locked":             "System roles cannot perform this action.",
		"role.default_role_locked":            "The default role cannot perform this action.",
		"user.initial_super_admin_locked":     "The initial super administrator cannot perform this action.",
		"user.super_admin_permissions_locked": "Permission overrides cannot be edited directly for super administrators.",
		"auth.password_policy":                "The password does not meet the current site policy.",
		"not_found":                           "The requested resource does not exist.",
		"method_not_allowed":                  "This request method is not supported.",
		"internal_error":                      "The server is temporarily unavailable. Please try again later.",
	},
}

var trustedRuntimeMessages = map[string]map[string]string{
	"zh-CN": {
		"extension.frontend_runtime_unavailable": "可信前端运行时暂不可用，请稍后重试。",
		"extension.frontend_digest_invalid":      "扩展前端摘要无效，请刷新扩展信息后重试。",
		"extension.frontend_trust_not_found":     "该扩展没有可撤销的可信前端授权。",
		"extension.web_release_not_found":        "Web Release 不存在，请刷新后重试。",
		"extension.web_release_conflict":         "当前 Web Release 状态已变化，或该操作不再安全，请刷新后重试。",
	},
	"en-US": {
		"extension.frontend_runtime_unavailable": "The trusted frontend runtime is temporarily unavailable. Try again later.",
		"extension.frontend_digest_invalid":      "The extension frontend digest is invalid. Refresh the extension details and try again.",
		"extension.frontend_trust_not_found":     "This extension has no trusted frontend grant to revoke.",
		"extension.web_release_not_found":        "The web release does not exist. Refresh and try again.",
		"extension.web_release_conflict":         "The web release state changed or this operation is no longer safe. Refresh and try again.",
	},
}

func Message(locale string, key string) string {
	normalized := Normalize(locale, []string{"zh-CN", "en-US"})
	if message, ok := lookupMessage(trustedRuntimeMessages, normalized, key); ok {
		return message
	}
	if catalog, ok := messages[normalized]; ok {
		if message, ok := catalog[key]; ok {
			return message
		}
	}
	if message, ok := lookupMessage(trustedRuntimeMessages, DefaultLocale, key); ok {
		return message
	}

	if catalog, ok := messages[DefaultLocale]; ok {
		if message, ok := catalog[key]; ok {
			return message
		}
	}

	return key
}

func lookupMessage(catalogs map[string]map[string]string, locale string, key string) (string, bool) {
	catalog, ok := catalogs[locale]
	if !ok {
		return "", false
	}
	message, ok := catalog[key]
	return message, ok
}

func NegotiateAcceptLanguage(header string, supported []string, fallback string) string {
	fallback = Normalize(fallback, supported)
	ranges := parseAcceptLanguage(header)
	if len(ranges) == 0 {
		return fallback
	}

	for _, item := range ranges {
		if locale, ok := matchSupportedLocale(item.tag, supported); ok {
			return locale
		}
	}

	return fallback
}

func matchSupportedLocale(locale string, supported []string) (string, bool) {
	candidate := strings.TrimSpace(locale)
	if candidate == "" {
		return "", false
	}

	if alias, ok := aliases[strings.ToLower(candidate)]; ok {
		candidate = alias
	}

	for _, item := range supported {
		if strings.EqualFold(candidate, item) {
			return item, true
		}
	}

	return "", false
}

type languageRange struct {
	tag   string
	q     float64
	order int
}

func parseAcceptLanguage(header string) []languageRange {
	parts := strings.Split(header, ",")
	ranges := make([]languageRange, 0, len(parts))

	for index, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}

		tag := part
		q := 1.0
		if semi := strings.Index(part, ";"); semi >= 0 {
			tag = strings.TrimSpace(part[:semi])
			for _, param := range strings.Split(part[semi+1:], ";") {
				param = strings.TrimSpace(param)
				if strings.HasPrefix(param, "q=") {
					parsed, err := strconv.ParseFloat(strings.TrimPrefix(param, "q="), 64)
					if err == nil {
						q = parsed
					}
				}
			}
		}

		if tag != "" && tag != "*" && q > 0 {
			ranges = append(ranges, languageRange{tag: tag, q: q, order: index})
		}
	}

	// Accept-Language 需要按 q 值优先，q 相同时保留浏览器发送顺序。
	sort.SliceStable(ranges, func(i, j int) bool {
		if ranges[i].q == ranges[j].q {
			return ranges[i].order < ranges[j].order
		}
		return ranges[i].q > ranges[j].q
	})

	return ranges
}
