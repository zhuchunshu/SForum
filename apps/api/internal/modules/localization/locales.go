package localization

import "strings"

const DefaultLocale = "zh-CN"

var aliases = map[string]string{
	"zh":    "zh-CN",
	"zh-cn": "zh-CN",
	"cn":    "zh-CN",
	"en":    "en-US",
	"en-us": "en-US",
}

func ParseSupportedLocales(raw string) []string {
	if strings.TrimSpace(raw) == "" {
		return []string{DefaultLocale, "en-US"}
	}

	parts := strings.Split(raw, ",")
	locales := make([]string, 0, len(parts))
	seen := map[string]bool{}

	for _, part := range parts {
		locale := Normalize(part, nil)
		if locale == "" || seen[locale] {
			continue
		}
		seen[locale] = true
		locales = append(locales, locale)
	}

	if len(locales) == 0 {
		return []string{DefaultLocale, "en-US"}
	}

	return locales
}

func Normalize(locale string, supported []string) string {
	candidate := strings.TrimSpace(locale)
	if candidate == "" {
		return DefaultLocale
	}

	if alias, ok := aliases[strings.ToLower(candidate)]; ok {
		candidate = alias
	}

	if len(supported) == 0 {
		return candidate
	}

	for _, item := range supported {
		if strings.EqualFold(candidate, item) {
			return item
		}
	}

	return DefaultLocale
}
