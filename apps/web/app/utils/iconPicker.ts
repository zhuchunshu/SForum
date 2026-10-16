export type IconCollectionId = 'tabler' | 'nuxt'
export type IconSource = 'preset' | 'custom'

/**
 * 宿主向插件公开的图标集（iconify 前缀）。
 * 与 `nuxt.config.ts` 的 `icon.collections` 以及 Go 侧贡献点校验必须保持一致。
 */
export type HostIconSet = 'lucide' | 'tabler'

export type IconPickerItem = {
  name: string
  label: string
  keywords: string[]
}

export type ParsedIconName = {
  set: HostIconSet
  /** 去掉集合前缀后的 iconify 图标名，例如 `arrow-up-right`。 */
  icon: string
}

export const ICON_PICKER_PAGE_SIZE = 60
export const ICON_PICKER_MAX_PAGE_SIZE = 120

/**
 * 宿主认可的图标名语法。管理端 surface 校验、贡献点校验、插件 bridge 解析
 * 共用这一条规则，避免出现「声明通过但运行时解析不到」的分叉。
 */
export const HOST_ICON_NAME_PATTERN = /^i-(lucide|tabler)-[a-z0-9]+(?:-[a-z0-9]+)*$/

/** 单次批量解析的图标名上限，防止插件用一次请求拉走整个图标集。 */
export const HOST_ICON_BATCH_LIMIT = 32

export function parseHostIconName(value: unknown): ParsedIconName | undefined {
  const name = typeof value === 'string' ? value.trim() : ''
  if (!HOST_ICON_NAME_PATTERN.test(name)) {
    return undefined
  }

  const set = name.startsWith('i-tabler-') ? 'tabler' : 'lucide'
  return { set, icon: name.slice(`i-${set}-`.length) }
}

export function isHostIconName(value: unknown): boolean {
  return parseHostIconName(value) !== undefined
}

export function normalizeIconName(value: string, fallbackPrefix: string) {
  const rawValue = value.trim().toLowerCase()
  if (!rawValue) {
    return ''
  }

  if (rawValue.startsWith('i-')) {
    return rawValue
  }

  if (/^[a-z0-9-]+:[a-z0-9-]+$/.test(rawValue)) {
    const [collection, name] = rawValue.split(':')
    return `i-${collection}-${name}`
  }

  if (/^[a-z0-9][a-z0-9-]*$/.test(rawValue)) {
    return `${fallbackPrefix}${rawValue}`
  }

  return rawValue
}

export function collectionFromName(value?: string): IconCollectionId | undefined {
  const name = value?.trim().toLowerCase()
  if (!name) {
    return undefined
  }

  if (name.startsWith('i-tabler-') || name.startsWith('tabler:')) {
    return 'tabler'
  }

  return 'nuxt'
}

export function isNuxtIconName(value: string) {
  return value.startsWith('i-') && value.length > 2
}

export function dedupeIconItems(items: IconPickerItem[]) {
  const seen = new Set<string>()
  return items.filter((item) => {
    if (seen.has(item.name)) {
      return false
    }

    seen.add(item.name)
    return true
  })
}

export function nextIconVisibleCount(current: number, total: number, pageSize = ICON_PICKER_PAGE_SIZE) {
  if (total <= 0) {
    return 0
  }

  return Math.min(Math.max(0, current) + Math.max(1, pageSize), total)
}
