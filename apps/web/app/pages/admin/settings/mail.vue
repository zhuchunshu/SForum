<script setup lang="ts">
import { useAdminPage } from '~/composables/admin/useAdminPage'
import type { Component } from 'vue'
import SFAdminFixedTabNav from '~/components/admin/settings/shared/SFAdminFixedTabNav.vue'
import SFAdminMailDeliveriesTab from '~/components/admin/settings/mail/tabs/SFAdminMailDeliveriesTab.vue'
import SFAdminMailOverviewTab from '~/components/admin/settings/mail/tabs/SFAdminMailOverviewTab.vue'
import SFAdminMailProviderTab from '~/components/admin/settings/mail/tabs/SFAdminMailProviderTab.vue'
import SFAdminNotificationChannels from '~/components/admin/settings/notifications/SFAdminNotificationChannels.vue'
import SFAdminNotificationPolicyPage from '~/components/admin/settings/notifications/SFAdminNotificationPolicyPage.vue'
import { usePermissions } from '~/composables/identity/usePermissions'

definePageMeta({ middleware: 'admin', layout: 'admin' })
defineOptions({ name: 'AdminMailSettings' })

type MailTab = 'overview' | 'provider' | 'deliveries' | 'policy' | 'channels'
type RefreshableTab = { refresh?: () => Promise<void>, pending?: boolean }
const { t } = useI18n()
const route = useRoute()
const router = useRouter()
const adminPage = useAdminPage('/settings/mail')
const { can } = usePermissions()
const canManageMail = computed(() => can('settings.mail.manage'))
const canManageNotifications = computed(() => can('settings.notifications.manage'))
const childRef = ref<RefreshableTab | null>(null)
const tabs = computed(() => [
  ...(canManageMail.value
    ? [
        { id: 'overview' as const, label: t('admin.mailSettings.overview'), icon: 'i-lucide-layout-dashboard' },
        { id: 'provider' as const, label: t('admin.mailSettings.mail'), icon: 'i-lucide-mail' },
        { id: 'deliveries' as const, label: t('admin.mailSettings.deliveries'), icon: 'i-lucide-list-checks' }
      ]
    : []),
  ...(canManageNotifications.value
    ? [
        { id: 'policy' as const, label: t('admin.notificationSettings.tabs.policy'), icon: 'i-lucide-list-checks' },
        { id: 'channels' as const, label: t('admin.notificationSettings.tabs.channels'), icon: 'i-lucide-radio-tower' }
      ]
    : [])
])
const tabIds = computed(() => tabs.value.map(tab => tab.id))
const activeTab = ref<MailTab>(normalizeTab(route.query.tab))
const components: Record<MailTab, Component> = {
  overview: SFAdminMailOverviewTab,
  provider: SFAdminMailProviderTab,
  deliveries: SFAdminMailDeliveriesTab,
  policy: SFAdminNotificationPolicyPage,
  channels: SFAdminNotificationChannels
}
const activeComponent = computed(() => components[activeTab.value])
const activeProps = computed(() => activeTab.value === 'channels' ? { canManage: canManageNotifications.value } : {})
const toolbarText = computed(() => {
  if (activeTab.value === 'channels') return t('admin.notificationSettings.channelsAdmin.description')
  if (activeTab.value === 'policy') return t('admin.notificationSettings.toolbar')
  return t('admin.mailSettings.description')
})
const toolbarIcon = computed(() => activeTab.value === 'policy' || activeTab.value === 'channels'
  ? 'i-lucide-bell-ring'
  : 'i-lucide-mail')

useSeoMeta({ title: t('admin.mailSettings.title') })

watch([() => route.query.tab, tabIds], ([value]) => {
  activeTab.value = normalizeTab(value)
})
watch(activeTab, async tab => {
  if (route.query.tab === tab) return
  await router.replace({ query: { ...route.query, tab } })
})

function normalizeTab(value: unknown): MailTab {
  const raw = Array.isArray(value) ? value[0] : value
  return typeof raw === 'string' && tabIds.value.includes(raw as MailTab)
    ? raw as MailTab
    : tabIds.value[0] || 'overview'
}
function setActiveTab(value: string) { activeTab.value = normalizeTab(value) }
</script>

<template>
  <div class="mb-4">
    <h2 class="flex items-center gap-2 text-xl font-bold text-slate-900 dark:text-zinc-100">
      <UIcon :name="adminPage.icon" class="size-5 text-[var(--sf-accent)] dark:text-[var(--sf-accent-dark)]" />
      {{ t('admin.mailSettings.title') }}
    </h2>
  </div>

  <UDashboardToolbar class="mb-6 rounded-lg border border-slate-200 bg-white px-4 py-2.5 text-slate-500 dark:border-zinc-800 dark:bg-zinc-900 dark:text-zinc-400">
    <template #left>
      <div class="flex min-w-0 items-center gap-2 text-sm">
        <UIcon :name="toolbarIcon" class="size-4" />
        <span class="hidden truncate sm:inline">{{ toolbarText }}</span>
      </div>
    </template>
    <template #right>
      <UButton
        leading-icon="i-lucide-refresh-cw"
        color="neutral"
        variant="outline"
        :loading="childRef?.pending"
        class="border-slate-200 dark:border-zinc-700"
        :aria-label="t('admin.home.refresh')"
        :title="t('admin.home.refresh')"
        @click="childRef?.refresh?.()"
      >
        <span class="hidden sm:inline">{{ t('admin.home.refresh') }}</span>
      </UButton>
    </template>
  </UDashboardToolbar>

  <div class="flex w-full min-w-0 flex-col gap-4">
    <template v-if="tabs.length">
      <SFAdminFixedTabNav :items="tabs" :model-value="activeTab" :ariaLabel="t('admin.mailSettings.title')" @update:model-value="setActiveTab" />
      <KeepAlive><component :is="activeComponent" :key="activeTab" v-bind="activeProps" ref="childRef" /></KeepAlive>
    </template>
    <UAlert v-else color="error" variant="soft" icon="i-lucide-shield-alert" :title="t('errors.permissionDenied')" />
  </div>
</template>
