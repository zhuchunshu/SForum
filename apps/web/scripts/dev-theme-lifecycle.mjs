import fs from 'node:fs'
import path from 'node:path'
import { readDesiredRelease, readReleaseManifest } from './web-release-contract.mjs'

export function clearNuxtRouteCache(buildDir) {
  fs.rmSync(path.join(buildDir, 'cache', 'nitro', 'routes'), { recursive: true, force: true })
}

function defaultSelection() {
  return { mode: 'default', layerPath: '' }
}

export function readThemeSelection(currentFile, {
  repoRoot = path.resolve(process.cwd(), '../../'),
  onError = (error) => console.error('[sforum-dev-runtime] invalid current release:', error.message),
} = {}) {
  try {
    const releaseRoot = path.dirname(currentFile)
    const desired = readDesiredRelease({ releaseRoot, legacyRoot: repoRoot, fallback: { serverEntry: '__development__' } })
    if (desired.kind === 'release') {
      const release = readReleaseManifest(desired)
      return {
        mode: 'uploaded',
        layerPath: release.themeLayer,
        registryRoot: release.registryRoot,
        devInput: release.devInput,
        releaseId: String(release.releaseId),
        compositionHash: release.compositionHash,
        artifactDigest: release.artifactDigest,
        serverEntry: release.serverEntry,
        themeId: release.themeId,
        themeVersion: release.themeVersion,
        reloadMode: release.reloadMode,
      }
    }
    if (desired.kind === 'legacy' && desired.themeLayer) {
      return { mode: 'uploaded', layerPath: desired.themeLayer }
    }
    return defaultSelection()
  } catch (error) {
    onError(error)
    return defaultSelection()
  }
}

export function themeSelectionKey(selection) {
  const base = `${selection.mode}:${selection.layerPath}`
  const release = [selection.registryRoot, selection.releaseId, selection.compositionHash]
    .map((value) => value || '')
  return release.some(Boolean) ? `${base}:${release.join(':')}` : base
}

function defaultSignalGroup(pid, signal) {
  try {
    process.kill(-pid, signal)
  } catch (error) {
    if (error.code !== 'ESRCH') throw error
  }
}

function defaultGroupExists(pid) {
  try {
    process.kill(-pid, 0)
    return true
  } catch (error) {
    if (error.code === 'ESRCH') return false
    throw error
  }
}

function sleep(ms) {
  return new Promise((resolve) => setTimeout(resolve, ms))
}

async function waitForGroupExit(pid, timeoutMs, {
  groupExists,
  pollMs,
  sleepFn,
}) {
  const deadline = Date.now() + timeoutMs
  while (true) {
    try {
      if (!groupExists(pid)) return true
    } catch (error) {
      // macOS 可能在进程组 leader 已退出但尚未回收时返回 EPERM，此时组仍需等待。
      if (error.code !== 'EPERM') throw error
    }
    if (Date.now() >= deadline) return false
    await sleepFn(Math.min(pollMs, Math.max(0, deadline - Date.now())))
  }
}

export async function stopProcessGroup(child, {
  graceMs = 5000,
  killWaitMs = 1000,
  pollMs = 50,
  signalGroup = defaultSignalGroup,
  groupExists = defaultGroupExists,
  sleepFn = sleep,
} = {}) {
  const pid = child?.pid
  if (!pid) return

  signalGroup(pid, 'SIGTERM')
  if (await waitForGroupExit(pid, graceMs, { groupExists, pollMs, sleepFn })) {
    return
  }

  signalGroup(pid, 'SIGKILL')
  if (!await waitForGroupExit(pid, killWaitMs, { groupExists, pollMs, sleepFn })) {
    throw new Error(`process group ${pid} did not exit after SIGKILL`)
  }
}

class CandidateCleanupError extends AggregateError {
  constructor(readinessError, cleanupError) {
    super(
      [readinessError, cleanupError],
      'candidate failed to start and its process group could not be stopped',
    )
    this.name = 'CandidateCleanupError'
  }
}

export function createDevThemeLifecycle({
  readSelection,
  launchChild,
  stopChild,
  setTarget,
  logger = console,
  onFatal = (error) => logger.error('[sforum-dev-runtime] recovery failed:', error.message),
  onActive = () => {},
  onCandidateFailed = () => {},
  recoveryDelayMs = 1000,
  setTimeoutFn = setTimeout,
  clearTimeoutFn = clearTimeout,
}) {
  let activeChild = null
  let activeSelection = null
  let startingChild = null
  let lastHealthySelection = null
  let desiredSelection = null
  let restartRequested = false
  let restartPromise = null
  let recoveryTimer = null
  let crashCleanupPromise = null
  let shuttingDown = false

  function installActive(started, selection, reason) {
    const installedChild = started.child
    activeChild = installedChild
    activeSelection = selection
    lastHealthySelection = selection
    setTarget(started.target)
    onActive(selection, reason)

    // 旧进程可能在新进程安装后才发出 exit，只允许当前实例清空活跃状态。
    installedChild.once('exit', (code, signal) => {
      if (activeChild !== installedChild) return

      activeChild = null
      activeSelection = null
      setTarget(null)
      if (shuttingDown) return

      logger.error(`[sforum-dev-runtime] nuxt dev exited code=${code ?? ''} signal=${signal ?? ''}`)
      const cleanupPromise = (async () => {
        try {
          await stopChild(installedChild)
        } catch (error) {
          onFatal(error)
          throw error
        }
        if (shuttingDown) return

        recoveryTimer = setTimeoutFn(() => {
          recoveryTimer = null
          void requestRestart('nuxt-dev-restart').catch(onFatal)
        }, recoveryDelayMs)
      })()
      crashCleanupPromise = cleanupPromise
      void cleanupPromise.then(
        () => {
          if (crashCleanupPromise === cleanupPromise) crashCleanupPromise = null
        },
        () => {
          if (crashCleanupPromise === cleanupPromise) crashCleanupPromise = null
        },
      )
    })
  }

  async function launchHealthy(selection, reason) {
    const launch = launchChild(selection, reason)
    startingChild = launch.child
    try {
      const target = await launch.ready
      if (shuttingDown) {
        await stopChild(launch.child)
        return null
      }
      return { child: launch.child, target }
    } catch (error) {
      try {
        await stopChild(launch.child)
      } catch (cleanupError) {
        throw new CandidateCleanupError(error, cleanupError)
      }
      if (shuttingDown) return null
      throw error
    } finally {
      if (startingChild === launch.child) startingChild = null
    }
  }

  async function runRestartLoop(initialReason) {
    let reason = initialReason
    while (restartRequested && !shuttingDown) {
      const crashCleanup = crashCleanupPromise
      if (crashCleanup) await crashCleanup
      if (shuttingDown) return

      restartRequested = false
      const requestedSelection = desiredSelection

      if (
        activeChild &&
        themeSelectionKey(activeSelection) === themeSelectionKey(requestedSelection)
      ) {
        reason = 'pending current.json change'
        continue
      }

      const previousChild = activeChild
      activeChild = null
      activeSelection = null
      setTarget(null)
      if (previousChild) await stopChild(previousChild)
      if (shuttingDown) return

      let started
      try {
        started = await launchHealthy(requestedSelection, reason)
      } catch (error) {
        logger.error(`[sforum-dev-runtime] (${reason}) candidate failed: ${error.message}`)
        onCandidateFailed(requestedSelection, error)
        if (error instanceof CandidateCleanupError) throw error
        if (restartRequested) {
          reason = 'newer current.json change'
          continue
        }
        if (
          !lastHealthySelection ||
          themeSelectionKey(lastHealthySelection) === themeSelectionKey(requestedSelection)
        ) {
          throw error
        }

        try {
          started = await launchHealthy(lastHealthySelection, 'last-known-good rollback')
        } catch (rollbackError) {
          throw new AggregateError(
            [error, rollbackError],
            'requested theme and last-known-good theme both failed to start',
          )
        }
        if (!started) return
        installActive(started, lastHealthySelection, 'last-known-good rollback')
        reason = 'pending current.json change'
        continue
      }

      if (!started) return
      // 候选启动期间若选择已变化，避免短暂发布一个已经过期的主题。
      if (
        restartRequested &&
        themeSelectionKey(desiredSelection) !== themeSelectionKey(requestedSelection)
      ) {
        await stopChild(started.child)
        reason = 'newer current.json change'
        continue
      }

      installActive(started, requestedSelection, reason)
      reason = 'pending current.json change'
    }
  }

  async function requestRestart(reason = 'current.json changed') {
    if (shuttingDown) return
    desiredSelection = readSelection()
    restartRequested = true
    if (!restartPromise) {
      restartPromise = runRestartLoop(reason).finally(() => {
        restartPromise = null
      })
    }
    return restartPromise
  }

  async function shutdown() {
    if (shuttingDown) return
    shuttingDown = true
    restartRequested = false
    if (recoveryTimer) {
      clearTimeoutFn(recoveryTimer)
      recoveryTimer = null
    }
    setTarget(null)

    const children = [...new Set([startingChild, activeChild].filter(Boolean))]
    const crashCleanup = crashCleanupPromise
    startingChild = null
    activeChild = null
    activeSelection = null
    await Promise.allSettled([
      ...children.map((child) => stopChild(child)),
      ...(crashCleanup ? [crashCleanup] : []),
    ])
    if (restartPromise) await restartPromise.catch(() => {})
  }

  return { requestRestart, shutdown }
}
