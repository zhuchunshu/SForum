// 扩展的本地化展示面：把 manifest 顶层的默认文案与 langs 覆盖合并成一份可渲染的
// 展示值。纯函数 + 类型，零副作用，因此可与 adminExtensions.ts 的其余契约分离。
//
// 这里只 `import type`：adminExtensions.ts 会 re-export 本文件，值级依赖会形成
// 运行期循环，而类型在编译后会被抹掉，方向保持单向。
import type { AdminExtension, AdminExtensionAuthor, AdminExtensionLocale } from './adminExtensions'

export type ExtensionLocalizedDisplay = {
  name: string
  description: string
  url: string
  author: AdminExtensionAuthor
}

function normalizeExtensionLocaleKey(value?: string | null) {
  const raw = `${value || ''}`.trim().replace(/_/g, '-')
  if (!raw) {
    return ''
  }
  const parts = raw.split('-').filter(Boolean)
  const first = parts[0]
  if (!first) {
    return ''
  }
  // 语言主码小写；两位区域码大写（zh-CN），其余段小写。
  parts[0] = first.toLowerCase()
  for (let index = 1; index < parts.length; index += 1) {
    const part = parts[index]
    if (!part) {
      continue
    }
    parts[index] = part.length === 2 ? part.toUpperCase() : part.toLowerCase()
  }
  return parts.join('-')
}

function extensionLocaleLookupCandidates(locale?: string | null) {
  const code = normalizeExtensionLocaleKey(locale)
  if (!code) {
    return [] as string[]
  }
  const candidates = [code]
  const primary = code.split('-')[0]
  if (primary && primary !== code) {
    candidates.push(primary)
  }
  return candidates
}

function lookupExtensionLocale(langs: Record<string, AdminExtensionLocale> | undefined, locale?: string | null) {
  if (!langs) {
    return undefined
  }
  const entries = Object.entries(langs)
  if (!entries.length) {
    return undefined
  }
  // 键做规范化后再匹配，兼容 zh / zh-CN / zh_CN。
  const normalized = new Map(
    entries
      .map(([key, value]) => [normalizeExtensionLocaleKey(key), value] as const)
      .filter(([key]) => Boolean(key))
  )
  for (const candidate of extensionLocaleLookupCandidates(locale)) {
    const match = normalized.get(candidate)
    if (match) {
      return match
    }
  }
  return undefined
}

function resolveLocaleCode(locale?: unknown) {
  // 兼容直接传入 string / Ref / ComputedRef，避免模板误传对象导致匹配失败。
  if (locale == null) {
    return ''
  }
  if (typeof locale === 'string') {
    return locale
  }
  if (typeof locale === 'object' && locale !== null && 'value' in locale) {
    const value = (locale as { value: unknown }).value
    return typeof value === 'string' ? value : `${value ?? ''}`
  }
  return `${locale}`
}

// 按当前 UI locale 解析展示文案；无 langs 或未命中时回退顶层默认英文。
export function extensionLocalizedDisplay(item: AdminExtension, locale?: unknown): ExtensionLocalizedDisplay {
  const manifest = item.manifest
  const display: ExtensionLocalizedDisplay = {
    name: item.name || manifest.name || '',
    description: manifest.description || '',
    url: manifest.url || '',
    author: {
      name: manifest.author?.name || '',
      url: manifest.author?.url,
      email: manifest.author?.email
    }
  }
  const override = lookupExtensionLocale(manifest.langs, resolveLocaleCode(locale))
  if (!override) {
    return display
  }
  if (override.name?.trim()) {
    display.name = override.name.trim()
  }
  if (override.description?.trim()) {
    display.description = override.description.trim()
  }
  if (override.url?.trim()) {
    display.url = override.url.trim()
  }
  if (override.author?.name?.trim()) {
    display.author.name = override.author.name.trim()
  }
  if (override.author?.url?.trim()) {
    display.author.url = override.author.url.trim()
  }
  if (override.author?.email?.trim()) {
    display.author.email = override.author.email.trim()
  }
  return display
}

export function extensionDisplayName(item: AdminExtension, locale?: unknown) {
  return extensionLocalizedDisplay(item, locale).name
}

export function extensionDisplayDescription(item: AdminExtension, locale?: unknown) {
  return extensionLocalizedDisplay(item, locale).description
}

export function extensionAuthorName(item: AdminExtension, locale?: unknown) {
  return extensionLocalizedDisplay(item, locale).author.name?.trim() || ''
}

export function extensionAuthorWebsite(item: AdminExtension, locale?: unknown) {
  const display = extensionLocalizedDisplay(item, locale)
  return display.author.url || display.url || ''
}
