export type IconCollectionId = 'tabler' | 'nuxt'
export type IconSource = 'preset' | 'custom'

export type IconPickerItem = {
  name: string
  label: string
  keywords: string[]
}

export const ICON_PICKER_PAGE_SIZE = 60
export const ICON_PICKER_MAX_PAGE_SIZE = 120

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
