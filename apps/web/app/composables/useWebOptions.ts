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

export type AppearanceTheme = 'pine_teal' | 'ocean_blue' | 'violet' | 'rose' | 'amber'
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

export const appearanceThemes: AppearanceTheme[] = ['pine_teal', 'ocean_blue', 'violet', 'rose', 'amber']

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
  const theme = value.trim().toLowerCase() as AppearanceTheme
  return appearanceThemes.includes(theme) ? theme : 'pine_teal'
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
