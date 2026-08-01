import { describe, expect, test } from 'bun:test'
import { readFileSync } from 'node:fs'

const source = (relative: string) => readFileSync(new URL(relative, import.meta.url), 'utf8')
const navbar = source('../../app/components/SFNavbar.vue')
const searchBar = source('../../app/components/navigation/SFPublicMobileSearchBar.vue')
const bottomNavigation = source('../../app/components/navigation/SFPublicMobileBottomNavigation.vue')

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
})
