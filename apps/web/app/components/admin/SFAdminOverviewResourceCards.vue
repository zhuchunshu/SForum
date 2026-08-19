<script setup lang="ts">
import { computed } from 'vue'
import SFResourceGauge from './SFResourceGauge.vue'
import {
  formatOverviewBytes,
  formatOverviewLoad,
  formatOverviewPercent,
  overviewMemoryDisplayBytes,
  overviewPSSDisplayBytes,
  type AdminOverviewMemoryBucket,
  type AdminOverviewPluginRuntimeUsage,
  type AdminOverviewRuntime
} from '~/utils/admin/adminOverview'

const props = defineProps<{
  runtime: AdminOverviewRuntime
}>()

const { t } = useI18n()

const resources = computed(() => props.runtime.resources)
const plugins = computed(() => resources.value?.plugins || [])

function unavailable() {
  return t('admin.home.resources.unavailable')
}

function memoryValue(bucket: AdminOverviewMemoryBucket) {
  return resources.value
    ? formatOverviewBytes(overviewMemoryDisplayBytes(resources.value, bucket))
    : unavailable()
}

const apiLabel = computed(() => resources.value?.workerEmbedded
  ? t('admin.home.resources.apiWithWorker')
  : t('admin.home.resources.api'))

const workerValue = computed(() => {
  const snapshot = resources.value
  if (!snapshot) return unavailable()
  if (snapshot.workerEmbedded) {
    const slots = Math.max(0, snapshot.workerConcurrency || 0)
    const running = props.runtime.queueLag?.running
    if (slots > 0 && typeof running === 'number') {
      return t('admin.home.resources.workerEmbeddedRunning', {
        running: Math.max(0, running),
        concurrency: slots
      })
    }
    if (slots > 0) {
      return t('admin.home.resources.workerEmbeddedSlots', { concurrency: slots })
    }
    return t('admin.home.resources.workerEmbedded')
  }
  if (!snapshot.workerFound) {
    return t('admin.home.resources.workerNotFound')
  }
  return memoryValue('worker')
})

const pluginProcessMeta = computed(() => {
  const snapshot = resources.value
  if (!snapshot) return ''
  return t('admin.home.resources.pluginProcesses', { count: snapshot.pluginChildCount })
})

const memoryBasis = computed(() => {
  const seconds = Math.max(1, resources.value?.memoryWindowSeconds || 60)
  return t('admin.home.resources.memoryBasis', { seconds })
})

const pssTotal = computed(() => {
  const snapshot = resources.value
  const value = snapshot ? overviewPSSDisplayBytes(snapshot, 'total') : 0
  return value
    ? t('admin.home.resources.pssTotal', {
        value: formatOverviewBytes(value),
        seconds: Math.max(1, snapshot?.memoryWindowSeconds || 60)
      })
    : ''
})

const anonHugePagesTotal = computed(() => {
  const value = resources.value?.totalAnonHugePagesBytes
  return value
    ? t('admin.home.resources.anonHugePages', { value: formatOverviewBytes(value) })
    : ''
})

function pluginPSSValue(plugin: AdminOverviewPluginRuntimeUsage) {
  const value = plugin.pssMedianBytes || plugin.pssBytes
  return value ? formatOverviewBytes(value) : ''
}

const cpuPercent = computed(() => resources.value ? resources.value.apiCpuPercent : 0)
const cpuValue = computed(() => resources.value ? formatOverviewPercent(cpuPercent.value) : '\u2014')
const cpuLabel = computed(() => resources.value ? 'CPU' : unavailable())

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
</script>

<template>
  <div class="grid min-w-0 gap-5 xl:grid-cols-3">
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
        <div class="flex min-w-0 items-center justify-between gap-3 py-2">
          <dt class="min-w-0 text-xs font-semibold text-slate-500 dark:text-zinc-400">
            {{ apiLabel }}
          </dt>
          <dd class="shrink-0 font-mono text-sm font-bold text-slate-900 dark:text-zinc-100">
            {{ memoryValue('api') }}
          </dd>
        </div>

        <div class="flex min-w-0 items-center justify-between gap-3 py-2">
          <dt class="min-w-0 text-xs font-semibold text-slate-500 dark:text-zinc-400">
            {{ t('admin.home.resources.worker') }}
          </dt>
          <dd class="max-w-[68%] text-right font-mono text-xs font-bold leading-5 text-slate-900 sm:text-sm dark:text-zinc-100">
            {{ workerValue }}
          </dd>
        </div>

        <div class="flex min-w-0 items-center justify-between gap-3 py-2">
          <dt class="min-w-0">
            <span class="block text-xs font-semibold text-slate-500 dark:text-zinc-400">
              {{ t('admin.home.resources.plugin') }}
            </span>
            <span v-if="resources" class="mt-0.5 block text-[11px] leading-4 text-slate-400 dark:text-zinc-500">
              {{ pluginProcessMeta }}
              <span
                v-if="resources.pluginOverlapCount > 0"
                class="ml-1 text-amber-600 dark:text-amber-400"
              >
                {{ t('admin.home.resources.pluginOverlap', { count: resources.pluginOverlapCount }) }}
              </span>
            </span>
          </dt>
          <div class="flex shrink-0 items-center gap-1.5">
            <dd class="font-mono text-sm font-bold text-slate-900 dark:text-zinc-100">
              {{ memoryValue('plugin') }}
            </dd>
            <UPopover
              v-if="plugins.length"
              :content="{ align: 'end', side: 'bottom', collisionPadding: 12 }"
            >
              <UButton
                icon="i-lucide-list-tree"
                color="neutral"
                variant="ghost"
                size="xs"
                square
                :aria-label="t('admin.home.resources.pluginDetails')"
                :title="t('admin.home.resources.pluginDetails')"
              />

              <template #content>
                <div class="max-h-[min(26rem,70vh)] w-[min(22rem,calc(100vw-2rem))] overflow-y-auto p-3">
                  <div class="flex items-center justify-between gap-3 pb-2">
                    <p class="text-sm font-semibold text-slate-900 dark:text-zinc-100">
                      {{ t('admin.home.resources.pluginDetails') }}
                    </p>
                    <span class="text-xs tabular-nums text-slate-500 dark:text-zinc-400">
                      {{ t('admin.home.resources.pluginProcesses', { count: resources?.pluginChildCount || 0 }) }}
                    </span>
                  </div>
                  <p
                    v-if="resources && resources.pluginOverlapCount > 0"
                    class="flex items-start gap-1.5 border-t border-amber-200 py-2 text-xs leading-5 text-amber-700 dark:border-amber-900/70 dark:text-amber-300"
                  >
                    <UIcon name="i-lucide-info" class="mt-0.5 size-3.5 shrink-0" aria-hidden="true" />
                    <span>{{ t('admin.home.resources.pluginOverlapDetail', { count: resources.pluginOverlapCount }) }}</span>
                  </p>
                  <ul class="divide-y divide-slate-100 border-t border-slate-100 dark:divide-zinc-800 dark:border-zinc-800">
                    <li
                      v-for="plugin in plugins"
                      :key="plugin.extensionId"
                      class="flex min-w-0 items-start justify-between gap-3 py-2.5"
                    >
                      <div class="min-w-0">
                        <p class="truncate font-mono text-xs font-semibold text-slate-800 dark:text-zinc-200" :title="plugin.extensionId">
                          {{ plugin.extensionId }}
                        </p>
                        <p class="mt-0.5 text-[11px] leading-4 text-slate-500 dark:text-zinc-400">
                          {{ t('admin.home.resources.pluginProcesses', { count: plugin.processCount }) }}
                          <span v-if="pluginPSSValue(plugin)">
                            · PSS {{ pluginPSSValue(plugin) }}
                          </span>
                          <span v-if="plugin.anonHugePagesBytes">
                            · THP {{ formatOverviewBytes(plugin.anonHugePagesBytes) }}
                          </span>
                        </p>
                      </div>
                      <span class="shrink-0 font-mono text-xs font-bold text-slate-900 dark:text-zinc-100">
                        {{ formatOverviewBytes(plugin.memoryBytes) }}
                      </span>
                    </li>
                  </ul>
                  <p class="border-t border-slate-100 pt-2 text-[11px] text-slate-400 dark:border-zinc-800 dark:text-zinc-500">
                    {{ t('admin.home.resources.pluginDetailsBasis') }}
                  </p>
                </div>
              </template>
            </UPopover>
          </div>
        </div>

        <div class="flex min-w-0 items-center justify-between gap-3 py-2">
          <dt class="min-w-0 text-xs font-semibold text-slate-500 dark:text-zinc-400">
            {{ t('admin.home.resources.total') }}
          </dt>
          <dd class="shrink-0 font-mono text-sm font-bold text-[var(--sf-accent)] dark:text-[var(--sf-accent-dark)]">
            {{ memoryValue('total') }}
          </dd>
        </div>
      </dl>

      <div class="mt-1 flex flex-wrap items-center justify-between gap-x-3 gap-y-1 border-t border-slate-100 pt-2 text-[11px] text-slate-400 dark:border-zinc-800 dark:text-zinc-500">
        <span>{{ memoryBasis }}</span>
        <div class="flex flex-wrap justify-end gap-x-3 gap-y-1">
          <span v-if="pssTotal">{{ pssTotal }}</span>
          <span v-if="anonHugePagesTotal">{{ anonHugePagesTotal }}</span>
        </div>
      </div>
    </UCard>

    <UCard
      class="elegant-card flex min-w-0 flex-col border-slate-200 bg-white text-slate-900 dark:border-zinc-800 dark:bg-zinc-900 dark:text-zinc-100"
      :ui="{ body: 'flex flex-1 items-center justify-center py-5 sm:py-6' }"
    >
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
      <SFResourceGauge
        class="text-blue-600 dark:text-blue-400"
        :value="cpuValue"
        :percent="cpuPercent"
        :label="cpuLabel"
        color="currentColor"
        size="lg"
      />
    </UCard>

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
