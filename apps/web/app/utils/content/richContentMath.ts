// KaTeX → MathML 渲染器。输出只保留 MathML（无内联 style），可扛严格 CSP。
// katex 是重依赖，只在真正渲染时动态 import；签名与返回语义遵循 RichRenderer。
import type { RichRenderer } from './richContentTypes'
import { RICH_MATH_LIMITS, sourceByteLength } from './richContentLimits'

export const renderMath: RichRenderer = async (source, target, context) => {
  if (context.signal.aborted) return { ok: false, reason: 'error' }
  if (sourceByteLength(source) > RICH_MATH_LIMITS.maxSourceBytes) {
    return { ok: false, reason: 'too_large' }
  }
  if (source.trim().length === 0) return { ok: false, reason: 'invalid' }

  try {
    const katex = (await import('katex')).default
    if (context.signal.aborted) return { ok: false, reason: 'error' }

    // 编排层通过 dataset 告知块级公式；没有该属性按行内处理。
    const displayMode = target.dataset.mathDisplay === '1'
    // 先渲染到游离容器，校验通过后再落盘，abort 或失败时目标 DOM 保持原样。
    const staging = target.ownerDocument.createElement('div')
    katex.render(source, staging, {
      output: 'mathml',
      throwOnError: false,
      trust: false,
      strict: 'ignore',
      maxExpand: RICH_MATH_LIMITS.maxExpand,
      displayMode
    })

    if (context.signal.aborted) return { ok: false, reason: 'error' }
    if (staging.querySelector('.katex-error') || !staging.querySelector('math')) {
      return { ok: false, reason: 'invalid' }
    }

    // KaTeX 少数命令（如 \fcolorbox）在 MathML 里仍会带 style，统一剥掉。
    for (const styled of staging.querySelectorAll('[style]')) styled.removeAttribute('style')

    target.replaceChildren(...Array.from(staging.childNodes))
    return { ok: true }
  } catch {
    return { ok: false, reason: 'error' }
  }
}
