<script setup lang="ts">
import type { ModerationDecision, ModerationSettings } from '~/composables/useModerationApi'

definePageMeta({ middleware: 'admin', layout: 'admin' })
defineOptions({ name: 'AdminModeration' })

const { t } = useI18n()
const adminPage = useAdminPage('/moderation')
const moderationApi = useModerationApi()

const { data: settings, pending: settingsPending, refresh: refreshSettings } = await useAsyncData(
  'admin-moderation-settings',
  () => moderationApi.getSettings()
)
const { data: history, pending: historyPending, refresh: refreshHistory } = await useAsyncData(
  'admin-moderation-history',
  () => moderationApi.listHistory({ page: 1, perPage: 30 }, true),
  { default: () => ({ items: [] as ModerationDecision[], total: 0, page: 1, perPage: 30 }) }
)

function settingsUpdated(value: ModerationSettings) {
  settings.value = value
  void refreshHistory()
}

async function refreshAll() {
  await Promise.all([refreshSettings(), refreshHistory()])
}
</script>

<template>
  <div class="space-y-6">
    <header class="flex flex-wrap items-start justify-between gap-3">
      <div>
        <h1 class="flex items-center gap-2 text-xl font-bold text-slate-900 dark:text-zinc-50">
          <UIcon :name="adminPage.icon" class="size-5 text-[var(--sf-accent)] dark:text-[var(--sf-accent-dark)]" />
          {{ t('admin.moderation.managementTitle') }}
        </h1>
        <p class="mt-1 text-sm text-slate-500 dark:text-zinc-400">
          {{ t('admin.moderation.managementDescription') }}
        </p>
      </div>
      <UButton icon="i-lucide-rotate-cw" color="neutral" variant="subtle" :loading="settingsPending || historyPending" @click="refreshAll">
        {{ t('admin.home.refresh') }}
      </UButton>
    </header>

    <ModerationSettingsForm
      v-if="settings"
      :model-value="settings"
      @updated="settingsUpdated"
    />
    <div v-else-if="settingsPending" class="space-y-3" aria-busy="true">
      <SFSkeleton width="36%" />
      <SFSkeleton width="100%" height="180px" />
    </div>

    <section aria-labelledby="moderation-audit-title">
      <div class="mb-3">
        <h2 id="moderation-audit-title" class="text-base font-semibold text-slate-900 dark:text-zinc-100">
          {{ t('admin.moderation.auditTitle') }}
        </h2>
        <p class="mt-1 text-sm text-slate-500 dark:text-zinc-400">{{ t('admin.moderation.auditDescription') }}</p>
      </div>
      <ModerationDecisionTable :items="history.items" :loading="historyPending" />
    </section>
  </div>
</template>
