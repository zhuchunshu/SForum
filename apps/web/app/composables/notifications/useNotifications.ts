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

export type NotificationListFilters = {
  category?: string
  type?: string
  unread?: boolean
}

type RevisionPayload = { revision?: unknown }
type RevisionRefresh = () => void | Promise<void>

let revisionSource: EventSource | null = null
let revisionURL = ''
let revisionRefreshTimer: ReturnType<typeof setTimeout> | null = null
const revisionRefreshers = new Set<RevisionRefresh>()

function scheduleRevisionRefresh() {
  if (revisionRefreshTimer) return
  revisionRefreshTimer = setTimeout(() => {
    revisionRefreshTimer = null
    for (const refresh of revisionRefreshers) void Promise.resolve(refresh()).catch(() => {})
  }, 100)
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
  async function markRead(id: number) { await request(`/notifications/${id}/read`, { method: 'PATCH' }); if (unreadCount.value > 0) unreadCount.value-- }
  async function markAllRead() { const data = await request<{ updated: number }>('/notifications/read-all', { method: 'POST' }); unreadCount.value = 0; return data.updated }

  function startRealtime(refresh: RevisionRefresh) {
    if (import.meta.server) return () => {}
    revisionRefreshers.add(refresh)
    const base = apiBaseUrl.replace(/\/+$/, '')
    const url = `${base}/notifications/stream?revision=${revision.value}`
    if (!revisionSource || revisionURL !== url) {
      revisionSource?.close()
      revisionURL = url
      try {
        revisionSource = new EventSource(url, { withCredentials: true })
      } catch {
        revisionSource = null
        revisionURL = ''
        return () => revisionRefreshers.delete(refresh)
      }
      revisionSource.addEventListener('revision', (event) => {
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
    return () => {
      revisionRefreshers.delete(refresh)
      if (revisionRefreshers.size === 0) {
        revisionSource?.close()
        revisionSource = null
        revisionURL = ''
      }
    }
  }

  function stopRealtime() {
    revisionRefreshers.clear()
    revisionSource?.close()
    revisionSource = null
    revisionURL = ''
  }

  return { unreadCount, revision, refreshUnreadCount, list, markRead, markAllRead, startRealtime, stopRealtime }
}
