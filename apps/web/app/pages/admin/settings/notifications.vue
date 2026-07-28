<script setup lang="ts">
import { useAdminPage } from '~/composables/admin/useAdminPage'
import SFAdminNotificationPolicyPage from '~/components/admin/settings/notifications/SFAdminNotificationPolicyPage.vue'

definePageMeta({ middleware: 'admin', layout: 'admin' })
defineOptions({ name: 'AdminNotificationSettings' })

const { t } = useI18n()
const adminPage = useAdminPage('/settings/notifications')
const pageRef = ref<{ refresh?: () => Promise<void>, pending?: boolean } | null>(null)

useSeoMeta({ title: t('admin.notificationSettings.metaTitle') })
</script>

<template>
  <div class="space-y-5">
    <header>
      <h1 class="flex items-center gap-2 font-bold"><UIcon :name="adminPage.icon" class="size-5 text-[var(--sf-accent)]" />{{ t('admin.notificationSettings.title') }}</h1>
      <p class="mt-1 text-sm text-muted">{{ t('admin.notificationSettings.description') }}</p>
    </header>
    <UDashboardToolbar class="rounded-lg border border-slate-200 bg-white px-4 py-2.5 text-slate-500 dark:border-zinc-800 dark:bg-zinc-900">
      <template #left><div class="flex min-w-0 items-center gap-2 text-sm"><UIcon name="i-lucide-bell-ring" class="size-4" /><span class="truncate">{{ t('admin.notificationSettings.toolbar') }}</span></div></template>
      <template #right><UButton color="neutral" variant="outline" icon="i-lucide-refresh-cw" :loading="pageRef?.pending" @click="pageRef?.refresh?.()">{{ t('admin.common.refresh') }}</UButton></template>
    </UDashboardToolbar>
    <SFAdminNotificationPolicyPage ref="pageRef" />
  </div>
</template>
