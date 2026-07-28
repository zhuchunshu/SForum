<script setup lang="ts">
import { useAdminPage } from '~/composables/admin/useAdminPage'
import type { Component } from 'vue'
import SFAdminFixedTabNav from '~/components/admin/settings/shared/SFAdminFixedTabNav.vue'
import SFAdminMailDeliveriesTab from '~/components/admin/settings/mail/tabs/SFAdminMailDeliveriesTab.vue'
import SFAdminMailOverviewTab from '~/components/admin/settings/mail/tabs/SFAdminMailOverviewTab.vue'
import SFAdminMailProviderTab from '~/components/admin/settings/mail/tabs/SFAdminMailProviderTab.vue'

definePageMeta({ middleware: 'admin', layout: 'admin' })
defineOptions({ name: 'AdminMailSettings' })

type MailTab = 'overview' | 'provider' | 'deliveries'
type RefreshableTab = { refresh?: () => Promise<void>, pending?: boolean }
const { t } = useI18n()
const route = useRoute()
const router = useRouter()
const adminPage = useAdminPage('/settings/mail')
const childRef = ref<RefreshableTab | null>(null)
const tabIds: MailTab[] = ['overview', 'provider', 'deliveries']
const activeTab = ref<MailTab>(normalizeTab(route.query.tab))
const tabs = computed(() => [
  { id: 'overview', label: t('admin.mailSettings.overview'), icon: 'i-lucide-layout-dashboard' },
  { id: 'provider', label: t('admin.mailSettings.mail'), icon: 'i-lucide-mail' },
  { id: 'deliveries', label: t('admin.mailSettings.deliveries'), icon: 'i-lucide-list-checks' }
])
const components: Record<MailTab, Component> = {
  overview: SFAdminMailOverviewTab,
  provider: SFAdminMailProviderTab,
  deliveries: SFAdminMailDeliveriesTab
}
const activeComponent = computed(() => components[activeTab.value])

useSeoMeta({ title: t('admin.mailSettings.title') })

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
  <div class="mb-4">
    <h2 class="flex items-center gap-2 text-xl font-bold text-slate-900 dark:text-zinc-100">
      <UIcon :name="adminPage.icon" class="size-5 text-[var(--sf-accent)] dark:text-[var(--sf-accent-dark)]" />
      {{ t('admin.mailSettings.title') }}
    </h2>
  </div>

  <UDashboardToolbar class="mb-6 rounded-lg border border-slate-200 bg-white px-4 py-2.5 text-slate-500 dark:border-zinc-800 dark:bg-zinc-900 dark:text-zinc-400">
    <template #left>
      <div class="flex min-w-0 items-center gap-2 text-sm">
        <UIcon name="i-lucide-mail" class="size-4" />
        <span class="hidden truncate sm:inline">{{ t('admin.mailSettings.description') }}</span>
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
    <SFAdminFixedTabNav :items="tabs" :model-value="activeTab" :ariaLabel="t('admin.mailSettings.title')" @update:model-value="setActiveTab" />
    <KeepAlive><component :is="activeComponent" :key="activeTab" ref="childRef" /></KeepAlive>
  </div>
</template>
