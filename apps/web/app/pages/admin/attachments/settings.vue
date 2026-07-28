<script setup lang="ts">
import type { Component } from 'vue'
import SFAdminAttachmentCompressionTab from '~/components/admin/settings/attachments/tabs/SFAdminAttachmentCompressionTab.vue'
import SFAdminAttachmentSettingsTab from '~/components/admin/settings/attachments/tabs/SFAdminAttachmentSettingsTab.vue'
import SFAdminFixedTabNav from '~/components/admin/settings/shared/SFAdminFixedTabNav.vue'
import { useAdminPage } from '~/composables/admin/useAdminPage'

definePageMeta({ middleware: 'admin', layout: 'admin' })
defineOptions({ name: 'AdminAttachmentSettings' })

type RefreshablePage = { refresh?: () => Promise<void>, pending?: boolean }
type AttachmentConfigurationTab = 'basic' | 'compression'

const { t } = useI18n()
const route = useRoute()
const router = useRouter()
const adminPage = useAdminPage('/attachments/settings')
const pageRef = ref<RefreshablePage | null>(null)
const tabs = computed(() => [
  { id: 'basic', label: t('admin.attachments.tabs.settings'), icon: 'i-lucide-sliders-horizontal' },
  { id: 'compression', label: t('admin.attachments.tabs.compression'), icon: 'i-lucide-file-archive' }
])
const activeTab = ref<AttachmentConfigurationTab>(normalizeTab(route.query.tab))
const components: Record<AttachmentConfigurationTab, Component> = {
  basic: SFAdminAttachmentSettingsTab,
  compression: SFAdminAttachmentCompressionTab
}
const activeComponent = computed(() => components[activeTab.value])
const toolbarPending = computed(() => Boolean(pageRef.value?.pending))
const canRefresh = computed(() => Boolean(pageRef.value?.refresh))
const toolbarText = computed(() => activeTab.value === 'compression'
  ? t('admin.attachments.compression.emptyDescription')
  : t('admin.attachments.toolbar'))

watch(() => route.query.tab, value => {
  activeTab.value = normalizeTab(value)
})
watch(activeTab, async tab => {
  if (route.query.tab === tab) return
  await router.replace({ query: { ...route.query, tab } })
})

useSeoMeta({ title: t('admin.attachments.configuration') })

function normalizeTab(value: unknown): AttachmentConfigurationTab {
  const raw = Array.isArray(value) ? value[0] : value
  return raw === 'compression' ? 'compression' : 'basic'
}

function setActiveTab(value: string) {
  activeTab.value = normalizeTab(value)
}

async function refreshPage() {
  await pageRef.value?.refresh?.()
}
</script>

<template>
  <div class="flex w-full min-w-0 flex-col gap-4">
    <header>
      <h2 class="flex items-center gap-2 text-xl font-bold text-slate-900 dark:text-zinc-100">
        <UIcon :name="adminPage.icon" class="size-5 text-[var(--sf-accent)] dark:text-[var(--sf-accent-dark)]" />
        {{ t('admin.attachments.configuration') }}
      </h2>
      <p class="text-sm text-slate-500 dark:text-zinc-400">
        {{ t('admin.attachments.configurationDescription') }}
      </p>
    </header>

    <UDashboardToolbar class="overflow-x-hidden rounded-lg border border-slate-200 bg-white px-4 py-2.5 text-slate-500 dark:border-zinc-800 dark:bg-zinc-900 dark:text-zinc-400">
      <template #left>
        <div class="flex min-w-0 items-center gap-2 text-sm">
          <UIcon name="i-lucide-paperclip" class="size-4" />
          <span class="truncate">{{ toolbarText }}</span>
        </div>
      </template>
      <template #right>
        <UButton v-if="canRefresh" color="neutral" variant="outline" leading-icon="i-lucide-refresh-cw" :loading="toolbarPending" @click="refreshPage">
          {{ t('admin.attachments.refresh') }}
        </UButton>
      </template>
    </UDashboardToolbar>

    <SFAdminFixedTabNav
      :items="tabs"
      :model-value="activeTab"
      :ariaLabel="t('admin.attachments.configurationTabsLabel')"
      @update:model-value="setActiveTab"
    />

    <KeepAlive>
      <component :is="activeComponent" :key="activeTab" ref="pageRef" />
    </KeepAlive>
  </div>
</template>
