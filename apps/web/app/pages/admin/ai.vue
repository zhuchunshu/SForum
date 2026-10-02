<script setup lang="ts">
import type { Component } from 'vue'
import SFAdminAIProvidersTab from '~/components/admin/ai/tabs/SFAdminAIProvidersTab.vue'
import SFAdminAIDiagnoseTab from '~/components/admin/ai/tabs/SFAdminAIDiagnoseTab.vue'
import SFAdminAIObservabilityTab from '~/components/admin/ai/tabs/SFAdminAIObservabilityTab.vue'
import SFAdminFixedTabNav from '~/components/admin/settings/shared/SFAdminFixedTabNav.vue'
import { ADMIN_AI_KEY, useAdminAI } from '~/composables/admin/useAdminAI'
import { useAdminPage } from '~/composables/admin/useAdminPage'

definePageMeta({
  middleware: 'admin',
  layout: 'admin'
})

defineOptions({
  name: 'AdminAI'
})

type AITab = 'providers' | 'diagnose' | 'observability'

const route = useRoute()
const { t } = useI18n()
const adminPage = useAdminPage('/ai')
// 页面壳是唯一的数据所有者：页签通过注入拿到同一份上下文，避免同一个
// useAsyncData key 被多个组件持有而在卸载时被清理。
const ai = useAdminAI()
provide(ADMIN_AI_KEY, ai)

const validTabs: AITab[] = ['providers', 'diagnose', 'observability']
const activeTab = ref<AITab>(normalizeTab(route.query.tab))

const tabs = computed(() => [
  { id: 'providers', label: t('admin.ai.tabs.providers'), icon: 'i-lucide-server' },
  { id: 'diagnose', label: t('admin.ai.tabs.diagnose'), icon: 'i-lucide-stethoscope' },
  { id: 'observability', label: t('admin.ai.tabs.observability'), icon: 'i-lucide-activity' }
])

const tabComponents: Record<AITab, Component> = {
  providers: SFAdminAIProvidersTab,
  diagnose: SFAdminAIDiagnoseTab,
  observability: SFAdminAIObservabilityTab
}

const activeComponent = computed(() => tabComponents[activeTab.value])

function normalizeTab(value: unknown): AITab {
  const candidate = Array.isArray(value) ? value[0] : value
  return validTabs.includes(candidate as AITab) ? (candidate as AITab) : 'providers'
}

// 只切换本地状态。把页签写回 query 会触发完整路由导航（page:loading →
// Suspense 重新挂载 → 重新 hydration），那是页签切换白屏的直接原因。
function setActiveTab(value: string) {
  activeTab.value = normalizeTab(value)
}
</script>

<template>
  <div class="mb-4">
    <h2 class="flex items-center gap-2 text-xl font-bold text-slate-900 dark:text-zinc-100">
      <UIcon :name="adminPage.icon" class="size-5 text-[var(--sf-accent)] dark:text-[var(--sf-accent-dark)]" />
      {{ t('admin.ai.title') }}
    </h2>
  </div>

  <UDashboardToolbar class="mb-6 rounded-lg border border-slate-200 bg-white px-4 py-2.5 text-slate-500 dark:border-zinc-800 dark:bg-zinc-900 dark:text-zinc-400">
    <template #left>
      <div class="flex min-w-0 items-center gap-2 text-sm">
        <UIcon name="i-lucide-sparkles" class="size-4" />
        <span class="truncate">{{ t('admin.ai.intro') }}</span>
      </div>
    </template>
    <template #right>
      <UButton
        color="neutral"
        variant="outline"
        leading-icon="i-lucide-refresh-cw"
        :loading="ai.loading.value"
        @click="ai.refresh()"
      >
        {{ t('admin.ai.refresh') }}
      </UButton>
    </template>
  </UDashboardToolbar>

  <div class="flex min-w-0 flex-col gap-4">
    <UAlert
      v-if="ai.loadFailed.value"
      color="error"
      variant="soft"
      icon="i-lucide-triangle-alert"
      :title="t('admin.ai.loadFailed')"
    />
    <SFAdminFixedTabNav
      :items="tabs"
      :model-value="activeTab"
      :ariaLabel="t('admin.ai.tabs.label')"
      @update:model-value="setActiveTab"
    />
    <KeepAlive>
      <component :is="activeComponent" :key="activeTab" />
    </KeepAlive>
  </div>
</template>
