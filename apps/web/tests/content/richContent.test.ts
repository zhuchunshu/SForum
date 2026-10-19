import { describe, expect, test } from 'bun:test'
import { Window } from 'happy-dom'
import { enhanceRichContent, rerenderRichContent } from '../../app/utils/content/richContent'
import type {
  RichContentOptions,
  RichRenderContext,
  RichRenderFailure,
  RichRenderer
} from '../../app/utils/content/richContentTypes'

const labels = {
  diagram: '图表',
  chart: '统计图',
  math: '公式',
  source: '查看源码',
  invalid: '图表语法有误，已显示源码',
  tooLarge: '图表超出大小限制，已显示源码',
  unsupported: '暂不支持该图表类型，已显示源码',
  cspBlocked: '当前页面安全策略下以源码显示',
  error: '图表渲染失败，已显示源码'
}

type Kind = 'mermaid' | 'chart' | 'math'

type Call = {
  kind: Kind
  source: string
  target: HTMLElement
  context: RichRenderContext
}

const tick = () => new Promise<void>((resolve) => { globalThis.setTimeout(resolve, 0) })

function setup() {
  const browser = new Window({ url: 'http://localhost/' })
  const root = browser.document.createElement('div')
  browser.document.body.append(root)
  return { browser, root, doc: browser.document }
}

/** 模拟 enhanceCodeBlocks 的产物：figure.sf-code-block[data-language] > pre > code。 */
function codeFigure(doc: Document, language: string, source: string) {
  const figure = doc.createElement('figure')
  figure.className = 'sf-code-block'
  figure.setAttribute('data-language', language)
  const pre = doc.createElement('pre')
  pre.className = 'sf-code-pre'
  const code = doc.createElement('code')
  code.textContent = source
  pre.append(code)
  figure.append(pre)
  return figure
}

function okRenderer(calls: Call[], kind: Kind, html: string): RichRenderer {
  return async (source, target, context) => {
    calls.push({ kind, source, target, context })
    target.innerHTML = html
    return { ok: true }
  }
}

function failingRenderer(calls: Call[], kind: Kind, reason: RichRenderFailure): RichRenderer {
  return async (source, target, context) => {
    calls.push({ kind, source, target, context })
    return { ok: false, reason }
  }
}

function options(calls: Call[], overrides: Partial<RichContentOptions> = {}): RichContentOptions {
  return {
    labels,
    colorMode: () => 'light',
    cspActive: () => false,
    renderers: {
      mermaid: okRenderer(calls, 'mermaid', '<svg></svg>'),
      chart: okRenderer(calls, 'chart', '<canvas></canvas>'),
      math: okRenderer(calls, 'math', '<math><mi>x</mi></math>')
    },
    ...overrides
  }
}

describe('rich content orchestration', () => {
  test('upgrades a mermaid fence into figure.sf-rich and keeps the source in details', async () => {
    const { root, doc } = setup()
    const source = 'graph TD\n  A --> B'
    root.append(codeFigure(doc, 'mermaid', source))
    const calls: Call[] = []

    enhanceRichContent(root, options(calls))
    await tick()

    const wrapper = root.querySelector('figure.sf-rich')
    expect(wrapper?.getAttribute('data-sf-rich')).toBe('mermaid')
    expect(wrapper?.getAttribute('data-sf-rich-state')).toBe('ready')

    const canvas = wrapper?.querySelector('.sf-rich__canvas')
    expect(canvas?.getAttribute('role')).toBe('img')
    expect(canvas?.getAttribute('aria-label')).toBe(labels.diagram)
    expect(canvas?.querySelector('svg')).not.toBeNull()

    const details = wrapper?.querySelector('details.sf-rich__source')
    expect(details?.querySelector('summary')?.textContent).toBe(labels.source)
    expect(details?.querySelector('figure.sf-code-block[data-language="mermaid"]')).not.toBeNull()
    expect(details?.querySelector('pre code')?.textContent).toBe(source)

    expect(wrapper?.querySelector('.sf-rich__note')?.hasAttribute('hidden')).toBe(true)
    expect(calls).toHaveLength(1)
    expect(calls[0]!.source).toBe(source)
    expect(calls[0]!.context.colorMode).toBe('light')
    expect(calls[0]!.context.cspActive).toBe(false)
  })

  test('labels a chart fence as a chart and renders into its own canvas', async () => {
    const { root, doc } = setup()
    root.append(codeFigure(doc, 'chart', '{"type":"bar","labels":["a"],"series":[{"data":[1]}]}'))
    const calls: Call[] = []

    enhanceRichContent(root, options(calls))
    await tick()

    const wrapper = root.querySelector('figure.sf-rich')
    expect(wrapper?.getAttribute('data-sf-rich')).toBe('chart')
    expect(wrapper?.querySelector('.sf-rich__canvas')?.getAttribute('aria-label')).toBe(labels.chart)
    expect(wrapper?.querySelector('canvas')).not.toBeNull()
    expect(calls).toHaveLength(1)
    expect(calls[0]!.kind).toBe('chart')
  })

  test('leaves fences with an unknown language untouched', async () => {
    const { root, doc } = setup()
    root.append(codeFigure(doc, 'python', 'print(1)'))
    const calls: Call[] = []

    enhanceRichContent(root, options(calls))
    await tick()

    expect(root.querySelector('.sf-rich')).toBeNull()
    expect(root.querySelector('figure.sf-code-block')?.getAttribute('data-language')).toBe('python')
    expect(calls).toHaveLength(0)
  })

  test('falls back to source on the seventh same-kind fence in one document', async () => {
    const { root, doc } = setup()
    for (let index = 0; index < 7; index += 1) {
      root.append(codeFigure(doc, 'mermaid', `graph TD\n  A${index} --> B`))
    }
    const calls: Call[] = []

    enhanceRichContent(root, options(calls))
    await tick()

    const wrappers = root.querySelectorAll('figure.sf-rich')
    expect(wrappers).toHaveLength(7)
    expect(wrappers[6]!.getAttribute('data-sf-rich-state')).toBe('fallback')
    expect(wrappers[6]!.querySelector('.sf-rich__note')?.textContent).toBe(labels.tooLarge)
    expect(calls).toHaveLength(6)
  })

  test('falls back without rendering when the source exceeds the byte limit', async () => {
    const { root, doc } = setup()
    root.append(codeFigure(doc, 'mermaid', 'x'.repeat(12_289)))
    const calls: Call[] = []

    enhanceRichContent(root, options(calls))
    await tick()

    const wrapper = root.querySelector('figure.sf-rich')
    expect(wrapper?.getAttribute('data-sf-rich-state')).toBe('fallback')
    expect(wrapper?.querySelector('.sf-rich__note')?.textContent).toBe(labels.tooLarge)
    expect(wrapper?.querySelector('details.sf-rich__source')?.hasAttribute('open')).toBe(true)
    expect(wrapper?.querySelector('figure.sf-code-block')).not.toBeNull()
    expect(calls).toHaveLength(0)
  })

  test('writes the matching note when a renderer fails', async () => {
    const { root, doc } = setup()
    root.append(codeFigure(doc, 'mermaid', 'graph TD'))
    root.append(codeFigure(doc, 'chart', '{"type":"bar"}'))
    const calls: Call[] = []

    enhanceRichContent(root, options(calls, {
      renderers: {
        mermaid: failingRenderer(calls, 'mermaid', 'invalid'),
        chart: failingRenderer(calls, 'chart', 'csp')
      }
    }))
    await tick()

    const [mermaid, chart] = root.querySelectorAll('figure.sf-rich')
    expect(mermaid!.getAttribute('data-sf-rich-state')).toBe('fallback')
    expect(mermaid!.querySelector('.sf-rich__note')?.textContent).toBe(labels.invalid)
    expect(mermaid!.querySelector('.sf-rich__canvas')?.childElementCount).toBe(0)
    expect(chart!.getAttribute('data-sf-rich-state')).toBe('fallback')
    expect(chart!.querySelector('.sf-rich__note')?.textContent).toBe(labels.cspBlocked)
  })

  test('turns inline math into span.sf-rich-math and leaves code blocks alone', async () => {
    const { root, doc } = setup()
    const paragraph = doc.createElement('p')
    paragraph.textContent = '复杂度是 $O(n\\log n)$ 级别'
    const pre = doc.createElement('pre')
    const code = doc.createElement('code')
    code.textContent = 'echo $HOME'
    pre.append(code)
    root.append(paragraph, pre)
    const calls: Call[] = []

    enhanceRichContent(root, options(calls))
    await tick()

    const spans = paragraph.querySelectorAll('span.sf-rich-math')
    expect(spans).toHaveLength(1)
    expect(spans[0]!.getAttribute('data-sf-rich-state')).toBe('ready')
    expect(spans[0]!.hasAttribute('data-math-display')).toBe(false)
    expect(spans[0]!.querySelector('math')).not.toBeNull()
    expect(paragraph.textContent).toContain('复杂度是')
    expect(paragraph.textContent).toContain('级别')

    expect(pre.querySelector('span.sf-rich-math')).toBeNull()
    expect(code.textContent).toBe('echo $HOME')

    const mathCalls = calls.filter(call => call.kind === 'math')
    expect(mathCalls).toHaveLength(1)
    expect(mathCalls[0]!.source).toBe('O(n\\log n)')
  })

  test('marks display math with data-math-display so renderers can pick display mode', async () => {
    const { root, doc } = setup()
    const paragraph = doc.createElement('p')
    paragraph.textContent = '$$\\sum_{i=1}^{n} i$$'
    root.append(paragraph)
    const calls: Call[] = []

    enhanceRichContent(root, options(calls))
    await tick()

    const span = paragraph.querySelector('span.sf-rich-math')
    expect(span?.getAttribute('data-math-display')).toBe('1')
    expect(span?.getAttribute('data-sf-rich-state')).toBe('ready')
  })

  test('is idempotent across repeated passes', async () => {
    const { root, doc } = setup()
    root.append(codeFigure(doc, 'mermaid', 'graph TD'))
    const paragraph = doc.createElement('p')
    paragraph.textContent = '设 $n$ 为规模'
    root.append(paragraph)
    const calls: Call[] = []

    const first = enhanceRichContent(root, options(calls))
    await tick()
    enhanceRichContent(root, options(calls))
    await tick()
    first()

    expect(root.querySelectorAll('figure.sf-rich')).toHaveLength(1)
    expect(root.querySelectorAll('span.sf-rich-math')).toHaveLength(1)
    expect(calls.filter(call => call.kind === 'mermaid')).toHaveLength(1)
    expect(calls.filter(call => call.kind === 'math')).toHaveLength(1)
  })

  test('stops writing DOM after cleanup', async () => {
    const { root, doc } = setup()
    root.append(codeFigure(doc, 'mermaid', 'graph TD'))
    const calls: Call[] = []
    let release = () => {}
    const gate = new Promise<void>((resolve) => { release = resolve })
    const renderer: RichRenderer = async (source, target, context) => {
      calls.push({ kind: 'mermaid', source, target, context })
      await gate
      if (context.signal.aborted) return { ok: false, reason: 'error' }
      target.innerHTML = '<svg></svg>'
      return { ok: true }
    }

    const cleanup = enhanceRichContent(root, options(calls, { renderers: { mermaid: renderer } }))
    const wrapper = root.querySelector('figure.sf-rich')!
    expect(wrapper.getAttribute('data-sf-rich-state')).toBe('pending')

    cleanup()
    release()
    await tick()

    expect(wrapper.getAttribute('data-sf-rich-state')).toBe('pending')
    expect(wrapper.querySelector('.sf-rich__canvas')?.childElementCount).toBe(0)
  })

  test('adopts a pending math span after the previous pass was cleaned up', async () => {
    const { root, doc } = setup()
    const paragraph = doc.createElement('p')
    paragraph.textContent = '设 $n$ 为规模'
    root.append(paragraph)
    const calls: Call[] = []
    let release = () => {}
    const gate = new Promise<void>((resolve) => { release = resolve })
    const renderer: RichRenderer = async (source, target, context) => {
      calls.push({ kind: 'math', source, target, context })
      await gate
      if (context.signal.aborted) return { ok: false, reason: 'error' }
      target.innerHTML = '<math><mi>n</mi></math>'
      return { ok: true }
    }

    const cleanup = enhanceRichContent(root, options(calls, { renderers: { math: renderer } }))
    const span = paragraph.querySelector('span.sf-rich-math')!
    cleanup()
    release()
    await tick()
    expect(span.getAttribute('data-sf-rich-state')).toBe('pending')

    enhanceRichContent(root, options(calls, { renderers: { math: renderer } }))
    await tick()

    expect(span.getAttribute('data-sf-rich-state')).toBe('ready')
    expect(calls).toHaveLength(2)
    expect(span.querySelector('math')).not.toBeNull()
  })

  test('lazily renders through IntersectionObserver with a 300px root margin', async () => {
    const { root, doc } = setup()
    root.append(codeFigure(doc, 'mermaid', 'graph TD'))
    const calls: Call[] = []
    const observed: Array<{ target: Element, options?: IntersectionObserverInit }> = []
    let notify: ((entries: Array<{ isIntersecting: boolean, target: Element }>) => void) | null = null

    class FakeObserver {
      readonly options?: IntersectionObserverInit
      constructor(callback: (entries: never) => void, init?: IntersectionObserverInit) {
        this.options = init
        notify = callback as unknown as typeof notify
      }
      observe(target: Element) { observed.push({ target, options: this.options }) }
      unobserve() {}
      disconnect() {}
    }

    const previous = (globalThis as Record<string, unknown>).IntersectionObserver
    ;(globalThis as Record<string, unknown>).IntersectionObserver = FakeObserver
    try {
      enhanceRichContent(root, options(calls))
      await tick()
      expect(calls).toHaveLength(0)
      expect(observed).toHaveLength(1)
      expect(observed[0]!.options?.rootMargin).toBe('300px')

      notify!([{ isIntersecting: true, target: observed[0]!.target }])
      await tick()
      expect(calls).toHaveLength(1)
      expect(root.querySelector('figure.sf-rich')?.getAttribute('data-sf-rich-state')).toBe('ready')
    } finally {
      (globalThis as Record<string, unknown>).IntersectionObserver = previous
    }
  })

  test('clears the SSR skeleton flag once the pass takes over', async () => {
    const { root, doc } = setup()
    doc.documentElement.dataset.sfRich = 'pending'
    root.append(codeFigure(doc, 'mermaid', 'graph TD'))
    const calls: Call[] = []

    enhanceRichContent(root, options(calls))

    expect(doc.documentElement.dataset.sfRich).toBe('ready')
  })

  test('rerenders fences with the current color mode', async () => {
    const { root, doc } = setup()
    root.append(codeFigure(doc, 'mermaid', 'graph TD'))
    const calls: Call[] = []
    let mode: 'light' | 'dark' = 'light'
    const richOptions = options(calls, { colorMode: () => mode })

    enhanceRichContent(root, richOptions)
    await tick()
    expect(calls).toHaveLength(1)
    expect(calls[0]!.context.colorMode).toBe('light')

    mode = 'dark'
    rerenderRichContent(root, richOptions)
    await tick()

    expect(calls).toHaveLength(2)
    expect(calls[1]!.context.colorMode).toBe('dark')
    expect(root.querySelectorAll('figure.sf-rich')).toHaveLength(1)
    expect(root.querySelectorAll('figure.sf-code-block')).toHaveLength(1)
  })
})
