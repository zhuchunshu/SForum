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

export function useApiClient() {
  const apiBaseUrl = useRuntimeConfig().public.apiBaseUrl as string
  const { locale } = useI18n()

  function apiLocale() {
    return locale.value === 'en' ? 'en-US' : locale.value
  }

  function apiHeaders(extra?: Record<string, string>) {
    return {
      ...(import.meta.server ? useRequestHeaders(['cookie']) : {}),
      'Accept-Language': apiLocale(),
      ...extra
    }
  }

  async function request<T>(path: string, options: ApiFetchOptions = {}) {
    const envelope = await $fetch<ApiEnvelope<T>>(`${apiBaseUrl}${path}`, {
      method: options.method,
      body: options.body,
      credentials: options.credentials ?? 'include',
      headers: apiHeaders(options.headers),
      timeout: options.timeout
    })

    return envelope.data
  }

  return { apiBaseUrl, apiHeaders, request }
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
