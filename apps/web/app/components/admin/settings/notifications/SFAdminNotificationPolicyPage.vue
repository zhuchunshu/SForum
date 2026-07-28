<script setup lang="ts">
import { apiErrorMessage } from '~/composables/useApiClient'
import { usePermissions } from '~/composables/identity/usePermissions'
import {
  canEditNotificationPolicy,
  notificationPolicyKey,
  notificationPolicyUpdates,
  type NotificationPolicyCatalog,
  type NotificationPolicyItem
} from '~/components/admin/settings/notifications/model'
import SFAdminNotificationChannels from '~/components/admin/settings/notifications/SFAdminNotificationChannels.vue'

const { t, te } = useI18n()
const toast = useToast()
const { request } = useApiClient()
const { can } = usePermissions()
const canManage = computed(() => can('settings.notifications.manage'))

const emptyCatalog = (): NotificationPolicyCatalog => ({ revision: 0, items: [] })
const catalog = ref<NotificationPolicyCatalog>(emptyCatalog())
const pending = ref(false)
const saving = ref(false)
const restoring = ref(false)
const errorMessage = ref('')
const snapshot = ref('')
const channelsRef = ref<{ refresh?: () => Promise<void>, pending?: boolean } | null>(null)

const categories = computed(() => {
  const groups = new Map<string, NotificationPolicyItem[]>()
  for (const item of catalog.value.items) {
    const items = groups.get(item.category) || []
    items.push(item)
    groups.set(item.category, items)
  }
  return [...groups.entries()].map(([category, items]) => ({ category, items }))
})
const hasChanges = computed(() => policySnapshot() !== snapshot.value)

function policySnapshot() {
  return JSON.stringify(catalog.value.items.map(item => [
    notificationPolicyKey(item), item.enabled, item.recommendedEnabled, item.userConfigurable
  ]).sort(([left], [right]) => String(left).localeCompare(String(right))))
}

function captureSnapshot() {
  snapshot.value = policySnapshot()
}

function label(prefix: string, value: string) {
  const key = `${prefix}.${value}`
  return te(key) ? t(key) : value
}

function categoryLabel(category: string) {
  return label('admin.notificationSettings.categories', category)
}

function typeLabel(item: NotificationPolicyItem) {
  const key = `admin.notificationSettings.types.${item.type}`
  return te(key) ? t(key) : item.type
}

function channelLabel(channel: string) {
  return label('admin.notificationSettings.channels', channel)
}

function itemStatus(item: NotificationPolicyItem) {
  if (item.required) return t('admin.notificationSettings.status.required')
  if (!item.active) return t('admin.notificationSettings.status.inactive')
  if (item.channelAvailable === false) return t('admin.notificationSettings.status.channelUnavailable')
  return item.enabled ? t('admin.notificationSettings.status.enabled') : t('admin.notificationSettings.status.disabled')
}

function applyCategory(category: string, enabled: boolean) {
  if (!canManage.value) return
  for (const item of catalog.value.items) {
    if (item.category === category && canEditNotificationPolicy(item)) {
      item.enabled = enabled
      if (enabled) item.recommendedEnabled = true
    }
  }
}

function resetChanges() {
  if (!snapshot.value) return
  const saved = JSON.parse(snapshot.value) as [string, boolean, boolean, boolean][]
  const byKey = new Map(saved.map(([key, enabled, recommended, configurable]) => [
    key,
    { enabled, recommended, configurable }
  ] as const))
  for (const item of catalog.value.items) {
    const state = byKey.get(notificationPolicyKey(item))
    if (!state) continue
    item.enabled = state.enabled
    item.recommendedEnabled = state.recommended
    item.userConfigurable = state.configurable
  }
  errorMessage.value = ''
  toast.add({ color: 'neutral', icon: 'i-lucide-rotate-ccw', title: t('admin.notificationSettings.resetChanges'), duration: 10000 })
}

async function load() {
  pending.value = true
  errorMessage.value = ''
  try {
    catalog.value = await request<NotificationPolicyCatalog>('/admin/notifications/policy')
    captureSnapshot()
  } catch (loadError) {
    errorMessage.value = apiErrorMessage(loadError) || t('admin.notificationSettings.loadFailed')
  } finally {
    pending.value = false
  }
}

async function save() {
  if (!canManage.value || !hasChanges.value || saving.value) return
  saving.value = true
  errorMessage.value = ''
  try {
    catalog.value = await request<NotificationPolicyCatalog>('/admin/notifications/policy', {
      method: 'PUT',
      body: { revision: catalog.value.revision, items: notificationPolicyUpdates(catalog.value.items) }
    })
    captureSnapshot()
    toast.add({ color: 'success', icon: 'i-lucide-check', title: t('admin.notificationSettings.saved'), duration: 10000 })
  } catch (saveError) {
    errorMessage.value = apiErrorMessage(saveError) || t('admin.notificationSettings.saveFailed')
  } finally {
    saving.value = false
  }
}

async function restoreDefaults() {
  if (!canManage.value || restoring.value) return
  restoring.value = true
  errorMessage.value = ''
  try {
    catalog.value = await request<NotificationPolicyCatalog>('/admin/notifications/policy/restore', {
      method: 'POST',
      body: { revision: catalog.value.revision }
    })
    captureSnapshot()
    toast.add({ color: 'success', icon: 'i-lucide-rotate-ccw', title: t('admin.notificationSettings.restored'), description: t('admin.notificationSettings.secretsPreserved'), duration: 10000 })
  } catch (restoreError) {
    errorMessage.value = apiErrorMessage(restoreError) || t('admin.notificationSettings.restoreFailed')
  } finally {
    restoring.value = false
  }
}

async function createTestNotification() {
  if (!canManage.value) return
  errorMessage.value = ''
  try {
    await request('/admin/notifications/test', { method: 'POST' })
    toast.add({ color: 'success', icon: 'i-lucide-bell', title: t('admin.notificationSettings.testCreated'), duration: 10000 })
  } catch (testError) {
    errorMessage.value = apiErrorMessage(testError) || t('admin.notificationSettings.testFailed')
  }
}

async function refresh() {
  await Promise.all([load(), channelsRef.value?.refresh?.()])
}

defineExpose({ refresh, pending })
onMounted(load)
</script>

<template>
  <div class="space-y-5">
    <UAlert v-if="errorMessage" color="error" variant="soft" icon="i-lucide-triangle-alert" :title="errorMessage" />
    <UAlert v-if="!canManage" color="warning" variant="soft" icon="i-lucide-lock-keyhole" :title="t('admin.notificationSettings.noPermission')" />

    <section class="rounded-md border border-teal-200 bg-teal-50/70 p-4 dark:border-teal-900/60 dark:bg-teal-950/25">
      <div class="flex gap-3">
        <UIcon name="i-lucide-sparkles" class="size-5 shrink-0 text-teal-700 dark:text-teal-300" />
        <div><h2 class="font-semibold">{{ t('admin.notificationSettings.recommendedTitle') }}</h2><p class="mt-1 text-sm text-muted">{{ t('admin.notificationSettings.recommendedDescription') }}</p></div>
      </div>
    </section>

    <div v-if="pending && catalog.items.length === 0" class="space-y-3" aria-busy="true">
      <SFSkeleton v-for="index in 3" :key="index" class="h-36 w-full" />
    </div>

    <SFEmptyState
      v-else-if="!pending && !errorMessage && categories.length === 0"
      icon-label="NOTIFY"
      :title="t('admin.notificationSettings.emptyTitle')"
      :description="t('admin.notificationSettings.emptyDescription')"
    />

    <section v-for="group in categories" :key="group.category" class="overflow-hidden rounded-md border border-slate-200 bg-white dark:border-zinc-800 dark:bg-zinc-900">
      <header class="flex flex-col gap-3 border-b border-slate-200 px-4 py-3 dark:border-zinc-800 sm:flex-row sm:items-center sm:justify-between">
        <div><h2 class="text-base font-semibold">{{ categoryLabel(group.category) }}</h2><p class="mt-1 text-xs text-muted">{{ t('admin.notificationSettings.categoryHelp') }}</p></div>
        <div class="flex flex-wrap gap-2">
          <UButton color="neutral" variant="outline" size="xs" icon="i-lucide-check-check" :disabled="!canManage" @click="applyCategory(group.category, true)">{{ t('admin.notificationSettings.enableCategory') }}</UButton>
          <UButton color="neutral" variant="outline" size="xs" icon="i-lucide-bell-off" :disabled="!canManage" @click="applyCategory(group.category, false)">{{ t('admin.notificationSettings.disableCategory') }}</UButton>
        </div>
      </header>
      <ul class="divide-y divide-slate-200 dark:divide-zinc-800">
        <li v-for="item in group.items" :key="notificationPolicyKey(item)" class="grid gap-4 px-4 py-4 lg:grid-cols-[minmax(12rem,1fr)_auto_auto_auto] lg:items-center">
          <div class="min-w-0">
            <div class="flex flex-wrap items-center gap-2"><h3 class="font-medium">{{ typeLabel(item) }}</h3><UBadge v-if="item.required" color="neutral" variant="soft">{{ t('admin.notificationSettings.required') }}</UBadge><UBadge v-else-if="item.ownerExtensionId" color="neutral" variant="soft">{{ item.ownerLabel || item.ownerExtensionId }}</UBadge></div>
            <p class="mt-1 text-sm text-muted">{{ channelLabel(item.channel) }} · {{ itemStatus(item) }}</p>
            <p v-if="item.ownerExtensionId" class="mt-1 text-xs text-muted">{{ t('admin.notificationSettings.pluginOwned') }}</p>
          </div>
          <UCheckbox v-model="item.enabled" :disabled="!canManage || !canEditNotificationPolicy(item)" :label="t('admin.notificationSettings.channelEnabled')" />
          <UCheckbox v-model="item.recommendedEnabled" :disabled="!canManage || !canEditNotificationPolicy(item) || !item.enabled" :label="t('admin.notificationSettings.recommendedEnabled')" />
          <UCheckbox v-model="item.userConfigurable" :disabled="!canManage || !canEditNotificationPolicy(item)" :label="t('admin.notificationSettings.userConfigurable')" />
        </li>
      </ul>
    </section>

    <div class="flex flex-col gap-3 border-t border-slate-200 pt-5 dark:border-zinc-800 sm:flex-row sm:items-center sm:justify-between">
      <p class="text-xs text-muted">{{ t('admin.notificationSettings.restoreHelp') }}</p>
      <div class="flex flex-wrap gap-2">
        <UButton color="neutral" variant="ghost" icon="i-lucide-bell" :disabled="!canManage" @click="createTestNotification">{{ t('admin.notificationSettings.testNotification') }}</UButton>
        <UButton color="neutral" variant="outline" icon="i-lucide-rotate-ccw" :loading="restoring" :disabled="!canManage" @click="restoreDefaults">{{ t('admin.notificationSettings.restoreDefaults') }}</UButton>
        <UButton color="primary" icon="i-lucide-save" :loading="saving" :disabled="!canManage || !hasChanges" @click="save">{{ t('admin.notificationSettings.save') }}</UButton>
      </div>
    </div>

    <SFAdminNotificationChannels ref="channelsRef" :can-manage="canManage" />
  </div>
</template>
