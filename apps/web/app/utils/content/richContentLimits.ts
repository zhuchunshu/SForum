// 富内容上限的 TypeScript 镜像。权威数据是
// contracts/fixtures/rich-content-v1.json；tests/content/richContentLimits.test.ts
// 逐字段比对两者，改一侧不改另一侧会直接红。

export type RichFenceLimit = {
  language: string
  kind: 'diagram' | 'chart'
  maxSourceBytes: number
  maxPerDocument: number
}

export const RICH_FENCE_LIMITS: readonly RichFenceLimit[] = [
  { language: 'mermaid', kind: 'diagram', maxSourceBytes: 12_288, maxPerDocument: 6 },
  { language: 'chart', kind: 'chart', maxSourceBytes: 8_192, maxPerDocument: 6 }
] as const

export const RICH_MATH_LIMITS = {
  maxSourceBytes: 2_048,
  maxPerDocument: 200,
  maxExpand: 1_000
} as const

export const RICH_CHART_LIMITS = {
  types: ['bar', 'line', 'area', 'pie', 'doughnut'],
  maxSeries: 6,
  maxLabels: 60,
  maxLabelChars: 120,
  maxTitleChars: 120,
  maxAbsValue: 1_000_000_000_000,
  maxSourceBytes: 8_192
} as const

/** 按语言取围栏上限；未知语言返回 undefined（调用方按普通代码块处理）。 */
export function richFenceLimit(language: string): RichFenceLimit | undefined {
  const normalized = language.trim().toLowerCase()
  return RICH_FENCE_LIMITS.find((limit) => limit.language === normalized)
}

/** 该语言是否由富内容渲染器接管。 */
export function isRichFenceLanguage(language: string): boolean {
  return richFenceLimit(language) !== undefined
}

/** 源码字节数（UTF-8），用于上限判断；TextEncoder 在 SSR 与浏览器都存在。 */
export function sourceByteLength(source: string): number {
  return new TextEncoder().encode(source).length
}
