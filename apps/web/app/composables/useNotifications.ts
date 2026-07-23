export type NotificationItem = {
  id: number
  type: 'reply' | 'mention' | 'moderation_approved' | 'moderation_rejected' | 'admin_test'
  actorUserId?: number
  targetType: string
  targetId: number
  payload: Record<string, unknown>
  readAt?: string
  createdAt: string
}

export function useNotifications() {
  const { request } = useApiClient()
  const unreadCount = useState<number>('notification-unread-count', () => 0)
  async function refreshUnreadCount() { const data = await request<{ count: number }>('/notifications/unread-count'); unreadCount.value = data.count; return data.count }
  async function list(beforeId = 0, limit = 20) {
    const params = new URLSearchParams()
    if (beforeId > 0) params.set('beforeId', String(beforeId))
    if (limit > 0) params.set('limit', String(limit))
    const query = params.toString()
    return request<{ items: NotificationItem[], hasMore: boolean }>(`/notifications${query ? `?${query}` : ''}`)
  }
  async function markRead(id: number) { await request(`/notifications/${id}/read`, { method: 'PATCH' }); if (unreadCount.value > 0) unreadCount.value-- }
  async function markAllRead() { const data = await request<{ updated: number }>('/notifications/read-all', { method: 'POST' }); unreadCount.value = 0; return data.updated }
  return { unreadCount, refreshUnreadCount, list, markRead, markAllRead }
}
