import { describe, expect, test } from 'bun:test'
import { readFileSync } from 'node:fs'
import { parse, compileScript } from '@vue/compiler-sfc'
import { computed, reactive, ref } from 'vue'

import { apiErrorFields, apiErrorMessage, apiErrorReason, useApiClient } from '../app/composables/useApiClient'
import { isApiConnectionError, useApiConnectionError } from '../app/composables/useApiConnectionError'
import { isUnauthenticatedAuthError } from '../app/composables/useAuthSession'
import { registerErrorMessage } from '../app/utils/registerErrors'

describe('api error helpers', () => {
  test('reads backend envelope message from fetch error data', () => {
    const error = {
      data: {
        code: 401,
        message: '账号或密码不正确。',
        data: {
          reason: 'auth.invalid_credentials',
          fields: {
            login: ['请检查用户名或邮箱。']
          }
        }
      }
    }

    expect(apiErrorMessage(error)).toBe('账号或密码不正确。')
    expect(apiErrorReason(error)).toBe('auth.invalid_credentials')
    expect(apiErrorFields(error)).toEqual({ login: ['请检查用户名或邮箱。'] })
  })

  test('reads backend envelope message from proxied response data', () => {
    const error = {
      response: {
        _data: {
          code: 422,
          message: '密码不符合安全要求。',
          data: {
            reason: 'auth.password_policy',
            fields: {
              password: ['密码至少需要 12 个字符。']
            }
          }
        }
      }
    }

    expect(apiErrorMessage(error)).toBe('密码不符合安全要求。')
    expect(apiErrorReason(error)).toBe('auth.password_policy')
    expect(apiErrorFields(error)).toEqual({ password: ['密码至少需要 12 个字符。'] })
  })

  test('reads fields from a direct backend envelope', () => {
    const error = {
      code: 422,
      message: '注册失败',
      data: {
        reason: 'auth.register_invalid',
        fields: {
          username: ['请填写用户名。'],
          email: ['邮箱格式不正确。']
        }
      }
    }

    expect(apiErrorFields(error)).toEqual({
      username: ['请填写用户名。'],
      email: ['邮箱格式不正确。']
    })
  })
})

describe('api connection error helpers', () => {
  test('detects backend api gateway and network failures', () => {
    expect(isApiConnectionError({
      response: {
        status: 502,
        _data: {
          code: 502,
          message: 'Bad Gateway',
          data: { reason: 'server.unavailable' }
        }
      }
    })).toBe(true)

    expect(isApiConnectionError(new TypeError('Failed to fetch'))).toBe(true)
  })

  test('does not treat business validation errors as api connection failures', () => {
    expect(isApiConnectionError({
      data: {
        code: 422,
        message: '请检查填写内容。',
        data: {
          reason: 'auth.register_invalid',
          fields: {
            email: ['邮箱格式不正确。']
          }
        }
      }
    })).toBe(false)
  })
})

describe('useApiClient CSRF handling', () => {
  test('primes a csrf token before the first unsafe browser request', async () => {
    const calls: Array<{ url: string, options?: { method?: string, headers?: Record<string, string> } }> = []
    const csrfCookie = ref('')

    await withApiClientGlobals(csrfCookie, async () => {
      globalThis.$fetch = async (url: string, options?: { method?: string, headers?: Record<string, string> }) => {
        calls.push({ url, options })
        if (url === '/api/v1/health') {
          csrfCookie.value = 'fresh-token'
          return { code: 200, message: 'ok', data: {} }
        }
        return { code: 200, message: 'ok', data: { saved: true } }
      }

      const { request } = useApiClient()
      await request('/admin/web-options', {
        method: 'PUT',
        body: { options: [] }
      })
    })

    expect(calls.map((call) => call.url)).toEqual(['/api/v1/health', '/api/v1/admin/web-options'])
    expect(calls[1].options?.headers?.['X-Csrf-Token']).toBe('fresh-token')
  })

  test('refreshes csrf token and retries once after csrf.invalid', async () => {
    const postHeaders: Array<Record<string, string> | undefined> = []
    const csrfCookie = ref('stale-token')
    let postAttempts = 0

    await withApiClientGlobals(csrfCookie, async () => {
      globalThis.$fetch = async (url: string, options?: { method?: string, headers?: Record<string, string> }) => {
        if (url === '/api/v1/health') {
          csrfCookie.value = 'fresh-token'
          return { code: 200, message: 'ok', data: {} }
        }
        postAttempts += 1
        postHeaders.push(options?.headers)
        if (postAttempts === 1) {
          throw {
            data: {
              code: 403,
              message: 'CSRF token invalid.',
              data: { reason: 'csrf.invalid' }
            }
          }
        }
        return { code: 200, message: 'ok', data: { saved: true } }
      }

      const { request } = useApiClient()
      await request('/admin/web-options', {
        method: 'PUT',
        body: { options: [] }
      })
    })

    expect(postAttempts).toBe(2)
    expect(postHeaders[0]?.['X-Csrf-Token']).toBe('stale-token')
    expect(postHeaders[1]?.['X-Csrf-Token']).toBe('fresh-token')
  })
})

describe('useApiClient global connection error state', () => {
  test('opens the global api connection modal state after a backend api connection failure', async () => {
    const csrfCookie = ref('')
    const stateStore = createStateStore()

    await withApiClientGlobals(csrfCookie, async () => {
      globalThis.useState = stateStore.useState
      globalThis.$fetch = async () => {
        throw {
          response: {
            status: 502,
            _data: {
              code: 502,
              message: 'Bad Gateway',
              data: { reason: 'server.unavailable' }
            }
          }
        }
      }

      const { request } = useApiClient()
      await expect(request('/admin/overview')).rejects.toBeDefined()

      const { state } = useApiConnectionError()
      expect(state.value.open).toBe(true)
      expect(state.value.statusCode).toBe(502)
      expect(state.value.path).toBe('/admin/overview')
    })
  })

  test('keeps the global api connection modal state closed for business errors', async () => {
    const csrfCookie = ref('')
    const stateStore = createStateStore()

    await withApiClientGlobals(csrfCookie, async () => {
      globalThis.useState = stateStore.useState
      globalThis.$fetch = async () => {
        throw {
          data: {
            code: 422,
            message: '请检查填写内容。',
            data: {
              reason: 'auth.register_invalid'
            }
          }
        }
      }

      const { request } = useApiClient()
      await expect(request('/auth/register', { method: 'POST' })).rejects.toBeDefined()

      const { state } = useApiConnectionError()
      expect(state.value.open).toBe(false)
    })
  })
})

describe('register error helpers', () => {
  const translate = (key: string) => ({
    'errors.sessionUnavailable': '账号已创建，但自动登录失败，请直接登录。',
    'errors.registerFailed': '注册失败，请检查填写内容后重试。'
  })[key] || key

  test('uses backend session unavailable message when present', () => {
    const error = {
      data: {
        code: 503,
        message: '账号已创建，但自动登录失败，请直接登录。',
        data: {
          reason: 'auth.session_unavailable'
        }
      }
    }

    expect(registerErrorMessage(error, translate)).toBe('账号已创建，但自动登录失败，请直接登录。')
  })

  test('falls back to translated session unavailable message from reason', () => {
    const error = {
      data: {
        code: 503,
        message: '',
        data: {
          reason: 'auth.session_unavailable'
        }
      }
    }

    expect(registerErrorMessage(error, translate)).toBe('账号已创建，但自动登录失败，请直接登录。')
  })
})

describe('auth session refresh helpers', () => {
  test('treats auth.required envelopes as logged out', () => {
    const error = {
      data: {
        code: 401,
        message: '请先登录。',
        data: {
          reason: 'auth.required'
        }
      }
    }

    expect(isUnauthenticatedAuthError(error)).toBe(true)
  })

  test('does not treat transient API failures as logged out', () => {
    const error = {
      response: {
        status: 502,
        _data: {
          code: 502,
          message: 'Bad Gateway',
          data: {
            reason: 'server.unavailable'
          }
        }
      }
    }

    expect(isUnauthenticatedAuthError(error)).toBe(false)
  })
})

describe('register page submit guard', () => {
  test('does not send a second registration request while submitting', async () => {
    const page = await loadRegisterPageForSubmitTest()

    const firstSubmit = page.context.submitRegister()
    const secondSubmit = page.context.submitRegister()

    expect(page.registerRequests()).toHaveLength(1)

    page.resolveRegister()
    await firstSubmit
    await secondSubmit

    expect(page.sessionUser()?.username).toBe('codex')
  })

  test('shows a success toast after successful registration', async () => {
    const page = await loadRegisterPageForSubmitTest()

    const submit = page.context.submitRegister()
    page.resolveRegister()
    await submit

    expect(page.toasts()).toContainEqual(expect.objectContaining({
      color: 'success',
      icon: 'i-lucide-check',
      title: '注册成功，欢迎加入。',
      duration: 10000
    }))
  })

  test('marks session unavailable errors for login guidance', async () => {
    const page = await loadRegisterPageForSubmitTest({
      registerError: {
        data: {
          code: 503,
          message: '账号已创建，但自动登录失败，请直接登录。',
          data: {
            reason: 'auth.session_unavailable'
          }
        }
      }
    })

    await page.context.submitRegister()

    expect(page.context.sessionUnavailable.value).toBe(true)
  })
})

describe('login page navigation', () => {
  test('hydrates the current user and navigates after successful login', async () => {
    const page = await loadLoginPageForSubmitTest()

    await page.context.submitLogin()

    expect(page.loginRequests()).toHaveLength(1)
    expect(page.sessionUser()?.username).toBe('admin')
    expect(page.navigations()).toEqual(['/control-panel'])
  })

  test('shows a success toast after successful login', async () => {
    const page = await loadLoginPageForSubmitTest()

    await page.context.submitLogin()

    expect(page.toasts()).toContainEqual(expect.objectContaining({
      color: 'success',
      icon: 'i-lucide-check',
      title: '登录成功，欢迎回来。',
      duration: 10000
    }))
  })
})

const defaultThemePagesUrl = new URL(
  '../../../extensions/builtin/themes/sforum-default/layer/app/pages/',
  import.meta.url
)

async function loadRegisterPageForSubmitTest(options: { registerError?: unknown } = {}) {
  const source = readFileSync(new URL('register.vue', defaultThemePagesUrl), 'utf8')
  const { descriptor } = parse(source, { filename: 'register.vue' })
  const compiled = compileScript(descriptor, { id: 'register-submit-test' }).content
  const transpiler = new Bun.Transpiler({ loader: 'ts', target: 'browser' })
  const executable = transpiler.transformSync(compiled)
    .replace(
      /import\s+\{\s*withAsyncContext\s+as\s+_withAsyncContext,\s*defineComponent\s+as\s+_defineComponent\s*\}\s+from\s+['"]vue['"];?/,
      'const { withAsyncContext: _withAsyncContext, defineComponent: _defineComponent } = __vue;'
    )
    .replace(/export default /, 'return ')

  const requests: Array<{ path: string, options?: unknown }> = []
  const toasts: unknown[] = []
  let sessionUser: { username?: string } | null = null
  let resolveRegister: (value?: unknown) => void = () => {}

  const request = (path: string, requestOptions?: unknown) => {
    requests.push({ path, options: requestOptions })
    if (path === '/auth/registration-status') {
      return Promise.resolve({ nextUserIsInitialSuperAdmin: false })
    }
    if (options.registerError) {
      return Promise.reject(options.registerError)
    }
    return new Promise((resolve) => {
      resolveRegister = resolve
    })
  }

  const factory = new Function(
    '__vue',
    'definePageMeta',
    'useI18n',
    'useToast',
    'useLocalePath',
    'useRuntimeConfig',
    'useAdminRoutes',
    'useApiClient',
    'useAuthSession',
    'useWebOptions',
    'useAsyncData',
    'useSeoMeta',
    'reactive',
    'ref',
    'computed',
    'apiErrorFields',
    'apiErrorReason',
    'registerErrorMessage',
    'navigateTo',
    executable
  )
  const component = factory(
    {
      withAsyncContext: (fn: () => Promise<unknown>) => [fn(), () => {}],
      defineComponent: (options: unknown) => options
    },
    () => {},
    () => ({ t: (key: string) => key, locale: ref('zh-CN') }),
    () => ({ add: (toast: unknown) => toasts.push(toast) }),
    () => (path: string) => path,
    () => ({ public: { humanVerificationProvider: 'disabled' } }),
    () => ({ path: (path: string) => `/control-panel${path}` }),
    () => ({ apiBaseUrl: '/api/v1', request }),
    () => ({
      setUser: (currentUser: { username?: string } | null) => {
        sessionUser = currentUser
      },
      can: () => false
    }),
    () => ({ siteName: 'SForum', humanVerificationEnabledFor: () => false }),
    async (_key: string, loader: () => Promise<unknown>) => ({ data: ref(await loader()) }),
    () => {},
    reactive,
    ref,
    computed,
    () => ({}),
    apiErrorReason,
    () => 'register failed',
    async () => {}
  )
  const context = await component.setup({}, { expose: () => {} })

  return {
    context,
    registerRequests: () => requests.filter((request) => request.path === '/auth/register'),
    resolveRegister: () => resolveRegister({ username: 'codex' }),
    sessionUser: () => sessionUser,
    toasts: () => toasts
  }
}

async function loadLoginPageForSubmitTest() {
  const source = readFileSync(new URL('login.vue', defaultThemePagesUrl), 'utf8')
  const { descriptor } = parse(source, { filename: 'login.vue' })
  const compiled = compileScript(descriptor, { id: 'login-submit-test' }).content
  const transpiler = new Bun.Transpiler({ loader: 'ts', target: 'browser' })
  const executable = transpiler.transformSync(compiled)
    .replace(
      /import\s+\{\s*defineComponent\s+as\s+_defineComponent\s*\}\s+from\s+['"]vue['"];?/,
      'const { defineComponent: _defineComponent } = __vue;'
    )
    .replace(/export default /, 'return ')

  const requests: Array<{ path: string, options?: unknown }> = []
  const navigations: string[] = []
  const toasts: unknown[] = []
  let sessionUser: { username?: string, roleKeys?: string[], permissions?: string[] } | null = null

  const request = (path: string, requestOptions?: unknown) => {
    requests.push({ path, options: requestOptions })
    return Promise.resolve({
      username: 'admin',
      roleKeys: ['super_admin'],
      permissions: []
    })
  }

  const factory = new Function(
    '__vue',
    'definePageMeta',
    'useI18n',
    'useToast',
    'useLocalePath',
    'useAdminRoutes',
    'useApiClient',
    'useAuthSession',
    'useWebOptions',
    'useSeoMeta',
    'reactive',
    'ref',
    'apiErrorMessage',
    'navigateTo',
    executable
  )
  const component = factory(
    {
      defineComponent: (options: unknown) => options
    },
    () => {},
    () => ({ t: (key: string) => key, locale: ref('zh-CN') }),
    () => ({ add: (toast: unknown) => toasts.push(toast) }),
    () => (path: string) => path,
    () => ({ path: (path: string) => path === '/' ? '/control-panel' : `/control-panel${path}` }),
    () => ({ request }),
    () => ({
      setUser: (currentUser: { username?: string, roleKeys?: string[], permissions?: string[] } | null) => {
        sessionUser = currentUser
      },
      can: (permission: string) => {
        return Boolean(
          sessionUser?.permissions?.includes(permission) ||
          sessionUser?.roleKeys?.includes('super_admin')
        )
      }
    }),
    () => ({ siteName: 'SForum' }),
    () => {},
    reactive,
    ref,
    () => '',
    async (path: string) => {
      navigations.push(path)
    }
  )
  const context = await component.setup({}, { expose: () => {} })

  return {
    context,
    loginRequests: () => requests.filter((request) => request.path === '/auth/login'),
    navigations: () => navigations,
    sessionUser: () => sessionUser,
    toasts: () => toasts
  }
}

async function withApiClientGlobals(csrfCookie: { value: string }, run: () => Promise<void>) {
  const originalFetch = globalThis.$fetch
  const originalUseRuntimeConfig = globalThis.useRuntimeConfig
  const originalUseNuxtApp = globalThis.useNuxtApp
  const originalUseCookie = globalThis.useCookie
  const originalUseState = globalThis.useState

  globalThis.useRuntimeConfig = () => ({
    public: {
      apiBaseUrl: '/api/v1',
      appLocale: 'zh-CN'
    }
  })
  globalThis.useNuxtApp = () => ({
    $i18n: {
      locale: ref('zh-CN')
    }
  })
  globalThis.useCookie = (name: string) => {
    if (name !== 'csrf_') {
      throw new Error(`unexpected cookie ${name}`)
    }
    return csrfCookie
  }

  try {
    await run()
  } finally {
    globalThis.$fetch = originalFetch
    globalThis.useRuntimeConfig = originalUseRuntimeConfig
    globalThis.useNuxtApp = originalUseNuxtApp
    globalThis.useCookie = originalUseCookie
    globalThis.useState = originalUseState
  }
}

function createStateStore() {
  const states = new Map<string, { value: unknown }>()

  return {
    useState: <T>(key: string, init: () => T) => {
      if (!states.has(key)) {
        states.set(key, ref(init()))
      }
      return states.get(key) as { value: T }
    }
  }
}
