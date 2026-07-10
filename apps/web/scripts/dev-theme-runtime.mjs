// 本地开发的主题感知 supervisor：包裹 `bun run dev:plain`（原始 nuxt dev）。
//
// Nuxt dev 会独占 buildDir lock、生成目录与 HMR 资源，因此本地主题切换采用串行重启：
// 清空代理目标 -> 停止并等待旧进程组退出 -> 启动最新主题 -> 健康后恢复代理。
// 切换期间允许短暂不可用；生产 runtime.mjs 继续使用 Nitro 蓝绿切换。
import { spawn } from 'node:child_process'
import fs from 'node:fs'
import path from 'node:path'

import {
  createDevThemeLifecycle,
  readThemeSelection,
  stopProcessGroup,
} from './dev-theme-lifecycle.mjs'
import {
  createThemeProxy,
  formatPublicDevUrl,
  healthCheckTcp,
  isNuxtDevAddressLine,
  parseDevPort,
} from './theme-proxy.mjs'

const repoRoot = path.resolve(process.cwd(), '../../')
const releaseRoot = process.env.SFORUM_THEME_RELEASE_ROOT || path.join(repoRoot, 'storage/theme-releases')
const currentFile = path.join(releaseRoot, 'current.json')
const bunPath = process.env.SFORUM_BUN_PATH || 'bun'
const externalPort = Number(process.env.PORT || process.env.WEB_PORT || '3000')
const externalHost = process.env.HOST || '0.0.0.0'
const publicDevUrl = formatPublicDevUrl(externalHost, externalPort)
const healthTimeoutMs = Number(process.env.SFORUM_THEME_HEALTH_TIMEOUT || '120000')

let restartTimer = null
let watcher = null
let lifecycle = null
let exiting = false

const proxy = createThemeProxy({ externalPort, host: externalHost })

function currentSelection() {
  return readThemeSelection(currentFile, {
    repoRoot,
    onError: (error) => console.error('[sforum-dev-runtime] invalid current release:', error.message),
  })
}

function launchDevChild(selection, reason) {
  const env = { ...process.env, PORT: '0' }
  if (selection.mode === 'uploaded') {
    if (!selection.layerPath || !fs.existsSync(selection.layerPath)) {
      throw new Error(`theme layer does not exist: ${selection.layerPath}`)
    }
    env.SFORUM_THEME_LAYER = selection.layerPath
    console.log(`[sforum-dev-runtime] (${reason}) starting nuxt dev with theme layer: ${selection.layerPath}`)
  } else {
    delete env.SFORUM_THEME_LAYER
    console.log(`[sforum-dev-runtime] (${reason}) starting nuxt dev with default theme`)
  }

  const candidate = spawn(bunPath, ['run', 'dev:plain'], {
    stdio: ['inherit', 'pipe', 'inherit'],
    env,
    detached: true,
  })

  let resolvePort
  const portPromise = new Promise((resolve) => {
    resolvePort = resolve
  })
  let rejectExit
  const exitPromise = new Promise((_, reject) => {
    rejectExit = reject
  })
  const onExit = (code, signal) => rejectExit(
    new Error(`nuxt dev exited before ready (code=${code}, signal=${signal})`),
  )
  const onError = (error) => rejectExit(error)
  candidate.once('exit', onExit)
  candidate.once('error', onError)

  let pending = ''
  candidate.stdout.on('data', (chunk) => {
    pending += chunk.toString()
    let newline
    while ((newline = pending.indexOf('\n')) >= 0) {
      const line = pending.slice(0, newline)
      pending = pending.slice(newline + 1)
      const address = parseDevPort(line)
      if (address) resolvePort(address)
      if (!isNuxtDevAddressLine(line)) process.stdout.write(`${line}\n`)
    }
  })
  candidate.stdout.on('end', () => {
    if (pending && !isNuxtDevAddressLine(pending)) process.stdout.write(pending)
    pending = ''
  })

  const healthPromise = (async () => {
    const { host, port } = await withTimeout(portPromise, healthTimeoutMs)
    await healthCheckTcp(candidate, host, port, { timeoutMs: healthTimeoutMs })
    return candidate._target
  })()
  const ready = Promise.race([healthPromise, exitPromise]).finally(() => {
    candidate.removeListener('exit', onExit)
    candidate.removeListener('error', onError)
  })

  return { child: candidate, ready }
}

function createLifecycle() {
  return createDevThemeLifecycle({
    readSelection: currentSelection,
    launchChild: launchDevChild,
    stopChild: (target) => stopProcessGroup(target),
    setTarget: (target) => proxy.setTarget(target),
    onFatal: (error) => { void failRuntime(error) },
    onActive: (_selection, reason) => {
      console.log(`[sforum-dev-runtime] (${reason}) switched nuxt dev; public URL: ${publicDevUrl}`)
    },
  })
}

function withTimeout(promise, ms) {
  return new Promise((resolve, reject) => {
    const timer = setTimeout(() => reject(new Error(`waiting for nuxt dev port timed out after ${ms}ms`)), ms)
    promise.then(
      (value) => {
        clearTimeout(timer)
        resolve(value)
      },
      (error) => {
        clearTimeout(timer)
        reject(error)
      },
    )
  })
}

function scheduleRestart(_eventType, filename) {
  const changed = filename ? filename.toString() : ''
  if (changed && changed !== 'current.json' && changed !== 'current.json.tmp') return

  clearTimeout(restartTimer)
  restartTimer = setTimeout(() => {
    lifecycle.requestRestart('current.json changed').catch((error) => {
      void failRuntime(error)
    })
  }, 250)
}

async function shutdown(exitCode = 0) {
  if (exiting) return
  exiting = true
  clearTimeout(restartTimer)
  watcher?.close()
  await lifecycle?.shutdown()
  await proxy.close()
  process.exit(exitCode)
}

async function failRuntime(error) {
  if (exiting) return
  console.error('[sforum-dev-runtime] fatal:', error.message)
  await shutdown(1)
}

async function main() {
  fs.mkdirSync(releaseRoot, { recursive: true })
  lifecycle = createLifecycle()
  process.on('SIGTERM', () => { void shutdown(0) })
  process.on('SIGINT', () => { void shutdown(0) })

  // 冷启动仍先等内部 Nuxt 健康，再占用对外端口，避免启动窗口返回误导性 502。
  await lifecycle.requestRestart('startup')
  if (!proxy.getTarget()) {
    throw new Error('initial nuxt dev did not become ready')
  }

  await proxy.listen()
  console.log(`[sforum-dev-runtime] proxy listening on ${externalHost}:${externalPort}`)
  console.log(`[sforum-dev-runtime] public URL: ${publicDevUrl}`)
  watcher = fs.watch(releaseRoot, scheduleRestart)
}

main().catch((error) => { void failRuntime(error) })
