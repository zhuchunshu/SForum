<script setup lang="ts">
import { apiErrorMessage } from '~/composables/useApiClient'
import {
  decodeApplicationServerKey,
  resolveCurrentWebPushSubscriptionId,
  serializePushSubscription,
  useWebPush,
  WEB_PUSH_SCOPE,
  WEB_PUSH_WORKER_PATH,
  type WebPushConfig,
  type WebPushSubscriptionItem
} from '~/composables/notifications/useWebPush'

const STORAGE_KEY = 'sforum.webPush.subscriptionId'
const { t } = useI18n()
const toast = useToast()
const api = useWebPush()
const config = ref<WebPushConfig | null>(null)
const subscriptions = ref<WebPushSubscriptionItem[]>([])
const permission = ref<NotificationPermission | 'unsupported'>('unsupported')
const browserSubscribed = ref(false)
const currentSubscriptionId = ref<number | null>(null)
const pending = ref(true)
const mutating = ref(false)
const revokingId = ref<number | null>(null)
const errorMessage = ref('')

const supported = computed(() => permission.value !== 'unsupported')
const canSubscribe = computed(() => supported.value && config.value?.available === true && permission.value !== 'denied')
const permissionLabel = computed(() => t(`notificationSettings.webPush.permission.${permission.value}`))

async function inspectBrowserSubscription() {
  if (!supported.value) return
  const registration = await navigator.serviceWorker.getRegistration(WEB_PUSH_SCOPE)
  const browserSubscription = await registration?.pushManager.getSubscription()
  currentSubscriptionId.value = resolveCurrentWebPushSubscriptionId(subscriptions.value, localStorage.getItem(STORAGE_KEY))
  browserSubscribed.value = Boolean(browserSubscription && currentSubscriptionId.value)
}

async function load() {
  pending.value = true
  errorMessage.value = ''
  try {
    const [nextConfig, result] = await Promise.all([api.config(), api.subscriptions()])
    config.value = nextConfig
    subscriptions.value = result.items
    await inspectBrowserSubscription()
  } catch (error) {
    errorMessage.value = apiErrorMessage(error) || t('notificationSettings.webPush.loadFailed')
  } finally {
    pending.value = false
  }
}

async function subscribeBrowser() {
  if (!canSubscribe.value || !config.value?.publicKey || mutating.value) return
  mutating.value = true
  errorMessage.value = ''
  let pushSubscription: PushSubscription | null = null
  let createdBrowserSubscription = false
  try {
    const nextPermission = await Notification.requestPermission()
    permission.value = nextPermission
    if (nextPermission !== 'granted') throw new Error('notification.permission_denied')
    if (config.value.workerPath !== WEB_PUSH_WORKER_PATH || config.value.scope !== WEB_PUSH_SCOPE) {
      throw new Error('notification.worker_contract_invalid')
    }
    const registration = await navigator.serviceWorker.register(WEB_PUSH_WORKER_PATH, { scope: WEB_PUSH_SCOPE })
    pushSubscription = await registration.pushManager.getSubscription()
    if (!pushSubscription) {
      pushSubscription = await registration.pushManager.subscribe({
        userVisibleOnly: true,
        applicationServerKey: decodeApplicationServerKey(config.value.publicKey)
      })
      createdBrowserSubscription = true
    }
    const item = await api.create(serializePushSubscription(pushSubscription))
    localStorage.setItem(STORAGE_KEY, String(item.id))
    await load()
    toast.add({ color: 'success', icon: 'i-lucide-bell-ring', title: t('notificationSettings.webPush.subscribed'), duration: 10000 })
  } catch (error) {
    if (pushSubscription && createdBrowserSubscription) await pushSubscription.unsubscribe().catch(() => false)
    errorMessage.value = error instanceof Error && error.message === 'notification.permission_denied'
      ? t('notificationSettings.webPush.permissionDenied')
      : apiErrorMessage(error) || t('notificationSettings.webPush.subscribeFailed')
  } finally {
    mutating.value = false
  }
}

async function unsubscribeBrowser() {
  if (!supported.value || mutating.value) return
  mutating.value = true
  errorMessage.value = ''
  try {
    const registration = await navigator.serviceWorker.getRegistration(WEB_PUSH_SCOPE)
    const browserSubscription = await registration?.pushManager.getSubscription()
    if (currentSubscriptionId.value) await api.revoke(currentSubscriptionId.value)
    await browserSubscription?.unsubscribe()
    localStorage.removeItem(STORAGE_KEY)
    await load()
    toast.add({ color: 'success', icon: 'i-lucide-bell-off', title: t('notificationSettings.webPush.unsubscribed'), duration: 10000 })
  } catch (error) {
    errorMessage.value = apiErrorMessage(error) || t('notificationSettings.webPush.unsubscribeFailed')
  } finally {
    mutating.value = false
  }
}

async function revoke(item: WebPushSubscriptionItem) {
  if (revokingId.value !== null) return
  revokingId.value = item.id
  errorMessage.value = ''
  try {
    await api.revoke(item.id)
    if (currentSubscriptionId.value === item.id) {
      const registration = await navigator.serviceWorker.getRegistration(WEB_PUSH_SCOPE)
      await (await registration?.pushManager.getSubscription())?.unsubscribe()
      localStorage.removeItem(STORAGE_KEY)
    }
    await load()
    toast.add({ color: 'success', icon: 'i-lucide-trash-2', title: t('notificationSettings.webPush.revoked'), duration: 10000 })
  } catch (error) {
    errorMessage.value = apiErrorMessage(error) || t('notificationSettings.webPush.revokeFailed')
  } finally {
    revokingId.value = null
  }
}

onMounted(() => {
  permission.value = 'Notification' in window && 'serviceWorker' in navigator && 'PushManager' in window
    ? Notification.permission
    : 'unsupported'
  load()
})
</script>

<template>
  <section class="mt-5 shrink-0 overflow-hidden rounded-md border border-slate-200 bg-white dark:border-zinc-800 dark:bg-zinc-900" data-testid="web-push-settings">
    <header class="flex flex-col gap-3 border-b border-slate-200 px-4 py-4 dark:border-zinc-800 sm:flex-row sm:items-start sm:justify-between">
      <div class="min-w-0">
        <div class="flex flex-wrap items-center gap-2">
          <UIcon name="i-lucide-smartphone" class="size-5 text-[var(--sf-accent)]" />
          <h2 class="font-semibold">{{ t('notificationSettings.webPush.title') }}</h2>
          <UBadge color="neutral" variant="soft">{{ permissionLabel }}</UBadge>
        </div>
        <p class="mt-1 text-sm text-muted">{{ t('notificationSettings.webPush.description') }}</p>
      </div>
      <UButton
        v-if="browserSubscribed"
        color="neutral"
        variant="outline"
        icon="i-lucide-bell-off"
        :loading="mutating"
        @click="unsubscribeBrowser"
      >{{ t('notificationSettings.webPush.disableBrowser') }}</UButton>
      <UButton
        v-else
        icon="i-lucide-bell-ring"
        :disabled="!canSubscribe || pending"
        :loading="mutating"
        @click="subscribeBrowser"
      >{{ t('notificationSettings.webPush.enableBrowser') }}</UButton>
    </header>

    <div class="space-y-3 px-4 py-4">
      <UAlert v-if="errorMessage" color="error" variant="soft" icon="i-lucide-triangle-alert" :title="errorMessage" />
      <UAlert v-else-if="!supported" color="warning" variant="soft" icon="i-lucide-monitor-x" :title="t('notificationSettings.webPush.unsupported')" />
      <UAlert v-else-if="!config?.available" color="neutral" variant="soft" icon="i-lucide-plug-zap" :title="t('notificationSettings.webPush.unavailable')" />
      <UAlert v-else-if="permission === 'denied'" color="warning" variant="soft" icon="i-lucide-bell-off" :title="t('notificationSettings.webPush.permissionDenied')" />

      <div class="flex items-center justify-between gap-3">
        <h3 class="text-sm font-medium">{{ t('notificationSettings.webPush.devices') }}</h3>
        <UButton color="neutral" variant="ghost" size="xs" icon="i-lucide-refresh-cw" :loading="pending" @click="load">{{ t('notificationSettings.refresh') }}</UButton>
      </div>
      <p v-if="!pending && subscriptions.length === 0" class="text-sm text-muted">{{ t('notificationSettings.webPush.empty') }}</p>
      <ul v-else class="divide-y divide-slate-200 dark:divide-zinc-800">
        <li v-for="item in subscriptions" :key="item.id" class="flex flex-col gap-3 py-3 sm:flex-row sm:items-center sm:justify-between">
          <div class="min-w-0">
            <div class="flex flex-wrap items-center gap-2">
              <span class="truncate text-sm font-medium">{{ item.endpointOrigin }}</span>
              <UBadge :color="item.status === 'active' ? 'success' : item.status === 'failed' ? 'error' : 'neutral'" variant="soft">{{ t(`notificationSettings.webPush.status.${item.status}`) }}</UBadge>
              <UBadge v-if="currentSubscriptionId === item.id" color="primary" variant="soft">{{ t('notificationSettings.webPush.thisBrowser') }}</UBadge>
            </div>
            <p class="mt-1 text-xs text-muted">{{ t('notificationSettings.webPush.updatedAt', { value: new Date(item.updatedAt).toLocaleString() }) }}</p>
          </div>
          <UButton v-if="item.status === 'active'" color="error" variant="ghost" size="xs" icon="i-lucide-trash-2" :loading="revokingId === item.id" @click="revoke(item)">{{ t('notificationSettings.webPush.revoke') }}</UButton>
        </li>
      </ul>
      <p class="text-xs text-muted">{{ t('notificationSettings.webPush.privacy') }}</p>
    </div>
  </section>
</template>
