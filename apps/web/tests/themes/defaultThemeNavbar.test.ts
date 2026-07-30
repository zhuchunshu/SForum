import { describe, expect, test } from 'bun:test'
import { readFileSync } from 'node:fs'

const source = readFileSync(
  new URL('../../app/components/SFNavbar.vue', import.meta.url),
  'utf8'
)
const navigationLinksSource = readFileSync(
  new URL('../../app/components/navigation/SFPublicNavigationLinks.vue', import.meta.url),
  'utf8'
)
const mobileNavigationSource = readFileSync(
  new URL('../../app/components/navigation/SFPublicMobileNavigation.vue', import.meta.url),
  'utf8'
)
const sidebarDrawerSource = readFileSync(
  new URL('../../app/composables/navigation/usePublicSidebarDrawer.ts', import.meta.url),
  'utf8'
)
const navigationComposableSource = readFileSync(
  new URL('../../app/composables/navigation/usePublicNavigation.ts', import.meta.url),
  'utf8'
)
const languageMenuComposableSource = readFileSync(
  new URL('../../app/composables/navigation/useNavbarLanguageMenu.ts', import.meta.url),
  'utf8'
)
const themeTemplateSource = readFileSync(
  new URL('../../app/components/SFThemeTemplate.vue', import.meta.url),
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
    expect(navigationLinksSource).toContain('function isActive')
    expect(navigationLinksSource).toContain("class=\"sf-public-navigation-links__link\"")
    expect(navigationLinksSource).toContain(":class=\"{ 'is-active': isActive(item) }\"")
    expect(navigationLinksSource).toContain('active-class=""')
    expect(navigationLinksSource).toContain('exact-active-class=""')
    expect(navigationLinksSource).toContain('.sf-public-navigation-links--topbar .sf-public-navigation-links__link.is-active::after')
  })

  test('orders the desktop identity, home nav, search, compose, utility, and session columns', () => {
    const identityIndex = source.indexOf('class="navbar__logo"')
    const navIndex = source.indexOf('class="navbar__desktop-nav"')
    const searchIndex = source.indexOf('<SFSearch')
    const newTopicIndex = source.indexOf('class="navbar__new-topic"')
    const utilityIndex = source.indexOf('class="navbar__utility"')
    const sessionIndex = source.indexOf('class="navbar__session"')
    const navMarkup = source.slice(navIndex, source.indexOf('/>', navIndex))

    expect(identityIndex).toBeGreaterThan(-1)
    expect(navIndex).toBeGreaterThan(identityIndex)
    expect(searchIndex).toBeGreaterThan(navIndex)
    expect(newTopicIndex).toBeGreaterThan(searchIndex)
    expect(utilityIndex).toBeGreaterThan(newTopicIndex)
    expect(sessionIndex).toBeGreaterThan(utilityIndex)
    expect(navMarkup).toContain('visibleTopbarItems')
    expect(source).toContain('publicTagPagesEnabled')
    expect(source).toContain('usePublicNavigation(props.fetchRemoteChrome)')
    expect(source).not.toContain('/site/nav-items')
    expect(source).not.toContain('fallback-home')
    expect(source).not.toContain('listPublicNav')
    expect(source).not.toContain('extensionItems')
    expect(source).not.toContain('desktopNavItems')
    expect(source).not.toContain("t('home.filter.latest')")
    expect(source).not.toContain("t('home.filter.categories')")
    expect(source).not.toContain("t('home.filter.tags')")
    expect(source).not.toContain('forumCategoriesIndexPath()')
    expect(source).not.toContain('forumTagsIndexPath()')
    expect(source).not.toContain('isSafePublicNavHref')
    expect(navigationComposableSource).toContain('/site/navigation?locations=')
    expect(navigationComposableSource).toContain('PUBLIC_NAVIGATION_LOCATIONS.topbar')
    expect(navigationComposableSource).toContain('PUBLIC_NAVIGATION_LOCATIONS.sidebar')
    expect(navigationComposableSource).not.toContain('PUBLIC_NAVIGATION_LOCATIONS.mobile')
    expect(navMarkup).not.toContain('/topics/new')
    expect(navMarkup).not.toContain('canCreateTopic')
    expect(navMarkup).not.toContain('热门')
    expect(navMarkup).not.toContain('排行')
    expect(navMarkup).not.toContain('scrollToHomeSection')
  })

  test('exposes language switch and cyclic appearance control in the topbar utility cluster', () => {
    expect(source).toContain('languageMenuItems')
    expect(source).toContain("t('nav.language')")
    expect(source).toContain('i-tabler-language')
    expect(source).toContain('cycleColorModePreference')
    expect(source).toContain('colorModeTriggerLabel')
    expect(source).toContain('const colorModeTriggerIcon = computed')
    expect(source).toContain('@click="cycleColorModePreference"')
    expect(source).toContain('i-tabler-language" class="size-5"')
    expect(source).toContain(':name="colorModeTriggerIcon" class="size-5"')
    expect(source).not.toContain('toggleColorMode')
    expect(source).not.toContain('MutationObserver')
    expect(source).not.toContain(':aria-pressed="isDarkMode"')
    // 会话区独立，便于默认主题与右栏同宽对齐
    expect(source).toContain('class="navbar__session"')
    expect(source).toContain('class="navbar__user-trigger"')
  })

  test('switches locale through the persisted user-language authority without locale-prefixed routes', () => {
    expect(source).toContain('useNavbarLanguageMenu')
    expect(languageMenuComposableSource).toContain('useUserLanguage')
    expect(languageMenuComposableSource).toContain('languageOptions')
    expect(languageMenuComposableSource).toMatch(/void updateLanguage\(entry\.value\)/)
    expect(languageMenuComposableSource).toContain('active: isCurrent')
    expect(languageMenuComposableSource).not.toContain('to: switchLocalePath')
    expect(languageMenuComposableSource).not.toContain('useSwitchLocalePath')
  })

  test('submits compact search to the locale-aware search page', () => {
    expect(source).toMatch(
      /function submitSearch\(query: string\)[\s\S]*path: localePath\(normalizedQuery \? '\/search' : '\/'\)[\s\S]*query: buildForumHomeQuery\(\{[\s\S]*query: normalizedQuery,[\s\S]*categorySlug: '',[\s\S]*tagSlug: ''/
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
    expect(source.match(/<ClientOnly(?:\s|>)/g)?.length).toBe(3)
    expect(source).toContain('data-ssr-fallback="navbar-language"')
    expect(source).toContain('data-ssr-fallback="navbar-appearance"')
    expect(source).toContain('data-ssr-fallback="navbar-user"')
    expect(source).toContain('i-tabler-brightness-filled')
    expect(source).toContain('class="navbar__session-loading"')
    expect(source).not.toContain('navbar__control-placeholder')
    expect(source).not.toContain('navbar__session-placeholder')
    expect(source.match(/aria-hidden="true"[\s\S]*?tabindex="-1"[\s\S]*?data-ssr-fallback=/g)?.length).toBe(3)
    expect(source).toContain('[data-ssr-fallback]')
    expect(source).toContain('pointer-events: none')
  })

  test('keeps navbar and footer statically reachable from runtime theme templates', () => {
    expect(themeTemplateSource).toContain("import SFNavbar from './SFNavbar.vue'")
    expect(themeTemplateSource).toContain("import SFFooter from './SFFooter.vue'")
    expect(themeTemplateSource).toContain("'navigation.component.navbar': SFNavbar")
    expect(themeTemplateSource).toContain("'navigation.component.footer': SFFooter")
    expect(themeTemplateSource).not.toContain("'navigation.component.navbar': resolveComponent('SFNavbar')")
    expect(themeTemplateSource).not.toContain("'navigation.component.footer': resolveComponent('SFFooter')")
  })

  test('keeps Host utilities available when canonical navigation is empty', () => {
    expect(source).toContain('<SFSearch')
    expect(source).toContain('class="navbar__utility"')
    expect(source).toContain('languageMenuItems')
    expect(source).toContain('cycleColorModePreference')
    expect(source).toContain('class="navbar__session"')
    expect(source).not.toContain('v-if="visibleTopbarItems.length"')
  })

  test('keeps mobile compose visible and renders the desktop sidebar source in the drawer', () => {
    const linkIndex = source.indexOf('class="navbar__mobile-new-topic"')
    const linkMarkup = source.slice(
      source.lastIndexOf('<NuxtLink', linkIndex),
      source.indexOf('</NuxtLink>', linkIndex)
    )
    expect(linkIndex).toBeGreaterThan(-1)
    expect(linkMarkup).toContain('v-if="canCreateTopic"')
    expect(linkMarkup).toContain(':to="localePath(\'/topics/new\')"')
    expect(linkMarkup).toContain('i-lucide-square-pen')
    expect(linkMarkup).toContain(':aria-label="t(\'nav.newTopic\')"')
    expect(source).toContain('<SFPublicMobileNavigation')
    expect(source).toContain(':items="visibleSidebarItems"')
    expect(source).toContain('const { topbarItems, sidebarItems } = usePublicNavigation')
    expect(source).not.toContain('mobileItems')
    expect(source).toContain('usePublicSidebarDrawer()')
    expect(source).toContain('mobileMenuOpen && !hasPageSidebarOwner')
    expect(sidebarDrawerSource).toContain("'public-sidebar-drawer-open'")
    expect(source).not.toContain("'public-mobile-navigation-open'")
    expect(source).not.toContain("'forum-mobile-menu-open'")
    expect(mobileNavigationSource).toContain('data-navigation-location="public.sidebar.primary"')
    expect(mobileNavigationSource).toContain('data-navigation-viewport="mobile"')
    expect(mobileNavigationSource).toContain('<SFMobileSidebarContent')
  })

  test('bounds topbar items and keeps external destinations safe', () => {
    expect(navigationLinksSource).toContain('visibleLimit: 4')
    expect(navigationLinksSource).toContain('<UDropdownMenu')
    expect(navigationLinksSource).toContain("t('nav.more')")
    expect(navigationLinksSource).toContain('target="_blank"')
    expect(navigationLinksSource).toContain('rel="noopener noreferrer"')
  })
})
