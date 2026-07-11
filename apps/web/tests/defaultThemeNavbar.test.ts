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

  test('surfaces site tagline and hides register when registration is closed', () => {
    expect(source).toContain('siteTagline')
    expect(source).toContain('navbar__logo-tagline')
    expect(source).toContain("'/auth/registration-status'")
    expect(source).toContain('showRegisterLinks')
    expect(source).toContain('v-if="showRegisterLinks"')
  })

  test('uses the V32 sticky topbar shell on every public page', () => {
    expect(source).not.toContain('isWorkbenchHome')
    expect(source).not.toContain('navbar--workbench')
    expect(source).toContain('min-height: var(--sf-public-topbar-height, 52px)')
    expect(source).toContain('background: var(--sf-public-surface)')
    expect(source).not.toContain('scrollToHomeSection')
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
    expect(navMarkup).toContain("t('home.filter.latest')")
    expect(navMarkup).toContain("t('home.filter.categories')")
    expect(navMarkup).toContain("t('home.filter.tags')")
    expect(navMarkup).toContain('forumCategoriesIndexPath()')
    expect(navMarkup).toContain('forumTagsIndexPath()')
    expect(navMarkup).toContain('publicTagPagesEnabled')
    expect(navMarkup).not.toContain('/topics/new')
    expect(navMarkup).not.toContain('canCreateTopic')
    expect(navMarkup).not.toContain('热门')
    expect(navMarkup).not.toContain('排行')
    expect(navMarkup).not.toContain('scrollToHomeSection')
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
})
