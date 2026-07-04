export type ApiEnvelope<T> = {
  code: number
  message: string
  data: T
}

export type ApiErrorData = {
  reason?: string
  fields?: Record<string, string[]>
}

type ApiFetchOptions = {
  method?: 'GET' | 'POST' | 'PATCH' | 'PUT' | 'DELETE'
  body?: unknown
  credentials?: RequestCredentials
  headers?: Record<string, string>
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
      headers: apiHeaders(options.headers)
    })

    return envelope.data
  }

  return { apiBaseUrl, apiHeaders, request }
}

export function apiErrorMessage(error: unknown) {
  const message = (error as { data?: { message?: unknown } })?.data?.message
  return typeof message === 'string' ? message : ''
}

export function apiErrorReason(error: unknown) {
  const reason = (error as { data?: ApiEnvelope<ApiErrorData> })?.data?.data?.reason
  return typeof reason === 'string' ? reason : ''
}
