import { describe, expect, test } from 'bun:test'
import { readFileSync } from 'node:fs'
import {
  buildForumHomeQuery,
  forumHomeFeedKey,
  parseForumHomeQuery
} from '../app/utils/forumHome'

describe('forum homepage query helpers', () => {
  test('normalizes scalar route query values and ignores arrays', () => {
    expect(parseForumHomeQuery({ q: '  nuxt  ', category: 'dev', tag: ['go'] }))
      .toEqual({ query: 'nuxt', categorySlug: 'dev', tagSlug: '' })
  })

  test('normalizes null and array route query values to empty filters', () => {
    expect(parseForumHomeQuery({ q: null, category: ['dev'], tag: null }))
      .toEqual({ query: '', categorySlug: '', tagSlug: '' })
  })

  test('omits empty filters when building route query', () => {
    expect(buildForumHomeQuery({ query: '', categorySlug: 'dev', tagSlug: '' }))
      .toEqual({ category: 'dev' })
  })

  test('omits filters containing only whitespace', () => {
    expect(buildForumHomeQuery({ query: '  ', categorySlug: '\n', tagSlug: ' \t ' }))
      .toEqual({})
  })

  test('round-trips committed filters', () => {
    const filters = { query: '搜索', categorySlug: '开发', tagSlug: 'nuxt' }
    expect(parseForumHomeQuery(buildForumHomeQuery(filters))).toEqual(filters)
  })

  test('changes the feed key when each committed filter changes', () => {
    const filters = { query: 'nuxt', categorySlug: 'dev', tagSlug: 'vue' }
    expect(forumHomeFeedKey(filters)).not.toBe(forumHomeFeedKey({ ...filters, query: 'go' }))
    expect(forumHomeFeedKey(filters)).not.toBe(forumHomeFeedKey({ ...filters, categorySlug: 'support' }))
    expect(forumHomeFeedKey(filters)).not.toBe(forumHomeFeedKey({ ...filters, tagSlug: 'go' }))
  })
})

describe('SFSearch contract', () => {
  test('preserves v-model and filter events while exposing an accessible search form', () => {
    const source = readFileSync(new URL('../app/components/SFSearch.vue', import.meta.url), 'utf8')

    expect(source).toContain('ariaLabel?: string')
    expect(source).toContain("'submit': [value: string]")
    expect(source).toContain('<form role="search"')
    expect(source).toContain('@submit.prevent')
    expect(source).toContain("emit('submit', modelValue.trim())")
    expect(source).toContain('kbd: undefined')
    expect(source).toContain("'update:modelValue': [value: string]")
    expect(source).toContain("'update:selectedFilter': [value: string]")
    expect(source).toContain(':aria-label="ariaLabel || placeholder"')
    expect(source).toContain('class="sf-search__filters" aria-label="搜索过滤" role="group"')
    expect(source).toContain(':aria-pressed="selectedFilter === filter.value"')
  })
})
