import type { PublicFrontendComponentRef } from '~/runtime/public-extensions/pagePolicy'
import { normalizePublicFrontendComponentRefs } from '~/runtime/public-extensions/pagePolicy'

/**
 * forum.page.regions 区域内容(GET /site/page-regions)。
 * 纯宿主描述符:link 链接卡 / action 扩展路由动作卡 / widget L2 组件引用。
 * widget 挂载仍由公开 L2 运行时(descriptor 签发 + 信任 + CSP)权威裁决。
 */

/** 与后端 RegionCatalog pageRegionMatrix 同步的白名单;其余页面不发区域请求。 */
export const PAGE_REGION_PAGES: ReadonlySet<string> = new Set([
  'forum.home',
  'forum.category.index',
  'forum.category.show',
  'forum.tag.index',
  'forum.tag.show',
  'forum.topic.show',
  'forum.profile.show',
  'forum.topic.create',
  'forum.topic.reply',
  'forum.notifications'
])

const ID_PATTERN = /^[a-z0-9][a-z0-9._-]{1,120}$/
const REGION_ID_PATTERN = /^[a-z0-9][a-z0-9_]{1,60}$/
const MAX_REGIONS = 8
const MAX_ITEMS_PER_REGION = 32
const PAGE_REGIONS_SCHEMA_VERSION = 'sforum.page-regions@1'
const PAGE_REGIONS_TIMEOUT_MS = 8000

export type PageRegionItem = {
  extensionId: string
  contributionId: string
  label: Record<string, string>
  icon: string
  kind: 'link' | 'action' | 'widget'
  href: string
  method: string
  path: string
  widget: PublicFrontendComponentRef | null
  order: number
}

export type PageRegion = {
  id: string
  kind: string
  items: PageRegionItem[]
}

export type PageRegionsPayload = {
  page: string
  regions: PageRegion[]
}

/** 站内相对路径白名单(与后端 safePublicHostLink 同规,前端二次防御)。 */
export function safePageRegionHref(href: string) {
  return href.startsWith('/')
    && !href.startsWith('//')
    && !href.includes('://')
    && !href.includes('..')
    && href !== '/api'
    && !href.startsWith('/api/')
}

function safeExtensionProxyPath(path: string) {
  return path.startsWith('/')
    && path !== '/'
    && !path.includes('://')
    && !path.includes('..')
    && !path.startsWith('/api')
}

function parseItem(raw: unknown): PageRegionItem | null {
  if (!raw || typeof raw !== 'object' || Array.isArray(raw)) {
    return null
  }
  const record = raw as Record<string, unknown>
  const extensionId = String(record.extensionId || '').trim()
  const contributionId = String(record.contributionId || '').trim()
  const kind = String(record.kind || '').trim()
  if (!ID_PATTERN.test(extensionId) || !ID_PATTERN.test(contributionId)) {
    return null
  }
  const label: Record<string, string> = {}
  if (record.label && typeof record.label === 'object' && !Array.isArray(record.label)) {
    for (const [locale, value] of Object.entries(record.label as Record<string, unknown>)) {
      if (typeof value === 'string' && value.trim()) {
        label[locale] = value.trim()
      }
    }
  }
  const item: PageRegionItem = {
    extensionId,
    contributionId,
    label,
    icon: typeof record.icon === 'string' ? record.icon.trim() : '',
    kind: 'link',
    href: '',
    method: '',
    path: '',
    widget: null,
    order: Number.isInteger(record.order) ? Number(record.order) : 0
  }
  switch (kind) {
    case 'link': {
      const href = String(record.href || '').trim()
      if (!safePageRegionHref(href)) {
        return null
      }
      item.kind = 'link'
      item.href = href
      return item
    }
    case 'action': {
      const method = String(record.method || '').trim().toUpperCase()
      const path = String(record.path || '').trim()
      if (!['GET', 'POST', 'PUT', 'PATCH', 'DELETE'].includes(method) || !safeExtensionProxyPath(path)) {
        return null
      }
      item.kind = 'action'
      item.method = method
      item.path = path
      return item
    }
    case 'widget': {
      const widget = record.widget as Record<string, unknown> | undefined
      const widgetExtensionId = String(widget?.extensionId || '').trim()
      const componentId = String(widget?.componentId || '').trim()
      // 组件必须归属贡献所在扩展(禁止跨包引用)。
      if (!ID_PATTERN.test(componentId) || widgetExtensionId !== extensionId) {
        return null
      }
      item.kind = 'widget'
      item.widget = { extensionId: widgetExtensionId, componentId }
      return item
    }
    default:
      return null
  }
}

/** 解析响应;形状非法整体丢弃(fail closed),单条非法仅丢该条。 */
export function parsePageRegionsPayload(raw: unknown, page: string): PageRegionsPayload | null {
  if (!raw || typeof raw !== 'object' || Array.isArray(raw)) {
    return null
  }
  const record = raw as Record<string, unknown>
  if (record.schemaVersion !== PAGE_REGIONS_SCHEMA_VERSION
    || String(record.page || '') !== page
    || !Array.isArray(record.regions)
    || record.regions.length > MAX_REGIONS
  ) {
    return null
  }
  const regions: PageRegion[] = []
  for (const rawRegion of record.regions) {
    if (!rawRegion || typeof rawRegion !== 'object' || Array.isArray(rawRegion)) {
      return null
    }
    const regionRecord = rawRegion as Record<string, unknown>
    const id = String(regionRecord.id || '').trim()
    if (!REGION_ID_PATTERN.test(id) || !Array.isArray(regionRecord.items) || regionRecord.items.length > MAX_ITEMS_PER_REGION) {
      return null
    }
    const items = regionRecord.items
      .map(parseItem)
      .filter((item): item is PageRegionItem => item !== null)
    if (items.length) {
      regions.push({ id, kind: String(regionRecord.kind || '').trim(), items })
    }
  }
  return { page, regions }
}

/** 收集页面所有 widget 引用,供 SSR 阶段 CSP 单点聚合。 */
export function collectRegionWidgetRefs(payload: PageRegionsPayload | null): PublicFrontendComponentRef[] {
  if (!payload) {
    return []
  }
  const refs: PublicFrontendComponentRef[] = []
  for (const region of payload.regions) {
    for (const item of region.items) {
      if (item.kind === 'widget' && item.widget) {
        refs.push(item.widget)
      }
    }
  }
  try {
    return normalizePublicFrontendComponentRefs(refs)
  } catch {
    // 非法引用整体失败关闭:不写 CSP、widget 不挂载,descriptor 卡不受影响。
    return []
  }
}

/** 跨组件共享同页区域数据:Resolver 拉取写入,SFRegionOutlet 只读。 */
export function usePageRegionsState(page: string) {
  return useState<PageRegionsPayload | null>(`sf-page-regions:${page}`, () => null)
}

/** 拉取失败返回 null(fail closed,页面照常渲染,区域为空)。 */
export async function fetchPageRegions(page: string): Promise<PageRegionsPayload | null> {
  try {
    const { request } = useApiClient()
    const raw = await request<unknown>(`/site/page-regions?page=${encodeURIComponent(page)}`, {
      timeout: PAGE_REGIONS_TIMEOUT_MS
    })
    return parsePageRegionsPayload(raw, page)
  } catch {
    return null
  }
}
