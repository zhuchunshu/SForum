package localization

import "strings"

// PluralForm returns a CLDR-like plural category for the locale and count.
// Supported categories: zero, one, other. Chinese treats all non-zero as other.
func PluralForm(locale string, count int) string {
	locale = Normalize(locale, nil)
	if count < 0 {
		count = -count
	}
	switch {
	case strings.HasPrefix(locale, "zh"):
		if count == 0 {
			return "zero"
		}
		return "other"
	default:
		// English and most Indo-European locales: one vs other.
		if count == 1 {
			return "one"
		}
		if count == 0 {
			return "zero"
		}
		return "other"
	}
}
