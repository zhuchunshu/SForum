import { describe, expect, test } from 'bun:test'
import { readFileSync } from 'node:fs'

const source = (path: string) => readFileSync(new URL(path, import.meta.url), 'utf8')
const homepage = () => source('../../../extensions/builtin/themes/sforum-default/layer/app/pages/index.vue')
const topicRow = () => source('../../../extensions/builtin/themes/sforum-default/layer/app/components/SFHomeTopicRow.vue')
const homeNav = () => source('../../../extensions/builtin/themes/sforum-default/layer/app/components/SFHomeNavigation.vue')
const defaultLayout = () => source('../../../extensions/builtin/themes/sforum-default/layer/app/layouts/default.vue')
const layerConfig = () => source('../../../extensions/builtin/themes/sforum-default/layer/nuxt.config.ts')
const homepageCss = () => source('../../../extensions/builtin/themes/sforum-default/layer/app/assets/css/sforum-home.css')
const themeCss = () => source('../../../extensions/builtin/themes/sforum-default/layer/app/assets/css/sforum-theme.css')
const footer = () => source('../../../extensions/builtin/themes/sforum-default/layer/app/components/SFFooter.vue')
const homeQueryCacheMiddleware = () => source('../server/middleware/home-query-cache.ts')

describe('default theme V32 left-nav homepage contract', () => {
  test('uses the shared public layout with left sidebar and topic table', () => {
    const page = homepage()

    expect(page).toContain('class="sforum-home"')
    expect(page).toContain('class="sforum-home__layout"')
    expect(page).toContain('class="sforum-home__sidebar"')
    expect(page).toContain('class="sforum-home__main"')
    expect(page).toContain('class="sforum-home__topic-table"')
    expect(page).toContain('class="sforum-home__notice"')
    expect(page).toContain('<SFHomeTopicRow')
    expect(page).toContain('<SFHomeNavigation')
    expect(page).toContain('desktop-only')
    expect(page).toContain('mobile-only')
    expect(page).toContain("t('home.filter.latest')")
    expect(page).not.toContain('layout: false')
    expect(page).not.toContain('sforum-home__hero')
    expect(page).not.toContain('sforum-home__aside')
    expect(page).not.toContain('<SFFooter')
    expect(page).not.toContain('participants')
    expect(page).not.toContain('latestTopics')
    expect(page).not.toContain('loadedReplyTotal')

    const layout = defaultLayout()
    expect(layout).toContain('<SFNavbar />')
    expect(layout).toContain('<SFFooter />')
  })

  test('renders topic table rows using only API-backed summary data', () => {
    const row = topicRow()

    expect(row).toContain('topic: ForumTopicSummary')
    expect(row).toContain('to: string')
    expect(row).toContain('activityLabel: string')
    expect(row).toContain('topic.commentCount')
    expect(row).toContain('topic.categoryName')
    expect(row).toContain('topic.author')
    expect(row).toContain('topic.isPinned')
    expect(row).toContain('SFAvatar')
    expect(row).toContain(':avatar="topic.author?.avatar"')
    expect(row).not.toContain('topic.excerpt')
    expect(row).not.toContain('participants')
    expect(row).not.toContain('$fetch')
    expect(row).not.toContain('useForumApi')
  })

  test('left navigation exposes compose, all topics, and category counts', () => {
    const nav = homeNav()

    expect(nav).toContain('sf-home-navigation__compose')
    expect(nav).toContain('sf-home-navigation__cat-dot')
    expect(nav).toContain('category.topicCount')
    expect(nav).toContain("t('home.allTopics')")
    expect(nav).toContain("t('home.sidebar.newTopic')")
    expect(nav).toContain('canCreateTopic')
    expect(nav).toContain("navigationMode?: 'filter' | 'route'")
    expect(nav).toContain('forumCategoryPath')
    expect(nav).not.toContain('unread')
    expect(nav).not.toContain('ranking')
  })

  test('category and tag pages reuse the V32 left-nav topic table shell', () => {
    const categoryPage = source('../../../extensions/builtin/themes/sforum-default/layer/app/pages/c/[categorySlug].vue')
    const tagPage = source('../../../extensions/builtin/themes/sforum-default/layer/app/pages/tags/[tagSlug].vue')

    for (const page of [categoryPage, tagPage]) {
      expect(page).toContain('class="sforum-home"')
      expect(page).toContain('class="sforum-home__layout"')
      expect(page).toContain('navigation-mode="route"')
      expect(page).toContain('<SFHomeTopicRow')
      expect(page).toContain('sforum-home__topic-table')
      expect(page).not.toContain('sf-public-page')
      expect(page).not.toContain('bg-[#E6F4F1]')
    }
  })

  test('commits filters through the URL with a debounced search draft', () => {
    const page = homepage()

    expect(page).toContain('parseForumHomeQuery')
    expect(page).toContain('buildForumHomeQuery')
    expect(page).toContain('forumHomeFeedKey')
    expect(page).toContain('const committedFilters = computed')
    expect(page).toContain('const searchDraft = ref')
    expect(page).toContain('const SEARCH_DEBOUNCE_MS = 300')
    expect(page).toContain('setTimeout')
    expect(page).toContain('SEARCH_DEBOUNCE_MS')
    expect(page).toContain('router.replace({')
    expect(page).toContain("path: localePath('/'),")
    expect(page).toContain('query: buildForumHomeQuery(nextFilters)')
    expect(page).toContain('function resetFilters()')
    expect(page).toContain('@select-category="selectCategory"')
    expect(page).toContain('@click="selectTag(tag.slug)"')
  })

  test('preserves SSR hydration, stale-response, deduplication, and infinite-loading guards', () => {
    const page = homepage()
    const feedReset = page.slice(
      page.indexOf('watch(activeFeedKey'),
      page.indexOf('async function loadMoreTopics')
    )

    expect(page).toContain("useState<ForumTopicSummary[]>('forum-home-loaded-topics'")
    expect(page).toContain('() => topicList.value.items')
    expect(page).toContain('() => topicList.value.total')
    expect(page).toContain("'forum-home-loaded-feed-key'")
    expect(page).toContain('shouldIgnoreClientEmptyHydration')
    expect(page).toContain('isForumHomeRequestCurrent')
    expect(page).toContain('hasReachedForumHomeEnd')
    expect(page).toContain('IntersectionObserver')
    expect(page).toContain('feedGeneration')
    expect(feedReset).toContain('feedGeneration += 1')
    expect(page).toContain('existingIds')
  })

  test('registers V32 left-nav homepage stylesheet tokens', () => {
    const config = layerConfig()
    const css = homepageCss()
    const theme = themeCss()

    expect(config).toContain('sforum-home.css')
    expect(css).toContain('grid-template-columns: var(--sf-public-sidebar-width) minmax(0, 1fr)')
    expect(css).toContain('.sforum-home__topic-table')
    expect(css).toContain('.sf-home-topic-row')
    expect(css).toContain('.sforum-home__notice')
    expect(css).toContain('var(--sf-public-row-hover)')
    expect(css).not.toContain('sforum-home__hero')
    expect(theme).toContain('--sf-public-sidebar-width')
    expect(theme).toContain('--sf-public-bg: #eef2f6')
  })

  test('keeps homepage query-cache middleware for root route payload safety', () => {
    const middleware = homeQueryCacheMiddleware()
    expect(middleware).toContain('no-store')
  })

  test('footer remains in the shared layout shell', () => {
    expect(footer()).toContain('sf-footer')
  })
})
