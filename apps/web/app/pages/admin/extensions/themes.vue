<script setup lang="ts">
import { apiErrorMessage } from '~/composables/useApiClient'
import { useAdminPage } from '~/composables/useAdminPage'
import { capabilityCount, extensionAdminPageRoute, filterExtensionsByType, themeActionState, themeStatusLabelKey } from '~/utils/adminExtensions'

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
  verifyExtension,
  statusColor
} = await useAdminExtensionsManager()

const themes = computed(() => filterExtensionsByType(extensions.value, 'theme'))

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
          <p v-if="themeActionState(item) === 'verifyOnly'" class="mt-2 text-xs leading-5 text-slate-500 dark:text-zinc-400">
            {{ t('admin.extensions.themes.runtimeUnavailable') }}
          </p>
        </div>
        <div class="flex items-center gap-2">
          <UButton
            size="sm"
            color="neutral"
            variant="ghost"
            icon="i-lucide-settings"
            :to="adminRoutes.path(extensionAdminPageRoute(item.id))"
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
</template>
