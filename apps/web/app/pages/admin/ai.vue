<script setup lang="ts">
import type { Component } from 'vue'
import SFAdminAIProvidersTab from '~/components/admin/ai/tabs/SFAdminAIProvidersTab.vue'
import SFAdminAIReplyPromptTab from '~/components/admin/ai/tabs/SFAdminAIReplyPromptTab.vue'
import SFAdminAIPurposesTab from '~/components/admin/ai/tabs/SFAdminAIPurposesTab.vue'
import SFAdminAIToolsTab from '~/components/admin/ai/tabs/SFAdminAIToolsTab.vue'
import SFAdminAIDiagnoseTab from '~/components/admin/ai/tabs/SFAdminAIDiagnoseTab.vue'
import SFAdminAIObservabilityTab from '~/components/admin/ai/tabs/SFAdminAIObservabilityTab.vue'
import SFAdminFixedTabNav from '~/components/admin/settings/shared/SFAdminFixedTabNav.vue'
import { ADMIN_AI_KEY, useAdminAI } from '~/composables/admin/useAdminAI'
import { ADMIN_AI_DRAFT_KEY, useAdminAISettingsForm } from '~/composables/admin/useAdminAISettingsForm'
import { useAdminPage } from '~/composables/admin/useAdminPage'

definePageMeta({
  middleware: 'admin',
  layout: 'admin'
})

defineOptions({
  name: 'AdminAI'
})

type AITab = 'providers' | 'replyPrompt' | 'purposes' | 'tools' | 'diagnose' | 'observability'

const route = useRoute()
const router = useRouter()
const { t } = useI18n()
const adminPage = useAdminPage('/ai')
// 页面壳是唯一的数据所有者：页签通过注入拿到同一份上下文，避免同一个
// useAsyncData key 被多个组件持有而在卸载时被清理。
const ai = useAdminAI()
// SSR 上等数据就绪再渲染：否则服务端按空兜底渲染、客户端按真实载荷水合，
// 两边的 DOM 结构不同，hydration 必然 mismatch。
await ai.ready
provide(ADMIN_AI_KEY, ai)
// 可编辑草稿同样由页面壳持有：回复提示词与供应商是两个页签，但它们属于同一次保存。
// 若各自持有一份副本，在提示词页签的改动切回供应商页签就消失了。
const draft = useAdminAISettingsForm(ai)
provide(ADMIN_AI_DRAFT_KEY, draft)

const validTabs: AITab[] = ['providers', 'replyPrompt', 'purposes', 'tools', 'diagnose', 'observability']
const activeTab = ref<AITab>(normalizeTab(route.query.tab))

const tabs = computed(() => [
  { id: 'providers', label: t('admin.ai.tabs.providers'), icon: 'i-lucide-server' },
  { id: 'replyPrompt', label: t('admin.ai.tabs.replyPrompt'), icon: 'i-lucide-message-square-text' },
  { id: 'purposes', label: t('admin.ai.tabs.purposes'), icon: 'i-lucide-blocks' },
  { id: 'tools', label: t('admin.ai.tabs.tools'), icon: 'i-lucide-wrench' },
  { id: 'diagnose', label: t('admin.ai.tabs.diagnose'), icon: 'i-lucide-stethoscope' },
  { id: 'observability', label: t('admin.ai.tabs.observability'), icon: 'i-lucide-activity' }
])

const tabComponents: Record<AITab, Component> = {
  providers: SFAdminAIProvidersTab,
  replyPrompt: SFAdminAIReplyPromptTab,
  purposes: SFAdminAIPurposesTab,
  tools: SFAdminAIToolsTab,
  diagnose: SFAdminAIDiagnoseTab,
  observability: SFAdminAIObservabilityTab
}

const activeComponent = computed(() => tabComponents[activeTab.value])

// 字段级错误按载荷路径索引（profiles[0].defaults.maxTokens 等）。汇总成一段可读
// 文本，避免运营者在五个页签之间找「到底哪一项错了」。
const fieldErrorSummary = computed(() =>
  Object.entries(ai.fieldErrors.value)
    .map(([path, messages]) => `${path}: ${messages.join(' ')}`)
    .join('\n')
)

function normalizeTab(value: unknown): AITab {
  const candidate = Array.isArray(value) ? value[0] : value
  return validTabs.includes(candidate as AITab) ? (candidate as AITab) : 'providers'
}

// 只切换本地状态。用 router.push 把页签写回 query 会触发完整路由导航
// （page:loading → Suspense 重新挂载 → 重新 hydration），那是页签切换白屏的
// 直接原因；replace 只改地址栏，不触发导航，因此刷新与深链能落在对应页签。
function setActiveTab(value: string) {
  const next = normalizeTab(value)
  activeTab.value = next
  if (route.query.tab !== next) {
    void router.replace({ query: { ...route.query, tab: next } })
  }
}
</script>

<template>
  <div class="mb-4">
    <h2 class="flex items-center gap-2 text-xl font-bold text-slate-900 dark:text-zinc-100">
      <UIcon :name="adminPage.icon" class="size-5 text-[var(--sf-accent)] dark:text-[var(--sf-accent-dark)]" />
      {{ t('admin.ai.title') }}
    </h2>
  </div>

  <UDashboardToolbar class="mb-6 overflow-x-hidden rounded-lg border border-slate-200 bg-white px-4 py-2.5 text-slate-500 [&_[data-slot=left]]:min-w-0 [&_[data-slot=left]]:flex-1 dark:border-zinc-800 dark:bg-zinc-900 dark:text-zinc-400">
    <template #left>
      <div class="flex min-w-0 items-center gap-2 text-sm">
        <UIcon name="i-lucide-sparkles" class="size-4 shrink-0" />
        <span class="min-w-0 truncate">{{ t('admin.ai.intro') }}</span>
      </div>
    </template>
    <template #right>
      <UButton
        class="shrink-0"
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

    <!-- 保存属于整份配置而不是某一个页签：放在这里，任一页签的改动都能就地保存，
         不需要先切回供应商页签。 -->
    <div v-if="draft.canManage.value" class="flex flex-wrap items-center gap-2">
      <UButton icon="i-lucide-save" :loading="ai.busy.value === 'settings'" :disabled="!draft.dirty.value" @click="draft.save">
        {{ t('admin.ai.save') }}
      </UButton>
      <UButton icon="i-lucide-rotate-ccw" color="neutral" variant="outline" :loading="ai.busy.value === 'reset'" @click="ai.resetSettings">
        {{ t('admin.ai.reset') }}
      </UButton>
      <span v-if="draft.dirty.value" class="text-xs text-amber-600 dark:text-amber-400">{{ t('admin.ai.dirtyHint') }}</span>
      <!-- 本地拦截原因：这一条会在发请求之前出现，说明该改哪个字段。 -->
      <span v-if="draft.saveIssue.value" class="text-xs text-red-600 dark:text-red-400">{{ draft.saveIssue.value }}</span>
    </div>

    <!-- 服务端的字段级 422 消息：任一页签都可能触发，因此在页面壳汇总展示；
         具体字段旁另有就地提示（见 providers 页签）。 -->
    <UAlert
      v-if="Object.keys(ai.fieldErrors.value).length > 0"
      color="error"
      variant="soft"
      icon="i-lucide-triangle-alert"
      :title="t('admin.ai.saveFailed')"
      :description="fieldErrorSummary"
    />
  </div>
</template>
