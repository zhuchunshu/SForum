import type { SitemapUrlInput } from '@nuxtjs/sitemap'

export default defineSitemapEventHandler(async () => {
  const settings = await loadServerSEOSettings()
  if (!settings.sitemapEnabled || !serverSEOIndexable(settings)) {
    return []
  }

  const urls: SitemapUrlInput[] = []
  if (settings.sitemapIncludeStaticPages) {
    // no_prefix：中英共用同一 URL，cookie 决定 UI 语言；sitemap 只收默认语入口。
    urls.push({ loc: '/' })
  }
  return urls
})
