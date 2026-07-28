import { describe, expect, test } from 'bun:test'
import { readFileSync } from 'node:fs'
import {
  parsePublicPage,
  publicPageLocation,
  publicPagePath
} from '../../app/utils/publicPagination'

describe('public pagination URL helpers', () => {
  test('accepts canonical positive integer pages only', () => {
    expect(parsePublicPage('2')).toBe(2)
    for (const value of [undefined, null, ['2'], '0', '-1', '1.5', ' 2 ', '01', 'x']) {
      expect(parsePublicPage(value)).toBe(1)
    }
  })

  test('omits page one and preserves declared query values', () => {
    expect(publicPageLocation('/c/general', 1)).toEqual({ path: '/c/general' })
    expect(publicPageLocation('/', 2, { category: 'general' })).toEqual({
      path: '/',
      query: { category: 'general', page: '2' }
    })
  })

  test('builds an SSR canonical path with encoded query values', () => {
    expect(publicPagePath('/', 2)).toBe('/?page=2')
    expect(publicPagePath('/', 3, { q: 'Go 语言' })).toBe('/?q=Go+%E8%AF%AD%E8%A8%80&page=3')
    expect(publicPagePath('/search', 1, { q: '啊' })).toBe('/search?q=%E5%95%8A')
  })
})

test('SFPagination supports crawler-visible links without changing button consumers', () => {
  const source = readFileSync(new URL('../../app/components/SFPagination.vue', import.meta.url), 'utf8')
  expect(source).toContain('pageTo?:')
  expect(source).toContain('<NuxtLink')
  expect(source).toContain(':to="linkTo(page + 1)"')
  expect(source).toContain("emit('update:page'")
})
