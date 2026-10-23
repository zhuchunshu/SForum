// API 连接错误分类纯函数：从 useApiConnectionError.ts 抽出，供门面与探测
// 引擎共享。只做判定，不触达 Nuxt 上下文，可在任意调用环境安全使用。

type ErrorRecord = Record<string, unknown>

const API_CONNECTION_STATUS_CODES = new Set([502, 503, 504])
const API_CONNECTION_REASONS = new Set(['server.unavailable'])
const NETWORK_ERROR_PATTERN = /(failed to fetch|fetch failed|networkerror|load failed|timeout|timed out|econnrefused|econnreset|enotfound|etimedout|eai_again|und_err_connect_timeout)/i

export function isApiConnectionError(error: unknown) {
  const statusCode = apiConnectionErrorStatusCode(error)
  const reason = apiConnectionErrorReason(error)
  if (statusCode && API_CONNECTION_STATUS_CODES.has(statusCode)) {
    // 结构化 5xx 业务错误（例如插件 Probe 暂不可用）应留在当前操作旁，不应误报为整个 API 断线。
    return !reason || API_CONNECTION_REASONS.has(reason)
  }

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

export function apiConnectionErrorReason(error: unknown) {
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

export function apiConnectionErrorMessage(error: unknown) {
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

function apiConnectionErrorCauseCode(error: unknown) {
  const cause = asRecord(asRecord(error)?.cause)
  const code = cause?.code
  return typeof code === 'string' ? code : ''
}

function asRecord(value: unknown) {
  return value && typeof value === 'object' ? value as ErrorRecord : undefined
}
