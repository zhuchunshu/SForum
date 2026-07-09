import { describe, expect, test } from 'bun:test'
import { readFileSync } from 'node:fs'

const homepage = () => readFileSync(
  new URL('../../../extensions/builtin/themes/sforum-default/layer/app/pages/index.vue', import.meta.url),
  'utf8'
)
const defaultLayout = () => readFileSync(
  new URL('../../../extensions/builtin/themes/sforum-default/layer/app/layouts/default.vue', import.meta.url),
  'utf8'
)
const componentCss = () => readFileSync(new URL('../app/assets/css/sforum-components.css', import.meta.url), 'utf8')
const themeCss = () => readFileSync(
  new URL('../../../extensions/builtin/themes/sforum-default/layer/app/assets/css/sforum-theme.css', import.meta.url),
  'utf8'
)

describe('default theme homepage layout contract', () => {
  test('renders the accepted A-direction compact forum shell as the homepage chrome', () => {
    const source = homepage()

    expect(source).toContain('sforum-home')
    expect(source).toContain('definePageMeta')
    expect(source).toContain('layout: false')
    expect(source).toContain('sforum-home__shell')
    expect(source).toContain('sforum-home__topbar')
    expect(source).toContain('sforum-home__brand')
    expect(source).toContain('sforum-home__top-links')
    expect(source).toContain('sforum-home__top-actions')
    expect(source).toContain('sforum-home__notice')
    expect(source).toContain('sforum-home__layout')
    expect(source).toContain('sforum-home__left')
    expect(source).toContain('sforum-home__main')
    expect(source).toContain('sforum-home__side-link')
    expect(source).toContain('sforum-home__feed-tabs')
    expect(source).not.toContain('sforum-home__content-toolbar')
    expect(source).not.toContain("<h1>{{ t('home.sidebar.navHome') }}</h1>")
    expect(source).not.toContain('sforum-home__dock')
  })

  test('keeps the reusable default layout available for non-home public pages', () => {
    const layoutSource = defaultLayout()

    expect(layoutSource).toContain('<SFNavbar />')
    expect(layoutSource).toContain('<SFFooter />')
  })

  test('keeps taxonomy controls available outside the desktop rail', () => {
    const source = homepage()

    expect(source).toContain('sforum-home__mobile-filters')
    expect(source).toContain('selectedCategorySlug')
    expect(source).toContain('selectedTagSlug')
    expect(source).toContain('selectCategory')
    expect(source).toContain('selectTag')
  })

  test('renders topics through the table-oriented feed row variant', () => {
    const source = homepage()

    expect(source).toContain('sforum-topic-table')
    expect(source).toContain('sforum-topic-table__head')
    expect(source).toContain('sforum-topic-row')
    expect(source).toContain('sforum-topic-row__participants')
    expect(source).toContain('topicActivity(topic)')
    expect(source).toContain('topicReplyStackLabel(topic)')
    expect(source).not.toContain(':excerpt="topic.excerpt"')
    expect(source).not.toContain('<SFFeedRow')
  })

  test('uses automatic feed loading instead of visible homepage pagination', () => {
    const source = homepage()

    expect(source).toContain('loadMoreTrigger')
    expect(source).toContain('IntersectionObserver')
    expect(source).toContain('loadMoreTopics')
    expect(source).toContain("useState<ForumTopicSummary[]>('forum-home-loaded-topics'")
    expect(source).toContain('() => topicList.value.items')
    expect(source).toContain('() => topicList.value.total')
    expect(source).toContain("'forum-home-loaded-feed-key'")
    expect(source).toContain('shouldIgnoreClientEmptyHydration')
    expect(source).toContain('activeFeedKey.value === loadedFeedKey.value')
    expect(source).toContain('sforum-topic-table__infinite-state')
    expect(source).not.toContain('<SFPagination')
  })

  test('css defines dense row, homepage shell, dark mode, and mobile collapse styles', () => {
    const componentSource = componentCss()
    const themeSource = themeCss()

    expect(componentSource).toContain('.sf-feed-row--table')
    expect(componentSource).toContain('@media (max-width: 700px)')
    expect(componentSource).toContain('.dark .sf-feed-row--table:hover')

    expect(themeSource).toContain('.sforum-home')
    expect(themeSource).toContain('--sforum-home-accent: var(--sf-accent);')
    expect(themeSource).toContain('--sforum-home-scroll-thumb: rgb(var(--sf-accent-rgb) / 0.28);')
    expect(themeSource).toContain('background: #e9eef4;')
    expect(themeSource).toContain('.sforum-home__shell')
    expect(themeSource).toContain('width: min(100%, 1520px);')
    expect(themeSource).toContain('.sforum-home__topbar')
    expect(themeSource).toContain('.sforum-home__brand')
    expect(themeSource).toContain('.sforum-home__top-actions .sf-search')
    expect(themeSource).toContain('height: 2.125rem;')
    expect(themeSource).toContain('color: var(--sf-fg);')
    expect(themeSource).toContain('.sforum-home__layout')
    expect(themeSource).toContain('grid-template-columns: 238px minmax(0, 1fr);')
    expect(themeSource).toContain('min-height: 740px;')
    expect(themeSource).toContain('.sforum-home__side-link')
    expect(themeSource).toContain('.sforum-home__notice')
    expect(themeSource).toContain('.sforum-home__feed-tabs')
    expect(themeSource).toContain('.sforum-topic-table')
    expect(themeSource).toContain('.sforum-topic-table__head')
    expect(themeSource).toContain('.sforum-topic-row')
    expect(themeSource).toContain('grid-template-columns: minmax(0, 1fr) 120px 86px;')
    expect(themeSource).toContain('.sforum-home__mobile-filters')
    expect(themeSource).toContain('max-height: calc(100vh - 3rem);')
    expect(themeSource).toContain('overflow-y: auto;')
    expect(themeSource).toContain('overscroll-behavior: contain;')
    expect(themeSource).not.toContain('.sforum-home__content-toolbar')
    expect(themeSource).not.toContain('rgba(15, 118, 110, 0.28)')
  })
})
