import { describe, expect, test } from 'bun:test'

import {
  collectionFromName,
  dedupeIconItems,
  nextIconVisibleCount,
  normalizeIconName
} from '../../app/utils/iconPicker'
import { getIconCatalogPage } from '../../server/utils/iconCatalog'

describe('icon catalog paging', () => {
  test('returns Tabler icons from the full local collection', () => {
    const firstPage = getIconCatalogPage('tabler', { page: 1, pageSize: 24 })

    expect(firstPage.collection).toBe('tabler')
    expect(firstPage.total).toBeGreaterThan(1000)
    expect(firstPage.items).toHaveLength(24)
    expect(firstPage.items[0]?.name).toStartWith('i-tabler-')
    expect(firstPage.hasMore).toBe(true)
  })

  test('searches the Nuxt Icon/Lucide collection by icon name', () => {
    const page = getIconCatalogPage('nuxt', { q: 'settings', page: 1, pageSize: 30 })

    expect(page.total).toBeGreaterThan(0)
    expect(page.items.every((item) => item.name.startsWith('i-lucide-'))).toBe(true)
    expect(page.items.some((item) => item.label.includes('settings'))).toBe(true)
  })

  test('clamps invalid page input to a stable first page', () => {
    const page = getIconCatalogPage('tabler', { page: -9, pageSize: 500 })

    expect(page.page).toBe(1)
    expect(page.pageSize).toBe(120)
    expect(page.items).toHaveLength(120)
  })
})

describe('icon picker helpers', () => {
  test('normalizes saved icon names for Tabler, Iconify, and bare values', () => {
    expect(normalizeIconName('database', 'i-tabler-')).toBe('i-tabler-database')
    expect(normalizeIconName('tabler:database', 'i-tabler-')).toBe('i-tabler-database')
    expect(normalizeIconName('i-lucide-settings-2', 'i-tabler-')).toBe('i-lucide-settings-2')
    expect(collectionFromName('i-tabler-database')).toBe('tabler')
    expect(collectionFromName('i-lucide-settings-2')).toBe('nuxt')
  })

  test('deduplicates paged icon options while preserving order', () => {
    const items = dedupeIconItems([
      { name: 'i-tabler-database', label: 'database', keywords: [] },
      { name: 'i-tabler-database', label: 'database', keywords: [] },
      { name: 'i-tabler-settings', label: 'settings', keywords: [] }
    ])

    expect(items.map((item) => item.name)).toEqual(['i-tabler-database', 'i-tabler-settings'])
  })

  test('calculates the next visible count for scroll pagination', () => {
    expect(nextIconVisibleCount(0, 6194, 60)).toBe(60)
    expect(nextIconVisibleCount(6180, 6194, 60)).toBe(6194)
    expect(nextIconVisibleCount(10, 0, 60)).toBe(0)
  })
})
