export type NotificationItem = {
  id: number
  type: string
  category?: string
  typeVersion?: number
  payloadVersion?: number
  actorUserId?: number
  targetType: string
  targetId: number
  targetAvailable?: boolean
  targetPath?: string
  payload: Record<string, unknown>
  readAt?: string
  createdAt: string
}

export type NotificationPreviewAuthor = {
  id: number
  username: string
  displayName: string
  avatar: {
    kind: 'uploaded' | 'initials' | 'gravatar' | 'static'
    url: string
    attachmentId?: number | null
    alt: string
  }
}

export type NotificationPreviewContent = {
  type: 'topic' | 'comment'
  id: number
  excerpt: string
  author?: NotificationPreviewAuthor
}

export type NotificationDetail = NotificationItem & {
  preview?: {
    topicId: number
    topicTitle: string
    content: NotificationPreviewContent
    context?: NotificationPreviewContent
  }
}

export type NotificationListFilters = {
  category?: string
  type?: string
  unread?: boolean
}

type RevisionPayload = { revision?: unknown }
type RevisionRefresh = () => void | Promise<void>

const REVISION_RECONNECT_DELAY_MS = 1000
const REVISION_FALLBACK_REFRESH_MS = 30_000

let revisionSource: EventSource | null = null
let revisionURL = ''
let revisionRefreshTimer: ReturnType<typeof setTimeout> | null = null
let revisionReconnectTimer: ReturnType<typeof setTimeout> | null = null
let revisionFallbackTimer: ReturnType<typeof setInterval> | null = null
let revisionReconnect: (() => void) | null = null
let visibilityListenerAttached = false
const revisionRefreshers = new Set<RevisionRefresh>()

function scheduleRevisionRefresh() {
  if (revisionRefreshTimer) return
  revisionRefreshTimer = setTimeout(() => {
    revisionRefreshTimer = null
    for (const refresh of revisionRefreshers) void Promise.resolve(refresh()).catch(() => {})
  }, 100)
}

function refreshWhenVisible() {
  if (typeof document !== 'undefined' && document.visibilityState !== 'visible') return
  scheduleRevisionRefresh()
  revisionReconnect?.()
}

function startRealtimeFallbacks() {
  if (!revisionFallbackTimer) {
    revisionFallbackTimer = setInterval(refreshWhenVisible, REVISION_FALLBACK_REFRESH_MS)
  }
  if (typeof document !== 'undefined' && !visibilityListenerAttached) {
    document.addEventListener('visibilitychange', refreshWhenVisible)
    visibilityListenerAttached = true
  }
}

function stopRealtimeFallbacks() {
  if (revisionRefreshTimer) clearTimeout(revisionRefreshTimer)
  if (revisionReconnectTimer) clearTimeout(revisionReconnectTimer)
  if (revisionFallbackTimer) clearInterval(revisionFallbackTimer)
  revisionRefreshTimer = null
  revisionReconnectTimer = null
  revisionFallbackTimer = null
  revisionReconnect = null
  if (typeof document !== 'undefined' && visibilityListenerAttached) {
    document.removeEventListener('visibilitychange', refreshWhenVisible)
    visibilityListenerAttached = false
  }
}

export function useNotifications() {
  const { apiBaseUrl, request } = useApiClient()
  const unreadCount = useState<number>('notification-unread-count', () => 0)
  const revision = useState<number>('notification-recipient-revision', () => 0)
  async function refreshUnreadCount() { const data = await request<{ count: number }>('/notifications/unread-count'); unreadCount.value = data.count; return data.count }
  async function list(beforeId = 0, limit = 20, filters: NotificationListFilters = {}) {
    const params = new URLSearchParams()
    if (beforeId > 0) params.set('beforeId', String(beforeId))
    if (limit > 0) params.set('limit', String(limit))
    if (filters.category) params.set('category', filters.category)
    if (filters.type) params.set('type', filters.type)
    if (filters.unread !== undefined) params.set('unread', String(filters.unread))
    const query = params.toString()
    return request<{ items: NotificationItem[], hasMore: boolean }>(`/notifications${query ? `?${query}` : ''}`)
  }
  async function get(id: number) {
    return request<NotificationDetail>(`/notifications/${id}`, { serverInternal: import.meta.server })
  }
  async function markRead(id: number) { await request(`/notifications/${id}/read`, { method: 'PATCH' }); if (unreadCount.value > 0) unreadCount.value-- }
  async function markAllRead() { const data = await request<{ updated: number }>('/notifications/read-all', { method: 'POST' }); unreadCount.value = 0; return data.updated }

  function startRealtime(refresh: RevisionRefresh) {
    if (import.meta.server) return () => {}
    revisionRefreshers.add(refresh)
    const base = apiBaseUrl.replace(/\/+$/, '')
    const connect = () => {
      if (revisionRefreshers.size === 0) return
      const url = `${base}/notifications/stream?revision=${revision.value}`
      if (revisionSource && revisionURL === url && revisionSource.readyState !== EventSource.CLOSED) return
      revisionSource?.close()
      revisionURL = url
      let source: EventSource
      try {
        source = new EventSource(url, { withCredentials: true })
        revisionSource = source
      } catch {
        revisionSource = null
        revisionURL = ''
        if (!revisionReconnectTimer) {
          revisionReconnectTimer = setTimeout(() => {
            revisionReconnectTimer = null
            connect()
          }, REVISION_RECONNECT_DELAY_MS)
        }
        return
      }
      source.addEventListener('open', scheduleRevisionRefresh)
      source.addEventListener('error', () => {
        // EventSource 会先自行重连；同步一次 REST，避免 badge 在断线期间冻结。
        scheduleRevisionRefresh()
        if (revisionSource !== source || source.readyState !== EventSource.CLOSED || revisionReconnectTimer) return
        source.close()
        revisionSource = null
        revisionURL = ''
        revisionReconnectTimer = setTimeout(() => {
          revisionReconnectTimer = null
          connect()
        }, REVISION_RECONNECT_DELAY_MS)
      })
      source.addEventListener('revision', (event) => {
        try {
          const payload = JSON.parse((event as MessageEvent<string>).data) as RevisionPayload
          const next = Number(payload.revision)
          if (Number.isSafeInteger(next) && next >= 0 && next !== revision.value) {
            revision.value = next
            scheduleRevisionRefresh()
          }
        } catch {
          // 畸形或旧服务端事件不影响手动 REST 刷新。
        }
      })
    }
    revisionReconnect = connect
    startRealtimeFallbacks()
    connect()
    return () => {
      revisionRefreshers.delete(refresh)
      if (revisionRefreshers.size === 0) {
        revisionSource?.close()
        revisionSource = null
        revisionURL = ''
        stopRealtimeFallbacks()
      }
    }
  }

  function stopRealtime() {
    revisionRefreshers.clear()
    revisionSource?.close()
    revisionSource = null
    revisionURL = ''
    stopRealtimeFallbacks()
  }

  return { unreadCount, revision, refreshUnreadCount, list, get, markRead, markAllRead, startRealtime, stopRealtime }
}
