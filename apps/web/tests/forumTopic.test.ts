import { describe, expect, test } from 'bun:test'

import {
  buildForumCommentQuery,
  forumAuthorName,
  forumTopicPath,
  forumUserProfilePath,
  FORUM_TOPIC_ACTIONS,
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
