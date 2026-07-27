import { describe, expect, test } from 'bun:test'
import { ref } from 'vue'

import { useAccountSecurityApi } from '../../app/composables/identity/useAccountSecurityApi'
import { useApiClient } from '../../app/composables/useApiClient'

// useAccountSecurityApi 内部依赖 Nuxt 自动导入的 useApiClient；测试环境手动注入。
;(globalThis as any).useApiClient = useApiClient

// 复用 useApiClient 测试的 globals 注入模式（runtimeConfig / useCookie 等）。
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

describe('useAccountSecurityApi', () => {
  test('listSessions calls GET /auth/sessions and unwraps data', async () => {
    const csrfCookie = ref('token')
    let calledUrl = ''

    await withApiGlobals(csrfCookie, async () => {
      globalThis.$fetch = async (url: string) => {
        calledUrl = url
        return { code: 200, message: 'ok', data: { items: [], total: 0, page: 1, perPage: 20 } }
      }
      const { listSessions } = useAccountSecurityApi()
      const result = await listSessions()
      expect(result.total).toBe(0)
    })

    expect(calledUrl).toBe('/api/v1/auth/sessions')
  })

  test('listSessions appends includeHistory and pagination query params', async () => {
    const csrfCookie = ref('token')
    let calledUrl = ''

    await withApiGlobals(csrfCookie, async () => {
      globalThis.$fetch = async (url: string) => {
        calledUrl = url
        return { code: 200, message: 'ok', data: { items: [], total: 0, page: 1, perPage: 50 } }
      }
      const { listSessions } = useAccountSecurityApi()
      await listSessions({ includeHistory: true, page: 2, perPage: 50 })
    })

    expect(calledUrl).toContain('includeHistory=true')
    expect(calledUrl).toContain('page=2')
    expect(calledUrl).toContain('perPage=50')
  })

  test('revokeSession calls DELETE with the opaque session id', async () => {
    const csrfCookie = ref('token')
    const calls: Array<{ url: string, method?: string }> = []

    await withApiGlobals(csrfCookie, async () => {
      globalThis.$fetch = async (url: string, options?: { method?: string }) => {
        calls.push({ url, method: options?.method })
        return { code: 200, message: 'ok', data: null }
      }
      const { revokeSession } = useAccountSecurityApi()
      await revokeSession('sid-abc')
    })

    expect(calls[0].method).toBe('DELETE')
    expect(calls[0].url).toBe('/api/v1/auth/sessions/sid-abc')
  })

  test('revokeSession URL-encodes the session id', async () => {
    const csrfCookie = ref('token')
    let calledUrl = ''

    await withApiGlobals(csrfCookie, async () => {
      globalThis.$fetch = async (url: string) => {
        calledUrl = url
        return { code: 200, message: 'ok', data: null }
      }
      const { revokeSession } = useAccountSecurityApi()
      await revokeSession('sid with spaces')
    })

    expect(calledUrl).toBe('/api/v1/auth/sessions/sid%20with%20spaces')
  })

  test('revokeOtherSessions calls POST /auth/sessions/revoke-others', async () => {
    const csrfCookie = ref('token')
    const calls: Array<{ url: string, method?: string }> = []

    await withApiGlobals(csrfCookie, async () => {
      globalThis.$fetch = async (url: string, options?: { method?: string }) => {
        calls.push({ url, method: options?.method })
        return { code: 200, message: 'ok', data: { revoked: 3 } }
      }
      const { revokeOtherSessions } = useAccountSecurityApi()
      const result = await revokeOtherSessions()
      expect(result.revoked).toBe(3)
    })

    expect(calls[0].method).toBe('POST')
    expect(calls[0].url).toBe('/api/v1/auth/sessions/revoke-others')
  })

  test('listExternalIdentities calls GET /auth/external-identities and redacts rows', async () => {
    const csrfCookie = ref('token')
    let calledUrl = ''

    await withApiGlobals(csrfCookie, async () => {
      globalThis.$fetch = async (url: string) => {
        calledUrl = url
        return {
          code: 200,
          message: 'ok',
          data: [{
            linkId: 9,
            providerId: 'example.oauth',
            status: 'active',
            linkedAt: '2026-07-27T12:00:00Z',
            providerSubject: 'must-not-leak'
          }]
        }
      }
      const { listExternalIdentities } = useAccountSecurityApi()
      const items = await listExternalIdentities()
      expect(items).toHaveLength(1)
      expect(items[0].linkId).toBe(9)
      expect(JSON.stringify(items)).not.toContain('must-not-leak')
    })

    expect(calledUrl).toBe('/api/v1/auth/external-identities')
  })

  test('unlinkExternalIdentity calls DELETE with link id', async () => {
    const csrfCookie = ref('token')
    const calls: Array<{ url: string, method?: string, body?: unknown }> = []

    await withApiGlobals(csrfCookie, async () => {
      globalThis.$fetch = async (url: string, options?: { method?: string, body?: unknown }) => {
        calls.push({ url, method: options?.method, body: options?.body })
        return { code: 200, message: 'ok', data: null }
      }
      const { unlinkExternalIdentity } = useAccountSecurityApi()
      await unlinkExternalIdentity(42, { expectedRevision: 3 })
    })

    expect(calls[0].method).toBe('DELETE')
    expect(calls[0].url).toBe('/api/v1/auth/external-identities/42')
    expect(calls[0].body).toEqual({ expectedRevision: 3 })
  })

  test('setupPassword posts to /auth/password', async () => {
    const csrfCookie = ref('token')
    const calls: Array<{ url: string, method?: string, body?: unknown }> = []

    await withApiGlobals(csrfCookie, async () => {
      globalThis.$fetch = async (url: string, options?: { method?: string, body?: unknown }) => {
        calls.push({ url, method: options?.method, body: options?.body })
        return { code: 200, message: 'ok', data: null }
      }
      const { setupPassword } = useAccountSecurityApi()
      await setupPassword('correct horse battery staple')
    })

    expect(calls[0].method).toBe('POST')
    expect(calls[0].url).toBe('/api/v1/auth/password')
    expect(calls[0].body).toEqual({ password: 'correct horse battery staple' })
  })
})
