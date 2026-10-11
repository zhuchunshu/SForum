<script setup lang="ts">
import type { Component } from 'vue'
import SFAdminFixedTabNav from '~/components/admin/settings/shared/SFAdminFixedTabNav.vue'
import SFAdminGamificationBackfillTab from '~/components/admin/gamification/tabs/SFAdminGamificationBackfillTab.vue'
import SFAdminGamificationBadgesTab from '~/components/admin/gamification/tabs/SFAdminGamificationBadgesTab.vue'
import SFAdminGamificationGrantsTab from '~/components/admin/gamification/tabs/SFAdminGamificationGrantsTab.vue'
import SFAdminGamificationLevelsTab from '~/components/admin/gamification/tabs/SFAdminGamificationLevelsTab.vue'
import { ADMIN_GAMIFICATION_KEY, GAMIFICATION_EXTENSION_ID, useAdminGamification } from '~/composables/admin/useAdminGamification'
import { useAdminPage } from '~/composables/admin/useAdminPage'

definePageMeta({
  middleware: 'admin',
  layout: 'admin'
})

defineOptions({
  name: 'AdminGamification'
})

type GamificationTab = 'badges' | 'levels' | 'grants' | 'backfill'

const route = useRoute()
const router = useRouter()
const { t } = useI18n()
const adminPage = useAdminPage('/gamification')
// 页面壳是唯一的数据所有者：页签通过注入拿到同一份上下文。
const gamification = useAdminGamification()
provide(ADMIN_GAMIFICATION_KEY, gamification)

const validTabs: GamificationTab[] = ['badges', 'levels', 'grants', 'backfill']
const activeTab = ref<GamificationTab>(normalizeTab(route.query.tab))

const tabs = computed(() => [
  { id: 'badges', label: t('admin.gamification.tabs.badges'), icon: 'i-lucide-award' },
  { id: 'levels', label: t('admin.gamification.tabs.levels'), icon: 'i-lucide-trending-up' },
  { id: 'grants', label: t('admin.gamification.tabs.grants'), icon: 'i-lucide-gift' },
  { id: 'backfill', label: t('admin.gamification.tabs.backfill'), icon: 'i-lucide-history' }
])
const tabComponents: Record<GamificationTab, Component> = {
  badges: SFAdminGamificationBadgesTab,
  levels: SFAdminGamificationLevelsTab,
  grants: SFAdminGamificationGrantsTab,
  backfill: SFAdminGamificationBackfillTab
}
const activeComponent = computed(() => tabComponents[activeTab.value])

watch(() => route.query.tab, (tab) => {
  const normalized = normalizeTab(tab)
  if (normalized !== activeTab.value) activeTab.value = normalized
})

// replace 只改地址栏，不触发完整导航（避免页签切换白屏）。
watch(activeTab, async (tab) => {
  if (route.query.tab === tab) return
  await router.replace({ query: { ...route.query, tab } })
})

useSeoMeta({
  title: t('admin.gamification.metaTitle')
})

function normalizeTab(value: unknown): GamificationTab {
  const candidate = Array.isArray(value) ? value[0] : value
  return typeof candidate === 'string' && validTabs.includes(candidate as GamificationTab)
    ? candidate as GamificationTab
    : 'badges'
}

function setActiveTab(value: string) {
  activeTab.value = normalizeTab(value)
}
</script>

<template>
  <div class="mb-4">
    <h2 class="flex items-center gap-2 text-xl font-bold text-slate-900 dark:text-zinc-100">
      <UIcon :name="adminPage.icon" class="size-5 text-[var(--sf-accent)] dark:text-[var(--sf-accent-dark)]" />
      {{ t('admin.gamification.title') }}
    </h2>
  </div>

  <UDashboardToolbar class="mb-6 rounded-lg border border-slate-200 bg-white px-4 py-2.5 text-slate-500 dark:border-zinc-800 dark:bg-zinc-900 dark:text-zinc-400">
    <template #left>
      <div class="flex min-w-0 items-center gap-2 text-sm">
        <UIcon name="i-lucide-award" class="size-4" />
        <span class="truncate">{{ t('admin.gamification.intro') }}</span>
      </div>
    </template>
    <template #right>
      <UButton
        color="neutral"
        variant="outline"
        leading-icon="i-lucide-refresh-cw"
        :loading="gamification.pending.value"
        @click="gamification.refreshAll()"
      >
        {{ t('admin.settings.refresh') }}
      </UButton>
    </template>
  </UDashboardToolbar>

  <div class="flex min-w-0 flex-col gap-4">
    <!-- 插件未启用：管理路由不存在，写操作必然失败，这里失败关闭并给出口。 -->
    <UAlert
      v-if="gamification.pluginStatusResolved.value && !gamification.pluginEnabled.value"
      color="warning"
      variant="soft"
      icon="i-lucide-plug"
      :title="t('admin.gamification.pluginDisabledTitle')"
      :description="t('admin.gamification.pluginDisabled')"
    >
      <template #actions>
        <UButton
          size="xs"
          color="neutral"
          variant="outline"
          leading-icon="i-lucide-puzzle"
          :to="`/admin/extensions?focus=${GAMIFICATION_EXTENSION_ID}`"
        >
          {{ t('admin.gamification.openExtensions') }}
        </UButton>
      </template>
    </UAlert>

    <UAlert
      v-if="gamification.error.value"
      color="error"
      variant="soft"
      icon="i-lucide-triangle-alert"
      :title="t('admin.gamification.loadFailed')"
      :description="(gamification.error.value as Error)?.message"
    />

    <SFAdminFixedTabNav
      :items="tabs"
      :model-value="activeTab"
      :ariaLabel="t('admin.gamification.tabs.label')"
      @update:model-value="setActiveTab"
    />
    <KeepAlive>
      <component :is="activeComponent" :key="activeTab" />
    </KeepAlive>
  </div>
</template>
