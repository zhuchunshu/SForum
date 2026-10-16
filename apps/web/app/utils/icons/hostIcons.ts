import { HOST_ICON_BATCH_LIMIT, HOST_ICON_NAME_PATTERN } from '../iconPicker'

/**
 * 宿主机图标解析器的浏览器侧实现。
 *
 * 插件预构建组件只能提供图标名，SVG 标记始终由宿主生成。这里做三件事：
 * 按语法过滤名称、按批次合并请求、缓存结果（含「解析不到」的空结果）。
 *
 * 宿主未提供 `icons` 时调用方必须优雅降级，见 `@sforum/plugin-ui` 的
 * `SPluginIcon`：渲染空装饰元素并只告警一次，绝不抛错阻塞挂载。
 */

const ICON_ENDPOINT = '/_sforum/icons'

export type HostIconResolver = Readonly<{
  resolve: (names: readonly string[]) => Promise<Readonly<Record<string, string>>>
}>

export type HostIconResponseItem = {
  name: string
  /** 可直接内联的完整 SVG 文档。 */
  svg: string
}

export type HostIconFetcher = (names: readonly string[]) => Promise<readonly HostIconResponseItem[]>

/** 已解析结果缓存；空字符串表示宿主确认该名称不可用，避免重复请求。 */
const markupCache = new Map<string, string>()
/** 同一批次的在途请求，多个组件同时挂载时只发一次网络请求。 */
const inflight = new Map<string, Promise<void>>()

export function createHostIconResolver(fetcher: HostIconFetcher = fetchHostIcons): HostIconResolver {
  return {
    resolve: async (names) => {
      const requested = normalizeIconNames(names)
      if (!requested.length) {
        return {}
      }
      await Promise.all(chunk(requested, HOST_ICON_BATCH_LIMIT).map((batch) => ensureBatch(batch, fetcher)))
      const result: Record<string, string> = {}
      for (const name of requested) {
        const markup = markupCache.get(name)
        if (markup) {
          result[name] = markup
        }
      }
      return result
    }
  }
}

/** 共享实例：跨组件、跨页面的图标请求都会命中同一份缓存。 */
export const hostIconResolver = createHostIconResolver()

/** 测试用：清空模块级缓存，避免用例互相污染。 */
export function resetHostIconCache() {
  markupCache.clear()
  inflight.clear()
}

const PLACEHOLDER_SELECTOR = '[data-sf-icon]'
const PLACEHOLDER_READY = 'data-sf-icon-ready'

/**
 * 填充由主题模板 `{{ icon "…" }}` 服务端渲染出的占位元素。
 *
 * 占位元素本身不带 SVG：宿主不在服务端内联图标集，而是在客户端用同一个
 * 解析器把标记注入进去。图标是装饰性的，因此解析不到、请求失败、或禁用
 * JavaScript 时都只留下一个空的行内元素。
 *
 * 返回本次填充成功的元素数量，便于测试与观测。
 */
export async function hydrateHostIconPlaceholders(
  root: ParentNode | null | undefined = typeof document === 'undefined' ? null : document,
  resolver: HostIconResolver = hostIconResolver
): Promise<number> {
  if (!root) {
    return 0
  }
  const placeholders = Array.from(root.querySelectorAll<HTMLElement>(PLACEHOLDER_SELECTOR))
    .filter((element) => !element.hasAttribute(PLACEHOLDER_READY))
  if (!placeholders.length) {
    return 0
  }
  // 先整体标记，避免并发调用重复处理同一批占位元素；失败也保持已标记，
  // 这样一次失败不会变成每次导航都重试的请求风暴。
  const names = placeholders.map((element) => {
    element.setAttribute(PLACEHOLDER_READY, '')
    return element.getAttribute('data-sf-icon') || ''
  })

  let resolved: Readonly<Record<string, string>> = {}
  try {
    resolved = await resolver.resolve(names)
  } catch {
    // 图标是装饰性的：解析失败只留下空占位元素，绝不冒泡到页面渲染流程。
    return 0
  }
  let filled = 0
  for (const element of placeholders) {
    const markup = resolved[element.getAttribute('data-sf-icon') || '']
    if (!markup) {
      continue
    }
    element.innerHTML = markup
    filled++
  }
  return filled
}

async function ensureBatch(batch: readonly string[], fetcher: HostIconFetcher) {
  const missing = batch.filter((name) => !markupCache.has(name))
  if (!missing.length) {
    return
  }
  const key = missing.join(',')
  const existing = inflight.get(key)
  if (existing) {
    return existing
  }
  const request = runBatch(missing, key, fetcher)
  inflight.set(key, request)
  return request
}

async function runBatch(names: readonly string[], key: string, fetcher: HostIconFetcher) {
  try {
    const items = await fetcher(names)
    for (const item of items) {
      if (typeof item?.name === 'string' && typeof item?.svg === 'string' && item.svg) {
        markupCache.set(item.name, item.svg)
      }
    }
  } catch {
    // 图标是装饰性的：请求失败只让本次解析缺项，不能冒泡到插件组件挂载流程。
  } finally {
    // 宿主没有返回的名称也要落缓存，否则每次渲染都会重新请求同一个坏名字。
    for (const name of names) {
      if (!markupCache.has(name)) {
        markupCache.set(name, '')
      }
    }
    inflight.delete(key)
  }
}

function normalizeIconNames(names: readonly unknown[]): string[] {
  const seen = new Set<string>()
  const result: string[] = []
  for (const value of names) {
    const name = typeof value === 'string' ? value.trim() : ''
    if (!name || seen.has(name) || !HOST_ICON_NAME_PATTERN.test(name)) {
      continue
    }
    seen.add(name)
    result.push(name)
  }
  return result
}

function chunk<T>(items: readonly T[], size: number): T[][] {
  const batches: T[][] = []
  for (let index = 0; index < items.length; index += size) {
    batches.push(items.slice(index, index + size))
  }
  return batches
}

async function fetchHostIcons(names: readonly string[]): Promise<readonly HostIconResponseItem[]> {
  // 图标只在浏览器侧解析：SSR 期既没有可用的绝对 URL，也没有挂载中的插件组件。
  // 直接返回空结果，避免在服务端制造无意义请求与污染性的负缓存。
  if (import.meta.server) {
    return []
  }
  const response = await fetch(`${ICON_ENDPOINT}?names=${encodeURIComponent(names.join(','))}`, {
    credentials: 'omit',
    headers: { accept: 'application/json' }
  })
  if (!response.ok) {
    return []
  }
  const payload = await response.json() as { items?: unknown }
  if (!Array.isArray(payload?.items)) {
    return []
  }
  return payload.items.filter(isHostIconResponseItem)
}

function isHostIconResponseItem(value: unknown): value is HostIconResponseItem {
  if (!value || typeof value !== 'object') {
    return false
  }
  const item = value as Partial<HostIconResponseItem>
  return typeof item.name === 'string' && typeof item.svg === 'string'
}
