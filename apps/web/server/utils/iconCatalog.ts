import lucideCollection from '@iconify-json/lucide/icons.json'
import tablerCollection from '@iconify-json/tabler/icons.json'

import {
  ICON_PICKER_MAX_PAGE_SIZE,
  ICON_PICKER_PAGE_SIZE,
  type IconCollectionId,
  type IconPickerItem
} from '../../app/utils/iconPicker'

type IconCollectionData = {
  icons: Record<string, unknown>
}

type IconCatalogConfig = {
  id: IconCollectionId
  iconifyPrefix: 'lucide' | 'tabler'
  nuxtPrefix: string
  names: string[]
}

export type IconCatalogPageParams = {
  q?: unknown
  page?: unknown
  pageSize?: unknown
}

export type IconCatalogPage = {
  collection: IconCollectionId
  iconifyPrefix: 'lucide' | 'tabler'
  page: number
  pageSize: number
  total: number
  hasMore: boolean
  items: IconPickerItem[]
}

const catalogs: Record<IconCollectionId, IconCatalogConfig> = {
  tabler: createCatalog('tabler', 'tabler', 'i-tabler-', tablerCollection),
  nuxt: createCatalog('nuxt', 'lucide', 'i-lucide-', lucideCollection)
}

export function isIconCatalogCollection(value: unknown): value is IconCollectionId {
  return value === 'tabler' || value === 'nuxt'
}

export function getIconCatalogPage(collection: IconCollectionId, params: IconCatalogPageParams = {}): IconCatalogPage {
  const catalog = catalogs[collection]
  const query = normalizeCatalogQuery(params.q)
  const page = positiveInteger(params.page, 1)
  const pageSize = clamp(positiveInteger(params.pageSize, ICON_PICKER_PAGE_SIZE), 1, ICON_PICKER_MAX_PAGE_SIZE)
  const names = query ? catalog.names.filter((name) => matchesIconQuery(catalog, name, query)) : catalog.names
  const start = (page - 1) * pageSize
  const items = names.slice(start, start + pageSize).map((name) => toIconPickerItem(catalog, name))

  return {
    collection,
    iconifyPrefix: catalog.iconifyPrefix,
    page,
    pageSize,
    total: names.length,
    hasMore: start + items.length < names.length,
    items
  }
}

function createCatalog(
  id: IconCollectionId,
  iconifyPrefix: 'lucide' | 'tabler',
  nuxtPrefix: string,
  collection: IconCollectionData
): IconCatalogConfig {
  return {
    id,
    iconifyPrefix,
    nuxtPrefix,
    names: Object.keys(collection.icons).sort((a, b) => a.localeCompare(b))
  }
}

function toIconPickerItem(catalog: IconCatalogConfig, name: string): IconPickerItem {
  return {
    name: `${catalog.nuxtPrefix}${name}`,
    label: name,
    keywords: [
      `${catalog.iconifyPrefix}:${name}`,
      name.replaceAll('-', ' ')
    ]
  }
}

function matchesIconQuery(catalog: IconCatalogConfig, name: string, query: string) {
  const searchText = [
    name,
    name.replaceAll('-', ' '),
    `${catalog.nuxtPrefix}${name}`,
    `${catalog.iconifyPrefix}:${name}`
  ].join(' ')

  return searchText.includes(query)
}

function normalizeCatalogQuery(value: unknown) {
  return String(Array.isArray(value) ? value[0] || '' : value || '')
    .trim()
    .toLowerCase()
}

function positiveInteger(value: unknown, fallback: number) {
  const parsed = Number.parseInt(String(Array.isArray(value) ? value[0] : value), 10)
  return Number.isFinite(parsed) && parsed > 0 ? parsed : fallback
}

function clamp(value: number, min: number, max: number) {
  return Math.min(Math.max(value, min), max)
}
