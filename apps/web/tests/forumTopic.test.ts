import { describe, expect, test } from 'bun:test'

import {
  buildForumCommentQuery,
  countCommentDescendants,
  forumAuthorName,
  forumTopicPath,
  forumUserProfilePath,
  shouldCollapseByDefault,
  FORUM_TOPIC_ACTIONS,
  type ForumComment,
  type ForumCommentListQuery
} from '../app/utils/forumTaxonomy'

describe('forum topic helpers', () => {
  test('builds canonical topic path with encoded slug', () => {
    expect(forumTopicPath({ id: 42, slug: 'hello-world' })).toBe('/t/42/hello-world')
    expect(forumTopicPath({ id: 7, slug: '你好 世界' })).toBe('/t/7/' + encodeURIComponent('你好 世界'))
  })

  test('builds user profile path', () => {
    expect(forumUserProfilePath('alice')).toBe('/u/alice')
    expect(forumUserProfilePath('a b')).toBe('/u/' + encodeURIComponent('a b'))
  })

  test('resolves author display name with fallbacks', () => {
    expect(forumAuthorName({ id: 1, username: 'alice', displayName: 'Alice' }, 1)).toBe('Alice')
    expect(forumAuthorName({ id: 1, username: 'alice', displayName: '' }, 1)).toBe('alice')
    expect(forumAuthorName(undefined, 99)).toBe('#99')
  })

  test('builds comment query params with view and pagination', () => {
    const query: ForumCommentListQuery = { view: 'tree', page: 2, perPage: 10 }
    expect(buildForumCommentQuery(query)).toEqual({ view: 'tree', page: '2', perPage: '10' })
  })

  test('omits invalid comment query params', () => {
    expect(buildForumCommentQuery({ view: 'invalid' as never, page: 0, perPage: -1 })).toEqual({})
    expect(buildForumCommentQuery({})).toEqual({})
  })

  test('exposes topic action keys matching backend contract', () => {
    expect(FORUM_TOPIC_ACTIONS).toEqual({
      hide: 'hide',
      restore: 'restore',
      lock: 'lock',
      unlock: 'unlock',
      pin: 'pin',
      unpin: 'unpin'
    })
  })
})

// 构造最小可用评论节点（仅填必填字段），children 可选。
function makeComment(children: ForumComment[] = []): ForumComment {
  return {
    id: Math.floor(Math.random() * 1_000_000),
    topicId: 1,
    authorUserId: 1,
    parentId: null,
    rootCommentId: 1,
    pathKey: '000000000001',
    depth: 0,
    replyCount: children.length,
    status: 'active',
    content: {
      rawContent: '',
      htmlContent: '',
      plainText: '',
      excerpt: '',
      sourceFormat: 'markdown',
      editorType: 'tiptap',
      renderVersion: 'goldmark-bluemonday-v1',
      contentHash: ''
    },
    createdAt: '2026-07-08T00:00:00Z',
    updatedAt: '2026-07-08T00:00:00Z',
    children
  }
}

describe('comment tree collapse helpers', () => {
  test('countCommentDescendants returns 0 for leaf or empty children', () => {
    expect(countCommentDescendants(null)).toBe(0)
    expect(countCommentDescendants(undefined)).toBe(0)
    expect(countCommentDescendants(makeComment())).toBe(0)
    expect(countCommentDescendants(makeComment([]))).toBe(0)
  })

  test('countCommentDescendants counts all descendants recursively', () => {
    // 树形结构：根有 2 个直接子评论；第一个子评论又有 1 个孙评论。
    // 后代总数 = 2（直接） + 1（孙） = 3
    const tree = makeComment([
      makeComment([makeComment()]),
      makeComment()
    ])
    expect(countCommentDescendants(tree)).toBe(3)
  })

  test('countCommentDescendants handles deep nesting', () => {
    // 4 层深的单链：根 → 子 → 孙 → 曾孙。后代总数 = 3
    const deep = makeComment([makeComment([makeComment([makeComment()])])])
    expect(countCommentDescendants(deep)).toBe(3)
  })

  test('shouldCollapseByDefault returns false below threshold', () => {
    // 3 个直接子评论 < 默认阈值 4，不折叠
    expect(shouldCollapseByDefault(makeComment([makeComment(), makeComment(), makeComment()]))).toBe(false)
  })

  test('shouldCollapseByDefault returns true at or above threshold', () => {
    // 4 个直接子评论 = 阈值，折叠
    const four = makeComment([makeComment(), makeComment(), makeComment(), makeComment()])
    expect(shouldCollapseByDefault(four)).toBe(true)
    // 5 个，折叠
    const five = makeComment([makeComment(), makeComment(), makeComment(), makeComment(), makeComment()])
    expect(shouldCollapseByDefault(five)).toBe(true)
  })

  test('shouldCollapseByDefault respects custom threshold', () => {
    // 阈值 2：2 个子评论即折叠
    const two = makeComment([makeComment(), makeComment()])
    expect(shouldCollapseByDefault(two, 2)).toBe(true)
    expect(shouldCollapseByDefault(two, 4)).toBe(false)
  })

  test('shouldCollapseByDefault returns false for null/undefined/leaf', () => {
    expect(shouldCollapseByDefault(null)).toBe(false)
    expect(shouldCollapseByDefault(undefined)).toBe(false)
    expect(shouldCollapseByDefault(makeComment())).toBe(false)
  })
})
