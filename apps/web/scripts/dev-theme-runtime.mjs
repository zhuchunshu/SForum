// 本地开发的主题感知 supervisor：包裹裸 `nuxt dev`。
//
// 默认（P1 dev-compose）：Nuxt **直连**固定 PORT（与裸 nuxt dev 同路径），
// 不经反向代理、不等 HTTP health——端口打印即视为 ready，体感接近秒启动。
//
// SFORUM_DEV_USE_RELEASE=1：完整 Web Release 模式仍用代理 + 随机 PORT，
// 便于切换 release 时串行换层。
//
// 主题/registry 切换时仍会停旧进程再起新 Nuxt（串行，短暂不可用）。
import { spawn } from 'node:child_process'
import fs from 'node:fs'
import net from 'node:net'
import path from 'node:path'

import {
  composeDevAdmin,
  DEV_COMPOSE_DIRNAME,
  DEV_COMPOSE_RELEASE_ID,
  watchDevAdminCompose,
} from './dev-admin-compose.mjs'
import {
  clearNuxtRouteCache,
  createDevThemeLifecycle,
  readThemeSelection,
  stopProcessGroup,
} from './dev-theme-lifecycle.mjs'
import {
  createThemeProxy,
  formatPublicDevUrl,
  isNuxtDevAddressLine,
  makeTarget,
  parseDevPort,
} from './theme-proxy.mjs'
import {
  watchableReleaseFile,
  writeActiveAcknowledgement,
  writeFailureAcknowledgement,
} from './web-release-contract.mjs'

const repoRoot = path.resolve(process.cwd(), '../../')
const releaseRoot = process.env.SFORUM_WEB_RELEASE_ROOT || process.env.SFORUM_THEME_RELEASE_ROOT || path.join(repoRoot, 'storage/theme-releases')
const currentFile = path.join(releaseRoot, 'current.json')
const bunPath = process.env.SFORUM_BUN_PATH || 'bun'
const nuxtBuildDir = path.resolve(process.cwd(), process.env.NUXT_BUILD_DIR || '.nuxt')
const externalPort = Number(process.env.PORT || process.env.WEB_PORT || '3000')
const externalHost = process.env.HOST || '0.0.0.0'
const publicDevUrl = formatPublicDevUrl(externalHost, externalPort)
const healthTimeoutMs = Number(process.env.SFORUM_THEME_HEALTH_TIMEOUT || '120000')
// 默认轻量源码 compose；设为 1/true 时消费完整 Web Release（current.json）。
const useFullRelease = ['1', 'true', 'yes'].includes(String(process.env.SFORUM_DEV_USE_RELEASE || '').toLowerCase())
// 完整 release 才需要代理换层；dev-compose 直连固定端口，少一层跳转与 health 等待。
const useProxy = useFullRelease
const composeOutDir = path.join(releaseRoot, DEV_COMPOSE_DIRNAME)

let restartTimer = null
let watcher = null
let composeWatcher = null
let lifecycle = null
let exiting = false
/** @type {ReturnType<typeof composeDevAdmin> | null} */
let latestCompose = null

const proxy = useProxy ? createThemeProxy({ externalPort, host: externalHost }) : null

function currentSelection() {
  if (!useFullRelease) {
    const composed = latestCompose || composeDevAdmin({ repoRoot, outDir: composeOutDir })
    latestCompose = composed
    return selectionFromCompose(composed)
  }
  return readThemeSelection(currentFile, {
    repoRoot,
    onError: (error) => console.error('[sforum-dev-runtime] invalid current release:', error.message),
  })
}

function selectionFromCompose(composed) {
  // 有 admin registry 即视为 compose 选择，不强制 theme Nuxt Layer。
  // 公开主题走 host Page Registry；layerPath 可为空。
  const hasRegistry = Boolean(composed.registryRoot)
  const hasLayer = Boolean(composed.themeLayer)
  return {
    mode: hasRegistry || hasLayer ? 'uploaded' : 'default',
    layerPath: composed.themeLayer || '',
    registryRoot: composed.registryRoot || '',
    devInput: composed.outDir,
    releaseId: composed.releaseId,
    compositionHash: composed.compositionHash,
    artifactDigest: '',
    serverEntry: '',
    themeId: composed.themeId || '',
    themeVersion: composed.themeVersion || '',
    reloadMode: 'prompt',
  }
}

function isNumericReleaseId(value) {
  return typeof value === 'number'
    ? Number.isInteger(value) && value > 0
    : /^\d+$/.test(String(value || ''))
}

function launchDevChild(selection, reason) {
  // 直连模式：Nuxt 占固定 public port；代理模式：PORT=0 随机端口再由 proxy 转发。
  const env = {
    ...process.env,
    PORT: useProxy ? '0' : String(externalPort),
    HOST: externalHost,
  }
  const hasRegistry = Boolean(selection.registryRoot)
  const hasLayer = Boolean(selection.layerPath)
  // uploaded / compose：可只注入 admin registry（无公开主题 Layer 是预期路径）。
  if (selection.mode === 'uploaded' || hasRegistry) {
    if (hasLayer) {
      if (!fs.existsSync(selection.layerPath)) {
        throw new Error(`theme layer does not exist: ${selection.layerPath}`)
      }
      env.SFORUM_THEME_LAYER = selection.layerPath
    } else {
      // 宿主公开页 + Page Registry；不再要求 Nuxt Layer。
      delete env.SFORUM_THEME_LAYER
    }
    if (hasRegistry) {
      env.SFORUM_ADMIN_REGISTRY_ROOT = selection.registryRoot
    } else {
      delete env.SFORUM_ADMIN_REGISTRY_ROOT
    }
    if (selection.releaseId) env.SFORUM_WEB_RELEASE_ID = String(selection.releaseId)
    env.SFORUM_WEB_RELEASE_RELOAD_MODE = selection.reloadMode === 'force' ? 'force' : 'prompt'
    const source = selection.releaseId === DEV_COMPOSE_RELEASE_ID ? 'dev-compose' : 'web-release'
    const layerNote = hasLayer ? selection.layerPath : '(none; host pages + admin registry)'
    console.log(`[sforum-dev-runtime] (${reason}) starting nuxt dev [${source}] layer=${layerNote}`)
  } else {
    delete env.SFORUM_THEME_LAYER
    delete env.SFORUM_ADMIN_REGISTRY_ROOT
    delete env.SFORUM_WEB_RELEASE_ID
    delete env.SFORUM_WEB_RELEASE_RELOAD_MODE
    console.log(`[sforum-dev-runtime] (${reason}) starting nuxt dev with default theme`)
  }

  // 仅主题/registry 切换时清 Nitro 路由缓存；冷启动保留 .nuxt 加速二次启动。
  if (reason !== 'startup') {
    clearNuxtRouteCache(nuxtBuildDir)
  }

  // 内层必须是裸 nuxt（dev:nuxt），不要用 dev:plain。
  const candidate = spawn(bunPath, ['run', 'dev:nuxt'], {
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
      // 直连模式：Nuxt 自己的 Local/Network 行就是用户该打开的地址，原样打印。
      // 代理模式：隐藏随机端口行，避免和 public URL 混淆。
      if (useProxy) {
        if (!isNuxtDevAddressLine(line)) process.stdout.write(`${line}\n`)
      } else {
        process.stdout.write(`${line}\n`)
      }
    }
  })
  candidate.stdout.on('end', () => {
    if (pending) {
      if (useProxy) {
        if (!isNuxtDevAddressLine(pending)) process.stdout.write(pending)
      } else {
        process.stdout.write(pending)
      }
    }
    pending = ''
  })

  const ready = Promise.race([
    withTimeout(portPromise, healthTimeoutMs).then(async (address) => {
      const host = address.host || '127.0.0.1'
      const port = address.port
      // 端口已监听即 ready（与裸 nuxt 打印 URL 时点一致）；不再轮询 GET / 等编译完。
      await waitForTcpListen(host, port, { timeoutMs: Math.min(healthTimeoutMs, 30_000) })
      const target = makeTarget({ host, port })
      candidate._target = target
      return target
    }),
    exitPromise,
  ]).finally(() => {
    candidate.removeListener('exit', onExit)
    candidate.removeListener('error', onError)
  })

  return { child: candidate, ready }
}

/** TCP 能 connect 即表示 listen 成功；比 HTTP health 早，接近裸 nuxt 可访问时点。 */
function waitForTcpListen(host, port, { timeoutMs = 30_000, intervalMs = 50 } = {}) {
  const deadline = Date.now() + timeoutMs
  return new Promise((resolve, reject) => {
    const tryOnce = () => {
      const socket = net.connect({ host, port }, () => {
        socket.end()
        resolve()
      })
      socket.on('error', () => {
        socket.destroy()
        if (Date.now() >= deadline) {
          reject(new Error(`tcp listen wait failed for ${host}:${port}`))
          return
        }
        setTimeout(tryOnce, intervalMs)
      })
    }
    tryOnce()
  })
}

function createLifecycle() {
  return createDevThemeLifecycle({
    readSelection: currentSelection,
    launchChild: launchDevChild,
    stopChild: (target) => stopProcessGroup(target),
    setTarget: (target) => {
      if (proxy) proxy.setTarget(target)
    },
    onFatal: (error) => { void failRuntime(error) },
    onActive: (selection, reason) => {
      console.log(`[sforum-dev-runtime] (${reason}) nuxt ready; public URL: ${publicDevUrl}`)
      // 仅数字 release id 写 active.json，避免把 dev-local 当成生产 release 确认。
      if (isNumericReleaseId(selection.releaseId)) {
        void writeActiveAcknowledgement(releaseRoot, {
          releaseId: Number(selection.releaseId),
          compositionHash: selection.compositionHash,
          artifactDigest: selection.artifactDigest,
          serverEntry: selection.serverEntry,
          themeId: selection.themeId,
          themeVersion: selection.themeVersion,
          reloadMode: selection.reloadMode,
        }).catch((error) => { void failRuntime(error) })
      }
    },
    onCandidateFailed: (selection, error) => {
      if (isNumericReleaseId(selection.releaseId)) {
        void writeFailureAcknowledgement(releaseRoot, {
          releaseId: Number(selection.releaseId),
          reason: 'web_release.start_failed',
          message: error.message,
        }).catch((writeError) => { void failRuntime(writeError) })
      }
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
  // dev-compose 目录下的重写不应触发「完整 release」重启逻辑。
  if (changed.startsWith(DEV_COMPOSE_DIRNAME)) return
  if (changed && !watchableReleaseFile(changed)) return

  clearTimeout(restartTimer)
  restartTimer = setTimeout(() => {
    lifecycle.requestRestart('current.json changed').catch((error) => {
      void failRuntime(error)
    })
  }, 250)
}

function onComposeUpdated(result, reason) {
  const previousHash = latestCompose?.compositionHash
  latestCompose = result
  // startup 由 main 统一 requestRestart；后续 hash 变化才重启（locales/manifest）。
  // 纯 .vue 改动 hash 不变（admin 路径固定），靠软链 + Vite HMR。
  if (reason === 'startup') return
  if (previousHash && previousHash === result.compositionHash) {
    console.log(`[sforum-dev-runtime] compose unchanged after ${reason}; relying on HMR`)
    return
  }
  if (!lifecycle) return
  lifecycle.requestRestart(`dev-compose:${reason}`).catch((error) => {
    void failRuntime(error)
  })
}

async function shutdown(exitCode = 0) {
  if (exiting) return
  exiting = true
  clearTimeout(restartTimer)
  watcher?.close()
  composeWatcher?.close()
  await lifecycle?.shutdown()
  if (proxy) await proxy.close()
  process.exit(exitCode)
}

async function failRuntime(error) {
  if (exiting) return
  console.error('[sforum-dev-runtime] fatal:', error.message)
  await shutdown(1)
}

async function main() {
  fs.mkdirSync(releaseRoot, { recursive: true })

  if (useFullRelease) {
    console.log('[sforum-dev-runtime] SFORUM_DEV_USE_RELEASE=1 → full Web Release + proxy')
  } else {
    console.log('[sforum-dev-runtime] dev-compose direct mode (no proxy); set SFORUM_DEV_USE_RELEASE=1 for full release')
    composeWatcher = watchDevAdminCompose({
      repoRoot,
      outDir: composeOutDir,
      onComposed: onComposeUpdated,
      onError: (error) => console.error('[sforum-dev-compose]', error.message),
    })
    latestCompose = composeWatcher.initial
  }

  lifecycle = createLifecycle()
  process.on('SIGTERM', () => { void shutdown(0) })
  process.on('SIGINT', () => { void shutdown(0) })

  // 代理模式：先占住对外端口再起 Nuxt，避免「端口空着」的空窗；直连模式由 Nuxt 自己 bind。
  if (proxy) {
    await proxy.listen()
    console.log(`[sforum-dev-runtime] proxy listening on ${externalHost}:${externalPort}`)
  }

  await lifecycle.requestRestart('startup')
  console.log(`[sforum-dev-runtime] public URL: ${publicDevUrl}`)

  if (useFullRelease) {
    watcher = fs.watch(releaseRoot, scheduleRestart)
  }
}

main().catch((error) => { void failRuntime(error) })
