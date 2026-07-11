<script setup lang="ts">
import { apiErrorMessage } from '~/composables/useApiClient'
import { useAdminPage } from '~/composables/useAdminPage'
import { extensionSettingDeclarations } from '~/utils/adminExtensions'

definePageMeta({
  middleware: 'admin',
  layout: 'admin'
})

defineOptions({
  name: 'AdminExtensionSettings'
})

const { t, locale } = useI18n()
const adminPage = useAdminPage('/extensions/settings')
const {
  extensions,
  pending,
  error,
  refresh
} = await useAdminExtensionsManager()

const settings = computed(() => extensionSettingDeclarations(extensions.value, locale.value))

useSeoMeta({
  title: t('admin.extensions.settings.metaTitle')
})
</script>

<template>
  <div class="mb-4 flex flex-col gap-1">
    <h2 class="text-xl font-bold flex items-center gap-2 text-slate-900 dark:text-zinc-100">
      <UIcon :name="adminPage.icon" class="size-5 text-[var(--sf-accent)] dark:text-[var(--sf-accent-dark)]" />
      {{ t('admin.extensions.settings.title') }}
    </h2>
    <p class="text-sm text-slate-500 dark:text-zinc-400">
      {{ t('admin.extensions.settings.intro') }}
    </p>
  </div>

  <UDashboardToolbar class="border border-slate-200 dark:border-zinc-800 bg-white dark:bg-zinc-900 rounded-lg px-4 py-2.5 mb-6 text-slate-500 dark:text-zinc-400">
    <template #left>
      <div class="flex min-w-0 items-center gap-2 text-sm">
        <UIcon name="i-lucide-sliders-horizontal" class="size-4" />
        <span class="truncate">{{ t('admin.extensions.settings.count', { count: settings.length }) }}</span>
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

  <div class="rounded-lg border border-slate-200 bg-white dark:border-zinc-800 dark:bg-zinc-900">
    <div v-if="settings.length === 0 && !pending" class="p-10">
      <SFEmptyState icon-label="CFG" :title="t('admin.extensions.settings.emptyTitle')" :description="t('admin.extensions.settings.emptyDescription')" />
    </div>
    <div v-else class="divide-y divide-slate-200 dark:divide-zinc-800">
      <div
        v-for="item in settings"
        :key="`${item.extensionId}:${item.setting.key}`"
        class="grid gap-3 px-4 py-4 md:grid-cols-[minmax(0,1fr)_180px]"
      >
        <div class="min-w-0">
          <div class="flex flex-wrap items-center gap-2">
            <UIcon :name="item.extensionType === 'theme' ? 'i-lucide-palette' : 'i-lucide-plug'" class="size-4 text-[var(--sf-accent)]" />
            <h3 class="truncate text-sm font-semibold text-slate-900 dark:text-zinc-100">
              {{ item.setting.label }}
            </h3>
            <UBadge color="neutral" variant="outline">
              {{ item.setting.key }}
            </UBadge>
          </div>
          <p class="mt-1 truncate text-xs text-slate-500 dark:text-zinc-400">
            {{ item.extensionName }} · {{ item.extensionId }}
          </p>
        </div>
        <div class="flex items-center md:justify-end">
          <UBadge color="primary" variant="subtle" icon="i-lucide-file-cog">
            {{ item.setting.type }}
          </UBadge>
        </div>
      </div>
    </div>
  </div>
</template>
