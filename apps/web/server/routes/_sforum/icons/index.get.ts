import { createError, getQuery, setHeader } from 'h3'

import { getHostIconAssets, renderHostIconInline, HOST_ICON_BATCH_LIMIT } from '../../../utils/iconSvg'

/**
 * 批量图标解析：`GET /_sforum/icons?names=i-lucide-x,i-tabler-y`
 *
 * 返回可直接内联的完整 SVG 文档（尺寸 `1em`，颜色继承 `currentColor`）。
 * 插件预构建组件只能提供图标名，SVG 标记始终由宿主生成。未知或非法的名称被
 * 静默跳过（而不是让整批失败），这样组件里一个笔误不会让整块 UI 空白。
 */
export default defineEventHandler((event) => {
  const raw = getQuery(event).names
  const value = Array.isArray(raw) ? raw.join(',') : String(raw || '')
  const names = value.split(',').map((item) => item.trim()).filter(Boolean)
  if (names.length > HOST_ICON_BATCH_LIMIT) {
    throw createError({
      statusCode: 400,
      statusMessage: `At most ${HOST_ICON_BATCH_LIMIT} icon names per request`
    })
  }

  // 图标数据完全静态（来自 @iconify-json 包），浏览器与 CDN 可长缓存。
  setHeader(event, 'cache-control', 'public, max-age=86400, s-maxage=604800, stale-while-revalidate=86400')
  return {
    items: getHostIconAssets(names).map((asset) => ({
      name: asset.name,
      svg: renderHostIconInline(asset)
    }))
  }
})
