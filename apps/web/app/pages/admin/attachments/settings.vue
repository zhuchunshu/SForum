<script setup lang="ts">
import SFAdminAttachmentSettingsTab from '~/components/admin/settings/attachments/tabs/SFAdminAttachmentSettingsTab.vue'
import { useAdminPage } from '~/composables/admin/useAdminPage'

definePageMeta({ middleware: 'admin', layout: 'admin' })
defineOptions({ name: 'AdminAttachmentSettings' })

type RefreshablePage = { refresh?: () => Promise<void>, pending?: boolean }

const { t } = useI18n()
const adminPage = useAdminPage('/attachments/settings')
const pageRef = ref<RefreshablePage | null>(null)
const toolbarPending = computed(() => Boolean(pageRef.value?.pending))

useSeoMeta({ title: t('admin.attachments.tabs.settings') })

async function refreshPage() {
  await pageRef.value?.refresh?.()
}
</script>

<template>
  <div class="flex w-full min-w-0 flex-col gap-4">
    <header>
      <h2 class="flex items-center gap-2 text-xl font-bold text-slate-900 dark:text-zinc-100">
        <UIcon :name="adminPage.icon" class="size-5 text-[var(--sf-accent)] dark:text-[var(--sf-accent-dark)]" />
        {{ t('admin.attachments.tabs.settings') }}
      </h2>
      <p class="text-sm text-slate-500 dark:text-zinc-400">
        {{ t('admin.attachments.settingsDescription') }}
      </p>
    </header>

    <UDashboardToolbar class="overflow-x-hidden rounded-lg border border-slate-200 bg-white px-4 py-2.5 text-slate-500 dark:border-zinc-800 dark:bg-zinc-900 dark:text-zinc-400">
      <template #left>
        <div class="flex min-w-0 items-center gap-2 text-sm">
          <UIcon name="i-lucide-paperclip" class="size-4" />
          <span class="truncate">{{ t('admin.attachments.toolbar') }}</span>
        </div>
      </template>
      <template #right>
        <UButton color="neutral" variant="outline" leading-icon="i-lucide-refresh-cw" :loading="toolbarPending" @click="refreshPage">
          {{ t('admin.attachments.refresh') }}
        </UButton>
      </template>
    </UDashboardToolbar>

    <SFAdminAttachmentSettingsTab ref="pageRef" />
  </div>
</template>
