type WebOption = {
  name: string
  value: string
}

type ApiEnvelope<T> = {
  data: T
}

export type ServerSEOSettings = {
  siteUrl: string
  allowIndexing: boolean
  sitemapEnabled: boolean
  sitemapIncludeStaticPages: boolean
  sitemapIncludeForumContent: boolean
  robotsExtraAllow: string[]
  robotsExtraDisallow: string[]
  blockAiBots: boolean
  blockNonSeoBots: boolean
}

const fallbackOptions: Record<string, string> = {
  'site.url': 'http://127.0.0.1:3000',
  'seo.allow_indexing': 'enabled',
  'seo.sitemap.enabled': 'enabled',
  'seo.sitemap.include_static_pages': 'enabled',
  'seo.sitemap.include_forum_content': 'disabled',
  'seo.robots.extra_allow': '',
  'seo.robots.extra_disallow': '',
  'seo.robots.block_ai_bots': 'disabled',
  'seo.robots.block_non_seo_bots': 'disabled'
}
const fallbackSiteURL = fallbackOptions['site.url'] || 'http://127.0.0.1:3000'

export async function loadServerSEOSettings(): Promise<ServerSEOSettings> {
  const values = { ...fallbackOptions }
  try {
    const apiBaseUrl = (process.env.NUXT_API_INTERNAL_BASE_URL || 'http://api:8080/api/v1').replace(/\/+$/, '')
    const envelope = await $fetch<ApiEnvelope<WebOption[]>>(`${apiBaseUrl}/web-options`, {
      timeout: 1200
    })
    for (const option of envelope.data || []) {
      values[option.name] = option.value
    }
  } catch {
    // API 热重载或离线预览时，SEO 输出使用保守默认值。
  }

  const siteUrl = values['site.url'] || fallbackSiteURL

  return {
    siteUrl,
    allowIndexing: enabled(values['seo.allow_indexing'], true),
    sitemapEnabled: enabled(values['seo.sitemap.enabled'], true),
    sitemapIncludeStaticPages: enabled(values['seo.sitemap.include_static_pages'], true),
    sitemapIncludeForumContent: enabled(values['seo.sitemap.include_forum_content']),
    robotsExtraAllow: pathList(values['seo.robots.extra_allow']),
    robotsExtraDisallow: pathList(values['seo.robots.extra_disallow']),
    blockAiBots: enabled(values['seo.robots.block_ai_bots']),
    blockNonSeoBots: enabled(values['seo.robots.block_non_seo_bots'])
  }
}

export function serverSEOIndexable(settings: ServerSEOSettings) {
  return settings.allowIndexing && !isLocalSiteUrl(settings.siteUrl)
}

export function absoluteServerUrl(settings: ServerSEOSettings, path: string) {
  const base = settings.siteUrl.replace(/\/+$/, '') || fallbackSiteURL
  return `${base}${path.startsWith('/') ? path : `/${path}`}`
}

function enabled(value: string | undefined, fallback = false) {
  switch (value?.trim().toLowerCase()) {
    case 'enabled':
    case 'true':
    case '1':
    case 'yes':
    case 'on':
      return true
    case 'disabled':
    case 'false':
    case '0':
    case 'no':
    case 'off':
      return false
    default:
      return fallback
  }
}

function pathList(value: string | undefined) {
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

function isLocalSiteUrl(value: string) {
  try {
    const url = new URL(value)
    return ['localhost', '127.0.0.1', '0.0.0.0', '::1'].includes(url.hostname)
  } catch {
    return true
  }
}
