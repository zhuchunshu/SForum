import type { H3Event } from 'h3'
import { createError, getProxyRequestHeaders, getRequestURL, getRequestWebStream, sendProxy } from 'h3'

export const INTERNAL_ROUTE_PROBE_HEADER = 'x-sforum-internal-route-probe'
export const INTERNAL_ROUTE_METHOD_HEADER = 'x-sforum-internal-route-method'
export const INTERNAL_ROUTE_RESULT_HEADER = 'x-sforum-internal-route-result'
export const INTERNAL_ROUTE_PROBE_VERSION = 'v1'
export const INTERNAL_ROUTE_MATCH = 'plugin'
export const INTERNAL_ROUTE_MISS = 'miss'

type ProbeFetch = (input: string | URL | Request, init?: RequestInit) => Promise<Response>

export function isCoreAPIPath(pathname: string) {
  return pathname === '/api/v1' || pathname.startsWith('/api/v1/')
}

export function buildPluginRouteTarget(apiBaseURL: string, requestURL: URL) {
  const target = new URL(apiBaseURL)
  if ((target.protocol !== 'http:' && target.protocol !== 'https:') || target.username || target.password) {
    throw new Error('NUXT_API_INTERNAL_BASE_URL must be an HTTP(S) URL without credentials')
  }
  target.pathname = requestURL.pathname
  target.search = requestURL.search
  target.hash = ''
  return target
}

export function pluginRouteProxyHeaders(input: HeadersInit) {
  const headers = new Headers(input)
  headers.delete(INTERNAL_ROUTE_PROBE_HEADER)
  headers.delete(INTERNAL_ROUTE_METHOD_HEADER)
  headers.delete(INTERNAL_ROUTE_RESULT_HEADER)
  return headers
}

export async function probePluginRoute(target: URL, method: string, fetcher: ProbeFetch = fetch) {
  const response = await fetcher(target, {
    method: 'HEAD',
    redirect: 'manual',
    headers: {
      [INTERNAL_ROUTE_PROBE_HEADER]: INTERNAL_ROUTE_PROBE_VERSION,
      [INTERNAL_ROUTE_METHOD_HEADER]: method
    }
  })
  const result = response.headers.get(INTERNAL_ROUTE_RESULT_HEADER)
  if (response.status === 204 && result === INTERNAL_ROUTE_MATCH) {
    return INTERNAL_ROUTE_MATCH
  }
  if (response.status === 404 && result === INTERNAL_ROUTE_MISS) {
    return INTERNAL_ROUTE_MISS
  }
  throw new Error('route probe returned invalid Host evidence')
}

export function canFallbackAfterRouteProbeFailure(method: string) {
  return method === 'GET' || method === 'HEAD'
}

export async function proxyDeclaredPluginRoute(event: H3Event) {
  const requestURL = getRequestURL(event)
  if (isCoreAPIPath(requestURL.pathname)) {
    return
  }

  const apiBaseURL = process.env.NUXT_API_INTERNAL_BASE_URL || 'http://api:8080/api/v1'
  const target = buildPluginRouteTarget(apiBaseURL, requestURL)
  try {
    if (await probePluginRoute(target, event.method) === INTERNAL_ROUTE_MISS) {
      return
    }
  } catch {
    // 无法证明 miss 时，unsafe 请求不能落到另一个潜在 writer。
    if (canFallbackAfterRouteProbeFailure(event.method)) {
      return
    }
    throw createError({ statusCode: 503, statusMessage: 'Route registry unavailable' })
  }

  return proxyRouteRequest(event, target)
}

export function proxyRouteRequest(event: H3Event, target: URL) {
  const headers = pluginRouteProxyHeaders(getProxyRequestHeaders(event))
  const hasRequestBody = event.method !== 'GET' && event.method !== 'HEAD'
  return sendProxy(event, target.toString(), {
    headers,
    sendStream: true,
    fetchOptions: {
      method: event.method,
      redirect: 'manual',
      body: hasRequestBody ? getRequestWebStream(event) : undefined,
      duplex: hasRequestBody ? 'half' : undefined
    }
  })
}
