import { notifyApiConnectionError } from './useApiConnectionError'

export type ApiEnvelope<T> = {
  code: number
  message: string
  data: T
}

export type ApiErrorData = {
  reason?: string
  fields?: Record<string, string[]>
}

type ApiRequestBody = BodyInit | Record<string, unknown> | null

type ApiFetchOptions = {
  method?: 'GET' | 'POST' | 'PATCH' | 'PUT' | 'DELETE'
  body?: ApiRequestBody
  credentials?: RequestCredentials
  headers?: Record<string, string>
  timeout?: number
  /** Nuxt 错误渲染等上下文必须绕过相对 URL，直连服务端 API。 */
  serverInternal?: boolean
}

const UNSAFE_METHODS = new Set(['POST', 'PATCH', 'PUT', 'DELETE'])
const CSRF_COOKIE_NAME = 'csrf_'
const CSRF_HEADER_NAME = 'X-Csrf-Token'
const CSRF_PRIME_PATH = '/health'

type ApiErrorEnvelopeLike = {
  code?: unknown
  message?: unknown
  data?: unknown
}

type ApiFetchErrorLike = {
  data?: unknown
  status?: unknown
  statusCode?: unknown
  response?: {
    status?: unknown
    _data?: unknown
  }
}

type RuntimeI18nLike = {
  locale?: unknown
}

export function useApiClient() {
  const runtimeConfig = useRuntimeConfig()
  const apiBaseUrl = runtimeConfig.public.apiBaseUrl as string
  const i18n = useNuxtApp().$i18n as RuntimeI18nLike | undefined
  // SSR 重试可能跨过 await；请求 cookie 必须在 composable 创建时从当前上下文捕获。
  const requestCookie = import.meta.server
    ? (useRequestHeaders(['cookie']).cookie || '')
    : ''

  function apiLocale() {
    const locale = localeString(i18n?.locale) || String(runtimeConfig.public.appLocale || 'zh-CN')
    return locale === 'en' ? 'en-US' : locale
  }

  function apiHeaders(extra?: Record<string, string>): Record<string, string> {
    const headers: Record<string, string> = {
      'Accept-Language': apiLocale()
    }
    if (requestCookie) {
      headers.cookie = requestCookie
    }
    return { ...headers, ...extra }
  }

  // csrfToken 读取后端在 GET 请求种下的 csrf_ cookie：
  // client 端用 useCookie 读浏览器 cookie；server 端（SSR）从透传的请求 cookie 头解析。
  // 返回空字符串表示无 token（首次访问尚未种下，unsafe 请求会被后端拒绝，正常流程先有 GET）。
  function csrfToken(): string {
    if (import.meta.server) {
      for (const part of requestCookie.split(';')) {
        const [k, ...rest] = part.trim().split('=')
        if (k === CSRF_COOKIE_NAME) {
          return decodeURIComponent(rest.join('='))
        }
      }
      return ''
    }
    return useCookie<string>(CSRF_COOKIE_NAME).value || ''
  }

  async function refreshCsrfToken() {
    if (import.meta.server) {
      return csrfToken()
    }

    await $fetch<ApiEnvelope<unknown>>(`${apiBaseUrl}${CSRF_PRIME_PATH}`, {
      credentials: 'include',
      headers: apiHeaders(),
      timeout: 2000
    })

    return csrfToken()
  }

  async function request<T>(path: string, options: ApiFetchOptions = {}) {
    const method = options.method?.toUpperCase()
    const unsafe = Boolean(method && UNSAFE_METHODS.has(method))
    const callerProvidedCsrf = Boolean(options.headers?.[CSRF_HEADER_NAME])
    const canRefreshCsrf = !import.meta.server

    async function requestHeaders(refreshToken: boolean) {
      // 首次进入页面或 token 过期时，先用安全 GET 让后端种/刷新 double-submit cookie。
      if (canRefreshCsrf && unsafe && !callerProvidedCsrf && (refreshToken || !csrfToken())) {
        await refreshCsrfToken()
      }

      const headers = apiHeaders(options.headers)
      // unsafe 方法必须携带 CSRF token（double-submit：cookie 值 == X-Csrf-Token header）。
      if (unsafe && !callerProvidedCsrf) {
        const token = csrfToken()
        if (token) {
          headers[CSRF_HEADER_NAME] = token
        }
      }
      return headers
    }

    async function send(refreshToken = false) {
      const headers = await requestHeaders(refreshToken)
      const requestBaseUrl = import.meta.server && options.serverInternal
        ? (process.env.NUXT_API_INTERNAL_BASE_URL || 'http://api:8080/api/v1')
        : apiBaseUrl
      const envelope = await $fetch<ApiEnvelope<T>>(`${requestBaseUrl.replace(/\/+$/, '')}${path}`, {
        method: options.method,
        body: options.body,
        credentials: options.credentials ?? 'include',
        headers,
        timeout: options.timeout
      })

      return envelope.data
    }

    try {
      return await send()
    } catch (error) {
      if (canRefreshCsrf && unsafe && !callerProvidedCsrf && apiErrorReason(error) === 'csrf.invalid') {
        try {
          return await send(true)
        } catch (retryError) {
          notifyApiConnectionError(retryError, path)
          throw retryError
        }
      }
      notifyApiConnectionError(error, path)
      throw error
    }
  }

  return { apiBaseUrl, apiHeaders, request }
}

function localeString(value: unknown) {
  if (typeof value === 'string') {
    return value
  }
  if (value && typeof value === 'object' && 'value' in value) {
    const refValue = (value as { value?: unknown }).value
    return typeof refValue === 'string' ? refValue : ''
  }
  return ''
}

function isApiEnvelope(value: unknown): value is ApiEnvelope<ApiErrorData> {
  if (!value || typeof value !== 'object') {
    return false
  }

  const envelope = value as ApiErrorEnvelopeLike
  return typeof envelope.code === 'number' && typeof envelope.message === 'string'
}

function apiErrorEnvelope(error: unknown) {
  if (isApiEnvelope(error)) {
    return error
  }

  // ofetch/Nuxt 在不同链路里可能把后端 envelope 放在 data 或 response._data。
  const fetchError = error && typeof error === 'object'
    ? error as ApiFetchErrorLike
    : {}
  if (isApiEnvelope(fetchError.data)) {
    return fetchError.data
  }
  if (isApiEnvelope(fetchError.response?._data)) {
    return fetchError.response._data
  }

  return undefined
}

export function apiErrorMessage(error: unknown) {
  return apiErrorEnvelope(error)?.message || ''
}

export function apiErrorReason(error: unknown) {
  const reason = apiErrorEnvelope(error)?.data?.reason
  return typeof reason === 'string' ? reason : ''
}

/** 统一读取 ofetch 与 Host envelope 的 HTTP 状态，供只读请求策略复用。 */
export function apiErrorStatusCode(error: unknown) {
  const fetchError = error && typeof error === 'object'
    ? error as ApiFetchErrorLike
    : {}
  const candidates = [
    apiErrorEnvelope(error)?.code,
    fetchError.statusCode,
    fetchError.status,
    fetchError.response?.status
  ]
  for (const value of candidates) {
    const status = Number(value)
    if (Number.isInteger(status) && status >= 400 && status <= 599) {
      return status
    }
  }
  return 0
}

export function apiErrorFields(error: unknown) {
  const fields = apiErrorEnvelope(error)?.data?.fields
  if (!fields || typeof fields !== 'object') {
    return {}
  }

  return Object.fromEntries(
    Object.entries(fields).filter(([, messages]) => Array.isArray(messages))
  ) as Record<string, string[]>
}
