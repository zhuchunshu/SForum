import { describe, expect, test } from 'bun:test'
import { readFileSync } from 'node:fs'

const source = (path: string) => readFileSync(new URL(path, import.meta.url), 'utf8')
const homepage = () => source('../../../extensions/builtin/themes/sforum-default/layer/app/pages/index.vue')
const navigation = () => source('../../../extensions/builtin/themes/sforum-default/layer/app/components/SFHomeNavigation.vue')
const topicRow = () => source('../../../extensions/builtin/themes/sforum-default/layer/app/components/SFHomeTopicRow.vue')
const defaultLayout = () => source('../../../extensions/builtin/themes/sforum-default/layer/app/layouts/default.vue')
const layerConfig = () => source('../../../extensions/builtin/themes/sforum-default/layer/nuxt.config.ts')
const homepageCss = () => source('../../../extensions/builtin/themes/sforum-default/layer/app/assets/css/sforum-home.css')
const themeCss = () => source('../../../extensions/builtin/themes/sforum-default/layer/app/assets/css/sforum-theme.css')
const footer = () => source('../../../extensions/builtin/themes/sforum-default/layer/app/components/SFFooter.vue')
const seoComposable = () => source('../app/composables/useSForumSeo.ts')
const homeQueryCacheMiddleware = () => source('../server/middleware/home-query-cache.ts')

function relativeLuminance(hex: string) {
  const channels = hex.slice(1).match(/.{2}/g)?.map((part) => Number.parseInt(part, 16) / 255) || []
  const [red = 0, green = 0, blue = 0] = channels.map((channel) => (
    channel <= 0.04045 ? channel / 12.92 : ((channel + 0.055) / 1.055) ** 2.4
  ))
  return 0.2126 * red + 0.7152 * green + 0.0722 * blue
}

function contrastRatio(foreground: string, background: string) {
  const light = Math.max(relativeLuminance(foreground), relativeLuminance(background))
  const dark = Math.min(relativeLuminance(foreground), relativeLuminance(background))
  return (light + 0.05) / (dark + 0.05)
}

describe('default theme hybrid homepage contract', () => {
  test('uses the shared public layout without duplicate or fake chrome', () => {
    const page = homepage()

    expect(page).toContain('class="sforum-home"')
    expect(page).toContain('<SFHomeNavigation')
    expect(page).toContain('<SFHomeTopicRow')
    expect(page).toContain('class="sforum-home__dock"')
    expect(page).toContain("t('home.loginToPost')")
    expect(page).toContain('dockTopics')
    expect(page).toContain('loadedReplyTotal')
    expect(page).not.toContain('layout: false')
    expect(page).not.toContain('sforum-home__topbar')
    expect(page).not.toContain('<SFFooter')
    expect(page).not.toContain('topicReplyStackLabel')
    expect(page).not.toContain('participants')
    expect(page).not.toContain('disabled')

    const layout = defaultLayout()
    expect(layout).toContain('<SFNavbar />')
    expect(layout).toContain('<SFFooter />')
  })

  test('renders typed navigation and topic rows using only API-backed data', () => {
    const nav = navigation()
    const row = topicRow()

    expect(nav).toContain('categories: ForumCategory[]')
    expect(nav).toContain('selectedCategorySlug: string')
    expect(nav).toContain('totalTopics: number')
    expect(nav).toContain("'select-category': [slug: string]")
    expect(nav).toContain("emit('select-category',")
    expect(nav).toContain('category.topicCount')
    expect(nav).not.toContain('ForumCategoryGroup')

    expect(row).toContain('topic: ForumTopicSummary')
    expect(row).toContain('to: string')
    expect(row).toContain('activityLabel: string')
    expect(row).toContain('topic.excerpt')
    expect(row).toContain('topic.commentCount')
    expect(row).not.toContain('participants')
    expect(row).not.toContain('$fetch')
    expect(row).not.toContain('useForumApi')
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
    expect(page).toContain('activeFeedKey.value === loadedFeedKey.value')
    expect(page).toContain('let feedGeneration = 0')
    expect(page).toContain('isForumHomeRequestCurrent')
    expect(page).toContain('hasReachedForumHomeEnd')
    expect(page).toContain('const topicDataKey = computed')
    expect(page).not.toContain('resolvedFirstPageFeedKey')
    expect(page).toContain('new Set(loadedTopics.value.map((topic) => topic.id))')
    expect(page).toContain('IntersectionObserver')
    expect(page).toContain('loadMoreTrigger')
    expect(page).toContain('loadMoreTopics(true)')
    expect(page).not.toContain('<SFPagination')
    expect(feedReset).toContain('isLoadingMore.value = false')
    expect(feedReset).toContain('feedGeneration += 1')
    expect(feedReset).toContain("{ flush: 'sync' }")
    expect(page).toContain('nextPage.value = nextList.page + 1')
    expect(page).not.toContain('perPage: ITEMS_PER_PAGE')
    expect(page).toContain('nextList.perPage')
  })

  test('uses one SSR reference time and UTC dates for hydration-stable activity labels', () => {
    const page = homepage()

    expect(page).toContain("useState<number>('forum-home-rendered-at'")
    expect(page).toContain('renderedAt.value - date.getTime()')
    expect(page).toContain('date.toISOString().slice(0, 10)')
    expect(page).not.toContain('Date.now() - date.getTime()')
  })

  test('uses the working homepage query for schema.org search', () => {
    const source = seoComposable()

    expect(source).toContain('/?q={search_term_string}')
    expect(source).not.toContain('/search?q={search_term_string}')
  })

  test('keeps the base homepage on SWR while query variants render without payload caching', () => {
    const config = source('../nuxt.config.ts')
    const middleware = homeQueryCacheMiddleware()

    expect(config).toContain("'/': publicHomepageRouteRule")
    expect(config).toContain("cache: false")
    expect(config).toContain("'cache-control': 's-maxage=600, stale-while-revalidate'")
    expect(middleware).toContain("url.pathname !== '/' && url.pathname !== '/en'")
    expect(middleware).toContain('!url.search')
    expect(middleware).toContain('routeRules.cache = false')
    expect(middleware).toContain('routeRules.swr = false')
    expect(middleware).toContain("setHeader(event, 'cache-control', 'no-store')")
  })

  test('registers the responsive C workbench visual system', () => {
    const config = layerConfig()
    const home = homepageCss()
    const theme = themeCss()
    const mobileStart = home.indexOf('@media (max-width: 700px)')
    const reducedMotionStart = home.indexOf('@media (prefers-reduced-motion: reduce)')
    const mobileStyles = home.slice(mobileStart, reducedMotionStart)
    const tagListStyles = home.match(/\.sforum-home__tag-list \{[\s\S]*?\n\}/)?.[0] || ''

    expect(config).toContain('sforum-home.css')
    expect(home).toContain('grid-template-columns: 74px minmax(0, 1fr) 310px;')
    expect(home).toMatch(/\.sforum-home__inner\s*\{[\s\S]*?padding: 0 19px 40px;/)
    expect(home).toMatch(/\.sf-home-navigation\s*\{[\s\S]*?top: 0;[\s\S]*?height: calc\(100vh - 55px\);/)
    expect(home).toContain('.sforum-home__dock')
    expect(home).toContain('.sf-home-topic-row__heat')
    expect(home).toContain('.sf-home-topic-row__metric')
    expect(home).toContain('--sforum-home-secondary: #bd5b43;')
    expect(home).toContain('.sforum-home .sforum-home__heading h1')
    expect(home).toContain('min-height: 40px;')
    expect(home).toContain('overflow-wrap: anywhere;')
    expect(home).toContain('prefers-reduced-motion: reduce')
    expect(home).toContain('.dark .sforum-home')
    expect(home).not.toContain('grid-template-columns: 208px minmax(0, 1fr);')
    expect(mobileStyles).toContain('.sforum-home__load-error .sf-button')
    expect(mobileStyles).toContain('.sforum-home__infinite-state .sf-button')
    expect(mobileStyles).toContain('.sforum-home__empty .sf-button')
    expect(mobileStyles).toContain('min-height: 40px;')
    expect(tagListStyles).toContain('padding: 4px;')
    expect(home).not.toContain('#0b1120')
    expect(home).not.toContain('#172033')
    expect(theme).not.toContain('#0b1120')
    expect(theme).not.toContain('#172033')
    expect(theme).not.toContain('.sforum-home')
  })

  test('keeps the homepage footer inside the C workbench shell', () => {
    const source = footer()

    expect(source).toContain("route.path === '/' || route.path === '/en'")
    expect(source).toContain("'sf-footer--workbench': isWorkbenchHome")
    expect(source).toContain('.sf-footer--workbench')
    expect(source).toContain('background: #f3f7f6;')
  })

  test('keeps small light-mode metadata at WCAG AA contrast', () => {
    const home = homepageCss()
    const faint = home.match(/--sforum-home-faint:\s*(#[0-9a-f]{6});/i)?.[1]

    expect(faint).toBeString()
    expect(contrastRatio(faint!, '#ffffff')).toBeGreaterThanOrEqual(4.5)
  })

  test('provides every visible hybrid homepage label in both locales', () => {
    for (const file of ['zh-CN.json', 'en-US.json']) {
      const messages = JSON.parse(source(`../i18n/locales/${file}`))

      for (const key of ['allTopics', 'categories', 'tags', 'clearFilters', 'searchResults']) {
        expect(messages.home[key]).toBeString()
        expect(messages.home[key].length).toBeGreaterThan(0)
      }
      expect(messages.home.emptyState.filteredDescription).toBeString()
      expect(messages.home.feed.loadMoreFailed).toBeString()
      expect(messages.home.feed.retryLoadMore).toBeString()
      expect(messages.home.feed.latestActivity).toBeString()
      expect(messages.home.feed.replyCount).toBeString()
      expect(messages.home.filter.all).toBeString()
      expect(messages.home.loginToPost).toBeString()
      expect(messages.home.dock.latestActivity).toBeString()
      expect(messages.home.dock.overview).toBeString()
    }
  })
})
