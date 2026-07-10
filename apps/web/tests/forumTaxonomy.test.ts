import { describe, expect, test } from 'bun:test'
import { readFileSync } from 'node:fs'

import {
  buildForumTopicQuery,
  forumTopicExtensionActionLabel,
  forumTopicExtensionActionRequest,
  forumTopicExtensionActionRequestPath,
  forumCategoryPath,
  forumTagPath,
  isForumTagSlug,
  normalizeForumTagSlugInput,
  parseForumTagPublicPagesOption
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
})
