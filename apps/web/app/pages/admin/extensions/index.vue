<script setup lang="ts">
import { apiErrorMessage } from '~/composables/useApiClient'
import { useAdminPage } from '~/composables/useAdminPage'
import {
  capabilityCount,
  extensionEventPage,
  extensionLocalizedDisplay,
  extensionManageRoute,
  hasExtensionReleaseInProgress,
  pluginWebReleaseProgress,
  themeActionState,
  themeActivationProgress,
  themeStatusLabelKey
} from '~/utils/adminExtensions'

definePageMeta({
  middleware: 'admin',
  layout: 'admin'
})

defineOptions({
  name: 'AdminExtensions'
})

const { format: formatSiteDateTime } = useSiteDateTime()
const { t, locale } = useI18n()
const adminPage = useAdminPage('/extensions')
const adminRoutes = useAdminRoutes()
const selectedEventPage = ref(1)
// 与当前 UI 语言绑定，切换语言时列表与详情文案会立刻重算。
const extensionRows = computed(() => extensions.value.map((item) => ({
  item,
  display: extensionLocalizedDisplay(item, locale.value)
})))
const selectedDisplay = computed(() => selected.value ? extensionLocalizedDisplay(selected.value, locale.value) : null)

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
  confirmEnableExtension,
  cancelEnableExtension,
  enableConfirmOpen,
  enableConfirmItem,
  openUninstallExtension,
  confirmUninstallExtension,
  cancelUninstallExtension,
  uninstallConfirmOpen,
  uninstallConfirmItem,
  disableExtension,
  activateTheme,
  loadEvents,
  statusColor,
  typeLabel,
  statusLabel
} = await useAdminExtensionsManager()
const selectedEventPageInfo = computed(() => extensionEventPage(selectedEvents.value, selectedEventPage.value))
// 主题 themeRelease 与插件 webRelease 任一进行中即轮询列表。
const activationPolling = computed(() => hasExtensionReleaseInProgress(extensions.value))
const expandedLogReleaseIds = ref<Record<number, boolean>>({})
let activationPollTimer: ReturnType<typeof setInterval> | null = null

useSeoMeta({
  title: t('admin.extensions.metaTitle')
})

watch(() => selected.value?.id, async (id) => {
  if (!id) {
    return
  }
  selectedId.value = id
  selectedEventPage.value = 1
  await loadEvents(id)
}, { immediate: true })

watch(() => selectedEventPageInfo.value.page, (page) => {
  selectedEventPage.value = page
})

async function refreshSelectedEvents() {
  if (selected.value) {
    await loadEvents(selected.value.id)
  }
}

function extensionStatusLabel(item: (typeof extensions.value)[number]) {
  return item.type === 'theme' ? t(themeStatusLabelKey(item)) : statusLabel(item.status)
}

function releaseProgress(item: (typeof extensions.value)[number]) {
  if (item.type === 'theme') {
    return themeActivationProgress(item.themeRelease)
  }
  return pluginWebReleaseProgress(item.webRelease)
}

function pluginActionBusy(item: (typeof extensions.value)[number]) {
  return item.type === 'plugin' && Boolean(pluginWebReleaseProgress(item.webRelease)?.active)
}

function extensionBuildLog(item: (typeof extensions.value)[number]) {
  if (item.type === 'theme') {
    return item.themeRelease?.buildLog?.trim() || ''
  }
  return item.webRelease?.buildLog?.trim() || ''
}

function extensionReleaseId(item: (typeof extensions.value)[number]) {
  return item.type === 'theme' ? item.themeRelease?.id : item.webRelease?.id
}

function hasBuildLogToggle(item: (typeof extensions.value)[number]) {
  const releaseId = extensionReleaseId(item)
  if (!releaseId) {
    return false
  }
  const status = item.type === 'theme' ? item.themeRelease?.status : item.webRelease?.status
  return Boolean(extensionBuildLog(item) || status === 'failed' || releaseProgress(item)?.active)
}

function isBuildLogOpen(item: (typeof extensions.value)[number]) {
  const releaseId = extensionReleaseId(item)
  if (!releaseId) {
    return false
  }
  if (Object.prototype.hasOwnProperty.call(expandedLogReleaseIds.value, releaseId)) {
    return expandedLogReleaseIds.value[releaseId]
  }
  const status = item.type === 'theme' ? item.themeRelease?.status : item.webRelease?.status
  return status === 'failed'
}

function toggleBuildLog(item: (typeof extensions.value)[number]) {
  const releaseId = extensionReleaseId(item)
  if (!releaseId) {
    return
  }
  expandedLogReleaseIds.value = {
    ...expandedLogReleaseIds.value,
    [releaseId]: !isBuildLogOpen(item)
  }
}

function startActivationPolling() {
  if (activationPollTimer || !import.meta.client) {
    return
  }
  activationPollTimer = setInterval(async () => {
    if (pending.value) {
      return
    }
    await refresh()
  }, 2000)
}

function stopActivationPolling() {
  if (!activationPollTimer) {
    return
  }
  clearInterval(activationPollTimer)
  activationPollTimer = null
}

watch(activationPolling, (active) => {
  if (active) {
    startActivationPolling()
  } else {
    stopActivationPolling()
  }
})

onMounted(() => {
  if (activationPolling.value) {
    startActivationPolling()
  }
})

onBeforeUnmount(stopActivationPolling)
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
            v-for="{ item, display } in extensionRows"
            :key="item.id"
            class="grid gap-4 px-4 py-4 transition hover:bg-slate-50 md:grid-cols-[minmax(0,1fr)_auto] dark:hover:bg-zinc-800/50"
            :class="selected?.id === item.id ? 'bg-slate-50 dark:bg-zinc-800/50' : ''"
          >
            <div class="min-w-0">
              <button
                type="button"
                class="block w-full min-w-0 rounded-md text-left outline-none focus-visible:ring-2 focus-visible:ring-[var(--sf-accent)] focus-visible:ring-offset-2 focus-visible:ring-offset-white dark:focus-visible:ring-offset-zinc-900"
                @click="selectedId = item.id"
              >
                <div class="flex flex-wrap items-center gap-2">
                  <UIcon :name="item.type === 'theme' ? 'i-lucide-palette' : 'i-lucide-blocks'" class="size-4 text-[var(--sf-accent)]" />
                  <h3 class="truncate text-sm font-semibold text-slate-900 dark:text-zinc-100">
                    {{ display.name }}
                  </h3>
                  <UBadge :color="statusColor(item.status)" variant="subtle">
                    {{ extensionStatusLabel(item) }}
                  </UBadge>
                  <UBadge color="neutral" variant="outline">
                    {{ typeLabel(item.type) }}
                  </UBadge>
                </div>
                <p
                  v-if="display.description"
                  class="mt-1.5 line-clamp-2 text-sm leading-5 text-slate-600 dark:text-zinc-300"
                >
                  {{ display.description }}
                </p>
                <p class="mt-1 truncate text-xs text-slate-500 dark:text-zinc-400">
                  {{ item.id }} · v{{ item.version }} · {{ t('admin.extensions.capabilityCount', { count: capabilityCount(item) }) }}
                </p>
              </button>
              <a
                v-if="display.author.url || display.url"
                :href="display.author.url || display.url"
                target="_blank"
                rel="noopener noreferrer"
                class="mt-2 inline-flex max-w-full items-center gap-1.5 rounded text-xs font-medium text-slate-500 transition hover:text-[var(--sf-accent)] focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-[var(--sf-accent)] focus-visible:ring-offset-2 focus-visible:ring-offset-white dark:text-zinc-400 dark:hover:text-[var(--sf-accent-dark)] dark:focus-visible:ring-offset-zinc-900"
                :title="t('admin.extensions.authorWebsiteTitle', { name: display.author.name })"
                :aria-label="t('admin.extensions.authorWebsiteTitle', { name: display.author.name })"
              >
                <UIcon name="i-lucide-user-round" class="size-3.5 shrink-0" />
                <span class="truncate">{{ t('admin.extensions.authorLinkLabel', { name: display.author.name }) }}</span>
                <UIcon name="i-lucide-external-link" class="size-3 shrink-0" />
              </a>
              <span v-else-if="display.author.name" class="mt-2 inline-flex max-w-full items-center gap-1.5 text-xs text-slate-500 dark:text-zinc-400">
                <UIcon name="i-lucide-user-round" class="size-3.5 shrink-0" />
                <span class="truncate">{{ t('admin.extensions.authorLinkLabel', { name: display.author.name }) }}</span>
              </span>
              <p v-if="item.themeRelease?.message || item.webRelease?.publicMessage" class="mt-2 text-xs leading-5 text-slate-500 dark:text-zinc-400">
                {{ item.themeRelease?.message || item.webRelease?.publicMessage }}
              </p>
              <div
                v-if="releaseProgress(item)"
                class="mt-3 max-w-xl rounded-md border border-slate-200 bg-slate-50 p-3 dark:border-zinc-800 dark:bg-zinc-950/50"
              >
                <div class="mb-2 flex items-center justify-between gap-3 text-xs">
                  <span class="inline-flex min-w-0 items-center gap-1.5 font-medium text-slate-700 dark:text-zinc-200">
                    <UIcon :name="releaseProgress(item)?.icon || 'i-lucide-hourglass'" class="size-3.5 shrink-0" />
                    <span class="truncate">{{ t(releaseProgress(item)?.labelKey || (item.type === 'theme' ? 'admin.extensions.themeRelease.queued' : 'admin.extensions.releases.statusLabels.queued')) }}</span>
                  </span>
                  <span class="tabular-nums text-slate-500 dark:text-zinc-400">
                    {{ releaseProgress(item)?.percent || 0 }}%
                  </span>
                </div>
                <UProgress
                  :model-value="releaseProgress(item)?.percent || 0"
                  :color="releaseProgress(item)?.color || 'neutral'"
                  size="sm"
                />
                <p class="mt-2 text-xs leading-5 text-slate-500 dark:text-zinc-400">
                  {{ t(releaseProgress(item)?.detailKey || (item.type === 'theme' ? 'admin.extensions.themeProgress.queued' : 'admin.extensions.webReleaseProgress.queued')) }}
                </p>
                <div v-if="hasBuildLogToggle(item)" class="mt-3">
                  <UButton
                    size="xs"
                    color="neutral"
                    variant="ghost"
                    :icon="isBuildLogOpen(item) ? 'i-lucide-chevron-up' : 'i-lucide-file-text'"
                    @click="toggleBuildLog(item)"
                  >
                    {{ isBuildLogOpen(item) ? t('admin.extensions.hideBuildLog') : t('admin.extensions.viewBuildLog') }}
                  </UButton>
                  <pre
                    v-if="isBuildLogOpen(item)"
                    class="mt-2 max-h-48 overflow-auto rounded-md border border-slate-200 bg-white p-3 text-xs leading-5 text-slate-700 dark:border-zinc-800 dark:bg-zinc-900 dark:text-zinc-200"
                  >{{ extensionBuildLog(item) || t('admin.extensions.emptyBuildLog') }}</pre>
                </div>
              </div>
            </div>
            <div class="flex items-center gap-2 md:justify-end">
              <UButton
                size="sm"
                color="neutral"
                variant="ghost"
                icon="i-lucide-settings"
                :to="adminRoutes.path(extensionManageRoute(item))"
              >
                {{ t('admin.extensions.manage') }}
              </UButton>
              <UButton
                v-if="pluginActionBusy(item)"
                size="sm"
                color="neutral"
                variant="subtle"
                icon="i-lucide-hourglass"
                disabled
              >
                {{ t(releaseProgress(item)?.labelKey || 'admin.extensions.releases.statusLabels.queued') }}
              </UButton>
              <UButton
                v-else-if="item.type === 'plugin' && item.status !== 'enabled'"
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
                v-else-if="themeActionState(item) === 'activate'"
                size="sm"
                icon="i-lucide-play"
                :loading="busyId === item.id"
                @click="activateTheme(item)"
              >
                {{ t('admin.extensions.activateTheme') }}
              </UButton>
              <UButton
                v-else-if="themeActionState(item) === 'failed'"
                size="sm"
                color="error"
                variant="subtle"
                icon="i-lucide-refresh-cw"
                :loading="busyId === item.id"
                @click="activateTheme(item)"
              >
                {{ t('admin.extensions.retryActivation') }}
              </UButton>
              <UButton
                v-else-if="['queued', 'building', 'activating'].includes(themeActionState(item))"
                size="sm"
                color="neutral"
                variant="subtle"
                icon="i-lucide-hourglass"
                disabled
              >
                {{ t(`admin.extensions.themeRelease.${item.themeRelease?.status || 'queued'}`) }}
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
              <UButton
                v-if="item.isDeletable && item.source !== 'builtin' && !item.isSystem && item.status !== 'enabled'"
                size="sm"
                color="error"
                variant="ghost"
                icon="i-lucide-trash-2"
                :loading="busyId === item.id"
                @click="openUninstallExtension(item)"
              >
                {{ t('admin.extensions.uninstall') }}
              </UButton>
            </div>
          </div>
        </div>
      </div>

      <aside class="space-y-4">
        <div class="rounded-lg border border-slate-200 bg-white p-4 dark:border-zinc-800 dark:bg-zinc-900">
          <h2 class="text-sm font-semibold text-slate-900 dark:text-zinc-100">
            {{ selectedDisplay?.name || t('admin.extensions.detailTitle') }}
          </h2>
          <p
            v-if="selectedDisplay?.description"
            class="mt-3 text-sm leading-6 text-slate-600 dark:text-zinc-300"
          >
            {{ selectedDisplay.description }}
          </p>
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
          <!-- F2.1 Host 能力授权列表 -->
          <div v-if="selected.capabilityGrants?.length" class="mt-4 space-y-2 border-t border-slate-100 pt-4 dark:border-zinc-800">
            <p class="text-xs font-medium uppercase tracking-wide text-slate-500 dark:text-zinc-400">
              {{ t('admin.extensions.hostCapabilities') }}
            </p>
            <div
              v-for="grant in selected.capabilityGrants"
              :key="grant.key"
              class="rounded-md bg-slate-50 px-3 py-2 text-sm dark:bg-zinc-950"
            >
              <div class="flex items-center justify-between gap-2">
                <span class="font-medium text-slate-900 dark:text-zinc-100">
                  {{ locale.startsWith('zh') ? grant.labelZh : grant.labelEn }}
                </span>
                <UBadge
                  size="xs"
                  variant="subtle"
                  :color="grant.risk === 'high' ? 'error' : grant.risk === 'medium' ? 'warning' : 'success'"
                >
                  {{ t(`admin.extensions.capabilityRisk.${grant.risk}`) }}
                </UBadge>
              </div>
              <p class="mt-1 text-xs text-slate-500 dark:text-zinc-400">
                <code class="text-[11px]">{{ grant.key }}</code>
                <span v-if="grant.implied" class="ml-2">{{ t('admin.extensions.capabilityImplied') }}</span>
              </p>
              <p class="mt-1 text-xs leading-5 text-slate-600 dark:text-zinc-300">
                {{ grant.description }}
              </p>
            </div>
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
            <div v-for="event in selectedEventPageInfo.items" :key="event.id" class="rounded-md bg-slate-50 p-3 text-sm dark:bg-zinc-900">
              <div class="flex items-center justify-between gap-3">
                <span class="font-medium text-slate-900 dark:text-zinc-100">{{ event.action }}</span>
                <span class="text-xs text-slate-500 dark:text-zinc-400">{{ formatSiteDateTime(event.createdAt) }}</span>
              </div>
              <p v-if="event.message" class="mt-1 text-slate-500 dark:text-zinc-400">
                {{ event.message }}
              </p>
            </div>
            <div
              v-if="selectedEventPageInfo.totalPages > 1"
              class="flex flex-col gap-3 border-t border-slate-200 pt-4 text-xs text-slate-500 sm:flex-row sm:items-center sm:justify-between dark:border-zinc-800 dark:text-zinc-400"
            >
              <span>
                {{ t('admin.extensions.eventPageSummary', { start: selectedEventPageInfo.start, end: selectedEventPageInfo.end, count: selectedEventPageInfo.total }) }}
              </span>
              <SFPagination v-model:page="selectedEventPage" :total-pages="selectedEventPageInfo.totalPages" />
            </div>
          </div>
        </div>
      </aside>
    </div>

    <!-- F2.1 启用前能力审查确认 -->
    <UModal v-model:open="enableConfirmOpen">
      <template #content>
        <div class="p-5 sm:p-6">
          <h2 class="text-base font-semibold text-slate-900 dark:text-zinc-100">
            {{ t('admin.extensions.confirmEnableTitle') }}
          </h2>
          <p class="mt-2 text-sm leading-6 text-slate-600 dark:text-zinc-300">
            {{ t('admin.extensions.confirmEnableBody', { name: enableConfirmItem?.name || '' }) }}
          </p>
          <ul v-if="enableConfirmItem?.capabilityGrants?.length" class="mt-4 max-h-64 space-y-2 overflow-y-auto">
            <li
              v-for="grant in enableConfirmItem.capabilityGrants"
              :key="grant.key"
              class="rounded-md border border-slate-200 px-3 py-2 text-sm dark:border-zinc-700"
            >
              <div class="flex items-center justify-between gap-2">
                <span class="font-medium">{{ locale.startsWith('zh') ? grant.labelZh : grant.labelEn }}</span>
                <UBadge
                  size="xs"
                  variant="subtle"
                  :color="grant.risk === 'high' ? 'error' : grant.risk === 'medium' ? 'warning' : 'success'"
                >
                  {{ t(`admin.extensions.capabilityRisk.${grant.risk}`) }}
                </UBadge>
              </div>
              <p class="mt-1 text-xs text-slate-500 dark:text-zinc-400">{{ grant.key }}</p>
            </li>
          </ul>
          <div class="mt-6 flex justify-end gap-2">
            <UButton color="neutral" variant="ghost" @click="cancelEnableExtension">
              {{ t('admin.extensions.confirmEnableCancel') }}
            </UButton>
            <UButton color="primary" icon="i-lucide-shield-check" @click="confirmEnableExtension">
              {{ t('admin.extensions.confirmEnableAction') }}
            </UButton>
          </div>
        </div>
      </template>
    </UModal>

    <UModal v-model:open="uninstallConfirmOpen">
      <template #content>
        <div class="p-5 sm:p-6">
          <h2 class="text-base font-semibold text-slate-900 dark:text-zinc-100">
            {{ t('admin.extensions.confirmUninstallTitle') }}
          </h2>
          <p class="mt-2 text-sm leading-6 text-slate-600 dark:text-zinc-300">
            {{ t('admin.extensions.confirmUninstallBody', { name: uninstallConfirmItem?.name || '' }) }}
          </p>
          <div class="mt-6 flex justify-end gap-2">
            <UButton color="neutral" variant="ghost" @click="cancelUninstallExtension">
              {{ t('admin.extensions.confirmUninstallCancel') }}
            </UButton>
            <UButton color="error" icon="i-lucide-trash-2" @click="confirmUninstallExtension">
              {{ t('admin.extensions.confirmUninstallAction') }}
            </UButton>
          </div>
        </div>
      </template>
    </UModal>
  </div>
</template>
