import { describe, expect, test } from 'bun:test'
import { readFileSync } from 'node:fs'

const source = readFileSync(
  new URL('../../../extensions/builtin/themes/sforum-default/layer/app/components/SFNavbar.vue', import.meta.url),
  'utf8'
)

describe('default theme shared navbar contract', () => {
  test('keeps only real destinations and permission-aware header actions', () => {
    expect(source).toContain('<SFSearch')
    expect(source).toContain('@submit="submitSearch"')
    expect(source).toContain(':to="localePath(\'/\')"')
    expect(source).toContain('canCreateTopic')
    expect(source).not.toContain('disabled')
  })

  test('uses the shared modern card flow shell on every public page', () => {
    expect(source).not.toContain('isWorkbenchHome')
    expect(source).not.toContain('navbar--workbench')
    expect(source).toContain('max-width: var(--sf-public-container);')
    expect(source).toContain('min-height: 60px;')
    expect(source).toContain('background: var(--sf-public-glass);')
  })

  test('orders the desktop identity, home nav, search, compose action, and session controls', () => {
    const identityIndex = source.indexOf('class="navbar__logo"')
    const navIndex = source.indexOf('class="navbar__desktop-nav"')
    const searchIndex = source.indexOf('<SFSearch')
    const newTopicIndex = source.indexOf('class="navbar__new-topic"')
    const actionsIndex = source.indexOf('class="navbar__actions"')
    const navMarkup = source.slice(navIndex, source.indexOf('</nav>', navIndex))

    expect(identityIndex).toBeGreaterThan(-1)
    expect(navIndex).toBeGreaterThan(identityIndex)
    expect(searchIndex).toBeGreaterThan(navIndex)
    expect(newTopicIndex).toBeGreaterThan(searchIndex)
    expect(actionsIndex).toBeGreaterThan(newTopicIndex)
    expect(navMarkup).toContain(':to="localePath(\'/\')"')
    expect(navMarkup).not.toContain('/topics/new')
    expect(navMarkup).not.toContain('canCreateTopic')
  })

  test('submits compact search to the locale-aware homepage query', () => {
    expect(source).toMatch(
      /function submitSearch\(query: string\)[\s\S]*navigateTo\(\{[\s\S]*path: localePath\('\/'\),[\s\S]*query: buildForumHomeQuery\(\{[\s\S]*query,[\s\S]*categorySlug: '',[\s\S]*tagSlug: ''/
    )
  })

  test('uses accessible Nuxt UI menus without hand-written dropdown state', () => {
    expect(source).toContain('<UDropdownMenu')
    expect(source).toContain('i-lucide-menu')
    expect(source).toContain(':aria-label="t(\'nav.openMenu\')"')
    expect(source).toContain(':aria-label="t(\'nav.search\')"')
    expect(source).toContain(':avatar="user.avatar"')
    expect(source).not.toContain('menuOpen')
    expect(source).not.toContain('langMenuOpen')
    expect(source).not.toContain('onClickOutside')
    expect(source).not.toContain("document.addEventListener('click'")
  })

  test('keeps mobile compose visible and out of the mobile dropdown', () => {
    const linkIndex = source.indexOf('class="navbar__mobile-new-topic"')
    const linkMarkup = source.slice(
      source.lastIndexOf('<NuxtLink', linkIndex),
      source.indexOf('</NuxtLink>', linkIndex)
    )
    const menuItemsStart = source.indexOf('const mobileMenuItems')
    const menuItemsEnd = source.indexOf('\n\nwatch(', menuItemsStart)
    const menuItems = source.slice(menuItemsStart, menuItemsEnd)

    expect(linkIndex).toBeGreaterThan(-1)
    expect(linkMarkup).toContain('v-if="canCreateTopic"')
    expect(linkMarkup).toContain(':to="localePath(\'/topics/new\')"')
    expect(linkMarkup).toContain('i-lucide-square-pen')
    expect(linkMarkup).toContain(':aria-label="t(\'nav.newTopic\')"')
    expect(menuItems).not.toContain('/topics/new')
    expect(menuItems).not.toContain("t('nav.newTopic')")
  })

  test('opens a mobile search panel and closes it after locale-aware submission', () => {
    expect(source).toContain('const mobileSearchOpen = ref(false)')
    expect(source).toMatch(
      /label: t\('nav.search'\)[\s\S]*onSelect: \(\) => \{[\s\S]*mobileSearchOpen.value = true/
    )
    expect(source).toContain('v-if="mobileSearchOpen"')
    expect(source).toContain('class="navbar__mobile-search-panel"')
    expect(source).toContain('@submit="submitMobileSearch"')
    expect(source).toContain(':aria-label="t(\'nav.closeSearch\')"')
    expect(source).toMatch(
      /function submitMobileSearch\(query: string\)[\s\S]*mobileSearchOpen.value = false[\s\S]*return submitSearch\(query\)/
    )
  })

  test('provides 40px mobile touch targets for direct header actions and search', () => {
    const mobileStyles = source.slice(source.indexOf('@media (max-width: 980px)'))

    for (const selector of [
      '.navbar__logo',
      '.navbar__mobile-new-topic',
      '.navbar__user-trigger',
      '.navbar__mobile-trigger',
      '.navbar__mobile-search-close',
      '.navbar__mobile-search'
    ]) {
      expect(mobileStyles).toContain(selector)
    }

    expect(mobileStyles).toContain('min-height: 40px')
    expect(mobileStyles).toContain('min-width: 40px')
  })

  test('provides the shared header labels in both locales', () => {
    const localeFiles = ['zh-CN.json', 'en-US.json']

    for (const file of localeFiles) {
      const messages = JSON.parse(readFileSync(new URL(`../i18n/locales/${file}`, import.meta.url), 'utf8'))

      for (const key of ['search', 'closeSearch', 'openMenu', 'newTopic', 'appearance', 'language']) {
        expect(messages.nav[key]).toBeString()
        expect(messages.nav[key].length).toBeGreaterThan(0)
      }
    }
  })
})
