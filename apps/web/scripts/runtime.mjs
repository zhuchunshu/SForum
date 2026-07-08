// 生产 web 进程的主题感知 supervisor（蓝绿零停机版）。
//
// 监听 theme-releases/current.json，切换到当前主题的 Nitro server 产物。
// 与旧版的关键区别：supervisor 自身监听对外端口（PORT，默认 3000）作为反向代理，
// 子进程（Nitro server）监听独立的 unix socket。切换主题时先起新子进程，
// 健康检查通过后再切代理 upstream 并 drain 旧进程——全程对用户无感。
//
// 之所以用 unix socket 而非 TCP：Nitro node-server 预设不支持 PORT=0
// （destr("0")||3000 会落到 3000），但原生支持 NITRO_UNIX_SOCKET，
// 且 socket 路径每次唯一，天然零端口冲突。
import { spawn } from 'node:child_process'
import fs from 'node:fs'
import os from 'node:os'
import path from 'node:path'

import {
  createThemeProxy,
  healthCheckUnix,
  replaceTarget,
} from './theme-proxy.mjs'

const releaseRoot = process.env.SFORUM_THEME_RELEASE_ROOT || '/var/lib/sforum/theme-releases'
const fallbackOutput = process.env.SFORUM_FALLBACK_OUTPUT || path.resolve(process.cwd(), '.output')
const currentFile = path.join(releaseRoot, 'current.json')
const externalPort = Number(process.env.PORT || '3000')
const externalHost = process.env.HOST || '0.0.0.0'
// 健康检查总超时：Nitro 产物冷启动通常几秒，留足余量。
const healthTimeoutMs = Number(process.env.SFORUM_THEME_HEALTH_TIMEOUT || '30000')
// socket 文件根目录，放容器/主机临时目录，避免污染 releaseRoot。
const socketDir = process.env.SFORUM_SOCKET_DIR || os.tmpdir()

let child = null
let activeServer = ''
let restartTimer = null
let switching = false
let pendingSelection = null

const proxy = createThemeProxy({ externalPort, host: externalHost })

function fallbackServer() {
  return path.join(fallbackOutput, 'server/index.mjs')
}

// 把 current.json 里的 server 路径解析成绝对路径。
// 后端 WriteCurrent 已统一写绝对路径，这里对历史/手写的相对路径做兜底：
// 相对路径以 releaseRoot 为基准解析（与 builder 的产物目录约定一致）。
function resolveServer(server) {
  if (!server) {
    return ''
  }
  return path.isAbsolute(server) ? server : path.resolve(releaseRoot, server)
}

// 读取当前主题选择。返回 { kind: 'default' | 'uploaded', server }。
// - 无 current.json 或 mode==='default'：回退到默认 .output。
// - 旧格式（只有 server）：视为 uploaded，兼容历史 current.json。
// - server 为空但 mode==='uploaded'：无法定位产物，视为不可用。
function readSelection() {
  let raw = ''
  try {
    raw = fs.readFileSync(currentFile, 'utf8')
  } catch (error) {
    if (error.code !== 'ENOENT') {
      console.error('[sforum-web-runtime] invalid current release:', error.message)
    }
    return { kind: 'default', server: fallbackServer() }
  }
  let current
  try {
    current = JSON.parse(raw)
  } catch (error) {
    console.error('[sforum-web-runtime] invalid current release:', error.message)
    return { kind: 'default', server: fallbackServer() }
  }
  if (current.mode === 'default') {
    return { kind: 'default', server: fallbackServer() }
  }
  const server = resolveServer(current.server)
  if (!server) {
    return { kind: 'default', server: fallbackServer() }
  }
  return { kind: 'uploaded', server }
}

// 停止子进程：用进程组信号杀掉 Nitro server 及其可能派生的子进程，
// 避免仅杀父进程后 socket/端口仍被孤儿子进程占用。
function stopChild(target) {
  if (!target) {
    return
  }
  const pid = target.pid
  if (!pid) {
    return
  }
  try {
    process.kill(-pid, 'SIGTERM')
  } catch (error) {
    if (error.code !== 'ESRCH') {
      console.error('[sforum-web-runtime] failed to signal child group:', error.message)
    }
  }
}

// 生成一个唯一的 unix socket 路径，并清理可能残留的同名文件。
function nextSocketPath() {
  const sock = path.join(socketDir, `sforum-web-${Date.now()}-${Math.random().toString(36).slice(2)}.sock`)
  try {
    fs.unlinkSync(sock)
  } catch (error) {
    if (error.code !== 'ENOENT') {
      console.error('[sforum-web-runtime] failed to clean stale socket:', error.message)
    }
  }
  return sock
}

// 蓝绿切换到指定 server 产物。候选子进程先起在临时 socket 上，
// 健康检查通过后切代理 upstream，再 drain 旧子进程。
// 候选不可用时保留旧 child 继续服务，绝不中断。
async function switchTo(server, kind) {
  const sockPath = nextSocketPath()
  console.log(`[sforum-web-runtime] starting ${kind} server: ${server} (socket ${sockPath})`)

  const result = await replaceTarget({
    proxy,
    oldChild: child,
    spawnCandidate: () => spawn(process.execPath, [server], {
      stdio: 'inherit',
      env: {
        ...process.env,
        // Nitro node-server 预设识别此变量并绑定到 unix socket。
        NITRO_UNIX_SOCKET: sockPath,
      },
      // detached 让子进程成为独立进程组组长，便于用 -pid 杀整个进程组。
      detached: true,
    }),
    healthCheck: (candidate) => healthCheckUnix(candidate, sockPath, { timeoutMs: healthTimeoutMs }),
    stopChild,
    healthTimeoutMs,
    onSpawnError: (err) => console.error('[sforum-web-runtime] spawn candidate failed:', err.message),
    onHealthFailed: (err) => console.error('[sforum-web-runtime] candidate health check failed:', err.message),
  })

  child = result.child
  if (result.ok) {
    activeServer = server
    console.log(`[sforum-web-runtime] switched to ${kind} server: ${server}`)
  }
  return result.ok
}

// 根据 current.json 决定要切到哪个 server 产物，并触发蓝绿切换。
// 进行中的切换会被记忆，切换串行执行避免并发 spawn。
async function startCurrent() {
  const selection = readSelection()
  // 候选 server 不存在时，若旧 child 还活着就保留它（不中断服务），
  // 只有在没有 child 运行时才回退默认产物，保证站点始终可用。
  if (!fs.existsSync(selection.server)) {
    console.error(`[sforum-web-runtime] selected server does not exist: ${selection.server}`)
    if (child) {
      return
    }
    selection.server = fallbackServer()
    if (!fs.existsSync(selection.server)) {
      console.error(`[sforum-web-runtime] fallback server does not exist: ${selection.server}`)
      return
    }
  }
  if (selection.server === activeServer && child) {
    return
  }
  if (switching) {
    // 已有切换在进行，记下最新意图，切换结束后再处理一次。
    pendingSelection = selection
    return
  }
  switching = true
  try {
    await switchTo(selection.server, selection.kind)
  } finally {
    switching = false
  }
  if (pendingSelection) {
    const next = pendingSelection
    pendingSelection = null
    if (next.server !== activeServer) {
      await startCurrent()
    }
  }
}

function scheduleRestart() {
  clearTimeout(restartTimer)
  restartTimer = setTimeout(() => {
    startCurrent().catch((err) => console.error('[sforum-web-runtime] restart failed:', err.message))
  }, 250)
}

async function main() {
  fs.mkdirSync(releaseRoot, { recursive: true })
  // 先注册信号处理：启动期间收到信号也要能优雅退出。
  // proxy.close() 在未 listen 时调用是安全的（server.close 直接回调）。
  process.on('SIGTERM', async () => {
    stopChild(child)
    child = null
    await proxy.close()
    process.exit(0)
  })
  process.on('SIGINT', async () => {
    stopChild(child)
    child = null
    await proxy.close()
    process.exit(0)
  })
  // 先启动 Nitro 子进程并等它通过健康检查，再对外监听。
  // 否则冷启动窗口期 activeTarget===null，所有请求返回 502，
  // SPA 页面拿不到 entry.js 就白屏。
  await startCurrent()
  if (!proxy.getTarget()) {
    console.error('[sforum-web-runtime] initial server did not become ready; exiting')
    process.exit(1)
  }
  await proxy.listen()
  console.log(`[sforum-web-runtime] proxy listening on ${externalHost}:${externalPort}`)
  fs.watch(releaseRoot, scheduleRestart)
}

main().catch((err) => {
  console.error('[sforum-web-runtime] fatal:', err.message)
  process.exit(1)
})
