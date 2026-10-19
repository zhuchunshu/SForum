// `chart` 围栏：解析器（chartSpec）与 Chart.js 渲染器（richContentChart）的行为测试。
//
// happy-dom 没有真实的 canvas 2D 上下文，Chart.js 无法真正绘制。这里用
// mock.module 替换 'chart.js/auto'，断言的是我们自己负责的那一层：类型映射、
// 颜色/动画选项、canvas 生命周期、重复渲染与 abort 行为。真实依赖是否可解析
// 由第一条用例单独验证（真实 import 发生在 mock 注册之前）。
import { afterAll, afterEach, expect, mock, test } from 'bun:test'
import { parseChartSpec } from '../../app/utils/content/chartSpec'
import type { RichRenderContext } from '../../app/utils/content/richContentTypes'

type ChartConfigLike = {
  type: string
  data: { labels: string[], datasets: Record<string, any>[] }
  options: Record<string, any>
}

const createdCharts: FakeChart[] = []
let failConstruction = false

class FakeChart {
  canvas: HTMLCanvasElement
  config: ChartConfigLike
  destroyed = false

  constructor(canvas: HTMLCanvasElement, config: ChartConfigLike) {
    if (failConstruction) throw new Error('no 2d context')
    this.canvas = canvas
    this.config = config
    createdCharts.push(this)
  }

  destroy() {
    this.destroyed = true
  }
}

mock.module('chart.js/auto', () => ({ Chart: FakeChart, default: FakeChart }))

const { renderChart } = await import('../../app/utils/content/richContentChart')

const labels: RichRenderContext['labels'] = {
  diagram: '图示',
  chart: '图表',
  math: '公式',
  source: '查看源码',
  invalid: '内容无效',
  tooLarge: '内容过大',
  unsupported: '不支持的图表类型',
  cspBlocked: '安全策略阻止渲染',
  error: '渲染失败'
}

const TYPES = ['bar', 'line', 'area', 'pie', 'doughnut'] as const

function makeTarget(): HTMLElement {
  const target = document.createElement('div')
  document.body.appendChild(target)
  return target
}

function makeContext(overrides: Partial<RichRenderContext> = {}): RichRenderContext {
  return { signal: new AbortController().signal, colorMode: 'light', cspActive: false, labels, ...overrides }
}

function chartSource(type: string, extra: Record<string, unknown> = {}): string {
  return JSON.stringify({
    type,
    labels: ['一月', '二月', '三月'],
    series: [{ name: '访问量', data: [12, 18, 9] }],
    ...extra
  })
}

function lastChart(): FakeChart {
  return createdCharts[createdCharts.length - 1]!
}

const originalMatchMedia = window.matchMedia
afterEach(() => {
  failConstruction = false
  ;(window as unknown as { matchMedia: unknown }).matchMedia = originalMatchMedia
  document.body.replaceChildren()
})
afterAll(() => {
  mock.restore()
})

test('chart.js 4.x 已安装且 import 说明符可解析', async () => {
  // 真实模块在 mock 注册之后会被替换，因此这里用包清单 + 解析结果证明依赖存在，
  // 而不是靠被顶替的模块对象。
  const chartPackage = await Bun.file(new URL('../../node_modules/chart.js/package.json', import.meta.url)).json()
  expect(String(chartPackage.version)).toMatch(/^4\./)
  const resolved = Bun.resolveSync('chart.js/auto', import.meta.dir)
  expect(resolved).toContain('chart.js')
})

test('五种契约类型都被接受，并返回互不共享引用的新对象', () => {
  for (const type of TYPES) {
    const result = parseChartSpec(chartSource(type))
    expect(result.ok).toBe(true)
  }
  const source = chartSource('bar')
  const first = parseChartSpec(source)
  const second = parseChartSpec(source)
  expect(first.ok && second.ok).toBe(true)
  if (!first.ok || !second.ok) return
  expect(first.spec).not.toBe(second.spec)
  expect(first.spec.labels).not.toBe(second.spec.labels)
  expect(first.spec.series[0]).not.toBe(second.spec.series[0])
  expect(first.spec.series[0]!.data).not.toBe(second.spec.series[0]!.data)
  expect(first.spec).toEqual(second.spec)
})

test('未知类型是 unsupported，结构损坏是 invalid', () => {
  expect(parseChartSpec(chartSource('radar'))).toEqual({ ok: false, reason: 'unsupported' })
  expect(parseChartSpec('{')).toEqual({ ok: false, reason: 'invalid' })
  expect(parseChartSpec('[]')).toEqual({ ok: false, reason: 'invalid' })
  expect(parseChartSpec('null')).toEqual({ ok: false, reason: 'invalid' })
  expect(parseChartSpec(JSON.stringify({ labels: ['a'], series: [{ data: [1] }] }))).toEqual({ ok: false, reason: 'invalid' })
  expect(parseChartSpec(JSON.stringify({ type: 7, labels: ['a'], series: [{ data: [1] }] }))).toEqual({ ok: false, reason: 'invalid' })
  expect(parseChartSpec(chartSource('bar', { theme: 'dark' }))).toEqual({ ok: false, reason: 'invalid' })
  expect(parseChartSpec(chartSource('bar', { stacked: 'yes' }))).toEqual({ ok: false, reason: 'invalid' })
})

test('未知键、长度不匹配与越界数值都被拒绝', () => {
  const withUnknownSeriesKey = JSON.stringify({
    type: 'bar',
    labels: ['a'],
    series: [{ data: [1], color: '#fff' }]
  })
  expect(parseChartSpec(withUnknownSeriesKey)).toEqual({ ok: false, reason: 'invalid' })
  expect(parseChartSpec(JSON.stringify({
    type: 'bar',
    labels: ['a', 'b'],
    series: [{ data: [1] }]
  }))).toEqual({ ok: false, reason: 'invalid' })
  expect(parseChartSpec('{"type":"bar","labels":["a"],"series":[{"data":[null]}]}')).toEqual({ ok: false, reason: 'invalid' })
  expect(parseChartSpec('{"type":"bar","labels":["a"],"series":[{"data":[1e999]}]}')).toEqual({ ok: false, reason: 'invalid' })
  expect(parseChartSpec(JSON.stringify({
    type: 'bar',
    labels: ['a'],
    series: [{ data: [1_000_000_000_001] }]
  }))).toEqual({ ok: false, reason: 'invalid' })
  expect(parseChartSpec(JSON.stringify({
    type: 'bar',
    labels: Array.from({ length: 61 }, (_, index) => `L${index}`),
    series: [{ data: Array.from({ length: 61 }, () => 1) }]
  }))).toEqual({ ok: false, reason: 'invalid' })
  expect(parseChartSpec(JSON.stringify({
    type: 'bar',
    labels: ['a'],
    series: Array.from({ length: 7 }, () => ({ data: [1] }))
  }))).toEqual({ ok: false, reason: 'invalid' })
  expect(parseChartSpec(chartSource('bar', { title: 'x'.repeat(121) }))).toEqual({ ok: false, reason: 'invalid' })
  expect(parseChartSpec(chartSource('bar', { yLabel: 'x'.repeat(121) }))).toEqual({ ok: false, reason: 'invalid' })
})

test('超过 maxSourceBytes 的源码返回 too_large', () => {
  const oversized = chartSource('bar', { title: 'x'.repeat(9_000) })
  expect(parseChartSpec(oversized)).toEqual({ ok: false, reason: 'too_large' })
  expect(parseChartSpec(chartSource('bar', { stacked: true, title: '标题', yLabel: '数值' })).ok).toBe(true)
})

test('渲染后 target 内只有一个 canvas，并映射成对应的 chart.js 类型', async () => {
  const expected: Record<string, string> = { bar: 'bar', line: 'line', area: 'line', pie: 'pie', doughnut: 'doughnut' }
  for (const type of TYPES) {
    const target = makeTarget()
    const result = await renderChart(chartSource(type), target, makeContext())
    expect(result).toEqual({ ok: true })
    const chart = lastChart()
    expect(chart.config.type).toBe(expected[type]!)
    expect(chart.canvas.parentElement).toBe(target)
    expect(target.querySelectorAll('canvas')).toHaveLength(1)
    expect(chart.canvas.style.height).toBe('280px')
    expect(chart.config.options.responsive).toBe(true)
    expect(chart.config.options.maintainAspectRatio).toBe(false)
    if (type === 'area') expect(chart.config.data.datasets[0]!.fill).toBe(true)
    if (type === 'pie' || type === 'doughnut') expect(chart.config.options.scales).toBeUndefined()
    else expect(chart.config.options.scales.x.stacked).toBe(false)
  }
})

test('stacked 映射到 x/y 轴，颜色取自 CSS 变量', async () => {
  const target = makeTarget()
  target.style.setProperty('--sf-rich-text', '#123456')
  target.style.setProperty('--sf-accent', '#ff0000')
  await renderChart(chartSource('bar', { stacked: true }), target, makeContext())
  const chart = lastChart()
  const dataset = chart.config.data.datasets[0]!
  expect(chart.config.options.scales.x.stacked).toBe(true)
  expect(chart.config.options.scales.y.stacked).toBe(true)
  expect(chart.config.options.scales.x.ticks.color).toBe('#123456')
  expect(dataset.borderColor).toBe('#ff0000')
  expect(dataset.backgroundColor).toBe('rgba(255, 0, 0, 0.55)')
})

test('缺少 CSS 变量时回退到 currentColor 与按 colorMode 选定的调色板', async () => {
  const target = makeTarget()
  await renderChart(chartSource('line'), target, makeContext({ colorMode: 'dark' }))
  const chart = lastChart()
  expect(chart.config.options.scales.y.ticks.color).toBe('currentColor')
  expect(chart.config.data.datasets[0]!.borderColor).toBe('#60a5fa')
})

test('prefers-reduced-motion 时关闭动画，否则保留默认动画', async () => {
  ;(window as unknown as { matchMedia: unknown }).matchMedia = () => ({ matches: true })
  const reducedTarget = makeTarget()
  await renderChart(chartSource('line'), reducedTarget, makeContext())
  expect(lastChart().config.options.animation).toBe(false)

  ;(window as unknown as { matchMedia: unknown }).matchMedia = () => ({ matches: false })
  const animatedTarget = makeTarget()
  await renderChart(chartSource('line'), animatedTarget, makeContext())
  expect(lastChart().config.options.animation).toBeUndefined()
})

test('重复渲染先销毁旧实例，target 里不叠加 canvas', async () => {
  const target = makeTarget()
  await renderChart(chartSource('bar'), target, makeContext())
  const first = lastChart()
  await renderChart(chartSource('pie'), target, makeContext())
  const second = lastChart()
  expect(second).not.toBe(first)
  expect(first.destroyed).toBe(true)
  expect(target.querySelectorAll('canvas')).toHaveLength(1)
})

test('abort 后不写 DOM；已画好的图只在 canvas 离开文档时才销毁', async () => {
  const aborted = new AbortController()
  aborted.abort()
  const target = makeTarget()
  const result = await renderChart(chartSource('bar'), target, makeContext({ signal: aborted.signal }))
  expect(result).toEqual({ ok: false, reason: 'error' })
  expect(target.childElementCount).toBe(0)

  // 一轮 pass 被 cleanup 会 abort，但 canvas 还挂在页面上：图不能消失。
  const controller = new AbortController()
  const live = makeTarget()
  await renderChart(chartSource('bar'), live, makeContext({ signal: controller.signal }))
  const chart = lastChart()
  controller.abort()
  await Promise.resolve()
  expect(chart.destroyed).toBe(false)
  expect(live.querySelector('canvas')).not.toBeNull()

  // 重渲染把整个 wrapper 换掉后，canvas 已离开文档，abort 才销毁实例。
  const rerender = new AbortController()
  const second = makeTarget()
  await renderChart(chartSource('bar'), second, makeContext({ signal: rerender.signal }))
  const chart2 = lastChart()
  second.querySelector('canvas')?.remove()
  rerender.abort()
  await Promise.resolve()
  expect(chart2.destroyed).toBe(true)
})

test('解析失败与 Chart.js 构造失败都不留 canvas', async () => {
  const invalidTarget = makeTarget()
  expect(await renderChart('{"type":"radar"}', invalidTarget, makeContext())).toEqual({ ok: false, reason: 'unsupported' })
  expect(await renderChart('not json', invalidTarget, makeContext())).toEqual({ ok: false, reason: 'invalid' })
  expect(invalidTarget.childElementCount).toBe(0)

  failConstruction = true
  const failingTarget = makeTarget()
  expect(await renderChart(chartSource('bar'), failingTarget, makeContext())).toEqual({ ok: false, reason: 'error' })
  expect(failingTarget.querySelector('canvas')).toBeNull()
})
