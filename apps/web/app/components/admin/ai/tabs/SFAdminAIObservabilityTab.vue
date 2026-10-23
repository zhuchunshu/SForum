<script setup lang="ts">
import type { AdminAIExecution } from '~/composables/admin/useAdminAI'
import { ADMIN_AI_KEY, MICRO_PER_UNIT, useAdminAIRunTimeline } from '~/composables/admin/useAdminAI'
import SFAdminAIRunTimeline from '~/components/admin/ai/blocks/SFAdminAIRunTimeline.vue'
import { executionExpandable, executionStatusColor } from '~/components/admin/ai/runTimeline'
import { replyOutcomeColor, replyOutcomeHref } from '~/components/admin/ai/replyOutcomes'

const { t, locale } = useI18n()
const ai = inject(ADMIN_AI_KEY)!

const site = computed(() => ai.siteUsage.value)
const budgetMicros = computed(() => ai.budgetMicros.value)
const budgetUsed = computed(() => {
  if (budgetMicros.value <= 0) return 0
  return Math.min(100, Math.round((site.value.monthSpendMicros / budgetMicros.value) * 100))
})

function yuan(micros: number) {
  return (micros / MICRO_PER_UNIT).toFixed(2)
}

// 后端 ExecutionRecord 的数值字段全是 `omitempty`：值为 0 时**字段不会出现**。
// 直接插值会渲染出「NaN」「ms」「/」这类残句（浏览器 QA 实测），看起来像数据坏了，
// 而真实含义只是「这一项是 0」。凡数值一律经由这里取值。
function num(value: number | undefined | null): number {
  return typeof value === 'number' && Number.isFinite(value) ? value : 0
}

// 明确 locale：toLocaleString() 不带参数时随运行环境变化，同一条记录在中英文
// 环境下会显示成不同的时间格式。locale code 是 zh-CN / en（language 才是 en-US）。
function stamp(value: string) {
  if (!value) return '-'
  const date = new Date(value)
  if (Number.isNaN(date.getTime())) return value
  return date.toLocaleString(locale.value === 'en' ? 'en-US' : 'zh-CN', {
    year: 'numeric',
    month: '2-digit',
    day: '2-digit',
    hour: '2-digit',
    minute: '2-digit'
  })
}

// 状态色由 runTimeline 提供：表格与时间线必须同一套口径。
function statusColor(status: AdminAIExecution['status']) {
  return executionStatusColor(status)
}

// 执行记录最多 50 条且混合三种状态：排查失败时要在里面翻，筛选是刚需。
//
// cache 是与三种状态并列的一个筛选维度，不是第四种状态：缓存命中行的 status 仍然
// 是 succeeded，只是本次没有出站（0 ms / 0 token 是正确值）。不把它单列出来，
// 一排「成功 + 0 ms + 0/0」只会被读成「供应商坏了」。
type ExecutionFilter = 'all' | AdminAIExecution['status'] | 'cache'
const statusFilter = ref<ExecutionFilter>('all')
const statusOptions = computed(() => [
  { label: t('admin.ai.filterAll'), value: 'all' as const },
  { label: t('admin.ai.statusSucceeded'), value: 'succeeded' as const },
  { label: t('admin.ai.statusFailed'), value: 'failed' as const },
  { label: t('admin.ai.statusDenied'), value: 'denied' as const },
  { label: t('admin.ai.filterCacheHit'), value: 'cache' as const }
])
const filteredExecutions = computed(() => {
  const items = ai.executions.value
  if (statusFilter.value === 'all') return items
  if (statusFilter.value === 'cache') return items.filter(item => item.cacheHit)
  return items.filter(item => item.status === statusFilter.value)
})

// 颜色与落点的唯一实现在 replyOutcomes.ts：表格、测试与将来的公共读面共用
// 同一套判定，模板里不再各自写一遍。
const outcomeColor = replyOutcomeColor
const outcomeHref = replyOutcomeHref

// 回放：同一时刻只展开一处。展开是查询能力的组合（GET /admin/ai/runs/:runId），
// 不是新的持久化面；工具调用明细在日志里，时间线不会补造它。
//
// 展开态按**记录 id** 记，不按 runId：同一 run 的多条记录各占一行（一次 run 通常
// 就是 2–3 行），按 runId 记会让每一行都渲染一个时间线——同一份内容重复三遍，
// 且 aria-controls 指向三个同 id 的面板（浏览器 QA 实测）。记录 id 才是行身份。
const expanded = ref<{ recordId: number, runId: string } | null>(null)
const expandedRunId = computed(() => expanded.value?.runId ?? null)
const {
  items: timelineItems,
  pending: timelinePending,
  loadFailed: timelineLoadFailed
} = useAdminAIRunTimeline(expandedRunId)

function isExpanded(item: AdminAIExecution) {
  return expanded.value?.recordId === item.id
}

function timelinePanelId(item: AdminAIExecution) {
  return `admin-ai-run-timeline-${item.id}`
}

// 行本身可点击（表格里整行是目标），但只有带 run 归属的记录有入口：
// A3 之前的历史记录展开只会得到空列表，入口不该存在。
function toggleRun(item: AdminAIExecution) {
  if (!executionExpandable(item)) return
  if (isExpanded(item)) {
    expanded.value = null
    return
  }
  expanded.value = { recordId: item.id, runId: (item.resourceId ?? '').trim() }
}

function closeTimeline() {
  expanded.value = null
}
</script>

<template>
  <div class="flex min-w-0 flex-col gap-4">
    <div class="grid grid-cols-2 gap-3 xl:grid-cols-4">
      <div class="min-w-0 rounded-lg border border-slate-200 bg-white p-4 dark:border-zinc-800 dark:bg-zinc-900">
        <p class="text-xs uppercase text-slate-500">{{ t('admin.ai.usageMinute') }}</p>
        <p class="mt-2 text-2xl font-semibold tabular-nums">{{ site.minuteCalls }}</p>
      </div>
      <div class="min-w-0 rounded-lg border border-slate-200 bg-white p-4 dark:border-zinc-800 dark:bg-zinc-900">
        <p class="text-xs uppercase text-slate-500">{{ t('admin.ai.usageToday') }}</p>
        <p class="mt-2 text-2xl font-semibold tabular-nums">{{ site.dayCalls }}</p>
      </div>
      <div class="min-w-0 rounded-lg border border-slate-200 bg-white p-4 dark:border-zinc-800 dark:bg-zinc-900">
        <p class="text-xs uppercase text-slate-500">{{ t('admin.ai.usageTokens') }}</p>
        <p class="mt-2 text-2xl font-semibold tabular-nums">{{ site.dayTokens }}</p>
      </div>
      <div class="min-w-0 rounded-lg border border-slate-200 bg-white p-4 dark:border-zinc-800 dark:bg-zinc-900">
        <p class="text-xs uppercase text-slate-500">{{ t('admin.ai.usageMonth') }}</p>
        <p class="mt-2 text-2xl font-semibold tabular-nums">{{ yuan(site.monthSpendMicros) }}</p>
      </div>
    </div>

    <section class="min-w-0 rounded-lg border border-slate-200 bg-white p-4 dark:border-zinc-800 dark:bg-zinc-900">
      <div class="flex flex-wrap items-center justify-between gap-2">
        <p class="text-sm font-semibold text-slate-900 dark:text-zinc-100">{{ t('admin.ai.budget') }}</p>
        <span class="text-xs text-slate-500">
          <template v-if="budgetMicros > 0">
            {{ yuan(site.monthSpendMicros) }} / {{ yuan(budgetMicros) }}
          </template>
          <template v-else>
            {{ t('admin.ai.budgetUnset') }}
          </template>
        </span>
      </div>
      <UProgress v-if="budgetMicros > 0" class="mt-3" :model-value="budgetUsed" :color="budgetUsed >= 90 ? 'error' : 'primary'" />
      <p class="mt-2 text-xs text-slate-500 dark:text-zinc-400">
        {{ budgetMicros > 0 ? t('admin.ai.budgetHint') : t('admin.ai.budgetUnsetHint') }}
      </p>
    </section>

    <section class="min-w-0 rounded-lg border border-slate-200 bg-white dark:border-zinc-800 dark:bg-zinc-900">
      <div class="flex flex-wrap items-center justify-between gap-2 border-b border-slate-200 px-4 py-3 dark:border-zinc-800">
        <div class="min-w-0">
          <p class="text-sm font-semibold text-slate-900 dark:text-zinc-100">{{ t('admin.ai.executions') }}</p>
          <p class="mt-1 text-xs text-slate-500 dark:text-zinc-400">{{ t('admin.ai.executionsIntro') }}</p>
        </div>
        <USelect
          v-model="statusFilter"
          :items="statusOptions"
          class="w-40"
          :aria-label="t('admin.ai.filterStatus')"
        />
      </div>
      <div class="overflow-x-auto">
        <table class="min-w-full text-left text-sm">
          <thead class="bg-slate-50 text-xs text-slate-500 dark:bg-zinc-950">
            <tr>
              <th class="px-3 py-3">{{ t('admin.ai.colTime') }}</th>
              <th class="px-3 py-3">{{ t('admin.ai.colPurpose') }}</th>
              <th class="px-3 py-3">{{ t('admin.ai.colCaller') }}</th>
              <th class="px-3 py-3">{{ t('admin.ai.colModel') }}</th>
              <th class="px-3 py-3">{{ t('admin.ai.colStatus') }}</th>
              <th class="px-3 py-3 text-right">{{ t('admin.ai.colLatency') }}</th>
              <th class="px-3 py-3 text-right">{{ t('admin.ai.colTokens') }}</th>
              <th class="px-3 py-3 text-right">{{ t('admin.ai.colSpend') }}</th>
            </tr>
          </thead>
          <tbody class="divide-y divide-slate-200 dark:divide-zinc-800">
            <template v-for="item in filteredExecutions" :key="item.id">
              <tr
                :class="executionExpandable(item) ? 'cursor-pointer hover:bg-slate-50 dark:hover:bg-zinc-800/50' : undefined"
                @click="toggleRun(item)"
              >
                <td class="whitespace-nowrap px-3 py-3 text-xs text-slate-500">
                  <div class="flex min-w-0 items-center gap-2">
                    <!-- 无 run 归属的历史记录不显示展开入口：点了只会得到空列表。 -->
                    <button
                      v-if="executionExpandable(item)"
                      type="button"
                      class="shrink-0 rounded p-0.5 text-slate-500 hover:text-slate-900 dark:hover:text-zinc-100"
                      :aria-label="isExpanded(item) ? t('admin.ai.runTimeline.collapse') : t('admin.ai.runTimeline.expand')"
                      :title="isExpanded(item) ? t('admin.ai.runTimeline.collapse') : t('admin.ai.runTimeline.expand')"
                      :aria-expanded="isExpanded(item)"
                      :aria-controls="isExpanded(item) ? timelinePanelId(item) : undefined"
                      @click.stop="toggleRun(item)"
                    >
                      <UIcon
                        :name="isExpanded(item) ? 'i-lucide-chevron-down' : 'i-lucide-chevron-right'"
                        class="size-3.5"
                        aria-hidden="true"
                      />
                    </button>
                    <span>{{ stamp(item.createdAt) }}</span>
                  </div>
                </td>
                <td class="px-3 py-3">
                  <p class="font-medium">{{ item.purpose }}</p>
                  <p v-if="item.promptVersion" class="text-xs text-slate-500">{{ item.promptVersion }}</p>
                </td>
                <td class="px-3 py-3 font-mono text-xs">{{ item.callerExtensionId || '-' }}</td>
                <td class="px-3 py-3 font-mono text-xs">{{ item.model || '-' }}</td>
                <td class="px-3 py-3">
                  <UBadge :color="statusColor(item.status)" variant="subtle">{{ item.status }}</UBadge>
                  <UBadge v-if="item.cacheHit" color="neutral" variant="subtle" class="ml-1">{{ t('admin.ai.cacheBadge') }}</UBadge>
                  <p v-if="item.gateReason" class="mt-1 text-xs text-slate-500">
                    {{ item.gateScope ? `${item.gateReason} · ${item.gateScope}` : item.gateReason }}
                  </p>
                  <p v-if="item.errorSummary" class="mt-1 max-w-xs truncate text-xs text-rose-500" :title="item.errorSummary">{{ item.errorSummary }}</p>
                </td>
                <td class="px-3 py-3 text-right tabular-nums">{{ num(item.latencyMs) }} ms</td>
                <td class="px-3 py-3 text-right tabular-nums">{{ num(item.inputTokens) }} / {{ num(item.outputTokens) }}</td>
                <td class="px-3 py-3 text-right tabular-nums">{{ yuan(num(item.spendMicros)) }}</td>
              </tr>
              <tr v-if="isExpanded(item)">
                <td colspan="8" class="px-3 py-3">
                  <SFAdminAIRunTimeline
                    :run-id="String(item.resourceId)"
                    :panel-id="timelinePanelId(item)"
                    :items="timelineItems"
                    :pending="timelinePending"
                    :load-failed="timelineLoadFailed"
                    @close="closeTimeline"
                  />
                </td>
              </tr>
            </template>
          </tbody>
        </table>
        <div v-if="!filteredExecutions.length && !ai.executionsPending.value" class="p-10 text-center text-sm text-slate-500">
          {{ statusFilter === 'all' ? t('admin.ai.executionsEmpty') : t('admin.ai.executionsEmptyFiltered') }}
        </div>
      </div>
      <p class="border-t border-slate-200 px-4 py-2 text-xs text-slate-500 dark:border-zinc-800 dark:text-zinc-400">
        {{ t('admin.ai.cacheHitHint') }}
      </p>
    </section>

    <section class="min-w-0 rounded-lg border border-slate-200 bg-white dark:border-zinc-800 dark:bg-zinc-900">
      <div class="border-b border-slate-200 px-4 py-3 dark:border-zinc-800">
        <p class="text-sm font-semibold text-slate-900 dark:text-zinc-100">{{ t('admin.ai.replyOutcomes') }}</p>
        <p class="mt-1 text-xs text-slate-500 dark:text-zinc-400">{{ t('admin.ai.replyOutcomesIntro') }}</p>
      </div>
      <div class="overflow-x-auto">
        <table class="min-w-full text-left text-sm">
          <thead class="bg-slate-50 text-xs text-slate-500 dark:bg-zinc-950">
            <tr>
              <th class="px-3 py-3">{{ t('admin.ai.colTime') }}</th>
              <th class="px-3 py-3">{{ t('admin.ai.colTrigger') }}</th>
              <th class="px-3 py-3">{{ t('admin.ai.colOutcome') }}</th>
              <th class="px-3 py-3">{{ t('admin.ai.colReason') }}</th>
              <th class="px-3 py-3 text-right">{{ t('admin.ai.colCalls') }}</th>
            </tr>
          </thead>
          <tbody class="divide-y divide-slate-200 dark:divide-zinc-800">
            <tr v-for="item in ai.replyOutcomes.value" :key="item.id">
              <td class="whitespace-nowrap px-3 py-3 text-xs text-slate-500">{{ stamp(item.createdAt) }}</td>
              <td class="px-3 py-3">
                <a
                  v-if="outcomeHref(item)"
                  :href="outcomeHref(item)"
                  target="_blank"
                  rel="noopener"
                  class="font-mono text-xs text-primary hover:underline"
                >#{{ item.topicId }}{{ item.commentId ? ` · #${item.commentId}` : '' }}</a>
                <span v-else class="font-mono text-xs">#{{ item.topicId }}</span>
                <p v-if="item.triggerCommentId" class="text-xs text-slate-500">
                  {{ t('admin.ai.replyOutcomeTrigger', { id: item.triggerCommentId }) }}
                </p>
              </td>
              <td class="px-3 py-3">
                <UBadge :color="outcomeColor(item.outcome)" variant="subtle">{{ t(`admin.ai.replyOutcome.${item.outcome}`) }}</UBadge>
              </td>
              <td class="px-3 py-3">
                <p class="font-mono text-xs">{{ item.reason || '-' }}</p>
                <p v-if="item.detail" class="mt-1 max-w-xs truncate text-xs text-slate-500" :title="item.detail">{{ item.detail }}</p>
              </td>
              <td class="px-3 py-3 text-right tabular-nums">
                {{ num(item.modelCalls) }} / {{ num(item.inputTokens) }} / {{ num(item.outputTokens) }}
              </td>
            </tr>
          </tbody>
        </table>
        <div v-if="!ai.replyOutcomes.value.length && !ai.replyOutcomesPending.value" class="p-10 text-center text-sm text-slate-500">
          {{ t('admin.ai.replyOutcomesEmpty') }}
        </div>
      </div>
    </section>
  </div>
</template>
