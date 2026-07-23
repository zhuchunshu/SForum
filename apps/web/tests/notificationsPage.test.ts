import { describe, expect, test } from 'bun:test'
import { readFileSync } from 'node:fs'
import { ref } from 'vue'

import { useApiClient } from '../app/composables/useApiClient'
import { useNotifications, type NotificationItem } from '../app/composables/useNotifications'
import {
  filterNotifications,
  groupNotificationsByDate,
  notificationPresentation,
  notificationTarget,
  unreadLoadedCount
} from '../app/utils/notificationsPresentation'

;(globalThis as any).useApiClient = useApiClient

const source = (path: string) => readFileSync(new URL(path, import.meta.url), 'utf8')
const pageSource = () => source('../app/components/SFNotificationsPage.vue')
const pageStyles = () => source('../app/components/SFNotificationsPage.css')
const notificationComposableSource = () => source('../app/composables/useNotifications.ts')
const zh = () => JSON.parse(source('../i18n/locales/zh-CN.json'))
const en = () => JSON.parse(source('../i18n/locales/en-US.json'))

function item(input: Partial<NotificationItem>): NotificationItem {
  return {
    id: input.id || 1,
    type: input.type || 'reply',
    targetType: input.targetType || 'comment',
    targetId: input.targetId || 11,
    payload: input.payload || {},
    createdAt: input.createdAt || '2026-07-23T08:00:00.000Z',
    readAt: input.readAt,
    actorUserId: input.actorUserId
  }
}

function createStateStore() {
  const state = new Map<string, ReturnType<typeof ref>>()
  return {
    useState<T>(key: string, init: () => T) {
      if (!state.has(key)) state.set(key, ref(init()))
      return state.get(key) as ReturnType<typeof ref<T>>
    }
  }
}

async function withApiGlobals(run: () => Promise<void>) {
  const originalFetch = globalThis.$fetch
  const originalUseRuntimeConfig = globalThis.useRuntimeConfig
  const originalUseNuxtApp = globalThis.useNuxtApp
  const originalUseCookie = globalThis.useCookie
  const originalUseState = globalThis.useState
  const csrfCookie = ref('csrf-token')
  const stateStore = createStateStore()

  globalThis.useRuntimeConfig = () => ({
    public: { apiBaseUrl: '/api/v1', appLocale: 'zh-CN' }
  })
  globalThis.useNuxtApp = () => ({ $i18n: { locale: ref('zh-CN') } })
  globalThis.useCookie = () => csrfCookie
  globalThis.useState = stateStore.useState

  try {
    await run()
  } finally {
    globalThis.$fetch = originalFetch
    globalThis.useRuntimeConfig = originalUseRuntimeConfig
    globalThis.useNuxtApp = originalUseNuxtApp
    globalThis.useCookie = originalUseCookie
    globalThis.useState = originalUseState
  }
}

describe('notification presentation helpers', () => {
  test('maps only API-backed notification fields into visible presentation', () => {
    const view = notificationPresentation(item({
      id: 42,
      type: 'mention',
      payload: { topicId: 9, commentId: 15, title: 'A real topic title' },
      readAt: undefined
    }))

    expect(view).toMatchObject({
      id: 42,
      type: 'mention',
      read: false,
      titleKey: 'notifications.types.mention',
      targetLabel: 'A real topic title',
      typeLabelKey: 'notifications.filter.mention',
      target: { path: '/t/9#comment-15', unavailable: false }
    })
    expect(view.bodyKey).toBe('notifications.body.mention')
  })

  test('groups by today and earlier using the current locale timezone boundary', () => {
    const rows = [
      item({ id: 3, createdAt: '2026-07-22T16:10:00.000Z' }),
      item({ id: 2, createdAt: '2026-07-22T15:50:00.000Z' }),
      item({ id: 1, createdAt: '2026-07-20T05:00:00.000Z' })
    ]

    const groups = groupNotificationsByDate(rows, new Date('2026-07-22T16:30:00.000Z'), 'Asia/Shanghai', 'zh-CN')

    expect(groups.map(group => group.key)).toEqual(['today', 'earlier'])
    expect(groups[0].items.map(row => row.id)).toEqual([3])
    expect(groups[1].items.map(row => row.id)).toEqual([2, 1])
  })

  test('preserves API order inside date groups and filters only loaded notifications', () => {
    const views = [
      notificationPresentation(item({ id: 7, type: 'reply', readAt: undefined })),
      notificationPresentation(item({ id: 6, type: 'mention', readAt: '2026-07-23T09:00:00.000Z' })),
      notificationPresentation(item({ id: 5, type: 'reply', readAt: '2026-07-23T10:00:00.000Z' }))
    ]

    expect(filterNotifications(views, 'reply').map(view => view.id)).toEqual([7, 5])
    expect(filterNotifications(views, 'unread').map(view => view.id)).toEqual([7])
    expect(unreadLoadedCount(views)).toBe(1)
  })

  test('does not fabricate a target link when the API lacks a reliable topic target', () => {
    const target = notificationTarget(item({ type: 'admin_test', targetType: 'system', targetId: 0, payload: {} }))

    expect(target).toEqual({ path: '/notifications', unavailable: true })
  })
})

describe('useNotifications', () => {
  test('uses the current notification API pagination contract', async () => {
    const calls: string[] = []

    await withApiGlobals(async () => {
      globalThis.$fetch = async (url: string) => {
        calls.push(url)
        return { code: 200, message: 'ok', data: { items: [], hasMore: false } }
      }

      const api = useNotifications()
      await api.list()
      await api.list(88, 10)
    })

    expect(calls).toEqual([
      '/api/v1/notifications?limit=20',
      '/api/v1/notifications?beforeId=88&limit=10'
    ])
  })

  test('syncs unread count after single and all-read operations', async () => {
    const calls: Array<{ url: string, method?: string }> = []

    await withApiGlobals(async () => {
      globalThis.$fetch = async (url: string, options?: { method?: string }) => {
        calls.push({ url, method: options?.method })
        if (url.endsWith('/unread-count')) return { code: 200, message: 'ok', data: { count: 3 } }
        if (url.endsWith('/read-all')) return { code: 200, message: 'ok', data: { updated: 2 } }
        return { code: 200, message: 'ok', data: { read: true } }
      }

      const api = useNotifications()
      await api.refreshUnreadCount()
      expect(api.unreadCount.value).toBe(3)
      await api.markRead(12)
      expect(api.unreadCount.value).toBe(2)
      const updated = await api.markAllRead()
      expect(updated).toBe(2)
      expect(api.unreadCount.value).toBe(0)
    })

    expect(calls.map(call => [call.url, call.method || 'GET'])).toEqual([
      ['/api/v1/notifications/unread-count', 'GET'],
      ['/api/v1/notifications/12/read', 'PATCH'],
      ['/api/v1/notifications/read-all', 'POST']
    ])
  })
})

describe('SFNotificationsPage contract', () => {
  test('compiles as a focused Page Registry island and keeps the route shell thin', () => {
    const route = source('../app/pages/notifications.vue')
    const page = pageSource()
    expect(page).toContain('<script setup lang="ts">')
    expect(page).toContain('<template>')
    expect(page).toContain('<style scoped src="./SFNotificationsPage.css"></style>')
    expect(route).toContain('definePageMeta({ requiresAuth: true })')
    expect(route).toContain('<SFPageOutlet page="forum.notifications">')
    expect(route).toContain('<SFNotificationsPage />')
    expect(route).not.toContain('useNotifications()')
  })

  test('uses the default theme three-column shell, shared community nav, and mobile drawer state', () => {
    const page = pageSource()
    const styles = pageStyles()

    expect(page).toContain('data-layout="fullwidth-3col"')
    expect(page).toContain('sforum-notifications__layout')
    expect(page).toContain('sforum-notifications__sidebar sforum-home__sidebar')
    expect(page).toContain('<SFHomeNavigation')
    expect(page).toContain('usePermissions()')
    expect(page).toContain('can(FORUM_PERMISSIONS.topicCreate)')
    expect(page).not.toContain("can('forum.topic.create')")
    expect(page).toContain("navigation-mode=\"route\"")
    expect(page).toContain("useState<boolean>('forum-mobile-menu-open'")
    expect(page).toContain("useState<boolean>('forum-mobile-info-open'")
    expect(page).toContain('sforum-mobile-drawer sforum-mobile-drawer--left')
    expect(page).toContain('sforum-mobile-drawer sforum-mobile-drawer--right')
    expect(styles).toContain('.sforum-mobile-drawer .sforum-notifications__right--drawer')
    expect(styles).toContain('display: block')
    expect(page).not.toContain('ssr: false')
  })

  test('keeps type filtering honest and avoids server-filter claims', () => {
    const page = pageSource()
    const composable = notificationComposableSource()

    expect(page).toContain('filterNotifications(presentedItems.value, activeFilter.value)')
    expect(page).toContain("t('notifications.filter.loadedScope'")
    expect(page).toContain('notificationFilters')
    expect(composable).not.toContain('type=')
    expect(composable).toContain('beforeId')
    expect(composable).toContain('limit')
  })

  test('contains real read operations, duplicate guards, optimistic rollback, persistent failures, and themed success toast', () => {
    const page = pageSource()

    expect(page).toContain('markingIds.value.has(item.id)')
    expect(page).toContain('markingAll.value')
    expect(page).toContain('previousUnreadCount')
    expect(page).toContain('setNotificationRead(item.id, previousReadAt)')
    expect(page).toContain('notifications.markRead(item.id)')
    expect(page).toContain('notifications.markAllRead()')
    expect(page).toContain("duration: 0")
    expect(page).toContain("duration: 10000")
    expect(page).toContain("color: 'primary'")
    expect(page).toContain("color: 'error'")
  })

  test('ships bilingual copy for every notification type and state added by the page', () => {
    for (const locale of [zh(), en()]) {
      expect(locale.notifications.types.admin_test).toBeTruthy()
      expect(locale.notifications.filter.loadedScope).toBeTruthy()
      expect(locale.notifications.groups.today).toBeTruthy()
      expect(locale.notifications.groups.earlier).toBeTruthy()
      expect(locale.notifications.body.moderation_rejected_note).toBeTruthy()
      expect(locale.notifications.actions.viewReply).toBeTruthy()
      expect(locale.notifications.detail.empty).toBeTruthy()
      expect(locale.notifications.targetUnavailableHelp).toBeTruthy()
    }
  })
})
