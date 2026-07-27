<script setup lang="ts">
import { useAdminPage } from '~/composables/admin/useAdminPage'
import type { Component } from 'vue'
import SFAdminFixedTabNav from '~/components/admin/settings/shared/SFAdminFixedTabNav.vue'
import SFAdminMailDeliveriesTab from '~/components/admin/settings/mail/tabs/SFAdminMailDeliveriesTab.vue'
import SFAdminMailNotificationsTab from '~/components/admin/settings/mail/tabs/SFAdminMailNotificationsTab.vue'
import SFAdminMailOverviewTab from '~/components/admin/settings/mail/tabs/SFAdminMailOverviewTab.vue'
import SFAdminMailProviderTab from '~/components/admin/settings/mail/tabs/SFAdminMailProviderTab.vue'

definePageMeta({ middleware: 'admin', layout: 'admin' })
defineOptions({ name: 'AdminMailSettings' })

type MailTab = 'overview' | 'provider' | 'notifications' | 'deliveries'
type RefreshableTab = { refresh?: () => Promise<void>, pending?: boolean }
const { t } = useI18n()
const route = useRoute()
const router = useRouter()
const adminPage = useAdminPage('/settings/mail')
const childRef = ref<RefreshableTab | null>(null)
const tabIds: MailTab[] = ['overview', 'provider', 'notifications', 'deliveries']
const activeTab = ref<MailTab>(normalizeTab(route.query.tab))
const tabs = computed(() => [
  { id: 'overview', label: t('admin.mailSettings.overview'), icon: 'i-lucide-layout-dashboard' },
  { id: 'provider', label: t('admin.mailSettings.mail'), icon: 'i-lucide-mail' },
  { id: 'notifications', label: t('admin.mailSettings.inApp'), icon: 'i-lucide-bell' },
  { id: 'deliveries', label: t('admin.mailSettings.deliveries'), icon: 'i-lucide-list-checks' }
])
const components: Record<MailTab, Component> = {
  overview: SFAdminMailOverviewTab,
  provider: SFAdminMailProviderTab,
  notifications: SFAdminMailNotificationsTab,
  deliveries: SFAdminMailDeliveriesTab
}
const activeComponent = computed(() => components[activeTab.value])

watch(() => route.query.tab, value => { activeTab.value = normalizeTab(value) })
watch(activeTab, async tab => {
  if (route.query.tab === tab) return
  await router.replace({ query: { ...route.query, tab } })
})

function normalizeTab(value: unknown): MailTab {
  const raw = Array.isArray(value) ? value[0] : value
  return typeof raw === 'string' && tabIds.includes(raw as MailTab) ? raw as MailTab : 'overview'
}
function setActiveTab(value: string) { activeTab.value = normalizeTab(value) }
</script>

<template>
  <div class="space-y-5">
    <header><h1 class="flex items-center gap-2 font-bold"><UIcon :name="adminPage.icon" class="size-5 text-[var(--sf-accent)]" />{{ t('admin.mailSettings.title') }}</h1><p class="mt-1 text-sm text-muted">{{ t('admin.mailSettings.description') }}</p></header>
    <UDashboardToolbar class="rounded-lg border border-slate-200 bg-white px-4 py-2.5 text-slate-500 dark:border-zinc-800 dark:bg-zinc-900">
      <template #left><div class="flex items-center gap-2 text-sm"><UIcon name="i-lucide-mail" class="size-4" /><span>{{ t('admin.mailSettings.description') }}</span></div></template>
      <template #right><UButton icon="i-lucide-rotate-cw" color="neutral" variant="outline" :loading="childRef?.pending" @click="childRef?.refresh?.()">{{ t('admin.home.refresh') }}</UButton></template>
    </UDashboardToolbar>
    <section class="rounded-md border border-teal-200 bg-teal-50/80 p-4 dark:border-teal-900/60 dark:bg-teal-950/30"><div class="flex gap-3"><UIcon name="i-lucide-sparkles" class="size-5 text-teal-700" /><div><h2 class="font-bold">{{ t('admin.mailSettings.recommendedTitle') }}</h2><p class="mt-1 text-sm">{{ t('admin.mailSettings.recommendedDescription') }}</p></div></div></section>
    <SFAdminFixedTabNav :items="tabs" :model-value="activeTab" :ariaLabel="t('admin.mailSettings.title')" @update:model-value="setActiveTab" />
    <KeepAlive><component :is="activeComponent" :key="activeTab" ref="childRef" /></KeepAlive>
  </div>
</template>
