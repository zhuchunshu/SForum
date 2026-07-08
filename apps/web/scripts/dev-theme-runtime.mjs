// 本地开发的主题感知 supervisor（蓝绿零停机版）：包裹 `bun run dev:plain`（原始 nuxt dev）。
//
// 工作方式：
// - supervisor 自身监听对外端口（PORT，默认 3000）作为反向代理；nuxt dev 子进程
//   监听独立的临时端口（PORT=0，由 listhen/get-port-please 自动分配）。
// - 启动时读 theme-releases/current.json，决定初始的 SFORUM_THEME_LAYER。
//   mode==='default' 或无文件：不设环境变量，nuxt 用默认主题 layer。
//   layerPath 非空：把它作为 SFORUM_THEME_LAYER 注入，nuxt 把上传主题作为优先 layer。
// - fs.watch 监听 releaseRoot，current.json 变化后 250ms 防抖触发蓝绿切换：
//   起新 nuxt dev 子进程（临时端口）→ 解析它 stdout 的监听端口 → 健康检查 →
//   切代理 upstream → drain 旧子进程。切换期间旧主题继续服务，用户无感。
// - 只管自己 spawn 的子进程，不碰端口 3000 上的其它进程。
//
// 与生产 runtime.mjs 的区别：生产子进程监听 unix socket（NITRO_UNIX_SOCKET），
// 开发 nuxt dev 不支持 socket，改用 PORT=0 临时 TCP 端口，靠解析 stdout 拿到端口。
import { spawn } from 'node:child_process'
import fs from 'node:fs'
import path from 'node:path'

import {
  createThemeProxy,
  formatPublicDevUrl,
  healthCheckTcp,
  isNuxtDevAddressLine,
  parseDevPort,
  replaceTarget,
} from './theme-proxy.mjs'

const releaseRoot = process.env.SFORUM_THEME_RELEASE_ROOT || path.resolve(process.cwd(), '../../storage/theme-releases')
const currentFile = path.join(releaseRoot, 'current.json')
const bunPath = process.env.SFORUM_BUN_PATH || 'bun'
const externalPort = Number(process.env.PORT || process.env.WEB_PORT || '3000')
const externalHost = process.env.HOST || '0.0.0.0'
const publicDevUrl = formatPublicDevUrl(externalHost, externalPort)
// 健康检查总超时：nuxt dev 冷启动可能要几十秒，给足时间。
const healthTimeoutMs = Number(process.env.SFORUM_THEME_HEALTH_TIMEOUT || '120000')

let child = null
let activeLayer = null
let restartTimer = null
let switching = false
let pendingLayer = null
let pendingLayerReason = null

const proxy = createThemeProxy({ externalPort, host: externalHost })

// 读取当前应使用的 layer 路径。返回 null 表示用默认主题。
// layerPath 可能是绝对路径（新版 builder）或相对路径（历史/手写），
// 相对路径按仓库根（apps/web 的上两级）解析。
function readLayerPath() {
  let raw
  try {
    raw = fs.readFileSync(currentFile, 'utf8')
  } catch (error) {
    if (error.code !== 'ENOENT') {
      console.error('[sforum-dev-runtime] invalid current release:', error.message)
    }
    return null
  }
  let current
  try {
    current = JSON.parse(raw)
  } catch (error) {
    console.error('[sforum-dev-runtime] invalid current release:', error.message)
    return null
  }
  if (current.mode === 'default') {
    return null
  }
  const layerPath = typeof current.layerPath === 'string' ? current.layerPath.trim() : ''
  if (!layerPath) {
    return null
  }
  if (path.isAbsolute(layerPath)) {
    return layerPath
  }
  // 仓库根 = apps/web 的父目录的父目录。
  return path.resolve(process.cwd(), '../../', layerPath)
}

// 停止子进程：用进程组信号（负 PID）杀掉 bun 以及它派生的 nuxt dev 子进程。
// 仅 kill bun 会让真正占用端口的 node 进程变成孤儿继续服务，
// 导致重启后新进程抢不到端口、前台停留在旧主题。
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
      console.error('[sforum-dev-runtime] failed to signal child group:', error.message)
    }
  }
}

// 蓝绿切换到指定 layer：spawn 新 nuxt dev 子进程（临时端口），解析它的监听端口，
// 健康检查通过后切代理 upstream 并 drain 旧子进程。
// 候选不可用时保留旧 child 继续服务。
async function switchTo(nextLayer, reason) {
  const env = { ...process.env }
  // PORT=0 让 listhen/get-port-please 自动选临时端口，避免与旧子进程争 3000。
  env.PORT = '0'
  if (nextLayer && fs.existsSync(nextLayer)) {
    env.SFORUM_THEME_LAYER = nextLayer
    console.log(`[sforum-dev-runtime] (${reason}) starting nuxt dev with theme layer: ${nextLayer}`)
  } else {
    delete env.SFORUM_THEME_LAYER
    console.log(`[sforum-dev-runtime] (${reason}) starting nuxt dev with default theme`)
  }

  // 用 Promise 在子进程 stdout 里解析出监听端口，再交给 replaceTarget。
  let resolvePort
  let rejectPort
  const portPromise = new Promise((resolve, reject) => {
    resolvePort = resolve
    rejectPort = reject
  })

  const result = await replaceTarget({
    proxy,
    oldChild: child,
    spawnCandidate: () => {
      const candidate = spawn(bunPath, ['run', 'dev:plain'], {
        // stdout 改 pipe 以解析监听端口；其余继承父进程便于调试。
        stdio: ['inherit', 'pipe', 'inherit'],
        env,
        // detached 让子进程成为独立进程组组长，便于用 -pid 杀整个进程组。
        detached: true,
      })
      // 子进程崩了就 reject，让 replaceTarget 走候选失败路径。
      candidate.on('exit', (code, signal) => {
        console.error(`[sforum-dev-runtime] nuxt dev exited code=${code ?? ''} signal=${signal ?? ''}`)
        rejectPort(new Error(`nuxt dev exited before ready (code=${code}, signal=${signal})`))
      })
      // 扫描每一行 stdout，匹配到监听端口就 resolve，并把后续 stdout 透传给父进程。
      let pending = ''
      candidate.stdout.on('data', (chunk) => {
        pending += chunk.toString()
        let nl
        while ((nl = pending.indexOf('\n')) >= 0) {
          const line = pending.slice(0, nl)
          pending = pending.slice(nl + 1)
          const parsed = parseDevPort(line)
          if (parsed) {
            resolvePort(parsed)
          }
          if (!isNuxtDevAddressLine(line)) {
            process.stdout.write(`${line}\n`)
          }
        }
      })
      candidate.stdout.on('end', () => {
        if (pending && !isNuxtDevAddressLine(pending)) {
          process.stdout.write(pending)
        }
        pending = ''
      })
      return candidate
    },
    healthCheck: async (candidate) => {
      const { host, port } = await withTimeout(portPromise, healthTimeoutMs)
      await healthCheckTcp(candidate, host, port, { timeoutMs: healthTimeoutMs })
    },
    stopChild,
    healthTimeoutMs,
    onSpawnError: (err) => console.error('[sforum-dev-runtime] spawn candidate failed:', err.message),
    onHealthFailed: (err) => console.error('[sforum-dev-runtime] candidate health check failed:', err.message),
  })

  child = result.child
  if (result.ok) {
    activeLayer = nextLayer
    console.log(`[sforum-dev-runtime] (${reason}) switched nuxt dev; public URL: ${publicDevUrl}`)
    // 主动重启场景下（非 current.json 变化）：进程已退出会触发 exit，
    // 这里 child 仍存活时无需额外处理；崩溃由 exit 回调兜底重拉。
    child.removeAllListeners('exit')
    child.on('exit', (code, signal) => {
      console.error(`[sforum-dev-runtime] nuxt dev exited code=${code ?? ''} signal=${signal ?? ''}`)
      child = null
      activeLayer = null
      // 非主动重启时，1s 后尝试拉起，避免 dev 崩溃后整个 supervisor 退出。
      if (code !== 0 && code !== null) {
        setTimeout(() => startDev('nuxt-dev-restart'), 1000)
      }
    })
  }
  return result.ok
}

// 根据 current.json 决定下一个 layer，并触发蓝绿切换。
// 进行中的切换会被记忆，切换串行执行避免并发 spawn。
async function startDev(reason = 'startup') {
  const nextLayer = readLayerPath()
  if (nextLayer === activeLayer && child) {
    return
  }
  if (nextLayer && !fs.existsSync(nextLayer)) {
    // layer 目录不存在时不要盲目重启，保留当前 dev 继续可用。
    console.error(`[sforum-dev-runtime] theme layer does not exist: ${nextLayer}`)
    if (child) {
      return
    }
  }
  if (switching) {
    pendingLayer = nextLayer
    pendingLayerReason = reason
    return
  }
  switching = true
  try {
    await switchTo(nextLayer, reason)
  } finally {
    switching = false
  }
  if (pendingLayer !== null && pendingLayer !== undefined && pendingLayer !== activeLayer) {
    const next = pendingLayer
    const nextReason = pendingLayerReason
    pendingLayer = null
    pendingLayerReason = null
    await startDev(nextReason || 'pending')
  }
}

// 给 Promise 套一个超时（dev 版本内联，避免从 theme-proxy 暴露内部 helper）。
function withTimeout(promise, ms) {
  return new Promise((resolve, reject) => {
    const timer = setTimeout(() => reject(new Error(`waiting for nuxt dev port timed out after ${ms}ms`)), ms)
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

function scheduleRestart() {
  clearTimeout(restartTimer)
  restartTimer = setTimeout(() => {
    startDev('current.json changed').catch((err) => console.error('[sforum-dev-runtime] restart failed:', err.message))
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
  // 先启动 nuxt dev 子进程并等它通过健康检查，再对外监听。
  // 否则冷启动窗口期 activeTarget===null，所有请求返回 502，
  // SPA 页面（如 /login）拿不到 entry.async.js 就白屏。
  await startDev('startup')
  if (!proxy.getTarget()) {
    console.error('[sforum-dev-runtime] initial nuxt dev did not become ready; exiting')
    process.exit(1)
  }
  await proxy.listen()
  console.log(`[sforum-dev-runtime] proxy listening on ${externalHost}:${externalPort}`)
  console.log(`[sforum-dev-runtime] public URL: ${publicDevUrl}`)
  fs.watch(releaseRoot, scheduleRestart)
}

main().catch((err) => {
  console.error('[sforum-dev-runtime] fatal:', err.message)
  process.exit(1)
})
