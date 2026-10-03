// 通知实时客户端：私信/回复等收件人级修订信号，消费通用 revisionStream 运行时。
// 协调键按「收件人 + API 基址」隔离，避免多账号标签页共享同一条连接。
import {
  createRevisionStreamClient,
  type RevisionStreamEnvironment
} from '~/utils/realtime/revisionStream'

export type NotificationRevisionState = { value: number }
export type NotificationRevisionRefresh = () => void | Promise<void>
export type NotificationRealtimeEnvironment = RevisionStreamEnvironment

export type NotificationRealtimeSubscription = {
  actorUserId: number
  apiBaseUrl: string
  revision: NotificationRevisionState
  refresh: NotificationRevisionRefresh
}

const FALLBACK_REFRESH_MS = 30_000
const CHANNEL_PREFIX = 'sforum:notifications:revision:user:'

function normalizeBaseURL(value: string) {
  return value.replace(/\/+$/, '')
}

function runtimeKey(actorUserId: number, apiBaseUrl: string) {
  return `${CHANNEL_PREFIX}${actorUserId}:${normalizeBaseURL(apiBaseUrl)}`
}

function extractRevision(payload: unknown): number | null {
  if (!payload || typeof payload !== 'object') return null
  const revision = Number((payload as { revision?: unknown }).revision)
  return Number.isSafeInteger(revision) && revision >= 0 ? revision : null
}

export function createNotificationRealtimeClient(environment?: NotificationRealtimeEnvironment) {
  const client = environment
    ? createRevisionStreamClient(environment)
    : createRevisionStreamClient()

  function subscribe(subscription: NotificationRealtimeSubscription) {
    if (!Number.isSafeInteger(subscription.actorUserId) || subscription.actorUserId <= 0) return () => {}
    const baseURL = normalizeBaseURL(subscription.apiBaseUrl)
    return client.subscribe({
      // 同一用户的多个标签页共用一条 EventSource；不同用户互不干扰。
      coordinationKey: runtimeKey(subscription.actorUserId, subscription.apiBaseUrl),
      streamUrl: revision => `${baseURL}/notifications/stream?revision=${revision}`,
      eventName: 'revision',
      revision: subscription.revision,
      extractRevision,
      refresh: subscription.refresh,
      fallbackIntervalMs: FALLBACK_REFRESH_MS
    })
  }

  return {
    subscribe,
    stopAll: () => client.stopAll()
  }
}
