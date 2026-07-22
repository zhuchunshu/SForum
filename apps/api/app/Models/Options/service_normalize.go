package options

import (
	"encoding/json"
	"net/url"
	"strconv"
	"strings"
	"time"

	identity "github.com/zhuchunshu/sforum/apps/api/app/Models/Identity"
	humanverify "github.com/zhuchunshu/sforum/apps/api/app/Support/HumanVerify"
	localization "github.com/zhuchunshu/sforum/apps/api/app/Support/Localization"
)

func copyValues(values map[string]string) map[string]string {
	copied := make(map[string]string, len(values))
	for key, value := range values {
		copied[key] = value
	}
	return copied
}

func allOptionNames() []string {
	names := make([]string, 0, len(optionDefinitions))
	for _, definition := range optionDefinitions {
		names = append(names, definition.name)
	}
	return names
}

func normalizeName(name string) string {
	return strings.TrimSpace(name)
}

func normalizeOptionValue(name string, value string) (string, bool) {
	value = strings.TrimSpace(value)
	switch name {
	case NameSiteName:
		return value, value != "" && len([]rune(value)) <= 80
	case NameSiteURL:
		return value, isValidURL(value)
	case NameSiteDefaultLocale:
		return normalizeLocaleChoice(value, builtInLocales)
	case NameSiteSupportedLocales:
		locales := parseStoredLocales(value)
		if len(locales) == 0 {
			return "", false
		}
		return strings.Join(locales, ","), true
	case NameSiteTagline:
		return normalizeSiteTagline(value)
	case NameSiteAdminEmail:
		return normalizeSiteAdminEmail(value)
	case NameSiteLogoURL, NameSiteLogoAttachmentID,
		NameSiteFaviconURL, NameSiteFaviconAttachmentID,
		NameSiteAppleTouchIconURL, NameSiteAppleTouchIconAttachmentID,
		NameLegalTermsBodyZHCN, NameLegalTermsBodyENUS,
		NameLegalPrivacyBodyZHCN, NameLegalPrivacyBodyENUS,
		NameLegalGuidelinesBodyZHCN, NameLegalGuidelinesBodyENUS:
		return normalizeSiteBrandOption(name, value)
	case NameSiteTimezone:
		return normalizeSiteTimezone(value)
	case NameSiteDateFormat:
		return normalizeSiteDateFormat(value)
	case NameSiteTimeFormat:
		return normalizeSiteTimeFormat(value)
	case NameSiteStartOfWeek:
		return normalizeSiteStartOfWeek(value)
	case NamePublicSurfaceRevision:
		return normalizePublicSurfaceRevision(value)
	case NameHumanVerificationProvider:
		return normalizeHumanVerificationProvider(value)
	case NameHumanVerificationRegister, NameHumanVerificationPasswordReset, NameHumanVerificationLoginRisk, NameHumanVerificationPostRisk:
		return normalizeEnabledOption(value)
	case NameAltchaSecret:
		return value, true
	case NameAltchaChallengeTTL:
		duration, ok := parsePositiveDuration(value)
		if !ok {
			return "", false
		}
		return duration.String(), true
	case NameAltchaCost:
		parsed, ok := parsePositiveInt(value)
		if !ok {
			return "", false
		}
		return strconv.Itoa(parsed), true
	case NameAltchaWidgetType:
		return normalizeAltchaWidgetType(value)
	case NameAltchaWidgetAuto:
		return normalizeAltchaWidgetAuto(value)
	case NameAltchaWidgetDisplay:
		return normalizeAltchaWidgetDisplay(value)
	case NameAltchaWidgetHideLogo, NameAltchaWidgetHideFooter:
		return normalizeEnabledOption(value)
	case NameAltchaWidgetWorkers:
		parsed, ok := parseBoundedInt(value, altchaWidgetWorkersMin, altchaWidgetWorkersMax)
		if !ok {
			return "", false
		}
		return strconv.Itoa(parsed), true
	case NameAltchaWidgetMinDuration:
		parsed, ok := parseBoundedInt(value, altchaWidgetMinDurationMin, altchaWidgetMinDurationMax)
		if !ok {
			return "", false
		}
		return strconv.Itoa(parsed), true
	case NameAppearanceTheme:
		return normalizeAppearanceTheme(value)
	case NameFooterCopyrightZHCN, NameFooterCopyrightENUS:
		return normalizeFooterCopyright(value)
	case NameFooterLinks:
		return normalizeFooterLinks(value)
	case NameIdentityPasswordMinLength:
		return normalizeBoundedInt(value, passwordMinLengthMin, passwordMinLengthMax)
	case NameIdentityPasswordMaxLength:
		return normalizeBoundedInt(value, passwordMaxLengthMin, passwordMaxLengthMax)
	case NameIdentityPasswordRequireLowercase, NameIdentityPasswordRequireUppercase, NameIdentityPasswordRequireNumber, NameIdentityPasswordRequireSymbol, NameIdentityRegistrationEnabled:
		return normalizeEnabledOption(value)
	case NameIdentityRegistrationMode,
		NameIdentityRegistrationRequireEmailVerification,
		NameIdentityRegistrationBlockPostingUntilVerified,
		NameIdentityUsernameMinLength,
		NameIdentityUsernameMaxLength,
		NameIdentityUsernameCharset,
		NameIdentityUsernameReserved,
		NameIdentityLoginMaxFailures,
		NameIdentityLoginLockoutMinutes,
		NameTrustNewUserDays,
		NameTrustNewUserTopicCooldownSeconds,
		NameTrustNewUserCommentCooldownSeconds,
		NameTrustNewUserDailyTopicLimit,
		NameTrustNewUserDailyCommentLimit,
		NameTrustNewUserForbidOutboundLinks,
		NameTrustNewUserForbidAttachments,
		NameSiteMaintenanceEnabled,
		NameSiteMaintenanceMessage,
		NameForumGuestRead,
		NameForumListDefaultSort,
		NameForumListHotWindowDays,
		NameForumTopicsAllowAuthorCloseReplies,
		NameForumTopicsAllowAuthorDelete,
		NameForumTopicsAutoLockIdleDays,
		NameForumTopicsShowEditMark,
		NameForumTopicsDuplicateTitlePolicy,
		NameForumCommentsShowEditMark,
		NameForumCommentsSoftDeleteVisibility,
		NameForumMentionsEnabled,
		NameForumMentionsMaxPerPost:
		return normalizeCommunityPolicyOption(name, value)
	case NameIdentitySessionsMaxDevices:
		// 限制在 1-20；非法值返回 false，上游保留默认值（beginner-friendly：配置错误不致功能失效）。
		return normalizeBoundedInt(value, sessionsMaxDevicesMin, sessionsMaxDevicesMax)
	case NameIdentitySessionsKeepDays:
		return normalizeBoundedInt(value, sessionsKeepDaysMin, sessionsKeepDaysMax)
	case NameForumDefaultCategorySlug:
		return normalizeForumSlug(value)
	case NameForumTagCreationMode:
		return normalizeForumTagCreationMode(value)
	case NameForumTagPublicPages:
		return normalizeEnabledOption(value)
	case NameFeatureSearch, NameFeatureRegistration, NameFeatureAttachments, NameFeatureMentions, NameFeaturePublicProfiles, NameFeatureWebhooks:
		// F4.5：产品开关仅接受 enabled/disabled。
		return normalizeEnabledOption(value)
	case NamePagesRegistryEnabled, NameThemesRuntimeL0Enabled, NameThemesRuntimeL1Enabled:
		// Page Registry / runtime theme dual-stack 开关。
		return normalizeEnabledOption(value)
	case NameForumTagMinPerTopic, NameForumTagMaxPerTopic:
		return normalizeBoundedInt(value, forumTagMaxPerTopicMin, forumTagMaxPerTopicMax)
	case NameForumTopicsPerPage, NameForumCommentsPerPage:
		return normalizeBoundedInt(value, forumPaginationMin, forumPaginationMax)
	case NameForumTopicTitleMinRunes:
		return normalizeBoundedInt(value, forumTitleMinRunesMin, forumTitleMinRunesMax)
	case NameForumTopicTitleMaxRunes:
		return normalizeBoundedInt(value, forumTitleMaxRunesMin, forumTitleMaxRunesMax)
	case NameForumTopicContentMinRunes:
		return normalizeBoundedInt(value, forumContentMinRunesMin, forumContentMinRunesMax)
	case NameForumTopicContentMaxRunes:
		return normalizeBoundedInt(value, forumContentMaxRunesMin, forumContentMaxRunesMax)
	case NameForumCommentMinRunes:
		return normalizeBoundedInt(value, forumCommentMinRunesMin, forumCommentMinRunesMax)
	case NameForumCommentMaxRunes:
		return normalizeBoundedInt(value, forumCommentMaxRunesMin, forumCommentMaxRunesMax)
	case NameForumCommentMaxNestingDepth:
		return normalizeBoundedInt(value, forumNestingMin, forumNestingMax)
	case NameForumCommentsTreeDescendantsPerRoot:
		return normalizeBoundedInt(value, forumTreeDescendantsMin, forumTreeDescendantsMax)
	case NameForumTopicEditWindowMinutes, NameForumCommentEditWindowMinutes:
		return normalizeBoundedInt(value, forumEditWindowMin, forumEditWindowMax)
	case NameForumTopicCooldownSeconds, NameForumCommentCooldownSeconds:
		return normalizeBoundedInt(value, forumCooldownMin, forumCooldownMax)
	case NameForumDailyTopicLimit, NameForumDailyCommentLimit:
		return normalizeBoundedInt(value, forumDailyLimitMin, forumDailyLimitMax)
	case NameForumExcerptRuneLimit:
		return normalizeBoundedInt(value, forumExcerptMin, forumExcerptMax)
	case NameSEOMetaTitleTemplate:
		return normalizeSEOTitleTemplate(value)
	case NameSEOMetaDescription:
		return normalizeBoundedText(value, seoDescriptionMaxRunes)
	case NameSEOMetaKeywords:
		return normalizeBoundedText(value, seoKeywordsMaxRunes)
	case NameSEOOGImageURL, NameSEOSchemaOrgOrganizationLogo:
		return normalizeOptionalURL(value)
	case NameSEOTwitterCard:
		return normalizeSEOTwitterCard(value)
	case NameSEOTwitterSite:
		return normalizeSEOTwitterSite(value)
	case NameSEOAllowIndexing, NameSEORobotsBlockAIBots, NameSEORobotsBlockNonSEOBots, NameSEOSitemapEnabled, NameSEOSitemapIncludeStaticPages, NameSEOSitemapIncludeForumContent, NameSEOSchemaOrgEnabled, NameSEOSchemaOrgSearchAction, NameSEOSchemaOrgDiscussion:
		return normalizeEnabledOption(value)
	case NameSEOGoogleVerification, NameSEOBingVerification, NameSEOBaiduVerification, NameSEOYandexVerification:
		return normalizeSEOVerificationToken(value)
	case NameSEORobotsExtraAllow, NameSEORobotsExtraDisallow:
		return normalizeSEORobotsPathList(value)
	case NameSEOTopicURLMode:
		// 帖子 URL 形态枚举，大小写归一后白名单校验。
		return normalizeChoice(value, []string{"id_slug", "id", "slug"})
	case NameAttachmentProvider:
		return normalizeAttachmentProvider(value)
	case NameAttachmentUploadEnabled, NameAttachmentFTPPassive, NameAttachmentFTPExplicitTLS:
		return normalizeEnabledOption(value)
	case NameAttachmentPathTemplate:
		return normalizeAttachmentPathTemplate(value)
	case NameAttachmentLocalRoot:
		return normalizeAttachmentLocalRoot(value)
	case NameAttachmentPublicBaseURL, NameAttachmentLocalPublicPrefix, NameAttachmentTencentCDNDomain, NameAttachmentFTPPublicBaseURL, NameAttachmentSFTPPublicBaseURL:
		return normalizeOptionalURL(value)
	case NameAttachmentMaxFileSizeMB:
		return normalizeBoundedInt(value, 1, 1024)
	case NameAttachmentAllowedExtensions:
		return normalizeAttachmentExtensions(value)
	case NameAttachmentAllowedMIMETypes:
		return normalizeAttachmentMIMETypes(value)
	case NameAttachmentDefaultVisibility:
		return normalizeStringChoice(value, attachmentVisibilities)
	case NameAttachmentCleanupOrphanDays:
		return normalizeBoundedInt(value, 1, 3650)
	case NameAttachmentFTPPort, NameAttachmentSFTPPort:
		return normalizeBoundedInt(value, 1, 65535)
	case NameAttachmentAliyunEndpoint, NameAttachmentAliyunBucket, NameAttachmentAliyunRegion, NameAttachmentAliyunAccessKeyID,
		NameAttachmentTencentRegion, NameAttachmentTencentBucket, NameAttachmentTencentSecretID,
		NameAttachmentFTPHost, NameAttachmentFTPUsername,
		NameAttachmentSFTPHost, NameAttachmentSFTPUsername, NameAttachmentSFTPHostKeyFingerprint:
		return normalizeBoundedText(value, attachmentProviderTextMaxRunes)
	case NameAttachmentFTPRootPath, NameAttachmentSFTPRootPath:
		return normalizeAttachmentRootPath(value)
	case NameAttachmentAliyunAccessKeySecret, NameAttachmentTencentSecretKey, NameAttachmentFTPPassword, NameAttachmentSFTPPassword, NameAttachmentSFTPPrivateKey, NameAttachmentSFTPPassphrase:
		return normalizeBoundedText(value, attachmentSecretMaxRunes)
	case NameAvatarAllowUpload, NameAvatarAllowGIF, NameAvatarCompressEnabled:
		return normalizeEnabledOption(value)
	case NameAvatarDefaultProvider:
		return normalizeAvatarProvider(value)
	case NameAvatarGravatarBaseURL:
		return normalizeAvatarGravatarBaseURL(value)
	case NameAvatarGravatarHashAlgorithm:
		return normalizeAvatarHashAlgorithm(value)
	case NameAvatarDefaultStaticURL:
		return normalizeOptionalURL(value)
	case NameAvatarMaxSizeKB:
		return normalizeBoundedInt(value, avatarMaxSizeKBMin, avatarMaxSizeKBMax)
	case NameAvatarMaxDimension:
		return normalizeBoundedInt(value, avatarDimensionMin, avatarDimensionMax)
	case NameAvatarTargetDimension:
		return normalizeBoundedInt(value, avatarDimensionMin, avatarDimensionMax)
	case NameAvatarCompressQuality:
		return normalizeBoundedInt(value, avatarCompressQualityMin, avatarCompressQualityMax)
	case NameNotificationReplyInApp, NameNotificationReplyEmail, NameNotificationMentionInApp, NameNotificationMentionEmail, NameNotificationModerationInApp, NameNotificationModerationEmail:
		return normalizeEnabledOption(value)
	default:
		return normalizeSEOOption(name, value)
	}
}

// 邮件 provider 白名单。

func isValidValueSet(values map[string]string) bool {
	if strings.TrimSpace(values[NameSiteName]) == "" || len([]rune(values[NameSiteName])) > 80 {
		return false
	}
	if !isValidURL(values[NameSiteURL]) {
		return false
	}

	supported := parseStoredLocales(values[NameSiteSupportedLocales])
	if len(supported) == 0 {
		return false
	}
	if _, ok := normalizeLocaleChoice(values[NameSiteDefaultLocale], supported); !ok {
		return false
	}
	if !isValidSiteDateTimeOptions(values) {
		return false
	}
	if !isValidSiteIdentityOptions(values) {
		return false
	}
	if !isValidSiteBrandOptions(values) {
		return false
	}
	if !isValidCommunityPolicyOptions(values) {
		return false
	}

	provider, ok := normalizeHumanVerificationProvider(values[NameHumanVerificationProvider])
	if !ok {
		return false
	}
	if provider == humanverify.ProviderAltcha && strings.TrimSpace(values[NameAltchaSecret]) == "" {
		return false
	}
	for _, scenario := range humanVerificationScenarios {
		if _, ok := normalizeEnabledOption(values[scenario.name]); !ok {
			return false
		}
	}
	if _, ok := parsePositiveDuration(values[NameAltchaChallengeTTL]); !ok {
		return false
	}
	if _, ok := parsePositiveInt(values[NameAltchaCost]); !ok {
		return false
	}
	if _, ok := normalizeAltchaWidgetType(values[NameAltchaWidgetType]); !ok {
		return false
	}
	if _, ok := normalizeAltchaWidgetAuto(values[NameAltchaWidgetAuto]); !ok {
		return false
	}
	if _, ok := normalizeAltchaWidgetDisplay(values[NameAltchaWidgetDisplay]); !ok {
		return false
	}
	if _, ok := normalizeEnabledOption(values[NameAltchaWidgetHideLogo]); !ok {
		return false
	}
	if _, ok := normalizeEnabledOption(values[NameAltchaWidgetHideFooter]); !ok {
		return false
	}
	if _, ok := parseBoundedInt(values[NameAltchaWidgetWorkers], altchaWidgetWorkersMin, altchaWidgetWorkersMax); !ok {
		return false
	}
	if _, ok := parseBoundedInt(values[NameAltchaWidgetMinDuration], altchaWidgetMinDurationMin, altchaWidgetMinDurationMax); !ok {
		return false
	}
	if _, ok := normalizeAppearanceTheme(values[NameAppearanceTheme]); !ok {
		return false
	}
	if _, ok := normalizeFooterCopyright(values[NameFooterCopyrightZHCN]); !ok {
		return false
	}
	if _, ok := normalizeFooterCopyright(values[NameFooterCopyrightENUS]); !ok {
		return false
	}
	if _, ok := normalizeFooterLinks(values[NameFooterLinks]); !ok {
		return false
	}
	minLength, minOK := parseBoundedInt(values[NameIdentityPasswordMinLength], passwordMinLengthMin, passwordMinLengthMax)
	maxLength, maxOK := parseBoundedInt(values[NameIdentityPasswordMaxLength], passwordMaxLengthMin, passwordMaxLengthMax)
	if !minOK || !maxOK || maxLength < minLength {
		return false
	}
	for _, name := range passwordPolicyBooleanOptionNames() {
		if _, ok := normalizeEnabledOption(values[name]); !ok {
			return false
		}
	}
	if _, ok := normalizeEnabledOption(values[NameIdentityRegistrationEnabled]); !ok {
		return false
	}
	if _, ok := normalizeForumSlug(values[NameForumDefaultCategorySlug]); !ok {
		return false
	}
	if _, ok := normalizeForumTagCreationMode(values[NameForumTagCreationMode]); !ok {
		return false
	}
	if _, ok := normalizeEnabledOption(values[NameForumTagPublicPages]); !ok {
		return false
	}
	if _, ok := parseBoundedInt(values[NameForumTagMinPerTopic], forumTagMaxPerTopicMin, forumTagMaxPerTopicMax); !ok {
		return false
	}
	if _, ok := parseBoundedInt(values[NameForumTagMaxPerTopic], forumTagMaxPerTopicMin, forumTagMaxPerTopicMax); !ok {
		return false
	}
	if minTags, okMin := parseBoundedInt(values[NameForumTagMinPerTopic], forumTagMaxPerTopicMin, forumTagMaxPerTopicMax); okMin {
		if maxTags, okMax := parseBoundedInt(values[NameForumTagMaxPerTopic], forumTagMaxPerTopicMin, forumTagMaxPerTopicMax); okMax && minTags > maxTags {
			return false
		}
	}
	for _, name := range []string{NameForumTopicsPerPage, NameForumCommentsPerPage} {
		if _, ok := parseBoundedInt(values[name], forumPaginationMin, forumPaginationMax); !ok {
			return false
		}
	}
	if !validForumContentLimitOptionValues(values) {
		return false
	}
	if _, ok := normalizeSEOTitleTemplate(values[NameSEOMetaTitleTemplate]); !ok {
		return false
	}
	if _, ok := normalizeBoundedText(values[NameSEOMetaDescription], seoDescriptionMaxRunes); !ok {
		return false
	}
	if _, ok := normalizeBoundedText(values[NameSEOMetaKeywords], seoKeywordsMaxRunes); !ok {
		return false
	}
	if _, ok := normalizeOptionalURL(values[NameSEOOGImageURL]); !ok {
		return false
	}
	if _, ok := normalizeSEOTwitterCard(values[NameSEOTwitterCard]); !ok {
		return false
	}
	if _, ok := normalizeSEOTwitterSite(values[NameSEOTwitterSite]); !ok {
		return false
	}
	for _, name := range seoEnabledOptionNames() {
		if _, ok := normalizeEnabledOption(values[name]); !ok {
			return false
		}
	}
	for _, name := range seoVerificationOptionNames() {
		if _, ok := normalizeSEOVerificationToken(values[name]); !ok {
			return false
		}
	}
	if _, ok := normalizeSEORobotsPathList(values[NameSEORobotsExtraAllow]); !ok {
		return false
	}
	if _, ok := normalizeSEORobotsPathList(values[NameSEORobotsExtraDisallow]); !ok {
		return false
	}
	if _, ok := normalizeOptionalURL(values[NameSEOSchemaOrgOrganizationLogo]); !ok {
		return false
	}
	if !isValidAttachmentOptions(values) {
		return false
	}
	if !isValidAvatarOptions(values) {
		return false
	}
	return true
}

func isKnownOption(name string) bool {
	_, ok := optionDefinitionFor(name)
	return ok
}

func isPublicOption(name string) bool {
	definition, ok := optionDefinitionFor(name)
	return ok && definition.public
}

func isSecretOption(name string) bool {
	definition, ok := optionDefinitionFor(name)
	return ok && definition.secret
}

// encryptValue 加密敏感值；未配置 cipher 时透明返回原文（开发环境）。
func (s *Service) encryptValue(value string) (string, error) {
	if s.cipher == nil {
		return value, nil
	}
	return s.cipher.Encrypt(value)
}

// decryptValue 解密敏感值；未配置 cipher 或值为明文时透明返回原文（兼容历史明文）。
func (s *Service) decryptValue(value string) string {
	if s.cipher == nil {
		return value
	}
	decrypted, err := s.cipher.Decrypt(value)
	if err != nil {
		// 解密失败（密钥轮换/数据损坏）时回退原文，避免单个坏值拖垮全部 options 读取。
		return value
	}
	return decrypted
}

func optionDefinitionFor(name string) (optionDefinition, bool) {
	for _, definition := range optionDefinitions {
		if definition.name == name {
			return definition, true
		}
	}
	return optionDefinition{}, false
}

func actorCanManageAnyOption(actor identity.Actor) bool {
	for _, definition := range optionDefinitions {
		if actor.Can(definition.managePermission) {
			return true
		}
	}
	return false
}

func isValidURL(value string) bool {
	parsed, err := url.Parse(strings.TrimSpace(value))
	if err != nil || parsed == nil || parsed.Host == "" {
		return false
	}
	return parsed.Scheme == "http" || parsed.Scheme == "https"
}

func normalizeLocaleList(values []string) []string {
	locales := make([]string, 0, len(values))
	seen := map[string]bool{}
	for _, value := range values {
		locale, ok := normalizeLocaleChoice(value, builtInLocales)
		if !ok || seen[locale] {
			continue
		}
		seen[locale] = true
		locales = append(locales, locale)
	}
	return locales
}

func parseStoredLocales(value string) []string {
	parts := strings.Split(value, ",")
	locales := make([]string, 0, len(parts))
	seen := map[string]bool{}
	for _, part := range parts {
		locale, ok := normalizeLocaleChoice(part, builtInLocales)
		if !ok || seen[locale] {
			continue
		}
		seen[locale] = true
		locales = append(locales, locale)
	}
	return locales
}

func normalizeLocaleChoice(value string, allowed []string) (string, bool) {
	value = strings.TrimSpace(value)
	if value == "" {
		return "", false
	}
	candidate := localization.Normalize(value, nil)
	for _, locale := range allowed {
		if strings.EqualFold(candidate, locale) {
			return locale, true
		}
	}
	return "", false
}

func normalizeHumanVerificationProvider(value string) (string, bool) {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "", humanverify.ProviderDisabled:
		return humanverify.ProviderDisabled, true
	case humanverify.ProviderAltcha:
		return humanverify.ProviderAltcha, true
	default:
		return "", false
	}
}

func normalizeEnabledOption(value string) (string, bool) {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "enabled", "true", "1", "yes", "on":
		return enabledOptionValue(true), true
	case "disabled", "false", "0", "no", "off":
		return enabledOptionValue(false), true
	default:
		return "", false
	}
}

func enabledOptionValue(enabled bool) string {
	if enabled {
		return "enabled"
	}
	return "disabled"
}

func isEnabledOption(value string) bool {
	normalized, ok := normalizeEnabledOption(value)
	return ok && normalized == enabledOptionValue(true)
}

func parsePositiveDuration(value string) (time.Duration, bool) {
	duration, err := time.ParseDuration(strings.TrimSpace(value))
	return duration, err == nil && duration > 0
}

func parsePositiveInt(value string) (int, bool) {
	parsed, err := strconv.Atoi(strings.TrimSpace(value))
	return parsed, err == nil && parsed > 0
}

func parseBoundedInt(value string, min int, max int) (int, bool) {
	parsed, err := strconv.Atoi(strings.TrimSpace(value))
	return parsed, err == nil && parsed >= min && parsed <= max
}

func normalizeAltchaWidgetType(value string) (string, bool) {
	return normalizeChoice(value, altchaWidgetTypes)
}

func normalizeAltchaWidgetAuto(value string) (string, bool) {
	return normalizeChoice(value, altchaWidgetAutoModes)
}

func normalizeAltchaWidgetDisplay(value string) (string, bool) {
	return normalizeChoice(value, altchaWidgetDisplays)
}

func normalizeChoice(value string, allowed []string) (string, bool) {
	value = strings.ToLower(strings.TrimSpace(value))
	for _, item := range allowed {
		if value == item {
			return item, true
		}
	}
	return "", false
}

func normalizeStringChoice(value string, allowed []string) (string, bool) {
	value = strings.TrimSpace(value)
	for _, item := range allowed {
		if strings.EqualFold(value, item) {
			return item, true
		}
	}
	return "", false
}

func normalizeForumSlug(value string) (string, bool) {
	value = strings.ToLower(strings.TrimSpace(value))
	return value, forumSlugPattern.MatchString(value)
}

func normalizeForumTagCreationMode(value string) (string, bool) {
	return normalizeChoice(value, forumTagCreationModes)
}

func normalizeBoundedInt(value string, min int, max int) (string, bool) {
	parsed, ok := parseBoundedInt(value, min, max)
	if !ok {
		return "", false
	}
	return strconv.Itoa(parsed), true
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
	value = strings.TrimSpace(strings.ToLower(value))
	value = strings.TrimPrefix(value, "#")
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

func normalizeFooterCopyright(value string) (string, bool) {
	value = strings.TrimSpace(value)
	return value, len([]rune(value)) <= footerCopyrightMaxRunes
}

func normalizeSEOTitleTemplate(value string) (string, bool) {
	return normalizeBoundedText(value, seoTitleTemplateMaxRunes)
}

func normalizeBoundedText(value string, maxRunes int) (string, bool) {
	value = strings.TrimSpace(value)
	return value, len([]rune(value)) <= maxRunes
}

func normalizeOptionalURL(value string) (string, bool) {
	value = strings.TrimSpace(value)
	if value == "" {
		return "", true
	}
	if !isValidURL(value) {
		return "", false
	}
	return value, true
}

func normalizeSEOTwitterCard(value string) (string, bool) {
	value = strings.ToLower(strings.TrimSpace(value))
	for _, card := range seoTwitterCards {
		if value == card {
			return card, true
		}
	}
	return "", false
}

func normalizeSEOTwitterSite(value string) (string, bool) {
	value = strings.TrimSpace(value)
	if value == "" {
		return "", true
	}
	if len([]rune(value)) > seoTwitterSiteMaxRunes {
		return "", false
	}
	if strings.HasPrefix(value, "@") {
		value = strings.TrimPrefix(value, "@")
	}
	if value == "" {
		return "", true
	}
	for _, char := range value {
		if (char < 'a' || char > 'z') && (char < 'A' || char > 'Z') && (char < '0' || char > '9') && char != '_' {
			return "", false
		}
	}
	return "@" + value, true
}

func normalizeSEOVerificationToken(value string) (string, bool) {
	value = strings.TrimSpace(value)
	if len([]rune(value)) > seoVerificationMaxRunes {
		return "", false
	}
	if value == "" {
		return "", true
	}
	for _, char := range value {
		if char <= ' ' || char == '<' || char == '>' || char == '"' || char == '\'' {
			return "", false
		}
	}
	return value, true
}

func normalizeSEORobotsPathList(value string) (string, bool) {
	value = strings.ReplaceAll(value, "\r\n", "\n")
	value = strings.ReplaceAll(value, "\r", "\n")
	lines := strings.Split(strings.TrimSpace(value), "\n")
	if strings.TrimSpace(value) == "" {
		return "", true
	}

	normalized := make([]string, 0, len(lines))
	seen := map[string]bool{}
	for _, line := range lines {
		path := strings.TrimSpace(line)
		if path == "" {
			continue
		}
		if !isValidRobotsPath(path) || seen[path] {
			return "", false
		}
		seen[path] = true
		normalized = append(normalized, path)
	}

	result := strings.Join(normalized, "\n")
	return result, len([]rune(result)) <= seoRobotsPathListMaxRunes
}

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
func coerceForumContentLimitOptions(coerced map[string]string, defaults map[string]string) {
	type bound struct {
		name string
		min  int
		max  int
	}
	for _, item := range []bound{
		{NameForumTopicTitleMinRunes, forumTitleMinRunesMin, forumTitleMinRunesMax},
		{NameForumTopicTitleMaxRunes, forumTitleMaxRunesMin, forumTitleMaxRunesMax},
		{NameForumTopicContentMinRunes, forumContentMinRunesMin, forumContentMinRunesMax},
		{NameForumTopicContentMaxRunes, forumContentMaxRunesMin, forumContentMaxRunesMax},
		{NameForumCommentMinRunes, forumCommentMinRunesMin, forumCommentMinRunesMax},
		{NameForumCommentMaxRunes, forumCommentMaxRunesMin, forumCommentMaxRunesMax},
		{NameForumCommentMaxNestingDepth, forumNestingMin, forumNestingMax},
		{NameForumCommentsTreeDescendantsPerRoot, forumTreeDescendantsMin, forumTreeDescendantsMax},
		{NameForumTopicEditWindowMinutes, forumEditWindowMin, forumEditWindowMax},
		{NameForumCommentEditWindowMinutes, forumEditWindowMin, forumEditWindowMax},
		{NameForumTopicCooldownSeconds, forumCooldownMin, forumCooldownMax},
		{NameForumCommentCooldownSeconds, forumCooldownMin, forumCooldownMax},
		{NameForumDailyTopicLimit, forumDailyLimitMin, forumDailyLimitMax},
		{NameForumDailyCommentLimit, forumDailyLimitMin, forumDailyLimitMax},
		{NameForumExcerptRuneLimit, forumExcerptMin, forumExcerptMax},
	} {
		if _, ok := parseBoundedInt(coerced[item.name], item.min, item.max); !ok {
			coerced[item.name] = defaults[item.name]
		}
	}
	// 标题/正文/评论 min 不得超过 max，否则回退整对。
	resetPair := func(minName, maxName string) {
		minVal, okMin := parseBoundedInt(coerced[minName], 0, 1<<30)
		maxVal, okMax := parseBoundedInt(coerced[maxName], 0, 1<<30)
		if okMin && okMax && minVal > maxVal {
			coerced[minName] = defaults[minName]
			coerced[maxName] = defaults[maxName]
		}
	}
	resetPair(NameForumTopicTitleMinRunes, NameForumTopicTitleMaxRunes)
	resetPair(NameForumTopicContentMinRunes, NameForumTopicContentMaxRunes)
	resetPair(NameForumCommentMinRunes, NameForumCommentMaxRunes)
}

func validForumContentLimitOptionValues(values map[string]string) bool {
	type bound struct {
		name string
		min  int
		max  int
	}
	for _, item := range []bound{
		{NameForumTopicTitleMinRunes, forumTitleMinRunesMin, forumTitleMinRunesMax},
		{NameForumTopicTitleMaxRunes, forumTitleMaxRunesMin, forumTitleMaxRunesMax},
		{NameForumTopicContentMinRunes, forumContentMinRunesMin, forumContentMinRunesMax},
		{NameForumTopicContentMaxRunes, forumContentMaxRunesMin, forumContentMaxRunesMax},
		{NameForumCommentMinRunes, forumCommentMinRunesMin, forumCommentMinRunesMax},
		{NameForumCommentMaxRunes, forumCommentMaxRunesMin, forumCommentMaxRunesMax},
		{NameForumCommentMaxNestingDepth, forumNestingMin, forumNestingMax},
		{NameForumCommentsTreeDescendantsPerRoot, forumTreeDescendantsMin, forumTreeDescendantsMax},
		{NameForumTopicEditWindowMinutes, forumEditWindowMin, forumEditWindowMax},
		{NameForumCommentEditWindowMinutes, forumEditWindowMin, forumEditWindowMax},
		{NameForumTopicCooldownSeconds, forumCooldownMin, forumCooldownMax},
		{NameForumCommentCooldownSeconds, forumCooldownMin, forumCooldownMax},
		{NameForumDailyTopicLimit, forumDailyLimitMin, forumDailyLimitMax},
		{NameForumDailyCommentLimit, forumDailyLimitMin, forumDailyLimitMax},
		{NameForumExcerptRuneLimit, forumExcerptMin, forumExcerptMax},
	} {
		if _, ok := parseBoundedInt(values[item.name], item.min, item.max); !ok {
			return false
		}
	}
	pairOK := func(minName, maxName string) bool {
		minVal, okMin := parseBoundedInt(values[minName], 0, 1<<30)
		maxVal, okMax := parseBoundedInt(values[maxName], 0, 1<<30)
		return okMin && okMax && minVal <= maxVal
	}
	return pairOK(NameForumTopicTitleMinRunes, NameForumTopicTitleMaxRunes) &&
		pairOK(NameForumTopicContentMinRunes, NameForumTopicContentMaxRunes) &&
		pairOK(NameForumCommentMinRunes, NameForumCommentMaxRunes)
}
