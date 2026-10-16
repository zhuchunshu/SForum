import { createError, setHeader } from 'h3'

import { getHostIconAsset, renderHostIconDocument } from '../../../../utils/iconSvg'

/**
 * 单图标 SVG：`GET /_sforum/icons/{lucide|tabler}/{name}.svg`
 *
 * 供主题模板与 CSS 使用。宿主把图标体清洗后包装成完整文档，`fill`/`stroke`
 * 使用 `currentColor`，因此 `<img>` 之外的用法仍可继承文字颜色。
 */
export default defineEventHandler((event) => {
  const collection = event.context.params?.collection || ''
  const rawName = event.context.params?.name || ''
  const name = rawName.endsWith('.svg') ? rawName.slice(0, -'.svg'.length) : rawName
  const asset = getHostIconAsset(`i-${collection}-${name}`)
  if (!asset) {
    throw createError({ statusCode: 404, statusMessage: 'Icon not found' })
  }

  setHeader(event, 'content-type', 'image/svg+xml; charset=utf-8')
  setHeader(event, 'cache-control', 'public, max-age=86400, s-maxage=604800, stale-while-revalidate=86400')
  setHeader(event, 'x-content-type-options', 'nosniff')
  setHeader(event, 'cross-origin-resource-policy', 'same-origin')
  return renderHostIconDocument(asset)
})
