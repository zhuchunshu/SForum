// 主题运行时共享代理核心：supervisor 对外监听固定端口，内部把流量代理到
// 当前活跃的子进程（Nitro 产物或 nuxt dev）。生产 runtime.mjs 使用下方
// replaceTarget 做蓝绿切换；本地 dev supervisor 串行重启，只复用代理与健康检查。
//
// 两个 supervisor（生产 runtime.mjs / 开发 dev-theme-runtime.mjs）共用本模块：
// - 生产：子进程监听 unix socket（NITRO_UNIX_SOCKET），healthCheckUnix 探测。
// - 开发：子进程监听 TCP 临时端口（PORT=0），healthCheckTcp 探测。
import http from 'node:http'
import net from 'node:net'

const DEFAULT_HEALTH_PATH = '/'
const browserExtensionSourceMapPaths = new Set([
  '/content.css.map',
  '/sidebar.css.map',
])

// 构造一个 upstream 目标描述。socketPath 非空表示走 unix socket，
// 否则按 TCP host:port 处理。两种地址类型在代理转发时分别处理。
export function makeTarget({ host, port, socketPath } = {}) {
  if (socketPath) {
    return { socketPath }
  }
  return { host: host || '127.0.0.1', port: Number(port) }
}

// 创建反向代理。externalPort 是 supervisor 对外监听的端口（默认 3000）。
// 返回的对象持有 http.Server 与当前活跃 upstream 的引用。
export function createThemeProxy({ externalPort = 3000, host = '0.0.0.0' } = {}) {
  // 当前代理目标。Node 单线程下单一变量赋值天然原子，无需锁。
  // 初始为 null：在第一个子进程 ready 前所有请求返回 502。
  let activeTarget = null

  const server = http.createServer((req, res) => {
    if (isKnownBrowserExtensionSourceMapRequest(req)) {
      // 部分 Chrome 扩展会把注入样式的 sourceMappingURL 解析到站点根路径。
      // 这些请求不是 SForum 静态资源，直接吞掉可避免 Nuxt Router 反复输出无匹配告警。
      res.writeHead(204, { 'cache-control': 'no-store' })
      res.end()
      return
    }

    const target = activeTarget
    if (!target) {
      res.writeHead(502, { 'content-type': 'text/plain; charset=utf-8' })
      res.end('SForum web is starting up. Please retry shortly.')
      return
    }
    forwardRequest(target, req, res)
  })

  // 转发 HTTP 升级（WebSocket）请求。Nuxt/Vite 的 HMR 依赖与页面同源的 WS，
  // 而 supervisor 对外只有一个端口，浏览器会向代理端口发起 upgrade 握手；
  // 不转发 upgrade，HMR 就完全失效（改文件不热更新）。
  // 这里不能用 http.request：它只处理 request/response 语义，101 之后的字节流
  // 必须以原始 socket 隧道透传，所以用 net.connect 打到 upstream 再双向 pipe。
  //
  // 注意：supervisor 必须用 node 运行。bun 的 node:http 兼容层在 'upgrade' 事件里
  // socket.write 会静默丢数据（bun#28157 / bun#9882），导致客户端收不到 101。
  // 见 apps/web/package.json 的 dev 脚本（node --env-file ...）。
  server.on('upgrade', (req, socket, head) => {
    const target = activeTarget
    if (!target) {
      // upstream 未就绪：直接销毁，浏览器 HMR client 会按自身退避策略重试。
      socket.destroy()
      return
    }
    forwardUpgrade(target, req, socket, head)
  })

  // 把 server.listen 包装成 Promise，方便 supervisor 启动时 await。
  function listen() {
    return new Promise((resolve, reject) => {
      const onError = (err) => reject(err)
      server.once('error', onError)
      server.listen(externalPort, host, () => {
        server.removeListener('error', onError)
        resolve()
      })
    })
  }

  // 原子切换代理目标。下一次进来的请求就走新 target。
  // 旧 target 的在途连接不受影响（http 仅在请求开始时读一次 target）。
  function setTarget(target) {
    activeTarget = target || null
  }

  function getTarget() {
    return activeTarget
  }

  // 优雅关闭代理 server 本身（不关子进程，子进程由 supervisor 管理）。
  function close() {
    return new Promise((resolve) => {
      server.close(() => resolve())
    })
  }

  return {
    server,
    listen,
    setTarget,
    getTarget,
    close,
  }
}

// 把一个入站请求转发到 target upstream，流式 pipe 请求体与响应体，
// 支持 SSE / 长连接 / 大文件上传。
function forwardRequest(target, req, res) {
  // unix socket 用 Node 原生 socketPath 选项（createConnection 选项在此场景不可靠）；
  // TCP 用 host/port。两种地址类型在请求选项上分叉，其余转发逻辑一致。
  const targetOpts = target.socketPath
    ? { socketPath: target.socketPath }
    : { host: target.host, port: target.port }

  const proxyReq = http.request(
    {
      ...targetOpts,
      method: req.method,
      path: req.url,
      headers: buildForwardHeaders(req),
    },
    (proxyRes) => {
      res.writeHead(proxyRes.statusCode, proxyRes.headers)
      proxyRes.pipe(res)
    },
  )

  proxyReq.on('error', (err) => {
    if (!res.headersSent) {
      res.writeHead(502, { 'content-type': 'text/plain; charset=utf-8' })
      res.end(`Upstream unavailable: ${err.message}`)
    } else {
      // 响应已开始，只能强制结束连接。
      res.destroy()
    }
  })

  // 客户端提前断开（响应还没结束）时，主动中止上游请求，避免句柄泄漏。
  // 注意必须用 res 的 close 事件而非 req 的，且只在响应未完成时中止：
  // 否则 keep-alive 的 GET 请求体读完后 req 立刻 close，会把刚建好的上游连接也毁了。
  res.on('close', () => {
    if (!res.writableEnded && !proxyReq.destroyed) {
      proxyReq.destroy()
    }
  })

  req.pipe(proxyReq)
}

// 转发 HTTP 升级（WebSocket）请求到 upstream。与 forwardRequest 对称，
// 但走原始 socket 隧道：连上 upstream 后把客户端发来的 upgrade 请求行 + headers
// 原样写入 upstream，再把两端 socket 双向 pipe，101 之后的 WS 帧透传。
// 同时覆盖 TCP（dev: nuxt dev）与 unix socket（生产: Nitro）两种 upstream，
// 让开发期 Vite HMR 和生产期 app 级 WS 都能正常工作。
//
// 依赖 node 运行时：bun 在 'upgrade' 事件里 socket.write 会静默丢数据，
// 在 bun 下本函数无效。supervisor 用 node 跑即避开此问题。
function forwardUpgrade(target, req, socket, head) {
  // 与 forwardRequest 同样的地址分叉：socketPath 走 unix socket，否则 TCP。
  const connectOpts = target.socketPath
    ? { path: target.socketPath }
    : { host: target.host, port: target.port }

  const upstream = net.connect(connectOpts)

  const cleanup = () => {
    socket.destroy()
    upstream.destroy()
  }

  upstream.on('error', () => {
    // upstream 连接失败：销毁客户端 socket，HMR client 会自动重试。
    cleanup()
  })
  socket.on('error', cleanup)

  upstream.on('connect', () => {
    // 连接建立后，先回写 head 里早到的字节（upgrade 事件可能携带已读到的数据）。
    if (head && head.length) {
      upstream.write(head)
    }
    // 重组原始请求行 + headers，原样发给 upstream。补 X-Forwarded-* 链便于
    // upstream 侧日志/鉴权；其余头（含 host、connection、upgrade 等）透传，
    // 保证 WS 子协议与握手语义不被破坏。
    const forwardHeaders = buildForwardHeaders(req)
    let raw = `${req.method} ${req.url} HTTP/1.1\r\n`
    for (const [key, value] of Object.entries(forwardHeaders)) {
      raw += `${key}: ${value}\r\n`
    }
    raw += '\r\n'
    upstream.write(raw)
    // 双向 pipe：此后任一端的数据（含 101 响应及 WS 帧）透传到对端。
    // 任一端关闭/出错时销毁另一端，避免句柄泄漏。
    socket.pipe(upstream)
    upstream.pipe(socket)
  })
  // 客户端或 upstream 任一端正常关闭时也兜底销毁，与上面 error 路径统一。
  socket.on('close', () => upstream.destroy())
  upstream.on('close', () => socket.destroy())
}

// 组装转发 header：透传原始 header，并维护 X-Forwarded-* 链。
function buildForwardHeaders(req) {
  const headers = { ...req.headers }
  const incoming = req.socket.remoteAddress
  if (incoming) {
    const prior = headers['x-forwarded-for']
    headers['x-forwarded-for'] = prior ? `${prior}, ${incoming}` : incoming
  }
  // 信任协议：有 x-forwarded-proto 就沿用，否则按入站 socket 推断。
  if (!headers['x-forwarded-proto']) {
    headers['x-forwarded-proto'] = req.socket.encrypted ? 'https' : 'http'
  }
  // 走 unix socket 时 host 头可能被 Nitro 用来拼 baseURL，保留原始 host。
  return headers
}

// 蓝绿切换核心：起候选子进程 → 健康检查 → 切 upstream → drain 旧子进程。
//
// 参数：
// - proxy: createThemeProxy 返回的对象
// - spawnCandidate: () => ChildProcess，负责 spawn 新子进程
// - healthCheck: () => Promise<void>，resolve 表示候选就绪；reject/超时表示不可用
// - stopChild: (child) => void，停止子进程的回调（supervisor 各自实现进程组信号逻辑）
// - onSpawnError / onHealthFailed: 可选日志回调
// - healthTimeoutMs: 健康检查总超时（默认 60s，覆盖 nuxt dev 冷启动）
//
// 返回 { child, ok }：ok=true 表示切换成功（返回新 child）；
// ok=false 表示候选不可用，旧 child 保持不变（caller 传进来用于回滚）。
export async function replaceTarget({
  proxy,
  oldChild,
  spawnCandidate,
  healthCheck,
  stopChild,
  healthTimeoutMs = 60_000,
  onSpawnError = () => {},
  onHealthFailed = () => {},
}) {
  let candidate
  try {
    candidate = spawnCandidate()
  } catch (err) {
    onSpawnError(err)
    return { child: oldChild, ok: false }
  }

  try {
    // healthCheck 接收 candidate 并负责把候选实际监听的地址写到 candidate._target，
    // 切换成功后 proxy.setTarget 会读取这个地址。
    await withTimeout(healthCheck(candidate), healthTimeoutMs)
  } catch (err) {
    // 候选不可用：杀掉候选，保留旧 child 继续服务。
    onHealthFailed(err)
    stopChild(candidate)
    return { child: oldChild, ok: false }
  }

  // 候选 ready：原子切 upstream，然后 drain 旧 child。
  // target 地址由 healthCheck 负责确定并写回 candidate（见 healthCheckTcp/Unix）。
  proxy.setTarget(candidate._target)
  if (oldChild) {
    stopChild(oldChild)
  }
  return { child: candidate, ok: true }
}

// 给 Promise 套一个总超时，超时则 reject。用于健康检查不能无限等。
function withTimeout(promise, ms) {
  return new Promise((resolve, reject) => {
    const timer = setTimeout(() => reject(new Error(`health check timed out after ${ms}ms`)), ms)
    promise.then(
      (v) => {
        clearTimeout(timer)
        resolve(v)
      },
      (e) => {
        clearTimeout(timer)
        reject(e)
      },
    )
  })
}

// TCP 健康检查：轮询 GET path 直到返回 <500，期间记录候选监听的 host:port。
// 解析到的地址会挂到 candidate._target，供 replaceTarget 切换 upstream 用。
export async function healthCheckTcp(candidate, host, port, {
  timeoutMs = 60_000,
  path = DEFAULT_HEALTH_PATH,
  intervalMs = 500,
} = {}) {
  const target = makeTarget({ host, port })
  candidate._target = target
  const deadline = Date.now() + timeoutMs
  while (Date.now() < deadline) {
    try {
      const ok = await probeTcp(target, path)
      if (ok) return
    } catch {
      // 上游尚未起来，继续轮询。
    }
    await sleep(intervalMs)
  }
  throw new Error(`tcp health check failed for ${host}:${port}`)
}

// Unix socket 健康检查：轮询 GET path 直到返回 <500。
export async function healthCheckUnix(candidate, socketPath, {
  timeoutMs = 60_000,
  path = DEFAULT_HEALTH_PATH,
  intervalMs = 500,
  requiredSuccesses = 1,
} = {}) {
  const target = makeTarget({ socketPath })
  candidate._target = target
  const deadline = Date.now() + timeoutMs
  let successes = 0
  while (Date.now() < deadline) {
    try {
      const ok = await probeUnix(target, path)
      if (ok) {
        successes += 1
        if (successes >= requiredSuccesses) return
      } else {
        successes = 0
      }
    } catch {
      // socket 尚未创建或 Nitro 未起来，继续轮询。
      successes = 0
    }
    await sleep(intervalMs)
  }
  throw new Error(`unix socket health check failed for ${socketPath}`)
}

// 探测 TCP upstream：一次 GET，状态码 <500 视为健康。
function probeTcp(target, probePath) {
  return new Promise((resolve, reject) => {
    const req = http.request(
      {
        method: 'GET',
        host: target.host,
        port: target.port,
        path: probePath,
        timeout: 2000,
      },
      (res) => {
        res.resume()
        resolve(res.statusCode < 500)
      },
    )
    req.on('timeout', () => {
      req.destroy()
      reject(new Error('probe timeout'))
    })
    req.on('error', reject)
    req.end()
  })
}

// 探测 unix socket upstream：一次 GET，状态码 <500 视为健康。
// 用 Node 原生 socketPath 选项发起请求（createConnection 在 unix socket 场景不可靠）。
function probeUnix(target, probePath) {
  return new Promise((resolve, reject) => {
    const req = http.request(
      {
        method: 'GET',
        path: probePath,
        socketPath: target.socketPath,
        timeout: 2000,
      },
      (res) => {
        res.resume()
        resolve(res.statusCode < 500)
      },
    )
    req.on('timeout', () => {
      req.destroy()
      reject(new Error('probe timeout'))
    })
    req.on('error', reject)
    req.end()
  })
}

function sleep(ms) {
  return new Promise((resolve) => setTimeout(resolve, ms))
}

// 从 listhen/nuxt dev 的 stdout 行里解析监听端口。
// 兼容多种格式：
//   ➜  Local:    http://localhost:3000/
//   ➜  Local:    http://0.0.0.0:53721/
//   ➜  Local:    http://[::]:3000/
// 返回 { host, port } 或 null。host 中的 0.0.0.0/:: 归一为回环地址，
// 因为 supervisor 在本机代理，用回环最稳。
export function parseDevPort(line) {
  if (typeof line !== 'string') return null
  const cleanLine = stripAnsi(line)
  // 匹配 "Local:" 行的 URL，host 部分兼容 IPv6 方括号写法。
  const m = cleanLine.match(/Local:\s+https?:\/\/(\[[^\]]+\]|[^:/]+):(\d+)/)
  if (!m) return null
  let host = m[1]
  const port = Number(m[2])
  if (!port) return null
  if (host.startsWith('[')) {
    host = host.slice(1, -1)
  }
  // 归一通配地址为回环，避免代理连到 0.0.0.0（部分平台不可连）。
  if (host === '0.0.0.0' || host === '::') {
    host = '127.0.0.1'
  }
  return { host, port }
}

function isKnownBrowserExtensionSourceMapRequest(req) {
  if (req.method !== 'GET' && req.method !== 'HEAD') {
    return false
  }
  const pathname = String(req.url || '').split('?')[0]
  return browserExtensionSourceMapPaths.has(pathname)
}

// 开发 supervisor 对外监听的是代理端口，Nuxt 子进程的 Local/Network 行只表示
// 内部临时端口。隐藏这些行，避免终端把内部端口误导成用户访问入口。
export function isNuxtDevAddressLine(line) {
  if (typeof line !== 'string') return false
  const cleanLine = stripAnsi(line)
  return /^\s*➜\s+(Local|Network):\s+https?:\/\/(\[[^\]]+\]|[^:/]+):\d+/.test(cleanLine)
}

export function formatPublicDevUrl(host, port) {
  const displayPort = Number(port) || 3000
  let displayHost = String(host || '127.0.0.1').trim()
  if (displayHost.startsWith('[') && displayHost.endsWith(']')) {
    displayHost = displayHost.slice(1, -1)
  }
  if (!displayHost || displayHost === '0.0.0.0' || displayHost === '::') {
    displayHost = '127.0.0.1'
  }
  if (displayHost.includes(':')) {
    displayHost = `[${displayHost}]`
  }
  return `http://${displayHost}:${displayPort}/`
}

function stripAnsi(value) {
  return value.replace(/\x1B\[[0-?]*[ -/]*[@-~]/g, '')
}
