import { describe, expect, test } from 'bun:test'
import { readFileSync } from 'node:fs'

const source = (relative: string) => readFileSync(new URL(relative, import.meta.url), 'utf8')
const navbar = source('../../app/components/SFNavbar.vue')
const searchBar = source('../../app/components/navigation/SFPublicMobileSearchBar.vue')
const bottomNavigation = source('../../app/components/navigation/SFPublicMobileBottomNavigation.vue')
const mobileRightDrawerHeader = source('../../app/components/navigation/SFPublicMobileRightDrawerHeader.vue')
const mobileUserMenu = source('../../app/components/navigation/SFPublicMobileUserMenu.vue')
const userMenuComposable = source('../../app/composables/navigation/usePublicUserMenu.ts')

describe('public mobile navbar composition', () => {
  test('keeps the shared navbar focused on chrome orchestration', () => {
    expect(navbar).toContain("import SFPublicMobileSearchBar from '~/components/navigation/SFPublicMobileSearchBar.vue'")
    expect(navbar).toContain("import SFPublicMobileBottomNavigation from '~/components/navigation/SFPublicMobileBottomNavigation.vue'")
    expect(navbar).toContain('v-model="searchQuery"')
    expect(navbar).toContain(':can-create-topic="canCreateTopic"')
    expect(navbar).toContain(':authenticated="Boolean(user)"')
    expect(navbar).not.toContain('sf-public-mobile-bottom-nav__badge')
  })

  test('keeps mobile search and compose destinations permission-aware', () => {
    expect(searchBar).toContain("emit('update:modelValue', value)")
    expect(searchBar).toContain("@submit=\"emit('submit', $event)\"")
    expect(searchBar).toContain("canCreateTopic ? localePath('/topics/new') : localePath('/login')")
    expect(searchBar).toContain("canCreateTopic ? 'i-lucide-square-pen' : 'i-lucide-log-in'")
  })

  test('keeps the fixed bottom navigation order and unread badge contract', () => {
    const homeIndex = bottomNavigation.indexOf("<span>{{ t('nav.home') }}</span>")
    const composeIndex = bottomNavigation.indexOf("<span>{{ t('nav.newTopic') }}</span>")
    const notificationsIndex = bottomNavigation.indexOf("<span>{{ t('nav.notifications') }}</span>")

    expect(homeIndex).toBeGreaterThan(-1)
    expect(composeIndex).toBeGreaterThan(homeIndex)
    expect(notificationsIndex).toBeGreaterThan(composeIndex)
    expect(bottomNavigation).toContain("authenticated ? localePath('/notifications') : localePath('/login')")
    expect(bottomNavigation).toContain('authenticated && notifications.unreadCount.value')
    expect(bottomNavigation).toContain("notifications.unreadCount.value > 99 ? '99+' : notifications.unreadCount.value")
  })

  test('opens the mobile right drawer from the authenticated user avatar', () => {
    expect(navbar).toContain('class="navbar__mobile-info-button"')
    expect(navbar).toContain('v-if="user"')
    expect(navbar).toContain(':avatar="user.avatar"')
    expect(navbar).toContain("v-else name=\"i-lucide-panel-right\"")
    expect(navbar).toContain("'navbar__session--authenticated': Boolean(user)")
    expect(navbar).toContain('.navbar__session--authenticated')
    expect(navbar).toContain('display: none')
  })

  test('renders the desktop avatar dropdown actions inside the mobile right drawer', () => {
    expect(navbar).toContain('usePublicUserMenu()')
    expect(navbar).toContain(':items="userMenuItems"')
    expect(mobileUserMenu).toContain('usePublicUserMenu()')
    expect(mobileUserMenu).toContain('v-for="(group, groupIndex) in menuGroups"')
    expect(mobileUserMenu).toContain('class="sf-public-mobile-user-menu__identity"')
    expect(mobileUserMenu).toContain('@click="closeDrawer"')
    expect(userMenuComposable).toContain("key: 'profile'")
    expect(userMenuComposable).toContain("key: 'profile-settings'")
    expect(userMenuComposable).toContain("key: 'moderation'")
    expect(userMenuComposable).toContain("key: 'logout'")
    expect(userMenuComposable).toContain("key: 'resend-email-verification'")
  })

  test('keeps the mobile user actions collapsed by default and exposes an accessible toggle', () => {
    expect(mobileUserMenu).toContain('const menuOpen = ref(false)')
    expect(mobileUserMenu).toContain(':aria-expanded="menuOpen"')
    expect(mobileUserMenu).toContain('aria-controls="sf-public-mobile-user-menu-actions"')
    expect(mobileUserMenu).toContain('v-if="menuOpen" id="sf-public-mobile-user-menu-actions"')
    expect(mobileUserMenu).toContain("menuOpen ? 'i-lucide-chevron-up' : 'i-lucide-chevron-down'")
  })

  test('uses personal center as the authenticated drawer heading without losing the guest context', () => {
    expect(mobileRightDrawerHeader).toContain("useAuthSession()")
    expect(mobileRightDrawerHeader).toContain("user ? t('nav.personalCenter') : title")
    expect(mobileRightDrawerHeader).toContain(':aria-label="closeLabel"')
    expect(mobileRightDrawerHeader).toContain("@click=\"$emit('close')\"")
  })
})
