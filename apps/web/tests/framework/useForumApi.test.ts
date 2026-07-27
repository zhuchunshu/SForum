import { describe, expect, test } from 'bun:test'
import { ref } from 'vue'

import { useApiClient } from '../../app/composables/useApiClient'
import { useForumApi } from '../../app/composables/forum/useForumApi'

// useForumApi 依赖 Nuxt 自动导入的 useApiClient；测试环境手动注入。
;(globalThis as any).useApiClient = useApiClient

async function withApiGlobals(csrfCookie: { value: string }, run: () => Promise<void>) {
  const originalFetch = globalThis.$fetch
  const originalUseRuntimeConfig = globalThis.useRuntimeConfig
  const originalUseNuxtApp = globalThis.useNuxtApp
  const originalUseCookie = globalThis.useCookie

  globalThis.useRuntimeConfig = () => ({
    public: { apiBaseUrl: '/api/v1', appLocale: 'zh-CN' }
  })
  globalThis.useNuxtApp = () => ({ $i18n: { locale: ref('zh-CN') } })
  globalThis.useCookie = (name: string) => {
    if (name !== 'csrf_') throw new Error(`unexpected cookie ${name}`)
    return csrfCookie
  }

  try {
    await run()
  } finally {
    globalThis.$fetch = originalFetch
    globalThis.useRuntimeConfig = originalUseRuntimeConfig
    globalThis.useNuxtApp = originalUseNuxtApp
    globalThis.useCookie = originalUseCookie
  }
}

describe('useForumApi', () => {
  test('createTopic nests editor content under content for the backend contract', async () => {
    const csrfCookie = ref('csrf-token')
    let body: unknown

    await withApiGlobals(csrfCookie, async () => {
      globalThis.$fetch = async (_url: string, options?: { body?: unknown }) => {
        body = options?.body
        return {
          code: 201,
          message: 'created',
          data: {
            id: 1,
            slug: 'hello',
            title: '标题',
            categoryId: 1,
            categorySlug: 'general',
            categoryName: '综合讨论',
            authorUserId: 1,
            status: 'active',
            isPinned: false,
            commentCount: 0,
            viewCount: 0,
            excerpt: '正文',
            createdAt: '2026-07-10T00:00:00Z',
            updatedAt: '2026-07-10T00:00:00Z',
            lastActivityAt: '2026-07-10T00:00:00Z',
            content: {
              id: 1,
              rawContent: '正文',
              htmlContent: '<p>正文</p>',
              plainText: '正文',
              excerpt: '正文',
              sourceFormat: 'markdown',
              editorType: 'tiptap',
              editorVersion: 'sf-editor-v1',
              renderVersion: 'goldmark-bluemonday-v2',
              contentHash: 'hash'
            }
          }
        }
      }

      const { createTopic } = useForumApi()
      await createTopic({
        title: '标题',
        categorySlug: 'general',
        tagSlugs: ['中文标签'],
        rawContent: '正文',
        sourceFormat: 'markdown',
        editorType: 'tiptap',
        editorVersion: 'sf-editor-v1',
        attachmentIds: [11, 12]
      })
    })

    expect(body).toEqual({
      title: '标题',
      categorySlug: 'general',
      tagSlugs: ['中文标签'],
      content: {
        rawContent: '正文',
        sourceFormat: 'markdown',
        editorType: 'tiptap',
        editorVersion: 'sf-editor-v1',
        attachmentIds: [11, 12]
      }
    })
  })
})
