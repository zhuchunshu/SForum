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

const categories = computed(() => {
  const groups = new Map<string, NotificationPolicyItem[]>()
  for (const item of catalog.value.items) {
    const items = groups.get(item.category) || []
    items.push(item)
    groups.set(item.category, items)
  }
  return [...groups.entries()].map(([category, items]) => ({ category, items }))
})
const hasChanges = computed(() => Boolean(snapshot.value) && policySnapshot() !== snapshot.value)

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

function channelEnabledLabel(item: NotificationPolicyItem) {
  return item.channel === 'email'
    ? t('admin.notificationSettings.emailEnabled')
    : t('admin.notificationSettings.channelEnabled')
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
      item.recommendedEnabled = enabled
    }
  }
}

function setChannelEnabled(item: NotificationPolicyItem, enabled: boolean | 'indeterminate') {
  item.enabled = enabled === true
  if (!item.enabled) item.recommendedEnabled = false
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

defineExpose({ refresh: load, pending })
onMounted(load)
</script>

<template>
  <form class="flex flex-col" @submit.prevent="save">
    <UCard
      class="border-slate-200 bg-white text-slate-900 dark:border-zinc-800 dark:bg-zinc-900 dark:text-zinc-100"
      :ui="{ footer: 'sticky bottom-0 z-20 bg-white/95 dark:bg-zinc-900/95 backdrop-blur-sm border-t border-slate-200 dark:border-zinc-800 p-4 sm:px-6' }"
    >
      <template #header>
        <div>
          <h2 class="text-base font-bold text-slate-900 dark:text-white">{{ t('admin.notificationSettings.policyTitle') }}</h2>
          <p class="mt-1 text-xs text-slate-500 dark:text-zinc-400">{{ t('admin.notificationSettings.policyDescription') }}</p>
        </div>
      </template>

      <div class="space-y-5">
        <UAlert v-if="errorMessage" color="error" variant="soft" icon="i-lucide-triangle-alert" :title="errorMessage" />
        <UAlert v-if="!canManage" color="warning" variant="soft" icon="i-lucide-lock-keyhole" :title="t('admin.notificationSettings.noPermission')" />
        <UAlert
          color="primary"
          variant="soft"
          icon="i-lucide-sparkles"
          :title="t('admin.notificationSettings.recommendedTitle')"
          :description="t('admin.notificationSettings.recommendedDescription')"
        />

        <div v-if="pending && catalog.items.length === 0" class="space-y-3" aria-busy="true">
          <SFSkeleton v-for="index in 3" :key="index" class="h-28 w-full" />
        </div>

        <SFEmptyState
          v-else-if="!pending && !errorMessage && categories.length === 0"
          icon-label="NTF"
          :title="t('admin.notificationSettings.emptyTitle')"
          :description="t('admin.notificationSettings.emptyDescription')"
        />

        <div v-else class="divide-y divide-slate-200 border-y border-slate-200 dark:divide-zinc-800 dark:border-zinc-800">
          <section v-for="group in categories" :key="group.category" class="py-4">
            <header class="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
              <div>
                <h3 class="text-sm font-semibold">{{ categoryLabel(group.category) }}</h3>
                <p class="mt-1 text-xs text-muted">{{ t('admin.notificationSettings.categoryHelp') }}</p>
              </div>
              <div class="flex flex-wrap gap-2">
                <UButton type="button" color="neutral" variant="outline" size="xs" icon="i-lucide-check-check" :disabled="!canManage" @click="applyCategory(group.category, true)">{{ t('admin.notificationSettings.enableCategory') }}</UButton>
                <UButton type="button" color="neutral" variant="outline" size="xs" icon="i-lucide-bell-off" :disabled="!canManage" @click="applyCategory(group.category, false)">{{ t('admin.notificationSettings.disableCategory') }}</UButton>
              </div>
            </header>
            <ul class="mt-3 divide-y divide-slate-100 dark:divide-zinc-800">
              <li v-for="item in group.items" :key="notificationPolicyKey(item)" class="grid gap-4 py-4 lg:grid-cols-[minmax(12rem,1fr)_auto_auto_auto] lg:items-center">
                <div class="min-w-0">
                  <div class="flex flex-wrap items-center gap-2"><h4 class="font-medium">{{ typeLabel(item) }}</h4><UBadge v-if="item.required" color="neutral" variant="soft">{{ t('admin.notificationSettings.required') }}</UBadge><UBadge v-else-if="item.ownerExtensionId" color="neutral" variant="soft">{{ item.ownerLabel || item.ownerExtensionId }}</UBadge></div>
                  <p class="mt-1 text-sm text-muted">{{ channelLabel(item.channel) }} · {{ itemStatus(item) }}</p>
                  <p v-if="item.ownerExtensionId" class="mt-1 text-xs text-muted">{{ t('admin.notificationSettings.pluginOwned') }}</p>
                </div>
                <UCheckbox :model-value="item.enabled" :disabled="!canManage || !canEditNotificationPolicy(item)" :label="channelEnabledLabel(item)" @update:model-value="setChannelEnabled(item, $event)" />
                <UCheckbox v-model="item.recommendedEnabled" :disabled="!canManage || !canEditNotificationPolicy(item) || !item.enabled" :label="t('admin.notificationSettings.recommendedEnabled')" />
                <UCheckbox v-model="item.userConfigurable" :disabled="!canManage || !canEditNotificationPolicy(item)" :label="t('admin.notificationSettings.userConfigurable')" />
              </li>
            </ul>
          </section>
        </div>
      </div>

      <template #footer>
        <div class="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
          <p class="text-xs text-muted">{{ t('admin.notificationSettings.restoreHelp') }}</p>
          <div class="flex flex-wrap gap-2">
            <UButton type="button" color="neutral" variant="ghost" icon="i-lucide-bell" :disabled="!canManage" @click="createTestNotification">{{ t('admin.notificationSettings.testNotification') }}</UButton>
            <UButton type="button" color="neutral" variant="outline" icon="i-lucide-rotate-ccw" :disabled="!hasChanges" @click="resetChanges">{{ t('admin.form.reset') }}</UButton>
            <UButton type="button" color="neutral" variant="outline" icon="i-lucide-undo-2" :loading="restoring" :disabled="!canManage" @click="restoreDefaults">{{ t('admin.notificationSettings.restoreDefaults') }}</UButton>
            <UButton type="submit" color="primary" icon="i-lucide-save" :loading="saving" :disabled="!canManage || !hasChanges">{{ t('admin.notificationSettings.save') }}</UButton>
          </div>
        </div>
      </template>
    </UCard>
  </form>
</template>
