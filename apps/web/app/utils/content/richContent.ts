// 富内容编排层。把 enhanceCodeBlocks 产出的 figure.sf-code-block 升级成 mermaid /
// chart 图块，并把正文里的公式定界符换成 .sf-rich-math 容器。
//
// 本文件只依赖 richContentTypes.ts 的冻结签名：三个渲染器由 A/B/C 三条轨道实现，
// 编排层不关心它们怎么画，只按 kind 取用、按结果码写状态。DOM 契约见 task-1：
//   figure.sf-rich[data-sf-rich="mermaid|chart"][data-sf-rich-state="pending|ready|fallback"]
//     > .sf-rich__canvas[role=img] + details.sf-rich__source > summary + p.sf-rich__note
import type {
  RichContentLabels,
  RichContentOptions,
  RichKind,
  RichRenderContext,
  RichRenderFailure,
  RichRenderResult,
  RichRenderer,
  RichRendererRegistry
} from './richContentTypes'
import type { RichFenceLimit } from './richContentLimits'
import {
  RICH_MATH_LIMITS,
  isRichFenceLanguage,
  richFenceLimit,
  sourceByteLength
} from './richContentLimits'
import { splitMathText } from './mathSyntax'
import { renderMermaid } from './richContentMermaid'
import { renderChart } from './richContentChart'
import { renderMath } from './richContentMath'

/** 生产默认渲染器；测试通过 options.renderers 逐 kind 覆盖。 */
const DEFAULT_RENDERERS: RichRendererRegistry = {
  mermaid: renderMermaid,
  chart: renderChart,
  math: renderMath
}

/** 围栏语言 → 渲染器 kind 与无障碍标签键（fixture：mermaid 的 kind 是 diagram，chart 是 chart）。 */
function fenceTarget(limit: RichFenceLimit): { renderer: 'mermaid' | 'chart', labelKey: 'diagram' | 'chart' } {
  return limit.kind === 'chart'
    ? { renderer: 'chart', labelKey: 'chart' }
    : { renderer: 'mermaid', labelKey: 'diagram' }
}

/** 回落原因码 → 冻结的文案键。 */
const REASON_LABELS: Record<RichRenderFailure, keyof RichContentLabels> = {
  invalid: 'invalid',
  too_large: 'tooLarge',
  unsupported: 'unsupported',
  csp: 'cspBlocked',
  error: 'error'
}

/** 公式扫描跳过的子树：代码、链接、脚本，以及已被接管的区域。 */
const SKIP_MATH = 'pre, code, a, script, style, .sf-rich, .sf-rich-math, [data-sf-rich-state]'

type FenceJob = {
  wrapper: HTMLElement
  canvas: HTMLElement
  details: HTMLDetailsElement
  note: HTMLElement
  source: string
  renderer: 'mermaid' | 'chart'
}

const noop = () => {}

// 每个 wrapper 当前归哪一轮 pass 所有。上一轮被 cleanup 中止的 pending 图块，
// 下一轮可以接手重渲染；两轮同时存在时后一轮不抢前一轮的活，避免重复渲染。
const passOwners = new WeakMap<Element, object>()

/** root 可能是 Document、Element 或 ShadowRoot；统一拿到它所属的 document。 */
function documentOf(root: ParentNode): Document | null {
  const owner = (root as { ownerDocument?: Document | null }).ownerDocument
  if (owner) return owner
  return typeof (root as Document).createElement === 'function' ? root as Document : null
}

/** 元素是否已经在视口附近（预留 300px）。没有布局信息（隐藏容器）时按不可见处理。 */
function inViewport(element: Element): boolean {
  const rect = element.getBoundingClientRect()
  if (rect.width === 0 && rect.height === 0) return false
  const height = Number(globalThis.innerHeight || 0)
  if (!height) return true
  return rect.top < height + 300 && rect.bottom > -300
}

/** 文档内已接管的同类图块数量，用于跨多次调用的 maxPerDocument 配额。 */
function countFigures(scope: ParentNode, language: string): number {
  return scope.querySelectorAll(`figure.sf-rich[data-sf-rich="${language}"]`).length
}

/**
 * 按文档顺序收集可参与公式切分的文本节点。
 * 只读遍历，先收齐再改 DOM，避免边遍历边替换。
 */
function collectTextNodes(root: ParentNode): Text[] {
  const found: Text[] = []
  const visit = (node: Node) => {
    for (let child = node.firstChild; child; child = child.nextSibling) {
      if (child.nodeType === 3) {
        found.push(child as Text)
        continue
      }
      if (child.nodeType !== 1) continue
      const element = child as Element
      if (element.matches(SKIP_MATH)) continue
      visit(element)
    }
  }
  visit(root as Node)
  return found
}

/**
 * 扫描 root 内的富内容并接管。可重复调用：已接管的节点带 data-sf-rich-state，
 * 扫描时跳过，因此 updated 钩子重复触发不会重复渲染。
 *
 * 返回 cleanup：中止所有在飞渲染并断开 IntersectionObserver。
 */
export function enhanceRichContent(
  root: ParentNode | null,
  options: RichContentOptions
): () => void {
  const doc = root ? documentOf(root) : null
  if (!root || !doc) return noop
  // 水合完成：摘掉 SSR 阶段的骨架标记（nuxt.config 的内联脚本设置），
  // 之后由每个 figure 自己的 pending 骨架接管，避免源码在首帧闪一下。
  if (doc.documentElement) doc.documentElement.dataset.sfRich = 'ready'

  const controller = new AbortController()
  const rendererFor = (kind: RichKind): RichRenderer => options.renderers?.[kind] ?? DEFAULT_RENDERERS[kind]
  const renderContext = (): RichRenderContext => ({
    signal: controller.signal,
    colorMode: options.colorMode(),
    cspActive: options.cspActive(),
    labels: options.labels
  })

  let stopped = false
  let observer: IntersectionObserver | null = null
  const jobs = new Map<Element, FenceJob>()
  const owner = {}
  const owned = new Set<Element>()

  const applyFailure = (job: FenceJob, reason: RichRenderFailure) => {
    job.canvas.replaceChildren()
    job.wrapper.dataset.sfRichState = 'fallback'
    job.details.open = true
    job.note.textContent = options.labels[REASON_LABELS[reason] ?? 'error']
    job.note.hidden = false
  }

  const paintFence = async (job: FenceJob) => {
    let result: RichRenderResult
    try {
      result = await rendererFor(job.renderer)(job.source, job.canvas, renderContext())
    } catch {
      result = { ok: false, reason: 'error' }
    }
    // cleanup 之后到达的结果一律丢弃，避免把已卸载的节点写回 ready。
    if (stopped || controller.signal.aborted) return
    if (result?.ok === true) {
      job.wrapper.dataset.sfRichState = 'ready'
      job.note.hidden = true
      return
    }
    applyFailure(job, result && result.ok === false ? result.reason : 'error')
  }

  const observe = (job: FenceJob) => {
    if (stopped) return
    owned.add(job.wrapper)
    passOwners.set(job.wrapper, owner)
    const Ctor = (globalThis as { IntersectionObserver?: typeof IntersectionObserver }).IntersectionObserver
    // 已经在视口里的图块立刻画：等 IntersectionObserver 的异步回调会留下几帧骨架，
    // 而这一轮 pass 若紧接着被 updated 收掉，在飞渲染会被 abort，图块就永远停在
    // pending（骨架不消失）。视口外的仍然懒加载。
    if (typeof Ctor !== 'function' || inViewport(job.wrapper)) {
      // 无 IO 的环境（SSR 之外的降级、单元测试）也走这里，行为与可见时一致。
      void paintFence(job)
      return
    }
    if (!observer) {
      observer = new Ctor((entries) => {
        for (const entry of entries) {
          if (!entry.isIntersecting) continue
          const target = jobs.get(entry.target)
          if (!target) continue
          jobs.delete(entry.target)
          observer?.unobserve(entry.target)
          void paintFence(target)
        }
      }, { rootMargin: '300px' })
    }
    jobs.set(job.wrapper, job)
    observer.observe(job.wrapper)
  }

  // 每种 kind 的文档级配额在扫描开始时快照一次，本轮新建的图块用局部计数累加。
  const claimed: Record<'mermaid' | 'chart', number> = {
    mermaid: countFigures(doc, 'mermaid'),
    chart: countFigures(doc, 'chart')
  }

  for (const figure of root.querySelectorAll<HTMLElement>('figure.sf-code-block[data-language]')) {
    // 已接管的源码块仍留在 details 里，重复扫描必须跳过。
    if (figure.closest('.sf-rich')) continue
    const language = (figure.getAttribute('data-language') || '').trim().toLowerCase()
    if (!isRichFenceLanguage(language)) continue
    const limit = richFenceLimit(language)
    if (!limit) continue

    const { renderer, labelKey } = fenceTarget(limit)
    const source = figure.querySelector('pre code')?.textContent ?? ''
    const withinQuota = claimed[renderer] < limit.maxPerDocument
    claimed[renderer] += 1

    const wrapper = doc.createElement('figure')
    wrapper.className = 'sf-rich'
    wrapper.dataset.sfRich = limit.language
    wrapper.dataset.sfRichState = 'pending'

    const canvas = doc.createElement('div')
    canvas.className = 'sf-rich__canvas'
    canvas.setAttribute('role', 'img')
    canvas.setAttribute('aria-label', options.labels[labelKey])

    const details = doc.createElement('details')
    details.className = 'sf-rich__source'
    const summary = doc.createElement('summary')
    summary.textContent = options.labels.source
    details.append(summary)

    const note = doc.createElement('p')
    note.className = 'sf-rich__note'
    note.hidden = true

    figure.replaceWith(wrapper)
    details.append(figure)
    wrapper.append(canvas, details, note)

    const job: FenceJob = { wrapper, canvas, details, note, source, renderer }
    // 用原代码块的高度做骨架高度，图渲染出来时不会把下面的内容顶下去。
    const sourceHeight = Math.round(figure.getBoundingClientRect().height)
    if (sourceHeight > 0) canvas.style.minHeight = `${Math.min(480, Math.max(120, sourceHeight))}px`
    if (!withinQuota || sourceByteLength(source) > limit.maxSourceBytes) {
      // 超配额或超字节：不渲染，直接以源码形态回落。
      applyFailure(job, 'too_large')
      continue
    }
    observe(job)
  }

  // 接手上一轮被 cleanup 中止、仍停在 pending 的图块。插件在 updated 里先 cleanup
  // 再重跑，这条路径保证正在渲染的图块不会永远停在空画布上。
  for (const wrapper of root.querySelectorAll<HTMLElement>('figure.sf-rich[data-sf-rich][data-sf-rich-state="pending"]')) {
    if (stopped || passOwners.has(wrapper)) continue
    const limit = richFenceLimit(wrapper.dataset.sfRich || '')
    if (!limit) continue
    const sourceFigure = wrapper.querySelector<HTMLElement>('figure.sf-code-block')
    const canvas = wrapper.querySelector<HTMLElement>('.sf-rich__canvas')
    const details = wrapper.querySelector<HTMLDetailsElement>('details.sf-rich__source')
    const note = wrapper.querySelector<HTMLElement>('.sf-rich__note')
    if (!sourceFigure || !canvas || !details || !note) continue
    const source = sourceFigure.querySelector('pre code')?.textContent ?? ''
    if (sourceByteLength(source) > limit.maxSourceBytes) {
      applyFailure({ wrapper, canvas, details, note, source, renderer: fenceTarget(limit).renderer }, 'too_large')
      continue
    }
    observe({
      wrapper,
      canvas,
      details,
      note,
      source,
      renderer: fenceTarget(limit).renderer
    })
  }

  // 公式渲染：span 先放 TeX 原文，渲染成功再替换成 MathML；失败保留 TeX。
  const paintMath = (span: HTMLElement, tex: string) => {
    if (stopped || passOwners.has(span)) return
    owned.add(span)
    passOwners.set(span, owner)
    void (async () => {
      let result: RichRenderResult
      try {
        result = await rendererFor('math')(tex, span, renderContext())
      } catch {
        result = { ok: false, reason: 'error' }
      }
      if (stopped || controller.signal.aborted) return
      if (result?.ok === true) {
        span.dataset.sfRichState = 'ready'
        return
      }
      span.dataset.sfRichState = 'fallback'
      span.textContent = tex
    })()
  }

  // 接手上一轮被 cleanup 中止的公式：span 里仍是 TeX 原文，直接重排一次。
  for (const span of root.querySelectorAll<HTMLElement>('.sf-rich-math[data-sf-rich-state="pending"]')) {
    const tex = span.textContent ?? ''
    if (tex) paintMath(span, tex)
  }

  // 公式：逐文本节点切分，命中定界符才建 span。超配额时整段保持原文，
  // 因为 MathSegment 不携带原始定界符，重建会丢字符。
  let mathUsed = doc.querySelectorAll('.sf-rich-math').length
  for (const node of collectTextNodes(root)) {
    const value = node.nodeValue ?? ''
    if (!value) continue
    // 切分是纯函数，但编排层承诺不抛异常：异常时该文本节点保持原样。
    let segments: ReturnType<typeof splitMathText>
    try {
      segments = splitMathText(value)
    } catch {
      continue
    }
    let mathCount = 0
    for (const segment of segments) {
      if (segment.kind === 'math') mathCount += 1
    }
    if (mathCount === 0) continue
    if (mathUsed + mathCount > RICH_MATH_LIMITS.maxPerDocument) continue
    mathUsed += mathCount

    const fragment = doc.createDocumentFragment()
    const pending: Array<{ span: HTMLElement, tex: string }> = []
    for (const segment of segments) {
      if (segment.kind === 'text') {
        if (segment.value) fragment.append(doc.createTextNode(segment.value))
        continue
      }
      const span = doc.createElement('span')
      span.className = 'sf-rich-math'
      span.dataset.sfRichState = 'pending'
      // renderMath 用该属性判定块级（RichRenderer 签名里没有 display 标志）。
      if (segment.display) span.dataset.mathDisplay = '1'
      span.textContent = segment.tex
      pending.push({ span, tex: segment.tex })
      fragment.append(span)
    }
    node.replaceWith(fragment)

    for (const item of pending) paintMath(item.span, item.tex)
  }

  return () => {
    if (stopped) return
    stopped = true
    controller.abort()
    observer?.disconnect()
    observer = null
    jobs.clear()
    // 释放归属：仍在 pending 的图块留给下一轮接手。
    for (const element of owned) {
      if (passOwners.get(element) === owner) passOwners.delete(element)
    }
    owned.clear()
  }
}

/**
 * 颜色模式切换时重渲染：把 figure.sf-rich 还原成原始 figure.sf-code-block，再跑一遍。
 * 公式不重渲染：KaTeX 输出的是 MathML，颜色继承 currentColor，主题切换自动生效。
 */
export function rerenderRichContent(
  root: ParentNode | null,
  options: RichContentOptions
): () => void {
  if (!root) return noop
  for (const wrapper of root.querySelectorAll('figure.sf-rich[data-sf-rich]')) {
    const source = wrapper.querySelector('figure.sf-code-block')
    if (source) wrapper.replaceWith(source)
    else wrapper.remove()
  }
  return enhanceRichContent(root, options)
}
