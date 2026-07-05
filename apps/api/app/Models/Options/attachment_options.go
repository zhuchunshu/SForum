package options

import (
	"strings"

	storage "github.com/zhuchunshu/sforum/apps/api/app/Support/Storage"
)

const attachmentPathTemplateMaxRunes = 160
const attachmentListOptionMaxRunes = 1000
const attachmentProviderTextMaxRunes = 240
const attachmentSecretMaxRunes = 8000

var attachmentProviders = []string{storage.ProviderLocal, storage.ProviderAliyunOSS, storage.ProviderTencentCOS, storage.ProviderFTP, storage.ProviderSFTP}
var attachmentVisibilities = []string{"public", "private"}

func attachmentOptionNames() []string {
	return []string{
		NameAttachmentProvider,
		NameAttachmentUploadEnabled,
		NameAttachmentPathTemplate,
		NameAttachmentPublicBaseURL,
		NameAttachmentMaxFileSizeMB,
		NameAttachmentAllowedExtensions,
		NameAttachmentAllowedMIMETypes,
		NameAttachmentDefaultVisibility,
		NameAttachmentCleanupOrphanDays,
		NameAttachmentLocalPublicPrefix,
		NameAttachmentAliyunEndpoint,
		NameAttachmentAliyunBucket,
		NameAttachmentAliyunRegion,
		NameAttachmentAliyunAccessKeyID,
		NameAttachmentAliyunAccessKeySecret,
		NameAttachmentTencentRegion,
		NameAttachmentTencentBucket,
		NameAttachmentTencentSecretID,
		NameAttachmentTencentSecretKey,
		NameAttachmentTencentCDNDomain,
		NameAttachmentFTPHost,
		NameAttachmentFTPPort,
		NameAttachmentFTPUsername,
		NameAttachmentFTPPassword,
		NameAttachmentFTPRootPath,
		NameAttachmentFTPPassive,
		NameAttachmentFTPExplicitTLS,
		NameAttachmentFTPPublicBaseURL,
		NameAttachmentSFTPHost,
		NameAttachmentSFTPPort,
		NameAttachmentSFTPUsername,
		NameAttachmentSFTPPassword,
		NameAttachmentSFTPPrivateKey,
		NameAttachmentSFTPPassphrase,
		NameAttachmentSFTPRootPath,
		NameAttachmentSFTPHostKeyFingerprint,
		NameAttachmentSFTPPublicBaseURL,
	}
}

func coerceAttachmentOptions(values map[string]string, defaults map[string]string) {
	for _, name := range attachmentOptionNames() {
		normalized, ok := normalizeOptionValue(name, values[name])
		if !ok {
			values[name] = defaults[name]
			continue
		}
		values[name] = normalized
	}
}

func isValidAttachmentOptions(values map[string]string) bool {
	for _, name := range attachmentOptionNames() {
		if _, ok := normalizeOptionValue(name, values[name]); !ok {
			return false
		}
	}

	provider, ok := normalizeAttachmentProvider(values[NameAttachmentProvider])
	if !ok {
		return false
	}
	switch provider {
	case storage.ProviderLocal:
		return true
	case storage.ProviderAliyunOSS:
		return allNonBlank(values, NameAttachmentAliyunEndpoint, NameAttachmentAliyunBucket, NameAttachmentAliyunAccessKeyID, NameAttachmentAliyunAccessKeySecret)
	case storage.ProviderTencentCOS:
		return allNonBlank(values, NameAttachmentTencentRegion, NameAttachmentTencentBucket, NameAttachmentTencentSecretID, NameAttachmentTencentSecretKey)
	case storage.ProviderFTP:
		return allNonBlank(values, NameAttachmentFTPHost, NameAttachmentFTPUsername, NameAttachmentFTPPassword)
	case storage.ProviderSFTP:
		if !allNonBlank(values, NameAttachmentSFTPHost, NameAttachmentSFTPUsername) {
			return false
		}
		return strings.TrimSpace(values[NameAttachmentSFTPPassword]) != "" || strings.TrimSpace(values[NameAttachmentSFTPPrivateKey]) != ""
	default:
		return false
	}
}

func normalizeAttachmentProvider(value string) (string, bool) {
	value = strings.TrimSpace(value)
	if value == "" {
		return storage.ProviderLocal, true
	}
	return normalizeStringChoice(value, attachmentProviders)
}

func normalizeAttachmentPathTemplate(value string) (string, bool) {
	value = strings.TrimSpace(strings.ReplaceAll(value, "\\", "/"))
	if value == "" || strings.HasPrefix(value, "/") || strings.Contains(value, "..") {
		return "", false
	}
	if !strings.Contains(value, "{public_id}") {
		return "", false
	}
	if len([]rune(value)) > attachmentPathTemplateMaxRunes {
		return "", false
	}
	for _, part := range strings.Split(value, "/") {
		if strings.TrimSpace(part) == "" {
			return "", false
		}
	}
	return value, true
}

func normalizeAttachmentExtensions(value string) (string, bool) {
	parts := strings.Split(value, ",")
	seen := map[string]bool{}
	extensions := []string{}
	for _, part := range parts {
		item := strings.ToLower(strings.TrimSpace(part))
		if item == "" {
			continue
		}
		if !strings.HasPrefix(item, ".") {
			item = "." + item
		}
		if len(item) < 2 || len(item) > 24 {
			return "", false
		}
		for _, char := range strings.TrimPrefix(item, ".") {
			if (char < 'a' || char > 'z') && (char < '0' || char > '9') {
				return "", false
			}
		}
		if !seen[item] {
			seen[item] = true
			extensions = append(extensions, item)
		}
	}
	joined := strings.Join(extensions, ",")
	return joined, joined != "" && len([]rune(joined)) <= attachmentListOptionMaxRunes
}

func normalizeAttachmentMIMETypes(value string) (string, bool) {
	parts := strings.Split(value, ",")
	seen := map[string]bool{}
	items := []string{}
	for _, part := range parts {
		item := strings.ToLower(strings.TrimSpace(part))
		if item == "" {
			continue
		}
		segments := strings.Split(item, "/")
		if len(segments) != 2 || segments[0] == "" || segments[1] == "" {
			return "", false
		}
		for _, segment := range segments {
			if segment == "*" {
				continue
			}
			for _, char := range segment {
				if (char < 'a' || char > 'z') && (char < '0' || char > '9') && char != '-' && char != '+' && char != '.' {
					return "", false
				}
			}
		}
		if !seen[item] {
			seen[item] = true
			items = append(items, item)
		}
	}
	joined := strings.Join(items, ",")
	return joined, joined != "" && len([]rune(joined)) <= attachmentListOptionMaxRunes
}

func normalizeAttachmentRootPath(value string) (string, bool) {
	value = strings.TrimSpace(strings.ReplaceAll(value, "\\", "/"))
	if value == "" {
		return "/", true
	}
	if strings.Contains(value, "..") || len([]rune(value)) > attachmentProviderTextMaxRunes {
		return "", false
	}
	return value, true
}

func allNonBlank(values map[string]string, names ...string) bool {
	for _, name := range names {
		if strings.TrimSpace(values[name]) == "" {
			return false
		}
	}
	return true
}
