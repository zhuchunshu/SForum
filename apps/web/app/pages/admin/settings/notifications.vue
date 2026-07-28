<script setup lang="ts">
import type { Component } from 'vue'
import { useAdminPage } from '~/composables/admin/useAdminPage'
import { usePermissions } from '~/composables/identity/usePermissions'
import SFAdminFixedTabNav from '~/components/admin/settings/shared/SFAdminFixedTabNav.vue'
import SFAdminNotificationChannels from '~/components/admin/settings/notifications/SFAdminNotificationChannels.vue'
import SFAdminNotificationPolicyPage from '~/components/admin/settings/notifications/SFAdminNotificationPolicyPage.vue'

definePageMeta({ middleware: 'admin', layout: 'admin' })
defineOptions({ name: 'AdminNotificationSettings' })

const { t } = useI18n()
const route = useRoute()
const router = useRouter()
const adminPage = useAdminPage('/settings/notifications')
const { can } = usePermissions()
const canManage = computed(() => can('settings.notifications.manage'))
const pageRef = ref<{ refresh?: () => Promise<void>, pending?: boolean } | null>(null)
type NotificationTab = 'policy' | 'channels'
const tabIds: NotificationTab[] = ['policy', 'channels']
const activeTab = ref<NotificationTab>(normalizeTab(route.query.tab))
const tabs = computed(() => [
  { id: 'policy', label: t('admin.notificationSettings.tabs.policy'), icon: 'i-lucide-list-checks' },
  { id: 'channels', label: t('admin.notificationSettings.tabs.channels'), icon: 'i-lucide-radio-tower' }
])
const components: Record<NotificationTab, Component> = {
  policy: SFAdminNotificationPolicyPage,
  channels: SFAdminNotificationChannels
}
const activeComponent = computed(() => components[activeTab.value])
const activeProps = computed(() => activeTab.value === 'channels' ? { canManage: canManage.value } : {})
const toolbarText = computed(() => activeTab.value === 'channels'
  ? t('admin.notificationSettings.channelsAdmin.description')
  : t('admin.notificationSettings.toolbar'))

watch(() => route.query.tab, value => { activeTab.value = normalizeTab(value) })
watch(activeTab, async (tab) => {
  if (route.query.tab === tab) return
  await router.replace({ query: { ...route.query, tab } })
})

function normalizeTab(value: unknown): NotificationTab {
  const raw = Array.isArray(value) ? value[0] : value
  return typeof raw === 'string' && tabIds.includes(raw as NotificationTab) ? raw as NotificationTab : 'policy'
}

function setActiveTab(value: string) {
  activeTab.value = normalizeTab(value)
}

useSeoMeta({ title: t('admin.notificationSettings.metaTitle') })
</script>

<template>
  <div class="mb-4">
    <h2 class="flex items-center gap-2 text-xl font-bold text-slate-900 dark:text-zinc-100">
      <UIcon :name="adminPage.icon" class="size-5 text-[var(--sf-accent)] dark:text-[var(--sf-accent-dark)]" />
      {{ t('admin.notificationSettings.title') }}
    </h2>
  </div>

  <UDashboardToolbar class="mb-6 rounded-lg border border-slate-200 bg-white px-4 py-2.5 text-slate-500 dark:border-zinc-800 dark:bg-zinc-900 dark:text-zinc-400">
    <template #left>
      <div class="flex min-w-0 items-center gap-2 text-sm">
        <UIcon name="i-lucide-bell-ring" class="size-4" />
        <span class="hidden truncate sm:inline">{{ toolbarText }}</span>
      </div>
    </template>
    <template #right>
      <UButton
        color="neutral"
        variant="outline"
        leading-icon="i-lucide-refresh-cw"
        :loading="pageRef?.pending"
        class="border-slate-200 dark:border-zinc-700"
        :aria-label="t('admin.common.refresh')"
        :title="t('admin.common.refresh')"
        @click="pageRef?.refresh?.()"
      >
        <span class="hidden sm:inline">{{ t('admin.common.refresh') }}</span>
      </UButton>
    </template>
  </UDashboardToolbar>

  <div class="flex w-full min-w-0 flex-col gap-4">
    <SFAdminFixedTabNav
      :items="tabs"
      :model-value="activeTab"
      :ariaLabel="t('admin.notificationSettings.tabs.label')"
      @update:model-value="setActiveTab"
    />
    <KeepAlive>
      <component :is="activeComponent" :key="activeTab" v-bind="activeProps" ref="pageRef" />
    </KeepAlive>
  </div>
</template>
