// Chart.js 渲染器（轨道 B）。把 ```chart 围栏渲染成 target 内的一个 <canvas>。
//
// 设计约束：
// - chart.js 只在真正渲染时动态 import（不进首屏 bundle，也不在模块顶层引入）。
// - 解析/校验全部交给 chartSpec.parseChartSpec，本文件不重复做结构判断。
// - 颜色从 CSS 变量取（--sf-rich-text / --sf-accent），缺失时退回 currentColor
//   与按 colorMode 选定的调色板，避免图表在深色主题里变成不可读的黑块。
// - 同一 target 重复渲染时先销毁旧实例，abort 后不再往 DOM 里写东西。
import { parseChartSpec, type ChartSpec } from './chartSpec'
import type { RichRenderer } from './richContentTypes'

type ChartInstance = { destroy: () => void }
type ChartConstructor = new (item: HTMLCanvasElement, config: unknown) => ChartInstance
type ChartHandle = {
  chart: ChartInstance
  canvas: HTMLCanvasElement
  signal: AbortSignal
  onAbort: () => void
}
type MediaQueryLike = (query: string) => { matches: boolean }
type ColorMode = 'light' | 'dark'
type Dataset = Record<string, unknown>

const handles = new WeakMap<HTMLElement, ChartHandle>()

const CANVAS_HEIGHT = '280px'
const REDUCED_MOTION_QUERY = '(prefers-reduced-motion: reduce)'

/** 变量缺失时的兜底调色板：浅色用饱和色，深色用提亮版本。 */
const SERIES_PALETTE: Record<ColorMode, readonly string[]> = {
  light: ['#2563eb', '#dc2626', '#059669', '#d97706', '#7c3aed', '#0891b2'],
  dark: ['#60a5fa', '#f87171', '#34d399', '#fbbf24', '#a78bfa', '#22d3ee']
}

type ResolvedColors = {
  text: string
  accent: string
  palette: readonly string[]
}

function computedStyleFor(target: HTMLElement): CSSStyleDeclaration | null {
  const view = target.ownerDocument?.defaultView
  if (view && typeof view.getComputedStyle === 'function') return view.getComputedStyle(target)
  if (typeof globalThis.getComputedStyle === 'function') return globalThis.getComputedStyle(target)
  return null
}

function readCssVar(target: HTMLElement, name: string): string | null {
  try {
    const value = computedStyleFor(target)?.getPropertyValue(name).trim()
    return value ? value : null
  } catch {
    return null
  }
}

function prefersReducedMotion(target: HTMLElement): boolean {
  const view = target.ownerDocument?.defaultView
  const mediaQuery: MediaQueryLike | undefined =
    view && typeof view.matchMedia === 'function'
      ? view.matchMedia.bind(view)
      : (globalThis as { matchMedia?: MediaQueryLike }).matchMedia
  if (!mediaQuery) return false
  try {
    return mediaQuery(REDUCED_MOTION_QUERY).matches === true
  } catch {
    return false
  }
}

/** 给 #rgb / #rrggbb 加透明度；currentColor、rgb() 等无法换算的原样返回。 */
function withAlpha(color: string, alpha: number): string {
  const match = /^#([0-9a-f]{3}|[0-9a-f]{6})$/i.exec(color.trim())
  if (!match) return color
  const digits = match[1]!
  const full = digits.length === 3
    ? digits.split('').map((char) => char + char).join('')
    : digits
  const red = Number.parseInt(full.slice(0, 2), 16)
  const green = Number.parseInt(full.slice(2, 4), 16)
  const blue = Number.parseInt(full.slice(4, 6), 16)
  return `rgba(${red}, ${green}, ${blue}, ${alpha})`
}

function resolveColors(target: HTMLElement, colorMode: ColorMode): ResolvedColors {
  const palette = SERIES_PALETTE[colorMode]
  return {
    text: readCssVar(target, '--sf-rich-text') ?? 'currentColor',
    accent: readCssVar(target, '--sf-accent') ?? palette[0]!,
    palette
  }
}

function seriesColor(colors: ResolvedColors, index: number): string {
  if (index === 0) return colors.accent
  return colors.palette[index % colors.palette.length]!
}

function isCartesian(kind: ChartSpec['type']): boolean {
  return kind === 'bar' || kind === 'line' || kind === 'area'
}

function chartJsType(kind: ChartSpec['type']): 'bar' | 'line' | 'pie' | 'doughnut' {
  // area 不是 chart.js 的类型：它等价于折线 + 区域填充。
  return kind === 'area' ? 'line' : kind
}

function seriesLabel(series: ChartSpec['series'][number]): string {
  // 文案由围栏作者提供；缺省留空，渲染器不自行拼本地化字符串。
  return series.name ?? ''
}

function cartesianDatasets(spec: ChartSpec, colors: ResolvedColors): Dataset[] {
  const isBar = spec.type === 'bar'
  return spec.series.map((series, index) => {
    const color = seriesColor(colors, index)
    const dataset: Dataset = {
      label: seriesLabel(series),
      data: [...series.data],
      borderColor: color,
      backgroundColor: withAlpha(color, isBar ? 0.55 : 0.18),
      borderWidth: isBar ? 1 : 2,
      pointRadius: isBar ? 0 : 2,
      tension: isBar ? 0 : 0.3,
      ...(isBar ? { borderRadius: 4, maxBarThickness: 56 } : {})
    }
    if (spec.type === 'area') dataset.fill = true
    return dataset
  })
}

function sliceDatasets(spec: ChartSpec, colors: ResolvedColors): Dataset[] {
  return spec.series.map((series) => ({
    label: seriesLabel(series),
    data: [...series.data],
    backgroundColor: series.data.map((_, sliceIndex) => withAlpha(seriesColor(colors, sliceIndex), 0.8)),
    borderColor: colors.text,
    borderWidth: 1
  }))
}

function buildOptions(spec: ChartSpec, colors: ResolvedColors, target: HTMLElement, cartesian: boolean): Record<string, unknown> {
  const options: Record<string, unknown> = {
    responsive: true,
    maintainAspectRatio: false,
    plugins: {
      legend: {
        display: cartesian ? spec.series.length > 1 : true,
        labels: { color: colors.text, boxWidth: 12 }
      },
      title: spec.title
        ? { display: true, text: spec.title, color: colors.text, font: { size: 14, weight: '600' } }
        : { display: false }
    }
  }

  if (prefersReducedMotion(target)) options.animation = false

  if (cartesian) {
    const stacked = spec.stacked === true
    const grid = { color: withAlpha(colors.text, 0.08) }
    options.scales = {
      x: { stacked, ticks: { color: colors.text }, grid },
      y: {
        stacked,
        beginAtZero: true,
        ticks: { color: colors.text },
        grid,
        title: spec.yLabel
          ? { display: true, text: spec.yLabel, color: colors.text }
          : { display: false }
      }
    }
  }

  return options
}

function buildConfig(spec: ChartSpec, colors: ResolvedColors, target: HTMLElement): Record<string, unknown> {
  const cartesian = isCartesian(spec.type)
  return {
    type: chartJsType(spec.type),
    data: {
      labels: [...spec.labels],
      datasets: cartesian ? cartesianDatasets(spec, colors) : sliceDatasets(spec, colors)
    },
    options: buildOptions(spec, colors, target, cartesian)
  }
}

/** 销毁 target 上已登记的图表实例，并移除它自己的 canvas。 */
function dispose(target: HTMLElement): void {
  const handle = handles.get(target)
  if (!handle) return
  handles.delete(target)
  try {
    handle.signal.removeEventListener('abort', handle.onAbort)
    handle.chart.destroy()
    handle.canvas.remove()
  } catch {
    // 实例或节点已失效时清理会抛；渲染器契约要求不向外传播异常。
  }
}

export const renderChart: RichRenderer = async (source, target, context) => {
  if (context.signal.aborted) return { ok: false, reason: 'error' }

  const parsed = parseChartSpec(source)
  if (!parsed.ok) return { ok: false, reason: parsed.reason }

  // 重复渲染：先拆掉上一个实例，保证 target 里只有一个 canvas。
  dispose(target)

  let canvas: HTMLCanvasElement | null = null
  try {
    canvas = target.ownerDocument.createElement('canvas')
    canvas.style.display = 'block'
    canvas.style.width = '100%'
    // 固定高度：maintainAspectRatio:false 时 chart.js 会跟随容器，容器没有高度就会
    // 被撑成整屏高，三根柱子占满一屏。
    canvas.style.height = CANVAS_HEIGHT
    target.appendChild(canvas)
    // 收窄成非空常量：abort 回调是闭包，TS 不会保留上面的赋值收窄。
    const canvasEl: HTMLCanvasElement = canvas

    const colors = resolveColors(target, context.colorMode)
    const config = buildConfig(parsed.spec, colors, target)

    const chartModule = await import('chart.js/auto')
    // 动态 import 期间页面可能已经卸载：abort 后不得留下 canvas。
    if (context.signal.aborted) {
      canvasEl.remove()
      return { ok: false, reason: 'error' }
    }

    const ChartCtor = chartModule.Chart as unknown as ChartConstructor
    const chart = new ChartCtor(canvasEl, config)
    // cleanup 会 abort 这一轮 pass，但已画好的图不该因此消失：只有 canvas 真的离开
    // 文档（重渲染把整个 wrapper 换掉）才销毁实例。延迟到微任务，是因为 rerender
    // 会先 abort、再同步拆 wrapper，直接判断会看到「还连着」的旧状态。
    const onAbort = () => {
      queueMicrotask(() => {
        if (!canvasEl.isConnected) dispose(target)
      })
    }
    context.signal.addEventListener('abort', onAbort, { once: true })
    handles.set(target, { chart, canvas: canvasEl, signal: context.signal, onAbort })
    return { ok: true }
  } catch {
    canvas?.remove()
    return { ok: false, reason: 'error' }
  }
}
