import { describe, expect, test } from 'bun:test'
import { renderLegalMarkdown } from '~/utils/legal/renderLegalMarkdown'

describe('renderLegalMarkdown', () => {
  test('renders markdown structure for legal documents', () => {
    const html = renderLegalMarkdown('## 社区指南\n\n1. 保持友善\n2. 就事论事')

    expect(html).toContain('<h2')
    expect(html).toContain('<ol>')
    expect(html).toContain('<li>')
  })

  test('drops raw html and unsafe links', () => {
    const html = renderLegalMarkdown('Hello <script>alert(1)</script>\n\n[Safe](/guidelines)\n\n[Bad](javascript:alert(1))')

    expect(html).not.toContain('<script>')
    expect(html).toContain('href="/guidelines"')
    expect(html).not.toContain('javascript:alert(1)')
  })
})
