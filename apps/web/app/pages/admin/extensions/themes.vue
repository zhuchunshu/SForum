<script setup lang="ts">
import { apiErrorMessage } from '~/composables/useApiClient'
import { useAdminPage } from '~/composables/useAdminPage'
import { capabilityCount, extensionLocalizedDisplay, extensionManageRoute, extensionSettingsPresentation, filterExtensionsByType, isExtensionArtifactAvailable, themeActionState, themeStatusLabelKey } from '~/utils/adminExtensions'

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
  themeActivateTrustMode,
  themeActivateConfirmOpen,
  themePreviewConfirmOpen,
  themeActivateConfirmItem,
  themeActivatePreview,
  themePreviewAddCount,
  themePreviewReplaceCount,
  themePreviewReactivating,
  themeActivateTrustStatus,
  themeActivateTrustChallenge,
  themeActivateTrustError,
  themeActivateTrustBusy,
  issueThemeActivateTrustChallenge,
  confirmThemeActivate,
  cancelThemeActivate,
  confirmThemePreviewActivate,
  cancelThemePreviewActivate,
  isSuperAdmin,
  openUninstallExtension,
  confirmUninstallExtension,
  cancelUninstallExtension,
  uninstallConfirmOpen,
  uninstallConfirmItem,
  uninstallRemovalMode,
  uninstallError,
  statusColor
} = await useAdminExtensionsManager()

const themes = computed(() => filterExtensionsByType(extensions.value, 'theme'))
// 与当前 UI 语言绑定，切换语言时主题名称/描述会立刻重算。
const themeRows = computed(() => themes.value.map((item) => ({
  item,
  display: extensionLocalizedDisplay(item, locale.value)
})))

function themeLabel(item: (typeof themes.value)[number]) {
  return t(themeStatusLabelKey(item))
}

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
              v-if="!isExtensionArtifactAvailable(item)"
              color="error"
              variant="subtle"
              icon="i-lucide-package-x"
            >
              {{ t('admin.extensions.artifact.missing') }}
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
          <p v-if="!isExtensionArtifactAvailable(item)" class="mt-1.5 text-sm text-red-600 dark:text-red-400">
            {{ t('admin.extensions.artifact.missingDescription') }}
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
            :to="isExtensionArtifactAvailable(item) ? adminRoutes.path(extensionManageRoute(item)) : undefined"
            :disabled="!isExtensionArtifactAvailable(item)"
            :title="!isExtensionArtifactAvailable(item) ? t('admin.extensions.artifact.actionUnavailable') : undefined"
          >
            {{ t('admin.extensions.manage') }}
          </UButton>
          <UButton
            v-if="themeActionState(item) === 'activateDefault'"
            size="sm"
            icon="i-lucide-rotate-ccw"
            :disabled="!isExtensionArtifactAvailable(item)"
            :loading="busyId === item.id"
            @click="activateTheme(item)"
          >
            {{ t('admin.extensions.restoreDefaultTheme') }}
          </UButton>
          <UButton
            v-else-if="themeActionState(item) === 'activate'"
            size="sm"
            icon="i-lucide-play"
            :disabled="!isExtensionArtifactAvailable(item)"
            :loading="busyId === item.id"
            @click="activateTheme(item)"
          >
            {{ t('admin.extensions.activateTheme') }}
          </UButton>
          <UButton
            v-else
            size="sm"
            color="neutral"
            variant="subtle"
            icon="i-lucide-refresh-cw"
            :disabled="!isExtensionArtifactAvailable(item)"
            :loading="busyId === item.id"
            :title="t('admin.extensions.reactivateThemeHint')"
            @click="activateTheme(item)"
          >
            {{ t('admin.extensions.reactivateTheme') }}
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

    <SFAdminExtensionUninstallDialog
      v-model:open="uninstallConfirmOpen"
      v-model:removal-mode="uninstallRemovalMode"
      :extension="uninstallConfirmItem"
      :busy="Boolean(uninstallConfirmItem && busyId === uninstallConfirmItem.id)"
      :error="uninstallError"
      @cancel="cancelUninstallExtension"
      @confirm="confirmUninstallExtension"
    />

    <!-- L0/L1 与 trust_not_required：页面预览确认 Modal（替代原生 confirm）。 -->
    <SFAdminThemeActivateDialog
      v-model:open="themePreviewConfirmOpen"
      :extension="themeActivateConfirmItem"
      :impacts="themeActivatePreview?.impacts || []"
      :add-count="themePreviewAddCount"
      :replace-count="themePreviewReplaceCount"
      :reactivating="themePreviewReactivating"
      :busy="Boolean(themeActivateConfirmItem && busyId === themeActivateConfirmItem.id)"
      @cancel="cancelThemePreviewActivate"
      @confirm="confirmThemePreviewActivate"
    />

    <!-- Executable (L2) theme activation reuses the exact trust challenge dialog pattern. -->
    <SFAdminExtensionEnableDialog
      v-model:open="themeActivateConfirmOpen"
      :extension="themeActivateConfirmItem"
      :mode="themeActivateTrustMode"
      :trust-status="themeActivateTrustStatus"
      :challenge="themeActivateTrustChallenge"
      :error="themeActivateTrustError"
      :busy="themeActivateTrustBusy || Boolean(themeActivateConfirmItem && busyId === themeActivateConfirmItem.id)"
      :is-super-admin="isSuperAdmin"
      purpose="activate"
      @cancel="cancelThemeActivate"
      @issue-challenge="issueThemeActivateTrustChallenge"
      @confirm="confirmThemeActivate"
    />
  </div>
</template>
