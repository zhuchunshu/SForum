// `mermaid` 围栏：richContentMermaid 渲染器的行为测试。
//
// happy-dom 没有真实 SVG 排版（getBBox / getComputedTextLength 恒为 0，canvas 没有
// 2D 上下文），mermaid 的布局引擎无法在其中产出几何；mermaid 内置的 DOMPurify 也
// 依赖浏览器 HTML 解析语义，在 happy-dom 下会漏掉 <script>。这里沿用
// richContentChart 测试的同一约定，用 mock.module 替换两个重依赖，断言本轨道负责的
// 那一层：安全配置、字节上限、原因码映射、清洗兜底与 DOM 生命周期。真实 mermaid
// 渲染放在浏览器 QA 里验证，不在单测里伪造。依赖是否可解析由第一条用例单独证明。
import { afterAll, afterEach, expect, mock, test } from 'bun:test'
import { richFenceLimit } from '../../app/utils/content/richContentLimits'
import { SECURE_KEYS } from '../../app/utils/content/richContentMermaid'
import type { RichRenderContext } from '../../app/utils/content/richContentTypes'

type MermaidConfig = Record<string, unknown>

const configs: MermaidConfig[] = []
const renders: Array<{ id: string, source: string }> = []
const effectiveSecurity: string[] = []
const purifyCalls: Array<{ html: string, config: unknown }> = []

let nextFailure: Error | null = null
let deferred: { promise: Promise<void>, release: () => void } | null = null

/** 取出源码里的 %%{init: {...}}%% 指令；解析失败时视为没有指令。 */
function directive(source: string): MermaidConfig {
  const match = /%%\{init:\s*(\{[\s\S]*?\})\s*\}%%/.exec(source)
  if (!match) return {}
  try {
    return JSON.parse(match[1]!) as MermaidConfig
  } catch {
    return {}
  }
}

/** 模拟 mermaid 的语义：secure 数组里的键不接受 %%{init}%% 覆盖。 */
function effectiveConfig(source: string): MermaidConfig {
  const base = configs[configs.length - 1] ?? {}
  const secure = Array.isArray(base.secure) ? base.secure as string[] : []
  const merged: MermaidConfig = { ...base }
  for (const [key, value] of Object.entries(directive(source))) {
    if (!secure.includes(key)) merged[key] = value
  }
  return merged
}

function escapeXml(value: string): string {
  const entities: Record<string, string> = { '<': '&lt;', '>': '&gt;', '&': '&amp;', '"': '&quot;' }
  return value.replace(/[<>&"]/g, char => entities[char]!)
}

const mermaidStub = {
  initialize(config: MermaidConfig) {
    configs.push({ ...config })
  },
  async render(id: string, source: string) {
    renders.push({ id, source })
    effectiveSecurity.push(String(effectiveConfig(source).securityLevel))
    if (deferred) await deferred.promise
    if (source.includes('%%INVALID%%')) {
      const error = new Error('Parse error on line 1: unexpected token')
      error.name = 'ParseError'
      throw error
    }
    if (nextFailure) {
      const error = nextFailure
      nextFailure = null
      throw error
    }
    if (source.includes('%%EMPTY%%')) return { svg: '' }
    if (source.includes('%%ANCHOR%%')) {
      // mermaid 11 的 click 指令会生成 <a xlink:href>；happy-dom 遇到 <script> 会吞掉
      // 后续节点，所以这里用干净 SVG 单独验证解包。
      return {
        svg: '<svg xmlns="http://www.w3.org/2000/svg"><g>'
          + '<a xlink:href="https://evil.example"><text>click me</text></a>'
          + '</g></svg>'
      }
    }
    if (source.includes('%%PAYLOAD%%')) {
      // 模拟上游（例如被 %%{init}%% 降级后的 loose 模式）把危险节点带进 SVG。
      return {
        svg: '<svg xmlns="http://www.w3.org/2000/svg"><g><script>alert(1)</script>'
          + '<rect width="10" height="10" onload="alert(1)"/>'
          + '<a xlink:href="https://evil.example"><text>click me</text></a>'
          + '<foreignObject><a href="javascript:alert(1)" xlink:href="javascript:alert(1)">x</a></foreignObject>'
          + '</g></svg>'
      }
    }
    return {
      svg: `<svg xmlns="http://www.w3.org/2000/svg" id="${id}"><g class="mindmap">`
        + `<text>${escapeXml(source)}</text></g></svg>`
    }
  }
}

// 替身只记录调用：真实 DOMPurify 在浏览器里负责第一层清洗，这里要检验的是本文件
// 自己的第二层兜底（script / on* / foreignObject href）与调用参数。
const domPurifyStub = {
  sanitize(html: string, config?: unknown) {
    purifyCalls.push({ html, config })
    return html
  }
}

mock.module('mermaid', () => ({ default: mermaidStub }))
mock.module('dompurify', () => ({ default: domPurifyStub }))

const { renderMermaid } = await import('../../app/utils/content/richContentMermaid')

const labels: RichRenderContext['labels'] = {
  diagram: '图示',
  chart: '图表',
  math: '公式',
  source: '查看源码',
  invalid: '内容无效',
  tooLarge: '内容过大',
  unsupported: '不支持的类型',
  cspBlocked: '安全策略阻止渲染',
  error: '渲染失败'
}

const MINDMAP = 'mindmap\n  root((SForum))\n    Web\n    API'

function makeTarget(): HTMLElement {
  const target = document.createElement('div')
  document.body.appendChild(target)
  return target
}

function makeContext(overrides: Partial<RichRenderContext> = {}): RichRenderContext {
  return { signal: new AbortController().signal, colorMode: 'light', cspActive: false, labels, ...overrides }
}

function lastConfig(): MermaidConfig {
  return configs[configs.length - 1]!
}

function attributesIn(root: ParentNode): Array<{ tag: string, name: string }> {
  return Array.from(root.querySelectorAll('*')).flatMap(element =>
    Array.from(element.attributes).map(attribute => ({ tag: element.tagName, name: attribute.name }))
  )
}

afterEach(() => {
  nextFailure = null
  deferred = null
  document.body.replaceChildren()
})
afterAll(() => {
  mock.restore()
})

test('mermaid 11.x 已安装且 import 说明符可解析', async () => {
  const mermaidPackage = await Bun.file(new URL('../../node_modules/mermaid/package.json', import.meta.url)).json()
  expect(String(mermaidPackage.version)).toMatch(/^11\./)
  expect(Bun.resolveSync('mermaid', import.meta.dir)).toContain('mermaid')
  expect(richFenceLimit('mermaid')).toBeDefined()
})

test('正常 mindmap 渲染出一个 <svg>，且先过 DOMPurify 的 svg profile', async () => {
  const target = makeTarget()
  const result = await renderMermaid(MINDMAP, target, makeContext())

  expect(result).toEqual({ ok: true })
  expect(target.querySelectorAll('svg')).toHaveLength(1)
  expect(target.querySelector('svg')!.textContent).toContain('SForum')

  const call = purifyCalls[purifyCalls.length - 1]!
  expect(call.html).toContain('<svg')
  expect(call.config).toEqual({ USE_PROFILES: { svg: true, svgFilters: true } })
})

test('initialize 拿到锁死的安全配置，maxTextSize 取自围栏上限', async () => {
  const target = makeTarget()
  await renderMermaid(MINDMAP, target, makeContext({ colorMode: 'light' }))

  const config = lastConfig()
  expect(config.securityLevel).toBe('strict')
  expect(config.htmlLabels).toBe(false)
  expect(config.startOnLoad).toBe(false)
  expect(config.suppressErrorRendering).toBe(true)
  expect(config.maxTextSize).toBe(richFenceLimit('mermaid')!.maxSourceBytes)
  expect(typeof config.fontFamily).toBe('string')
  expect(config.fontFamily).not.toBe('inherit')
  expect(config.theme).toBe('base')
  expect(config.themeVariables?.primaryColor).toBe('#eef2f6')
  for (const key of SECURE_KEYS) expect(config.secure).toContain(key)

  await renderMermaid(MINDMAP, makeTarget(), makeContext({ colorMode: 'dark' }))
  expect(lastConfig().theme).toBe('base')
  expect(lastConfig().themeVariables?.primaryColor).toBe('#1f2a28')
})

test('同一主题只初始化一次，切换主题才重放 initialize', async () => {
  await renderMermaid(MINDMAP, makeTarget(), makeContext({ colorMode: 'dark' }))
  const afterFirst = configs.length
  await renderMermaid(MINDMAP, makeTarget(), makeContext({ colorMode: 'dark' }))
  expect(configs.length).toBe(afterFirst)

  await renderMermaid(MINDMAP, makeTarget(), makeContext({ colorMode: 'light' }))
  expect(configs.length).toBe(afterFirst + 1)
  expect(lastConfig().themeVariables?.primaryColor).toBe('#eef2f6')
})

test('%%{init}%% 里的 securityLevel 被 secure 锁住，不能降级成 loose', async () => {
  const target = makeTarget()
  const source = `%%{init: {"securityLevel": "loose", "htmlLabels": true}}%%\n${MINDMAP}`
  const result = await renderMermaid(source, target, makeContext())

  expect(result).toEqual({ ok: true })
  // 源码原样交给 mermaid，靠 initialize 的 secure 声明锁死，而不是自己删指令。
  expect(renders[renders.length - 1]!.source).toContain('securityLevel')
  expect(effectiveSecurity[effectiveSecurity.length - 1]).toBe('strict')
  expect(lastConfig().securityLevel).toBe('strict')
  expect(lastConfig().htmlLabels).toBe(false)
  expect(target.querySelector('script')).toBeNull()
})

test('%%{init}%% 改不动主题类键：themeCSS / themeVariables 不能写进 SVG 的 <style>', async () => {
  const source = `%%{init: {"theme": "forest", "themeVariables": {"primaryColor": "red"}, "themeCSS": "rect{fill:red}", "fontFamily": "serif"}}%%\n${MINDMAP}`
  const result = await renderMermaid(source, makeTarget(), makeContext({ colorMode: 'light' }))

  expect(result).toEqual({ ok: true })
  const effective = effectiveConfig(source)
  expect(effective.theme).toBe('base')
  // 站点主题变量仍在，但指令里的 {"primaryColor":"red"} 已被 secure 丢弃。
  expect((effective.themeVariables as Record<string, unknown>)?.primaryColor).toBe('#eef2f6')
  expect(effective.themeCSS).toBeUndefined()
  expect(effective.fontFamily).not.toBe('serif')
})

test('上游 SVG 里的 script / on* / foreignObject href 不会落进 target', async () => {
  const target = makeTarget()
  target.innerHTML = '<p>旧内容</p>'
  const result = await renderMermaid(`%%PAYLOAD%%\n${MINDMAP}`, target, makeContext())

  expect(result).toEqual({ ok: true })
  expect(target.querySelector('script')).toBeNull()
  expect(target.querySelector('p')).toBeNull()
  // click 生成的 <a> 会被解包：图内链接绕过正文的站外中转与 rel 策略。
  expect(target.querySelector('a')).toBeNull()
  const attributes = attributesIn(target)
  expect(attributes.filter(attribute => attribute.name.toLowerCase().startsWith('on'))).toHaveLength(0)
  expect(attributes.filter(attribute => attribute.name.toLowerCase().endsWith('href'))).toHaveLength(0)
})

test('click 生成的图内链接被解包，文字保留', async () => {
  const target = makeTarget()
  const result = await renderMermaid(`%%ANCHOR%%\n${MINDMAP}`, target, makeContext())

  expect(result).toEqual({ ok: true })
  expect(target.querySelector('a')).toBeNull()
  expect(target.querySelectorAll('text')).toHaveLength(1)
  expect(target.textContent).toContain('click me')
})

test('超过 maxSourceBytes 返回 too_large，不调用 render', async () => {
  const target = makeTarget()
  target.innerHTML = '<p>保留</p>'
  const before = renders.length
  const oversized = `flowchart TD\n${'A-->B\n'.repeat(3_000)}`

  expect(await renderMermaid(oversized, target, makeContext())).toEqual({ ok: false, reason: 'too_large' })
  expect(renders.length).toBe(before)
  expect(target.innerHTML).toBe('<p>保留</p>')
})

test('cspActive 直接返回 csp，不做渲染', async () => {
  const target = makeTarget()
  const before = renders.length
  expect(await renderMermaid(MINDMAP, target, makeContext({ cspActive: true }))).toEqual({ ok: false, reason: 'csp' })
  expect(renders.length).toBe(before)
  expect(target.childElementCount).toBe(0)
})

test('解析失败是 invalid，空 SVG 也是 invalid，都不抛异常', async () => {
  const target = makeTarget()
  await expect(renderMermaid(`%%INVALID%%\n${MINDMAP}`, target, makeContext()))
    .resolves.toEqual({ ok: false, reason: 'invalid' })
  await expect(renderMermaid(`%%EMPTY%%\n${MINDMAP}`, target, makeContext()))
    .resolves.toEqual({ ok: false, reason: 'invalid' })
  expect(target.childElementCount).toBe(0)
})

test('非解析异常返回 error', async () => {
  nextFailure = new Error('Could not find a suitable point for the given distance')
  const target = makeTarget()
  await expect(renderMermaid(MINDMAP, target, makeContext())).resolves.toEqual({ ok: false, reason: 'error' })
  expect(target.childElementCount).toBe(0)
})

test('已 abort 时不渲染、不写 DOM', async () => {
  const target = makeTarget()
  target.innerHTML = '<p>保留</p>'
  const before = renders.length
  const controller = new AbortController()
  controller.abort()

  expect(await renderMermaid(MINDMAP, target, makeContext({ signal: controller.signal })))
    .toEqual({ ok: false, reason: 'error' })
  expect(renders.length).toBe(before)
  expect(target.innerHTML).toBe('<p>保留</p>')
})

test('渲染途中 abort：迟到的 SVG 不写进 target', async () => {
  const target = makeTarget()
  target.innerHTML = '<p>保留</p>'
  let release!: () => void
  const promise = new Promise<void>((resolve) => {
    release = resolve
  })
  deferred = { promise, release }

  const controller = new AbortController()
  const pending = renderMermaid(MINDMAP, target, makeContext({ signal: controller.signal }))
  controller.abort()
  release()

  expect(await pending).toEqual({ ok: false, reason: 'error' })
  expect(target.innerHTML).toBe('<p>保留</p>')
})

test('重复渲染先清空 target，并给 mermaid 传唯一 id', async () => {
  const target = makeTarget()
  target.innerHTML = '<p>旧内容</p>'
  await renderMermaid(MINDMAP, target, makeContext())
  await renderMermaid(MINDMAP, target, makeContext())

  expect(target.querySelectorAll('svg')).toHaveLength(1)
  const ids = renders.slice(-2).map(entry => entry.id)
  expect(new Set(ids).size).toBe(2)
})
