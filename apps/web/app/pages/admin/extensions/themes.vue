<script setup lang="ts">
import { apiErrorMessage } from '~/composables/useApiClient'
import { useAdminPage } from '~/composables/useAdminPage'
import { capabilityCount, extensionLocalizedDisplay, extensionManageRoute, extensionSettingsPresentation, filterExtensionsByType, hasThemeActivationInProgress, themeActionState, themeActivationProgress, themeStatusLabelKey, type AdminExtension } from '~/utils/adminExtensions'

definePageMeta({
  middleware: 'admin',
  layout: 'admin'
})

defineOptions({
  name: 'AdminExtensionThemes'
})

const { t, locale } = useI18n()
const adminPage = useAdminPage('/extensions/themes')
const adminRoutes = useAdminRoutes()
const {
  extensions,
  pending,
  error,
  refresh,
  busyId,
  activateTheme,
  openUninstallExtension,
  confirmUninstallExtension,
  cancelUninstallExtension,
  uninstallConfirmOpen,
  uninstallConfirmItem,
  statusColor
} = await useAdminExtensionsManager()

const themes = computed(() => filterExtensionsByType(extensions.value, 'theme'))
// 与当前 UI 语言绑定，切换语言时主题名称/描述会立刻重算。
const themeRows = computed(() => themes.value.map((item) => ({
  item,
  display: extensionLocalizedDisplay(item, locale.value)
})))
const activationPolling = computed(() => hasThemeActivationInProgress(themes.value))
let activationPollTimer: ReturnType<typeof setInterval> | null = null

function themeLabel(item: (typeof themes.value)[number]) {
  return t(themeStatusLabelKey(item))
}

function releaseProgress(item: AdminExtension) {
  return themeActivationProgress(item.themeRelease)
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
        v-for="{ item, display } in themeRows"
        :key="item.id"
        class="grid gap-4 px-4 py-4 md:grid-cols-[1fr_auto]"
      >
        <div class="min-w-0">
          <div class="flex flex-wrap items-center gap-2">
            <UIcon name="i-lucide-palette" class="size-4 text-[var(--sf-accent)]" />
            <h3 class="truncate text-sm font-semibold text-slate-900 dark:text-zinc-100">
              {{ display.name }}
            </h3>
            <UBadge :color="statusColor(item.status)" variant="subtle">
              {{ themeLabel(item) }}
            </UBadge>
            <UBadge v-if="item.source === 'builtin'" color="primary" variant="subtle" icon="i-lucide-shield-check">
              {{ t('admin.extensions.source.builtin') }}
            </UBadge>
            <UBadge
              :color="extensionSettingsPresentation(item).color"
              variant="subtle"
              :icon="extensionSettingsPresentation(item).icon"
            >
              {{ t(extensionSettingsPresentation(item).labelKey) }}
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
