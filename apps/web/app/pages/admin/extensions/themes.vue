<script setup lang="ts">
import { apiErrorMessage } from '~/composables/useApiClient'
import { useAdminPage } from '~/composables/useAdminPage'
import { capabilityCount, extensionAuthorName, extensionAuthorWebsite, extensionManageRoute, filterExtensionsByType, hasThemeActivationInProgress, themeActionState, themeActivationProgress, themeStatusLabelKey, type AdminExtension } from '~/utils/adminExtensions'

definePageMeta({
  middleware: 'admin',
  layout: 'admin'
})

defineOptions({
  name: 'AdminExtensionThemes'
})

const { t } = useI18n()
const adminPage = useAdminPage('/extensions/themes')
const adminRoutes = useAdminRoutes()
const {
  extensions,
  pending,
  error,
  refresh,
  busyId,
  activateTheme,
  statusColor
} = await useAdminExtensionsManager()

const themes = computed(() => filterExtensionsByType(extensions.value, 'theme'))
const activationPolling = computed(() => hasThemeActivationInProgress(themes.value))
const expandedLogReleaseIds = ref<Record<number, boolean>>({})
let activationPollTimer: ReturnType<typeof setInterval> | null = null

function themeLabel(item: (typeof themes.value)[number]) {
  return t(themeStatusLabelKey(item))
}

function releaseProgress(item: AdminExtension) {
  return themeActivationProgress(item.themeRelease)
}

function themeBuildLog(item: AdminExtension) {
  return item.themeRelease?.buildLog?.trim() || ''
}

function hasBuildLogToggle(item: AdminExtension) {
  return Boolean(item.themeRelease && (themeBuildLog(item) || item.themeRelease.status === 'failed'))
}

function isBuildLogOpen(item: AdminExtension) {
  const release = item.themeRelease
  if (!release) {
    return false
  }
  if (Object.prototype.hasOwnProperty.call(expandedLogReleaseIds.value, release.id)) {
    return expandedLogReleaseIds.value[release.id]
  }
  return release.status === 'failed'
}

function toggleBuildLog(item: AdminExtension) {
  const release = item.themeRelease
  if (!release) {
    return
  }
  expandedLogReleaseIds.value = {
    ...expandedLogReleaseIds.value,
    [release.id]: !isBuildLogOpen(item)
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

useSeoMeta({
  title: t('admin.extensions.themes.metaTitle')
})
</script>

<template>
  <div class="mb-4 flex flex-col gap-1">
    <h2 class="text-xl font-bold flex items-center gap-2 text-slate-900 dark:text-zinc-100">
      <UIcon :name="adminPage.icon" class="size-5 text-[var(--sf-accent)] dark:text-[var(--sf-accent-dark)]" />
      {{ t('admin.extensions.themes.title') }}
    </h2>
    <p class="text-sm text-slate-500 dark:text-zinc-400">
      {{ t('admin.extensions.themes.intro') }}
    </p>
  </div>

  <UDashboardToolbar class="border border-slate-200 dark:border-zinc-800 bg-white dark:bg-zinc-900 rounded-lg px-4 py-2.5 mb-6 text-slate-500 dark:text-zinc-400">
    <template #left>
      <div class="flex min-w-0 items-center gap-2 text-sm">
        <UIcon name="i-lucide-palette" class="size-4" />
        <span class="truncate">{{ t('admin.extensions.themes.count', { count: themes.length }) }}</span>
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
    <div v-if="themes.length === 0 && !pending" class="p-10">
      <SFEmptyState icon-label="THM" :title="t('admin.extensions.themes.emptyTitle')" :description="t('admin.extensions.themes.emptyDescription')" />
    </div>
    <div v-else class="divide-y divide-slate-200 dark:divide-zinc-800">
      <div
        v-for="item in themes"
        :key="item.id"
        class="grid gap-4 px-4 py-4 md:grid-cols-[1fr_auto]"
      >
        <div class="min-w-0">
          <div class="flex flex-wrap items-center gap-2">
            <UIcon name="i-lucide-palette" class="size-4 text-[var(--sf-accent)]" />
            <h3 class="truncate text-sm font-semibold text-slate-900 dark:text-zinc-100">
              {{ item.name }}
            </h3>
            <UBadge :color="statusColor(item.status)" variant="subtle">
              {{ themeLabel(item) }}
            </UBadge>
            <UBadge v-if="item.source === 'builtin'" color="primary" variant="subtle" icon="i-lucide-shield-check">
              {{ t('admin.extensions.source.builtin') }}
            </UBadge>
          </div>
          <p class="mt-1 truncate text-xs text-slate-500 dark:text-zinc-400">
            {{ item.id }} · v{{ item.version }} · {{ t('admin.extensions.capabilityCount', { count: capabilityCount(item) }) }}
          </p>
          <a
            v-if="extensionAuthorWebsite(item)"
            :href="extensionAuthorWebsite(item)"
            target="_blank"
            rel="noopener noreferrer"
            class="mt-2 inline-flex max-w-full items-center gap-1.5 rounded text-xs font-medium text-slate-500 transition hover:text-[var(--sf-accent)] focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-[var(--sf-accent)] focus-visible:ring-offset-2 focus-visible:ring-offset-white dark:text-zinc-400 dark:hover:text-[var(--sf-accent-dark)] dark:focus-visible:ring-offset-zinc-900"
            :title="t('admin.extensions.authorWebsiteTitle', { name: extensionAuthorName(item) })"
            :aria-label="t('admin.extensions.authorWebsiteTitle', { name: extensionAuthorName(item) })"
          >
            <UIcon name="i-lucide-user-round" class="size-3.5 shrink-0" />
            <span class="truncate">{{ t('admin.extensions.authorLinkLabel', { name: extensionAuthorName(item) }) }}</span>
            <UIcon name="i-lucide-external-link" class="size-3 shrink-0" />
          </a>
          <span v-else-if="extensionAuthorName(item)" class="mt-2 inline-flex max-w-full items-center gap-1.5 text-xs text-slate-500 dark:text-zinc-400">
            <UIcon name="i-lucide-user-round" class="size-3.5 shrink-0" />
            <span class="truncate">{{ t('admin.extensions.authorLinkLabel', { name: extensionAuthorName(item) }) }}</span>
          </span>
          <p v-if="item.themeRelease?.message" class="mt-2 text-xs leading-5 text-slate-500 dark:text-zinc-400">
            {{ item.themeRelease.message }}
          </p>
          <div
            v-if="releaseProgress(item)"
            class="mt-3 max-w-xl rounded-md border border-slate-200 bg-slate-50 p-3 dark:border-zinc-800 dark:bg-zinc-950/50"
          >
            <div class="mb-2 flex items-center justify-between gap-3 text-xs">
              <span class="inline-flex min-w-0 items-center gap-1.5 font-medium text-slate-700 dark:text-zinc-200">
                <UIcon :name="releaseProgress(item)?.icon || 'i-lucide-hourglass'" class="size-3.5 shrink-0" />
                <span class="truncate">{{ t(releaseProgress(item)?.labelKey || 'admin.extensions.themeRelease.queued') }}</span>
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
              {{ t(releaseProgress(item)?.detailKey || 'admin.extensions.themeProgress.queued') }}
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
              >{{ themeBuildLog(item) || t('admin.extensions.emptyBuildLog') }}</pre>
            </div>
          </div>
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
            v-if="themeActionState(item) === 'activateDefault'"
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
        </div>
      </div>
    </div>
  </div>
</template>
