import type { ApiEnvelope } from '~/composables/useApiClient'

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

export type SEOSettings = {
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
  'site.default_locale': 'zh-CN',
  'site.supported_locales': 'zh-CN,en-US',
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
  'forum.default_category_slug': 'general',
  'forum.tags.creation_mode': 'controlled',
  'forum.tags.public_pages': enabledOption,
  'forum.tags.max_per_topic': '5',
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
  'seo.sitemap.include_forum_content': disabledOption,
  'seo.schema_org.enabled': enabledOption,
  'seo.schema_org.search_action_enabled': enabledOption,
  'seo.schema_org.discussion_enabled': enabledOption,
  'seo.schema_org.organization_logo_url': ''
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
  const defaultLocale = computed(() => webOption('site.default_locale', 'zh-CN'))
  const supportedLocales = computed(() => parseSupportedLocales(webOption('site.supported_locales', 'zh-CN,en-US')))
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
  const seoSettings = computed(() => resolveSEOSettings(options.value))
  const seoIndexable = computed(() => isSEOIndexingAllowed(seoSettings.value, siteUrl.value))

  function humanVerificationEnabledFor(scenario: HumanVerificationScenario) {
    return humanVerificationProvider.value === 'altcha' && humanVerificationScenarioSettings.value[scenario]
  }

  function footerCopyrightTemplate(localeCode: string) {
    return footerCopyright.value[normalizeFooterLocale(localeCode)]
  }

  function footerLinkLabel(link: FooterLinkOption, localeCode: string) {
    return link.labels[normalizeFooterLocale(localeCode)] || link.labels['zh-CN']
  }

  return {
    options,
    siteName,
    siteUrl,
    defaultLocale,
    supportedLocales,
    appearanceTheme,
    resolvedAppearanceTheme,
    footerCopyright,
    footerLinks,
    humanVerificationProvider,
    humanVerificationScenarioSettings,
    altchaWidgetSettings,
    passwordPolicy,
    seoSettings,
    seoIndexable,
    humanVerificationEnabledFor,
    webOption,
    footerCopyrightTemplate,
    footerLinkLabel,
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
  return {
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
    sitemapIncludeForumContent: normalizeEnabledOption(option('seo.sitemap.include_forum_content')),
    schemaOrgEnabled: normalizeEnabledOption(option('seo.schema_org.enabled'), true),
    schemaOrgSearchActionEnabled: normalizeEnabledOption(option('seo.schema_org.search_action_enabled'), true),
    schemaOrgDiscussionEnabled: normalizeEnabledOption(option('seo.schema_org.discussion_enabled'), true),
    schemaOrgOrganizationLogoUrl: option('seo.schema_org.organization_logo_url').trim()
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
  const met = requirements.filter(item => item.met).length
  return Math.round((met / requirements.length) * 100)
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
