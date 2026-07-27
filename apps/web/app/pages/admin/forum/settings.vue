<script setup lang="ts">
import type { Component } from 'vue'
import type { ForumCategoryGroup, ForumSettings } from '~/utils/forum/forumTaxonomy'
import SFAdminFixedTabNav from '~/components/admin/settings/shared/SFAdminFixedTabNav.vue'
import SFAdminForumBehaviorTab from '~/components/admin/settings/forum/tabs/SFAdminForumBehaviorTab.vue'
import SFAdminForumCommentsTab from '~/components/admin/settings/forum/tabs/SFAdminForumCommentsTab.vue'
import SFAdminForumGeneralTab from '~/components/admin/settings/forum/tabs/SFAdminForumGeneralTab.vue'
import SFAdminForumReadingTab from '~/components/admin/settings/forum/tabs/SFAdminForumReadingTab.vue'
import SFAdminForumSearchTab from '~/components/admin/settings/forum/tabs/SFAdminForumSearchTab.vue'
import SFAdminForumTagsTab from '~/components/admin/settings/forum/tabs/SFAdminForumTagsTab.vue'
import SFAdminForumTopicsTab from '~/components/admin/settings/forum/tabs/SFAdminForumTopicsTab.vue'
import { useAdminPage } from '~/composables/admin/useAdminPage'
import { createAdminForumApi, createDefaultForumSettings } from '~/utils/admin/adminForum'

definePageMeta({
  middleware: 'admin',
  layout: 'admin'
})

defineOptions({
  name: 'AdminForumSettings'
})

type ForumSettingsTab = 'general' | 'topics' | 'comments' | 'tags' | 'reading' | 'behavior' | 'search'

const { t } = useI18n()
const route = useRoute()
const router = useRouter()
const { request } = useApiClient()
const forumApi = createAdminForumApi(request)
const adminPage = useAdminPage('/forum/settings')
const tabIds: ForumSettingsTab[] = ['general', 'topics', 'comments', 'tags', 'reading', 'behavior', 'search']
const activeTab = ref<ForumSettingsTab>(normalizeTab(route.query.tab))

const tabs = computed(() => tabIds.map(id => ({
  id,
  label: t(`admin.forum.settings.tabs.${id}`),
  icon: tabIcon(id)
})))
const tabComponents: Record<ForumSettingsTab, Component> = {
  general: SFAdminForumGeneralTab,
  topics: SFAdminForumTopicsTab,
  comments: SFAdminForumCommentsTab,
  tags: SFAdminForumTagsTab,
  reading: SFAdminForumReadingTab,
  behavior: SFAdminForumBehaviorTab,
  search: SFAdminForumSearchTab
}
const activeComponent = computed(() => tabComponents[activeTab.value])

const { data, pending, error, refresh } = await useAsyncData('admin-forum-settings', async () => {
  const [groups, settings] = await Promise.all([
    forumApi.listCategoryGroups(),
    forumApi.getSettings()
  ])
  return { groups, settings }
}, {
  default: () => ({ groups: [] as ForumCategoryGroup[], settings: createDefaultForumSettings() })
})
const categories = computed(() => data.value.groups.flatMap(group => group.categories || []))

watch(() => route.query.tab, value => {
  const normalized = normalizeTab(value)
  if (normalized !== activeTab.value) activeTab.value = normalized
})

watch(activeTab, async tab => {
  if (route.query.tab === tab) return
  await router.replace({ query: { ...route.query, tab } })
})

useSeoMeta({
  title: t('admin.forum.settings.metaTitle')
})

function applySettings(settings: ForumSettings) {
  data.value = { ...data.value, settings }
}

function setActiveTab(value: string) {
  activeTab.value = normalizeTab(value)
}

function normalizeTab(value: unknown): ForumSettingsTab {
  const raw = Array.isArray(value) ? value[0] : value
  return typeof raw === 'string' && tabIds.includes(raw as ForumSettingsTab) ? raw as ForumSettingsTab : 'general'
}

function tabIcon(id: ForumSettingsTab) {
  return {
    general: 'i-lucide-sliders-horizontal',
    topics: 'i-lucide-file-pen-line',
    comments: 'i-lucide-messages-square',
    tags: 'i-lucide-tags',
    reading: 'i-lucide-book-open-text',
    behavior: 'i-lucide-shield-check',
    search: 'i-lucide-search'
  }[id]
}
</script>

<template>
  <div class="mb-4">
    <h2 class="flex items-center gap-2 text-xl font-bold text-slate-900 dark:text-zinc-100">
      <UIcon :name="adminPage.icon" class="size-5 text-[var(--sf-accent)] dark:text-[var(--sf-accent-dark)]" />
      {{ t('admin.forum.settings.title') }}
    </h2>
  </div>

  <UDashboardToolbar class="mb-6 rounded-lg border border-slate-200 bg-white px-4 py-2.5 text-slate-500 dark:border-zinc-800 dark:bg-zinc-900 dark:text-zinc-400">
    <template #left>
      <div class="flex min-w-0 items-center gap-2 text-sm"><UIcon name="i-lucide-sliders-horizontal" class="size-4" /><span class="truncate">{{ t('admin.forum.settings.intro') }}</span></div>
    </template>
    <template #right>
      <UButton color="neutral" variant="outline" leading-icon="i-lucide-refresh-cw" :loading="pending" @click="refresh()">{{ t('admin.common.refresh') }}</UButton>
    </template>
  </UDashboardToolbar>

  <div class="flex w-full min-w-0 flex-col gap-4">
    <UAlert v-if="error" color="error" variant="soft" icon="i-lucide-triangle-alert" :title="t('admin.forum.settings.loadFailed')" />
    <UAlert color="primary" variant="soft" icon="i-lucide-sparkles" :title="t('admin.forum.settings.recommendedTitle')" :description="t('admin.forum.settings.recommendedDescription')" />
    <SFAdminFixedTabNav :items="tabs" :model-value="activeTab" :ariaLabel="t('admin.forum.settings.tabs.label')" @update:model-value="setActiveTab" />
    <KeepAlive>
      <component
        :is="activeComponent"
        :key="activeTab"
        :settings="data.settings"
        :categories="categories"
        :pending="pending"
        @saved="applySettings"
        @refresh="refresh"
      />
    </KeepAlive>
  </div>
</template>
