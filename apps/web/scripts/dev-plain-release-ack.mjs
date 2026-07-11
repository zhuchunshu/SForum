// plain nuxt 开发旁路：只确认 Web Release 激活，不重启进程、不注入主题层。
//
// 完整 `bun run dev` 会监听 current.json 并重启带主题 layer 的 nuxt；
// `dev:plain` 故意绕过那套，但仍需要写 active.json，否则插件启停会卡在 activating 88%。
import fs from 'node:fs'
import path from 'node:path'

import {
  readDesiredRelease,
  watchableReleaseFile,
  writeActiveAcknowledgement,
} from './web-release-contract.mjs'

/**
 * @param {{
 *   releaseRoot: string
 *   log?: (message: string) => void
 *   error?: (message: string) => void
 * }} options
 */
export function startPlainReleaseAck({
  releaseRoot,
  log = (message) => console.log(`[sforum-dev-plain] ${message}`),
  error = (message) => console.error(`[sforum-dev-plain] ${message}`),
} = {}) {
  if (!releaseRoot) {
    throw new Error('releaseRoot is required')
  }
  fs.mkdirSync(releaseRoot, { recursive: true })

  let timer = null
  let closed = false
  /** @type {Promise<void>} */
  let chain = Promise.resolve()

  // 串行化确认，避免 startup 与 watch 并发时 await sync 提前返回。
  function sync(reason = 'current.json changed') {
    if (closed) return Promise.resolve()
    const run = chain.then(() => {
      if (closed) return
      return acknowledgeDesired(reason)
    })
    chain = run.catch(() => {})
    return run
  }

  async function acknowledgeDesired(reason) {
    const desired = readDesiredRelease({
      releaseRoot,
      fallback: { serverEntry: '__plain_dev__' },
    })
    if (desired.kind !== 'release') {
      return
    }
    // plain 模式不切换主题 layer / 不跑产物 server；只告诉 API「开发侧已接受该 composition」。
    await writeActiveAcknowledgement(releaseRoot, {
      releaseId: desired.releaseId,
      compositionHash: desired.compositionHash,
      artifactDigest: desired.artifactDigest,
      serverEntry: desired.serverEntry,
      themeId: desired.themeId,
      themeVersion: desired.themeVersion,
      reloadMode: desired.reloadMode,
    })
    log(`(${reason}) acknowledged web release #${desired.releaseId} without restarting nuxt`)
  }

  function schedule(_eventType, filename) {
    const changed = filename ? filename.toString() : ''
    if (changed && !watchableReleaseFile(changed)) return
    clearTimeout(timer)
    timer = setTimeout(() => {
      sync('current.json changed').catch((err) => {
        error(`release ack failed: ${err.message}`)
      })
    }, 250)
  }

  // 启动时先同步一次，避免 API 已写 current.json 而 plain 进程后起的窗口。
  void sync('startup').catch((err) => {
    error(`startup release ack failed: ${err.message}`)
  })

  let watcher = null
  try {
    watcher = fs.watch(releaseRoot, schedule)
  } catch (err) {
    error(`watch ${releaseRoot} failed: ${err.message}`)
  }

  return {
    sync,
    close() {
      closed = true
      clearTimeout(timer)
      watcher?.close()
    },
  }
}

export function resolvePlainReleaseRoot({
  cwd = process.cwd(),
  env = process.env,
} = {}) {
  const repoRoot = path.resolve(cwd, '../../')
  return env.SFORUM_WEB_RELEASE_ROOT
    || env.SFORUM_THEME_RELEASE_ROOT
    || path.join(repoRoot, 'storage/theme-releases')
}
