<script setup lang="ts">
import { useAuthSession } from '~/composables/identity/useAuthSession'
import type { Component } from 'vue'
import SFAdminAttachmentManagerTab from '~/components/admin/settings/attachments/tabs/SFAdminAttachmentManagerTab.vue'
import SFAdminAttachmentSettingsTab from '~/components/admin/settings/attachments/tabs/SFAdminAttachmentSettingsTab.vue'
import SFAdminFixedTabNav from '~/components/admin/settings/shared/SFAdminFixedTabNav.vue'
import { useAdminPage } from '~/composables/admin/useAdminPage'

definePageMeta({
  middleware: 'admin',
  layout: 'admin'
})

defineOptions({
  name: 'AdminAttachments'
})

type AttachmentTab = 'settings' | 'manager'
type RefreshableTab = { refresh?: () => Promise<void>, pending?: boolean }

const { t } = useI18n()
const route = useRoute()
const router = useRouter()
const { can } = useAuthSession()
const adminPage = useAdminPage('/attachments')
const childRef = ref<RefreshableTab | null>(null)
const allTabs = [
  { id: 'settings' as const, labelKey: 'admin.attachments.tabs.settings', icon: 'i-lucide-sliders-horizontal', permission: 'attachment.settings.manage' },
  { id: 'manager' as const, labelKey: 'admin.attachments.tabs.manager', icon: 'i-lucide-folder-search', permission: 'attachment.manage' }
]
const tabs = computed(() => allTabs
  .filter(tab => can(tab.permission))
  .map(tab => ({ id: tab.id, label: t(tab.labelKey), icon: tab.icon })))
const activeTab = ref<AttachmentTab>(normalizeTab(route.query.tab))
const components: Record<AttachmentTab, Component> = {
  settings: SFAdminAttachmentSettingsTab,
  manager: SFAdminAttachmentManagerTab
}
const activeComponent = computed(() => components[activeTab.value])
const toolbarPending = computed(() => Boolean(childRef.value?.pending))

watch(tabs, available => {
  if (!available.some(tab => tab.id === activeTab.value) && available[0]) activeTab.value = available[0].id
}, { immediate: true })
watch(() => route.query.tab, value => {
  activeTab.value = normalizeTab(value)
})
watch(activeTab, async tab => {
  if (!tabs.value.some(item => item.id === tab) || route.query.tab === tab) return
  await router.replace({ query: { ...route.query, tab } })
})

useSeoMeta({ title: t('admin.attachments.metaTitle') })

function normalizeTab(value: unknown): AttachmentTab {
  const raw = Array.isArray(value) ? value[0] : value
  const candidate = raw === 'manager' ? 'manager' : 'settings'
  return tabs.value.some(tab => tab.id === candidate) ? candidate : tabs.value[0]?.id || 'settings'
}

function setActiveTab(value: string) {
  activeTab.value = normalizeTab(value)
}

async function refreshActive() {
  await childRef.value?.refresh?.()
}
</script>

<template>
  <div class="flex w-full min-w-0 flex-col gap-4">
    <header>
      <h2 class="flex items-center gap-2 text-xl font-bold text-slate-900 dark:text-zinc-100">
        <UIcon :name="adminPage.icon" class="size-5 text-[var(--sf-accent)] dark:text-[var(--sf-accent-dark)]" />
        {{ t('admin.attachments.title') }}
      </h2>
      <p class="text-sm text-slate-500 dark:text-zinc-400">
        {{ t('admin.attachments.intro') }}
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
        <UButton color="neutral" variant="outline" leading-icon="i-lucide-refresh-cw" :loading="toolbarPending" @click="refreshActive">
          {{ t('admin.attachments.refresh') }}
        </UButton>
      </template>
    </UDashboardToolbar>

    <template v-if="tabs.length">
      <SFAdminFixedTabNav
        :items="tabs"
        :model-value="activeTab"
        :ariaLabel="t('admin.attachments.tabs.label')"
        @update:model-value="setActiveTab"
      />
      <KeepAlive>
        <component :is="activeComponent" :key="activeTab" ref="childRef" />
      </KeepAlive>
    </template>

    <UAlert
      v-else
      color="warning"
      variant="soft"
      icon="i-lucide-lock"
      :title="t('errors.permissionDenied')"
    />
  </div>
</template>
