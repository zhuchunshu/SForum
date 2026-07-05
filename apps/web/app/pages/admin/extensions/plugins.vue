<script setup lang="ts">
import { apiErrorMessage } from '~/composables/useApiClient'
import { useAdminPage } from '~/composables/useAdminPage'
import { capabilityCount, filterExtensionsByType } from '~/utils/adminExtensions'

definePageMeta({
  middleware: 'admin',
  layout: 'admin'
})

defineOptions({
  name: 'AdminExtensionPlugins'
})

const { t } = useI18n()
const adminPage = useAdminPage('/extensions/plugins')
const {
  extensions,
  pending,
  error,
  refresh,
  busyId,
  enableExtension,
  disableExtension,
  statusColor,
  statusLabel
} = await useAdminExtensionsManager()

const plugins = computed(() => filterExtensionsByType(extensions.value, 'plugin'))

useSeoMeta({
  title: t('admin.extensions.plugins.metaTitle')
})
</script>

<template>
  <div class="mb-4 flex flex-col gap-1">
    <h2 class="text-xl font-bold flex items-center gap-2 text-slate-900 dark:text-zinc-100">
      <UIcon :name="adminPage.icon" class="size-5 text-[var(--sf-accent)] dark:text-[var(--sf-accent-dark)]" />
      {{ t('admin.extensions.plugins.title') }}
    </h2>
    <p class="text-sm text-slate-500 dark:text-zinc-400">
      {{ t('admin.extensions.plugins.intro') }}
    </p>
  </div>

  <UDashboardToolbar class="border border-slate-200 dark:border-zinc-800 bg-white dark:bg-zinc-900 rounded-lg px-4 py-2.5 mb-6 text-slate-500 dark:text-zinc-400">
    <template #left>
      <div class="flex min-w-0 items-center gap-2 text-sm">
        <UIcon name="i-lucide-plug" class="size-4" />
        <span class="truncate">{{ t('admin.extensions.plugins.count', { count: plugins.length }) }}</span>
      </div>
    </template>
    <template #right>
      <UButton icon="i-lucide-rotate-cw" color="neutral" variant="subtle" :loading="pending" @click="refresh()">
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

  <div class="overflow-hidden rounded-lg border border-slate-200 bg-white dark:border-zinc-800 dark:bg-zinc-900">
    <div v-if="plugins.length === 0 && !pending" class="p-10">
      <SFEmptyState icon-label="PLG" :title="t('admin.extensions.plugins.emptyTitle')" :description="t('admin.extensions.plugins.emptyDescription')" />
    </div>
    <div v-else class="divide-y divide-slate-200 dark:divide-zinc-800">
      <div
        v-for="item in plugins"
        :key="item.id"
        class="grid gap-4 px-4 py-4 md:grid-cols-[1fr_auto]"
      >
        <div class="min-w-0">
          <div class="flex flex-wrap items-center gap-2">
            <UIcon name="i-lucide-plug" class="size-4 text-[var(--sf-accent)]" />
            <h3 class="truncate text-sm font-semibold text-slate-900 dark:text-zinc-100">
              {{ item.name }}
            </h3>
            <UBadge :color="statusColor(item.status)" variant="subtle">
              {{ statusLabel(item.status) }}
            </UBadge>
          </div>
          <p class="mt-1 truncate text-xs text-slate-500 dark:text-zinc-400">
            {{ item.id }} · v{{ item.version }} · {{ t('admin.extensions.capabilityCount', { count: capabilityCount(item) }) }}
          </p>
        </div>
        <div class="flex items-center gap-2">
          <UButton
            v-if="item.status !== 'enabled'"
            size="sm"
            icon="i-lucide-play"
            :loading="busyId === item.id"
            @click="enableExtension(item)"
          >
            {{ t('admin.extensions.enable') }}
          </UButton>
          <UButton
            v-else
            size="sm"
            color="neutral"
            variant="subtle"
            icon="i-lucide-pause"
            :loading="busyId === item.id"
            @click="disableExtension(item)"
          >
            {{ t('admin.extensions.disable') }}
          </UButton>
          <UButton
            size="sm"
            color="neutral"
            variant="ghost"
            icon="i-lucide-refresh-cw"
            disabled
            :title="t('admin.extensions.restartUnavailable')"
          >
            {{ t('admin.extensions.restart') }}
          </UButton>
        </div>
      </div>
    </div>
  </div>
</template>
