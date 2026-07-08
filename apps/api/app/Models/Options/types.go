package options

import "errors"

const (
	NameSiteName                         = "site.name"
	NameSiteURL                          = "site.url"
	NameSiteDefaultLocale                = "site.default_locale"
	NameSiteSupportedLocales             = "site.supported_locales"
	NameHumanVerificationProvider        = "human_verification.provider"
	NameHumanVerificationRegister        = "human_verification.scenarios.register"
	NameHumanVerificationPasswordReset   = "human_verification.scenarios.password_reset"
	NameHumanVerificationLoginRisk       = "human_verification.scenarios.login_risk"
	NameHumanVerificationPostRisk        = "human_verification.scenarios.post_risk"
	NameAltchaSecret                     = "human_verification.altcha.secret"
	NameAltchaChallengeTTL               = "human_verification.altcha.challenge_ttl"
	NameAltchaCost                       = "human_verification.altcha.cost"
	NameAltchaWidgetType                 = "human_verification.altcha.widget.type"
	NameAltchaWidgetAuto                 = "human_verification.altcha.widget.auto"
	NameAltchaWidgetDisplay              = "human_verification.altcha.widget.display"
	NameAltchaWidgetHideLogo             = "human_verification.altcha.widget.hide_logo"
	NameAltchaWidgetHideFooter           = "human_verification.altcha.widget.hide_footer"
	NameAltchaWidgetWorkers              = "human_verification.altcha.widget.workers"
	NameAltchaWidgetMinDuration          = "human_verification.altcha.widget.min_duration_ms"
	NameAppearanceTheme                  = "appearance.theme"
	NameFooterCopyrightZHCN              = "footer.copyright.zh-CN"
	NameFooterCopyrightENUS              = "footer.copyright.en-US"
	NameFooterLinks                      = "footer.links"
	NameIdentityPasswordMinLength        = "identity.password.min_length"
	NameIdentityPasswordMaxLength        = "identity.password.max_length"
	NameIdentityPasswordRequireLowercase = "identity.password.require_lowercase"
	NameIdentityPasswordRequireUppercase = "identity.password.require_uppercase"
	NameIdentityPasswordRequireNumber    = "identity.password.require_number"
	NameIdentityPasswordRequireSymbol    = "identity.password.require_symbol"
	NameForumDefaultCategorySlug         = "forum.default_category_slug"
	NameForumTagCreationMode             = "forum.tags.creation_mode"
	NameForumTagPublicPages              = "forum.tags.public_pages"
	NameForumTagMaxPerTopic              = "forum.tags.max_per_topic"
	NameSEOMetaTitleTemplate             = "seo.meta_title_template"
	NameSEOMetaDescription               = "seo.meta_description"
	NameSEOMetaKeywords                  = "seo.meta_keywords"
	NameSEOOGImageURL                    = "seo.og_image_url"
	NameSEOTwitterCard                   = "seo.twitter_card"
	NameSEOTwitterSite                   = "seo.twitter_site"
	NameSEOAllowIndexing                 = "seo.allow_indexing"
	NameSEOGoogleVerification            = "seo.google_verification"
	NameSEOBingVerification              = "seo.bing_verification"
	NameSEOBaiduVerification             = "seo.baidu_verification"
	NameSEOYandexVerification            = "seo.yandex_verification"
	NameSEORobotsExtraAllow              = "seo.robots.extra_allow"
	NameSEORobotsExtraDisallow           = "seo.robots.extra_disallow"
	NameSEORobotsBlockAIBots             = "seo.robots.block_ai_bots"
	NameSEORobotsBlockNonSEOBots         = "seo.robots.block_non_seo_bots"
	NameSEOSitemapEnabled                = "seo.sitemap.enabled"
	NameSEOSitemapIncludeStaticPages     = "seo.sitemap.include_static_pages"
	NameSEOSitemapIncludeForumContent    = "seo.sitemap.include_forum_content"
	NameSEOSchemaOrgEnabled              = "seo.schema_org.enabled"
	NameSEOSchemaOrgSearchAction         = "seo.schema_org.search_action_enabled"
	NameSEOSchemaOrgDiscussion           = "seo.schema_org.discussion_enabled"
	NameSEOSchemaOrgOrganizationLogo     = "seo.schema_org.organization_logo_url"
	// 帖子详情页 URL 形态：id_slug | id | slug。默认 id_slug。
	NameSEOTopicURLMode                  = "seo.topic_url_mode"
	NameAttachmentProvider               = "attachment.provider"
	NameAttachmentUploadEnabled          = "attachment.upload.enabled"
	NameAttachmentPathTemplate           = "attachment.path_template"
	NameAttachmentPublicBaseURL          = "attachment.public_base_url"
	NameAttachmentMaxFileSizeMB          = "attachment.max_file_size_mb"
	NameAttachmentAllowedExtensions      = "attachment.allowed_extensions"
	NameAttachmentAllowedMIMETypes       = "attachment.allowed_mime_types"
	NameAttachmentDefaultVisibility      = "attachment.default_visibility"
	NameAttachmentCleanupOrphanDays      = "attachment.cleanup_orphan_after_days"
	NameAttachmentLocalRoot              = "attachment.local.root"
	NameAttachmentLocalPublicPrefix      = "attachment.local.public_prefix"
	NameAttachmentAliyunEndpoint         = "attachment.aliyun_oss.endpoint"
	NameAttachmentAliyunBucket           = "attachment.aliyun_oss.bucket"
	NameAttachmentAliyunRegion           = "attachment.aliyun_oss.region"
	NameAttachmentAliyunAccessKeyID      = "attachment.aliyun_oss.access_key_id"
	NameAttachmentAliyunAccessKeySecret  = "attachment.aliyun_oss.access_key_secret"
	NameAttachmentTencentRegion          = "attachment.tencent_cos.region"
	NameAttachmentTencentBucket          = "attachment.tencent_cos.bucket"
	NameAttachmentTencentSecretID        = "attachment.tencent_cos.secret_id"
	NameAttachmentTencentSecretKey       = "attachment.tencent_cos.secret_key"
	NameAttachmentTencentCDNDomain       = "attachment.tencent_cos.cdn_domain"
	NameAttachmentFTPHost                = "attachment.ftp.host"
	NameAttachmentFTPPort                = "attachment.ftp.port"
	NameAttachmentFTPUsername            = "attachment.ftp.username"
	NameAttachmentFTPPassword            = "attachment.ftp.password"
	NameAttachmentFTPRootPath            = "attachment.ftp.root_path"
	NameAttachmentFTPPassive             = "attachment.ftp.passive"
	NameAttachmentFTPExplicitTLS         = "attachment.ftp.explicit_tls"
	NameAttachmentFTPPublicBaseURL       = "attachment.ftp.public_base_url"
	NameAttachmentSFTPHost               = "attachment.sftp.host"
	NameAttachmentSFTPPort               = "attachment.sftp.port"
	NameAttachmentSFTPUsername           = "attachment.sftp.username"
	NameAttachmentSFTPPassword           = "attachment.sftp.password"
	NameAttachmentSFTPPrivateKey         = "attachment.sftp.private_key"
	NameAttachmentSFTPPassphrase         = "attachment.sftp.passphrase"
	NameAttachmentSFTPRootPath           = "attachment.sftp.root_path"
	NameAttachmentSFTPHostKeyFingerprint = "attachment.sftp.host_key_fingerprint"
	NameAttachmentSFTPPublicBaseURL      = "attachment.sftp.public_base_url"

	// 头像运行时选项。统一头像策略：用户上传优先，否则按 default_provider 回退。
	NameAvatarAllowUpload       = "avatar.allow_upload"
	NameAvatarDefaultProvider   = "avatar.default_provider"
	NameAvatarGravatarBaseURL   = "avatar.gravatar_base_url"
	NameAvatarDefaultStaticURL  = "avatar.default_static_url"
	NameAvatarMaxSizeKB         = "avatar.max_size_kb"
	NameAvatarMaxDimension      = "avatar.max_dimension"
	NameAvatarAllowGIF          = "avatar.allow_gif"
	NameAvatarCompressEnabled   = "avatar.compress_enabled"
	NameAvatarTargetDimension   = "avatar.target_dimension"
	NameAvatarCompressQuality   = "avatar.compress_quality"

	// 邮件运行时选项。
	NameMailProvider       = "mail.provider"
	NameMailFromAddress    = "mail.from_address"
	NameMailFromName       = "mail.from_name"
	NameMailSMTPHost       = "mail.smtp.host"
	NameMailSMTPPort       = "mail.smtp.port"
	NameMailSMTPUsername   = "mail.smtp.username"
	NameMailSMTPPassword   = "mail.smtp.password"
	NameMailSMTPEncryption = "mail.smtp.encryption"

	CodeInvalid = "options.invalid"
)

var ErrInvalidOption = errors.New("options: invalid option")

type Option struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

type AdminOption struct {
	Name      string `json:"name"`
	Value     string `json:"value"`
	Public    bool   `json:"public"`
	Secret    bool   `json:"secret"`
	SecretSet bool   `json:"secretSet"`
}

type UpdateInput struct {
	Name  string
	Value string
}
