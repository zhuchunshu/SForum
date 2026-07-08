import { describe, expect, test } from 'bun:test'
import { readFileSync } from 'node:fs'

const homepage = () => readFileSync(
  new URL('../../../extensions/builtin/themes/sforum-default/layer/app/pages/index.vue', import.meta.url),
  'utf8'
)
const feedRow = () => readFileSync(new URL('../app/components/SFFeedRow.vue', import.meta.url), 'utf8')
const componentCss = () => readFileSync(new URL('../app/assets/css/sforum-components.css', import.meta.url), 'utf8')
const themeCss = () => readFileSync(
  new URL('../../../extensions/builtin/themes/sforum-default/layer/app/assets/css/sforum-theme.css', import.meta.url),
  'utf8'
)

describe('default theme homepage layout contract', () => {
  test('uses the high-density three-column homepage shell', () => {
    const source = homepage()

    expect(source).toContain('sforum-home')
    expect(source).toContain('sforum-home__layout')
    expect(source).toContain('lg:grid-cols-[240px_minmax(0,1fr)_262px]')
    expect(source).toContain('sforum-home__left')
    expect(source).toContain('sforum-home__main')
    expect(source).toContain('sforum-home__right')
    expect(source).toContain('lg:sticky lg:top-6')
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
    expect(source).toContain('layout="table"')
    expect(source).toContain(':last-activity-label="topicMeta(topic)"')
    expect(source).toContain(':last-actor="topicAuthor(topic)"')
    expect(source).not.toContain(':excerpt="topic.excerpt"')
  })

  test('feed row exposes table layout props without removing compact defaults', () => {
    const source = feedRow()

    expect(source).toContain("layout?: 'compact' | 'table'")
    expect(source).toContain("layout: 'compact'")
    expect(source).toContain('lastActivityLabel?: string')
    expect(source).toContain('lastActor?: string')
    expect(source).toContain('showAvatar?: boolean')
    expect(source).toContain('sf-feed-row--table')
    expect(source).toContain('sf-feed-row__stat')
    expect(source).toContain('sf-feed-row__stat--views')
    expect(source).toContain('sf-feed-row__last-activity')
  })

  test('css defines dense row, homepage shell, dark mode, and mobile collapse styles', () => {
    const componentSource = componentCss()
    const themeSource = themeCss()

    expect(componentSource).toContain('.sf-feed-row--table')
    expect(componentSource).toContain('grid-template-columns: minmax(0, 1fr) minmax(3.5rem, 4.5rem) minmax(3.5rem, 4.5rem) minmax(6rem, 7.5rem);')
    expect(componentSource).toContain('.sf-feed-row__last-activity')
    expect(componentSource).toContain('@media (max-width: 700px)')
    expect(componentSource).toContain('.dark .sf-feed-row--table:hover')

    expect(themeSource).toContain('.sforum-home')
    expect(themeSource).toContain('.sforum-home__rail')
    expect(themeSource).toContain('.sforum-topic-table')
    expect(themeSource).toContain('.sforum-home__mobile-filters')
  })
})
