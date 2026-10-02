<script setup lang="ts">
import type { AdminAIExecution } from '~/composables/admin/useAdminAI'
import { ADMIN_AI_KEY, MICRO_PER_UNIT } from '~/composables/admin/useAdminAI'

const { t } = useI18n()
const ai = inject(ADMIN_AI_KEY)!

const site = computed(() => ai.siteUsage.value)
const budgetMicros = computed(() => ai.budgetMicros.value)
const budgetUsed = computed(() => {
  if (budgetMicros.value <= 0) return 0
  return Math.min(100, Math.round((site.value.monthSpendMicros / budgetMicros.value) * 100))
})

function yuan(micros: number) {
  return (micros / MICRO_PER_UNIT).toFixed(4)
}

function stamp(value: string) {
  if (!value) return '-'
  const date = new Date(value)
  return Number.isNaN(date.getTime()) ? value : date.toLocaleString()
}

function statusColor(status: AdminAIExecution['status']) {
  if (status === 'succeeded') return 'success'
  if (status === 'denied') return 'warning'
  return 'error'
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
        <span class="text-xs text-slate-500">{{ yuan(site.monthSpendMicros) }} / {{ yuan(budgetMicros) }}</span>
      </div>
      <UProgress class="mt-3" :model-value="budgetUsed" :color="budgetUsed >= 90 ? 'error' : 'primary'" />
      <p class="mt-2 text-xs text-slate-500 dark:text-zinc-400">{{ t('admin.ai.budgetHint') }}</p>
    </section>

    <section class="min-w-0 rounded-lg border border-slate-200 bg-white dark:border-zinc-800 dark:bg-zinc-900">
      <div class="border-b border-slate-200 px-4 py-3 dark:border-zinc-800">
        <p class="text-sm font-semibold text-slate-900 dark:text-zinc-100">{{ t('admin.ai.executions') }}</p>
        <p class="mt-1 text-xs text-slate-500 dark:text-zinc-400">{{ t('admin.ai.executionsIntro') }}</p>
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
            <tr v-for="item in ai.executions.value" :key="item.id">
              <td class="whitespace-nowrap px-3 py-3 text-xs text-slate-500">{{ stamp(item.createdAt) }}</td>
              <td class="px-3 py-3">
                <p class="font-medium">{{ item.purpose }}</p>
                <p v-if="item.promptVersion" class="text-xs text-slate-500">{{ item.promptVersion }}</p>
              </td>
              <td class="px-3 py-3 font-mono text-xs">{{ item.callerExtensionId || '-' }}</td>
              <td class="px-3 py-3 font-mono text-xs">{{ item.model || '-' }}</td>
              <td class="px-3 py-3">
                <UBadge :color="statusColor(item.status)" variant="subtle">{{ item.status }}</UBadge>
                <p v-if="item.gateReason" class="mt-1 text-xs text-slate-500">{{ item.gateReason }} · {{ item.gateScope }}</p>
                <p v-if="item.errorSummary" class="mt-1 max-w-xs truncate text-xs text-rose-500" :title="item.errorSummary">{{ item.errorSummary }}</p>
              </td>
              <td class="px-3 py-3 text-right tabular-nums">{{ item.latencyMs }} ms</td>
              <td class="px-3 py-3 text-right tabular-nums">{{ item.inputTokens }} / {{ item.outputTokens }}</td>
              <td class="px-3 py-3 text-right tabular-nums">{{ yuan(item.spendMicros) }}</td>
            </tr>
          </tbody>
        </table>
        <div v-if="!ai.executions.value.length && !ai.executionsPending.value" class="p-10 text-center text-sm text-slate-500">
          {{ t('admin.ai.executionsEmpty') }}
        </div>
      </div>
    </section>
  </div>
</template>
