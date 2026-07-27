<script setup lang="ts">
import type { Component } from 'vue'
import type { AdminWebOption } from '~/composables/useWebOptions'
import SFAdminFixedTabNav from '~/components/admin/settings/shared/SFAdminFixedTabNav.vue'
import SFAdminSeoContentTab from '~/components/admin/settings/seo/tabs/SFAdminSeoContentTab.vue'
import SFAdminSeoMetaTab from '~/components/admin/settings/seo/tabs/SFAdminSeoMetaTab.vue'
import SFAdminSeoOverviewTab from '~/components/admin/settings/seo/tabs/SFAdminSeoOverviewTab.vue'
import SFAdminSeoPermalinksTab from '~/components/admin/settings/seo/tabs/SFAdminSeoPermalinksTab.vue'
import SFAdminSeoRobotsTab from '~/components/admin/settings/seo/tabs/SFAdminSeoRobotsTab.vue'
import SFAdminSeoSchemaTab from '~/components/admin/settings/seo/tabs/SFAdminSeoSchemaTab.vue'
import SFAdminSeoSearchTab from '~/components/admin/settings/seo/tabs/SFAdminSeoSearchTab.vue'
import SFAdminSeoSitemapTab from '~/components/admin/settings/seo/tabs/SFAdminSeoSitemapTab.vue'
import SFAdminSeoVerificationTab from '~/components/admin/settings/seo/tabs/SFAdminSeoVerificationTab.vue'
import { useAdminPage } from '~/composables/admin/useAdminPage'

definePageMeta({
  middleware: 'admin',
  layout: 'admin'
})

defineOptions({
  name: 'AdminSeo'
})

type SeoTab = 'overview' | 'search' | 'content' | 'meta' | 'robots' | 'sitemap' | 'schema' | 'verification' | 'permalinks'

const { t } = useI18n()
const route = useRoute()
const router = useRouter()
const { options, siteName, siteUrl, fetchAdminEnvelope } = useWebOptions()
const adminPage = useAdminPage('/seo')
const tabIds: SeoTab[] = ['overview', 'search', 'content', 'meta', 'robots', 'sitemap', 'schema', 'verification', 'permalinks']
const activeTab = ref<SeoTab>(normalizeTab(route.query.tab))
const tabs = computed(() => tabIds.map(id => ({ id, label: t(`admin.seo.tabs.${id}`), icon: tabIcon(id) })))
const tabComponents: Record<SeoTab, Component> = {
  overview: SFAdminSeoOverviewTab,
  search: SFAdminSeoSearchTab,
  content: SFAdminSeoContentTab,
  meta: SFAdminSeoMetaTab,
  robots: SFAdminSeoRobotsTab,
  sitemap: SFAdminSeoSitemapTab,
  schema: SFAdminSeoSchemaTab,
  verification: SFAdminSeoVerificationTab,
  permalinks: SFAdminSeoPermalinksTab
}
const activeComponent = computed(() => tabComponents[activeTab.value])

const { data: adminSeoItems, pending, error, refresh } = await useAsyncData<AdminWebOption[]>(
  'admin-seo-options',
  async () => (await fetchAdminEnvelope()).data
)

watch(() => route.query.tab, value => {
  const normalized = normalizeTab(value)
  if (normalized !== activeTab.value) activeTab.value = normalized
})

watch(activeTab, async tab => {
  if (route.query.tab === tab) return
  await router.replace({ query: { ...route.query, tab } })
})

useSeoMeta({
  title: t('admin.seo.metaTitle')
})

function applySavedOptions(items: AdminWebOption[]) {
  adminSeoItems.value = items
  const publicOptions = items.filter(item => item.public && !item.secret)
  options.value = { ...options.value, ...Object.fromEntries(publicOptions.map(item => [item.name, item.value])) }
}

function setActiveTab(value: string) {
  activeTab.value = normalizeTab(value)
}

function normalizeTab(value: unknown): SeoTab {
  const raw = Array.isArray(value) ? value[0] : value
  return typeof raw === 'string' && tabIds.includes(raw as SeoTab) ? raw as SeoTab : 'overview'
}

function tabIcon(id: SeoTab) {
  return {
    overview: 'i-lucide-gauge',
    search: 'i-lucide-search',
    content: 'i-lucide-files',
    meta: 'i-lucide-file-text',
    robots: 'i-lucide-bot',
    sitemap: 'i-lucide-map',
    schema: 'i-lucide-braces',
    verification: 'i-lucide-badge-check',
    permalinks: 'i-lucide-link'
  }[id]
}
</script>

<template>
  <div class="mb-4 flex flex-col gap-1">
    <h2 class="flex items-center gap-2 text-xl font-bold text-slate-900 dark:text-zinc-100"><UIcon :name="adminPage.icon" class="size-5 text-[var(--sf-accent)]" />{{ t('admin.seo.title') }}</h2>
    <p class="text-sm text-muted">{{ t('admin.seo.intro') }}</p>
  </div>
  <UDashboardToolbar class="mb-6 rounded-lg border border-slate-200 bg-white px-4 py-2.5 text-slate-500 dark:border-zinc-800 dark:bg-zinc-900">
    <template #left><div class="flex items-center gap-2 text-sm"><UIcon name="i-lucide-search-check" class="size-4" /><span>{{ t('admin.seo.toolbar') }}</span></div></template>
    <template #right><UButton color="neutral" variant="outline" leading-icon="i-lucide-refresh-cw" :loading="pending" @click="refresh()">{{ t('admin.seo.refresh') }}</UButton></template>
  </UDashboardToolbar>
  <div class="flex flex-col gap-4">
    <UAlert v-if="error" color="error" variant="soft" icon="i-lucide-triangle-alert" :title="t('admin.seo.loadFailed')" />
    <SFAdminFixedTabNav :items="tabs" :model-value="activeTab" :ariaLabel="t('admin.seo.tabs.label')" @update:model-value="setActiveTab" />
    <KeepAlive>
      <component
        :is="activeComponent"
        :key="activeTab"
        :items="adminSeoItems || []"
        :product-site-name="siteName"
        :site-url="siteUrl"
        @saved="applySavedOptions"
      />
    </KeepAlive>
  </div>
</template>
