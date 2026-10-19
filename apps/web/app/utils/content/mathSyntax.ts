// 公式定界符切分：纯函数，不依赖 window/document，SSR 可直接调用。
// 定界符与上限的权威数据在 contracts/fixtures/rich-content-v1.json；
// 这里只负责把正文拆成 text / math 片段，真正的渲染在 richContentMath.ts。
import { RICH_MATH_LIMITS, sourceByteLength } from './richContentLimits'

export type MathSegment =
  | { kind: 'text', value: string }
  | { kind: 'math', tex: string, display: boolean }

type MathDelimiter = { open: string, close: string, display: boolean }

/** 与 fixture 的 math.inlineDelimiters / math.displayDelimiters 同序。 */
export const MATH_DELIMITERS: {
  inline: readonly MathDelimiter[]
  display: readonly MathDelimiter[]
} = {
  inline: [
    { open: '$', close: '$', display: false },
    { open: '\\(', close: '\\)', display: false }
  ],
  display: [
    { open: '$$', close: '$$', display: true },
    { open: '\\[', close: '\\]', display: true }
  ]
}

const MAX_TEX_BYTES = RICH_MATH_LIMITS.maxSourceBytes
const BLANK_LINE = /\n[ \t\r]*\n/
const WHITESPACE = /\s/

type MathSpan = { start: number, end: number, tex: string }

/** 位置前有奇数个反斜杠时视为被转义。 */
function isEscaped(text: string, index: number): boolean {
  let count = 0
  for (let i = index - 1; i >= 0 && text[i] === '\\'; i -= 1) count += 1
  return count % 2 === 1
}

/** 找未被转义的闭合定界符。 */
function findClose(text: string, from: number, close: string): number {
  let index = text.indexOf(close, from)
  while (index !== -1) {
    if (!isEscaped(text, index)) return index
    index = text.indexOf(close, index + 1)
  }
  return -1
}

/** 文本里的 \$ 是转义美元符，展示时还原成字面 $。 */
function unescapeDollars(value: string): string {
  return value.replace(/\\\$/g, '$')
}

function appendText(segments: MathSegment[], value: string): void {
  if (value.length === 0) return
  const last = segments[segments.length - 1]
  if (last?.kind === 'text') last.value += value
  else segments.push({ kind: 'text', value })
}

function appendSegments(segments: MathSegment[], incoming: MathSegment[]): void {
  for (const segment of incoming) {
    if (segment.kind === 'text') appendText(segments, segment.value)
    else segments.push(segment)
  }
}

/** 该位置的 $ 是否属于 $$ 序列（用于阻止行内规则吃掉块级定界符）。 */
function isDollarPair(text: string, index: number): boolean {
  if (text[index + 1] === '$') return true
  return text[index - 1] === '$' && !isEscaped(text, index - 1)
}

/**
 * 行内 $ 的启发式（防 `$5 和 $10`）：开 $ 后不紧跟空白、闭 $ 前不是空白、
 * 中间不跨空行；另外跳过 $$ 序列与转义 $。
 */
function findInlineClose(text: string, open: number): number {
  let index = text.indexOf('$', open + 1)
  while (index !== -1) {
    const paired = isDollarPair(text, index)
    const content = text.slice(open + 1, index)
    const valid = !paired
      && !isEscaped(text, index)
      && index > open + 1
      && !WHITESPACE.test(text[index - 1] ?? '')
      && !BLANK_LINE.test(content)
    if (valid) return index
    index = text.indexOf('$', index + 1)
  }
  return -1
}

/** 先扫块级定界符，保证 $$…$$ / \[…\] 全局优先于行内匹配。 */
function findDisplaySpans(input: string): MathSpan[] {
  const spans: MathSpan[] = []
  let index = 0
  while (index < input.length) {
    const delimiter = MATH_DELIMITERS.display.find((candidate) =>
      input.startsWith(candidate.open, index) && !isEscaped(input, index)
    )
    if (delimiter) {
      const close = findClose(input, index + delimiter.open.length, delimiter.close)
      if (close !== -1) {
        spans.push({
          start: index,
          end: close + delimiter.close.length,
          tex: input.slice(index + delimiter.open.length, close)
        })
        index = close + delimiter.close.length
        continue
      }
    }
    index += 1
  }
  return spans
}

/** 在纯文本片段里切行内定界符。 */
function splitInline(text: string): MathSegment[] {
  const segments: MathSegment[] = []
  let buffer = ''
  let index = 0

  const flush = () => {
    if (buffer.length === 0) return
    appendText(segments, buffer)
    buffer = ''
  }

  while (index < text.length) {
    const char = text[index]

    if (char === '\\' && text[index + 1] === '$') {
      buffer += '$'
      index += 2
      continue
    }

    if (char === '\\' && text[index + 1] === '(' && !isEscaped(text, index)) {
      const close = findClose(text, index + 2, '\\)')
      if (close !== -1) {
        const tex = text.slice(index + 2, close)
        flush()
        if (tex.length > 0 && sourceByteLength(tex) <= MAX_TEX_BYTES) {
          segments.push({ kind: 'math', tex, display: false })
        } else {
          appendText(segments, unescapeDollars(text.slice(index, close + 2)))
        }
        index = close + 2
        continue
      }
      buffer += '\\('
      index += 2
      continue
    }

    if (char === '$' && !isDollarPair(text, index)) {
      const next = text[index + 1]
      if (next !== undefined && !WHITESPACE.test(next)) {
        const close = findInlineClose(text, index)
        if (close !== -1) {
          const tex = text.slice(index + 1, close)
          flush()
          if (tex.length > 0 && sourceByteLength(tex) <= MAX_TEX_BYTES) {
            segments.push({ kind: 'math', tex, display: false })
          } else {
            appendText(segments, unescapeDollars(text.slice(index, close + 1)))
          }
          index = close + 1
          continue
        }
      }
      buffer += '$'
      index += 1
      continue
    }

    buffer += char
    index += 1
  }

  flush()
  return segments
}

/**
 * 把正文拆成 text / math 片段。块级定界符先扫，未闭合、超长或空公式
 * 都原样退回文本，保证内容不丢。
 */
export function splitMathText(input: string): MathSegment[] {
  if (input.length === 0) return []

  const segments: MathSegment[] = []
  let cursor = 0

  for (const span of findDisplaySpans(input)) {
    if (span.start > cursor) {
      appendSegments(segments, splitInline(input.slice(cursor, span.start)))
    }
    const raw = input.slice(span.start, span.end)
    if (span.tex.length > 0 && sourceByteLength(span.tex) <= MAX_TEX_BYTES) {
      segments.push({ kind: 'math', tex: span.tex, display: true })
    } else {
      appendText(segments, unescapeDollars(raw))
    }
    cursor = span.end
  }

  if (cursor < input.length) {
    appendSegments(segments, splitInline(input.slice(cursor)))
  }
  return segments
}
