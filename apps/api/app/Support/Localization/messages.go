package localization

import (
	"sort"
	"strconv"
	"strings"
)

var messages = map[string]map[string]string{
	"zh-CN": {
		"ok":                              "OK",
		"auth.required":                   "请先登录。",
		"auth.invalid_credentials":        "登录失败：请检查用户名/邮箱和密码；如果还没有账号，请先注册。",
		"auth.register_invalid":           "注册失败：请按标出的提示修改后再提交。",
		"auth.session_unavailable":        "账号已创建，但自动登录失败，请直接登录。",
		"auth.username_required":          "请填写用户名。",
		"auth.email_required":             "请填写邮箱地址。",
		"auth.email_invalid":              "邮箱格式不正确，请填写可接收邮件的地址。",
		"auth.password_min_length":        "密码至少需要 12 个字符。",
		"auth.username_taken":             "这个用户名已被使用，请换一个。",
		"auth.email_taken":                "这个邮箱已经注册过，请直接登录或换一个邮箱。",
		"permission.denied":               "没有权限执行此操作。",
		"options.invalid":                 "站点设置不正确，请检查后重试。",
		"validation.invalid":              "请求参数不正确。",
		"human_verification.required":     "请先完成人机验证。",
		"human_verification.invalid":      "人机验证失败，请重新验证。",
		"human_verification.expired":      "人机验证已过期，请重新验证。",
		"human_verification.replayed":     "本次人机验证已使用，请重新验证。",
		"rate_limit.exceeded":             "操作过于频繁，请稍后再试。",
		"role.system_role_locked":         "系统角色不能执行此操作。",
		"role.default_role_locked":        "默认角色不能执行此操作。",
		"user.initial_super_admin_locked": "初始超级管理员不能执行此操作。",
		"auth.password_policy":            "密码至少需要 12 个字符。",
		"not_found":                       "请求的资源不存在。",
		"method_not_allowed":              "不支持当前请求方法。",
		"internal_error":                  "服务器暂时不可用，请稍后再试。",
	},
	"en-US": {
		"ok":                              "OK",
		"auth.required":                   "Please sign in first.",
		"auth.invalid_credentials":        "Login failed: check your username/email and password, or register first if you do not have an account.",
		"auth.register_invalid":           "Registration failed: fix the highlighted fields and submit again.",
		"auth.session_unavailable":        "Your account was created, but automatic sign-in failed. Please sign in directly.",
		"auth.username_required":          "Enter a username.",
		"auth.email_required":             "Enter an email address.",
		"auth.email_invalid":              "Enter a valid email address that can receive mail.",
		"auth.password_min_length":        "Use at least 12 characters for your password.",
		"auth.username_taken":             "This username is already taken. Try another one.",
		"auth.email_taken":                "This email is already registered. Sign in or use another email.",
		"permission.denied":               "You do not have permission to perform this action.",
		"options.invalid":                 "The site setting is invalid. Check it and try again.",
		"validation.invalid":              "The request parameters are invalid.",
		"human_verification.required":     "Please complete human verification first.",
		"human_verification.invalid":      "Human verification failed. Please try again.",
		"human_verification.expired":      "Human verification expired. Please verify again.",
		"human_verification.replayed":     "This human verification has already been used. Please verify again.",
		"rate_limit.exceeded":             "Too many attempts. Please try again later.",
		"role.system_role_locked":         "System roles cannot perform this action.",
		"role.default_role_locked":        "The default role cannot perform this action.",
		"user.initial_super_admin_locked": "The initial super administrator cannot perform this action.",
		"auth.password_policy":            "Use at least 12 characters for your password.",
		"not_found":                       "The requested resource does not exist.",
		"method_not_allowed":              "This request method is not supported.",
		"internal_error":                  "The server is temporarily unavailable. Please try again later.",
	},
}

func Message(locale string, key string) string {
	normalized := Normalize(locale, []string{"zh-CN", "en-US"})
	if catalog, ok := messages[normalized]; ok {
		if message, ok := catalog[key]; ok {
			return message
		}
	}

	if catalog, ok := messages[DefaultLocale]; ok {
		if message, ok := catalog[key]; ok {
			return message
		}
	}

	return key
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
