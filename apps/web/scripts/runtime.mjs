// 生产 web 进程的主题感知 supervisor。
// 监听 theme-releases/current.json，切换到当前主题的 Nitro server 产物。
// 保留旧 child 直到新候选验证可用，避免切到坏主题导致整个站点挂掉。
import { spawn } from 'node:child_process'
import fs from 'node:fs'
import path from 'node:path'

const releaseRoot = process.env.SFORUM_THEME_RELEASE_ROOT || '/var/lib/sforum/theme-releases'
const fallbackOutput = process.env.SFORUM_FALLBACK_OUTPUT || path.resolve(process.cwd(), '.output')
const currentFile = path.join(releaseRoot, 'current.json')

let child = null
let activeServer = ''
let restartTimer = null

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

function stopChild() {
  if (!child) {
    return
  }
  const pid = child.pid
  child = null
  if (!pid) {
    return
  }
  try {
    // 用进程组信号杀掉 Nitro server 及其可能派生的子进程，
    // 避免仅杀父进程后端口仍被孤儿子进程占用。
    process.kill(-pid, 'SIGTERM')
  } catch (error) {
    if (error.code !== 'ESRCH') {
      console.error('[sforum-web-runtime] failed to signal child group:', error.message)
    }
  }
}

function startCurrent() {
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
  stopChild()
  activeServer = selection.server
  console.log(`[sforum-web-runtime] starting ${selection.kind} server: ${selection.server}`)
  child = spawn(process.execPath, [selection.server], {
    stdio: 'inherit',
    env: {
      ...process.env,
      HOST: process.env.HOST || '0.0.0.0',
      PORT: process.env.PORT || '3000'
    },
    // detached 让子进程成为独立进程组组长，便于用 -pid 杀掉整个进程组。
    detached: true
  })
  child.on('exit', (code, signal) => {
    console.error(`[sforum-web-runtime] child exited code=${code ?? ''} signal=${signal ?? ''}`)
    child = null
    activeServer = ''
    setTimeout(startCurrent, 1000)
  })
}

function scheduleRestart() {
  clearTimeout(restartTimer)
  restartTimer = setTimeout(startCurrent, 250)
}

fs.mkdirSync(releaseRoot, { recursive: true })
fs.watch(releaseRoot, scheduleRestart)
process.on('SIGTERM', () => {
  stopChild()
  process.exit(0)
})
process.on('SIGINT', () => {
  stopChild()
  process.exit(0)
})

startCurrent()
