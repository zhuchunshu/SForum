import { describe, expect, test } from 'bun:test'
import { readFileSync } from 'node:fs'

const css = readFileSync(
  new URL('../../app/assets/css/sforum-comment.css', import.meta.url),
  'utf8'
)

describe('flat comment stream visual contract', () => {
  test('does not draw faux scrollbar separators between comments', () => {
    const flatCommentRule = css.match(
      /\/\* Default flat discussion stream:[^*]+\*\/\s*\.sf-comment-list > \.sf-comment \{([^}]*)\}/
    )

    expect(flatCommentRule).not.toBeNull()
    expect(flatCommentRule?.[1]).toContain('border: 0;')
    expect(flatCommentRule?.[1]).not.toContain('border-bottom')
  })
})
