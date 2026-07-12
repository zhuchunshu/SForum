<script setup lang="ts">
/**
 * 仪表盘「最近 N 天社区趋势」01D：三列独立迷你图。
 * 主题 / 回复 / 新用户各自刻度，避免数量级差导致读数失真。
 */
import {
  formatOverviewCount,
  overviewTrendDateLabel,
  overviewTrendDeltaPercent,
  overviewTrendPeakDate,
  overviewTrendSparkPath,
  overviewTrendSum,
  type AdminOverviewTrendDay,
  type AdminOverviewTrendField
} from '~/utils/adminOverview'

const props = defineProps<{
  days: AdminOverviewTrendDay[]
  windowDays: number
}>()

const { t } = useI18n()

const SPARK_WIDTH = 280
const SPARK_HEIGHT = 72

type TrendSeriesCard = {
  field: AdminOverviewTrendField
  label: string
  unit: string
  toneClass: string
  stroke: string
  fillSoft: string
  total: number
  today: number
  deltaPercent: number
  deltaUp: boolean
  spark: ReturnType<typeof overviewTrendSparkPath>
  dayValues: number[]
  dateLabels: string[]
  peakLabel: string
}

const seriesCards = computed<TrendSeriesCard[]>(() => {
  const days = props.days || []
  const defs: Array<{
    field: AdminOverviewTrendField
    labelKey: string
    unitKey: string
    toneClass: string
    stroke: string
    fillSoft: string
  }> = [
    {
      field: 'topicCount',
      labelKey: 'admin.home.trend.topics',
      unitKey: 'admin.home.trend.unitTopics',
      toneClass: 'trend-topics',
      stroke: 'var(--sf-accent)',
      fillSoft: 'color-mix(in srgb, var(--sf-accent) 16%, transparent)'
    },
    {
      field: 'commentCount',
      labelKey: 'admin.home.trend.comments',
      unitKey: 'admin.home.trend.unitComments',
      toneClass: 'trend-comments',
      stroke: '#3b82f6',
      fillSoft: 'rgba(59, 130, 246, 0.14)'
    },
    {
      field: 'userCount',
      labelKey: 'admin.home.trend.users',
      unitKey: 'admin.home.trend.unitUsers',
      toneClass: 'trend-users',
      stroke: '#16a34a',
      fillSoft: 'rgba(22, 163, 74, 0.14)'
    }
  ]

  return defs.map((def) => {
    const values = days.map(day => Math.max(0, Number(day[def.field]) || 0))
    const today = values[values.length - 1] || 0
    const previous = values.length > 1 ? values[values.length - 2] || 0 : 0
    const deltaPercent = overviewTrendDeltaPercent(today, previous)

    return {
      field: def.field,
      label: t(def.labelKey),
      unit: t(def.unitKey),
      toneClass: def.toneClass,
      stroke: def.stroke,
      fillSoft: def.fillSoft,
      total: overviewTrendSum(days, def.field),
      today,
      deltaPercent,
      deltaUp: deltaPercent >= 0,
      spark: overviewTrendSparkPath(values, SPARK_WIDTH, SPARK_HEIGHT),
      dayValues: values,
      dateLabels: days.map(day => overviewTrendDateLabel(day.date)),
      peakLabel: overviewTrendDateLabel(overviewTrendPeakDate(days, def.field))
    }
  })
})

const peakFootnotes = computed(() => seriesCards.value.map(card => ({
  field: card.field,
  label: card.label,
  peakLabel: card.peakLabel
})))
</script>

<template>
  <div data-testid="admin-overview-trend-trio" class="min-w-0">
    <div class="grid gap-3 sm:grid-cols-2 xl:grid-cols-3">
      <article
        v-for="card in seriesCards"
        :key="card.field"
        class="min-w-0 rounded-xl border border-slate-200 bg-white p-3.5 dark:border-zinc-800 dark:bg-zinc-950/40"
        :class="card.toneClass"
      >
        <div class="mb-1.5 flex items-start justify-between gap-2">
          <p class="text-xs font-semibold text-slate-500 dark:text-zinc-400">
            {{ card.label }}
          </p>
          <span
            class="shrink-0 rounded-full px-2 py-0.5 text-[11px] font-bold tabular-nums"
            :class="card.deltaUp
              ? 'bg-[color-mix(in_srgb,var(--series)_14%,transparent)] text-[var(--series)]'
              : 'bg-red-500/10 text-red-600 dark:text-red-400'"
          >
            {{ card.deltaUp ? '▲' : '▼' }}
            {{ t('admin.home.trend.vsYesterday', { delta: Math.abs(card.deltaPercent) }) }}
          </span>
        </div>

        <p class="text-2xl font-black tracking-tight text-slate-900 tabular-nums dark:text-white">
          {{ formatOverviewCount(card.total) }}
        </p>
        <p class="mt-0.5 text-[11px] font-medium text-slate-500 dark:text-zinc-400">
          {{ t('admin.home.trend.totalMeta', {
            days: windowDays,
            today: formatOverviewCount(card.today),
            unit: card.unit
          }) }}
        </p>

        <div
          class="mt-3 h-[88px] rounded-lg px-1.5 pb-1 pt-2"
          :style="{ background: card.fillSoft }"
        >
          <svg
            class="block h-full w-full overflow-visible"
            :viewBox="`0 0 ${SPARK_WIDTH} ${SPARK_HEIGHT}`"
            preserveAspectRatio="none"
            role="img"
            :aria-label="card.label"
          >
            <path
              v-if="card.spark.area"
              :d="card.spark.area"
              :fill="card.stroke"
              fill-opacity="0.18"
            />
            <path
              v-if="card.spark.line"
              :d="card.spark.line"
              fill="none"
              :stroke="card.stroke"
              stroke-width="2.5"
              stroke-linecap="round"
              stroke-linejoin="round"
            />
            <circle
              v-for="(point, index) in card.spark.points"
              :key="`${card.field}-${index}`"
              :cx="point.x"
              :cy="point.y"
              r="3.5"
              fill="#fff"
              :stroke="card.stroke"
              stroke-width="2"
              class="dark:fill-zinc-900"
            />
          </svg>
        </div>

        <div class="mt-2 grid grid-cols-7 gap-0.5">
          <div
            v-for="(value, index) in card.dayValues"
            :key="`${card.field}-day-${index}`"
            class="min-w-0 text-center"
          >
            <span
              class="block truncate text-[11px] font-bold tabular-nums"
              :style="{ color: card.stroke }"
            >
              {{ formatOverviewCount(value) }}
            </span>
            <span class="mt-0.5 block truncate text-[9.5px] font-semibold text-slate-400 dark:text-zinc-500">
              {{ card.dateLabels[index] }}
            </span>
          </div>
        </div>
      </article>
    </div>

    <div
      v-if="peakFootnotes.some(item => item.peakLabel)"
      class="mt-3 flex flex-wrap gap-x-4 gap-y-1 text-xs text-slate-500 dark:text-zinc-400"
    >
      <span
        v-for="item in peakFootnotes"
        :key="`peak-${item.field}`"
      >
        {{ t('admin.home.trend.peakDay', { label: item.label }) }}
        <strong class="font-semibold tabular-nums text-slate-800 dark:text-zinc-200">{{ item.peakLabel }}</strong>
      </span>
    </div>
  </div>
</template>

<style scoped>
.trend-topics {
  --series: var(--sf-accent);
}
.dark .trend-topics {
  --series: var(--sf-accent-dark, var(--sf-accent));
}
.trend-comments {
  --series: #3b82f6;
}
.dark .trend-comments {
  --series: #60a5fa;
}
.trend-users {
  --series: #16a34a;
}
.dark .trend-users {
  --series: #4ade80;
}
</style>
