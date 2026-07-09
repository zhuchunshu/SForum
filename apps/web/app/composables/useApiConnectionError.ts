export type ApiConnectionErrorState = {
  open: boolean
  message: string
  reason: string
  statusCode: number | null
  path: string
  failedAt: number | null
}

type ErrorRecord = Record<string, unknown>

const API_CONNECTION_ERROR_STATE_KEY = 'api:connection-error'
const API_CONNECTION_STATUS_CODES = new Set([502, 503, 504])
const API_CONNECTION_REASONS = new Set(['server.unavailable'])
const NETWORK_ERROR_PATTERN = /(failed to fetch|fetch failed|networkerror|load failed|timeout|timed out|econnrefused|econnreset|enotfound|etimedout|eai_again|und_err_connect_timeout)/i

export function useApiConnectionError() {
  const state = useState<ApiConnectionErrorState>(API_CONNECTION_ERROR_STATE_KEY, () => ({
    open: false,
    message: '',
    reason: '',
    statusCode: null,
    path: '',
    failedAt: null
  }))

  function show(error: unknown, path = '') {
    if (!isApiConnectionError(error)) {
      return
    }

    // 多个并发请求同时失败时保留第一个错误，避免弹窗内容反复跳动。
    if (state.value.open) {
      return
    }

    state.value = {
      open: true,
      message: apiConnectionErrorMessage(error),
      reason: apiConnectionErrorReason(error),
      statusCode: apiConnectionErrorStatusCode(error),
      path,
      failedAt: Date.now()
    }
  }

  function close() {
    state.value.open = false
  }

  function reload() {
    if (import.meta.server) {
      return
    }
    window.location.reload()
  }

  return { state, show, close, reload }
}

export function notifyApiConnectionError(error: unknown, path = '') {
  if (import.meta.server || !isApiConnectionError(error)) {
    return
  }

  const { show } = useApiConnectionError()
  show(error, path)
}

export function isApiConnectionError(error: unknown) {
  const statusCode = apiConnectionErrorStatusCode(error)
  if (statusCode && API_CONNECTION_STATUS_CODES.has(statusCode)) {
    return true
  }

  const reason = apiConnectionErrorReason(error)
  if (reason && API_CONNECTION_REASONS.has(reason)) {
    return true
  }

  return NETWORK_ERROR_PATTERN.test([
    apiConnectionErrorMessage(error),
    apiConnectionErrorCauseCode(error)
  ].filter(Boolean).join(' '))
}

export function apiConnectionErrorStatusCode(error: unknown) {
  const direct = asRecord(error)
  const response = asRecord(direct?.response)
  const data = asRecord(direct?.data)
  const responseData = asRecord(response?._data)
  const candidates = [
    direct?.status,
    direct?.statusCode,
    response?.status,
    data?.code,
    responseData?.code
  ]

  for (const candidate of candidates) {
    if (typeof candidate === 'number') {
      return candidate
    }
  }

  return null
}

function apiConnectionErrorMessage(error: unknown) {
  if (error instanceof Error) {
    return error.message
  }

  const direct = asRecord(error)
  const data = asRecord(direct?.data)
  const responseData = asRecord(asRecord(direct?.response)?._data)
  const candidates = [
    direct?.message,
    data?.message,
    responseData?.message
  ]

  for (const candidate of candidates) {
    if (typeof candidate === 'string') {
      return candidate
    }
  }

  return ''
}

function apiConnectionErrorReason(error: unknown) {
  const direct = asRecord(error)
  const directData = asRecord(direct?.data)
  const responseData = asRecord(asRecord(direct?.response)?._data)
  const directEnvelopeData = asRecord(directData?.data)
  const responseEnvelopeData = asRecord(responseData?.data)
  const candidates = [
    directData?.reason,
    directEnvelopeData?.reason,
    responseEnvelopeData?.reason
  ]

  for (const candidate of candidates) {
    if (typeof candidate === 'string') {
      return candidate
    }
  }

  return ''
}

function apiConnectionErrorCauseCode(error: unknown) {
  const cause = asRecord(asRecord(error)?.cause)
  const code = cause?.code
  return typeof code === 'string' ? code : ''
}

function asRecord(value: unknown) {
  return value && typeof value === 'object' ? value as ErrorRecord : undefined
}
