<script setup lang="ts">
import type { AdminAIExecution } from '~/composables/admin/useAdminAI'
import {
  cacheHitOnlyRun,
  deniedWithoutOutbound,
  executionStatusColor,
  timelineEntries
} from '~/components/admin/ai/runTimeline'

// 一次 run 的时间线。它是纯渲染：数据由页签（唯一调用 useAdminAIRunTimeline 的地方）
// 传入，这里不重新拉取，也不推断任何后端没有的事实。
const props = defineProps<{
  runId: string
  /**
   * 面板 id 由调用方给：展开按钮的 aria-controls 必须与它逐字相同。
   * 组件自己按 runId 拼 id 会让两者分叉（一次 run 多条记录时按钮按记录 id 命名，
   * 面板按 runId 命名，引用随即失效——浏览器 QA 实测）。
   */
  panelId: string
  items: AdminAIExecution[]
  pending: boolean
  loadFailed: boolean
}>()

const emit = defineEmits<{ close: [] }>()

const { t } = useI18n()

const entries = computed(() => timelineEntries(props.items))
// 「命中缓存」与「没调工具」是两件事：前者是整次 run 只有一条 cacheHit 记录。
const cacheHitOnly = computed(() => cacheHitOnlyRun(props.items))
</script>

<template>
  <section
    :id="panelId"
    class="min-w-0 w-full max-w-[calc(100vw-2rem)] rounded-lg border border-slate-200 bg-slate-50 p-4 dark:border-zinc-800 dark:bg-zinc-950"
    role="region"
    :aria-label="t('admin.ai.runTimeline.title')"
  >
    <div class="flex min-w-0 flex-wrap items-start justify-between gap-2">
      <div class="min-w-0">
        <p class="text-sm font-semibold text-slate-900 dark:text-zinc-100">{{ t('admin.ai.runTimeline.title') }}</p>
        <p class="mt-1 text-xs text-slate-500 dark:text-zinc-400">{{ t('admin.ai.runTimeline.intro') }}</p>
      </div>
      <button
        type="button"
        class="inline-flex shrink-0 items-center gap-1 rounded-md border border-slate-300 px-2 py-1 text-xs text-slate-600 dark:border-zinc-700 dark:text-zinc-300"
        :aria-label="t('admin.ai.runTimeline.close')"
        :title="t('admin.ai.runTimeline.close')"
        @click="emit('close')"
      >
        <UIcon name="i-lucide-x" class="size-3.5" aria-hidden="true" />
        <span>{{ t('admin.ai.runTimeline.close') }}</span>
      </button>
    </div>

    <p class="mt-3 truncate font-mono text-xs text-slate-500 dark:text-zinc-400" :title="runId">
      {{ t('admin.ai.runTimeline.runIdLabel') }}: {{ runId }}
    </p>

    <p v-if="pending" class="mt-3 text-sm text-slate-500 dark:text-zinc-400">{{ t('admin.ai.runTimeline.loading') }}</p>
    <UAlert
      v-else-if="loadFailed"
      class="mt-3"
      color="error"
      variant="soft"
      icon="i-lucide-triangle-alert"
      :title="t('admin.ai.runTimeline.loadFailed')"
    />
    <p v-else-if="!entries.length" class="mt-3 text-sm text-slate-500 dark:text-zinc-400">
      {{ t('admin.ai.runTimeline.empty') }}
    </p>

    <template v-else>
      <p
        v-if="cacheHitOnly"
        class="mt-3 rounded-md border border-sky-200 bg-sky-50 px-3 py-2 text-xs text-sky-800 dark:border-sky-900 dark:bg-sky-950 dark:text-sky-200"
        role="status"
        data-sf-run-timeline-cache-hit
      >
        {{ t('admin.ai.runTimeline.cacheHitOnly') }}
      </p>

      <ol class="mt-3 flex min-w-0 flex-col gap-2">
        <li
          v-for="entry in entries"
          :key="entry.index"
          class="min-w-0 rounded-md border border-slate-200 bg-white p-3 dark:border-zinc-800 dark:bg-zinc-900"
        >
          <div class="flex min-w-0 flex-wrap items-center gap-x-3 gap-y-1">
            <span class="text-xs font-semibold tabular-nums text-slate-500 dark:text-zinc-400">
              {{ t('admin.ai.runTimeline.step', { n: entry.index }) }}
            </span>
            <UBadge :color="executionStatusColor(entry.record.status)" variant="subtle">
              {{ entry.record.status }}
            </UBadge>
            <UBadge v-if="entry.record.cacheHit" color="info" variant="subtle">
              {{ t('admin.ai.runTimeline.cacheHitRecord') }}
            </UBadge>
            <UBadge v-if="deniedWithoutOutbound(entry.record)" color="warning" variant="subtle">
              {{ t('admin.ai.runTimeline.deniedBadge') }}
            </UBadge>
            <span class="min-w-0 truncate font-mono text-xs text-slate-600 dark:text-zinc-300" :title="entry.record.model || '-'">
              {{ entry.record.model || '-' }}
            </span>
          </div>

          <div class="mt-2 flex min-w-0 flex-wrap gap-x-4 gap-y-1 text-xs text-slate-500 dark:text-zinc-400">
            <span class="tabular-nums">{{ t('admin.ai.runTimeline.latency', { ms: entry.record.latencyMs ?? 0 }) }}</span>
            <span class="tabular-nums">
              {{ t('admin.ai.runTimeline.tokens', { input: entry.record.inputTokens ?? 0, output: entry.record.outputTokens ?? 0 }) }}
            </span>
          </div>

          <!-- denied 记录被闸门拦下，没有出站：延迟与 token 都是 0 而不是「用量为 0 的调用」。
               gateReason 是解释它的唯一字段，必须原样展示。 -->
          <p
            v-if="deniedWithoutOutbound(entry.record)"
            class="mt-2 break-words text-xs text-amber-700 dark:text-amber-300"
          >
            {{ t('admin.ai.runTimeline.deniedHint') }}<template v-if="entry.record.gateReason"> · {{ entry.record.gateReason }}</template>
            <template v-if="entry.record.gateScope"> · {{ entry.record.gateScope }}</template>
          </p>

          <p
            v-if="entry.record.errorSummary"
            class="mt-2 break-words text-xs text-rose-600 dark:text-rose-400"
          >
            {{ entry.record.errorSummary }}
          </p>
        </li>
      </ol>

      <!-- 执行记录不含工具调用明细（它在日志里）。写明这一点，界面才不会暗示
           「这里本该有工具步」而让人去猜是不是数据丢了。 -->
      <p class="mt-3 text-xs text-slate-500 dark:text-zinc-400">{{ t('admin.ai.runTimeline.traceHonesty') }}</p>
    </template>
  </section>
</template>
