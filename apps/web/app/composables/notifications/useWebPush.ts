import { useApiClient } from '~/composables/useApiClient'

export const WEB_PUSH_WORKER_PATH = '/_sforum/notifications/sw.js'
export const WEB_PUSH_SCOPE = '/_sforum/notifications/'

export type WebPushConfig = {
  available: boolean
  publicKey?: string
  workerPath: typeof WEB_PUSH_WORKER_PATH
  scope: typeof WEB_PUSH_SCOPE
}

export type WebPushSubscriptionItem = {
  id: number
  endpointOrigin: string
  contentEncoding: 'aes128gcm'
  status: 'active' | 'revoked' | 'failed'
  lastFailureReason?: string
  createdAt: string
  updatedAt: string
  revokedAt?: string
}

export function resolveCurrentWebPushSubscriptionId(items: WebPushSubscriptionItem[], storedValue: string | null) {
  const stored = Number(storedValue)
  return Number.isSafeInteger(stored) && stored > 0 && items.some(item => item.id === stored && item.status === 'active')
    ? stored
    : null
}

export function decodeApplicationServerKey(value: string) {
  const padded = `${value}${'='.repeat((4 - value.length % 4) % 4)}`
  const binary = atob(padded.replace(/-/g, '+').replace(/_/g, '/'))
  return Uint8Array.from(binary, character => character.charCodeAt(0))
}

export function serializePushSubscription(subscription: PushSubscription) {
  const payload = subscription.toJSON()
  if (!payload.endpoint || !payload.keys?.p256dh || !payload.keys.auth) {
    throw new Error('notification.subscription_invalid')
  }
  return { endpoint: payload.endpoint, keys: { p256dh: payload.keys.p256dh, auth: payload.keys.auth } }
}

export function useWebPush() {
  const { request } = useApiClient()

  const config = () => request<WebPushConfig>('/web-push/config')
  const subscriptions = () => request<{ items: WebPushSubscriptionItem[] }>('/web-push/subscriptions')
  const create = (body: ReturnType<typeof serializePushSubscription>) => request<WebPushSubscriptionItem>('/web-push/subscriptions', { method: 'POST', body })
  const revoke = (id: number) => request<{ revoked: true }>(`/web-push/subscriptions/${id}`, { method: 'DELETE' })

  return { config, subscriptions, create, revoke }
}
