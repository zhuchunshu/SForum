import { describe, expect, test } from 'bun:test'
import { readFileSync } from 'node:fs'

const css = readFileSync(
  new URL('../../app/assets/css/sforum-comment.css', import.meta.url),
  'utf8'
)

const themeCss = readFileSync(
  new URL('../../../../extensions/builtin/themes/sforum-default/assets/hybrid-forum.css', import.meta.url),
  'utf8'
)

describe('flat comment stream visual contract', () => {
  test('does not draw faux scrollbar separators between comments', () => {
    // 用「注释锚点 + 该段首个行规则」定位，而不是紧邻匹配：分隔线规则
    //（.sf-comment-list > .sf-comment + .sf-comment）插在两者之间时紧邻匹配会误判。
    const anchor = '/* Default flat discussion stream'
    expect(css).toContain(anchor)

    const flatCommentRule = css.slice(css.indexOf(anchor)).match(/\.sf-comment-list > \.sf-comment \{([^}]*)\}/)

    expect(flatCommentRule).not.toBeNull()
    expect(flatCommentRule?.[1]).toContain('border: 0;')
    expect(flatCommentRule?.[1]).not.toContain('border-bottom')
  })
})

// 移动端「正文拉满整行」依赖两个不变量，任一条被改回硬编码都会让正文列凭空变窄：
// 1) --sf-comment-lead 必须由「头像尺寸 + entry gap」算出，不能是手调字面量；
// 2) 拉满靠 width 撑开，所以 max-width 必须放开，否则只剩左移、右缘永远短 lead 宽。
describe('mobile comment full-width contract', () => {
  test('derives the pull offset from avatar size and entry gap instead of a literal', () => {
    expect(css).toMatch(
      /--sf-comment-avatar-size: 32px;[\s\S]*?--sf-comment-entry-gap: 11px;[\s\S]*?--sf-comment-lead: calc\(var\(--sf-comment-avatar-size\) \+ var\(--sf-comment-entry-gap\)\);/
    )
    expect(css).not.toMatch(/--sf-comment-lead: \d+px/)
  })

  test('theme re-binds lead to its own avatar box and gap', () => {
    expect(themeCss).toMatch(
      /--sf-comment-lead: calc\(var\(--sf-comment-avatar-size\) \+ var\(--sf-comment-entry-gap\)\);/
    )
    expect(themeCss).not.toMatch(/--sf-comment-lead: \d+px/)
    // 主题的头像盒与 gap 必须走同一组变量，否则算 lead 用的尺寸会和真实头像盒脱节。
    expect(themeCss).toMatch(
      /\.sf-theme--default \.sf-comment__entry \{[\s\S]*?--sf-comment-entry-gap: 14px;[\s\S]*?gap: var\(--sf-comment-entry-gap\);/
    )
    expect(themeCss).toMatch(
      /\.sf-theme--default \.sf-comment__avatar-trigger > \.sf-avatar,[\s\S]*?width: var\(--sf-comment-avatar-size\);/
    )
  })

  test('pulled content elements release max-width on mobile', () => {
    expect(css).toMatch(
      /\.sf-comment__reply-to,\s*\.sf-comment__content,\s*\.sf-comment__deleted,\s*\.sf-comment__inline-editor,\s*\.sf-comment__actions \{\s*width: calc\(100% \+ var\(--sf-comment-lead\)\);\s*max-width: none;/
    )
  })

  test('mobile comment row stays flush with the list column', () => {
    expect(themeCss).toMatch(
      /@media \(max-width: 640px\) \{[\s\S]*?\.sf-theme--default \.sf-comment-list > \.sf-comment \{\s*margin: 0;\s*padding: 0 2px;/
    )
  })
})
