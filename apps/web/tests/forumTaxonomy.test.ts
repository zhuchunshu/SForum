import { describe, expect, test } from 'bun:test'

import {
  buildForumTopicQuery,
  forumCategoryPath,
  forumTagPath,
  parseForumTagPublicPagesOption
} from '../app/utils/forumTaxonomy'

describe('forum taxonomy helpers', () => {
  test('parses public tag page option values', () => {
    expect(parseForumTagPublicPagesOption('enabled')).toBe(true)
    expect(parseForumTagPublicPagesOption('disabled')).toBe(false)
  })

  test('omits empty topic filter query params', () => {
    expect(buildForumTopicQuery({
      categorySlug: 'general',
      tagSlug: '',
      query: '   ',
      page: 2,
      perPage: undefined
    })).toEqual({
      categorySlug: 'general',
      page: '2'
    })
  })

  test('builds category and tag route paths', () => {
    expect(forumCategoryPath('general')).toBe('/c/general')
    expect(forumTagPath('nuxt')).toBe('/tags/nuxt')
  })
})
