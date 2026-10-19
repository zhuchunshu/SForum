// Mermaid 渲染器（轨道 A）。把 ```mermaid 围栏渲染成 target 内的一个 <svg>。
//
// 设计约束：
// - mermaid / dompurify 只在真正渲染时动态 import：SSR 阶段不加载重依赖。
// - 安全配置由本文件给出，并用 secure 锁死 securityLevel / startOnLoad /
//   maxTextSize / suppressErrorRendering，源码里的 %%{init: ...}%% 改不动它们。
// - mermaid 的输出仍按不可信 HTML 处理：先过 DOMPurify 的 svg profile，再剥掉
//   <script>、on* 属性与 foreignObject 里的 href / xlink:href。
// - 渲染器契约要求不抛异常：失败一律返回原因码，abort 之后不写 DOM。
import type { MermaidConfig } from 'mermaid'
import { richFenceLimit, sourceByteLength } from './richContentLimits'
import type { RichRenderer } from './richContentTypes'

type MermaidApi = (typeof import('mermaid'))['default']
type ColorMode = 'light' | 'dark'

const MAX_SOURCE_BYTES = richFenceLimit('mermaid')!.maxSourceBytes

/** 单个图块的渲染上限；mermaid 没有内置超时，卡死的布局不应无限占用页面。 */
const RENDER_TIMEOUT_MS = 15_000

/** 锁死的安全项：任何 %%{init}%% 指令都不得改写（mermaid 的 secure 语义）。
 *  除了安全开关，也锁住主题类键：themeCSS / themeVariables 会写进 in-SVG <style> 的
 *  CSS 文本，而 DOMPurify 不净化 CSS，单图覆盖等于给用户一个样式注入面。 */
export const SECURE_KEYS = [
  'securityLevel',
  'startOnLoad',
  'maxTextSize',
  'suppressErrorRendering',
  'theme',
  'themeVariables',
  'themeCSS',
  'fontFamily'
] as const

/** DOMPurify 的 SVG profile；htmlLabels 已关闭，这里只需要 SVG 与滤镜。 */
const PURIFY_CONFIG = { USE_PROFILES: { svg: true, svgFilters: true } } as const

let mermaidModule: Promise<MermaidApi> | null = null
let appliedTheme: ColorMode | null = null
let renderSequence = 0

function mermaidConfig(
  variables: MermaidConfig['themeVariables'],
  fontFamily: string
): MermaidConfig {
  return {
    startOnLoad: false,
    securityLevel: 'strict',
    htmlLabels: false,
    maxTextSize: MAX_SOURCE_BYTES,
    suppressErrorRendering: true,
    secure: [...SECURE_KEYS],
    // base + 站点 token 的主题变量：默认主题的高饱和色块放在论坛正文里太吵。
    theme: 'base',
    themeVariables: { ...variables, fontFamily },
    flowchart: { htmlLabels: false, curve: 'basis', padding: 10 },
    mindmap: { padding: 12 },
    // 必须是具体字体族：mermaid 用这个值量文字宽度来定节点尺寸，写 inherit 会量到
    // 与实际渲染不同的字体，节点就装不下文字（中文尤其明显）。
    fontFamily
  }
}

/** 站点字体栈兜底；正常路径读容器上的 computed font-family。 */
const FALLBACK_FONT_FAMILY = 'Inter, "Noto Sans SC", "PingFang SC", "Microsoft YaHei", sans-serif'

type ThemeSpec = {
  variables: MermaidConfig['themeVariables']
  fontFamily: string
}

/** 从正文容器的 CSS 变量读出主题色；缺失时按深浅给一套中性兜底。 */
function themeSpecFor(target: HTMLElement, colorMode: ColorMode): ThemeSpec {
  const view = target.ownerDocument?.defaultView
  const read = (name: string, fallback: string) => {
    try {
      const value = view?.getComputedStyle(target).getPropertyValue(name).trim()
      return value || fallback
    } catch {
      return fallback
    }
  }
  const dark = colorMode === 'dark'
  const surface = read('--sf-rich-node-bg', dark ? '#1f2a28' : '#eef2f6')
  const border = read('--sf-rich-border', dark ? '#3b4a47' : '#cfd8e3')
  const text = read('--sf-rich-text', dark ? '#e6efec' : '#1f2937')
  const muted = read('--sf-rich-muted', dark ? '#87938f' : '#64748b')
  const accent = read('--sf-rich-accent', dark ? '#3b82f6' : '#2563eb')
  const edge = read('--sf-rich-edge', dark ? '#5b6b66' : '#94a3b8')
  let fontFamily = FALLBACK_FONT_FAMILY
  try {
    const computed = view?.getComputedStyle(target).fontFamily?.trim()
    if (computed) fontFamily = computed
  } catch {
    // 读不到就用站点兜底字体。
  }

  // mindmap 按层级取色：cScale0 是根，cScale1..11 是各层分支；cScaleInv* 是连线色。
  // 只给根上强调色，其余层统一表面色，靠边框与层级线区分，不用十二种高饱和色块。
  const scale: Record<string, string> = {}
  for (let index = 0; index < 12; index += 1) {
    // 统一表面色：根节点不靠高饱和填充区分（mermaid classic 对根节点的填充规则
    // 与文字色不同步，白字会落在浅底上），层级由结构与连线表达。
    scale[`cScale${index}`] = surface
    scale[`cScaleLabel${index}`] = text
    scale[`cScaleInv${index}`] = edge
  }

  return {
    fontFamily,
    variables: {
      background: 'transparent',
      primaryColor: surface,
      primaryTextColor: text,
      primaryBorderColor: border,
      secondaryColor: surface,
      secondaryTextColor: text,
      secondaryBorderColor: border,
      tertiaryColor: surface,
      tertiaryTextColor: text,
      tertiaryBorderColor: border,
      lineColor: edge,
      textColor: text,
      nodeBorder: border,
      clusterBkg: surface,
      clusterBorder: border,
      edgeLabelBackground: 'transparent',
      titleColor: text,
      fontSize: '14px',
      ...scale,
      // 甘特/时序等图的文字色。
      actorBkg: surface,
      actorBorder: border,
      actorTextColor: text,
      signalColor: text,
      signalTextColor: text,
      labelBoxBkgColor: surface,
      labelBoxBorderColor: border,
      labelTextColor: text,
      loopTextColor: text,
      noteBkgColor: surface,
      noteBorderColor: border,
      noteTextColor: text,
      // 饼图/象限图需要可区分的色相，交给 accent + 中性的弱化色。
      pie1: accent,
      pie2: surface,
      pie3: muted,
      pie4: border
    }
  }
}

/**
 * 动态加载 mermaid 并按 colorMode 初始化。
 *
 * mermaid.initialize 是覆盖写的幂等调用，所以并发首调时两边都执行也无害；
 * 主题变化必须重放一次，否则深色模式会沿用上一次的 theme。
 */
async function loadMermaid(colorMode: ColorMode, target: HTMLElement): Promise<MermaidApi> {
  if (!mermaidModule) mermaidModule = import('mermaid').then((module) => module.default)
  const mermaid = await mermaidModule
  if (appliedTheme !== colorMode) {
    const spec = themeSpecFor(target, colorMode)
    mermaid.initialize(mermaidConfig(spec.variables, spec.fontFamily))
    appliedTheme = colorMode
  }
  return mermaid
}

class MermaidTimeoutError extends Error {}

/** 超时后返回 error；迟到的渲染结果被丢弃，不会写进 DOM。 */
function withTimeout<T>(task: Promise<T>, timeoutMs: number): Promise<T> {
  return new Promise<T>((resolve, reject) => {
    const timer = setTimeout(() => reject(new MermaidTimeoutError('mermaid render timed out')), timeoutMs)
    task.then(
      (value) => {
        clearTimeout(timer)
        resolve(value)
      },
      (error: unknown) => {
        clearTimeout(timer)
        reject(error)
      }
    )
  })
}

/** 解析期失败归 invalid，其余（布局、超时、环境）归 error。 */
function isParseFailure(error: unknown): boolean {
  if (!(error instanceof Error)) return false
  if (error.name === 'ParseError' || error.name === 'UnknownDiagramError') return true
  // jison / peggy 的解析错误带 hash（token、loc、expected），比 message 更可靠。
  if ('hash' in error) return true
  return /parse error|syntax error|lexical error|diagram .*not found|no diagram type/i.test(error.message)
}

/** DOMPurify 之外的第二层兜底：即使上游把危险节点带进 SVG，也不落进宿主文档。 */
function scrubSvg(container: HTMLElement): void {
  for (const script of container.querySelectorAll('script')) script.remove()

  for (const element of container.querySelectorAll('*')) {
    for (const attribute of Array.from(element.attributes)) {
      if (attribute.name.toLowerCase().startsWith('on')) element.removeAttribute(attribute.name)
    }
  }

  // htmlLabels 已关闭；仍兜底剥掉 foreignObject 里的链接，避免它变成可点击导航。
  for (const foreign of container.querySelectorAll('foreignObject, foreignobject')) {
    for (const node of [foreign, ...Array.from(foreign.querySelectorAll('*'))]) {
      for (const attribute of Array.from(node.attributes)) {
        const name = attribute.name.toLowerCase()
        if (name === 'href' || name === 'xlink:href') node.removeAttribute(attribute.name)
      }
    }
  }

  // click 指令在 mermaid 11 里仍会生成 <a xlink:href>（securityLevel 拦的是 JS 回调，
  // 不是 URL 链接）。图内链接会绕过正文的站外中转页与 rel 策略，因此统一解包成普通节点：
  // 节点与文字保留，只去掉可点击能力。
  for (const anchor of container.querySelectorAll('a')) {
    anchor.replaceWith(...Array.from(anchor.childNodes))
  }
}

function nextRenderId(): string {
  renderSequence += 1
  return `sf-mermaid-${renderSequence}`
}

export const renderMermaid: RichRenderer = async (source, target, context) => {
  if (context.signal.aborted) return { ok: false, reason: 'error' }
  if (sourceByteLength(source) > MAX_SOURCE_BYTES) return { ok: false, reason: 'too_large' }
  // mermaid 的 SVG 依赖内联 <style>，文档 CSP 会把它拦掉；此时直接回落源码。
  if (context.cspActive) return { ok: false, reason: 'csp' }

  let svg: string
  try {
    const mermaid = await loadMermaid(context.colorMode, target)
    if (context.signal.aborted) return { ok: false, reason: 'error' }
    const result = await withTimeout(mermaid.render(nextRenderId(), source), RENDER_TIMEOUT_MS)
    svg = typeof result?.svg === 'string' ? result.svg : ''
  } catch (error) {
    return { ok: false, reason: isParseFailure(error) ? 'invalid' : 'error' }
  }

  if (context.signal.aborted) return { ok: false, reason: 'error' }
  if (!svg.includes('<svg')) return { ok: false, reason: 'invalid' }

  let sanitized: string
  try {
    const { default: DOMPurify } = await import('dompurify')
    sanitized = String(DOMPurify.sanitize(svg, PURIFY_CONFIG))
  } catch {
    return { ok: false, reason: 'error' }
  }
  if (context.signal.aborted) return { ok: false, reason: 'error' }

  const container = target.ownerDocument.createElement('div')
  container.innerHTML = sanitized
  scrubSvg(container)
  if (!container.querySelector('svg')) return { ok: false, reason: 'invalid' }

  // 写入前清空 target：重复渲染不会叠加旧图。
  target.replaceChildren(...Array.from(container.childNodes))
  return { ok: true }
}
