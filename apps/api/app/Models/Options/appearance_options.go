package options

import "strings"

const customAppearanceThemePrefix = "custom:"

var appearanceThemes = []string{"pine_teal", "ocean_blue", "violet", "rose", "amber"}
var appearanceLightBackgrounds = []string{
	"pure_white",
	"porcelain",
	"paper",
	"parchment",
	"mist_gray",
	"cool_frost",
	"cloud_blue",
	"mint_mist",
	"sage",
	"sakura",
	"lilac_mist",
	"morning_apricot",
}

func normalizeAppearanceTheme(value string) (string, bool) {
	value = strings.ToLower(strings.TrimSpace(value))
	for _, theme := range appearanceThemes {
		if value == theme {
			return theme, true
		}
	}

	if strings.HasPrefix(value, customAppearanceThemePrefix) {
		color, ok := normalizeAppearanceThemeColor(strings.TrimPrefix(value, customAppearanceThemePrefix))
		if ok {
			return customAppearanceThemePrefix + color, true
		}
	}
	return "", false
}

func normalizeAppearanceThemeColor(value string) (string, bool) {
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

func normalizeAppearanceLightBackground(value string) (string, bool) {
	return normalizeChoice(value, appearanceLightBackgrounds)
}

func coerceAppearanceOptions(values map[string]string, defaults map[string]string) {
	if value, ok := normalizeAppearanceTheme(values[NameAppearanceTheme]); ok {
		values[NameAppearanceTheme] = value
	} else {
		values[NameAppearanceTheme] = defaults[NameAppearanceTheme]
	}
	if value, ok := normalizeAppearanceLightBackground(values[NameAppearanceLightBackground]); ok {
		values[NameAppearanceLightBackground] = value
	} else {
		values[NameAppearanceLightBackground] = defaults[NameAppearanceLightBackground]
	}
}

func validateAppearanceOptions(values map[string]string) bool {
	if _, ok := normalizeAppearanceTheme(values[NameAppearanceTheme]); !ok {
		return false
	}
	_, ok := normalizeAppearanceLightBackground(values[NameAppearanceLightBackground])
	return ok
}
