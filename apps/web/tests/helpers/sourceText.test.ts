import { describe, expect, test } from 'bun:test'

import { stripHtmlComments } from './sourceText'

describe('stripHtmlComments', () => {
  test('removes a plain comment without gluing its neighbours together', () => {
    expect(stripHtmlComments('a<!-- note -->b')).toBe('a b')
  })

  test('removes every comment in a multi-line source', () => {
    expect(stripHtmlComments('<div>\n<!-- 一 -->\nx\n<!-- 二 -->\ny\n</div>'))
      .toBe('<div>\n \nx\n \ny\n</div>')
  })

  test('never rebuilds a comment opener from a nested construct', () => {
    // 单次替换会得到 `<!-->`，仍含 `<!--`；这正是 CodeQL 告警的原始形态。
    for (const nested of ['<!<!-- -->-->', '<!<!<!-- -->-->>', '<!--<!-- a -->-->', '<!<!-- x -->--><p>']) {
      const stripped = stripHtmlComments(nested)
      expect(stripped.includes('<!--')).toBe(false)
    }
  })

  test('cannot splice a forbidden token out of a comment', () => {
    // 空串删除会让 `v-` + `html` 拼成 `v-html`，把断言骗过去。
    expect(stripHtmlComments('v-<!-- x -->html').includes('v-html')).toBe(false)
    expect(stripHtmlComments('inner<!-- x -->HTML').includes('innerHTML')).toBe(false)
  })

  test('is idempotent', () => {
    const once = stripHtmlComments('<!<!-- -->--><p>x</p>')
    expect(stripHtmlComments(once)).toBe(once)
  })

  test('leaves a source without comments untouched', () => {
    const plain = '<template>\n  <p>{{ entry.excerpt }}</p>\n</template>'
    expect(stripHtmlComments(plain)).toBe(plain)
  })

  test('handles the empty string', () => {
    expect(stripHtmlComments('')).toBe('')
  })
})
