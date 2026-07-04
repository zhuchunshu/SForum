import { describe, expect, test } from 'bun:test'
import { readFileSync } from 'node:fs'
import { parse, compileScript } from '@vue/compiler-sfc'
import { computed, reactive, ref } from 'vue'

import { apiErrorFields, apiErrorMessage, apiErrorReason } from '../app/composables/useApiClient'
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

describe('register page submit guard', () => {
  test('does not send a second registration request while submitting', async () => {
    const page = await loadRegisterPageForSubmitTest()

    const firstSubmit = page.context.submitRegister()
    const secondSubmit = page.context.submitRegister()

    expect(page.registerRequests()).toHaveLength(1)

    page.resolveRegister()
    await firstSubmit
    await secondSubmit
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

async function loadRegisterPageForSubmitTest(options: { registerError?: unknown } = {}) {
  const source = readFileSync(new URL('../app/pages/register.vue', import.meta.url), 'utf8')
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
    'useLocalePath',
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
    () => (path: string) => path,
    () => ({ path: (path: string) => `/control-panel${path}` }),
    () => ({ apiBaseUrl: '/api/v1', request }),
    () => ({ refresh: async () => {}, can: () => false }),
    () => ({ siteName: 'SForum' }),
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
    resolveRegister: () => resolveRegister({})
  }
}
