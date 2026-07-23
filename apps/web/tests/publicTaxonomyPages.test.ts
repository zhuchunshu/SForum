import { describe, expect, test } from 'bun:test'
import { readFileSync } from 'node:fs'

// 呈现已迁到 body 岛；路由页仅为 SEO/outlet 壳。
const tagsPage = () => readFileSync(
  new URL('../../../apps/web/app/components/SFTagIndexPage.vue', import.meta.url),
  'utf8'
)
const categoriesPage = () => readFileSync(
  new URL('../../../apps/web/app/components/SFCategoryIndexPage.vue', import.meta.url),
  'utf8'
)
const tagsRoute = () => readFileSync(
  new URL('../../../apps/web/app/pages/tags/index.vue', import.meta.url),
  'utf8'
)
const categoriesRoute = () => readFileSync(
  new URL('../../../apps/web/app/pages/categories/index.vue', import.meta.url),
  'utf8'
)
const taxonomyCss = () => readFileSync(
  new URL('../../../apps/web/app/assets/css/sforum-taxonomy.css', import.meta.url),
  'utf8'
)
const navbar = () => readFileSync(
  new URL('../../../apps/web/app/components/SFNavbar.vue', import.meta.url),
  'utf8'
)
const hostNuxtConfig = () => readFileSync(new URL('../nuxt.config.ts', import.meta.url), 'utf8')
const zh = () => JSON.parse(readFileSync(new URL('../i18n/locales/zh-CN.json', import.meta.url), 'utf8'))
const en = () => JSON.parse(readFileSync(new URL('../i18n/locales/en-US.json', import.meta.url), 'utf8'))

describe('public taxonomy list pages (T01 + C04)', () => {
  test('taxonomy routes are thin outlet shells with island fallbacks', () => {
    expect(tagsRoute()).toContain('SFPageOutlet')
    expect(tagsRoute()).toContain('page="forum.tag.index"')
    expect(tagsRoute()).toContain('<SFTagIndexPage')
    expect(tagsRoute()).not.toContain('sforum-taxonomy__cloud')
    expect(categoriesRoute()).toContain('SFPageOutlet')
    expect(categoriesRoute()).toContain('page="forum.category.index"')
    expect(categoriesRoute()).toContain('<SFCategoryIndexPage')
    expect(categoriesRoute()).not.toContain('sforum-taxonomy__tile')
  })

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

  test('categories index uses category groups, directory layout, focus, filter, and category detail links', () => {
    const source = categoriesPage()
    expect(source).toContain('forumApi.listCategoryGroups()')
    expect(source).toContain('visibleCategoryDirectoryGroups')
    expect(source).toContain('buildCategoryDirectoryDisplayGroups')
    expect(source).toContain('summarizeCategoryDirectory')
    expect(source).toContain('activeCategoryDirectoryCategories')
    expect(source).toContain('forumCategoryPath(category.slug)')
    expect(source).toContain('sforum-category-directory__board')
    expect(source).toContain('sforum-home__layout--with-right')
    expect(source).toContain('SFHomeNavigation')
    expect(source).toContain('mobileMenuOpen')
    expect(source).toContain('mobileInfoOpen')
    expect(source).toContain('route.query.group')
    expect(source).toContain('filterDraft')
    expect(source).toContain("sort === 'default'")
    expect(source).toContain("sort === 'active'")
    expect(source).toContain("sort === 'name'")
    expect(source).toContain('category.iconColor')
    expect(source).not.toContain('latestReply')
    expect(source).not.toContain('viewCount')
    expect(source).not.toContain('growth')
  })

  test('theme registers taxonomy CSS and host routeRules cover list roots', () => {
    expect(hostNuxtConfig()).toContain('sforum-taxonomy.css')
    expect(taxonomyCss()).toContain('.sforum-taxonomy__cloud')
    expect(taxonomyCss()).toContain('.sforum-taxonomy__tile')
    expect(taxonomyCss()).toContain('.sforum-category-directory__board-grid')
    expect(taxonomyCss()).toContain('.sforum-category-directory__facts')
    expect(hostNuxtConfig()).toContain("'/categories': { swr: 600 }")
    expect(hostNuxtConfig()).toContain("'/tags': { swr: 600 }")
  })

  test('topbar search remains the global search entry', () => {
    const source = navbar()
    expect(source).toContain('<SFSearch')
    expect(source).toContain("t('home.searchPlaceholder')")
    expect(source).toContain('@submit="submitSearch"')
    expect(source).toContain("path: localePath('/')")
    expect(source).toContain('buildForumHomeQuery')
    expect(source).not.toContain('filterDraft')
    expect(source).not.toContain('taxonomy.categories.filterPlaceholder')
  })

  test('bilingual taxonomy copy is present', () => {
    expect(zh().taxonomy.tags.title).toBe('全部标签')
    expect(zh().taxonomy.categories.title).toBe('全部分类')
    expect(zh().taxonomy.categories.sorts.all).toBe('默认顺序')
    expect(zh().taxonomy.categories.filterPlaceholder).toContain('筛选分类')
    expect(en().taxonomy.tags.title).toBe('All tags')
    expect(en().taxonomy.categories.title).toBe('All categories')
    expect(en().taxonomy.categories.sorts.all).toBe('Default')
    expect(en().taxonomy.categories.filterPlaceholder).toContain('Filter category')
  })

  test('theme L1 taxonomy shells mark presentation ownership and chrome islands', () => {
    const root = new URL('../../../extensions/builtin/themes/', import.meta.url)
    for (const theme of ['sforum-default', 'sforum-nocturne']) {
      for (const [file, pageId] of [
        ['tag-index.html', 'forum.tag.index'],
        ['category-index.html', 'forum.category.index'],
        ['tag-show.html', 'forum.tag.show'],
        ['category-show.html', 'forum.category.show'],
      ] as const) {
        const tpl = readFileSync(new URL(`${theme}/templates/${file}`, root), 'utf8')
        expect(tpl).toContain('data-theme-owned="presentation"')
        expect(tpl).toContain(`data-page="${pageId}"`)
        expect(tpl).toContain('<sf-navbar')
        expect(tpl).toContain('<sf-footer')
      }
    }
  })
})
