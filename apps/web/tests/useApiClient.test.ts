import { describe, expect, test } from 'bun:test'

import { apiErrorFields, apiErrorMessage, apiErrorReason } from '../app/composables/useApiClient'

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
