import type { SitemapUrlInput } from '@nuxtjs/sitemap'

export default defineSitemapEventHandler(async () => {
  const settings = await loadServerSEOSettings()
  if (!settings.sitemapEnabled || !serverSEOIndexable(settings)) {
    return []
  }

  const urls: SitemapUrlInput[] = []
  if (settings.sitemapIncludeStaticPages) {
    const zh = absoluteServerUrl(settings, '/')
    const en = absoluteServerUrl(settings, '/en')
    const alternatives = [
      { hreflang: 'zh-CN', href: zh },
      { hreflang: 'en-US', href: en },
      { hreflang: 'x-default', href: zh }
    ]
    urls.push(
      { loc: '/', alternatives },
      { loc: '/en', alternatives }
    )
  }
  return urls
})
