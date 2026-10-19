import { describe, expect, test } from 'bun:test'
import fixture from '../../../../contracts/fixtures/rich-content-v1.json'
import {
  MATH_DELIMITERS,
  splitMathText,
  type MathSegment
} from '../../app/utils/content/mathSyntax'
import { renderMath } from '../../app/utils/content/richContentMath'
import { RICH_MATH_LIMITS } from '../../app/utils/content/richContentLimits'
import type { RichRenderContext } from '../../app/utils/content/richContentTypes'

// happy-dom 20 没有实现 document.compatMode；KaTeX 在模块求值时若发现不是
// CSS1Compat 会把 render 换成直接抛错的实现。这里补上标准模式，避免把测试
// 环境的缺陷误判成渲染失败（真实浏览器有 doctype，compatMode 就是 CSS1Compat）。
Object.defineProperty(document, 'compatMode', { value: 'CSS1Compat', configurable: true })

const labels = {
  diagram: '图',
  chart: '图表',
  math: '公式',
  source: '源码',
  invalid: '无效',
  tooLarge: '过大',
  unsupported: '不支持',
  cspBlocked: '被 CSP 阻止',
  error: '错误'
}

function context(signal: AbortSignal = new AbortController().signal): RichRenderContext {
  return { signal, colorMode: 'light', cspActive: false, labels }
}

function text(value: string): MathSegment {
  return { kind: 'text', value }
}

function math(tex: string, display: boolean): MathSegment {
  return { kind: 'math', tex, display }
}

describe('splitMathText', () => {
  test('delimiter table mirrors the frozen fixture', () => {
    expect(MATH_DELIMITERS.inline.map((item) => [item.open, item.close])).toEqual(fixture.math.inlineDelimiters)
    expect(MATH_DELIMITERS.display.map((item) => [item.open, item.close])).toEqual(fixture.math.displayDelimiters)
    expect(RICH_MATH_LIMITS.maxSourceBytes).toBe(fixture.math.maxSourceBytes)
    expect(RICH_MATH_LIMITS.maxPerDocument).toBe(fixture.math.maxPerDocument)
    expect(RICH_MATH_LIMITS.maxExpand).toBe(fixture.math.maxExpand)
  })

  test('keeps inline $...$ as inline math', () => {
    expect(splitMathText('$O(n\\log n)$')).toEqual([math('O(n\\log n)', false)])
  })

  test('renders $$...$$ as display math', () => {
    expect(splitMathText('前 $$\\sum_{i=1}^{n} i$$ 后')).toEqual([
      text('前 '),
      math('\\sum_{i=1}^{n} i', true),
      text(' 后')
    ])
  })

  test('block delimiters win over inline matching', () => {
    expect(splitMathText('$a $$b$$ c$')).toEqual([
      text('$a '),
      math('b', true),
      text(' c$')
    ])
  })

  test('does not turn prices into math', () => {
    expect(splitMathText('$5 和 $10')).toEqual([text('$5 和 $10')])
    expect(splitMathText('价格 $ 5 和 $10')).toEqual([text('价格 $ 5 和 $10')])
  })

  test('supports \\(...\\) and \\[...\\] delimiters', () => {
    expect(splitMathText('a \\(x\\) b \\[y\\] c')).toEqual([
      text('a '),
      math('x', false),
      text(' b '),
      math('y', true),
      text(' c')
    ])
  })

  test('treats unclosed delimiters as text', () => {
    expect(splitMathText('$x 未闭合')).toEqual([text('$x 未闭合')])
    expect(splitMathText('$$x 未闭合')).toEqual([text('$$x 未闭合')])
    expect(splitMathText('\\(x 未闭合')).toEqual([text('\\(x 未闭合')])
  })

  test('handles \\$ as a literal dollar', () => {
    expect(splitMathText('价格 \\$5 和 \\$10')).toEqual([text('价格 $5 和 $10')])
  })

  test('inline math cannot cross a blank line', () => {
    expect(splitMathText('$a\n\nb$')).toEqual([text('$a\n\nb$')])
    expect(splitMathText('$a\nb$')).toEqual([math('a\nb', false)])
  })

  test('falls back to text when a single formula exceeds maxSourceBytes', () => {
    const inline = `$${'x'.repeat(RICH_MATH_LIMITS.maxSourceBytes + 1)}$`
    const display = `$$${'y'.repeat(RICH_MATH_LIMITS.maxSourceBytes + 1)}$$`
    expect(splitMathText(inline)).toEqual([text(inline)])
    expect(splitMathText(display)).toEqual([text(display)])
  })

  test('accepts a formula exactly at maxSourceBytes', () => {
    const tex = 'z'.repeat(RICH_MATH_LIMITS.maxSourceBytes)
    expect(splitMathText(`$${tex}$`)).toEqual([math(tex, false)])
  })

  test('empty input yields no segments', () => {
    expect(splitMathText('')).toEqual([])
  })
})

describe('renderMath', () => {
  test('renders inline math as MathML without inline styles', async () => {
    const target = document.createElement('div')
    const result = await renderMath('O(n\\log n)', target, context())

    expect(result).toEqual({ ok: true })
    expect(target.innerHTML).toContain('<math')
    expect(target.querySelector('math')?.getAttribute('display')).toBeNull()
    expect(target.querySelector('[style]')).toBeNull()
  })

  test('uses dataset.mathDisplay for block math', async () => {
    const target = document.createElement('div')
    target.dataset.mathDisplay = '1'
    const result = await renderMath('\\sum_{i=1}^{n} i', target, context())

    expect(result).toEqual({ ok: true })
    expect(target.querySelector('math')?.getAttribute('display')).toBe('block')
  })

  test('reports invalid for a syntax error and leaves the target untouched', async () => {
    const target = document.createElement('div')
    target.textContent = '占位'
    const result = await renderMath('\\frac{1}{', target, context())

    expect(result).toEqual({ ok: false, reason: 'invalid' })
    expect(target.textContent).toBe('占位')
  })

  test('does not write DOM after abort', async () => {
    const controller = new AbortController()
    controller.abort()
    const target = document.createElement('div')
    const result = await renderMath('x^2', target, context(controller.signal))

    expect(result).toEqual({ ok: false, reason: 'error' })
    expect(target.innerHTML).toBe('')
  })

  test('reports too_large before importing katex', async () => {
    const target = document.createElement('div')
    const result = await renderMath('x'.repeat(RICH_MATH_LIMITS.maxSourceBytes + 1), target, context())

    expect(result).toEqual({ ok: false, reason: 'too_large' })
    expect(target.innerHTML).toBe('')
  })

  test('reports invalid for empty source', async () => {
    const target = document.createElement('div')
    expect(await renderMath('   ', target, context())).toEqual({ ok: false, reason: 'invalid' })
    expect(target.innerHTML).toBe('')
  })

  test('turns unexpected failures into error', async () => {
    const result = await renderMath('x^2', {} as unknown as HTMLElement, context())
    expect(result).toEqual({ ok: false, reason: 'error' })
  })

  test('survives a \\def expansion bomb', async () => {
    const target = document.createElement('div')
    const result = await renderMath('\\def\\a{\\a\\a}\\a', target, context())

    expect(result).toEqual({ ok: false, reason: 'invalid' })
    expect(target.innerHTML).toBe('')
  })

  test('strips styles KaTeX may add for \\fcolorbox', async () => {
    const target = document.createElement('div')
    const result = await renderMath('\\fcolorbox{red}{blue}{x}', target, context())

    expect(result).toEqual({ ok: true })
    expect(target.innerHTML).toContain('<math')
    expect(target.querySelector('[style]')).toBeNull()
  })

  test('replaces previous output on re-render', async () => {
    const target = document.createElement('div')
    await renderMath('x', target, context())
    await renderMath('y^2', target, context())

    expect(target.querySelectorAll('math')).toHaveLength(1)
    expect(target.innerHTML).toContain('y')
  })
})
