const emptySitemapIndex = [
  '<?xml version="1.0" encoding="UTF-8"?>',
  '<sitemapindex xmlns="http://www.sitemaps.org/schemas/sitemap/0.9">',
  '</sitemapindex>'
].join('\n')

const emptyUrlSet = [
  '<?xml version="1.0" encoding="UTF-8"?>',
  '<urlset xmlns="http://www.sitemaps.org/schemas/sitemap/0.9">',
  '</urlset>'
].join('\n')

export default defineNitroPlugin((nitroApp) => {
  nitroApp.hooks.hook('sitemap:input', async (ctx: { urls: unknown[] }) => {
    if (await shouldSuppressSitemap()) {
      ctx.urls = []
    }
  })

  nitroApp.hooks.hook('sitemap:index-resolved', async (ctx: { sitemaps: unknown[] }) => {
    if (await shouldSuppressSitemap()) {
      ctx.sitemaps = []
    }
  })

  nitroApp.hooks.hook('sitemap:output', async (ctx: { sitemapName: string, sitemap: string }) => {
    if (!await shouldSuppressSitemap()) {
      return
    }

    // sitemap 模块可能从缓存或 i18n 自动源补内容，输出阶段再兜底清空。
    ctx.sitemap = ctx.sitemapName === 'sitemap' ? emptySitemapIndex : emptyUrlSet
  })
})

async function shouldSuppressSitemap() {
  const settings = await loadServerSEOSettings()
  return !settings.sitemapEnabled || !serverSEOIndexable(settings)
}
