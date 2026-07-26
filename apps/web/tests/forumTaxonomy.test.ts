import { describe, expect, test } from 'bun:test'
import { readFileSync } from 'node:fs'

import {
  buildForumTopicQuery,
  formatForumTopicListTotal,
  forumTopicExtensionActionLabel,
  forumCommentExtensionActionRequest,
  forumTopicExtensionActionRequest,
  forumTopicExtensionActionRequestPath,
  forumEditorInitialContent,
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
  test('restores editor-document JSON without exposing it as Markdown text', () => {
    const native = { type: 'doc', content: [{ type: 'paragraph' }] }
    expect(forumEditorInitialContent({
      rawContent: JSON.stringify(native),
      sourceFormat: 'editor-document'
    })).toEqual(native)
    expect(forumEditorInitialContent({
      rawContent: '# Markdown',
      sourceFormat: 'markdown'
    })).toBe('# Markdown')
    expect(forumEditorInitialContent({
      rawContent: '{broken',
      sourceFormat: 'editor-document'
    })).toBe('{broken')
  })

  test('edit entrypoints load editor-document via initialContent not raw JSON v-model', () => {
    const editor = readFileSync(new URL('../app/components/SFEditor.vue', import.meta.url), 'utf8')
    const editPage = readFileSync(new URL('../app/components/SFTopicEditPage.vue', import.meta.url), 'utf8')
    const showPage = readFileSync(new URL('../app/components/SFTopicShowPage.vue', import.meta.url), 'utf8')
    const topicEditor = readFileSync(new URL('../app/components/SFTopicEditor.vue', import.meta.url), 'utf8')
    const adminComment = readFileSync(
      new URL('../app/components/admin/forum/SFAdminForumCommentEditor.vue', import.meta.url),
      'utf8'
    )

    expect(editor).toContain('initialContent')
    expect(editor).toContain("typeof initialContent === 'string' ? { contentType: 'markdown' as const }")
    expect(editPage).toContain('forumEditorInitialContent')
    expect(editPage).not.toContain('.content.rawContent')
    expect(showPage).toContain('forumEditorInitialContent(comment.content)')
    expect(showPage).not.toContain('comment.content.rawContent')
    expect(topicEditor).toContain('forumEditorInitialContent(props.topic.content)')
    expect(topicEditor).not.toContain('props.topic.content.rawContent')
    expect(adminComment).toContain('forumEditorInitialContent(props.comment.content)')
    expect(adminComment).not.toContain('props.comment.content.rawContent')
  })

  test('formats list total with 约 only when totalApproximate', () => {
    const t = (key: string, params?: Record<string, unknown>) => {
      if (key === 'home.feed.topicCountMetaApprox') {
        return `约 ${params?.count}`
      }
      if (key === 'home.feed.topicCountMeta') {
        return `${params?.count} 个主题`
      }
      return key
    }
    expect(formatForumTopicListTotal({ total: 200000, totalApproximate: false }, t)).toBe('200000 个主题')
    expect(formatForumTopicListTotal({ total: 1000000, totalApproximate: true }, t)).toBe('约 1000000')
    expect(formatForumTopicListTotal({ total: 12 }, t)).toBe('12 个主题')
  })

  test('category and tag pages rely on API pagination defaults', () => {
    const categoryPage = readFileSync(new URL('../../../apps/web/app/components/SFCategoryShowPage.vue', import.meta.url), 'utf8')
    const tagPage = readFileSync(new URL('../../../apps/web/app/components/SFTagShowPage.vue', import.meta.url), 'utf8')
    expect(categoryPage).not.toContain('perPage: ITEMS_PER_PAGE')
    expect(tagPage).not.toContain('perPage: ITEMS_PER_PAGE')
    expect(categoryPage).toContain('topicList.value.perPage')
    expect(tagPage).toContain('topicList.value.perPage')
    // D1：列表 total 经 formatForumTopicListTotal，近似才显示「约」
    expect(categoryPage).toContain('formatForumTopicListTotal')
    expect(tagPage).toContain('formatForumTopicListTotal')
  })

  test('topic show page loads detail via useAsyncData once (D3 no double-count path)', () => {
    const page = readFileSync(new URL('../../../apps/web/app/components/SFTopicShowPage.vue', import.meta.url), 'utf8')
    // 导航加载走 useAsyncData；无 onMounted 再拉详情、无独立 POST view 端点调用。
    expect(page).toContain('useAsyncData')
    expect(page).toContain('loadTopicFromCandidates')
    expect(page).not.toMatch(/onMounted\s*\(\s*async/)
    expect(page).not.toMatch(/post\s*\(\s*['`][^'`]*\/view/)
    expect(page).not.toContain('recordView')
    expect(page).not.toContain('postTopicView')
    // M4：URL 含 id 时 topic+comments 同步启动；客户端仅等待正文，SSR 保持完整评论输出。
    expect(page).toContain('const commentsAsync = useAsyncData(')
    expect(page).toContain('const topicResult = await topicAsync')
    expect(page).toContain('if (import.meta.server)')
    expect(page).toContain('Promise.all([commentsAsync, categoryGroupsAsync])')
    expect(page).toContain('urlTopicID')
    expect(page).toContain('await topicAsync')
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

  test('M5 after cursor omits page in topic query', () => {
    expect(buildForumTopicQuery({
      categorySlug: 'general',
      page: 50,
      after: 'opaque-cursor-token'
    })).toEqual({
      categorySlug: 'general',
      after: 'opaque-cursor-token'
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
