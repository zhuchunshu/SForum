import fs from 'node:fs'
import path from 'node:path'

function defaultSelection() {
  return { mode: 'default', layerPath: '' }
}

export function readThemeSelection(currentFile, {
  repoRoot = path.resolve(process.cwd(), '../../'),
  onError = (error) => console.error('[sforum-dev-runtime] invalid current release:', error.message),
} = {}) {
  let raw
  try {
    raw = fs.readFileSync(currentFile, 'utf8')
  } catch (error) {
    if (error.code !== 'ENOENT') onError(error)
    return defaultSelection()
  }

  let current
  try {
    current = JSON.parse(raw)
  } catch (error) {
    onError(error)
    return defaultSelection()
  }

  if (current.mode === 'default') return defaultSelection()

  const rawLayerPath = typeof current.layerPath === 'string'
    ? current.layerPath.trim()
    : ''
  if (!rawLayerPath) return defaultSelection()

  return {
    mode: 'uploaded',
    layerPath: path.isAbsolute(rawLayerPath)
      ? rawLayerPath
      : path.resolve(repoRoot, rawLayerPath),
  }
}

export function themeSelectionKey(selection) {
  return `${selection.mode}:${selection.layerPath}`
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
  while (groupExists(pid)) {
    if (Date.now() >= deadline) return false
    await sleepFn(Math.min(pollMs, Math.max(0, deadline - Date.now())))
  }
  return true
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
