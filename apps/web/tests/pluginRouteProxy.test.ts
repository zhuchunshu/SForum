import { describe, expect, test } from 'bun:test'
import { readFileSync } from 'node:fs'
import { createServer, type Server } from 'node:http'
import { createApp, eventHandler, getRequestURL, toNodeListener } from 'h3'

import {
  buildPluginRouteTarget,
  canFallbackAfterRouteProbeFailure,
  INTERNAL_ROUTE_METHOD_HEADER,
  INTERNAL_ROUTE_MISS,
  INTERNAL_ROUTE_MATCH,
  INTERNAL_ROUTE_PROBE_HEADER,
  INTERNAL_ROUTE_PROBE_VERSION,
  INTERNAL_ROUTE_RESULT_HEADER,
  isCoreAPIPath,
  pluginRouteProxyHeaders,
  probePluginRoute,
  proxyDeclaredPluginRoute
} from '../server/utils/pluginRouteProxy'

const source = (path: string) => readFileSync(new URL(path, import.meta.url), 'utf8')

describe('trusted plugin arbitrary-route proxy', () => {
  test('keeps core API paths on the existing proxy and admits root public/admin paths', () => {
    expect(isCoreAPIPath('/api/v1/topics')).toBe(true)
    expect(isCoreAPIPath('/api/v10/topics')).toBe(false)
    expect(isCoreAPIPath('/plugin/docs')).toBe(false)
    expect(isCoreAPIPath('/admin/plugin/rebuild')).toBe(false)
  })

  test('builds only same configured API-origin targets without path-based SSRF', () => {
    const target = buildPluginRouteTarget(
      'http://api:8080/api/v1',
      new URL('https://forum.example.com//evil.example/path?q=1')
    )
    expect(target.origin).toBe('http://api:8080')
    expect(target.pathname).toBe('//evil.example/path')
    expect(target.search).toBe('?q=1')
    expect(() => buildPluginRouteTarget('file:///tmp/api', new URL('https://forum.example.com/x'))).toThrow()
    expect(() => buildPluginRouteTarget('http://user:secret@api:8080', new URL('https://forum.example.com/x'))).toThrow()
  })

  test('uses a body-free HEAD probe bound to the original method', async () => {
    let requestInit: RequestInit | undefined
    const matched = await probePluginRoute(new URL('http://api:8080/admin/plugin'), 'PATCH', async (_input, init) => {
      requestInit = init
      return new Response(null, {
        status: 204,
        headers: { [INTERNAL_ROUTE_RESULT_HEADER]: 'plugin' }
      })
    })
    expect(matched).toBe(INTERNAL_ROUTE_MATCH)
    expect(requestInit?.method).toBe('HEAD')
    expect(new Headers(requestInit?.headers).get(INTERNAL_ROUTE_PROBE_HEADER)).toBe(INTERNAL_ROUTE_PROBE_VERSION)
    expect(new Headers(requestInit?.headers).get(INTERNAL_ROUTE_METHOD_HEADER)).toBe('PATCH')
  })

  test('requires Host match evidence and strips all reserved authority headers', async () => {
    expect(await probePluginRoute(new URL('http://api:8080/x'), 'GET', async () => new Response(null, {
      status: 404,
      headers: { [INTERNAL_ROUTE_RESULT_HEADER]: INTERNAL_ROUTE_MISS }
    }))).toBe(INTERNAL_ROUTE_MISS)
    await expect(probePluginRoute(new URL('http://api:8080/x'), 'GET', async () => new Response(null, { status: 404 }))).rejects.toThrow()
    await expect(probePluginRoute(new URL('http://api:8080/x'), 'GET', async () => new Response(null, { status: 204 }))).rejects.toThrow()

    expect(canFallbackAfterRouteProbeFailure('GET')).toBe(true)
    expect(canFallbackAfterRouteProbeFailure('HEAD')).toBe(true)
    expect(canFallbackAfterRouteProbeFailure('POST')).toBe(false)

    const headers = pluginRouteProxyHeaders({
      authorization: 'Bearer sft_real',
      [INTERNAL_ROUTE_PROBE_HEADER]: 'forged',
      [INTERNAL_ROUTE_METHOD_HEADER]: 'DELETE',
      [INTERNAL_ROUTE_RESULT_HEADER]: 'plugin'
    })
    expect(headers.get('authorization')).toBe('Bearer sft_real')
    expect(headers.get(INTERNAL_ROUTE_PROBE_HEADER)).toBeNull()
    expect(headers.get(INTERNAL_ROUTE_METHOD_HEADER)).toBeNull()
    expect(headers.get(INTERNAL_ROUTE_RESULT_HEADER)).toBeNull()
  })

  test('middleware probes before proxying while ordinary API proxy remains streaming', () => {
    const middleware = source('../server/middleware/plugin-route-proxy.ts')
    const apiProxy = source('../server/routes/api/v1/[...path].ts')
    const proxyUtility = source('../server/utils/pluginRouteProxy.ts')
    expect(middleware).toContain('proxyDeclaredPluginRoute(event)')
    expect(apiProxy).toContain('proxyRouteRequest(event, target)')
    expect(proxyUtility).toContain('getRequestWebStream(event)')
    expect(proxyUtility).toContain('sendStream: true')
    expect(proxyUtility).toContain("duplex: hasRequestBody ? 'half' : undefined")

    const caddy = source('../../../deploy/caddy/Caddyfile')
    const websocketProxy = 'reverse_proxy @host_api_websocket 127.0.0.1:{$API_PORT:18080}'
    const webProxy = 'reverse_proxy 127.0.0.1:{$WEB_PORT:3000}'
    expect(caddy).toContain('header_regexp connection_upgrade Connection (?i)(^|.*,\\s*)upgrade(\\s*,.*|$)')
    expect(caddy).toContain('header_regexp websocket_upgrade Upgrade (?i)^websocket$')
    expect(caddy).toContain('not header Sec-WebSocket-Protocol *vite-hmr*')
    expect(caddy).toContain(websocketProxy)
    expect(caddy).toContain(webProxy)
    expect(caddy.indexOf(websocketProxy)).toBeLessThan(caddy.indexOf(webProxy))
    expect(caddy).not.toMatch(/header_up\s+(?:Host|Origin|Cookie|Authorization|Sec-WebSocket-\S+)/i)

    const productionCompose = source('../../../compose.prod.yaml')
    expect(productionCompose).toContain('- "127.0.0.1:${API_PORT:-18080}:8080"')
    expect(source('../../../.env.production.example')).toContain('API_PORT=18080')
  })

  test('proxies matched unsafe bodies and leaves explicit misses to Nuxt', async () => {
    const actualRequests: Array<{ method?: string, body: string, authorization?: string, probe?: string }> = []
    const api = createServer(async (request, response) => {
      const url = new URL(request.url || '/', 'http://api.test')
      if (request.method === 'HEAD' && request.headers[INTERNAL_ROUTE_PROBE_HEADER] === INTERNAL_ROUTE_PROBE_VERSION) {
        if (url.pathname === '/probe-invalid') {
          response.statusCode = 500
          response.end()
          return
        }
        const matched = url.pathname === '/plugin/echo'
        response.statusCode = matched ? 204 : 404
        response.setHeader(INTERNAL_ROUTE_RESULT_HEADER, matched ? INTERNAL_ROUTE_MATCH : INTERNAL_ROUTE_MISS)
        response.end()
        return
      }

      const chunks: Buffer[] = []
      for await (const chunk of request) {
        chunks.push(Buffer.from(chunk))
      }
      const body = Buffer.concat(chunks).toString()
      actualRequests.push({
        method: request.method,
        body,
        authorization: request.headers.authorization,
        probe: request.headers[INTERNAL_ROUTE_PROBE_HEADER] as string | undefined
      })
      response.statusCode = 200
      response.end(`plugin:${request.method}:${url.search}:${body}`)
    })
    const apiURL = await listen(api)

    const app = createApp()
    app.use(eventHandler(event => proxyDeclaredPluginRoute(event)))
    app.use(eventHandler(event => `nuxt:${event.method}:${getRequestURL(event).pathname}`))
    const web = createServer(toNodeListener(app))
    const webURL = await listen(web)
    const oldBaseURL = process.env.NUXT_API_INTERNAL_BASE_URL
    process.env.NUXT_API_INTERNAL_BASE_URL = `${apiURL}/api/v1`

    try {
      const pluginResponse = await fetch(`${webURL}/plugin/echo?q=1`, {
        method: 'POST',
        headers: {
          authorization: 'Bearer sft_real',
          'content-type': 'text/plain',
          [INTERNAL_ROUTE_PROBE_HEADER]: INTERNAL_ROUTE_PROBE_VERSION,
          [INTERNAL_ROUTE_METHOD_HEADER]: 'DELETE'
        },
        body: 'streamed-body'
      })
      expect(pluginResponse.status).toBe(200)
      expect(await pluginResponse.text()).toBe('plugin:POST:?q=1:streamed-body')
      expect(actualRequests).toEqual([{
        method: 'POST', body: 'streamed-body', authorization: 'Bearer sft_real', probe: undefined
      }])

      const hostResponse = await fetch(`${webURL}/host-page`)
      expect(hostResponse.status).toBe(200)
      expect(await hostResponse.text()).toBe('nuxt:GET:/host-page')

      const unsafeUnavailable = await fetch(`${webURL}/probe-invalid`, { method: 'POST', body: 'must-not-fallback' })
      expect(unsafeUnavailable.status).toBe(503)
      expect(actualRequests).toHaveLength(1)

      const safeUnavailable = await fetch(`${webURL}/probe-invalid`)
      expect(safeUnavailable.status).toBe(200)
      expect(await safeUnavailable.text()).toBe('nuxt:GET:/probe-invalid')
    } finally {
      if (oldBaseURL === undefined) {
        delete process.env.NUXT_API_INTERNAL_BASE_URL
      } else {
        process.env.NUXT_API_INTERNAL_BASE_URL = oldBaseURL
      }
      await close(web)
      await close(api)
    }
  })
})

async function listen(server: Server) {
  await new Promise<void>((resolve, reject) => {
    server.once('error', reject)
    server.listen(0, '127.0.0.1', resolve)
  })
  const address = server.address()
  if (!address || typeof address === 'string') {
    throw new Error('test server did not expose a TCP address')
  }
  return `http://127.0.0.1:${address.port}`
}

async function close(server: Server) {
  server.closeAllConnections()
  if (server.listening) {
    server.close()
  }
}
