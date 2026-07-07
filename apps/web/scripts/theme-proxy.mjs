// 主题切换蓝绿代理核心：supervisor 对外监听固定端口，内部把流量代理到
// 当前活跃的子进程（Nitro 产物或 nuxt dev）。切换主题时先起新子进程，
// 等它健康检查通过再切 upstream，最后 drain 旧子进程，实现零停机切换。
//
// 两个 supervisor（生产 runtime.mjs / 开发 dev-theme-runtime.mjs）共用本模块，
// 只在「子进程监听方式」和「健康检查方式」上分叉：
// - 生产：子进程监听 unix socket（NITRO_UNIX_SOCKET），healthCheckUnix 探测。
// - 开发：子进程监听 TCP 临时端口（PORT=0），healthCheckTcp 探测。
import http from 'node:http'

const DEFAULT_HEALTH_PATH = '/'

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
    const target = activeTarget
    if (!target) {
      res.writeHead(502, { 'content-type': 'text/plain; charset=utf-8' })
      res.end('SForum web is starting up. Please retry shortly.')
      return
    }
    forwardRequest(target, req, res)
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
} = {}) {
  const target = makeTarget({ socketPath })
  candidate._target = target
  const deadline = Date.now() + timeoutMs
  while (Date.now() < deadline) {
    try {
      const ok = await probeUnix(target, path)
      if (ok) return
    } catch {
      // socket 尚未创建或 Nitro 未起来，继续轮询。
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
