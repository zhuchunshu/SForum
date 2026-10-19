package extensionmanifest

import (
	"net/mail"
	"strings"
)

// LocalizedDisplay 按 locale 解析可展示字段；无 langs 或未命中时回退顶层默认英文。
func LocalizedDisplay(manifest Manifest, locale string) ManifestLocale {
	manifest = Normalize(manifest)
	display := ManifestLocale{
		Name:        manifest.Name,
		Description: manifest.Description,
		URL:         manifest.URL,
		Author:      manifest.Author,
	}
	override, ok := lookupManifestLocale(manifest.Langs, locale)
	if !ok {
		return display
	}
	if override.Name != "" {
		display.Name = override.Name
	}
	if override.Description != "" {
		display.Description = override.Description
	}
	if override.URL != "" {
		display.URL = override.URL
	}
	if override.Author.Name != "" {
		display.Author.Name = override.Author.Name
	}
	if override.Author.URL != "" {
		display.Author.URL = override.Author.URL
	}
	if override.Author.Email != "" {
		display.Author.Email = override.Author.Email
	}
	return display
}

func normalizeManifestLangs(langs map[string]ManifestLocale) map[string]ManifestLocale {
	if len(langs) == 0 {
		return nil
	}
	normalized := make(map[string]ManifestLocale, len(langs))
	for key, locale := range langs {
		code := normalizeLocaleKey(key)
		if code == "" {
			continue
		}
		locale.Name = strings.TrimSpace(locale.Name)
		locale.Description = strings.TrimSpace(locale.Description)
		locale.URL = strings.TrimSpace(locale.URL)
		locale.Author.Name = strings.TrimSpace(locale.Author.Name)
		locale.Author.URL = strings.TrimSpace(locale.Author.URL)
		locale.Author.Email = strings.TrimSpace(locale.Author.Email)
		// 空覆盖无意义，直接丢弃。
		if locale.Name == "" && locale.Description == "" && locale.URL == "" && locale.Author.Name == "" && locale.Author.URL == "" && locale.Author.Email == "" {
			continue
		}
		normalized[code] = locale
	}
	if len(normalized) == 0 {
		return nil
	}
	return normalized
}

func validateManifestLangs(langs map[string]ManifestLocale) error {
	if len(langs) == 0 {
		return nil
	}
	for key, locale := range langs {
		if normalizeLocaleKey(key) == "" {
			return ErrInvalidManifest
		}
		name := strings.TrimSpace(locale.Name)
		description := strings.TrimSpace(locale.Description)
		localeURL := strings.TrimSpace(locale.URL)
		authorName := strings.TrimSpace(locale.Author.Name)
		authorURL := strings.TrimSpace(locale.Author.URL)
		authorEmail := strings.TrimSpace(locale.Author.Email)
		// 局部字段可选；整段覆盖不能全空。
		if name == "" && description == "" && localeURL == "" && authorName == "" && authorURL == "" && authorEmail == "" {
			return ErrInvalidManifest
		}
		if localeURL != "" && !validHTTPURL(localeURL) {
			return ErrInvalidManifest
		}
		if authorURL != "" && !validHTTPURL(authorURL) {
			return ErrInvalidManifest
		}
		if authorEmail != "" {
			if _, err := mail.ParseAddress(authorEmail); err != nil {
				return ErrInvalidManifest
			}
		}
	}
	return nil
}

func lookupManifestLocale(langs map[string]ManifestLocale, locale string) (ManifestLocale, bool) {
	if len(langs) == 0 {
		return ManifestLocale{}, false
	}
	for _, candidate := range localeLookupCandidates(locale) {
		if item, ok := langs[candidate]; ok {
			return item, true
		}
	}
	return ManifestLocale{}, false
}

func localeLookupCandidates(locale string) []string {
	code := normalizeLocaleKey(locale)
	if code == "" {
		return nil
	}
	candidates := []string{code}
	if primary, _, ok := strings.Cut(code, "-"); ok && primary != "" && primary != code {
		candidates = append(candidates, primary)
	}
	return candidates
}

func normalizeLocaleKey(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}
	value = strings.ReplaceAll(value, "_", "-")
	parts := strings.Split(value, "-")
	if len(parts) == 0 || parts[0] == "" {
		return ""
	}
	parts[0] = strings.ToLower(parts[0])
	for index := 1; index < len(parts); index++ {
		// 地区码常见为大写（CN/US），语言码小写。
		if len(parts[index]) == 2 {
			parts[index] = strings.ToUpper(parts[index])
		} else {
			parts[index] = strings.ToLower(parts[index])
		}
	}
	return strings.Join(parts, "-")
}
