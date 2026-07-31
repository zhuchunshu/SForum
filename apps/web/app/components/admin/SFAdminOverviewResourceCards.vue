<script setup lang="ts">
import { computed } from 'vue'
import SFResourceGauge from './SFResourceGauge.vue'
import {
  formatOverviewBytes,
  formatOverviewLoad,
  formatOverviewPercent,
  type AdminOverviewRuntime
} from '~/utils/admin/adminOverview'

const props = defineProps<{
  runtime: AdminOverviewRuntime
}>()

const { t } = useI18n()

const resources = computed(() => props.runtime.resources)

function unavailable() {
  return t('admin.home.resources.unavailable')
}

const cpuPercent = computed(() => resources.value ? resources.value.apiCpuPercent : 0)
const cpuValue = computed(() => resources.value ? formatOverviewPercent(cpuPercent.value) : unavailable())

const loadRows = computed(() => {
  const average = props.runtime.loadAverage
  return [
    {
      key: 'oneMinute',
      label: t('admin.home.resources.loadOneMinute'),
      value: average ? formatOverviewLoad(average.oneMinute) : unavailable()
    },
    {
      key: 'fiveMinutes',
      label: t('admin.home.resources.loadFiveMinutes'),
      value: average ? formatOverviewLoad(average.fiveMinutes) : unavailable()
    },
    {
      key: 'fifteenMinutes',
      label: t('admin.home.resources.loadFifteenMinutes'),
      value: average ? formatOverviewLoad(average.fifteenMinutes) : unavailable()
    }
  ]
})

const memoryRows = computed(() => {
  const snapshot = resources.value
  return [
    {
      key: 'api',
      label: t('admin.home.resources.api'),
      value: snapshot ? formatOverviewBytes(snapshot.apiMemoryBytes) : unavailable(),
      prominent: false
    },
    {
      key: 'worker',
      label: t('admin.home.resources.worker'),
      value: snapshot
        ? (!snapshot.workerFound && snapshot.workerMemoryBytes === 0
            ? t('admin.home.resources.workerEmbedded')
            : formatOverviewBytes(snapshot.workerMemoryBytes))
        : unavailable(),
      prominent: false
    },
    {
      key: 'plugin',
      label: t('admin.home.resources.plugin'),
      value: snapshot ? formatOverviewBytes(snapshot.pluginMemoryBytes) : unavailable(),
      prominent: false
    },
    {
      key: 'total',
      label: t('admin.home.resources.total'),
      value: snapshot ? formatOverviewBytes(snapshot.totalMemoryBytes) : unavailable(),
      prominent: true
    }
  ]
})
</script>

<template>
  <div class="grid min-w-0 gap-5 xl:grid-cols-3">
    <!-- Memory Breakdown -->
    <UCard class="elegant-card min-w-0 border-slate-200 bg-white text-slate-900 dark:border-zinc-800 dark:bg-zinc-900 dark:text-zinc-100">
      <template #header>
        <div class="flex items-center justify-between gap-3">
          <h3 class="truncate text-base font-bold text-slate-900 dark:text-white">
            {{ t('admin.home.resources.memoryTitle') }}
          </h3>
          <span class="icon-glass-box shrink-0 text-[var(--sf-accent)] dark:text-[var(--sf-accent-dark)]">
            <UIcon name="i-lucide-memory-stick" class="z-10 size-5" />
          </span>
        </div>
      </template>
      <dl class="min-w-0 divide-y divide-slate-100 pt-1 dark:divide-zinc-800">
        <div
          v-for="row in memoryRows"
          :key="row.key"
          class="flex min-w-0 items-center justify-between gap-3 py-2"
        >
          <dt class="min-w-0 truncate text-xs font-semibold text-slate-500 dark:text-zinc-400">
            {{ row.label }}
          </dt>
          <dd
            class="shrink-0 font-mono text-sm font-bold text-slate-900 dark:text-zinc-100"
            :class="row.prominent ? 'text-[var(--sf-accent)] dark:text-[var(--sf-accent-dark)]' : ''"
          >
            {{ row.value }}
          </dd>
        </div>
      </dl>
    </UCard>

    <!-- CPU Gauge -->
    <UCard class="elegant-card min-w-0 border-slate-200 bg-white text-slate-900 dark:border-zinc-800 dark:bg-zinc-900 dark:text-zinc-100">
      <template #header>
        <div class="flex items-center justify-between gap-3">
          <h3 class="truncate text-base font-bold text-slate-900 dark:text-white">
            {{ t('admin.home.resources.cpuTitle') }}
          </h3>
          <span class="icon-glass-box shrink-0 text-blue-600 dark:text-blue-400">
            <UIcon name="i-lucide-gauge" class="z-10 size-5" />
          </span>
        </div>
      </template>
      <div class="flex flex-col items-center justify-center h-full pt-4">
        <SFResourceGauge
          :value="cpuValue"
          :percent="cpuPercent"
          label="CPU"
          icon="i-lucide-gauge"
        />
      </div>
    </UCard>

    <!-- System Load Average -->
    <UCard class="elegant-card min-w-0 border-slate-200 bg-white text-slate-900 dark:border-zinc-800 dark:bg-zinc-900 dark:text-zinc-100">
      <template #header>
        <div class="flex items-center justify-between gap-3">
          <h3 class="truncate text-base font-bold text-slate-900 dark:text-white">
            {{ t('admin.home.resources.loadTitle') }}
          </h3>
          <span class="icon-glass-box shrink-0 text-emerald-600 dark:text-emerald-400">
            <UIcon name="i-lucide-activity" class="z-10 size-5" />
          </span>
        </div>
      </template>
      <dl class="min-w-0 divide-y divide-slate-100 pt-1 dark:divide-zinc-800">
        <div
          v-for="row in loadRows"
          :key="row.key"
          class="flex min-w-0 items-center justify-between gap-3 py-2"
        >
          <dt class="min-w-0 truncate text-xs font-semibold text-slate-500 dark:text-zinc-400">
            {{ row.label }}
          </dt>
          <dd class="shrink-0 font-mono text-sm font-bold text-slate-900 dark:text-zinc-100">
            {{ row.value }}
          </dd>
        </div>
      </dl>
    </UCard>
  </div>
</template>
