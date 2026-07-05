<script setup lang="ts">
import { apiErrorMessage } from '~/composables/useApiClient'
import { useAdminPage } from '~/composables/useAdminPage'

definePageMeta({
  middleware: 'admin',
  layout: 'admin'
})

defineOptions({
  name: 'AdminExtensionEvents'
})

const { t } = useI18n()
const adminPage = useAdminPage('/extensions/events')
const {
  extensions,
  pending,
  error,
  refresh,
  aggregatedEvents,
  loadingAllEvents,
  loadAllEvents
} = await useAdminExtensionsManager()

useSeoMeta({
  title: t('admin.extensions.eventLog.metaTitle')
})

watch(() => extensions.value.map(item => item.id).join('|'), () => {
  void loadAllEvents()
}, { immediate: true })

async function refreshEvents() {
  await refresh()
  await loadAllEvents()
}

function extensionName(id: string) {
  return extensions.value.find(item => item.id === id)?.name || id
}
</script>

<template>
  <div class="mb-4 flex flex-col gap-1">
    <h2 class="text-xl font-bold flex items-center gap-2 text-slate-900 dark:text-zinc-100">
      <UIcon :name="adminPage.icon" class="size-5 text-[var(--sf-accent)] dark:text-[var(--sf-accent-dark)]" />
      {{ t('admin.extensions.eventLog.title') }}
    </h2>
    <p class="text-sm text-slate-500 dark:text-zinc-400">
      {{ t('admin.extensions.eventLog.intro') }}
    </p>
  </div>

  <UDashboardToolbar class="border border-slate-200 dark:border-zinc-800 bg-white dark:bg-zinc-900 rounded-lg px-4 py-2.5 mb-6 text-slate-500 dark:text-zinc-400">
    <template #left>
      <div class="flex min-w-0 items-center gap-2 text-sm">
        <UIcon name="i-lucide-scroll-text" class="size-4" />
        <span class="truncate">{{ t('admin.extensions.eventLog.count', { count: aggregatedEvents.length }) }}</span>
      </div>
    </template>
    <template #right>
      <UButton icon="i-lucide-rotate-cw" color="neutral" variant="subtle" :loading="pending || loadingAllEvents" @click="refreshEvents">
        {{ t('admin.extensions.refresh') }}
      </UButton>
    </template>
  </UDashboardToolbar>

  <UAlert
    v-if="error"
    color="error"
    icon="i-lucide-triangle-alert"
    variant="subtle"
    :title="apiErrorMessage(error) || t('admin.extensions.loadFailed')"
    class="mb-6"
  />

  <div class="rounded-lg border border-slate-200 bg-white dark:border-zinc-800 dark:bg-zinc-900">
    <div v-if="aggregatedEvents.length === 0 && !pending && !loadingAllEvents" class="p-10">
      <SFEmptyState icon-label="LOG" :title="t('admin.extensions.eventLog.emptyTitle')" :description="t('admin.extensions.eventLog.emptyDescription')" />
    </div>
    <div v-else class="divide-y divide-slate-200 dark:divide-zinc-800">
      <div
        v-for="event in aggregatedEvents"
        :key="`${event.extensionId}:${event.id}`"
        class="grid gap-3 px-4 py-4 md:grid-cols-[minmax(0,1fr)_220px]"
      >
        <div class="min-w-0">
          <div class="flex flex-wrap items-center gap-2">
            <UIcon name="i-lucide-activity" class="size-4 text-[var(--sf-accent)]" />
            <h3 class="truncate text-sm font-semibold text-slate-900 dark:text-zinc-100">
              {{ event.action }}
            </h3>
            <UBadge color="neutral" variant="outline">
              {{ extensionName(event.extensionId) }}
            </UBadge>
          </div>
          <p v-if="event.message" class="mt-1 text-sm text-slate-500 dark:text-zinc-400">
            {{ event.message }}
          </p>
          <p class="mt-1 truncate text-xs text-slate-500 dark:text-zinc-400">
            {{ event.extensionId }}
          </p>
        </div>
        <div class="flex items-center text-xs text-slate-500 md:justify-end dark:text-zinc-400">
          {{ new Date(event.createdAt).toLocaleString() }}
        </div>
      </div>
    </div>
  </div>
</template>
