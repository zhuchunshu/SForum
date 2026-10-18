import lucideCollection from '@iconify-json/lucide/icons.json'
import tablerCollection from '@iconify-json/tabler/icons.json'

import {
  HOST_ICON_BATCH_LIMIT,
  parseHostIconName,
  type HostIconSet
} from '../../app/utils/iconPicker'

/**
 * 宿主图标集服务：把 `@iconify-json` 的图标体解析成插件可用的 SVG。
 *
 * 数据完全静态（来自构建期依赖），因此这里只做两件事：解析名称、清洗标记。
 * 清洗不是为了防住今天的图标源，而是让未来更换图标源不会静默变成 XSS 面：
 * 插件只能提供图标名，标记永远由宿主生成。
 */

type IconifyIcon = {
  body?: string
  parent?: string
  width?: number
  height?: number
}

type IconifyCollection = {
  icons: Record<string, IconifyIcon>
  aliases?: Record<string, IconifyIcon>
  width?: number
  height?: number
}

export type HostIconAsset = {
  /** 完整宿主图标名，例如 `i-lucide-arrow-up-right`。 */
  name: string
  set: HostIconSet
  icon: string
  width: number
  height: number
  /** 已清洗的 SVG 内部标记。 */
  body: string
}

const collections: Record<HostIconSet, IconifyCollection> = {
  lucide: lucideCollection as IconifyCollection,
  tabler: tablerCollection as IconifyCollection
}

/** 允许出现在图标体里的元素：当前两个集合实际只用到了这些。 */
const ALLOWED_ELEMENTS = new Set([
  'path',
  'circle',
  'ellipse',
  'rect',
  'line',
  'polyline',
  'polygon',
  'g',
  'defs',
  'use',
  'title',
  'desc'
])

/** 允许保留的属性白名单。任何未列出的属性（含所有 `on*` 与 `style`）都会被丢弃。 */
const ALLOWED_ATTRIBUTES = new Set([
  'id',
  'class',
  'd',
  'cx',
  'cy',
  'r',
  'rx',
  'ry',
  'x',
  'y',
  'x1',
  'y1',
  'x2',
  'y2',
  'width',
  'height',
  'points',
  'href',
  'fill',
  'fill-rule',
  'fill-opacity',
  'clip-rule',
  'clip-path',
  'stroke',
  'stroke-width',
  'stroke-linecap',
  'stroke-linejoin',
  'stroke-opacity',
  'stroke-dasharray',
  'stroke-dashoffset',
  'stroke-miterlimit',
  'opacity',
  'transform'
])

/** 只能引用同文档片段、十六进制色值或纯关键字的属性。 */
const REFERENCE_ATTRIBUTES = new Set(['href', 'fill', 'stroke', 'clip-path'])

const TAG_PATTERN = /<\s*(\/?)\s*([a-zA-Z][a-zA-Z0-9:-]*)((?:"[^"]*"|'[^']*'|[^>"'])*?)(\/?)\s*>/g
const ATTRIBUTE_PATTERN = /([a-zA-Z_:][-a-zA-Z0-9_:.]*)\s*=\s*(?:"([^"]*)"|'([^']*)')/g
const DROPPED_BLOCK_PATTERN = /<\s*(script|style)\b[^>]*>[\s\S]*?<\s*\/\s*\1\s*>/gi
const UNSAFE_VALUE_PATTERN = /javascript:|vbscript:|data:text\/html|expression\s*\(|<\s*\/?\s*(script|iframe)/i
const CLASS_VALUE_PATTERN = /^[A-Za-z0-9 _-]*$/
const HEX_COLOR_PATTERN = /^#(?:[0-9a-fA-F]{3,4}|[0-9a-fA-F]{6}|[0-9a-fA-F]{8})$/
const KEYWORD_PATTERN = /^[A-Za-z][A-Za-z-]*$/
const URL_REFERENCE_PATTERN = /^url\(\s*#([A-Za-z0-9_:.-]+)\s*\)$/

/**
 * 解析一个宿主图标名并返回可直接内联的 SVG 资产。
 * 名称非法、集合内不存在、或图标体清洗后为空时返回 `undefined`。
 */
export function getHostIconAsset(value: unknown): HostIconAsset | undefined {
  const parsed = parseHostIconName(value)
  if (!parsed) {
    return undefined
  }

  const collection = collections[parsed.set]
  const icon = resolveIcon(collection, parsed.icon)
  if (!icon?.body) {
    return undefined
  }

  const body = sanitizeIconBody(icon.body, `${parsed.set}-${parsed.icon}`)
  if (!body) {
    return undefined
  }

  return {
    name: `i-${parsed.set}-${parsed.icon}`,
    set: parsed.set,
    icon: parsed.icon,
    width: icon.width || collection.width || 24,
    height: icon.height || collection.height || 24,
    body
  }
}

/**
 * 批量解析。非法或未知的名称被静默跳过（而不是让整批失败）；
 * 名称去重后，最多处理 `HOST_ICON_BATCH_LIMIT` 个，避免一次调用带走整个图标集。
 */
export function getHostIconAssets(values: readonly unknown[]): HostIconAsset[] {
  const seen = new Set<string>()
  const assets: HostIconAsset[] = []
  for (const value of values) {
    if (seen.size >= HOST_ICON_BATCH_LIMIT) break
    const name = typeof value === 'string' ? value.trim() : ''
    if (!name || seen.has(name)) continue
    seen.add(name)
    const asset = getHostIconAsset(name)
    if (asset) assets.push(asset)
  }
  return assets
}

/** 把图标资产包装成完整 SVG 文档，供 `/_sforum/icons/...` 单图标路由使用。 */
export function renderHostIconDocument(asset: HostIconAsset): string {
  return buildSvgDocument(asset, String(asset.width), String(asset.height))
}

/**
 * 包装成可内联的 SVG：用 `1em` 作为尺寸，颜色继承 `currentColor`。
 * 插件 bridge 返回这一形态，组件无需知道集合的 viewBox 尺寸。
 */
export function renderHostIconInline(asset: HostIconAsset): string {
  return buildSvgDocument(asset, '1em', '1em')
}

function buildSvgDocument(asset: HostIconAsset, width: string, height: string): string {
  return `<svg xmlns="http://www.w3.org/2000/svg" width="${width}" height="${height}" viewBox="0 0 ${asset.width} ${asset.height}" fill="none" aria-hidden="true">${asset.body}</svg>`
}

export { HOST_ICON_BATCH_LIMIT }

function resolveIcon(collection: IconifyCollection, name: string, depth = 0): IconifyIcon | undefined {
  const direct = collection.icons[name]
  if (direct?.body) {
    return direct
  }
  // iconify 的 alias 只指向同集合内的 parent；限制深度避免畸形数据造成递归。
  if (depth >= 4) {
    return undefined
  }
  const alias = collection.aliases?.[name]
  if (!alias) {
    return undefined
  }
  const parentName = alias.parent || name
  const parent = resolveIcon(collection, parentName, depth + 1)
  if (!parent) {
    return undefined
  }
  return {
    body: parent.body,
    width: alias.width || parent.width,
    height: alias.height || parent.height
  }
}

/**
 * 白名单重建图标体：
 * 1. 先整体移除 script/style 块；
 * 2. 再逐个标签只保留白名单元素与白名单属性；
 * 3. 引用型属性只接受同文档片段、`url(#id)`、或纯色关键字；
 * 4. `id` 加名称前缀，避免同一文档内联多个图标时片段引用互相覆盖。
 *
 * 导出给契约测试使用；生产调用方应使用 `getHostIconAsset`。
 */
export function sanitizeIconBody(body: string, idPrefix: string): string {
  if (!body) {
    return ''
  }

  // 先整体移除 script/style 块，再判断剩余标记；否则整块检测会把合法的
  // `<script>` 嵌套串连同正常的图标体一起判死。
  // 剥离必须反复到不动点：嵌套构造（如 `<scr<script>a</script>ipt>`）单次替换后
  // 会重新拼出 `<script>`，正是 js/incomplete-multi-character-sanitization 的形态。
  let stripped = body
  for (;;) {
    const next = stripped.replace(DROPPED_BLOCK_PATTERN, '')
    if (next === stripped) {
      break
    }
    stripped = next
  }
  if (UNSAFE_VALUE_PATTERN.test(stripped)) {
    return ''
  }

  const ids = new Set<string>()
  for (const match of stripped.matchAll(/\sid\s*=\s*(?:"([^"]*)"|'([^']*)')/g)) {
    const id = (match[1] ?? match[2] ?? '').trim()
    if (id) ids.add(id)
  }

  let output = ''
  let sawTag = false
  TAG_PATTERN.lastIndex = 0
  let match: RegExpExecArray | null
  while ((match = TAG_PATTERN.exec(stripped)) !== null) {
    // 标签之间的文本节点（图标体里只可能是空白）一律丢弃。
    sawTag = true
    const closing = match[1] === '/'
    const element = (match[2] || '').toLowerCase()
    const selfClosing = match[4] === '/'
    if (!ALLOWED_ELEMENTS.has(element)) {
      continue
    }
    const attributes = rebuildAttributes(match[3] || '', idPrefix, ids)
    // 属性被清空的自闭合元素已经没有任何作用，丢掉以免留下 `<use/>` 噪声。
    if (!closing && selfClosing && !attributes) {
      continue
    }
    output += closing
      ? `</${element}>`
      : `<${element}${attributes}${selfClosing ? '/>' : '>'}`
  }
  return sawTag ? output : ''
}

function rebuildAttributes(raw: string, idPrefix: string, ids: Set<string>): string {
  let output = ''
  ATTRIBUTE_PATTERN.lastIndex = 0
  let match: RegExpExecArray | null
  while ((match = ATTRIBUTE_PATTERN.exec(raw)) !== null) {
    const name = (match[1] || '').toLowerCase()
    const value = (match[2] ?? match[3] ?? '').trim()
    if (!ALLOWED_ATTRIBUTES.has(name) || !value) {
      continue
    }
    const safeValue = sanitizeAttributeValue(name, value, idPrefix, ids)
    if (safeValue === undefined) {
      continue
    }
    output += ` ${name}="${safeValue}"`
  }
  return output
}

function sanitizeAttributeValue(
  name: string,
  value: string,
  idPrefix: string,
  ids: Set<string>
): string | undefined {
  if (UNSAFE_VALUE_PATTERN.test(value)) {
    return undefined
  }
  if (name === 'class') {
    return CLASS_VALUE_PATTERN.test(value) ? value : undefined
  }
  if (name === 'id') {
    // 只保留被同文档片段引用到的 id，其余丢弃以免污染页面 id 空间。
    return ids.has(value) ? `${idPrefix}-${value}` : undefined
  }
  if (name === 'href') {
    const target = value.startsWith('#') ? value.slice(1) : ''
    return ids.has(target) ? `#${idPrefix}-${target}` : undefined
  }
  if (REFERENCE_ATTRIBUTES.has(name)) {
    const urlReference = value.match(URL_REFERENCE_PATTERN)
    if (urlReference) {
      const target = urlReference[1] || ''
      return ids.has(target) ? `url(#${idPrefix}-${target})` : undefined
    }
    if (HEX_COLOR_PATTERN.test(value) || KEYWORD_PATTERN.test(value)) {
      return value
    }
    return undefined
  }
  return value
}
