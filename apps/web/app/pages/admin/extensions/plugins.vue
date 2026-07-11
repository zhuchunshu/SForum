<script setup lang="ts">
import { apiErrorMessage } from '~/composables/useApiClient'
import { useAdminPage } from '~/composables/useAdminPage'
import {
  canRestartPlugin,
  capabilityCount,
  extensionLocalizedDisplay,
  extensionManageRoute,
  filterExtensionsByType,
  hasPluginWebReleaseInProgress,
  pluginWebReleaseProgress,
  runtimeCapabilitySummary,
  runtimeStatusLabelKey,
  type AdminExtension,
  type AdminRuntimeState
} from '~/utils/adminExtensions'

definePageMeta({
  middleware: 'admin',
  layout: 'admin'
})

defineOptions({
  name: 'AdminExtensionPlugins'
})

const { t, locale } = useI18n()
const adminPage = useAdminPage('/extensions/plugins')
const adminRoutes = useAdminRoutes()
const {
  extensions,
  pending,
  error,
  refresh,
  busyId,
  enableExtension,
  disableExtension,
  restartExtension,
  statusColor,
  statusLabel
} = await useAdminExtensionsManager()

const plugins = computed(() => filterExtensionsByType(extensions.value, 'plugin'))
// 与当前 UI 语言绑定，切换语言时列表文案会立刻重算。
const pluginRows = computed(() => plugins.value.map((item) => ({
  item,
  display: extensionLocalizedDisplay(item, locale.value)
})))
const releasePolling = computed(() => hasPluginWebReleaseInProgress(plugins.value))
const expandedLogReleaseIds = ref<Record<number, boolean>>({})
let releasePollTimer: ReturnType<typeof setInterval> | null = null

function runtimeColor(state?: AdminRuntimeState) {
  if (state === 'running') {
    return 'success'
  }
  if (state === 'failed') {
    return 'error'
  }
  if (state === 'starting') {
    return 'warning'
  }
  return 'neutral'
}

function releaseProgress(item: AdminExtension) {
  return pluginWebReleaseProgress(item.webRelease)
}

function pluginActionBusy(item: AdminExtension) {
  return Boolean(releaseProgress(item)?.active)
}

// 与主题页一致：进度条下方可展开构建日志，失败时默认打开。
function pluginBuildLog(item: AdminExtension) {
  return item.webRelease?.buildLog?.trim() || ''
}

function hasBuildLogToggle(item: AdminExtension) {
  return Boolean(item.webRelease && (pluginBuildLog(item) || item.webRelease.status === 'failed' || releaseProgress(item)?.active))
}

function isBuildLogOpen(item: AdminExtension) {
  const release = item.webRelease
  if (!release) {
    return false
  }
  if (Object.prototype.hasOwnProperty.call(expandedLogReleaseIds.value, release.id)) {
    return expandedLogReleaseIds.value[release.id]
  }
  return release.status === 'failed'
}

function toggleBuildLog(item: AdminExtension) {
  const release = item.webRelease
  if (!release) {
    return
  }
  expandedLogReleaseIds.value = {
    ...expandedLogReleaseIds.value,
    [release.id]: !isBuildLogOpen(item)
  }
}

function startReleasePolling() {
  if (releasePollTimer || !import.meta.client) {
    return
  }
  releasePollTimer = setInterval(async () => {
    if (pending.value) {
      return
    }
    await refresh()
  }, 2000)
}

function stopReleasePolling() {
  if (!releasePollTimer) {
    return
  }
  clearInterval(releasePollTimer)
  releasePollTimer = null
}

watch(releasePolling, (active) => {
  if (active) {
    startReleasePolling()
  } else {
    stopReleasePolling()
  }
})

onMounted(() => {
  if (releasePolling.value) {
    startReleasePolling()
  }
})

onBeforeUnmount(stopReleasePolling)

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
        v-for="{ item, display } in pluginRows"
        :key="item.id"
        class="grid gap-4 px-4 py-4 md:grid-cols-[1fr_auto]"
      >
        <div class="min-w-0">
          <div class="flex flex-wrap items-center gap-2">
            <UIcon name="i-lucide-plug" class="size-4 text-[var(--sf-accent)]" />
            <h3 class="truncate text-sm font-semibold text-slate-900 dark:text-zinc-100">
              {{ display.name }}
            </h3>
            <UBadge :color="statusColor(item.status)" variant="subtle">
              {{ statusLabel(item.status) }}
            </UBadge>
            <UBadge :color="runtimeColor(item.runtime?.state)" variant="subtle">
              {{ t(runtimeStatusLabelKey(item)) }}
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
          <p class="mt-1 flex flex-wrap gap-x-3 gap-y-1 text-xs text-slate-500 dark:text-zinc-400">
            <span>{{ t('admin.extensions.capability.routes', { count: runtimeCapabilitySummary(item).routes }) }}</span>
            <span>{{ t('admin.extensions.capability.hooks', { count: runtimeCapabilitySummary(item).hooks }) }}</span>
            <span>{{ t('admin.extensions.capability.events', { count: runtimeCapabilitySummary(item).events }) }}</span>
            <span>{{ t('admin.extensions.capability.providers', { count: runtimeCapabilitySummary(item).providers }) }}</span>
          </p>
          <p v-if="item.runtime?.lastError" class="mt-1 truncate text-xs text-red-600 dark:text-red-400">
            {{ item.runtime.lastError }}
          </p>
          <p v-if="item.webRelease?.publicMessage" class="mt-2 text-xs leading-5 text-slate-500 dark:text-zinc-400">
            {{ item.webRelease.publicMessage }}
          </p>
          <div
            v-if="releaseProgress(item)"
            class="mt-3 max-w-xl rounded-md border border-slate-200 bg-slate-50 p-3 dark:border-zinc-800 dark:bg-zinc-950/50"
          >
            <div class="mb-2 flex items-center justify-between gap-3 text-xs">
              <span class="inline-flex min-w-0 items-center gap-1.5 font-medium text-slate-700 dark:text-zinc-200">
                <UIcon :name="releaseProgress(item)?.icon || 'i-lucide-hourglass'" class="size-3.5 shrink-0" />
                <span class="truncate">{{ t(releaseProgress(item)?.labelKey || 'admin.extensions.releases.statusLabels.queued') }}</span>
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
              {{ t(releaseProgress(item)?.detailKey || 'admin.extensions.webReleaseProgress.queued') }}
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
              >{{ pluginBuildLog(item) || t('admin.extensions.emptyBuildLog') }}</pre>
            </div>
          </div>
          <SFAdminFrontendTrustPanel :extension="item" />
        </div>
        <div class="flex items-center gap-2">
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
            v-else-if="item.status !== 'enabled'"
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
            :disabled="!canRestartPlugin(item) || pluginActionBusy(item)"
            :loading="busyId === item.id && canRestartPlugin(item)"
            :title="canRestartPlugin(item) ? t('admin.extensions.restart') : t('admin.extensions.restartUnavailable')"
            @click="restartExtension(item)"
          >
            {{ t('admin.extensions.restart') }}
          </UButton>
        </div>
      </div>
    </div>
  </div>
</template>
