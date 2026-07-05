<script setup lang="ts">
import { apiErrorMessage } from '~/composables/useApiClient'
import { useAdminPage } from '~/composables/useAdminPage'
import { capabilityCount, themeActionState, themeStatusLabelKey } from '~/utils/adminExtensions'

definePageMeta({
  middleware: 'admin',
  layout: 'admin'
})

defineOptions({
  name: 'AdminExtensions'
})

const { t } = useI18n()
const adminPage = useAdminPage('/extensions')

const {
  extensions,
  pending,
  error,
  refresh,
  fileInput,
  selectedId,
  selected,
  selectedEvents,
  uploading,
  busyId,
  loadingEvents,
  stats,
  openUpload,
  uploadArchive,
  enableExtension,
  disableExtension,
  activateTheme,
  verifyExtension,
  loadEvents,
  statusColor,
  typeLabel,
  statusLabel
} = await useAdminExtensionsManager()

useSeoMeta({
  title: t('admin.extensions.metaTitle')
})

watch(selected, async (item) => {
  if (!item) {
    return
  }
  selectedId.value = item.id
  await loadEvents(item.id)
}, { immediate: true })

async function refreshSelectedEvents() {
  if (selected.value) {
    await loadEvents(selected.value.id)
  }
}

function extensionStatusLabel(item: (typeof extensions.value)[number]) {
  return item.type === 'theme' ? t(themeStatusLabelKey(item)) : statusLabel(item.status)
}
</script>

<template>
  <div class="mb-4 flex flex-col gap-1">
    <h2 class="text-xl font-bold flex items-center gap-2 text-slate-900 dark:text-zinc-100">
      <UIcon :name="adminPage.icon" class="size-5 text-[var(--sf-accent)] dark:text-[var(--sf-accent-dark)]" />
      {{ t('admin.extensions.title') }}
    </h2>
    <p class="text-sm text-slate-500 dark:text-zinc-400">
      {{ t('admin.extensions.intro') }}
    </p>
  </div>

  <UDashboardToolbar class="border border-slate-200 dark:border-zinc-800 bg-white dark:bg-zinc-900 rounded-lg px-4 py-2.5 mb-6 text-slate-500 dark:text-zinc-400">
    <template #left>
      <div class="flex min-w-0 items-center gap-2 text-sm text-slate-500 dark:text-zinc-400">
        <UIcon name="i-lucide-package" class="size-4" />
        <span class="truncate">{{ t('admin.extensions.installedCount', { count: extensions.length }) }}</span>
      </div>
      <input ref="fileInput" class="hidden" type="file" accept=".zip,application/zip" @change="uploadArchive">
    </template>
    <template #right>
      <UButton icon="i-lucide-rotate-cw" color="neutral" variant="subtle" :loading="pending" @click="refresh()">
        {{ t('admin.extensions.refresh') }}
      </UButton>
      <UButton icon="i-lucide-upload" color="primary" :loading="uploading" @click="openUpload">
        {{ t('admin.extensions.upload') }}
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

  <div class="grid gap-5 md:grid-cols-3 mb-6">
    <UCard class="elegant-card border-slate-200 dark:border-zinc-800 bg-white dark:bg-zinc-900 text-slate-900 dark:text-zinc-100">
      <div class="flex items-center justify-between gap-4">
        <div class="min-w-0">
          <p class="text-xs font-semibold text-slate-500 dark:text-zinc-400 uppercase tracking-wider">
            {{ t('admin.extensions.stats.plugins') }}
          </p>
          <p class="mt-2.5 truncate text-3xl font-black text-slate-900 dark:text-white tracking-tight">
            {{ stats.pluginCount }}
          </p>
        </div>
        <span class="icon-glass-box shrink-0 text-[var(--sf-accent)] dark:text-[var(--sf-accent-dark)]">
          <UIcon name="i-lucide-blocks" class="size-5 z-10" />
        </span>
      </div>
    </UCard>
    <UCard class="elegant-card border-slate-200 dark:border-zinc-800 bg-white dark:bg-zinc-900 text-slate-900 dark:text-zinc-100">
      <div class="flex items-center justify-between gap-4">
        <div class="min-w-0">
          <p class="text-xs font-semibold text-slate-500 dark:text-zinc-400 uppercase tracking-wider">
            {{ t('admin.extensions.stats.themes') }}
          </p>
          <p class="mt-2.5 truncate text-3xl font-black text-slate-900 dark:text-white tracking-tight">
            {{ stats.themeCount }}
          </p>
        </div>
        <span class="icon-glass-box shrink-0 text-purple-600 dark:text-purple-400">
          <UIcon name="i-lucide-palette" class="size-5 z-10" />
        </span>
      </div>
    </UCard>
    <UCard class="elegant-card border-slate-200 dark:border-zinc-800 bg-white dark:bg-zinc-900 text-slate-900 dark:text-zinc-100">
      <div class="flex items-center justify-between gap-4">
        <div class="min-w-0">
          <p class="text-xs font-semibold text-slate-500 dark:text-zinc-400 uppercase tracking-wider">
            {{ t('admin.extensions.stats.enabledPlugins') }}
          </p>
          <p class="mt-2.5 truncate text-3xl font-black text-slate-900 dark:text-white tracking-tight">
            {{ stats.enabledPluginCount }}
          </p>
        </div>
        <span class="icon-glass-box shrink-0 text-green-600 dark:text-green-400">
          <UIcon name="i-lucide-play" class="size-5 z-10" />
        </span>
      </div>
    </UCard>
  </div>

  <div class="flex flex-col gap-6">
    <div class="grid gap-6 xl:grid-cols-[minmax(0,1fr)_420px]">
      <div class="overflow-hidden rounded-lg border border-slate-200 bg-white dark:border-zinc-800 dark:bg-zinc-900">
        <div class="border-b border-slate-200 px-4 py-3 dark:border-zinc-800">
          <h2 class="text-sm font-semibold text-slate-900 dark:text-zinc-100">
            {{ t('admin.extensions.installed') }}
          </h2>
        </div>
        <div v-if="extensions.length === 0 && !pending" class="p-10">
          <SFEmptyState icon-label="ZIP" :title="t('admin.extensions.emptyTitle')" :description="t('admin.extensions.emptyDescription')" />
        </div>
        <div v-else class="divide-y divide-slate-200 dark:divide-zinc-800">
          <div
            v-for="item in extensions"
            :key="item.id"
            class="grid gap-4 px-4 py-4 transition hover:bg-slate-50 md:grid-cols-[minmax(0,1fr)_auto] dark:hover:bg-zinc-800/50"
            :class="selected?.id === item.id ? 'bg-slate-50 dark:bg-zinc-800/50' : ''"
          >
            <button
              type="button"
              class="min-w-0 rounded-md text-left outline-none focus-visible:ring-2 focus-visible:ring-[var(--sf-accent)] focus-visible:ring-offset-2 focus-visible:ring-offset-white dark:focus-visible:ring-offset-zinc-900"
              @click="selectedId = item.id"
            >
              <div class="flex flex-wrap items-center gap-2">
                <UIcon :name="item.type === 'theme' ? 'i-lucide-palette' : 'i-lucide-blocks'" class="size-4 text-[var(--sf-accent)]" />
                <h3 class="truncate text-sm font-semibold text-slate-900 dark:text-zinc-100">
                  {{ item.name }}
                </h3>
                <UBadge :color="statusColor(item.status)" variant="subtle">
                  {{ extensionStatusLabel(item) }}
                </UBadge>
                <UBadge color="neutral" variant="outline">
                  {{ typeLabel(item.type) }}
                </UBadge>
              </div>
              <p class="mt-1 truncate text-xs text-slate-500 dark:text-zinc-400">
                {{ item.id }} · v{{ item.version }} · {{ t('admin.extensions.capabilityCount', { count: capabilityCount(item) }) }}
              </p>
            </button>
            <div class="flex items-center gap-2 md:justify-end">
              <UButton
                v-if="item.type === 'plugin' && item.status !== 'enabled'"
                size="sm"
                icon="i-lucide-play"
                :loading="busyId === item.id"
                @click="enableExtension(item)"
              >
                {{ t('admin.extensions.enable') }}
              </UButton>
              <UButton
                v-else-if="item.type === 'plugin'"
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
                v-if="item.type === 'plugin'"
                size="sm"
                color="neutral"
                variant="ghost"
                icon="i-lucide-refresh-cw"
                disabled
                :title="t('admin.extensions.restartUnavailable')"
              >
                {{ t('admin.extensions.restart') }}
              </UButton>
              <UButton
                v-else-if="themeActionState(item) === 'activateDefault'"
                size="sm"
                icon="i-lucide-rotate-ccw"
                :loading="busyId === item.id"
                @click="activateTheme(item)"
              >
                {{ t('admin.extensions.restoreDefaultTheme') }}
              </UButton>
              <UButton
                v-else-if="themeActionState(item) === 'verifyOnly'"
                size="sm"
                color="neutral"
                variant="subtle"
                icon="i-lucide-shield-check"
                :loading="busyId === item.id"
                @click="verifyExtension(item)"
              >
                {{ t('admin.extensions.verify') }}
              </UButton>
              <UButton
                v-else
                size="sm"
                color="primary"
                variant="subtle"
                icon="i-lucide-check-circle-2"
                disabled
              >
                {{ t('admin.extensions.activeTheme') }}
              </UButton>
            </div>
          </div>
        </div>
      </div>

      <aside class="space-y-4">
        <div class="rounded-lg border border-slate-200 bg-white p-4 dark:border-zinc-800 dark:bg-zinc-900">
          <h2 class="text-sm font-semibold text-slate-900 dark:text-zinc-100">
            {{ selected?.name || t('admin.extensions.detailTitle') }}
          </h2>
          <dl v-if="selected" class="mt-4 space-y-3 text-sm">
            <div class="flex justify-between gap-3">
              <dt class="text-slate-500 dark:text-zinc-400">{{ t('admin.extensions.manifest.id') }}</dt>
              <dd class="truncate text-slate-900 dark:text-zinc-100">{{ selected.id }}</dd>
            </div>
            <div class="flex justify-between gap-3">
              <dt class="text-slate-500 dark:text-zinc-400">{{ t('admin.extensions.manifest.version') }}</dt>
              <dd class="text-slate-900 dark:text-zinc-100">{{ selected.version }}</dd>
            </div>
            <div class="flex justify-between gap-3">
              <dt class="text-slate-500 dark:text-zinc-400">{{ t('admin.extensions.manifest.sforum') }}</dt>
              <dd class="text-slate-900 dark:text-zinc-100">{{ selected.manifest.sforumVersion }}</dd>
            </div>
          </dl>
        </div>

        <div v-if="selected" class="rounded-lg border border-slate-200 bg-white p-4 dark:border-zinc-800 dark:bg-zinc-900">
          <h2 class="text-sm font-semibold text-slate-900 dark:text-zinc-100">
            {{ t('admin.extensions.capabilities') }}
          </h2>
          <div class="mt-4 flex flex-wrap gap-2">
            <UBadge v-if="selected.manifest.backend?.entry" color="warning" variant="subtle" icon="i-lucide-terminal">
              {{ t('admin.extensions.capability.backend') }}
            </UBadge>
            <UBadge v-if="selected.manifest.frontend?.layer" color="primary" variant="subtle" icon="i-lucide-layers">
              {{ t('admin.extensions.capability.frontend') }}
            </UBadge>
            <UBadge v-if="selected.manifest.routes?.length" color="neutral" variant="subtle" icon="i-lucide-route">
              {{ t('admin.extensions.capability.routes', { count: selected.manifest.routes.length }) }}
            </UBadge>
            <UBadge v-if="selected.manifest.hooks?.length" color="neutral" variant="subtle" icon="i-lucide-webhook">
              {{ t('admin.extensions.capability.hooks', { count: selected.manifest.hooks.length }) }}
            </UBadge>
            <UBadge v-if="selected.manifest.jobs?.length" color="neutral" variant="subtle" icon="i-lucide-list-checks">
              {{ t('admin.extensions.capability.jobs', { count: selected.manifest.jobs.length }) }}
            </UBadge>
          </div>
        </div>

        <div v-if="selected" class="rounded-lg border border-slate-200 bg-white p-4 dark:border-zinc-800 dark:bg-zinc-900">
          <div class="flex items-center justify-between gap-3">
            <h2 class="text-sm font-semibold text-slate-900 dark:text-zinc-100">
              {{ t('admin.extensions.events') }}
            </h2>
            <UButton size="xs" color="neutral" variant="ghost" icon="i-lucide-rotate-cw" :loading="loadingEvents" @click="refreshSelectedEvents" />
          </div>
          <div class="mt-4 space-y-3">
            <p v-if="selectedEvents.length === 0" class="text-sm text-slate-500 dark:text-zinc-400">
              {{ t('admin.extensions.noEvents') }}
            </p>
            <div v-for="event in selectedEvents" :key="event.id" class="rounded-md bg-slate-50 p-3 text-sm dark:bg-zinc-900">
              <div class="flex items-center justify-between gap-3">
                <span class="font-medium text-slate-900 dark:text-zinc-100">{{ event.action }}</span>
                <span class="text-xs text-slate-500 dark:text-zinc-400">{{ new Date(event.createdAt).toLocaleString() }}</span>
              </div>
              <p v-if="event.message" class="mt-1 text-slate-500 dark:text-zinc-400">
                {{ event.message }}
              </p>
            </div>
          </div>
        </div>
      </aside>
    </div>
  </div>
</template>
