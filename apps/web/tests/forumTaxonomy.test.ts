import { describe, expect, test } from 'bun:test'
import { readFileSync } from 'node:fs'

import {
  buildForumTopicQuery,
  forumTopicExtensionActionLabel,
  forumCommentExtensionActionRequest,
  forumTopicExtensionActionRequest,
  forumTopicExtensionActionRequestPath,
  forumCategoriesIndexPath,
  forumCategoryPath,
  forumTagPath,
  forumTagsIndexPath,
  isCreatedWithinDays,
  isForumTagSlug,
  normalizeForumTagSlugInput,
  parseForumTagPublicPagesOption,
  tagCloudSizeBucket,
  tagHotThreshold
} from '../app/utils/forumTaxonomy'

describe('forum taxonomy helpers', () => {
  test('category and tag pages rely on API pagination defaults', () => {
    const categoryPage = readFileSync(new URL('../../../extensions/builtin/themes/sforum-default/layer/app/pages/c/[categorySlug].vue', import.meta.url), 'utf8')
    const tagPage = readFileSync(new URL('../../../extensions/builtin/themes/sforum-default/layer/app/pages/tags/[tagSlug].vue', import.meta.url), 'utf8')
    expect(categoryPage).not.toContain('perPage: ITEMS_PER_PAGE')
    expect(tagPage).not.toContain('perPage: ITEMS_PER_PAGE')
    expect(categoryPage).toContain('topicList.value.perPage')
    expect(tagPage).toContain('topicList.value.perPage')
  })
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
    expect(forumTagPath('中文标签')).toBe('/tags/' + encodeURIComponent('中文标签'))
    expect(forumCategoriesIndexPath()).toBe('/categories')
    expect(forumTagsIndexPath()).toBe('/tags')
  })

  test('maps topic counts to tag cloud size buckets with log scale', () => {
    expect(tagCloudSizeBucket(0, 0, 0)).toBe(3)
    expect(tagCloudSizeBucket(10, 10, 10)).toBe(3)
    expect(tagCloudSizeBucket(1, 1, 1000)).toBe(1)
    expect(tagCloudSizeBucket(1000, 1, 1000)).toBe(6)
    const mid = tagCloudSizeBucket(50, 1, 1000)
    expect(mid).toBeGreaterThanOrEqual(2)
    expect(mid).toBeLessThanOrEqual(5)
  })

  test('computes hot threshold from the upper quartile of positive counts', () => {
    expect(tagHotThreshold([])).toBe(1)
    expect(tagHotThreshold([0, 0])).toBe(1)
    expect(tagHotThreshold([12])).toBe(12)
    // 8 个正计数 → 前 25% 取第 2 大（index 1）
    expect(tagHotThreshold([100, 80, 60, 40, 30, 20, 10, 5])).toBe(80)
  })

  test('detects tags created within a day window', () => {
    const now = Date.parse('2026-07-12T12:00:00Z')
    expect(isCreatedWithinDays('2026-07-10T00:00:00Z', 7, now)).toBe(true)
    expect(isCreatedWithinDays('2026-06-01T00:00:00Z', 7, now)).toBe(false)
    expect(isCreatedWithinDays('not-a-date', 7, now)).toBe(false)
  })

  test('normalizes forum tag slug input with Chinese characters', () => {
    expect(normalizeForumTagSlugInput(' 中文标签 ')).toBe('中文标签')
    expect(normalizeForumTagSlugInput('Nuxt-UI')).toBe('nuxt-ui')
    expect(isForumTagSlug('中文标签')).toBe(true)
    expect(isForumTagSlug('中文 标签')).toBe(false)
    expect(isForumTagSlug('bad_tag')).toBe(false)
  })

  test('formats safe topic extension actions', () => {
    const action = {
      extensionId: 'demo.plugin',
      id: 'demo.bookmark',
      label: { 'zh-CN': '收藏', 'en-US': 'Bookmark' },
      icon: 'i-lucide-bookmark',
      method: 'POST' as const,
      url: '/extensions/demo.plugin/topic-actions/bookmark',
      confirm: true
    }

    expect(forumTopicExtensionActionLabel(action, 'en-US')).toBe('Bookmark')
    expect(forumTopicExtensionActionLabel(action, 'fr-FR')).toBe('收藏')
    expect(forumTopicExtensionActionRequestPath(action)).toBe('/extensions/demo.plugin/topic-actions/bookmark')
    expect(forumTopicExtensionActionRequestPath({ ...action, url: 'https://example.com/callback' })).toBe('')
    expect(forumTopicExtensionActionRequestPath({ ...action, url: '/api/v1/topics/1' })).toBe('')
    expect(forumTopicExtensionActionRequest(action, 42)).toEqual({
      path: '/extensions/demo.plugin/topic-actions/bookmark',
      method: 'POST',
      body: { topicId: 42 }
    })
    expect(forumTopicExtensionActionRequest({ ...action, method: 'GET' as 'POST' }, 42)).toBeNull()
  })

  test('builds safe comment extension action requests with topic and comment ids', () => {
    const action = {
      extensionId: 'demo.plugin',
      id: 'demo.flag',
      method: 'POST' as const,
      url: '/extensions/demo.plugin/comment-actions/flag',
      requiresAuth: true
    }
    expect(forumCommentExtensionActionRequest(action, 42, 7)).toEqual({
      path: '/extensions/demo.plugin/comment-actions/flag',
      method: 'POST',
      body: { topicId: 42, commentId: 7 }
    })
    expect(forumCommentExtensionActionRequest(action, 42, 0)).toBeNull()
    expect(forumCommentExtensionActionRequest({ ...action, url: '/api/v1/x' }, 42, 7)).toBeNull()
  })
})
