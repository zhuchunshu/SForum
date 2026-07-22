import { describe, expect, test } from 'bun:test'
import { readFileSync } from 'node:fs'

const source = readFileSync(
  new URL('../../../apps/web/app/components/SFNavbar.vue', import.meta.url),
  'utf8'
)
const themeTemplateSource = readFileSync(
  new URL('../../../apps/web/app/components/SFThemeTemplate.vue', import.meta.url),
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

  test('marks desktop nav active with exact home match and underline class', () => {
    expect(source).toContain('function isDesktopNavActive')
    expect(source).toContain("class=\"navbar__nav-link\"")
    expect(source).toContain(":class=\"{ 'is-active': isDesktopNavActive(item.href) }\"")
    expect(source).toContain('active-class=""')
    expect(source).toContain('exact-active-class=""')
    expect(source).toContain('.navbar__nav-link.is-active::after')
  })

  test('orders the desktop identity, home nav, search, compose, utility, and session columns', () => {
    const identityIndex = source.indexOf('class="navbar__logo"')
    const navIndex = source.indexOf('class="navbar__desktop-nav"')
    const searchIndex = source.indexOf('<SFSearch')
    const newTopicIndex = source.indexOf('class="navbar__new-topic"')
    const utilityIndex = source.indexOf('class="navbar__utility"')
    const sessionIndex = source.indexOf('class="navbar__session"')
    const navMarkup = source.slice(navIndex, source.indexOf('</nav>', navIndex))

    expect(identityIndex).toBeGreaterThan(-1)
    expect(navIndex).toBeGreaterThan(identityIndex)
    expect(searchIndex).toBeGreaterThan(navIndex)
    expect(newTopicIndex).toBeGreaterThan(searchIndex)
    expect(utilityIndex).toBeGreaterThan(newTopicIndex)
    expect(sessionIndex).toBeGreaterThan(utilityIndex)
    expect(navMarkup).toContain('desktopNavItems')
    expect(source).toContain("t('home.filter.latest')")
    expect(source).toContain("t('home.filter.categories')")
    expect(source).toContain("t('home.filter.tags')")
    expect(source).toContain('forumCategoriesIndexPath()')
    expect(source).toContain('forumTagsIndexPath()')
    expect(source).toContain('publicTagPagesEnabled')
    // E2.3：运营 items 在前，扩展 extensionItems 次之
    expect(source).toContain('listPublicNav')
    expect(source).toContain('extensionItems')
    expect(source).toContain('isSafePublicNavHref')
    expect(navMarkup).not.toContain('/topics/new')
    expect(navMarkup).not.toContain('canCreateTopic')
    expect(navMarkup).not.toContain('热门')
    expect(navMarkup).not.toContain('排行')
    expect(navMarkup).not.toContain('scrollToHomeSection')
  })

  test('exposes language switch and color-mode toggle in the topbar utility cluster', () => {
    expect(source).toContain('languageMenuItems')
    expect(source).toContain("t('nav.language')")
    expect(source).toContain('toggleColorMode')
    expect(source).toContain('themeToggleLabel')
    expect(source).toContain('i-lucide-globe')
    expect(source).toContain('themeToggleIcon')
    // 会话区独立，便于默认主题与右栏同宽对齐
    expect(source).toContain('class="navbar__session"')
    expect(source).toContain('class="navbar__user-trigger"')
  })

  test('switches locale via setLocale without locale-prefixed routes', () => {
    // no_prefix：setLocale 只换文案 + cookie，不生成 /en 路径
    expect(source).toContain('setLocale')
    expect(source).toMatch(/void setLocale\(entry\.code\)/)
    expect(source).toContain('active: isCurrent')
    expect(source).toContain("const supportedLocaleCodes = ['zh-CN', 'en'] as const")
    expect(source).toContain('function isLocaleCode(value: string): value is LocaleCode')
    expect(source).toContain('if (typeof code !== \'string\' || !isLocaleCode(code))')
    expect(source).not.toContain('type LocaleCode = string')
    expect(source).not.toContain('to: switchLocalePath')
    expect(source).not.toContain('useSwitchLocalePath')
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
    expect(source).toContain('i-tabler-message-circle-filled')
    expect(source).toContain(':avatar="user.avatar"')
    expect(source).not.toContain('menuOpen')
    expect(source).not.toContain('langMenuOpen')
    expect(source).not.toContain('onClickOutside')
    expect(source).not.toContain("document.addEventListener('click'")
    expect(source.match(/<ClientOnly>/g)?.length).toBe(1)
    expect(source).toContain('navbar__control-placeholder')
    expect(source).toContain('navbar__session-placeholder')
  })

  test('keeps navbar and footer statically reachable from runtime theme templates', () => {
    expect(themeTemplateSource).toContain("import SFNavbar from './SFNavbar.vue'")
    expect(themeTemplateSource).toContain("import SFFooter from './SFFooter.vue'")
    expect(themeTemplateSource).toContain("'navigation.component.navbar': SFNavbar")
    expect(themeTemplateSource).toContain("'navigation.component.footer': SFFooter")
    expect(themeTemplateSource).not.toContain("'navigation.component.navbar': resolveComponent('SFNavbar')")
    expect(themeTemplateSource).not.toContain("'navigation.component.footer': resolveComponent('SFFooter')")
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
