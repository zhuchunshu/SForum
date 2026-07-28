import { describe, expect, test } from 'bun:test'
import { readFileSync } from 'node:fs'
import { ref } from 'vue'

import { useApiClient } from '../../app/composables/useApiClient'
import { useNotifications, type NotificationItem } from '../../app/composables/notifications/useNotifications'
import {
  filterNotifications,
  groupNotificationsByDate,
  notificationFilterCounts,
  notificationPresentation,
  notificationTarget,
  unreadLoadedCount
} from '../../app/utils/notifications/notificationsPresentation'

;(globalThis as any).useApiClient = useApiClient

const source = (path: string) => readFileSync(new URL(path, import.meta.url), 'utf8')
const pageSource = () => source('../../app/components/notifications/SFNotificationsPage.vue')
const pageStyles = () => source('../../app/components/notifications/SFNotificationsPage.css')
const detailPageSource = () => source('../../app/components/notifications/detail/SFNotificationDetailPage.vue')
const typeNavSource = () => source('../../app/components/notifications/SFNotificationTypeNav.vue')
const detailRouteSource = () => source('../../app/pages/notifications/[notificationId].vue')
const notificationComposableSource = () => source('../../app/composables/notifications/useNotifications.ts')
const navbarSource = () => source('../../app/components/SFNavbar.vue')
const zh = () => JSON.parse(source('../../i18n/locales/zh-CN.json'))
const en = () => JSON.parse(source('../../i18n/locales/en-US.json'))

function item(input: Partial<NotificationItem>): NotificationItem {
  return {
    id: input.id || 1,
    type: input.type || 'reply',
    targetType: input.targetType || 'comment',
    targetId: input.targetId || 11,
    payload: input.payload || {},
    createdAt: input.createdAt || '2026-07-23T08:00:00.000Z',
    readAt: input.readAt,
    actorUserId: input.actorUserId,
    actor: input.actor
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

  test('uses actor avatars only for user-authored notification types', () => {
    const actor = {
      id: 8,
      username: 'alice',
      displayName: 'Alice',
      avatar: { kind: 'uploaded' as const, url: '/api/v1/attachments/avatar/content', alt: 'Alice' }
    }

    expect(notificationPresentation(item({ type: 'reply', actor })).actor).toEqual(actor)
    expect(notificationPresentation(item({ type: 'mention', actor })).actor).toEqual(actor)

    const system = notificationPresentation(item({ type: 'moderation_approved', actor }))
    expect(system.actor).toBeUndefined()
    expect(system.icon).toBe('i-tabler-shield-check')
  })

  test('counts the loaded notification scope for the shared type menu', () => {
    const counts = notificationFilterCounts([
      { type: 'reply', read: false },
      { type: 'reply', read: true },
      { type: 'mention', read: true }
    ])

    expect(counts.get('all')).toBe(3)
    expect(counts.get('unread')).toBe(1)
    expect(counts.get('reply')).toBe(2)
    expect(counts.get('mention')).toBe(1)
    expect(counts.get('moderation_approved')).toBeUndefined()
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
      await api.list(88, 10, { category: 'moderation', type: 'moderation_approved', unread: true })
      await api.get(42)
    })

    expect(calls).toEqual([
      '/api/v1/notifications?limit=20',
      '/api/v1/notifications?beforeId=88&limit=10&category=moderation&type=moderation_approved&unread=true',
      '/api/v1/notifications/42'
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

  test('shares one EventSource and coalesces revision refreshes', async () => {
    const originalEventSource = globalThis.EventSource
    const sources: FakeEventSource[] = []
    class FakeEventSource {
      static readonly CONNECTING = 0
      static readonly OPEN = 1
      static readonly CLOSED = 2
      readonly url: string
      readyState = FakeEventSource.OPEN
      closed = false
      listeners = new Map<string, Array<(event: MessageEvent<string>) => void>>()
      constructor(url: string) {
        this.url = url
        sources.push(this)
      }
      addEventListener(type: string, listener: EventListenerOrEventListenerObject) {
        const callback = listener as (event: MessageEvent<string>) => void
        this.listeners.set(type, [...(this.listeners.get(type) || []), callback])
      }
      emit(type: string, data: string) {
        for (const listener of this.listeners.get(type) || []) listener({ data } as MessageEvent<string>)
      }
      close() { this.closed = true; this.readyState = FakeEventSource.CLOSED }
    }

    await withApiGlobals(async () => {
      globalThis.EventSource = FakeEventSource as unknown as typeof EventSource
      const first = useNotifications()
      const second = useNotifications()
      let firstRefreshes = 0
      let secondRefreshes = 0
      const stopFirst = first.startRealtime(() => { firstRefreshes++ })
      const stopSecond = second.startRealtime(() => { secondRefreshes++ })
      expect(sources).toHaveLength(1)
      expect(sources[0]?.url).toBe('/api/v1/notifications/stream?revision=0')

      sources[0]?.emit('revision', '{"revision":1}')
      sources[0]?.emit('revision', '{"revision":2}')
      await new Promise(resolve => setTimeout(resolve, 130))
      expect(firstRefreshes).toBe(1)
      expect(secondRefreshes).toBe(1)
      expect(first.revision.value).toBe(2)

      stopFirst()
      expect(sources[0]?.closed).toBe(false)
      sources[0]?.emit('revision', '{"revision":3}')
      second.stopRealtime()
      expect(sources[0]?.closed).toBe(true)
      await new Promise(resolve => setTimeout(resolve, 130))
      expect(firstRefreshes).toBe(1)
      expect(secondRefreshes).toBe(1)
      stopSecond()
    })
    globalThis.EventSource = originalEventSource
  })

  test('closes a failed EventSource and reconnects it through the controlled timer', async () => {
    const originalEventSource = globalThis.EventSource
    const sources: RecoveringFakeEventSource[] = []
    class RecoveringFakeEventSource {
      static readonly CONNECTING = 0
      static readonly OPEN = 1
      static readonly CLOSED = 2
      readonly url: string
      readyState = RecoveringFakeEventSource.OPEN
      listeners = new Map<string, Array<(event: Event) => void>>()
      constructor(url: string) {
        this.url = url
        sources.push(this)
      }
      addEventListener(type: string, listener: EventListenerOrEventListenerObject) {
        const callback = listener as (event: Event) => void
        this.listeners.set(type, [...(this.listeners.get(type) || []), callback])
      }
      failWhileConnecting() {
        this.readyState = RecoveringFakeEventSource.CONNECTING
        for (const listener of this.listeners.get('error') || []) listener(new Event('error'))
      }
      close() { this.readyState = RecoveringFakeEventSource.CLOSED }
    }

    await withApiGlobals(async () => {
      globalThis.EventSource = RecoveringFakeEventSource as unknown as typeof EventSource
      const api = useNotifications()
      let refreshes = 0
      const stop = api.startRealtime(() => { refreshes++ })

      expect(sources).toHaveLength(1)
      sources[0]?.failWhileConnecting()
      expect(sources[0]?.readyState).toBe(RecoveringFakeEventSource.CLOSED)
      await new Promise(resolve => setTimeout(resolve, 130))
      expect(refreshes).toBe(1)

      await new Promise(resolve => setTimeout(resolve, 950))
      expect(sources).toHaveLength(2)
      stop()
    })
    globalThis.EventSource = originalEventSource
  })

  test('keeps manual REST available when EventSource construction fails', async () => {
    const originalEventSource = globalThis.EventSource
    await withApiGlobals(async () => {
      globalThis.EventSource = class {
        constructor() { throw new Error('SSE unavailable') }
      } as unknown as typeof EventSource
      globalThis.$fetch = async () => ({ code: 200, message: 'ok', data: { count: 4 } })
      const api = useNotifications()
      const stop = api.startRealtime(() => {})
      expect(await api.refreshUnreadCount()).toBe(4)
      stop()
    })
    globalThis.EventSource = originalEventSource
  })
})

describe('SFNotificationsPage contract', () => {
  test('starts navbar realtime from the current auth state without a mount race', () => {
    const navbar = navbarSource()
    expect(navbar).toContain('const stopNotificationUserWatch = watch(user, current =>')
    expect(navbar).toContain('{ immediate: true }')
    expect(navbar).toContain('stopNotificationUserWatch()')
  })

  test('compiles as a focused Page Registry island and keeps the route shell thin', () => {
    const route = source('../../app/pages/notifications/index.vue')
    const page = pageSource()
    expect(page).toContain('<script setup lang="ts">')
    expect(page).toContain('<template>')
    expect(page).toContain('<style scoped src="./SFNotificationsPage.css"></style>')
    expect(route).toContain('definePageMeta({ requiresAuth: true })')
    expect(route).toContain('<SFPageOutlet page="forum.notifications">')
    expect(route).toContain('<SFNotificationsPage />')
    expect(route).not.toContain('useNotifications()')
  })

  test('renders user-authored sources as avatars and system sources as Tabler icons', () => {
    const page = pageSource()
    const styles = pageStyles()

    expect(page).toContain('v-if="item.actor"')
    expect(page).toContain(':avatar="item.actor.avatar"')
    expect(page).toContain('sforum-notifications__source-icon')
    expect(page).not.toContain(':name="filterLabel(item.type)"')
    expect(styles).toContain('.sforum-notifications__source-icon')
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
    expect(page).not.toContain('<SFContentColumnFooter')
    expect(styles).toContain('.sforum-mobile-drawer .sforum-notifications__right--drawer')
    expect(styles).toContain('display: block')
    expect(styles).toContain('.sforum-notifications__main {\n    height: 100%;\n    min-height: 0;\n    overflow-y: auto;')
    expect(styles).toContain('.sforum-notifications__sidebar,\n  .sforum-notifications__right {\n    position: static;')
    expect(styles).toContain('overflow: hidden;')
    expect(page).not.toContain('ssr: false')
  })

  test('uses server-authoritative type filters while preserving cursor pagination', () => {
    const page = pageSource()
    const composable = notificationComposableSource()

    expect(page).toContain('serverFilters(activeFilter.value)')
    expect(page).toContain('notifications.list(beforeId, PAGE_LIMIT, serverFilters(activeFilter.value))')
    expect(page).toContain('notificationFilters')
    expect(composable).toContain("params.set('type', filters.type)")
    expect(composable).toContain("params.set('unread', String(filters.unread))")
    expect(composable).toContain('beforeId')
    expect(composable).toContain('limit')
  })

  test('shares the notification type menu between list and detail sidebars', () => {
    const page = pageSource()
    const detail = detailPageSource()
    const typeNav = typeNavSource()

    expect(page).toContain('<SFNotificationTypeNav')
    expect(detail).toContain('<SFNotificationTypeNav')
    expect(detail).toContain("notifications.list(0, 20)")
    expect(detail).toContain("useState<NotificationFilter>('notification-inbox-filter'")
    expect(detail).toContain("router.push(localePath('/notifications'))")
    expect(typeNav).toContain('v-for="filter in notificationFilters"')
    expect(typeNav).toContain("t('notifications.filter.loadedScope', { count: loadedCount })")
  })

  test('opens an independent detail route and keeps all-read rollback and themed feedback', () => {
    const page = pageSource()

    expect(page).toContain("useState<NotificationFilter>('notification-inbox-filter'")
    expect(page).toContain('router.push(localePath(`/notifications/${item.id}`))')
    expect(page).not.toContain('selectedNotification')
    expect(page).toContain('markingAll.value')
    expect(page).toContain('previousUnreadCount')
    expect(page).toContain('setNotificationRead(previous.id, previous.readAt)')
    expect(page).toContain('notifications.markAllRead()')
    expect(page).toContain("duration: 0")
    expect(page).toContain("duration: 10000")
    expect(page).toContain("color: 'primary'")
    expect(page).toContain("color: 'error'")
  })

  test('uses an authenticated Page Registry route and marks read only after detail mounts', () => {
    const route = detailRouteSource()
    const detail = detailPageSource()

    expect(route).toContain('definePageMeta({ requiresAuth: true })')
    expect(route).toContain('<SFPageOutlet page="forum.notification.show">')
    expect(route).toContain('<SFNotificationDetailPage />')
    expect(route).not.toContain('useNotifications()')
    expect(detail).toContain('notifications.get(notificationId)')
    expect(detail).toContain('apiErrorStatusCode(detailAsync.error.value) === 404')
    expect(detail).toContain('onMounted(async () =>')
    expect(detail).toContain('await notifications.markRead(detail.value.id)')
    expect(detail).toContain('preview.context')
    expect(detail).toContain('router.push(localePath(presented.value.target.path))')
    expect(detail).toContain('data-sforum-island-body="forum.component.notification_detail"')
    expect(detail).toContain('data-layout="fullwidth-3col"')
    expect(detail).not.toContain('<SFContentColumnFooter')
    expect(detail).not.toContain('ssr: false')
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
      expect(locale.notifications.detailPage.repliedComment).toBeTruthy()
      expect(locale.notifications.detailPage.originalTopic).toBeTruthy()
      expect(locale.notifications.targetUnavailableHelp).toBeTruthy()
    }
  })
})
