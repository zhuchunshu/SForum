import { describe, expect, test } from 'bun:test'
import { readFileSync } from 'node:fs'

const tagsPage = () => readFileSync(
  new URL('../../../apps/web/app/pages/tags/index.vue', import.meta.url),
  'utf8'
)
const categoriesPage = () => readFileSync(
  new URL('../../../apps/web/app/pages/categories/index.vue', import.meta.url),
  'utf8'
)
const taxonomyCss = () => readFileSync(
  new URL('../../../apps/web/app/assets/css/sforum-taxonomy.css', import.meta.url),
  'utf8'
)
const hostNuxtConfig = () => readFileSync(new URL('../nuxt.config.ts', import.meta.url), 'utf8')
const zh = () => JSON.parse(readFileSync(new URL('../i18n/locales/zh-CN.json', import.meta.url), 'utf8'))
const en = () => JSON.parse(readFileSync(new URL('../i18n/locales/en-US.json', import.meta.url), 'utf8'))

describe('public taxonomy list pages (T01 + C04)', () => {
  test('tags index uses listTags, public_pages gate, cloud buckets, and tag detail links', () => {
    const source = tagsPage()
    expect(source).toContain('forumApi.listTags()')
    expect(source).toContain("webOption('forum.tags.public_pages'")
    expect(source).toContain('parseForumTagPublicPagesOption')
    expect(source).toContain('tagCloudSizeBucket')
    expect(source).toContain('tagHotThreshold')
    expect(source).toContain('forumTagPath(tag.slug)')
    expect(source).toContain("filter === 'all'")
    expect(source).toContain("filter === 'hot'")
    expect(source).toContain("filter === 'week'")
    expect(source).toContain("filter === 'az'")
    expect(source).toContain('sforum-taxonomy__cloud')
  })

  test('categories index uses category groups, visibility filter, and category detail links', () => {
    const source = categoriesPage()
    expect(source).toContain('forumApi.listCategoryGroups()')
    expect(source).toContain("visibility !== 'hidden'")
    expect(source).toContain('forumCategoryPath(category.slug)')
    expect(source).toContain('sforum-taxonomy__tile')
    expect(source).toContain("sort === 'default'")
    expect(source).toContain("sort === 'active'")
    expect(source).toContain("sort === 'name'")
    expect(source).toContain('iconColor')
  })

  test('theme registers taxonomy CSS and host routeRules cover list roots', () => {
    expect(hostNuxtConfig()).toContain('sforum-taxonomy.css')
    expect(taxonomyCss()).toContain('.sforum-taxonomy__cloud')
    expect(taxonomyCss()).toContain('.sforum-taxonomy__tile')
    expect(hostNuxtConfig()).toContain("'/categories': { swr: 600 }")
    expect(hostNuxtConfig()).toContain("'/tags': { swr: 600 }")
  })

  test('bilingual taxonomy copy is present', () => {
    expect(zh().taxonomy.tags.title).toBe('全部标签')
    expect(zh().taxonomy.categories.title).toBe('全部分类')
    expect(en().taxonomy.tags.title).toBe('All tags')
    expect(en().taxonomy.categories.title).toBe('All categories')
  })
})
