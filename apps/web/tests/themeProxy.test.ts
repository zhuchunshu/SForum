import { describe, expect, test, afterAll, beforeAll } from 'bun:test'
import http from 'node:http'
import net from 'node:net'
import os from 'node:os'
import path from 'node:path'
import fs from 'node:fs'

import {
  createThemeProxy,
  formatPublicDevUrl,
  healthCheckTcp,
  healthCheckUnix,
  isNuxtDevAddressLine,
  makeTarget,
  parseDevPort,
  replaceTarget,
} from '../scripts/theme-proxy.mjs'

// 找一个空闲端口供测试的代理或上游使用。
function freePort(): Promise<number> {
  return new Promise((resolve, reject) => {
    const srv = net.createServer()
    srv.unref()
    srv.on('error', reject)
    srv.listen(0, '127.0.0.1', () => {
      const port = (srv.address() as net.AddressInfo).port
      srv.close(() => resolve(port))
    })
  })
}

// 起一个伪造上游，把入站请求的标记头回显出来，便于断言代理是否转发正确。
function startUpstream(marker: string): Promise<{ server: http.Server; port: number; requests: http.IncomingMessage[] }> {
  return new Promise(async (resolve, reject) => {
    const requests: http.IncomingMessage[] = []
    const server = http.createServer((req, res) => {
      let body = ''
      req.on('data', (chunk) => {
        body += chunk
      })
      req.on('end', () => {
        requests.push(req)
        res.writeHead(200, { 'content-type': 'application/json', 'x-upstream': marker })
        res.end(JSON.stringify({ marker, method: req.method, path: req.url, body, headers: req.headers }))
      })
    })
    const port = await freePort()
    server.on('error', reject)
    server.listen(port, '127.0.0.1', () => resolve({ server, port, requests }))
  })
}

function close(server: http.Server): Promise<void> {
  return new Promise((resolve) => server.close(() => resolve()))
}

function request(port: number, opts: { method?: string; path?: string; headers?: Record<string, string>; body?: string } = {}): Promise<{ status: number; headers: http.IncomingHttpHeaders; body: string }> {
  return new Promise((resolve, reject) => {
    const req = http.request(
      {
        method: opts.method || 'GET',
        host: '127.0.0.1',
        port,
        path: opts.path || '/',
        headers: opts.headers || {},
      },
      (res) => {
        let body = ''
        res.on('data', (c) => { body += c })
        res.on('end', () => resolve({ status: res.statusCode || 0, headers: res.headers, body }))
      },
    )
    req.on('error', reject)
    if (opts.body) req.write(opts.body)
    req.end()
  })
}

describe('theme-proxy: parseDevPort', () => {
  test('解析 localhost 形式', () => {
    expect(parseDevPort('  ➜  Local:    http://localhost:3000/')).toEqual({ host: 'localhost', port: 3000 })
  })

  test('解析 0.0.0.0 并归一为回环', () => {
    expect(parseDevPort('  ➜  Local:    http://0.0.0.0:53721/')).toEqual({ host: '127.0.0.1', port: 53721 })
  })

  test('解析 IPv6 形式并归一为回环', () => {
    expect(parseDevPort('  ➜  Local:    http://[::]:3000/')).toEqual({ host: '127.0.0.1', port: 3000 })
  })

  test('非 Local 行返回 null', () => {
    expect(parseDevPort('  ➜  Network:  http://192.168.1.5:3000/')).toBeNull()
    expect(parseDevPort('some random log')).toBeNull()
    expect(parseDevPort(null as unknown as string)).toBeNull()
  })
})

describe('theme-proxy: dev startup display helpers', () => {
  test('把通配监听地址显示成可访问的本机 URL', () => {
    expect(formatPublicDevUrl('0.0.0.0', 3000)).toBe('http://127.0.0.1:3000/')
    expect(formatPublicDevUrl('::', 3000)).toBe('http://127.0.0.1:3000/')
    expect(formatPublicDevUrl('localhost', 3100)).toBe('http://localhost:3100/')
    expect(formatPublicDevUrl('::1', 3100)).toBe('http://[::1]:3100/')
  })

  test('识别 Nuxt 子进程的内部地址行，避免把临时端口显示成访问端口', () => {
    expect(isNuxtDevAddressLine('  ➜  Local:    http://0.0.0.0:53721/')).toBe(true)
    expect(isNuxtDevAddressLine('  ➜  Network:  http://192.168.1.5:53721/')).toBe(true)
    expect(isNuxtDevAddressLine('  ➜  Network:  use --host to expose')).toBe(false)
    expect(isNuxtDevAddressLine('[sforum-dev-runtime] switched nuxt dev')).toBe(false)
  })
})

describe('theme-proxy: createThemeProxy 透传与 502', () => {
  let upstream: { server: http.Server; port: number; requests: http.IncomingMessage[] }
  let proxyPort: number
  let proxyServer: http.Server

  beforeAll(async () => {
    upstream = await startUpstream('upstream-a')
    const proxy = createThemeProxy({ externalPort: await freePort(), host: '127.0.0.1' })
    await proxy.listen()
    proxyServer = proxy.server
    proxyPort = (proxyServer.address() as net.AddressInfo).port
    proxy.setTarget(makeTarget({ host: '127.0.0.1', port: upstream.port }))
  })

  afterAll(async () => {
    await close(upstream.server)
    await close(proxyServer)
  })

  test('GET 透传方法/路径/状态码/响应头', async () => {
    const res = await request(proxyPort, { method: 'GET', path: '/forum/1' })
    expect(res.status).toBe(200)
    expect(res.headers['x-upstream']).toBe('upstream-a')
    const json = JSON.parse(res.body)
    expect(json.method).toBe('GET')
    expect(json.path).toBe('/forum/1')
  })

  test('POST 透传请求体', async () => {
    const res = await request(proxyPort, {
      method: 'POST',
      path: '/topics',
      headers: { 'content-type': 'text/plain' },
      body: 'hello-body',
    })
    expect(res.status).toBe(200)
    const json = JSON.parse(res.body)
    expect(json.method).toBe('POST')
    expect(json.body).toBe('hello-body')
  })

  test('透传 X-Forwarded-For 链', async () => {
    await request(proxyPort, { headers: { 'x-forwarded-for': '10.0.0.1' } })
    const seen = upstream.requests[upstream.requests.length - 1]
    expect(seen.headers['x-forwarded-for']).toContain('10.0.0.1')
    expect(seen.headers['x-forwarded-for']).toMatch(/127\.0\.0\.1$/)
  })

  test('吞掉已知浏览器扩展注入样式的 sourcemap 探测', async () => {
    const before = upstream.requests.length

    const contentMap = await request(proxyPort, { path: '/content.css.map' })
    const sidebarMap = await request(proxyPort, { path: '/sidebar.css.map' })

    expect(contentMap.status).toBe(204)
    expect(sidebarMap.status).toBe(204)
    expect(upstream.requests).toHaveLength(before)
  })
})

describe('theme-proxy: 无 upstream 时返回 502', () => {
  let proxyServer: http.Server
  let proxyPort: number

  beforeAll(async () => {
    const proxy = createThemeProxy({ externalPort: await freePort(), host: '127.0.0.1' })
    await proxy.listen()
    proxyServer = proxy.server
    proxyPort = (proxyServer.address() as net.AddressInfo).port
    // 故意不 setTarget
  })

  afterAll(async () => close(proxyServer))

  test('未就绪时返回 502', async () => {
    const res = await request(proxyPort)
    expect(res.status).toBe(502)
  })
})

describe('theme-proxy: replaceTarget 蓝绿切换', () => {
  let upstreamOld: { server: http.Server; port: number; requests: http.IncomingMessage[] }
  let upstreamNew: { server: http.Server; port: number; requests: http.IncomingMessage[] }
  let proxy: ReturnType<typeof createThemeProxy>
  let proxyPort: number
  let stopped: string[]

  beforeAll(async () => {
    upstreamOld = await startUpstream('old')
    upstreamNew = await startUpstream('new')
    proxy = createThemeProxy({ externalPort: await freePort(), host: '127.0.0.1' })
    await proxy.listen()
    proxyPort = (proxy.server.address() as net.AddressInfo).port
    // 初始指向 old
    proxy.setTarget(makeTarget({ host: '127.0.0.1', port: upstreamOld.port }))
    stopped = []
  })

  afterAll(async () => {
    await close(upstreamOld.server)
    await close(upstreamNew.server)
    await close(proxy.server)
  })

  test('切换前流量走旧上游', async () => {
    const res = await request(proxyPort)
    expect(JSON.parse(res.body).marker).toBe('old')
  })

  test('切换成功后流量走新上游，旧子进程被停止', async () => {
    const fakeNewChild = { pid: 0 } as unknown as import('node:child_process').ChildProcess
    const fakeOldChild = { pid: 0 } as unknown as import('node:child_process').ChildProcess
    const result = await replaceTarget({
      proxy,
      oldChild: fakeOldChild,
      spawnCandidate: () => fakeNewChild,
      healthCheck: async (candidate) => {
        candidate._target = makeTarget({ host: '127.0.0.1', port: upstreamNew.port })
      },
      stopChild: (c) => { stopped.push(c === fakeOldChild ? 'old' : c === fakeNewChild ? 'new' : 'unknown') },
      healthTimeoutMs: 5000,
    })
    expect(result.ok).toBe(true)
    expect(result.child).toBe(fakeNewChild)
    // 旧子进程被停止
    expect(stopped).toContain('old')
    // 新请求走新上游
    const res = await request(proxyPort)
    expect(JSON.parse(res.body).marker).toBe('new')
  })

  test('候选健康检查失败时保留旧上游，候选被停止', async () => {
    // 先把 upstreamNew 接回去作为"当前"
    proxy.setTarget(makeTarget({ host: '127.0.0.1', port: upstreamNew.port }))
    const fakeNewChild = { pid: 0 } as unknown as import('node:child_process').ChildProcess
    const fakeOldChild = { pid: 0 } as unknown as import('node:child_process').ChildProcess
    const result = await replaceTarget({
      proxy,
      oldChild: fakeOldChild,
      spawnCandidate: () => fakeNewChild,
      healthCheck: async () => { throw new Error('unhealthy') },
      stopChild: (c) => { stopped.push(c === fakeOldChild ? 'old-kept' : 'new-killed') },
      healthTimeoutMs: 5000,
    })
    expect(result.ok).toBe(false)
    // 旧 child 保持不变
    expect(result.child).toBe(fakeOldChild)
    // 候选被停止
    expect(stopped).toContain('new-killed')
    // upstream 仍指向新上游（未被破坏）
    const res = await request(proxyPort)
    expect(JSON.parse(res.body).marker).toBe('new')
  })
})

describe('theme-proxy: healthCheckTcp 与 healthCheckUnix', () => {
  test('healthCheckTcp 对健康上游 resolve', async () => {
    const upstream = await startUpstream('tcp-ok')
    const fakeChild = { _target: null } as unknown as import('node:child_process').ChildProcess
    await healthCheckTcp(fakeChild, '127.0.0.1', upstream.port, { timeoutMs: 3000 })
    expect((fakeChild as { _target: { host: string; port: number } })._target.port).toBe(upstream.port)
    await close(upstream.server)
  })

  test('healthCheckTcp 对无响应端口超时 reject', async () => {
    const fakeChild = {} as import('node:child_process').ChildProcess
    // 用一个几乎肯定没服务的端口
    await expect(healthCheckTcp(fakeChild, '127.0.0.1', 19999, { timeoutMs: 800, intervalMs: 100 })).rejects.toThrow()
  })

  test('healthCheckUnix 对 unix socket 上游 resolve', async () => {
    const sockPath = path.join(os.tmpdir(), `sforum-test-${Date.now()}.sock`)
    const server = http.createServer((_req, res) => {
      res.writeHead(200)
      res.end('ok')
    })
    await new Promise<void>((resolve, reject) => {
      server.on('error', reject)
      server.listen(sockPath, () => resolve())
    })
    const fakeChild = { _target: null } as unknown as import('node:child_process').ChildProcess
    await healthCheckUnix(fakeChild, sockPath, { timeoutMs: 3000 })
    expect((fakeChild as { _target: { socketPath: string } })._target.socketPath).toBe(sockPath)
    await close(server)
    try { fs.unlinkSync(sockPath) } catch { /* already gone */ }
  })
})

// 起一个支持 WebSocket upgrade 的伪造上游：
// upgrade 事件签名 (req, socket, head)。握手回 101，随后把 req 头记录下来供断言，
// 并在隧道里 echo 一段标记字节，让客户端能验证双向 socket 真的被透传。
function startWsUpstream(
  marker: string,
  listenOpts: { port?: number; socketPath?: string },
): Promise<{ server: http.Server; port?: number; lastUpgrade: () => http.IncomingMessage | null }> {
  return new Promise(async (resolve, reject) => {
    let lastUpgrade: http.IncomingMessage | null = null
    const server = http.createServer((_req, res) => {
      // 非 upgrade 的普通请求兜底返回 200，供 healthCheck 用。
      res.writeHead(200)
      res.end('ok')
    })
    server.on('upgrade', (req, socket) => {
      lastUpgrade = req
      socket.write(
        'HTTP/1.1 101 Switching Protocols\r\n' +
          'Upgrade: websocket\r\n' +
          'Connection: Upgrade\r\n' +
          `\r\n${marker}`,
      )
    })
    server.on('error', reject)
    if (listenOpts.socketPath) {
      try { fs.unlinkSync(listenOpts.socketPath) } catch { /* 不存在无所谓 */ }
      server.listen(listenOpts.socketPath, () => resolve({ server, lastUpgrade: () => lastUpgrade }))
    } else {
      const port = listenOpts.port ?? (await freePort())
      server.listen(port, '127.0.0.1', () => resolve({ server, port, lastUpgrade: () => lastUpgrade }))
    }
  })
}

// 用原始 socket 手搓一次 WebSocket 握手，返回 upstream 隧道里收到的首段数据与底层 socket。
// 不引第三方 ws 库，与现有测试零依赖风格一致。
function rawUpgrade(
  port: number,
  opts: { path?: string; headers?: Record<string, string> } = {},
): Promise<{ firstByte: string; socket: net.Socket }> {
  return new Promise((resolve, reject) => {
    const socket = net.connect(port, '127.0.0.1')
    let settled = false
    socket.on('error', (err) => {
      if (!settled) reject(err)
    })
    socket.on('close', () => {
      if (!settled) reject(new Error('socket closed before handshake data arrived'))
    })
    socket.on('data', (chunk) => {
      // 跳过 101 响应头，取隧道体首段作为 marker。
      settled = true
      const text = chunk.toString()
      const sep = text.indexOf('\r\n\r\n')
      const firstByte = sep >= 0 ? text.slice(sep + 4) : text
      resolve({ firstByte, socket })
    })
    let raw = `GET ${opts.path || '/'} HTTP/1.1\r\n` +
      'Host: localhost\r\n' +
      'Upgrade: websocket\r\n' +
      'Connection: Upgrade\r\n'
    for (const [key, value] of Object.entries(opts.headers || {})) {
      raw += `${key}: ${value}\r\n`
    }
    raw += '\r\n'
    socket.write(raw)
  })
}

// supervisor 必须用 node 跑：bun 的 node:http 兼容层在 'upgrade' 事件里
// socket.write 会静默丢数据（bun#28157），客户端收不到 101。
// 用 bun 运行测试时跳过这些用例（运行时缺陷，非实现问题），仅由 node 覆盖。
const isBun = typeof (globalThis as { Bun?: unknown }).Bun !== 'undefined'
const wsDescribe = isBun ? describe.skip : describe

wsDescribe('theme-proxy: WebSocket upgrade 转发（需 node 运行时）', () => {
  test('TCP upstream 的 WS 握手被透传，且补全 X-Forwarded-For', async () => {
    const upstream = await startWsUpstream('ws-tcp', {})
    const proxy = createThemeProxy({ externalPort: await freePort(), host: '127.0.0.1' })
    await proxy.listen()
    const proxyPort = (proxy.server.address() as net.AddressInfo).port
    proxy.setTarget(makeTarget({ host: '127.0.0.1', port: upstream.port! }))

    const { firstByte, socket } = await rawUpgrade(proxyPort)
    expect(firstByte).toBe('ws-tcp')
    // upstream 收到了补全的 X-Forwarded-For（含代理本机地址）。
    expect(upstream.lastUpgrade()?.headers['x-forwarded-for']).toMatch(/127\.0\.0\.1$/)
    // upgrade 头被原样透传。
    expect(upstream.lastUpgrade()?.headers['upgrade']).toBe('websocket')

    socket.destroy()
    await close(upstream.server)
    await close(proxy.server)
  })

  test('activeTarget 未就绪时 WS 连接被销毁', async () => {
    const proxy = createThemeProxy({ externalPort: await freePort(), host: '127.0.0.1' })
    await proxy.listen()
    const proxyPort = (proxy.server.address() as net.AddressInfo).port
    // 故意不 setTarget，模拟冷启动窗口。
    await expect(rawUpgrade(proxyPort)).rejects.toThrow()
    await close(proxy.server)
  })

  test('unix socket upstream 的 WS 握手也被透传（生产路径）', async () => {
    const sockPath = path.join(os.tmpdir(), `sforum-ws-test-${Date.now()}.sock`)
    const upstream = await startWsUpstream('ws-unix', { socketPath: sockPath })
    const proxy = createThemeProxy({ externalPort: await freePort(), host: '127.0.0.1' })
    await proxy.listen()
    const proxyPort = (proxy.server.address() as net.AddressInfo).port
    proxy.setTarget(makeTarget({ socketPath: sockPath }))

    const { firstByte, socket } = await rawUpgrade(proxyPort)
    expect(firstByte).toBe('ws-unix')

    socket.destroy()
    await close(upstream.server)
    await close(proxy.server)
    try { fs.unlinkSync(sockPath) } catch { /* already gone */ }
  })
})
