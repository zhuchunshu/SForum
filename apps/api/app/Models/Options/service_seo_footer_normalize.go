package options

import (
	"encoding/json"
	"net/url"
	"strconv"
	"strings"
)

func isValidRobotsPath(value string) bool {
	if !strings.HasPrefix(value, "/") || strings.HasPrefix(value, "//") {
		return false
	}
	for _, char := range value {
		if char <= ' ' || char == '<' || char == '>' || char == '"' || char == '\'' {
			return false
		}
	}
	return true
}

func seoEnabledOptionNames() []string {
	return []string{
		NameSEOAllowIndexing,
		NameSEORobotsBlockAIBots,
		NameSEORobotsBlockNonSEOBots,
		NameSEOSitemapEnabled,
		NameSEOSitemapIncludeStaticPages,
		NameSEOSitemapIncludeForumContent,
		NameSEOSchemaOrgEnabled,
		NameSEOSchemaOrgSearchAction,
		NameSEOSchemaOrgDiscussion,
	}
}

func passwordPolicyBooleanOptionNames() []string {
	return []string{
		NameIdentityPasswordRequireLowercase,
		NameIdentityPasswordRequireUppercase,
		NameIdentityPasswordRequireNumber,
		NameIdentityPasswordRequireSymbol,
	}
}

func seoVerificationOptionNames() []string {
	return []string{
		NameSEOGoogleVerification,
		NameSEOBingVerification,
		NameSEOBaiduVerification,
		NameSEOYandexVerification,
	}
}

func defaultFooterLinksValue() string {
	value, _ := marshalFooterLinks(defaultFooterLinks())
	return value
}

func defaultFooterLinks() []footerLinkOption {
	return []footerLinkOption{
		{
			Key:    "terms",
			Labels: footerLinkLabels{ZHCN: "服务条款", ENUS: "Terms of Service"},
			URL:    "#",
		},
		{
			Key:    "privacy",
			Labels: footerLinkLabels{ZHCN: "隐私政策", ENUS: "Privacy Policy"},
			URL:    "#",
		},
		{
			Key:    "guidelines",
			Labels: footerLinkLabels{ZHCN: "社区指南", ENUS: "Guidelines"},
			URL:    "#",
		},
	}
}

func normalizeFooterLinks(value string) (string, bool) {
	var links []footerLinkOption
	if err := json.Unmarshal([]byte(strings.TrimSpace(value)), &links); err != nil {
		return "", false
	}
	if len(links) != len(footerLinkKeys) {
		return "", false
	}

	byKey := map[string]footerLinkOption{}
	for _, link := range links {
		key := strings.TrimSpace(link.Key)
		if !isFooterLinkKey(key) || byKey[key].Key != "" {
			return "", false
		}

		normalized := footerLinkOption{
			Key: key,
			Labels: footerLinkLabels{
				ZHCN: strings.TrimSpace(link.Labels.ZHCN),
				ENUS: strings.TrimSpace(link.Labels.ENUS),
			},
			URL: strings.TrimSpace(link.URL),
		}
		if !isValidFooterLinkLabel(normalized.Labels.ZHCN) || !isValidFooterLinkLabel(normalized.Labels.ENUS) {
			return "", false
		}
		if !isValidFooterURL(normalized.URL) {
			return "", false
		}
		byKey[key] = normalized
	}

	ordered := make([]footerLinkOption, 0, len(footerLinkKeys))
	for _, key := range footerLinkKeys {
		link, ok := byKey[key]
		if !ok {
			return "", false
		}
		ordered = append(ordered, link)
	}
	return marshalFooterLinks(ordered)
}

func marshalFooterLinks(links []footerLinkOption) (string, bool) {
	value, err := json.Marshal(links)
	if err != nil {
		return "", false
	}
	return string(value), true
}

func isFooterLinkKey(value string) bool {
	for _, key := range footerLinkKeys {
		if value == key {
			return true
		}
	}
	return false
}

func isValidFooterLinkLabel(value string) bool {
	return value != "" && len([]rune(value)) <= footerLinkLabelMaxRunes
}

func isValidFooterURL(value string) bool {
	if value == "" || value == "#" {
		return true
	}
	if strings.HasPrefix(value, "/") && !strings.HasPrefix(value, "//") {
		return true
	}

	parsed, err := url.Parse(value)
	if err != nil || parsed == nil || parsed.Host == "" {
		return false
	}
	return parsed.Scheme == "http" || parsed.Scheme == "https"
}

// strictAtoi 解析整数字符串；非数字返回 false。
func strictAtoi(value string) (int, bool) {
	parsed, err := strconv.Atoi(strings.TrimSpace(value))
	if err != nil {
		return 0, false
	}
	return parsed, true
}

// coerceForumContentLimitOptions 将发帖/评论限制回退到合法推荐默认。
