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

export type FooterLinkOption = {
  key: FooterLinkKey
  labels: Record<FooterLocale, string>
  url: string
}

type RefreshOptions = {
  timeout?: number
}

export const appearanceThemes: AppearanceThemePreset[] = ['pine_teal', 'ocean_blue', 'violet', 'rose', 'amber']
export const defaultCustomThemeColor = '#2563eb'

const customThemePrefix = 'custom:'

const defaultFooterLinks: FooterLinkOption[] = [
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
  'appearance.theme': 'pine_teal',
  'footer.copyright.zh-CN': '© {year} {siteName}。保留所有权利。',
  'footer.copyright.en-US': '© {year} {siteName}. All rights reserved.',
  'footer.links': JSON.stringify(defaultFooterLinks)
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
  const appearanceTheme = computed(() => parseAppearanceTheme(webOption('appearance.theme', 'pine_teal')))
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
      return defaultFooterLinks
    }

    const normalized = parsed
      .map(normalizeFooterLink)
      .filter((item): item is FooterLinkOption => item !== null)
    if (normalized.length !== defaultFooterLinks.length) {
      return defaultFooterLinks
    }

    const byKey = new Map(normalized.map((item) => [item.key, item]))
    return defaultFooterLinks.map((item) => byKey.get(item.key) || item)
  } catch {
    return defaultFooterLinks
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

export function normalizeAppearanceThemeValue(value: string | undefined): AppearanceTheme {
  const raw = value?.trim().toLowerCase() || ''
  if (isAppearanceThemePreset(raw)) {
    return raw
  }

  if (raw.startsWith(customThemePrefix)) {
    const color = normalizeHexColor(raw.slice(customThemePrefix.length))
    return color ? (`${customThemePrefix}${color}` as AppearanceTheme) : 'pine_teal'
  }

  const color = normalizeHexColor(raw)
  return color ? (`${customThemePrefix}${color}` as AppearanceTheme) : 'pine_teal'
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
    '--sf-primary-950': mixHex(accent, '#000000', 0.62)
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
