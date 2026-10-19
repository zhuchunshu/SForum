// `chart` 围栏的严格解析器。权威契约在 contracts/fixtures/rich-content-v1.json
// 的 chart 段，上限的 TypeScript 镜像在 richContentLimits.ts。
//
// 这里只做纯校验：不碰 DOM、不读全局状态、不抛异常。返回值永远是新对象，
// 输入的解析结果不会被透传（防止调用方改到内部引用，也避免原型污染字段复活）。
import { RICH_CHART_LIMITS, sourceByteLength } from './richContentLimits'
import type { RichRenderFailure } from './richContentTypes'

/** 允许的图表类型，直接来自契约镜像，新增类型必须先改 fixture 与 limits。 */
export type ChartKind = (typeof RICH_CHART_LIMITS.types)[number]

export type ChartSeries = {
  name?: string
  data: number[]
}

export type ChartSpec = {
  type: ChartKind
  labels: string[]
  series: ChartSeries[]
  title?: string
  yLabel?: string
  stacked?: boolean
}

export type ChartSpecResult =
  | { ok: true, spec: ChartSpec }
  | { ok: false, reason: RichRenderFailure }

/** 顶层白名单：多一个键就整段拒绝，避免把未知字段当成配置走私进去。 */
const SPEC_KEYS: ReadonlySet<string> = new Set(['type', 'title', 'labels', 'series', 'yLabel', 'stacked'])
const SERIES_KEYS: ReadonlySet<string> = new Set(['name', 'data'])

const fail = (reason: RichRenderFailure): ChartSpecResult => ({ ok: false, reason })

function isRecord(value: unknown): value is Record<string, unknown> {
  return typeof value === 'object' && value !== null && !Array.isArray(value)
}

function hasUnknownKey(value: Record<string, unknown>, allowed: ReadonlySet<string>): boolean {
  return Object.keys(value).some((key) => !allowed.has(key))
}

/** 字符串按纯文本处理：只校验长度，不做任何 HTML/Markdown 解析。 */
function readText(value: unknown, maxChars: number): string | null {
  if (typeof value !== 'string') return null
  return value.length <= maxChars ? value : null
}

function readLabels(value: unknown): string[] | null {
  if (!Array.isArray(value)) return null
  if (value.length < 1 || value.length > RICH_CHART_LIMITS.maxLabels) return null
  const labels: string[] = []
  for (const entry of value) {
    const text = readText(entry, RICH_CHART_LIMITS.maxLabelChars)
    if (text === null) return null
    labels.push(text)
  }
  return labels
}

function readSeries(value: unknown, labelCount: number): ChartSeries[] | null {
  if (!Array.isArray(value)) return null
  if (value.length < 1 || value.length > RICH_CHART_LIMITS.maxSeries) return null
  const series: ChartSeries[] = []
  for (const entry of value) {
    if (!isRecord(entry) || hasUnknownKey(entry, SERIES_KEYS)) return null
    const data = entry.data
    if (!Array.isArray(data) || data.length !== labelCount) return null
    const points: number[] = []
    for (const point of data) {
      if (typeof point !== 'number' || !Number.isFinite(point)) return null
      if (Math.abs(point) > RICH_CHART_LIMITS.maxAbsValue) return null
      points.push(point)
    }
    const item: ChartSeries = { data: points }
    if ('name' in entry) {
      const name = readText(entry.name, RICH_CHART_LIMITS.maxLabelChars)
      if (name === null) return null
      item.name = name
    }
    series.push(item)
  }
  return series
}

/**
 * 解析 ```chart 围栏正文。
 *
 * 判定顺序：字节上限（先拦超长，避免解析大字符串）→ JSON 语法 → 结构白名单
 * → 类型白名单 → 标签/系列/数值。`type` 缺失或不是字符串属于结构损坏（invalid）；
 * 是字符串但不在契约列表里才是 unsupported。
 */
export function parseChartSpec(source: string): ChartSpecResult {
  if (sourceByteLength(source) > RICH_CHART_LIMITS.maxSourceBytes) return fail('too_large')

  let raw: unknown
  try {
    raw = JSON.parse(source)
  } catch {
    return fail('invalid')
  }

  if (!isRecord(raw) || hasUnknownKey(raw, SPEC_KEYS)) return fail('invalid')

  const type = raw.type
  if (typeof type !== 'string') return fail('invalid')
  if (!(RICH_CHART_LIMITS.types as readonly string[]).includes(type)) return fail('unsupported')

  const labels = readLabels(raw.labels)
  if (labels === null) return fail('invalid')

  const series = readSeries(raw.series, labels.length)
  if (series === null) return fail('invalid')

  const spec: ChartSpec = { type: type as ChartKind, labels, series }

  if ('title' in raw) {
    const title = readText(raw.title, RICH_CHART_LIMITS.maxTitleChars)
    if (title === null) return fail('invalid')
    spec.title = title
  }
  if ('yLabel' in raw) {
    // 轴标题与标题同级，共用 maxTitleChars 上限。
    const yLabel = readText(raw.yLabel, RICH_CHART_LIMITS.maxTitleChars)
    if (yLabel === null) return fail('invalid')
    spec.yLabel = yLabel
  }
  if ('stacked' in raw) {
    if (typeof raw.stacked !== 'boolean') return fail('invalid')
    spec.stacked = raw.stacked
  }

  return { ok: true, spec }
}
