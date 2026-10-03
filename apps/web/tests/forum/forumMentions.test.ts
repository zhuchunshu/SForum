import { describe, expect, test } from 'bun:test'
import { Window } from 'happy-dom'

import {
  FORUM_MENTION_MAX_USERNAME_LENGTH,
  forumMentionPreviewPosition,
  forumMentionTokens,
  linkifyForumMentions,
  mentionAnchorFromEvent,
  mentionUsernameFromEvent
} from '../../app/utils/forum/forumMentions'

function testDocument() {
  return new Window({ url: 'https://forum.test/t/1' }).document
}

describe('forum mention syntax', () => {
  test('matches the server grammar for boundaries, charset, and CJK names', () => {
    const tokens = forumMentionTokens('hi @alice, @张三 和 @zhang-san 以及 user@example.com')
    expect(tokens.map(token => token.username)).toEqual(['alice', '张三', 'zhang-san'])
    expect(tokens.map(token => token.text)).toEqual(['@alice', '@张三', '@zhang-san'])
  })

  test('keeps at-signs inside words and repeated mentions addressable', () => {
    expect(forumMentionTokens('a@b c@ 中文@名')).toEqual([])
    expect(forumMentionTokens('@alice then @bob').map(token => token.start)).toEqual([0, 12])
  })

  test('does not treat over-long tokens as usernames', () => {
    const longName = 'a'.repeat(FORUM_MENTION_MAX_USERNAME_LENGTH)
    expect(forumMentionTokens(`@${longName}`)).toHaveLength(1)
    // 超长串整体放弃，避免高亮只截取前半段。
    expect(forumMentionTokens(`@${longName}b`)).toEqual([])
  })
})

describe('forum mention linkify', () => {
  test('links visible text and skips links, inline code, and code blocks', () => {
    const document = testDocument()
    const root = document.createElement('div')
    root.innerHTML = [
      '<p>@alice 你好</p>',
      '<pre><code>@in_block</code></pre>',
      '<p>行内 <code>@in_span</code> 不处理</p>',
      '<p><a href="/u/bob" data-sf-mention="bob" class="sf-mention">@bob</a></p>'
    ].join('')

    expect(linkifyForumMentions(root)).toBe(1)

    const anchors = Array.from(root.querySelectorAll('a.sf-mention'))
    expect(anchors).toHaveLength(2)
    const created = root.querySelector('p a.sf-mention') as HTMLAnchorElement
    expect(created.getAttribute('data-sf-mention')).toBe('alice')
    expect(created.getAttribute('href')).toBe('/u/alice')
    expect(created.getAttribute('aria-haspopup')).toBe('dialog')
    expect(created.getAttribute('aria-expanded')).toBe('false')
    expect(created.textContent).toBe('@alice')
    expect(root.textContent).toContain('@in_block')
    expect(root.querySelector('pre code')?.textContent).toBe('@in_block')
    expect(root.querySelector('p code')?.textContent).toBe('@in_span')
  })

  test('is idempotent across repeated decoration passes', () => {
    const document = testDocument()
    const root = document.createElement('div')
    root.innerHTML = '<p>@alice and @bob</p>'

    expect(linkifyForumMentions(root)).toBe(2)
    expect(linkifyForumMentions(root)).toBe(0)
    expect(root.querySelectorAll('a.sf-mention')).toHaveLength(2)
    expect(root.textContent).toBe('@alice and @bob')
  })

  test('uses the provided profile href resolver', () => {
    const document = testDocument()
    const root = document.createElement('div')
    root.innerHTML = '<p>@alice</p>'

    linkifyForumMentions(root, { hrefFor: username => `/en-US/u/${username}` })

    expect(root.querySelector('a.sf-mention')?.getAttribute('href')).toBe('/en-US/u/alice')
  })

  test('resolves the mentioned username from a click target', () => {
    const document = testDocument()
    const root = document.createElement('div')
    root.innerHTML = '<p>@alice</p>'
    linkifyForumMentions(root)

    const anchor = root.querySelector('a.sf-mention') as HTMLAnchorElement
    expect(mentionUsernameFromEvent(anchor)).toBe('alice')
    expect(mentionUsernameFromEvent(root.querySelector('p'))).toBeNull()

    // 只留类名（属性被剥离）时仍能反解锚点与用户名。
    anchor.removeAttribute('data-sf-mention')
    expect(mentionAnchorFromEvent(anchor)).toBe(anchor)
    expect(mentionUsernameFromEvent(anchor)).toBe('alice')
  })
})

describe('forum mention preview placement', () => {
  const anchor = { top: 400, bottom: 420, left: 300, width: 60 }

  test('opens below the mention by default', () => {
    expect(forumMentionPreviewPosition(anchor, { width: 1280, height: 800 })).toEqual({
      placement: 'below',
      top: 428,
      left: 170,
      width: 320
    })
  })

  test('flips above near the viewport bottom and stays inside the viewport', () => {
    const nearBottom = forumMentionPreviewPosition(
      { top: 700, bottom: 720, left: 4, width: 40 },
      { width: 1280, height: 800 }
    )
    expect(nearBottom.placement).toBe('above')
    expect(nearBottom.top).toBe(692)

    const narrow = forumMentionPreviewPosition(
      { top: 100, bottom: 120, left: 4, width: 40 },
      { width: 375, height: 700 }
    )
    expect(narrow.width).toBe(320)
    expect(narrow.left).toBe(8)
  })
})
