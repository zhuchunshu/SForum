<script setup lang="ts">
import { useAdminRoutes } from '~/composables/admin/useAdminRoutes'
import { useAdminExtensionsManager } from '~/composables/admin/useAdminExtensionsManager'
import { useAdminExtensionUpgradeFlow } from '~/composables/admin/useAdminExtensionUpgradeFlow'
import { apiErrorMessage } from '~/composables/useApiClient'
import { useAdminPage } from '~/composables/admin/useAdminPage'
import SFAdminExtensionEnableDialog from '~/components/admin/SFAdminExtensionEnableDialog.vue'
import SFAdminExtensionLifecycleDialog from '~/components/admin/SFAdminExtensionLifecycleDialog.vue'
import SFAdminExtensionUninstallDialog from '~/components/admin/SFAdminExtensionUninstallDialog.vue'
import SFAdminFrontendTrustPanel from '~/components/admin/SFAdminFrontendTrustPanel.vue'
import {
  canRestartPlugin,
  canUpgradePlugin,
  capabilityCount,
  extensionLocalizedDisplay,
  extensionManageRoute,
  extensionSettingsPresentation,
  filterExtensionsByType,
  formatPluginMemoryBytes,
  isExtensionArtifactAvailable,
  isLifecycleV2Plugin,
  runtimeCapabilitySummary,
  runtimeStatusLabelKey,
  type AdminExtension,
  type AdminRuntimeState
} from '~/utils/admin/adminExtensions'

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
  uninstallRemovalMode,
  uninstallError,
  lifecycleDialogOpen,
  lifecycleDialogItem,
  lifecycleOperations,
  lifecycleOperation,
  lifecycleLoading,
  lifecycleRecoveryBusy,
  lifecycleError,
  openLifecycleExtension,
  selectLifecycleOperation,
  recoverLifecycleOperation,
  disableExtension,
  restartExtension,
  statusColor,
  statusLabel
} = await useAdminExtensionsManager()
const {
  upgradeCandidates,
  upgradeConfirmOpen,
  upgradeConfirmItem,
  upgradeTrustMode,
  upgradeTrustStatus,
  upgradeTrustChallenge,
  upgradeTrustError,
  upgradeTrustBusy,
  upgradeBusyId,
  upgradeAllBusy,
  upgradeAllCompleted,
  upgradeAllTotal,
  upgradeExtension,
  upgradeAllExtensions,
  issueUpgradeTrustChallenge,
  confirmUpgradeExtension,
  cancelUpgradeExtension
} = useAdminExtensionUpgradeFlow({ extensions, refresh, isSuperAdmin })

const plugins = computed(() => filterExtensionsByType(extensions.value, 'plugin'))
const inspectorMenuItems = computed(() => [[
  { label: t('admin.nav.extensionRouteProviders'), icon: 'i-lucide-route', to: adminRoutes.path('/extensions/route-providers') },
  { label: t('admin.nav.extensionRouteInspector'), icon: 'i-lucide-scan-search', to: adminRoutes.path('/extensions/route-inspector') },
  { label: t('admin.nav.extensionCacheInspector'), icon: 'i-lucide-database-zap', to: adminRoutes.path('/extensions/cache-inspector') },
  { label: t('admin.nav.extensionAssetInspector'), icon: 'i-lucide-package', to: adminRoutes.path('/extensions/asset-inspector') },
  { label: t('admin.nav.extensionTemplateInspector'), icon: 'i-lucide-layout-template', to: adminRoutes.path('/extensions/template-inspector') },
  { label: t('admin.nav.extensionComponentInspector'), icon: 'i-lucide-boxes', to: adminRoutes.path('/extensions/component-inspector') },
  { label: t('admin.nav.extensionNavigationInspector'), icon: 'i-lucide-map', to: adminRoutes.path('/extensions/navigation-inspector') },
  { label: t('admin.nav.extensionRegistryCatalogs'), icon: 'i-lucide-library', to: adminRoutes.path('/extensions/registry-catalogs') },
  { label: t('admin.nav.extensionProviderSlots'), icon: 'i-lucide-waypoints', to: adminRoutes.path('/extensions/provider-slots') }
]])
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

function storageInstancesRoute(item: AdminExtension) {
  const provider = item.manifest.providers?.find(candidate => candidate.slot === 'attachment.storage.provider' && candidate.multiInstance)
  if (!provider) return ''
  return `${adminRoutes.path('/attachments/settings')}?provider=${encodeURIComponent(item.id)}`
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
        <span class="hidden truncate sm:inline">{{ t('admin.extensions.plugins.count', { count: plugins.length }) }}</span>
      </div>
    </template>
    <template #right>
      <UButton
        v-if="upgradeCandidates.length"
        icon="i-lucide-package-plus"
        color="warning"
        variant="subtle"
        :loading="upgradeAllBusy"
        :aria-label="upgradeAllBusy
          ? t('admin.extensions.upgradeAllProgress', { completed: upgradeAllCompleted, total: upgradeAllTotal })
          : t('admin.extensions.upgradeAll', { count: upgradeCandidates.length })"
        :title="upgradeAllBusy
          ? t('admin.extensions.upgradeAllProgress', { completed: upgradeAllCompleted, total: upgradeAllTotal })
          : t('admin.extensions.upgradeAll', { count: upgradeCandidates.length })"
        @click="upgradeAllExtensions"
      >
        <span class="hidden xl:inline">
          {{ upgradeAllBusy
            ? t('admin.extensions.upgradeAllProgress', { completed: upgradeAllCompleted, total: upgradeAllTotal })
            : t('admin.extensions.upgradeAll', { count: upgradeCandidates.length }) }}
        </span>
      </UButton>
      <UDropdownMenu :items="inspectorMenuItems" :content="{ align: 'end' }">
        <UButton
          class="xl:hidden"
          icon="i-lucide-ellipsis"
          color="neutral"
          variant="subtle"
          :aria-label="t('common.more')"
          :title="t('common.more')"
        />
      </UDropdownMenu>
      <UButton
        class="hidden xl:inline-flex"
        icon="i-lucide-route"
        color="neutral"
        variant="subtle"
        :to="adminRoutes.path('/extensions/route-providers')"
      >
        {{ t('admin.nav.extensionRouteProviders') }}
      </UButton>
      <UButton
        class="hidden xl:inline-flex"
        icon="i-lucide-scan-search"
        color="neutral"
        variant="subtle"
        :to="adminRoutes.path('/extensions/route-inspector')"
      >
        {{ t('admin.nav.extensionRouteInspector') }}
      </UButton>
      <UButton
        class="hidden xl:inline-flex"
        icon="i-lucide-database-zap"
        color="neutral"
        variant="subtle"
        :to="adminRoutes.path('/extensions/cache-inspector')"
      >
        {{ t('admin.nav.extensionCacheInspector') }}
      </UButton>
      <UButton
        class="hidden xl:inline-flex"
        icon="i-lucide-package"
        color="neutral"
        variant="subtle"
        :to="adminRoutes.path('/extensions/asset-inspector')"
      >
        {{ t('admin.nav.extensionAssetInspector') }}
      </UButton>
      <UButton
        class="hidden xl:inline-flex"
        icon="i-lucide-layout-template"
        color="neutral"
        variant="subtle"
        :to="adminRoutes.path('/extensions/template-inspector')"
      >
        {{ t('admin.nav.extensionTemplateInspector') }}
      </UButton>
      <UButton
        class="hidden xl:inline-flex"
        icon="i-lucide-boxes"
        color="neutral"
        variant="subtle"
        :to="adminRoutes.path('/extensions/component-inspector')"
      >
        {{ t('admin.nav.extensionComponentInspector') }}
      </UButton>
      <UButton
        class="hidden xl:inline-flex"
        icon="i-lucide-map"
        color="neutral"
        variant="subtle"
        :to="adminRoutes.path('/extensions/navigation-inspector')"
      >
        {{ t('admin.nav.extensionNavigationInspector') }}
      </UButton>
      <UButton
        class="hidden xl:inline-flex"
        icon="i-lucide-library"
        color="neutral"
        variant="subtle"
        :to="adminRoutes.path('/extensions/registry-catalogs')"
      >
        {{ t('admin.nav.extensionRegistryCatalogs') }}
      </UButton>
      <UButton
        class="hidden xl:inline-flex"
        icon="i-lucide-waypoints"
        color="neutral"
        variant="subtle"
        :to="adminRoutes.path('/extensions/provider-slots')"
      >
        {{ t('admin.nav.extensionProviderSlots') }}
      </UButton>
      <UButton
        icon="i-lucide-rotate-cw"
        color="neutral"
        variant="subtle"
        :loading="pending"
        :aria-label="t('admin.extensions.refresh')"
        :title="t('admin.extensions.refresh')"
        @click="refresh()"
      >
        <span class="hidden xl:inline">{{ t('admin.extensions.refresh') }}</span>
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
              v-if="item.stagedVersion"
              color="warning"
              variant="outline"
              icon="i-lucide-package-plus"
              :title="t('admin.extensions.stagedVersionHint', { version: item.stagedVersion.version, current: item.version })"
            >
              {{ t('admin.extensions.stagedVersionBadge', { version: item.stagedVersion.version }) }}
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
              v-if="item.runtime?.protocolDeprecated"
              color="warning"
              variant="subtle"
              icon="i-lucide-history"
            >
              {{ t('admin.extensions.runtime.protocolDeprecated', { version: item.runtime.protocolVersion }) }}
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
            <UBadge
              v-if="item.runtime?.memoryBytes"
              color="neutral"
              variant="subtle"
              icon="i-lucide-memory-stick"
              :title="t('admin.extensions.runtime.memoryHint')"
            >
              {{ t('admin.extensions.runtime.memory', { size: formatPluginMemoryBytes(item.runtime.memoryBytes) }) }}
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
          <p
            v-if="item.runtime?.protocolVersion"
            class="mt-1 flex flex-wrap gap-x-3 gap-y-1 text-xs text-slate-500 dark:text-zinc-400"
          >
            <span>{{ t('admin.extensions.runtime.protocol', { transport: item.runtime.protocolTransport, version: item.runtime.protocolVersion }) }}</span>
            <span>{{ t('admin.extensions.runtime.protocolStarts', { count: item.runtime.protocolStartCount || 0 }) }}</span>
            <span>{{ t('admin.extensions.runtime.protocolCalls', { count: item.runtime.protocolCallCount || 0 }) }}</span>
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
        <div class="flex flex-wrap items-center gap-2 [&>a]:shrink-0 [&>button]:shrink-0">
          <UButton
            v-if="item.stagedVersion"
            size="sm"
            color="warning"
            variant="subtle"
            icon="i-lucide-package-plus"
            :disabled="upgradeAllBusy || !canUpgradePlugin(item)"
            :loading="upgradeBusyId === item.id"
            :title="canUpgradePlugin(item)
              ? t('admin.extensions.stagedVersionHint', { version: item.stagedVersion.version, current: item.version })
              : t('admin.extensions.upgradeUnavailable')"
            @click="upgradeExtension(item)"
          >
            {{ t('admin.extensions.upgrade') }}
          </UButton>
          <UButton
            v-if="storageInstancesRoute(item)"
            size="sm"
            color="neutral"
            variant="subtle"
            icon="i-lucide-database"
            :to="item.status === 'enabled' ? storageInstancesRoute(item) : undefined"
            :disabled="item.status !== 'enabled'"
            :title="item.status === 'enabled' ? t('admin.extensions.configureStorage') : t('admin.extensions.configureStorageEnableFirst')"
          >
            {{ t('admin.extensions.configureStorage') }}
          </UButton>
          <UButton
            size="sm"
            color="neutral"
            variant="ghost"
            icon="i-lucide-history"
            :title="t('admin.extensions.lifecycle.title')"
            :aria-label="t('admin.extensions.lifecycle.title')"
            @click="openLifecycleExtension(item)"
          />
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
            v-if="item.status !== 'enabled'"
            size="sm"
            icon="i-lucide-play"
            :disabled="!isExtensionArtifactAvailable(item)"
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
            :disabled="item.status === 'enabled' && !isLifecycleV2Plugin(item)"
            :loading="busyId === item.id"
            :title="item.status === 'enabled' && !isLifecycleV2Plugin(item) ? t('admin.extensions.confirmUninstallBody', { name: item.name }) : t('admin.extensions.uninstall')"
            @click="openUninstallExtension(item)"
          >
            {{ t('admin.extensions.uninstall') }}
          </UButton>
        </div>
      </div>
    </div>

    <SFAdminExtensionEnableDialog
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

    <SFAdminExtensionEnableDialog
      v-model:open="upgradeConfirmOpen"
      :extension="upgradeConfirmItem"
      :mode="upgradeTrustMode"
      :trust-status="upgradeTrustStatus"
      :challenge="upgradeTrustChallenge"
      :error="upgradeTrustError"
      :busy="upgradeTrustBusy || Boolean(upgradeConfirmItem && upgradeBusyId === upgradeConfirmItem.id)"
      :is-super-admin="isSuperAdmin"
      purpose="upgrade"
      @cancel="cancelUpgradeExtension"
      @issue-challenge="issueUpgradeTrustChallenge"
      @confirm="confirmUpgradeExtension"
    />

    <SFAdminExtensionUninstallDialog
      v-model:open="uninstallConfirmOpen"
      v-model:removal-mode="uninstallRemovalMode"
      :extension="uninstallConfirmItem"
      :busy="Boolean(uninstallConfirmItem && busyId === uninstallConfirmItem.id)"
      :error="uninstallError"
      @cancel="cancelUninstallExtension"
      @confirm="confirmUninstallExtension"
    />

    <SFAdminExtensionLifecycleDialog
      v-model:open="lifecycleDialogOpen"
      :extension="lifecycleDialogItem"
      :operations="lifecycleOperations"
      :operation="lifecycleOperation"
      :loading="lifecycleLoading"
      :recovery-busy="lifecycleRecoveryBusy"
      :error="lifecycleError"
      :is-super-admin="isSuperAdmin"
      @select="selectLifecycleOperation"
      @recover="recoverLifecycleOperation"
    />
  </div>
</template>
