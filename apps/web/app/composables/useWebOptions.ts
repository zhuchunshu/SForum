import type { ApiEnvelope } from '~/composables/useApiClient'
// TopicUrlMode 的权威定义在 forumTaxonomy（纯工具模块）；此处复用以避免
// Nuxt auto-import 的重复类型声明警告。
import type { TopicUrlMode } from '~/utils/forumTaxonomy'
import type { SEOContentPolicy, SEOPageType, SEOResolverSettings } from '~/utils/seoResolver'

export type WebOption = {
  name: string
  value: string
}

export type AdminWebOption = WebOption & {
  public: boolean
  secret: boolean
  secretSet: boolean
}

export type AppearanceThemePreset = 'pine_teal' | 'ocean_blue' | 'violet' | 'rose' | 'amber'
export type AppearanceTheme = AppearanceThemePreset | `custom:${string}`
export type ResolvedAppearanceTheme = {
  theme: AppearanceTheme
  dataTheme: AppearanceThemePreset | 'custom'
  customColor: string
  cssVars: Record<string, string>
  style: string
}
export type FooterLocale = 'zh-CN' | 'en-US'
export type FooterLinkKey = 'terms' | 'privacy' | 'guidelines'
export type HumanVerificationScenario = 'register' | 'password_reset' | 'login_risk' | 'post_risk'
export type AltchaWidgetType = 'native' | 'checkbox' | 'switch'
export type AltchaWidgetAuto = 'off' | 'onfocus' | 'onload' | 'onsubmit'
export type AltchaWidgetDisplay = 'standard' | 'bar' | 'floating' | 'overlay' | 'invisible'
export type AltchaWidgetSettings = {
  type: AltchaWidgetType
  auto: AltchaWidgetAuto
  display: AltchaWidgetDisplay
  hideLogo: boolean
  hideFooter: boolean
  workers: number
  minDuration: number
}
export type PasswordPolicy = {
  minLength: number
  maxLength: number
  requireLowercase: boolean
  requireUppercase: boolean
  requireNumber: boolean
  requireSymbol: boolean
}
export type PasswordRequirement = {
  key: 'length' | 'lowercase' | 'uppercase' | 'number' | 'symbol'
  met: boolean
}
export type SEOTwitterCard = 'summary' | 'summary_large_image'
export type AvatarDefaultProvider = 'initials' | 'gravatar' | 'static'
export type AvatarGravatarHashAlgorithm = 'sha256' | 'md5'
export type AvatarSettings = {
  allowUpload: boolean
  defaultProvider: AvatarDefaultProvider
  gravatarBaseUrl: string
  gravatarHashAlgorithm: AvatarGravatarHashAlgorithm
  defaultStaticUrl: string
  maxSizeKb: number
  maxDimension: number
  allowGif: boolean
  compressEnabled: boolean
  targetDimension: number
  compressQuality: number
}

// 帖子详情页 URL 形态枚举与推荐默认值。TopicUrlMode 类型复用自 forumTaxonomy。
export const topicUrlModes: TopicUrlMode[] = ['id_slug', 'id', 'slug']
export const recommendedTopicUrlMode: TopicUrlMode = 'id_slug'
export const avatarDefaultProviders: AvatarDefaultProvider[] = ['initials', 'gravatar', 'static']
export const avatarHashAlgorithms: AvatarGravatarHashAlgorithm[] = ['sha256', 'md5']
export const recommendedAvatarSettings: AvatarSettings = {
  allowUpload: true,
  defaultProvider: 'initials',
  gravatarBaseUrl: 'https://gravatar.com/avatar/',
  gravatarHashAlgorithm: 'sha256',
  defaultStaticUrl: '',
  maxSizeKb: 2048,
  maxDimension: 2048,
  allowGif: false,
  compressEnabled: true,
  targetDimension: 256,
  compressQuality: 85
}

export type SEOSettings = SEOResolverSettings & {
  metaTitleTemplate: string
  metaDescription: string
  metaKeywords: string
  ogImageUrl: string
  twitterCard: SEOTwitterCard
  twitterSite: string
  allowIndexing: boolean
  googleVerification: string
  bingVerification: string
  baiduVerification: string
  yandexVerification: string
  robotsExtraAllow: string[]
  robotsExtraDisallow: string[]
  blockAiBots: boolean
  blockNonSeoBots: boolean
  sitemapEnabled: boolean
  sitemapIncludeStaticPages: boolean
  sitemapIncludeForumContent: boolean
  schemaOrgEnabled: boolean
  schemaOrgSearchActionEnabled: boolean
  schemaOrgDiscussionEnabled: boolean
  schemaOrgOrganizationLogoUrl: string
  topicUrlMode: TopicUrlMode
}

export type FooterLinkOption = {
  key: FooterLinkKey
  labels: Record<FooterLocale, string>
  url: string
}

type RefreshOptions = {
  timeout?: number
}

export const appearanceThemes: AppearanceThemePreset[] = ['pine_teal', 'ocean_blue', 'violet', 'rose', 'amber']
export const recommendedAppearanceTheme: AppearanceThemePreset = 'pine_teal'
export const defaultCustomThemeColor = '#2563eb'
export const humanVerificationScenarios: HumanVerificationScenario[] = ['register', 'password_reset', 'login_risk', 'post_risk']
export const altchaWidgetTypes: AltchaWidgetType[] = ['native', 'checkbox', 'switch']
export const altchaWidgetAutoModes: AltchaWidgetAuto[] = ['off', 'onfocus', 'onload', 'onsubmit']
export const altchaWidgetDisplays: AltchaWidgetDisplay[] = ['standard', 'bar', 'floating', 'overlay', 'invisible']
export const recommendedPasswordPolicy: PasswordPolicy = {
  minLength: 12,
  maxLength: 128,
  requireLowercase: false,
  requireUppercase: false,
  requireNumber: false,
  requireSymbol: false
}

const customThemePrefix = 'custom:'
const enabledOption = 'enabled'
const disabledOption = 'disabled'
const humanVerificationScenarioDefaults: Record<HumanVerificationScenario, boolean> = {
  register: true,
  password_reset: false,
  login_risk: false,
  post_risk: false
}

export const recommendedFooterCopyright: Record<FooterLocale, string> = {
  'zh-CN': '© {year} {siteName}。保留所有权利。',
  'en-US': '© {year} {siteName}. All rights reserved.'
}

export const recommendedFooterLinks: FooterLinkOption[] = [
  {
    key: 'terms',
    labels: { 'zh-CN': '服务条款', 'en-US': 'Terms of Service' },
    url: '#'
  },
  {
    key: 'privacy',
    labels: { 'zh-CN': '隐私政策', 'en-US': 'Privacy Policy' },
    url: '#'
  },
  {
    key: 'guidelines',
    labels: { 'zh-CN': '社区指南', 'en-US': 'Guidelines' },
    url: '#'
  }
]

const fallbackOptions: Record<string, string> = {
  'site.name': 'SForum',
  'site.url': 'http://127.0.0.1:3000',
  // 站点副标题（可空）；管理邮箱为 admin-only，不在 public fallback 中暴露。
  'site.tagline': '',
  // Wave 2 品牌资源：空 → 主题默认图标；附件 ID 与 URL 二选一或并存（前台优先附件解析）。
  'site.logo_url': '',
  'site.logo_attachment_id': '',
  'site.favicon_url': '',
  'site.favicon_attachment_id': '',
  'site.apple_touch_icon_url': '',
  'site.apple_touch_icon_attachment_id': '',
  // 法律页 Markdown stubs（与后端 recommended 文案对齐的最小占位；完整 stub 由 API EnsureDefaults 写入）。
  'legal.terms.body.zh-CN': '## 服务条款\n\n欢迎使用本社区。',
  'legal.terms.body.en-US': '## Terms of Service\n\nWelcome to this community.',
  'legal.privacy.body.zh-CN': '## 隐私政策\n\n我们仅收集运营社区所必需的信息。',
  'legal.privacy.body.en-US': '## Privacy Policy\n\nWe collect only what is needed to run the community.',
  'legal.guidelines.body.zh-CN': '## 社区指南\n\n请保持友善、就事论事。',
  'legal.guidelines.body.en-US': '## Community Guidelines\n\nBe kind and constructive.',
  'site.default_locale': 'zh-CN',
  'site.supported_locales': 'zh-CN,en-US',
  // 站点展示时区与日期时间格式（与后端 recommended 默认对齐）。
  'site.timezone': 'UTC',
  'site.date_format': 'Y-m-d',
  'site.time_format': 'H:i',
  'site.start_of_week': '1',
  'human_verification.provider': 'disabled',
  'human_verification.scenarios.register': enabledOption,
  'human_verification.scenarios.password_reset': disabledOption,
  'human_verification.scenarios.login_risk': disabledOption,
  'human_verification.scenarios.post_risk': disabledOption,
  'human_verification.altcha.widget.type': 'checkbox',
  'human_verification.altcha.widget.auto': 'off',
  'human_verification.altcha.widget.display': 'standard',
  'human_verification.altcha.widget.hide_logo': enabledOption,
  'human_verification.altcha.widget.hide_footer': enabledOption,
  'human_verification.altcha.widget.workers': '2',
  'human_verification.altcha.widget.min_duration_ms': '500',
  'appearance.theme': recommendedAppearanceTheme,
  'footer.copyright.zh-CN': recommendedFooterCopyright['zh-CN'],
  'footer.copyright.en-US': recommendedFooterCopyright['en-US'],
  'footer.links': JSON.stringify(recommendedFooterLinks),
  'identity.password.min_length': String(recommendedPasswordPolicy.minLength),
  'identity.password.max_length': String(recommendedPasswordPolicy.maxLength),
  'identity.password.require_lowercase': disabledOption,
  'identity.password.require_uppercase': disabledOption,
  'identity.password.require_number': disabledOption,
  'identity.password.require_symbol': disabledOption,
  'identity.registration.enabled': enabledOption,
  'forum.default_category_slug': 'general',
  'forum.tags.creation_mode': 'controlled',
  'forum.tags.public_pages': enabledOption,
  'forum.tags.min_per_topic': '0',
  'forum.tags.max_per_topic': '5',
  'forum.pagination.topics_per_page': '20',
  'forum.pagination.comments_per_page': '20',
  'forum.topics.title_min_runes': '2',
  'forum.topics.title_max_runes': '100',
  'forum.topics.content_min_runes': '0',
  'forum.topics.content_max_runes': '50000',
  'forum.topics.edit_window_minutes': '0',
  'forum.topics.cooldown_seconds': '0',
  'forum.topics.daily_limit': '0',
  'forum.comments.min_runes': '1',
  'forum.comments.max_runes': '10000',
  'forum.comments.max_nesting_depth': '5',
  'forum.comments.edit_window_minutes': '0',
  'forum.comments.cooldown_seconds': '0',
  'forum.comments.daily_limit': '0',
  'forum.reading.excerpt_rune_limit': '180',
  'seo.meta_title_template': '',
  'seo.meta_description': '',
  'seo.meta_keywords': '',
  'seo.og_image_url': '',
  'seo.twitter_card': 'summary_large_image',
  'seo.twitter_site': '',
  'seo.allow_indexing': 'enabled',
  'seo.google_verification': '',
  'seo.bing_verification': '',
  'seo.baidu_verification': '',
  'seo.yandex_verification': '',
  'seo.robots.extra_allow': '',
  'seo.robots.extra_disallow': '',
  'seo.robots.block_ai_bots': disabledOption,
  'seo.robots.block_non_seo_bots': disabledOption,
  'seo.sitemap.enabled': enabledOption,
  'seo.sitemap.include_static_pages': enabledOption,
  'seo.sitemap.include_forum_content': enabledOption,
  'seo.schema_org.enabled': enabledOption,
  'seo.schema_org.search_action_enabled': enabledOption,
  'seo.schema_org.discussion_enabled': enabledOption,
  'seo.schema_org.organization_logo_url': '',
  'seo.topic_url_mode': recommendedTopicUrlMode,
  'seo.site.inherit_site_name': enabledOption,
  'seo.site.name': '',
  'seo.home.title': '',
  'seo.home.description': '',
  'seo.home.keywords': '',
  'seo.home.og_title': '',
  'seo.home.og_description': '',
  'seo.home.og_image_url': '',
  'seo.page.title_template': '{pageTitle} | {seoSiteName}',
  'seo.page.default_description': '',
  'seo.page.title_separator': '|',
  'avatar.allow_upload': enabledOption,
  'avatar.default_provider': recommendedAvatarSettings.defaultProvider,
  'avatar.gravatar_base_url': recommendedAvatarSettings.gravatarBaseUrl,
  'avatar.gravatar_hash_algorithm': recommendedAvatarSettings.gravatarHashAlgorithm,
  'avatar.default_static_url': recommendedAvatarSettings.defaultStaticUrl,
  'avatar.max_size_kb': String(recommendedAvatarSettings.maxSizeKb),
  'avatar.max_dimension': String(recommendedAvatarSettings.maxDimension),
  'avatar.allow_gif': disabledOption,
  'avatar.compress_enabled': enabledOption,
  'avatar.target_dimension': String(recommendedAvatarSettings.targetDimension),
  'avatar.compress_quality': String(recommendedAvatarSettings.compressQuality)
}

export const useWebOptions = () => {
  const { apiBaseUrl, apiHeaders, request } = useApiClient()
  const options = useState<Record<string, string>>('web-options', () => ({ ...fallbackOptions }))

  async function refresh(requestOptions: RefreshOptions = {}) {
    const items = await request<WebOption[]>('/web-options', {
      timeout: requestOptions.timeout
    })
    options.value = {
      ...fallbackOptions,
      ...Object.fromEntries(items.map((item) => [item.name, item.value]))
    }
    return options.value
  }

  async function save(name: string, value: string) {
    const item = await request<WebOption>('/web-options', {
      method: 'PUT',
      body: { name, value }
    })
    options.value = {
      ...options.value,
      [item.name]: item.value
    }
    return item
  }

  async function fetchEnvelope() {
    return await $fetch<ApiEnvelope<WebOption[]>>(`${apiBaseUrl}/web-options`, {
      credentials: 'include',
      headers: apiHeaders()
    })
  }

  async function fetchAdminEnvelope() {
    return await $fetch<ApiEnvelope<AdminWebOption[]>>(`${apiBaseUrl}/admin/web-options`, {
      credentials: 'include',
      headers: apiHeaders()
    })
  }

  async function saveMany(items: WebOption[]) {
    const adminItems = await request<AdminWebOption[]>('/admin/web-options', {
      method: 'PUT',
      body: { options: items }
    })

    const publicItems = adminItems.filter((item) => item.public && !item.secret)
    options.value = {
      ...options.value,
      ...Object.fromEntries(publicItems.map((item) => [item.name, item.value]))
    }
    return adminItems
  }

  function webOption(name: string, fallback = '') {
    return options.value[name] ?? fallbackOptions[name] ?? fallback
  }

  const siteName = computed(() => webOption('site.name', 'SForum'))
  const siteUrl = computed(() => webOption('site.url', 'http://127.0.0.1:3000'))
  const siteTagline = computed(() => webOption('site.tagline', '').trim())
  // 品牌资源：URL 优先给简单场景；附件 ID 留给后续解析附件公开地址。
  const siteLogoUrl = computed(() => webOption('site.logo_url', '').trim())
  const siteLogoAttachmentId = computed(() => webOption('site.logo_attachment_id', '').trim())
  const siteFaviconUrl = computed(() => webOption('site.favicon_url', '').trim())
  const siteFaviconAttachmentId = computed(() => webOption('site.favicon_attachment_id', '').trim())
  const siteAppleTouchIconUrl = computed(() => webOption('site.apple_touch_icon_url', '').trim())
  const siteAppleTouchIconAttachmentId = computed(() => webOption('site.apple_touch_icon_attachment_id', '').trim())
  const defaultLocale = computed(() => webOption('site.default_locale', 'zh-CN'))
  const supportedLocales = computed(() => parseSupportedLocales(webOption('site.supported_locales', 'zh-CN,en-US')))
  const siteTimezone = computed(() => webOption('site.timezone', 'UTC'))
  const siteDateFormat = computed(() => webOption('site.date_format', 'Y-m-d'))
  const siteTimeFormat = computed(() => webOption('site.time_format', 'H:i'))
  const siteStartOfWeek = computed(() => {
    const n = Number.parseInt(webOption('site.start_of_week', '1'), 10)
    return Number.isFinite(n) && n >= 0 && n <= 6 ? n : 1
  })
  const appearanceTheme = computed(() => parseAppearanceTheme(webOption('appearance.theme', recommendedAppearanceTheme)))
  const resolvedAppearanceTheme = computed(() => resolveAppearanceTheme(appearanceTheme.value))
  const footerCopyright = computed<Record<FooterLocale, string>>(() => ({
    'zh-CN': webOption('footer.copyright.zh-CN', fallbackOptions['footer.copyright.zh-CN']),
    'en-US': webOption('footer.copyright.en-US', fallbackOptions['footer.copyright.en-US'])
  }))
  const footerLinks = computed(() => parseFooterLinks(webOption('footer.links', fallbackOptions['footer.links'])))
  const humanVerificationProvider = computed(() => {
    const provider = webOption('human_verification.provider', 'disabled').trim().toLowerCase()
    return provider === 'altcha' ? 'altcha' : 'disabled'
  })
  const humanVerificationScenarioSettings = computed<Record<HumanVerificationScenario, boolean>>(() => {
    return Object.fromEntries(
      humanVerificationScenarios.map((scenario) => [
        scenario,
        normalizeEnabledOption(
          webOption(humanVerificationScenarioOptionName(scenario)),
          humanVerificationScenarioDefaults[scenario]
        )
      ])
    ) as Record<HumanVerificationScenario, boolean>
  })
  const altchaWidgetSettings = computed(() => resolveAltchaWidgetSettings(options.value))
  const passwordPolicy = computed(() => resolvePasswordPolicy(options.value))
  // 开放注册运营意图（不含 bootstrap 覆盖；权威仍看 /auth/registration-status）。
  const registrationEnabled = computed(() => normalizeEnabledOption(webOption('identity.registration.enabled'), true))
  const seoSettings = computed(() => resolveSEOSettings(options.value))
  const avatarSettings = computed(() => resolveAvatarSettings(options.value))
  const seoIndexable = computed(() => isSEOIndexingAllowed(seoSettings.value, siteUrl.value))
  // 帖子 URL 形态：列表/详情/SEO 链接生成均依赖此值。
  const topicUrlMode = computed<TopicUrlMode>(() => seoSettings.value.topicUrlMode)

  function humanVerificationEnabledFor(scenario: HumanVerificationScenario) {
    return humanVerificationProvider.value === 'altcha' && humanVerificationScenarioSettings.value[scenario]
  }

  function footerCopyrightTemplate(localeCode: string) {
    return footerCopyright.value[normalizeFooterLocale(localeCode)]
  }

  function footerLinkLabel(link: FooterLinkOption, localeCode: string) {
    return link.labels[normalizeFooterLocale(localeCode)] || link.labels['zh-CN']
  }

  /** 法律页正文：按 key + 当前/指定 locale 读取 Markdown。 */
  function legalBody(key: 'terms' | 'privacy' | 'guidelines', localeCode?: string) {
    const locale = normalizeFooterLocale(localeCode || defaultLocale.value)
    const name = `legal.${key}.body.${locale}`
    const fallbackName = `legal.${key}.body.zh-CN`
    return webOption(name, webOption(fallbackName, '')).trim()
  }

  return {
    options,
    siteName,
    siteUrl,
    siteTagline,
    siteLogoUrl,
    siteLogoAttachmentId,
    siteFaviconUrl,
    siteFaviconAttachmentId,
    siteAppleTouchIconUrl,
    siteAppleTouchIconAttachmentId,
    defaultLocale,
    supportedLocales,
    siteTimezone,
    siteDateFormat,
    siteTimeFormat,
    siteStartOfWeek,
    appearanceTheme,
    resolvedAppearanceTheme,
    footerCopyright,
    footerLinks,
    humanVerificationProvider,
    humanVerificationScenarioSettings,
    altchaWidgetSettings,
    passwordPolicy,
    registrationEnabled,
    seoSettings,
    avatarSettings,
    seoIndexable,
    topicUrlMode,
    humanVerificationEnabledFor,
    webOption,
    footerCopyrightTemplate,
    footerLinkLabel,
    legalBody,
    refresh,
    save,
    saveMany,
    fetchEnvelope,
    fetchAdminEnvelope
  }
}

function parseSupportedLocales(value: string) {
  const locales = value.split(',').map((item) => item.trim()).filter(Boolean)
  return locales.length > 0 ? locales : ['zh-CN', 'en-US']
}

function parseAppearanceTheme(value: string): AppearanceTheme {
  return normalizeAppearanceThemeValue(value)
}

function parseFooterLinks(value: string): FooterLinkOption[] {
  try {
    const parsed = JSON.parse(value)
    if (!Array.isArray(parsed)) {
      return cloneFooterLinks(recommendedFooterLinks)
    }

    const normalized = parsed
      .map(normalizeFooterLink)
      .filter((item): item is FooterLinkOption => item !== null)
    if (normalized.length !== recommendedFooterLinks.length) {
      return cloneFooterLinks(recommendedFooterLinks)
    }

    const byKey = new Map(normalized.map((item) => [item.key, item]))
    return recommendedFooterLinks.map((item) => byKey.get(item.key) || cloneFooterLink(item))
  } catch {
    return cloneFooterLinks(recommendedFooterLinks)
  }
}

export function cloneFooterLinks(links: FooterLinkOption[]) {
  return links.map(cloneFooterLink)
}

export function cloneFooterLink(link: FooterLinkOption): FooterLinkOption {
  return {
    key: link.key,
    labels: { ...link.labels },
    url: link.url
  }
}

function normalizeFooterLink(value: unknown): FooterLinkOption | null {
  if (!value || typeof value !== 'object') {
    return null
  }

  const item = value as Partial<FooterLinkOption>
  if (!isFooterLinkKey(item.key) || !item.labels || typeof item.labels !== 'object') {
    return null
  }

  const labels = item.labels as Partial<Record<FooterLocale, unknown>>
  const zhCN = typeof labels['zh-CN'] === 'string' ? labels['zh-CN'].trim() : ''
  const enUS = typeof labels['en-US'] === 'string' ? labels['en-US'].trim() : ''
  const url = typeof item.url === 'string' ? item.url.trim() : ''
  if (!zhCN || !enUS) {
    return null
  }

  return {
    key: item.key,
    labels: {
      'zh-CN': zhCN,
      'en-US': enUS
    },
    url
  }
}

function isFooterLinkKey(value: unknown): value is FooterLinkKey {
  return value === 'terms' || value === 'privacy' || value === 'guidelines'
}

function normalizeFooterLocale(localeCode: string): FooterLocale {
  return localeCode.toLowerCase().startsWith('en') ? 'en-US' : 'zh-CN'
}

export function resolveSEOSettings(values: Record<string, string>): SEOSettings {
  const option = (name: string) => values[name] ?? fallbackOptions[name] ?? ''
  const siteName = option('site.name').trim() || 'SForum'
  const inheritSiteName = normalizeEnabledOption(option('seo.site.inherit_site_name'), true)
  const seoSiteName = inheritSiteName ? siteName : option('seo.site.name').trim() || siteName
  const pageDefaultDescription = option('seo.page.default_description').trim() || option('seo.meta_description').trim()
  const policies = Object.fromEntries(
    (['home', 'category', 'tag', 'topic', 'profile', 'static'] as SEOPageType[]).map(type => [type, resolveSEOContentPolicy(type, option)])
  ) as Record<SEOPageType, SEOContentPolicy>
  return {
    siteName,
    siteUrl: option('site.url').trim() || 'http://127.0.0.1:3000',
    seoSiteName,
    homeTitle: option('seo.home.title').trim() || seoSiteName,
    homeDescription: option('seo.home.description').trim() || option('seo.meta_description').trim() || seoSiteName,
    homeKeywords: option('seo.home.keywords').trim() || option('seo.meta_keywords').trim(),
    homeOGTitle: option('seo.home.og_title').trim(),
    homeOGDescription: option('seo.home.og_description').trim(),
    homeOGImageUrl: option('seo.home.og_image_url').trim() || option('seo.og_image_url').trim(),
    pageTitleTemplate: option('seo.page.title_template').trim() || option('seo.meta_title_template').trim() || '{pageTitle} | {seoSiteName}',
    pageDefaultDescription,
    policies,
    metaTitleTemplate: option('seo.meta_title_template').trim(),
    metaDescription: option('seo.meta_description').trim(),
    metaKeywords: option('seo.meta_keywords').trim(),
    ogImageUrl: option('seo.og_image_url').trim(),
    twitterCard: normalizeSEOTwitterCard(option('seo.twitter_card')),
    twitterSite: normalizeSEOTwitterSite(option('seo.twitter_site')),
    allowIndexing: normalizeEnabledOption(option('seo.allow_indexing'), true),
    googleVerification: normalizeSEOVerificationToken(option('seo.google_verification')),
    bingVerification: normalizeSEOVerificationToken(option('seo.bing_verification')),
    baiduVerification: normalizeSEOVerificationToken(option('seo.baidu_verification')),
    yandexVerification: normalizeSEOVerificationToken(option('seo.yandex_verification')),
    robotsExtraAllow: parseSEORobotsPathList(option('seo.robots.extra_allow')),
    robotsExtraDisallow: parseSEORobotsPathList(option('seo.robots.extra_disallow')),
    blockAiBots: normalizeEnabledOption(option('seo.robots.block_ai_bots')),
    blockNonSeoBots: normalizeEnabledOption(option('seo.robots.block_non_seo_bots')),
    sitemapEnabled: normalizeEnabledOption(option('seo.sitemap.enabled'), true),
    sitemapIncludeStaticPages: normalizeEnabledOption(option('seo.sitemap.include_static_pages'), true),
    sitemapIncludeForumContent: normalizeEnabledOption(option('seo.sitemap.include_forum_content'), true),
    schemaOrgEnabled: normalizeEnabledOption(option('seo.schema_org.enabled'), true),
    schemaOrgSearchActionEnabled: normalizeEnabledOption(option('seo.schema_org.search_action_enabled'), true),
    schemaOrgDiscussionEnabled: normalizeEnabledOption(option('seo.schema_org.discussion_enabled'), true),
    schemaOrgOrganizationLogoUrl: option('seo.schema_org.organization_logo_url').trim(),
    topicUrlMode: normalizeTopicUrlMode(option('seo.topic_url_mode'))
  }
}

function resolveSEOContentPolicy(type: SEOPageType, option: (name: string) => string): SEOContentPolicy {
  const defaults: Record<SEOPageType, SEOContentPolicy> = {
    home: { titleTemplate: '{seoSiteName}', descriptionSources: ['content', 'site_default'], defaultImageUrl: '', indexMode: 'index', includeInSitemap: true, schemaType: 'WebSite' },
    category: { titleTemplate: '{categoryName} | {seoSiteName}', descriptionSources: ['category_description', 'site_default'], defaultImageUrl: '', indexMode: 'index', includeInSitemap: true, schemaType: 'CollectionPage' },
    tag: { titleTemplate: '{tagName} | {seoSiteName}', descriptionSources: ['tag_description', 'site_default'], defaultImageUrl: '', indexMode: 'index', includeInSitemap: true, schemaType: 'CollectionPage' },
    topic: { titleTemplate: '{topicTitle} | {seoSiteName}', descriptionSources: ['topic_summary', 'topic_excerpt', 'site_default'], defaultImageUrl: '', indexMode: 'index', includeInSitemap: true, schemaType: 'DiscussionForumPosting' },
    profile: { titleTemplate: '{authorName} | {seoSiteName}', descriptionSources: ['profile_bio', 'site_default'], defaultImageUrl: '', indexMode: 'noindex', includeInSitemap: false, schemaType: 'ProfilePage' },
    static: { titleTemplate: '{pageTitle} | {seoSiteName}', descriptionSources: ['page_description', 'site_default'], defaultImageUrl: '', indexMode: 'index', includeInSitemap: true, schemaType: 'WebPage' }
  }
  if (type === 'home') {
    return defaults.home
  }
  const prefix = `seo.content_type.${type}`
  const fallback = defaults[type]
  const schemaValue = option(`${prefix}.schema_type`).trim() as SEOContentPolicy['schemaType']
  const indexValue = option(`${prefix}.index_mode`).trim().toLowerCase()
  const schemaTypes: SEOContentPolicy['schemaType'][] = ['WebSite', 'CollectionPage', 'DiscussionForumPosting', 'ProfilePage', 'WebPage']
  return {
    titleTemplate: option(`${prefix}.title_template`).trim() || fallback.titleTemplate,
    descriptionSources: option(`${prefix}.description_source`).split(',').map(value => value.trim()).filter(Boolean).length
      ? option(`${prefix}.description_source`).split(',').map(value => value.trim()).filter(Boolean)
      : fallback.descriptionSources,
    defaultImageUrl: option(`${prefix}.default_image_url`).trim(),
    indexMode: indexValue === 'index' || indexValue === 'noindex' ? indexValue : fallback.indexMode,
    includeInSitemap: normalizeEnabledOption(option(`${prefix}.include_in_sitemap`), fallback.includeInSitemap),
    schemaType: schemaTypes.includes(schemaValue) ? schemaValue : fallback.schemaType
  }
}

// normalizeTopicUrlMode 将存储值归一为合法枚举，非法值回退默认 id_slug。
export function normalizeTopicUrlMode(value: string | undefined): TopicUrlMode {
  const raw = value?.trim().toLowerCase() as TopicUrlMode
  return topicUrlModes.includes(raw) ? raw : recommendedTopicUrlMode
}

export function resolveAvatarSettings(values: Record<string, string>): AvatarSettings {
  const option = (name: string) => values[name] ?? fallbackOptions[name] ?? ''
  return {
    allowUpload: normalizeEnabledOption(option('avatar.allow_upload'), recommendedAvatarSettings.allowUpload),
    defaultProvider: normalizeAvatarDefaultProvider(option('avatar.default_provider')),
    gravatarBaseUrl: normalizeAvatarBaseUrl(option('avatar.gravatar_base_url')),
    gravatarHashAlgorithm: normalizeAvatarHashAlgorithm(option('avatar.gravatar_hash_algorithm')),
    defaultStaticUrl: option('avatar.default_static_url').trim(),
    maxSizeKb: normalizeBoundedInteger(option('avatar.max_size_kb'), recommendedAvatarSettings.maxSizeKb, 1, 10240),
    maxDimension: normalizeBoundedInteger(option('avatar.max_dimension'), recommendedAvatarSettings.maxDimension, 32, 4096),
    allowGif: normalizeEnabledOption(option('avatar.allow_gif'), recommendedAvatarSettings.allowGif),
    compressEnabled: normalizeEnabledOption(option('avatar.compress_enabled'), recommendedAvatarSettings.compressEnabled),
    targetDimension: normalizeBoundedInteger(option('avatar.target_dimension'), recommendedAvatarSettings.targetDimension, 32, 4096),
    compressQuality: normalizeBoundedInteger(option('avatar.compress_quality'), recommendedAvatarSettings.compressQuality, 1, 100)
  }
}

export function normalizeAvatarDefaultProvider(value: string | undefined): AvatarDefaultProvider {
  const raw = value?.trim().toLowerCase() as AvatarDefaultProvider
  return avatarDefaultProviders.includes(raw) ? raw : recommendedAvatarSettings.defaultProvider
}

export function normalizeAvatarHashAlgorithm(value: string | undefined): AvatarGravatarHashAlgorithm {
  const raw = value?.trim().toLowerCase() as AvatarGravatarHashAlgorithm
  return avatarHashAlgorithms.includes(raw) ? raw : recommendedAvatarSettings.gravatarHashAlgorithm
}

export function normalizeAvatarBaseUrl(value: string | undefined) {
  const raw = value?.trim() || recommendedAvatarSettings.gravatarBaseUrl
  try {
    const url = new URL(raw)
    if (url.protocol !== 'http:' && url.protocol !== 'https:') {
      return recommendedAvatarSettings.gravatarBaseUrl
    }
    return raw.replace(/\/+$/, '') + '/'
  } catch {
    return recommendedAvatarSettings.gravatarBaseUrl
  }
}

export function normalizeSEOTwitterCard(value: string | undefined): SEOTwitterCard {
  return value?.trim().toLowerCase() === 'summary' ? 'summary' : 'summary_large_image'
}

export function normalizeSEOTwitterSite(value: string | undefined) {
  const raw = value?.trim() || ''
  if (!raw) {
    return ''
  }
  const withoutAt = raw.replace(/^@+/, '')
  return withoutAt ? `@${withoutAt}` : ''
}

export function normalizeSEOVerificationToken(value: string | undefined) {
  const raw = value?.trim() || ''
  if (!raw || raw.length > 120) {
    return ''
  }
  return /[\s<>"']/.test(raw) ? '' : raw
}

export function parseSEORobotsPathList(value: string | undefined) {
  const seen = new Set<string>()
  return (value || '')
    .split(/\r?\n/)
    .map((item) => item.trim())
    .filter((item) => {
      if (!item || !item.startsWith('/') || item.startsWith('//') || /[\s<>"']/.test(item) || seen.has(item)) {
        return false
      }
      seen.add(item)
      return true
    })
}

export function isLocalSiteUrl(value: string | undefined) {
  try {
    const url = new URL(value || '')
    return ['localhost', '127.0.0.1', '0.0.0.0', '::1'].includes(url.hostname)
  } catch {
    return true
  }
}

export function isSEOIndexingAllowed(settings: SEOSettings, siteUrl: string) {
  return settings.allowIndexing && !isLocalSiteUrl(siteUrl)
}

export function applySEOTitleTemplate(title: string, template: string, siteName: string) {
  const cleanTitle = title.trim()
  const cleanSiteName = siteName.trim() || 'SForum'
  const cleanTemplate = template.trim()
  if (!cleanTitle) {
    return cleanSiteName
  }
  if (!cleanTemplate) {
    return `${cleanTitle} - ${cleanSiteName}`
  }
  return cleanTemplate
    .replaceAll('{title}', cleanTitle)
    .replaceAll('{siteName}', cleanSiteName)
}

export function normalizeAppearanceThemeValue(value: string | undefined): AppearanceTheme {
  const raw = value?.trim().toLowerCase() || ''
  if (isAppearanceThemePreset(raw)) {
    return raw
  }

  if (raw.startsWith(customThemePrefix)) {
    const color = normalizeHexColor(raw.slice(customThemePrefix.length))
    return color ? (`${customThemePrefix}${color}` as AppearanceTheme) : recommendedAppearanceTheme
  }

  const color = normalizeHexColor(raw)
  return color ? (`${customThemePrefix}${color}` as AppearanceTheme) : recommendedAppearanceTheme
}

export function isAppearanceThemePreset(value: string): value is AppearanceThemePreset {
  return appearanceThemes.includes(value as AppearanceThemePreset)
}

export function buildCustomAppearanceThemeValue(color: string): AppearanceTheme {
  return `${customThemePrefix}${normalizeHexColor(color) || defaultCustomThemeColor}` as AppearanceTheme
}

export function customColorFromAppearanceTheme(value: string): string | null {
  const normalized = normalizeAppearanceThemeValue(value)
  if (!normalized.startsWith(customThemePrefix)) {
    return null
  }
  return normalizeHexColor(normalized.slice(customThemePrefix.length))
}

export function resolveAppearanceTheme(value: string): ResolvedAppearanceTheme {
  const theme = normalizeAppearanceThemeValue(value)
  const customColor = customColorFromAppearanceTheme(theme) || defaultCustomThemeColor

  if (isAppearanceThemePreset(theme)) {
    return {
      theme,
      dataTheme: theme,
      customColor,
      cssVars: {},
      style: ''
    }
  }

  const cssVars = buildCustomThemeVars(customColor)
  return {
    theme,
    dataTheme: 'custom',
    customColor,
    cssVars,
    style: cssVarsToStyle(cssVars)
  }
}

export function normalizeHexColor(value: string | undefined): string | null {
  const raw = value?.trim().toLowerCase().replace(/^#/, '') || ''
  return /^[0-9a-f]{6}$/.test(raw) ? `#${raw}` : null
}

export function humanVerificationScenarioOptionName(scenario: HumanVerificationScenario) {
  return `human_verification.scenarios.${scenario}`
}

export function resolveAltchaWidgetSettings(values: Record<string, string>): AltchaWidgetSettings {
  const option = (name: string) => values[name] ?? fallbackOptions[name] ?? ''
  return {
    type: normalizeAltchaWidgetType(option('human_verification.altcha.widget.type')),
    auto: normalizeAltchaWidgetAuto(option('human_verification.altcha.widget.auto')),
    display: normalizeAltchaWidgetDisplay(option('human_verification.altcha.widget.display')),
    hideLogo: normalizeEnabledOption(option('human_verification.altcha.widget.hide_logo'), true),
    hideFooter: normalizeEnabledOption(option('human_verification.altcha.widget.hide_footer'), true),
    workers: normalizeBoundedInteger(option('human_verification.altcha.widget.workers'), 2, 1, 16),
    minDuration: normalizeBoundedInteger(option('human_verification.altcha.widget.min_duration_ms'), 500, 0, 10000)
  }
}

export function resolvePasswordPolicy(values: Record<string, string>): PasswordPolicy {
  const option = (name: string) => values[name] ?? fallbackOptions[name] ?? ''
  const minLength = normalizeBoundedInteger(option('identity.password.min_length'), recommendedPasswordPolicy.minLength, 8, 128)
  const maxLength = normalizeBoundedInteger(option('identity.password.max_length'), recommendedPasswordPolicy.maxLength, 64, 512)
  if (maxLength < minLength) {
    return { ...recommendedPasswordPolicy }
  }

  return {
    minLength,
    maxLength,
    requireLowercase: normalizeEnabledOption(option('identity.password.require_lowercase')),
    requireUppercase: normalizeEnabledOption(option('identity.password.require_uppercase')),
    requireNumber: normalizeEnabledOption(option('identity.password.require_number')),
    requireSymbol: normalizeEnabledOption(option('identity.password.require_symbol'))
  }
}

export function passwordPolicyRequirements(password: string, policy: PasswordPolicy): PasswordRequirement[] {
  const length = Array.from(password).length
  const requirements: PasswordRequirement[] = [
    {
      key: 'length',
      met: length >= policy.minLength && length <= policy.maxLength
    }
  ]

  if (policy.requireLowercase) {
    requirements.push({ key: 'lowercase', met: /\p{Ll}/u.test(password) })
  }
  if (policy.requireUppercase) {
    requirements.push({ key: 'uppercase', met: /\p{Lu}/u.test(password) })
  }
  if (policy.requireNumber) {
    requirements.push({ key: 'number', met: /\p{N}/u.test(password) })
  }
  if (policy.requireSymbol) {
    requirements.push({ key: 'symbol', met: /[\p{P}\p{S}]/u.test(password) })
  }
  return requirements
}

export function passwordPolicyProgress(password: string, policy: PasswordPolicy) {
  const requirements = passwordPolicyRequirements(password, policy)
  if (requirements.length === 0) {
    return 0
  }
  const passwordLength = Array.from(password).length
  const lengthScore = passwordLength > policy.maxLength
    ? 0
    : Math.min(passwordLength / policy.minLength, 1)
  // 长度是连续体验分；字符类型要求仍按是否满足计分。
  const score = requirements.reduce((total, item) => {
    if (item.key === 'length') {
      return total + lengthScore
    }
    return total + (item.met ? 1 : 0)
  }, 0)
  return Math.round((score / requirements.length) * 100)
}

export type PasswordProgressLevel = 'empty' | 'weak' | 'medium' | 'strong'

// 按合格度分数返回语义级别，用于进度条颜色分档：
// 空=中性灰、弱=红、中=黄、强(100%)=主题色。仅依赖百分比，与计分逻辑解耦。
export function passwordPolicyProgressLevel(progress: number): PasswordProgressLevel {
  if (progress >= 100) return 'strong'
  if (progress >= 51) return 'medium'
  if (progress >= 1) return 'weak'
  return 'empty'
}

export function normalizeAltchaWidgetType(value: string | undefined): AltchaWidgetType {
  return normalizeStringChoice(value, altchaWidgetTypes, 'checkbox')
}

export function normalizeAltchaWidgetAuto(value: string | undefined): AltchaWidgetAuto {
  return normalizeStringChoice(value, altchaWidgetAutoModes, 'off')
}

export function normalizeAltchaWidgetDisplay(value: string | undefined): AltchaWidgetDisplay {
  return normalizeStringChoice(value, altchaWidgetDisplays, 'standard')
}

export function enabledOptionValue(enabled: boolean) {
  return enabled ? enabledOption : disabledOption
}

export function normalizeEnabledOption(value: string | undefined, fallback = false) {
  switch (value?.trim().toLowerCase()) {
    case enabledOption:
    case 'true':
    case '1':
    case 'yes':
    case 'on':
      return true
    case disabledOption:
    case 'false':
    case '0':
    case 'no':
    case 'off':
      return false
    default:
      return fallback
  }
}

function normalizeStringChoice<T extends string>(value: string | undefined, choices: readonly T[], fallback: T): T {
  const normalized = value?.trim().toLowerCase()
  return choices.find((choice) => choice === normalized) || fallback
}

function normalizeBoundedInteger(value: string | undefined, fallback: number, min: number, max: number) {
  const parsed = Number(value)
  if (!Number.isFinite(parsed)) {
    return fallback
  }
  const normalized = Math.trunc(parsed)
  return normalized >= min && normalized <= max ? normalized : fallback
}

function buildCustomThemeVars(color: string): Record<string, string> {
  const accent = normalizeHexColor(color) || defaultCustomThemeColor
  const rgb = hexToRgb(accent)
  const hover = mixHex(accent, '#000000', 0.16)
  const dark = mixHex(accent, '#ffffff', 0.32)

  return {
    '--sf-accent': accent,
    '--sf-accent-hover': hover,
    '--sf-accent-soft': mixHex(accent, '#ffffff', 0.92),
    '--sf-accent-soft-border': mixHex(accent, '#ffffff', 0.68),
    '--sf-accent-dark': dark,
    '--sf-accent-rgb': `${rgb.r} ${rgb.g} ${rgb.b}`,
    '--sf-accent-contrast': relativeLuminance(rgb) > 0.55 ? '#111827' : '#ffffff',
    '--sf-primary-50': mixHex(accent, '#ffffff', 0.94),
    '--sf-primary-100': mixHex(accent, '#ffffff', 0.88),
    '--sf-primary-200': mixHex(accent, '#ffffff', 0.72),
    '--sf-primary-300': mixHex(accent, '#ffffff', 0.52),
    '--sf-primary-400': mixHex(accent, '#ffffff', 0.28),
    '--sf-primary-500': accent,
    '--sf-primary-600': accent,
    '--sf-primary-700': hover,
    '--sf-primary-800': mixHex(accent, '#000000', 0.3),
    '--sf-primary-900': mixHex(accent, '#000000', 0.45),
    '--sf-primary-950': mixHex(accent, '#000000', 0.62),
    // Nuxt UI 的 success 色槽用于成功 Toast，保持与当前外观主色一致。
    '--ui-color-success-50': 'var(--sf-primary-50)',
    '--ui-color-success-100': 'var(--sf-primary-100)',
    '--ui-color-success-200': 'var(--sf-primary-200)',
    '--ui-color-success-300': 'var(--sf-primary-300)',
    '--ui-color-success-400': 'var(--sf-primary-400)',
    '--ui-color-success-500': 'var(--sf-primary-500)',
    '--ui-color-success-600': 'var(--sf-primary-600)',
    '--ui-color-success-700': 'var(--sf-primary-700)',
    '--ui-color-success-800': 'var(--sf-primary-800)',
    '--ui-color-success-900': 'var(--sf-primary-900)',
    '--ui-color-success-950': 'var(--sf-primary-950)'
  }
}

function cssVarsToStyle(vars: Record<string, string>) {
  return Object.entries(vars)
    .map(([name, value]) => `${name}: ${value}`)
    .join('; ')
}

type RGB = {
  r: number
  g: number
  b: number
}

function hexToRgb(hex: string): RGB {
  const value = (normalizeHexColor(hex) || defaultCustomThemeColor).slice(1)
  return {
    r: Number.parseInt(value.slice(0, 2), 16),
    g: Number.parseInt(value.slice(2, 4), 16),
    b: Number.parseInt(value.slice(4, 6), 16)
  }
}

function mixHex(from: string, to: string, amount: number) {
  const start = hexToRgb(from)
  const end = hexToRgb(to)
  const mix = (a: number, b: number) => Math.round(a + (b - a) * amount)
  return rgbToHex({
    r: mix(start.r, end.r),
    g: mix(start.g, end.g),
    b: mix(start.b, end.b)
  })
}

function rgbToHex(rgb: RGB) {
  return `#${[rgb.r, rgb.g, rgb.b].map((value) => value.toString(16).padStart(2, '0')).join('')}`
}

function relativeLuminance(rgb: RGB) {
  const channel = (value: number) => {
    const normalized = value / 255
    return normalized <= 0.03928
      ? normalized / 12.92
      : ((normalized + 0.055) / 1.055) ** 2.4
  }
  return 0.2126 * channel(rgb.r) + 0.7152 * channel(rgb.g) + 0.0722 * channel(rgb.b)
}
