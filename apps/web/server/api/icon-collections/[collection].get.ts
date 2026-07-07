import { createError, getQuery } from 'h3'

import { getIconCatalogPage, isIconCatalogCollection } from '../../utils/iconCatalog'

export default defineEventHandler((event) => {
  const collection = event.context.params?.collection
  if (!isIconCatalogCollection(collection)) {
    throw createError({
      statusCode: 404,
      statusMessage: 'Icon collection not found'
    })
  }

  // 图标目录数据完全静态（来自 @iconify-json 包），浏览器/CDN 可长缓存。
  setHeader(event, 'cache-control', 'public, max-age=86400, s-maxage=604800, stale-while-revalidate=86400')
  return getIconCatalogPage(collection, getQuery(event))
})
