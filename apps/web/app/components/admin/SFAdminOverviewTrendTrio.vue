<script setup lang="ts">
/**
 * 仪表盘「最近 N 天社区趋势」。
 * 稀疏尖峰数据（多数天为 0）下，柱/折线都会很难看；
 * 改为：大数字汇总 + 等宽 7 日活动格（强度着色），零日安静、峰值日突出。
 */
import {
  formatOverviewCount,
  formatOverviewTrendDayCount,
  overviewTrendDateLabel,
  overviewTrendDeltaKind,
  overviewTrendDeltaPercent,
  overviewTrendFieldMax,
  overviewTrendPeakDate,
  overviewTrendSum,
  type AdminOverviewTrendDay,
  type AdminOverviewTrendField,
  type OverviewTrendDeltaKind
} from '~/utils/adminOverview'

const props = defineProps<{
  days: AdminOverviewTrendDay[]
  windowDays: number
}>()

const { t } = useI18n()

type DayCell = {
  value: number
  label: string
  intensity: number
  isPeak: boolean
  isZero: boolean
}

type TrendSeriesCard = {
  field: AdminOverviewTrendField
  label: string
  unit: string
  toneClass: string
  stroke: string
  total: number
  today: number
  deltaPercent: number
  deltaKind: OverviewTrendDeltaKind
  cells: DayCell[]
  peakLabel: string
  peakValue: number
}

const seriesCards = computed<TrendSeriesCard[]>(() => {
  const days = props.days || []
  const defs: Array<{
    field: AdminOverviewTrendField
    labelKey: string
    unitKey: string
    toneClass: string
    stroke: string
  }> = [
    {
      field: 'topicCount',
      labelKey: 'admin.home.trend.topics',
      unitKey: 'admin.home.trend.unitTopics',
      toneClass: 'trend-topics',
      stroke: 'var(--sf-accent)'
    },
    {
      field: 'commentCount',
      labelKey: 'admin.home.trend.comments',
      unitKey: 'admin.home.trend.unitComments',
      toneClass: 'trend-comments',
      stroke: '#3b82f6'
    },
    {
      field: 'userCount',
      labelKey: 'admin.home.trend.users',
      unitKey: 'admin.home.trend.unitUsers',
      toneClass: 'trend-users',
      stroke: '#16a34a'
    }
  ]

  return defs.map((def) => {
    const values = days.map(day => Math.max(0, Number(day[def.field]) || 0))
    const today = values[values.length - 1] || 0
    const previous = values.length > 1 ? values[values.length - 2] || 0 : 0
    const fieldMax = overviewTrendFieldMax(days, def.field)
    const peakDate = overviewTrendPeakDate(days, def.field)
    const peakIndex = Math.max(0, days.findIndex(day => day.date === peakDate))
    const peakValue = values[peakIndex] || 0

    // 今日无新增时不展示「较昨日 -100%」——对运营几乎无信息量，还显刺眼
    let deltaKind = overviewTrendDeltaKind(today, previous)
    if (today === 0 && previous > 0) {
      deltaKind = 'none'
    }

    const cells: DayCell[] = values.map((value, index) => ({
      value,
      label: overviewTrendDateLabel(days[index]?.date || ''),
      intensity: fieldMax > 0 ? value / fieldMax : 0,
      isPeak: index === peakIndex && value > 0,
      isZero: value <= 0
    }))

    return {
      field: def.field,
      label: t(def.labelKey),
      unit: t(def.unitKey),
      toneClass: def.toneClass,
      stroke: def.stroke,
      total: overviewTrendSum(days, def.field),
      today,
      deltaPercent: overviewTrendDeltaPercent(today, previous),
      deltaKind,
      cells,
      peakLabel: overviewTrendDateLabel(peakDate),
      peakValue
    }
  })
})

function deltaBadgeClass(kind: OverviewTrendDeltaKind) {
  if (kind === 'down') {
    return 'bg-red-500/10 text-red-600 dark:text-red-400'
  }
  if (kind === 'up') {
    return 'bg-[color-mix(in_srgb,var(--series)_14%,transparent)] text-[var(--series)]'
  }
  return 'bg-slate-100 text-slate-500 dark:bg-zinc-800 dark:text-zinc-400'
}

function deltaBadgeText(kind: OverviewTrendDeltaKind, deltaPercent: number) {
  if (kind === 'none') {
    return ''
  }
  if (kind === 'flat') {
    return t('admin.home.trend.vsYesterdayFlat')
  }
  const arrow = kind === 'up' ? '▲' : '▼'
  return `${arrow} ${t('admin.home.trend.vsYesterday', { delta: Math.abs(deltaPercent) })}`
}

/** 活动格背景：零日极淡，峰值日实色，中间按强度插值 */
function cellStyle(card: TrendSeriesCard, cell: DayCell) {
  if (cell.isZero) {
    return {
      background: 'transparent',
      color: undefined as string | undefined,
      borderColor: undefined as string | undefined
    }
  }
  const alpha = cell.isPeak ? 0.22 : 0.08 + cell.intensity * 0.16
  return {
    background: `color-mix(in srgb, ${card.stroke} ${Math.round(alpha * 100)}%, transparent)`,
    color: card.stroke,
    borderColor: cell.isPeak ? card.stroke : undefined
  }
}
</script>

<template>
  <div data-testid="admin-overview-trend-trio" class="min-w-0">
    <div class="grid gap-4 md:grid-cols-3">
      <article
        v-for="card in seriesCards"
        :key="card.field"
        class="min-w-0 rounded-xl border border-slate-200 bg-white p-4 dark:border-zinc-800 dark:bg-zinc-950/50"
        :class="card.toneClass"
      >
        <div class="flex items-start justify-between gap-2">
          <div class="min-w-0">
            <p class="text-sm font-semibold text-slate-600 dark:text-zinc-300">
              {{ card.label }}
            </p>
            <p class="mt-1.5 text-3xl font-black tracking-tight text-slate-900 tabular-nums dark:text-white">
              {{ formatOverviewCount(card.total) }}
            </p>
            <p class="mt-1 text-xs font-medium text-slate-500 dark:text-zinc-400">
              {{ t('admin.home.trend.totalMeta', {
                days: windowDays,
                today: formatOverviewCount(card.today),
                unit: card.unit
              }) }}
            </p>
          </div>
          <span
            v-if="card.deltaKind !== 'none'"
            class="shrink-0 rounded-full px-2 py-0.5 text-[11px] font-bold tabular-nums"
            :class="deltaBadgeClass(card.deltaKind)"
          >
            {{ deltaBadgeText(card.deltaKind, card.deltaPercent) }}
          </span>
        </div>

        <!-- 7 日活动格：等宽、强度着色，稀疏数据也不会「一根柱竖在空旷里」 -->
        <div class="mt-4 grid grid-cols-7 gap-1.5">
          <div
            v-for="(cell, index) in card.cells"
            :key="`${card.field}-cell-${index}`"
            class="flex min-w-0 flex-col items-center gap-1"
            :title="`${cell.label}: ${cell.value}`"
          >
            <div
              class="flex h-11 w-full flex-col items-center justify-center rounded-lg border tabular-nums"
              :class="cell.isZero
                ? 'border-slate-100 bg-slate-50 text-slate-300 dark:border-zinc-800 dark:bg-zinc-900/60 dark:text-zinc-600'
                : cell.isPeak
                  ? 'border-current font-bold shadow-sm'
                  : 'border-transparent font-semibold'"
              :style="cell.isZero ? undefined : cellStyle(card, cell)"
            >
              <span class="text-[11px] leading-none">
                {{ formatOverviewTrendDayCount(cell.value) }}
              </span>
            </div>
            <span
              class="text-[10px] font-semibold tabular-nums leading-none"
              :class="cell.isPeak
                ? 'text-slate-700 dark:text-zinc-200'
                : 'text-slate-400 dark:text-zinc-500'"
            >
              {{ cell.label }}
            </span>
          </div>
        </div>

        <p
          v-if="card.peakValue > 0"
          class="mt-3 flex items-center gap-1.5 text-[11px] text-slate-500 dark:text-zinc-400"
        >
          <span
            class="inline-block size-1.5 shrink-0 rounded-full"
            :style="{ background: card.stroke }"
          />
          {{ t('admin.home.trend.peakDayInline', {
            date: card.peakLabel,
            count: formatOverviewTrendDayCount(card.peakValue)
          }) }}
        </p>
      </article>
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
