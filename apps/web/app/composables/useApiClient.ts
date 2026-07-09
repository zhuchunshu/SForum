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
}

const UNSAFE_METHODS = new Set(['POST', 'PATCH', 'PUT', 'DELETE'])
const CSRF_COOKIE_NAME = 'csrf_'
const CSRF_HEADER_NAME = 'X-Csrf-Token'

type ApiErrorEnvelopeLike = {
  code?: unknown
  message?: unknown
  data?: unknown
}

type ApiFetchErrorLike = {
  data?: unknown
  response?: {
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

  function apiLocale() {
    const locale = localeString(i18n?.locale) || String(runtimeConfig.public.appLocale || 'zh-CN')
    return locale === 'en' ? 'en-US' : locale
  }

  function apiHeaders(extra?: Record<string, string>): Record<string, string> {
    const headers: Record<string, string> = {
      'Accept-Language': apiLocale()
    }
    if (import.meta.server) {
      const cookie = useRequestHeaders(['cookie']).cookie
      if (cookie) {
        headers.cookie = cookie
      }
    }
    return { ...headers, ...extra }
  }

  // csrfToken 读取后端在 GET 请求种下的 csrf_ cookie：
  // client 端用 useCookie 读浏览器 cookie；server 端（SSR）从透传的请求 cookie 头解析。
  // 返回空字符串表示无 token（首次访问尚未种下，unsafe 请求会被后端拒绝，正常流程先有 GET）。
  function csrfToken(): string {
    if (import.meta.server) {
      const raw = useRequestHeaders(['cookie']).cookie || ''
      for (const part of raw.split(';')) {
        const [k, ...rest] = part.trim().split('=')
        if (k === CSRF_COOKIE_NAME) {
          return decodeURIComponent(rest.join('='))
        }
      }
      return ''
    }
    return useCookie<string>(CSRF_COOKIE_NAME).value || ''
  }

  async function request<T>(path: string, options: ApiFetchOptions = {}) {
    const headers = apiHeaders(options.headers)
    // unsafe 方法必须携带 CSRF token（double-submit：cookie 值 == X-Csrf-Token header）。
    if (options.method && UNSAFE_METHODS.has(options.method)) {
      const token = csrfToken()
      if (token && !headers[CSRF_HEADER_NAME]) {
        headers[CSRF_HEADER_NAME] = token
      }
    }
    const envelope = await $fetch<ApiEnvelope<T>>(`${apiBaseUrl}${path}`, {
      method: options.method,
      body: options.body,
      credentials: options.credentials ?? 'include',
      headers,
      timeout: options.timeout
    })

    return envelope.data
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
  const fetchError = error as ApiFetchErrorLike
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

export function apiErrorFields(error: unknown) {
  const fields = apiErrorEnvelope(error)?.data?.fields
  if (!fields || typeof fields !== 'object') {
    return {}
  }

  return Object.fromEntries(
    Object.entries(fields).filter(([, messages]) => Array.isArray(messages))
  ) as Record<string, string[]>
}
