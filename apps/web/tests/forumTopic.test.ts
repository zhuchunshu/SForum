import { describe, expect, test } from 'bun:test'

import {
  buildForumCommentQuery,
  flattenCommentTree,
  forumAuthorName,
  forumTopicPath,
  forumUserProfilePath,
  parseTopicPath,
  FORUM_TOPIC_ACTIONS,
  type ForumComment,
  type ForumCommentListQuery
} from '../app/utils/forumTaxonomy'

describe('forum topic helpers', () => {
  test('builds canonical topic path with encoded slug', () => {
    expect(forumTopicPath({ id: 42, slug: 'hello-world' })).toBe('/t/42/hello-world')
    expect(forumTopicPath({ id: 7, slug: '你好 世界' })).toBe('/t/7/' + encodeURIComponent('你好 世界'))
  })

  test('builds topic path per configured url mode', () => {
    const topic = { id: 42, slug: 'hello-world' }
    // 默认 id_slug
    expect(forumTopicPath(topic)).toBe('/t/42/hello-world')
    // 纯 id
    expect(forumTopicPath(topic, 'id')).toBe('/t/42')
    // 纯 slug
    expect(forumTopicPath(topic, 'slug')).toBe('/t/hello-world')
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

describe('parseTopicPath', () => {
  test('id_slug mode parses [id, slug] segments', () => {
    expect(parseTopicPath(['42', 'hello-world'], 'id_slug')).toEqual({ topicId: 42, slug: 'hello-world' })
    // 缺少 slug 段时仍返回 id（详情页可据此加载再规范化重定向）。
    expect(parseTopicPath(['42'], 'id_slug')).toEqual({ topicId: 42, slug: '' })
  })

  test('id mode parses single numeric segment', () => {
    expect(parseTopicPath(['42'], 'id')).toEqual({ topicId: 42 })
    // 非数字视为无效。
    expect(parseTopicPath(['hello'], 'id')).toBeNull()
  })

  test('slug mode parses single slug segment', () => {
    expect(parseTopicPath(['hello-world'], 'slug')).toEqual({ slug: 'hello-world' })
    // 编码后的 slug 应被解码。
    expect(parseTopicPath([encodeURIComponent('你好 世界')], 'slug')).toEqual({ slug: '你好 世界' })
  })

  test('rejects empty segments', () => {
    expect(parseTopicPath([], 'id_slug')).toBeNull()
    expect(parseTopicPath(undefined, 'id_slug')).toBeNull()
  })

  test('id_slug rejects non-numeric leading segment', () => {
    expect(parseTopicPath(['abc', 'slug'], 'id_slug')).toBeNull()
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

describe('flattenCommentTree', () => {
  test('returns empty array for empty input', () => {
    expect(flattenCommentTree([])).toEqual([])
  })

  test('flattens a flat list of root comments unchanged', () => {
    const roots = [makeComment(), makeComment()]
    const flat = flattenCommentTree(roots)
    expect(flat).toHaveLength(2)
    expect(flat[0].id).toBe(roots[0].id)
    expect(flat[1].id).toBe(roots[1].id)
  })

  test('flattens nested children into a single flat list', () => {
    // 根 → 子 → 孙（3层），拍平后应是 3 条
    const root = makeComment([makeComment([makeComment()])])
    const flat = flattenCommentTree([root])
    expect(flat).toHaveLength(3)
  })

  test('fills replyTo from parent for children missing replyTo', () => {
    // 子评论没有 replyTo，扁平化后应补上父评论作为引用
    const root = makeComment([makeComment()])
    const flat = flattenCommentTree([root])
    // flat[0] 是根，flat[1] 是子
    expect(flat[1].replyTo).toBeDefined()
    expect(flat[1].replyTo?.id).toBe(root.id)
  })

  test('preserves existing replyTo when backend already filled it', () => {
    const childWithReplyTo = makeComment()
    childWithReplyTo.replyTo = {
      id: 999,
      author: undefined,
      excerpt: '后端给的引用',
      depth: 0
    }
    const root = makeComment([childWithReplyTo])
    const flat = flattenCommentTree([root])
    // 已有 replyTo 不应被覆盖
    expect(flat[1].replyTo?.id).toBe(999)
    expect(flat[1].replyTo?.excerpt).toBe('后端给的引用')
  })

  test('handles multiple roots with mixed nesting', () => {
    // 两个根：一个有回复，一个没有
    const roots = [
      makeComment([makeComment()]),
      makeComment()
    ]
    const flat = flattenCommentTree(roots)
    expect(flat).toHaveLength(3)
  })
})
