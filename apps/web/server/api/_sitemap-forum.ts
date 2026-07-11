import type { SitemapUrlInput } from '@nuxtjs/sitemap'

type SitemapEntries = { items: Array<{ path: string, lastModified: string }>, page: number, perPage: number, hasMore: boolean }
type ApiEnvelope<T> = { data: T }

export async function forumSitemapUrls(type: 'categories' | 'tags' | 'topics' | 'profiles'): Promise<SitemapUrlInput[]> {
  const settings = await loadServerSEOSettings()
  if (!settings.sitemapEnabled || !settings.sitemapIncludeForumContent || !serverSEOIndexable(settings)) return []
  const apiBaseUrl = (process.env.NUXT_API_INTERNAL_BASE_URL || 'http://api:8080/api/v1').replace(/\/+$/, '')
  const urls: SitemapUrlInput[] = []
  for (let page = 1; page <= 1000; page++) {
    const envelope = await $fetch<ApiEnvelope<SitemapEntries>>(`${apiBaseUrl}/seo/sitemap-entries`, { query: { type, page, perPage: 5000 }, timeout: 5000 })
    urls.push(...envelope.data.items.map(item => ({ loc: item.path, lastmod: item.lastModified })))
    if (!envelope.data.hasMore) break
  }
  return urls
}
