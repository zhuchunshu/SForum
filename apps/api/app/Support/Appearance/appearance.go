// Package appearance owns the stable Core appearance value contract shared by
// operator options and private user preferences.
package appearance

import "strings"

const customThemePrefix = "custom:"

var themes = map[string]struct{}{
	"pine_teal":  {},
	"ocean_blue": {},
	"violet":     {},
	"rose":       {},
	"amber":      {},
}

var lightBackgrounds = map[string]struct{}{
	"pure_white":      {},
	"porcelain":       {},
	"paper":           {},
	"parchment":       {},
	"mist_gray":       {},
	"cool_frost":      {},
	"cloud_blue":      {},
	"mint_mist":       {},
	"sage":            {},
	"sakura":          {},
	"lilac_mist":      {},
	"morning_apricot": {},
}

func NormalizeTheme(value string) (string, bool) {
	value = strings.ToLower(strings.TrimSpace(value))
	if _, ok := themes[value]; ok {
		return value, true
	}
	if strings.HasPrefix(value, customThemePrefix) {
		color, ok := normalizeHexColor(strings.TrimPrefix(value, customThemePrefix))
		if ok {
			return customThemePrefix + color, true
		}
	}
	return "", false
}

func NormalizeLightBackground(value string) (string, bool) {
	value = strings.ToLower(strings.TrimSpace(value))
	_, ok := lightBackgrounds[value]
	return value, ok
}

func normalizeHexColor(value string) (string, bool) {
	value = strings.TrimPrefix(strings.TrimSpace(strings.ToLower(value)), "#")
	if len(value) != 6 {
		return "", false
	}
	for _, char := range value {
		if (char < '0' || char > '9') && (char < 'a' || char > 'f') {
			return "", false
		}
	}
	return "#" + value, true
}
