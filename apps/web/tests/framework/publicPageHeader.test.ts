import { describe, expect, test } from 'bun:test'
import { readFileSync } from 'node:fs'
import { fileURLToPath } from 'node:url'
import { compileVueSfc, mount } from '../helpers/vueSfc'

const componentPath = fileURLToPath(
  new URL('../../app/components/public/SFPublicPageHeader.vue', import.meta.url)
)
const PublicPageHeader = await compileVueSfc(componentPath, 'public-page-header')

const source = (path: string) => readFileSync(new URL(path, import.meta.url), 'utf8')

describe('shared public page header', () => {
  test('renders page and section heading semantics through one component', () => {
    const page = mount(PublicPageHeader, {
      props: {
        titleId: 'page-title',
        title: 'All categories',
        subtitle: 'Browse every category.'
      },
      slots: {
        eyebrow: '<span data-test="eyebrow">Forum</span>',
        meta: '<span data-test="meta">12 groups</span>',
        aside: '<button data-test="aside">Filter</button>'
      }
    })
    const section = mount(PublicPageHeader, {
      props: {
        titleId: 'section-title',
        title: 'Latest discussions',
        level: 2,
        variant: 'section'
      }
    })

    try {
      expect(page.classes()).toContain('sf-public-page-header--page')
      expect(page.get('h1').attributes('id')).toBe('page-title')
      expect(page.get('h1').text()).toBe('All categories')
      expect(page.get('.sf-public-page-header__subtitle').text()).toBe('Browse every category.')
      expect(page.get('[data-test="eyebrow"]').text()).toBe('Forum')
      expect(page.get('[data-test="meta"]').text()).toBe('12 groups')
      expect(page.get('[data-test="aside"]').text()).toBe('Filter')

      expect(section.classes()).toContain('sf-public-page-header--section')
      expect(section.find('h1').exists()).toBe(false)
      expect(section.get('h2').attributes('id')).toBe('section-title')
      expect(section.find('.sf-public-page-header__subtitle').exists()).toBe(false)
    } finally {
      page.unmount()
      section.unmount()
    }
  })

  test('centralizes typography tokens and removes page-owned title rules', () => {
    const theme = source('../../app/assets/css/sforum-theme.css')
    expect(theme).toContain('--sf-public-page-title-size: 1.375rem')
    expect(theme).toContain('--sf-public-page-title-mobile-size: 1.25rem')
    expect(theme).toContain('--sf-public-section-title-size: 20px')

    for (const component of [
      '../../app/components/forum/SFHomePage.vue',
      '../../app/components/forum/SFCategoryIndexPage.vue',
      '../../app/components/forum/SFCategoryShowPage.vue',
      '../../app/components/forum/SFTagIndexPage.vue',
      '../../app/components/forum/SFTagShowPage.vue',
      '../../app/components/notifications/SFNotificationsPage.vue',
      '../../app/components/notifications/detail/SFNotificationDetailPage.vue',
      '../../app/components/settings/SFSettingsShell.vue',
      '../../app/components/moderation/SFModerationReviewPage.vue'
    ]) {
      expect(source(component)).toContain('<SFPublicPageHeader')
    }

    const ownedCss = [
      source('../../app/assets/css/sforum-home.css'),
      source('../../app/assets/css/sforum-taxonomy.css'),
      source('../../app/assets/css/sforum-tags.css'),
      source('../../app/assets/css/sforum-settings.css'),
      source('../../app/assets/css/sforum-moderation.css'),
      source('../../app/components/notifications/SFNotificationsPage.css')
    ].join('\n')
    expect(ownedCss).not.toContain('sforum-home__feed-title')
    expect(ownedCss).not.toContain('sforum-home__page-header h1')
    expect(ownedCss).not.toContain('sforum-taxonomy__head h1')
    expect(ownedCss).not.toContain('sforum-category-directory__headline h1')
    expect(ownedCss).not.toContain('sforum-tags-page__head h1')
    expect(ownedCss).not.toContain('sforum-settings__head h1')
    expect(ownedCss).not.toContain('sforum-moderation__head h1')
    expect(ownedCss).not.toContain('sforum-notifications__head h1')
  })
})
