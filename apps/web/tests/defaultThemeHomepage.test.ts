import { describe, expect, test } from 'bun:test'
import { readFileSync } from 'node:fs'
import { compileScript, parse } from '@vue/compiler-sfc'

const source = (path: string) => readFileSync(new URL(path, import.meta.url), 'utf8')
// 首页呈现已迁到 SFHomePage 岛；pages/index 仅为 SEO + fail-closed 壳。
const homepageIsland = () => source('../app/components/SFHomePage.vue')
const homepageRoute = () => source('../app/pages/index.vue')
const defaultHomeTemplate = () => source('../../../extensions/builtin/themes/sforum-default/templates/home.html')
const nocturneHomeTemplate = () => source('../../../extensions/builtin/themes/sforum-nocturne/templates/home.html')
const themeTemplate = () => source('../app/components/SFThemeTemplate.vue')
const topicRow = () => source('../../../apps/web/app/components/SFHomeTopicRow.vue')
const homeNav = () => source('../../../apps/web/app/components/SFHomeNavigation.vue')
const defaultLayout = () => source('../../../apps/web/app/layouts/default.vue')
const hostConfig = () => source('../nuxt.config.ts')
const homepageCss = () => source('../../../apps/web/app/assets/css/sforum-home.css')
const themeCss = () => source('../../../apps/web/app/assets/css/sforum-theme.css')
const themePackage = () => source('../../../extensions/builtin/themes/sforum-default/theme.json')
const footer = () => source('../../../apps/web/app/components/SFFooter.vue')
const homeQueryCacheMiddleware = () => source('../server/middleware/home-query-cache.ts')
const pluginDocsTemplate = () => source('../../../extensions/fixtures/plugins/page-registry-demo/templates/docs.html')

describe('default theme V32 left-nav homepage contract', () => {
  test('route shell is thin SEO + outlet; presentation lives on SFHomePage island', () => {
    const route = homepageRoute()
    expect(route).toContain('SFPageOutlet')
    expect(route).toContain('page="forum.home"')
    expect(route).toContain('<SFHomePage')
    expect(route).toContain('useSForumSeo')
    expect(route).not.toContain('class="sforum-home"')
    expect(route).not.toContain('sforum-home__topic-table')
    expect(route).not.toContain('loadMoreTopics')

    const template = themeTemplate()
    expect(template).toContain("'forum.component.home_page': resolveComponent('SFHomePage')")
    expect(template).not.toContain("'forum.component.home_page': HostPageIsland")

    const defaultTpl = defaultHomeTemplate()
    expect(defaultTpl).toContain('data-theme-owned="presentation"')
    expect(defaultTpl).toContain('<sf-home-page>')
    expect(defaultTpl).toContain('sf-theme-home-shell')
    expect(defaultTpl).toContain('data-layout="fullwidth-3col"')
    expect(defaultTpl).toContain('sf-theme-shell--fullwidth-3col')

    const nocturneTpl = nocturneHomeTemplate()
    expect(nocturneTpl).toContain('data-theme-owned="presentation"')
    expect(nocturneTpl).toContain('<sf-home-page>')
    expect(nocturneTpl).toContain('nh-hero')
  })

  test('uses the shared public layout with left sidebar and topic table', () => {
    const page = homepageIsland()

    expect(page).toContain('class="sforum-home"')
    expect(page).toContain('sforum-home__layout')
    expect(page).toContain('sforum-home__layout--with-right')
    expect(page).toContain('rightRailEnabled')
    expect(page).toContain('class="sforum-home__sidebar"')
    expect(page).toContain('class="sforum-home__main"')
    expect(page).toContain('data-sf-region="topic-list"')
    expect(page).toContain('useActiveThemeSettings')
    expect(page).toContain('v-if="homeNotice"')
    expect(page).toContain('homeEmptyTitle')
    expect(page).toContain('rightRailHotLimit')
    expect(page).toContain('rightRailTagLimit')
    expect(page).toContain('<SFHomeTopicRow')
    expect(page).toContain('<SFHomeNavigation')
    expect(page).toContain('<SFHomeRightRail')
    expect(page).toContain('v-if="rightRailEnabled"')
    expect(page).toContain(':hot-topics="hotTopics"')
    expect(page).toContain('desktop-only')
    expect(page).toContain('mobile-only')
    expect(page).toContain("t('home.filter.latest')")
    expect(page).toContain('forum-feed-title')
    // 列表呈现：宿主 Tailwind，不再依赖 BEM main-bar / topic-table
    expect(page).not.toContain('sforum-home__main-bar')
    expect(page).not.toContain('sforum-home__topic-table')
    expect(page).not.toContain('layout: false')
    expect(page).not.toContain('sforum-home__hero')
    expect(page).not.toContain('sforum-home__aside')
    expect(page).not.toContain('<SFFooter')
    expect(page).not.toContain('participants')
    expect(page).not.toContain('latestTopics')
    expect(page).not.toContain('loadedReplyTotal')

    // Nuxt default layout 透传；chrome 在主题 L1 与 fail-closed 宿主壳上。
    const layout = defaultLayout()
    expect(layout).toContain('<slot')
    expect(layout).not.toContain('<SFNavbar')
    expect(defaultHomeTemplate()).toContain('<sf-navbar>')
    expect(defaultHomeTemplate()).toContain('<sf-footer>')
    expect(nocturneHomeTemplate()).toContain('<sf-navbar>')
    expect(nocturneHomeTemplate()).toContain('<sf-footer>')
    const hostChrome = source('../app/components/SFHostPublicChrome.vue')
    expect(hostChrome).toContain('<SFNavbar />')
    expect(hostChrome).toContain('layoutShowFooter')
    expect(hostChrome).toContain('layoutShowAnnouncements')
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
    // 列表头像只走全局 SFAvatar（size=list + AvatarView），禁止强制字头/局部改尺寸
    expect(row).toContain('size="list"')
    expect(row).not.toContain('prefer-initials')
    expect(row).not.toContain('!size-9')
    // fullwidth-3col：宿主 Tailwind 行 + 扩展钩子；chip 色板在组件内
    expect(row).toContain('data-sf-component="forum.topic_list_row"')
    expect(row).toContain('forumCategoryChipToneClass')
    expect(row).toContain('grid-cols-[36px_minmax(0,1fr)_88px_72px]')
    expect(row).not.toContain('sf-home-topic-row__chip')
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
    expect(nav).toContain('navShowCompose')
    expect(nav).toContain('navShowCounts')
    expect(nav).toContain('categories?: ForumCategory[]')
    expect(nav).toContain('categories: () => []')
    expect(nav).toContain("selectedCategorySlug: ''")
    expect(nav).toContain('totalTopics: 0')
    expect(nav).toContain("navigationMode?: 'filter' | 'route'")
    expect(nav).toContain('forumCategoryPath')
    expect(nav).not.toContain('unread')
    expect(nav).not.toContain('ranking')
  })

  test('legacy theme navigation islands compile with safe empty defaults', () => {
    const { descriptor, errors } = parse(homeNav(), { filename: 'SFHomeNavigation.vue' })
    expect(errors).toHaveLength(0)

    const compiled = compileScript(descriptor, { id: 'sf-home-navigation' }).content
    expect(pluginDocsTemplate()).toContain('<sf-home-navigation></sf-home-navigation>')
    expect(compiled).toContain('categories: { type: Array, required: false, default: () => [] }')
    expect(compiled).toContain("selectedCategorySlug: { type: String, required: false, default: '' }")
    expect(compiled).toContain('totalTopics: { type: Number, required: false, default: 0 }')
  })

  test('category and tag pages reuse the V32 left-nav topic table shell', () => {
    const categoryPage = source('../app/components/SFCategoryShowPage.vue')
    const tagPage = source('../app/components/SFTagShowPage.vue')

    for (const page of [categoryPage, tagPage]) {
      expect(page).toContain('class="sforum-home"')
      expect(page).toContain('class="sforum-home__layout"')
      expect(page).toContain('navigation-mode="route"')
      expect(page).toContain('<SFHomeTopicRow')
      expect(page).toContain('data-sf-region="topic-list"')
      expect(page).not.toContain('sforum-home__topic-table')
      expect(page).not.toContain('sf-public-page')
      expect(page).not.toContain('bg-[#E6F4F1]')
    }
  })

  test('commits filters through the URL with a debounced search draft', () => {
    const page = homepageIsland()

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
    const page = homepageIsland()
    const feedReset = page.slice(
      page.indexOf('watch(activePageFeedKey'),
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
    expect(page).toContain('parsePublicPage(route.query.page)')
    expect(page).toContain(':page-to="homePageTo"')
    expect(page).toContain('feedGeneration')
    expect(feedReset).toContain('feedGeneration += 1')
    expect(page).toContain('existingIds')
  })

  test('registers three-column homepage stylesheet tokens with right rail', () => {
    const config = hostConfig()
    const css = homepageCss()
    const theme = themeCss()
    const themePkgCss = source('../../../extensions/builtin/themes/sforum-default/assets/theme.css')
    const themeTokens = source('../../../extensions/builtin/themes/sforum-default/assets/tokens.css')
    const pkg = themePackage()
    const rightRail = source('../../../apps/web/app/components/SFHomeRightRail.vue')
    const page = homepageIsland()

    expect(config).toContain('sforum-home.css')
    expect(pkg).toContain('forum.home')
    expect(page).toContain('data-layout="fullwidth-3col"')
    expect(css).toContain('.sforum-home__layout--with-right')
    expect(css).toContain('var(--sf-public-right-rail-width)')
    expect(css).toContain('.sforum-home__right')
    expect(css).toContain('.sf-home-right-rail')
    // 列表行 BEM 已迁出：宿主 Tailwind + 主题 token，禁止双写
    expect(css).not.toContain('.sf-home-topic-row__chip')
    expect(css).not.toContain('.sforum-home__main-bar')
    expect(css).toContain('width: 100%')
    expect(css).not.toContain('sforum-home__hero')
    expect(theme).toContain('--sf-public-sidebar-width')
    expect(theme).toContain('--sf-public-right-rail-width')
    expect(theme).toContain('--sf-public-bg: #f5f6f8')
    expect(theme).toContain('--sf-public-shadow: none')
    expect(themePkgCss).toContain('--sf-public-bg: #f5f6f8')
    expect(themePkgCss).toContain('.sforum-home__layout--with-right')
    expect(themePkgCss).not.toContain('.sf-home-topic-row__chip--tone-0')
    expect(themeTokens).toContain('--sf-public-bg: #f5f6f8')
    expect(themeTokens).toContain('--sf-public-sidebar-width: 220px')
    // 主色归站点 appearance，默认主题 tokens 不得覆盖 --sf-accent*
    expect(themeTokens).not.toMatch(/--sf-accent\s*:/)
    expect(themeTokens).not.toContain('#3b6cf5')
    expect(rightRail).toContain('hotTopics')
    expect(rightRail).toContain('home.sidebar.hotThreads')
    expect(rightRail).toContain('home.sidebar.forumStats')
    expect(rightRail).toContain('useAuthSession')
    expect(rightRail).toContain('home.sidebar.welcomeTitle')
    expect(rightRail).toContain('home.sidebar.userCard')
    expect(rightRail).toContain('rightRailWelcome')
    expect(rightRail).toContain("emit('select-tag'")
    // 不伪造参与者堆/点赞等 demo 专属 UI
    expect(rightRail).not.toContain('participants')
    expect(page).not.toContain('participants')
  })

  test('keeps query-bearing public pages out of unsafe shared payload caches', () => {
    const middleware = homeQueryCacheMiddleware()
    const config = hostConfig()

    expect(middleware).toContain('no-store')
    for (const route of ['/c/**', '/en/c/**', '/tags/**', '/en/tags/**']) {
      expect(config).toContain(`'${route}': { cache: false }`)
    }
  })

  test('footer remains available via theme L1 and fail-closed host chrome', () => {
    expect(footer()).toContain('sf-footer')
    expect(defaultHomeTemplate()).toContain('<sf-footer>')
    expect(source('../app/components/SFHostPublicChrome.vue')).toContain('<SFFooter')
  })
})
