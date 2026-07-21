import { describe, expect, test } from 'bun:test'
import { readFileSync } from 'node:fs'
import * as forumHome from '../app/utils/forumHome'
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

  test('rejects an old request when filters cycle from A to B and back to A', async () => {
    const isRequestCurrent = (forumHome as Record<string, unknown>).isForumHomeRequestCurrent
    expect(typeof isRequestCurrent).toBe('function')
    if (typeof isRequestCurrent !== 'function') return

    let resolveRequest!: () => void
    const pending = new Promise<void>((resolve) => {
      resolveRequest = resolve
    })
    let generation = 0
    let activeFeedKey = 'A'
    const applied: string[] = []
    const oldRequest = { generation, feedKey: activeFeedKey }

    const applyOldRequest = (async () => {
      await pending
      if (isRequestCurrent(oldRequest, generation, activeFeedKey)) {
        applied.push('old A')
      }
    })()

    generation += 1
    activeFeedKey = 'B'
    generation += 1
    activeFeedKey = 'A'
    resolveRequest()
    await applyOldRequest

    expect(applied).toEqual([])
    expect(isRequestCurrent({ generation, feedKey: 'A' }, generation, activeFeedKey)).toBe(true)
  })

  test('ends pagination when the backend clamps or a full page adds no new topics', () => {
    const hasReachedEnd = (forumHome as Record<string, unknown>).hasReachedForumHomeEnd
    expect(typeof hasReachedEnd).toBe('function')
    if (typeof hasReachedEnd !== 'function') return

    expect(hasReachedEnd({
      requestedPage: 201,
      responsePage: 200,
      responseItemCount: 10,
      newItemCount: 0,
      loadedCount: 2000,
      total: 6001,
      perPage: 10
    })).toBe(true)

    expect(hasReachedEnd({
      requestedPage: 2,
      responsePage: 2,
      responseItemCount: 10,
      newItemCount: 0,
      loadedCount: 10,
      total: 100,
      perPage: 10
    })).toBe(true)

    expect(hasReachedEnd({
      requestedPage: 2,
      responsePage: 2,
      responseItemCount: 10,
      newItemCount: 10,
      loadedCount: 20,
      total: 100,
      perPage: 10
    })).toBe(false)

    // M5：API hasMore=false 立即结束（不依赖 total）
    expect(hasReachedEnd({
      requestedPage: 2,
      responsePage: 1,
      responseItemCount: 10,
      newItemCount: 10,
      loadedCount: 30,
      total: 1_000_000,
      perPage: 10,
      hasMore: false
    })).toBe(true)

    expect(hasReachedEnd({
      requestedPage: 2,
      responsePage: 1,
      responseItemCount: 10,
      newItemCount: 10,
      loadedCount: 30,
      total: 1_000_000,
      perPage: 10,
      hasMore: true
    })).toBe(false)
  })

  test('home page prefers nextCursor for infinite scroll load-more', () => {
    const source = readFileSync(new URL('../app/components/SFHomePage.vue', import.meta.url), 'utf8')
    expect(source).toContain('nextCursor')
    expect(source).toContain('after')
    expect(source).toContain('hasMore')
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
