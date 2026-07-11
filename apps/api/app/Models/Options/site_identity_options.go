package options

import (
	"net/mail"
	"strings"
	"unicode/utf8"

	identity "github.com/zhuchunshu/sforum/apps/api/app/Models/Identity"
)

// 副标题与管理邮箱的长度边界。
const (
	siteTaglineMaxRunes    = 160
	siteAdminEmailMaxRunes = 254
)

func init() {
	optionDefinitions = append(optionDefinitions, siteIdentityOptionDefinitions()...)
}

func siteIdentityOptionDefinitions() []optionDefinition {
	return []optionDefinition{
		// 标语对前台可见，用于导航/登录页。
		{name: NameSiteTagline, public: true, managePermission: identity.PermissionSettingsSiteManage},
		// 管理邮箱仅后台可读，降低公开爬取风险。
		{name: NameSiteAdminEmail, public: false, managePermission: identity.PermissionSettingsSiteManage},
	}
}

func siteIdentityRecommendedDefaults() map[string]string {
	return map[string]string{
		NameSiteTagline:    "",
		NameSiteAdminEmail: "",
	}
}

func normalizeSiteTagline(value string) (string, bool) {
	// 允许空；去掉首尾空白；限制 rune 长度。
	value = strings.TrimSpace(value)
	if utf8.RuneCountInString(value) > siteTaglineMaxRunes {
		return "", false
	}
	return value, true
}

func normalizeSiteAdminEmail(value string) (string, bool) {
	value = strings.TrimSpace(value)
	if value == "" {
		return "", true
	}
	if utf8.RuneCountInString(value) > siteAdminEmailMaxRunes {
		return "", false
	}
	// 与 identity 注册校验一致：net/mail 解析且整串即地址。
	parsed, err := mail.ParseAddress(value)
	if err != nil || parsed.Address != value {
		return "", false
	}
	return value, true
}

func coerceSiteIdentityOptions(coerced map[string]string, defaults map[string]string) {
	if value, ok := normalizeSiteTagline(coerced[NameSiteTagline]); ok {
		coerced[NameSiteTagline] = value
	} else {
		coerced[NameSiteTagline] = defaults[NameSiteTagline]
	}
	if value, ok := normalizeSiteAdminEmail(coerced[NameSiteAdminEmail]); ok {
		coerced[NameSiteAdminEmail] = value
	} else {
		coerced[NameSiteAdminEmail] = defaults[NameSiteAdminEmail]
	}
}

func isValidSiteIdentityOptions(values map[string]string) bool {
	if _, ok := normalizeSiteTagline(values[NameSiteTagline]); !ok {
		return false
	}
	if _, ok := normalizeSiteAdminEmail(values[NameSiteAdminEmail]); !ok {
		return false
	}
	return true
}
