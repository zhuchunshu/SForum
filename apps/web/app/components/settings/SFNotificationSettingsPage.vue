<script setup lang="ts">
import { apiErrorMessage } from '~/composables/useApiClient'
import {
  canOverrideNotificationPreference,
  notificationPreferenceKey,
  preferenceUpdateItems,
  useNotificationPreferences,
  type NotificationPreferenceCatalog,
  type NotificationPreferenceItem,
  type NotificationPreferenceState
} from '~/composables/notifications/useNotificationPreferences'
import { useSForumSeo } from '~/composables/seo/useSForumSeo'
import SFSettingsShell from '~/components/settings/SFSettingsShell.vue'
import SFWebPushSettingsSection from '~/components/settings/SFWebPushSettingsSection.vue'

const { t, te } = useI18n()
const toast = useToast()
const preferencesApi = useNotificationPreferences()

useSForumSeo({
  title: () => t('notificationSettings.metaTitle'),
  description: () => t('notificationSettings.metaDescription'),
  type: 'website',
  noindex: true
})

const emptyCatalog = (): NotificationPreferenceCatalog => ({ revision: 0, items: [] })
const { data: catalog, pending, error, refresh } = await useAsyncData(
  'notification-preferences',
  () => preferencesApi.list(),
  { default: emptyCatalog }
)
const draft = reactive<Record<string, NotificationPreferenceState>>({})
const saving = ref(false)
const restoring = ref(false)
const actionError = ref('')

const stateItems = computed(() => [
  { label: t('notificationSettings.states.inherit'), value: 'inherit' },
  { label: t('notificationSettings.states.enabled'), value: 'enabled' },
  { label: t('notificationSettings.states.disabled'), value: 'disabled' }
])
const categories = computed(() => {
  const grouped = new Map<string, NotificationPreferenceItem[]>()
  for (const item of catalog.value.items) {
    const items = grouped.get(item.category) || []
    items.push(item)
    grouped.set(item.category, items)
  }
  return [...grouped.entries()].map(([category, items]) => ({ category, items }))
})
const hasChanges = computed(() => catalog.value.items.some(item => draft[notificationPreferenceKey(item)] !== item.state))

watch(catalog, value => {
  for (const item of value.items) {
    draft[notificationPreferenceKey(item)] = item.state
  }
}, { immediate: true })

function label(prefix: string, value: string) {
  const key = `${prefix}.${value}`
  return te(key) ? t(key) : value
}

function channelLabel(channel: string) {
  return label('notificationSettings.channels', channel)
}

function categoryLabel(category: string) {
  return label('notificationSettings.categories', category)
}

function typeLabel(item: NotificationPreferenceItem) {
  const key = `notificationSettings.types.${item.type}`
  return te(key) ? t(key) : item.type
}

function itemStatus(item: NotificationPreferenceItem) {
  if (item.required) return t('notificationSettings.status.required')
  if (!item.active || !item.enabled) return t('notificationSettings.status.disabledBySite')
  if (item.channelAvailable === false) return t('notificationSettings.status.channelUnavailable')
  if (!item.userConfigurable) return t('notificationSettings.status.managedBySite')
  return item.effective ? t('notificationSettings.status.effectiveEnabled') : t('notificationSettings.status.effectiveDisabled')
}

function applyCategory(category: string, state: NotificationPreferenceState) {
  for (const item of catalog.value.items) {
    if (item.category === category && canOverrideNotificationPreference(item)) {
      draft[notificationPreferenceKey(item)] = state
    }
  }
}

function resetDraft() {
  for (const item of catalog.value.items) {
    draft[notificationPreferenceKey(item)] = item.state
  }
  actionError.value = ''
}

async function save() {
  if (!hasChanges.value || saving.value) return
  saving.value = true
  actionError.value = ''
  try {
    catalog.value = await preferencesApi.replace(catalog.value.revision, preferenceUpdateItems(catalog.value.items, draft))
    toast.add({ color: 'success', icon: 'i-lucide-check', title: t('notificationSettings.saved'), duration: 10000 })
  } catch (saveError) {
    actionError.value = apiErrorMessage(saveError) || t('notificationSettings.saveFailed')
  } finally {
    saving.value = false
  }
}

async function restoreDefaults() {
  if (restoring.value) return
  restoring.value = true
  actionError.value = ''
  try {
    catalog.value = await preferencesApi.restore(catalog.value.revision)
    toast.add({ color: 'success', icon: 'i-lucide-rotate-ccw', title: t('notificationSettings.restored'), duration: 10000 })
  } catch (restoreError) {
    actionError.value = apiErrorMessage(restoreError) || t('notificationSettings.restoreFailed')
  } finally {
    restoring.value = false
  }
}
</script>

<template>
  <SFSettingsShell
    class="sforum-settings-notifications"
    data-sforum-island-body="notifications.component.settings"
    active="notifications"
    title-id="notification-settings-title"
    :title="t('notificationSettings.title')"
    :description="t('notificationSettings.intro')"
    :rail-label="t('notificationSettings.rail.ariaLabel')"
    :rail-open-label="t('notificationSettings.rail.open')"
  >
    <template #head-actions>
      <SFButton variant="secondary" size="sm" :disabled="!hasChanges || saving" @click="resetDraft">
        <UIcon name="i-lucide-rotate-ccw" class="mr-1" />
        {{ t('notificationSettings.resetChanges') }}
      </SFButton>
    </template>

    <UAlert
      v-if="actionError || error"
      class="mt-2"
      color="error"
      variant="soft"
      icon="i-lucide-triangle-alert"
      :title="actionError || t('notificationSettings.loadFailed')"
    />

    <section class="mt-5 rounded-md border border-teal-200 bg-teal-50/70 p-4 dark:border-teal-900/60 dark:bg-teal-950/25">
      <div class="flex gap-3">
        <UIcon name="i-lucide-sparkles" class="size-5 shrink-0 text-teal-700 dark:text-teal-300" />
        <div>
          <h2 class="font-semibold">{{ t('notificationSettings.recommendedTitle') }}</h2>
          <p class="mt-1 text-sm text-muted">{{ t('notificationSettings.recommendedDescription') }}</p>
        </div>
      </div>
    </section>

    <div v-if="pending && catalog.items.length === 0" class="mt-5 space-y-3" aria-busy="true">
      <SFSkeleton v-for="index in 3" :key="index" class="h-32 w-full" />
    </div>

    <SFEmptyState
      v-else-if="!pending && !error && categories.length === 0"
      class="mt-5"
      icon-label="NOTIFY"
      :title="t('notificationSettings.emptyTitle')"
      :description="t('notificationSettings.emptyDescription')"
    />

    <div v-else class="mt-5 space-y-4">
      <section v-for="group in categories" :key="group.category" class="overflow-hidden rounded-md border border-slate-200 bg-white dark:border-zinc-800 dark:bg-zinc-900">
        <header class="flex flex-col gap-3 border-b border-slate-200 px-4 py-3 dark:border-zinc-800 sm:flex-row sm:items-center sm:justify-between">
          <div>
            <h2 class="text-base font-semibold">{{ categoryLabel(group.category) }}</h2>
            <p class="mt-1 text-xs text-muted">{{ t('notificationSettings.categoryHelp') }}</p>
          </div>
          <div class="flex flex-wrap gap-2">
            <UButton color="neutral" variant="outline" size="xs" icon="i-lucide-check-check" @click="applyCategory(group.category, 'enabled')">{{ t('notificationSettings.enableCategory') }}</UButton>
            <UButton color="neutral" variant="outline" size="xs" icon="i-lucide-bell-off" @click="applyCategory(group.category, 'disabled')">{{ t('notificationSettings.disableCategory') }}</UButton>
            <UButton color="neutral" variant="ghost" size="xs" icon="i-lucide-undo-2" @click="applyCategory(group.category, 'inherit')">{{ t('notificationSettings.inheritCategory') }}</UButton>
          </div>
        </header>
        <ul class="divide-y divide-slate-200 dark:divide-zinc-800">
          <li v-for="item in group.items" :key="notificationPreferenceKey(item)" class="grid gap-3 px-4 py-4 sm:grid-cols-[minmax(0,1fr)_10rem] sm:items-center">
            <div class="min-w-0">
              <div class="flex flex-wrap items-center gap-2">
                <h3 class="font-medium">{{ typeLabel(item) }}</h3>
                <UBadge v-if="item.required" color="neutral" variant="soft">{{ t('notificationSettings.required') }}</UBadge>
                <UBadge v-else-if="item.ownerExtensionId" color="neutral" variant="soft">{{ item.ownerLabel || item.ownerExtensionId }}</UBadge>
              </div>
              <p class="mt-1 text-sm text-muted">{{ channelLabel(item.channel) }} · {{ itemStatus(item) }}</p>
              <p v-if="item.ownerExtensionId" class="mt-1 text-xs text-muted">{{ t('notificationSettings.pluginOwned') }}</p>
            </div>
            <USelect
              v-if="canOverrideNotificationPreference(item)"
              v-model="draft[notificationPreferenceKey(item)]"
              :items="stateItems"
              value-key="value"
              label-key="label"
              :disabled="!canOverrideNotificationPreference(item)"
              class="w-full"
              :aria-label="`${typeLabel(item)} ${channelLabel(item.channel)}`"
            />
            <UBadge v-else color="neutral" variant="soft" class="justify-center py-1.5">
              {{ itemStatus(item) }}
            </UBadge>
          </li>
        </ul>
      </section>
    </div>

    <SFWebPushSettingsSection />

    <div class="mt-5 flex flex-col gap-3 border-t border-slate-200 pt-5 dark:border-zinc-800 sm:flex-row sm:items-center sm:justify-between">
      <p class="text-xs text-muted">{{ t('notificationSettings.restoreHelp') }}</p>
      <div class="flex flex-wrap gap-2">
        <UButton color="neutral" variant="outline" icon="i-lucide-rotate-ccw" :loading="restoring" @click="restoreDefaults">{{ t('notificationSettings.restoreDefaults') }}</UButton>
        <SFButton :loading="saving" :disabled="!hasChanges || pending" @click="save">
          <UIcon name="i-lucide-save" class="mr-1" />
          {{ t('notificationSettings.save') }}
        </SFButton>
      </div>
    </div>

    <template #rail>
      <section class="space-y-3">
        <h2 class="text-sm font-semibold">{{ t('notificationSettings.rail.title') }}</h2>
        <p class="text-sm text-muted">{{ t('notificationSettings.rail.description') }}</p>
        <UButton color="neutral" variant="outline" icon="i-lucide-refresh-cw" :loading="pending" block @click="refresh()">{{ t('notificationSettings.refresh') }}</UButton>
      </section>
    </template>
  </SFSettingsShell>
</template>
