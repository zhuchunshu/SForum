package mail

import (
	"context"
	"fmt"
	"net/url"
	"strconv"
	"strings"
)

// Brand is immutable public visual identity serialized with a mail delivery.
type Brand struct {
	SiteName, SiteURL                         string
	LogoURL, IconURL, Mark                    string
	AccentColor, AccentSoft, AccentSoftBorder string
}

type BrandResolver interface {
	MailBrand(context.Context) (Brand, error)
}

func DefaultBrand(siteName, siteURL string) Brand {
	return BrandFromSettings(siteName, siteURL, "", "", "pine_teal")
}

func BrandFromSettings(siteName, siteURL, logoURL, iconURL, appearanceTheme string) Brand {
	name := strings.TrimSpace(siteName)
	if name == "" {
		name = "SForum"
	}
	accent, soft, border := themePalette(appearanceTheme)
	return Brand{SiteName: name, SiteURL: strings.TrimSpace(siteURL), LogoURL: resolveAssetURL(siteURL, logoURL), IconURL: resolveAssetURL(siteURL, iconURL), Mark: siteMark(name), AccentColor: accent, AccentSoft: soft, AccentSoftBorder: border}
}

func (b Brand) TemplateData() map[string]string {
	brand := NormalizeBrand(b)
	return map[string]string{"siteName": brand.SiteName, "siteUrl": brand.SiteURL, "brandLogoUrl": brand.LogoURL, "brandIconUrl": brand.IconURL, "brandMark": brand.Mark, "brandAccent": brand.AccentColor, "brandAccentSoft": brand.AccentSoft, "brandAccentSoftBorder": brand.AccentSoftBorder}
}

func ResolveBrand(ctx context.Context, resolver BrandResolver) Brand {
	if resolver != nil {
		if brand, err := resolver.MailBrand(ctx); err == nil {
			return NormalizeBrand(brand)
		}
	}
	return DefaultBrand("SForum", "")
}

func NormalizeBrand(brand Brand) Brand {
	fallback := DefaultBrand(brand.SiteName, brand.SiteURL)
	if strings.TrimSpace(brand.LogoURL) != "" {
		fallback.LogoURL = strings.TrimSpace(brand.LogoURL)
	}
	if strings.TrimSpace(brand.IconURL) != "" {
		fallback.IconURL = strings.TrimSpace(brand.IconURL)
	}
	if strings.TrimSpace(brand.Mark) != "" {
		fallback.Mark = strings.TrimSpace(brand.Mark)
	}
	if isHexColor(brand.AccentColor) {
		fallback.AccentColor = strings.ToLower(brand.AccentColor)
	}
	if isHexColor(brand.AccentSoft) {
		fallback.AccentSoft = strings.ToLower(brand.AccentSoft)
	}
	if isHexColor(brand.AccentSoftBorder) {
		fallback.AccentSoftBorder = strings.ToLower(brand.AccentSoftBorder)
	}
	return fallback
}

func themePalette(theme string) (accent, soft, border string) {
	switch strings.ToLower(strings.TrimSpace(theme)) {
	case "ocean_blue":
		accent, soft = "#2563eb", "#eff6ff"
	case "violet":
		accent, soft = "#7c3aed", "#f3e8ff"
	case "rose":
		accent, soft = "#e11d48", "#fff1f2"
	case "amber":
		accent, soft = "#d97706", "#fffbeb"
	case "pine_teal":
		accent, soft = "#0f766e", "#e6f4f1"
	default:
		accent = strings.TrimPrefix(strings.ToLower(strings.TrimSpace(theme)), "custom:")
		if !isHexColor(accent) {
			accent, soft = "#0f766e", "#e6f4f1"
		}
	}
	if soft == "" {
		soft = mixHex(accent, "#ffffff", 0.92)
	}
	return accent, soft, mixHex(accent, "#ffffff", 0.68)
}

func resolveAssetURL(siteURL, assetURL string) string {
	assetURL = strings.TrimSpace(assetURL)
	if assetURL == "" {
		return ""
	}
	parsed, err := url.Parse(assetURL)
	if err == nil && (parsed.Scheme == "http" || parsed.Scheme == "https") && parsed.Host != "" {
		return parsed.String()
	}
	base, baseErr := url.Parse(strings.TrimSpace(siteURL))
	if err != nil || baseErr != nil || base.Scheme == "" || base.Host == "" || !strings.HasPrefix(assetURL, "/") {
		return ""
	}
	return base.ResolveReference(parsed).String()
}

func siteMark(siteName string) string {
	for _, runeValue := range strings.TrimSpace(siteName) {
		return string(runeValue)
	}
	return "S"
}

func isHexColor(value string) bool {
	value = strings.TrimSpace(value)
	if len(value) != 7 || value[0] != '#' {
		return false
	}
	_, err := strconv.ParseUint(value[1:], 16, 24)
	return err == nil
}

func mixHex(from, to string, amount float64) string {
	fromValue, _ := strconv.ParseUint(strings.TrimPrefix(from, "#"), 16, 24)
	toValue, _ := strconv.ParseUint(strings.TrimPrefix(to, "#"), 16, 24)
	mix := func(shift uint) int {
		start := float64((fromValue >> shift) & 0xff)
		end := float64((toValue >> shift) & 0xff)
		return int(start + (end-start)*amount + 0.5)
	}
	return fmt.Sprintf("#%02x%02x%02x", mix(16), mix(8), mix(0))
}
