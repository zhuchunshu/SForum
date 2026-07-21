<script setup lang="ts">
import type { ModerationDecision, ModerationSettings } from '~/composables/useModerationApi'

definePageMeta({ middleware: 'admin', layout: 'admin' })
defineOptions({ name: 'AdminModeration' })

const { t } = useI18n()
const adminPage = useAdminPage('/moderation')
const moderationApi = useModerationApi()

const settingsModalOpen = ref(false)

const { data: settings, pending: settingsPending, refresh: refreshSettings, error: settingsError } = await useAsyncData(
  'admin-moderation-settings',
  () => moderationApi.getSettings()
)
const { data: history, pending: historyPending, refresh: refreshHistory, error: historyError } = await useAsyncData(
  'admin-moderation-history',
  () => moderationApi.listHistory({ page: 1, perPage: 30 }, true),
  { default: () => ({ items: [] as ModerationDecision[], total: 0, page: 1, perPage: 30 }) }
)

const loadError = computed(() => {
  const err = settingsError.value || historyError.value
  if (!err) return ''
  return typeof err === 'object' && err !== null && 'message' in err
    ? String((err as { message?: unknown }).message || '')
    : String(err)
})

const currentModeLabel = computed(() => {
  const mode = settings.value?.mode
  if (!mode) return ''
  return t(`admin.moderation.mode.${mode}`)
})

function settingsUpdated(value: ModerationSettings) {
  settings.value = value
  void refreshHistory()
}

function openSettingsModal() {
  settingsModalOpen.value = true
}

function closeSettingsModal() {
  settingsModalOpen.value = false
}

async function refreshAll() {
  await Promise.all([refreshSettings(), refreshHistory()])
}

useSeoMeta({
  title: t('admin.moderation.managementTitle')
})
</script>

<template>
  <div class="mb-4 flex flex-col gap-1">
    <h2 class="flex items-center gap-2 text-xl font-bold text-slate-900 dark:text-zinc-100">
      <UIcon :name="adminPage.icon" class="size-5 text-[var(--sf-accent)] dark:text-[var(--sf-accent-dark)]" />
      {{ t('admin.moderation.managementTitle') }}
    </h2>
    <p class="text-sm text-slate-500 dark:text-zinc-400">
      {{ t('admin.moderation.managementDescription') }}
    </p>
  </div>

  <UDashboardToolbar class="mb-6 rounded-lg border border-slate-200 bg-white px-4 py-2.5 text-slate-500 dark:border-zinc-800 dark:bg-zinc-900 dark:text-zinc-400">
    <template #left>
      <div class="flex min-w-0 flex-wrap items-center gap-2 text-sm">
        <UIcon name="i-lucide-shield-check" class="size-4" />
        <span class="truncate">{{ t('admin.moderation.toolbar') }}</span>
        <UBadge v-if="settings" color="primary" variant="soft">
          {{ t('admin.moderation.currentMode', { mode: currentModeLabel }) }}
        </UBadge>
      </div>
    </template>
    <template #right>
      <div class="flex flex-wrap items-center gap-2">
        <UButton
          color="neutral"
          variant="outline"
          leading-icon="i-lucide-refresh-cw"
          :loading="settingsPending || historyPending"
          class="border-slate-200 dark:border-zinc-700"
          @click="refreshAll()"
        >
          {{ t('admin.common.refresh') }}
        </UButton>
        <UButton
          leading-icon="i-lucide-settings-2"
          :loading="settingsPending && !settings"
          @click="openSettingsModal"
        >
          {{ t('admin.moderation.openSettings') }}
        </UButton>
      </div>
    </template>
  </UDashboardToolbar>

  <div class="flex w-full min-w-0 flex-col gap-4">
    <UAlert
      v-if="loadError"
      color="error"
      variant="soft"
      icon="i-lucide-triangle-alert"
      class="w-full shrink-0"
      :title="loadError || t('admin.moderation.loadFailed')"
    />

    <UAlert
      color="primary"
      variant="soft"
      icon="i-lucide-sparkles"
      class="w-full shrink-0"
      :title="t('admin.moderation.recommendedTitle')"
      :description="t('admin.moderation.recommendedDescription')"
    />

    <UCard class="border-slate-200 bg-white text-slate-900 dark:border-zinc-800 dark:bg-zinc-900 dark:text-zinc-100">
      <template #header>
        <div>
          <h2 id="moderation-audit-title" class="text-base font-bold text-slate-900 dark:text-white">
            {{ t('admin.moderation.auditTitle') }}
          </h2>
          <p class="mt-1 text-xs text-slate-500 dark:text-zinc-400">
            {{ t('admin.moderation.auditDescription') }}
          </p>
        </div>
      </template>
      <ModerationDecisionTable :items="history.items" :loading="historyPending" />
    </UCard>
  </div>

  <!-- 发布审核策略设置 -->
  <UModal
    v-model:open="settingsModalOpen"
    :ui="{ content: 'sm:max-w-3xl' }"
  >
    <template #content>
      <div class="max-h-[90vh] overflow-y-auto p-1 sm:p-2">
        <div class="flex items-center justify-end px-3 pt-2">
          <UButton
            type="button"
            color="neutral"
            variant="ghost"
            icon="i-lucide-x"
            :aria-label="t('admin.common.cancel')"
            @click="closeSettingsModal"
          />
        </div>
        <div v-if="settings" class="px-2 pb-2">
          <ModerationSettingsForm
            :model-value="settings"
            @updated="settingsUpdated"
          />
        </div>
        <div v-else class="space-y-3 px-5 py-8" aria-busy="true">
          <SFSkeleton width="36%" />
          <SFSkeleton width="100%" height="180px" />
        </div>
      </div>
    </template>
  </UModal>
</template>
