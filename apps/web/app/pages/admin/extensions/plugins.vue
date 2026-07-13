<script setup lang="ts">
import { apiErrorMessage } from '~/composables/useApiClient'
import { useAdminPage } from '~/composables/useAdminPage'
import {
  canRestartPlugin,
  capabilityCount,
  extensionLocalizedDisplay,
  extensionManageRoute,
  extensionSettingsPresentation,
  filterExtensionsByType,
  runtimeCapabilitySummary,
  runtimeStatusLabelKey,
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
  confirmEnableExtension,
  issueEnableTrustChallenge,
  cancelEnableExtension,
  enableConfirmOpen,
  enableConfirmItem,
  enableTrustMode,
  enableTrustStatus,
  enableTrustChallenge,
  enableTrustError,
  enableTrustBusy,
  isSuperAdmin,
  openUninstallExtension,
  confirmUninstallExtension,
  cancelUninstallExtension,
  uninstallConfirmOpen,
  uninstallConfirmItem,
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

function runtimeColor(state?: AdminRuntimeState) {
  if (state === 'running') {
    return 'success'
  }
  if (state === 'failed') {
    return 'error'
  }
  if (state === 'degraded' || state === 'starting') {
    return 'warning'
  }
  return 'neutral'
}

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
            <UBadge
              :color="extensionSettingsPresentation(item).color"
              variant="subtle"
              :icon="extensionSettingsPresentation(item).icon"
            >
              {{ t(extensionSettingsPresentation(item).labelKey) }}
            </UBadge>
            <UBadge
              v-if="item.runtime?.circuitOpen"
              color="warning"
              variant="subtle"
              icon="i-lucide-zap-off"
            >
              {{ t('admin.extensions.runtime.circuitOpen') }}
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
          <p
            v-if="item.runtime?.state === 'degraded' || item.runtime?.consecutiveFailures"
            class="mt-1 text-xs text-amber-700 dark:text-amber-300"
          >
            <span v-if="item.runtime?.consecutiveFailures">
              {{ t('admin.extensions.runtime.failures', { count: item.runtime.consecutiveFailures }) }}
            </span>
            <span v-if="item.runtime?.lastFailureReason" class="ml-1">
              · {{ t('admin.extensions.runtime.lastFailure', { reason: item.runtime.lastFailureReason }) }}
            </span>
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
			:disabled="!canRestartPlugin(item)"
            :loading="busyId === item.id && canRestartPlugin(item)"
            :title="canRestartPlugin(item) ? t('admin.extensions.restart') : t('admin.extensions.restartUnavailable')"
            @click="restartExtension(item)"
          >
            {{ t('admin.extensions.restart') }}
          </UButton>
          <UButton
            v-if="item.isDeletable && item.source !== 'builtin' && !item.isSystem"
            size="sm"
            color="error"
            variant="ghost"
            icon="i-lucide-trash-2"
			:disabled="item.status === 'enabled'"
            :loading="busyId === item.id"
            :title="item.status === 'enabled' ? t('admin.extensions.confirmUninstallBody', { name: item.name }) : t('admin.extensions.uninstall')"
            @click="openUninstallExtension(item)"
          >
            {{ t('admin.extensions.uninstall') }}
          </UButton>
        </div>
      </div>
    </div>

    <SFExtensionEnableDialog
      v-model:open="enableConfirmOpen"
      :extension="enableConfirmItem"
      :mode="enableTrustMode"
      :trust-status="enableTrustStatus"
      :challenge="enableTrustChallenge"
      :error="enableTrustError"
      :busy="enableTrustBusy || Boolean(enableConfirmItem && busyId === enableConfirmItem.id)"
      :is-super-admin="isSuperAdmin"
      @cancel="cancelEnableExtension"
      @issue-challenge="issueEnableTrustChallenge"
      @confirm="confirmEnableExtension"
    />

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
