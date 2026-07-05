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
      { loc: '/', changefreq: 'daily', priority: 1, alternatives },
      { loc: '/en', changefreq: 'daily', priority: 0.9, alternatives }
    )
  }

  // 论坛内容 sitemap 会在分类、主题、公开资料 read model 落地后接入。
  if (settings.sitemapIncludeForumContent) {
    return urls
  }

  return urls
})
