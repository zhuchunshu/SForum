// 富内容围栏的共享契约。权威数据在 contracts/fixtures/rich-content-v1.json，
// 本文件只声明形状；三个渲染器（mermaid / chart / math）都必须实现 RichRenderer，
// 编排层只依赖这里，不依赖具体渲染器实现。

/** 富内容种类。mermaid 与 chart 来自代码围栏，math 来自正文定界符。 */
export type RichKind = 'mermaid' | 'chart' | 'math'

/** 回落原因码。与 fixture 的 fallbackReasons 一一对应，且必须各有本地化文案。 */
export type RichRenderFailure = 'invalid' | 'too_large' | 'unsupported' | 'csp' | 'error'

export type RichRenderResult =
  | { ok: true }
  | { ok: false, reason: RichRenderFailure }

/** 渲染器可用的本地化文案。文案键在 P0 冻结，渲染器不得自行拼中文。 */
export type RichContentLabels = {
  diagram: string
  chart: string
  math: string
  source: string
  invalid: string
  tooLarge: string
  unsupported: string
  cspBlocked: string
  error: string
}

export type RichRenderContext = {
  /** 页面卸载或颜色模式重渲染时中止，渲染器必须尊重它。 */
  signal: AbortSignal
  colorMode: 'light' | 'dark'
  /** 页面文档 CSP 生效时为 true（mermaid 会降级为源码）。 */
  cspActive: boolean
  labels: RichContentLabels
}

/**
 * 渲染器契约：把源码渲染进 target，成功返回 ok，失败返回原因码。
 * 不得抛异常、不得写 DOM 以外的东西、不得在顶层 import 重依赖。
 */
export type RichRenderer = (
  source: string,
  target: HTMLElement,
  context: RichRenderContext
) => Promise<RichRenderResult>

export type RichContentOptions = {
  labels: RichContentLabels
  colorMode: () => 'light' | 'dark'
  cspActive: () => boolean
  /** 测试或上层覆盖用：按 kind 替换渲染器。生产不传，走默认注册表。 */
  renderers?: Partial<RichRendererRegistry>
}

/** 渲染器实现按 kind 注册；编排层只按 kind 取用。 */
export type RichRendererRegistry = Record<RichKind, RichRenderer>
