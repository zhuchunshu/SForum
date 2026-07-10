# Dev Theme Runtime Serial Restart Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Make local `current.json` theme changes restart exactly one Nuxt dev process serially, converge on the latest selection, and recover the last healthy theme when a requested layer cannot start.

**Architecture:** Keep production `runtime.mjs` and the shared proxy blue-green behavior unchanged. Extract a testable local lifecycle coordinator that represents the default theme explicitly, stops the owned process group before launching a replacement, serializes rapid changes, and protects active state from stale child exit events; keep OS/process/stdout wiring in `dev-theme-runtime.mjs`.

**Tech Stack:** Node.js ESM, Nuxt 4.4, Bun test, Node `child_process`, Node `fs.watch`, existing `theme-proxy.mjs` TCP health checks.

---

## File Structure

- Create `apps/web/scripts/dev-theme-lifecycle.mjs`: parse explicit theme selections, stop process groups with TERM/KILL waiting, and coordinate serial restart/rollback/crash recovery with injected process and proxy operations.
- Create `apps/web/tests/devThemeLifecycle.test.ts`: behavior tests for selection parsing, process-group waiting, ordering, rapid changes, stale exits, rollback, duplicate events, crash recovery, and shutdown.
- Modify `apps/web/scripts/dev-theme-runtime.mjs`: retain environment resolution, Nuxt stdout parsing, child spawning, file watching, signal handling, and proxy startup; delegate lifecycle state to the new module.
- Modify `apps/web/scripts/theme-proxy.mjs`: update comments so zero-downtime claims remain scoped to the production Nitro caller; do not change proxy behavior.
- Modify `apps/web/tests/devRuntimeStartup.test.ts`: lock the executable wiring to the lifecycle module and reject the old `replaceTarget` development path.
- Modify `tests/validate-theme-runtime.js`: validate the split local runtime contract while keeping production assertions intact.
- Modify `knowledge/index.md`, `knowledge/modules/frontend.md`, and `knowledge/modules/extensions.md`: distinguish production blue-green activation from local serial restart.
- Create `knowledge/sessions/2026-07-10-dev-theme-runtime-serial-restart.md`: record the reproduced Nuxt lock failure, decision, changed files, and verification evidence.

### Task 1: Explicit Theme Selection And Process-Group Stop

**Files:**
- Create: `apps/web/scripts/dev-theme-lifecycle.mjs`
- Create: `apps/web/tests/devThemeLifecycle.test.ts`

- [ ] **Step 1: Write failing selection and process-stop tests**

Create `apps/web/tests/devThemeLifecycle.test.ts` with the first behavior block:

```ts
import { afterEach, describe, expect, test } from 'bun:test'
import fs from 'node:fs'
import os from 'node:os'
import path from 'node:path'

import {
  readThemeSelection,
  stopProcessGroup,
  themeSelectionKey,
} from '../scripts/dev-theme-lifecycle.mjs'

const tempRoots: string[] = []
function tempRoot(): string {
  const root = fs.mkdtempSync(path.join(os.tmpdir(), 'sforum-dev-theme-'))
  tempRoots.push(root)
  return root
}

afterEach(() => {
  for (const root of tempRoots.splice(0)) {
    fs.rmSync(root, { recursive: true, force: true })
  }
})

describe('dev theme selection', () => {
  test('represents a missing or default current.json explicitly', () => {
    const root = tempRoot()
    const currentFile = path.join(root, 'current.json')
    expect(readThemeSelection(currentFile, { repoRoot: root })).toEqual({
      mode: 'default',
      layerPath: '',
    })

    fs.writeFileSync(currentFile, JSON.stringify({ mode: 'default' }))
    const selection = readThemeSelection(currentFile, { repoRoot: root })
    expect(selection).toEqual({ mode: 'default', layerPath: '' })
    expect(themeSelectionKey(selection)).toBe('default:')
  })

  test('resolves an uploaded relative layer against the repository root', () => {
    const root = tempRoot()
    const currentFile = path.join(root, 'current.json')
    fs.writeFileSync(currentFile, JSON.stringify({
      mode: 'uploaded',
      layerPath: 'extensions/dev/themes/example/layer',
    }))
    const selection = readThemeSelection(currentFile, { repoRoot: root })
    expect(selection).toEqual({
      mode: 'uploaded',
      layerPath: path.join(root, 'extensions/dev/themes/example/layer'),
    })
    expect(themeSelectionKey(selection)).toContain('uploaded:')
  })
})

describe('stopProcessGroup', () => {
  test('waits after SIGTERM and escalates to SIGKILL when the group remains alive', async () => {
    const signals: string[] = []
    const alive = [true, false]
    await stopProcessGroup({ pid: 42 }, {
      graceMs: 0,
      killWaitMs: 0,
      signalGroup: (_pid, signal) => { signals.push(signal) },
      groupExists: () => alive.shift() ?? false,
    })
    expect(signals).toEqual(['SIGTERM', 'SIGKILL'])
  })
})
```

- [ ] **Step 2: Run the tests and verify RED**

Run:

```bash
cd apps/web && bun test tests/devThemeLifecycle.test.ts
```

Expected: FAIL because `../scripts/dev-theme-lifecycle.mjs` does not exist.

- [ ] **Step 3: Implement the selection and process-group primitives**

Create `apps/web/scripts/dev-theme-lifecycle.mjs` with these exports:

```js
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
```

- [ ] **Step 4: Run the focused tests and verify GREEN**

Run:

```bash
cd apps/web && bun test tests/devThemeLifecycle.test.ts
```

Expected: 3 PASS, 0 FAIL.

- [ ] **Step 5: Commit Task 1**

Run `git add apps/web/scripts/dev-theme-lifecycle.mjs apps/web/tests/devThemeLifecycle.test.ts` and `git commit -m "test: define dev theme lifecycle primitives"`.

### Task 2: Serial Restart Coordinator

**Files:**
- Modify: `apps/web/scripts/dev-theme-lifecycle.mjs`
- Modify: `apps/web/tests/devThemeLifecycle.test.ts`

- [ ] **Step 1: Add failing coordinator tests**

Append imports and helpers to `apps/web/tests/devThemeLifecycle.test.ts`:

```ts
import { EventEmitter } from 'node:events'
import { createDevThemeLifecycle } from '../scripts/dev-theme-lifecycle.mjs'
class FakeChild extends EventEmitter {
  pid: number

  constructor(pid: number) {
    super()
    this.pid = pid
  }
}

function deferred<T>() {
  let resolve!: (value: T) => void
  let reject!: (error: Error) => void
  const promise = new Promise<T>((res, rej) => {
    resolve = res
    reject = rej
  })
  return { promise, resolve, reject }
}
function target(port: number) {
  return { host: '127.0.0.1', port }
}

async function waitFor(condition: () => boolean, timeoutMs = 500) {
  const deadline = Date.now() + timeoutMs
  while (!condition()) {
    if (Date.now() >= deadline) throw new Error('condition was not met before timeout')
    await new Promise((resolve) => setTimeout(resolve, 1))
  }
}
```

Append the coordinator behavior tests:

```ts
describe('createDevThemeLifecycle', () => {
  test('stops the active child before launching its replacement', async () => {
    let selection = { mode: 'default', layerPath: '' }
    const launches: Array<{ child: FakeChild; ready: ReturnType<typeof deferred<ReturnType<typeof target>>> }> = []
    const stopGate = deferred<void>()
    const stopped: FakeChild[] = []
    const lifecycle = createDevThemeLifecycle({
      readSelection: () => selection,
      launchChild: () => {
        const child = new FakeChild(100 + launches.length)
        const ready = deferred<ReturnType<typeof target>>()
        launches.push({ child, ready })
        return { child, ready: ready.promise }
      },
      stopChild: async (child) => {
        stopped.push(child)
        await stopGate.promise
      },
      setTarget: () => {},
    })
    const startup = lifecycle.requestRestart('startup')
    launches[0].ready.resolve(target(4101))
    await startup
    selection = { mode: 'uploaded', layerPath: '/tmp/theme-a' }
    const switching = lifecycle.requestRestart('current.json changed')
    await waitFor(() => stopped.length === 1)
    expect(stopped).toEqual([launches[0].child])
    expect(launches).toHaveLength(1)
    stopGate.resolve()
    await waitFor(() => launches.length === 2)
    expect(launches).toHaveLength(2)
    launches[1].ready.resolve(target(4102))
    await switching
  })

  test('converges on default when default is requested during an uploaded start', async () => {
    let selection = { mode: 'default', layerPath: '' }
    const launches: Array<{ selection: typeof selection; child: FakeChild; ready: ReturnType<typeof deferred<ReturnType<typeof target>>> }> = []
    const targets: Array<ReturnType<typeof target> | null> = []
    const lifecycle = createDevThemeLifecycle({
      readSelection: () => selection,
      launchChild: (nextSelection) => {
        const child = new FakeChild(200 + launches.length)
        const ready = deferred<ReturnType<typeof target>>()
        launches.push({ selection: nextSelection, child, ready })
        return { child, ready: ready.promise }
      },
      stopChild: async (child) => { child.emit('exit', null, 'SIGTERM') },
      setTarget: (nextTarget) => { targets.push(nextTarget) },
    })
    const startup = lifecycle.requestRestart('startup')
    launches[0].ready.resolve(target(4201))
    await startup
    selection = { mode: 'uploaded', layerPath: '/tmp/theme-a' }
    const uploaded = lifecycle.requestRestart('uploaded')
    await waitFor(() => launches.length === 2)
    selection = { mode: 'default', layerPath: '' }
    void lifecycle.requestRestart('default')
    launches[1].ready.resolve(target(4202))
    await waitFor(() => launches.length === 3)
    launches[2].ready.resolve(target(4203))
    await uploaded
    expect(launches.map((item) => item.selection.mode)).toEqual([
      'default',
      'uploaded',
      'default',
    ])
    expect(targets.at(-1)).toEqual(target(4203))
  })

  test('ignores a delayed exit from the replaced child', async () => {
    let selection = { mode: 'default', layerPath: '' }
    const launches: Array<{ child: FakeChild; ready: ReturnType<typeof deferred<ReturnType<typeof target>>> }> = []
    const targets: Array<ReturnType<typeof target> | null> = []
    const lifecycle = createDevThemeLifecycle({
      readSelection: () => selection,
      launchChild: () => {
        const child = new FakeChild(300 + launches.length)
        const ready = deferred<ReturnType<typeof target>>()
        launches.push({ child, ready })
        return { child, ready: ready.promise }
      },
      stopChild: async () => {},
      setTarget: (nextTarget) => { targets.push(nextTarget) },
    })
    const startup = lifecycle.requestRestart('startup')
    launches[0].ready.resolve(target(4301))
    await startup
    selection = { mode: 'uploaded', layerPath: '/tmp/theme-b' }
    const switching = lifecycle.requestRestart('switch')
    await waitFor(() => launches.length === 2)
    launches[1].ready.resolve(target(4302))
    await switching
    launches[0].child.emit('exit', null, 'SIGTERM')
    expect(targets.at(-1)).toEqual(target(4302))
  })

  test('restarts the last healthy selection when the requested selection fails', async () => {
    let selection = { mode: 'default', layerPath: '' }
    const launchedModes: string[] = []
    let launchNumber = 0
    const targets: Array<ReturnType<typeof target> | null> = []
    const lifecycle = createDevThemeLifecycle({
      readSelection: () => selection,
      launchChild: (nextSelection) => {
        launchNumber += 1
        launchedModes.push(nextSelection.mode)
        const child = new FakeChild(400 + launchNumber)
        if (launchNumber === 2) {
          return { child, ready: Promise.reject(new Error('candidate failed')) }
        }
        return { child, ready: Promise.resolve(target(4400 + launchNumber)) }
      },
      stopChild: async (child) => { child.emit('exit', null, 'SIGTERM') },
      setTarget: (nextTarget) => { targets.push(nextTarget) },
      logger: { error: () => {} },
    })
    await lifecycle.requestRestart('startup')
    selection = { mode: 'uploaded', layerPath: '/tmp/broken-theme' }
    await lifecycle.requestRestart('switch')
    expect(launchedModes).toEqual(['default', 'uploaded', 'default'])
    expect(targets.at(-1)).toEqual(target(4403))
  })

  test('coalesces duplicate events and recovers one unexpected active-child exit', async () => {
    const selection = { mode: 'default', layerPath: '' }
    const children: FakeChild[] = []
    let launches = 0
    const lifecycle = createDevThemeLifecycle({
      readSelection: () => selection,
      launchChild: () => {
        launches += 1
        const child = new FakeChild(500 + launches)
        children.push(child)
        return { child, ready: Promise.resolve(target(4500 + launches)) }
      },
      stopChild: async (child) => { child.emit('exit', null, 'SIGTERM') },
      setTarget: () => {},
      logger: { error: () => {} },
      recoveryDelayMs: 0,
    })
    await lifecycle.requestRestart('startup')
    await lifecycle.requestRestart('duplicate-1')
    await lifecycle.requestRestart('duplicate-2')
    expect(launches).toBe(1)
    children[0].emit('exit', 1, null)
    await waitFor(() => launches === 2)
    expect(launches).toBe(2)
    await lifecycle.shutdown()
    children[1].emit('exit', null, 'SIGTERM')
    await new Promise((resolve) => setTimeout(resolve, 1))
    expect(launches).toBe(2)
  })
})
```

- [ ] **Step 2: Run the coordinator tests and verify RED**

Run:

```bash
cd apps/web && bun test tests/devThemeLifecycle.test.ts
```

Expected: FAIL because `createDevThemeLifecycle` is not exported.

- [ ] **Step 3: Implement the serial coordinator**

Append this export to `apps/web/scripts/dev-theme-lifecycle.mjs`:

```js
export function createDevThemeLifecycle({
  readSelection,
  launchChild,
  stopChild,
  setTarget,
  logger = console,
  onFatal = (error) => logger.error('[sforum-dev-runtime] recovery failed:', error.message),
  onActive = () => {},
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
  let shuttingDown = false
  function installActive(started, selection, reason) {
    const installedChild = started.child
    activeChild = installedChild
    activeSelection = selection
    lastHealthySelection = selection
    setTarget(started.target)
    onActive(selection, reason)
    installedChild.once('exit', (code, signal) => {
      if (activeChild !== installedChild) return
      activeChild = null
      activeSelection = null
      setTarget(null)
      if (shuttingDown) return
      logger.error(`[sforum-dev-runtime] nuxt dev exited code=${code ?? ''} signal=${signal ?? ''}`)
      recoveryTimer = setTimeoutFn(() => {
        recoveryTimer = null
        void requestRestart('nuxt-dev-restart').catch(onFatal)
      }, recoveryDelayMs)
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
      await stopChild(launch.child)
      if (shuttingDown) return null
      throw error
    } finally {
      if (startingChild === launch.child) startingChild = null
    }
  }
  async function runRestartLoop(initialReason) {
    let reason = initialReason
    while (restartRequested && !shuttingDown) {
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
    startingChild = null
    activeChild = null
    activeSelection = null
    await Promise.allSettled(children.map((child) => stopChild(child)))
    if (restartPromise) await restartPromise.catch(() => {})
  }

  return { requestRestart, shutdown }
}
```

- [ ] **Step 4: Run the lifecycle tests and verify GREEN**

Run:

```bash
cd apps/web && bun test tests/devThemeLifecycle.test.ts
```

Expected: 8 PASS, 0 FAIL. All asynchronous assertions use `waitFor`; do not replace it with arbitrary sleeps.

- [ ] **Step 5: Commit Task 2**

Run `git add apps/web/scripts/dev-theme-lifecycle.mjs apps/web/tests/devThemeLifecycle.test.ts` and `git commit -m "fix: serialize local theme dev restarts"`.

### Task 3: Wire The Local Supervisor To The Coordinator

**Files:**
- Modify: `apps/web/scripts/dev-theme-runtime.mjs:1-286`
- Modify: `apps/web/scripts/theme-proxy.mjs:1-8`
- Modify: `apps/web/tests/devRuntimeStartup.test.ts`
- Modify: `tests/validate-theme-runtime.js`

- [ ] **Step 1: Write failing executable-contract tests**

Extend `apps/web/tests/devRuntimeStartup.test.ts`:

```ts
  test('dev supervisor uses serial lifecycle instead of the production blue-green helper', () => {
    const runtime = readFileSync(new URL('../scripts/dev-theme-runtime.mjs', import.meta.url), 'utf8')
    const lifecycle = readFileSync(new URL('../scripts/dev-theme-lifecycle.mjs', import.meta.url), 'utf8')

    expect(runtime).toContain('createDevThemeLifecycle')
    expect(runtime).toContain('stopProcessGroup')
    expect(runtime).not.toContain('replaceTarget')
    expect(lifecycle).toContain("signalGroup(pid, 'SIGTERM')")
    expect(lifecycle).toContain("signalGroup(pid, 'SIGKILL')")

    const startup = runtime.indexOf("await lifecycle.requestRestart('startup')")
    const listen = runtime.indexOf('await proxy.listen()')
    expect(startup).toBeGreaterThan(-1)
    expect(listen).toBeGreaterThan(startup)
  })
```

Update `tests/validate-theme-runtime.js` so it reads the new lifecycle module and asserts the local/production split:

```js
const devLifecycleScript = fs.readFileSync(
  path.join(root, 'apps/web/scripts/dev-theme-lifecycle.mjs'),
  'utf8'
)

assertIncludes(devRuntimeScript, 'createDevThemeLifecycle', 'dev supervisor must use the serial lifecycle coordinator')
assertIncludes(devRuntimeScript, 'stopProcessGroup', 'dev supervisor must stop the old process group before replacement')
if (devRuntimeScript.includes('replaceTarget')) {
  throw new Error('dev supervisor must not start parallel Nuxt candidates through replaceTarget')
}
assertIncludes(devLifecycleScript, "mode === 'default'", 'dev lifecycle must represent default mode explicitly')
assertIncludes(devLifecycleScript, 'restartRequested', 'dev lifecycle must coalesce current.json changes')
assertIncludes(devLifecycleScript, "signalGroup(pid, 'SIGKILL')", 'dev lifecycle must bound process-group shutdown')
```

Move the old `mode === 'default'` assertion from `devRuntimeScript` to `devLifecycleScript`; keep all production `runtimeScript` assertions unchanged.

- [ ] **Step 2: Run the executable-contract tests and verify RED**

Run:

```bash
cd apps/web && bun test tests/devRuntimeStartup.test.ts
node tests/validate-theme-runtime.js
```

Expected: FAIL because `dev-theme-runtime.mjs` still imports and calls `replaceTarget`.

- [ ] **Step 3: Replace the blue-green local wiring**

In `apps/web/scripts/dev-theme-runtime.mjs`:

1. Replace the `replaceTarget` import and local state variables with:

```js
import {
  createDevThemeLifecycle,
  readThemeSelection,
  stopProcessGroup,
} from './dev-theme-lifecycle.mjs'

const repoRoot = path.resolve(process.cwd(), '../../')
let restartTimer = null
let watcher = null
let lifecycle = null
let exiting = false
```

2. Replace `readLayerPath`, `stopChild`, `switchTo`, and `startDev` with a child launcher that returns the child immediately and a separate readiness promise:

```js
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
  let rejectPort
  const portPromise = new Promise((resolve, reject) => {
    resolvePort = resolve
    rejectPort = reject
  })
  const onExit = (code, signal) => rejectPort(
    new Error(`nuxt dev exited before ready (code=${code}, signal=${signal})`),
  )
  const onError = (error) => rejectPort(error)
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
  const ready = (async () => {
    const { host, port } = await withTimeout(portPromise, healthTimeoutMs)
    await healthCheckTcp(candidate, host, port, { timeoutMs: healthTimeoutMs })
    return candidate._target
  })().finally(() => {
    candidate.removeListener('exit', onExit)
    candidate.removeListener('error', onError)
  })

  return { child: candidate, ready }
}
```

3. Create lifecycle and watcher helpers:

```js
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
```

4. Replace signal/fatal/main handling with identity-safe shutdown:

```js
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
```

Update the file header comments to state that production is blue-green but local theme changes intentionally stop and restart one Nuxt process because Nuxt dev locks its build directory and HMR resources.

Update only the opening comments in `apps/web/scripts/theme-proxy.mjs`: describe `runtime.mjs` as the blue-green caller and `dev-theme-runtime.mjs` as the serial caller that reuses the same HTTP/WebSocket forwarding and health-check primitives. Do not change `replaceTarget` or any proxy behavior because production still depends on it.

- [ ] **Step 4: Run focused runtime tests and syntax checks**

Run:

```bash
cd apps/web && bun test tests/devThemeLifecycle.test.ts tests/devRuntimeStartup.test.ts tests/themeProxy.test.ts
node --check apps/web/scripts/dev-theme-lifecycle.mjs
node --check apps/web/scripts/dev-theme-runtime.mjs
node tests/validate-theme-runtime.js
```

Expected: all Bun tests PASS, both syntax checks exit 0, and the validator prints `Theme runtime validation passed.`

- [ ] **Step 5: Commit Task 3**

Run `git add apps/web/scripts/dev-theme-runtime.mjs apps/web/scripts/theme-proxy.mjs apps/web/tests/devRuntimeStartup.test.ts tests/validate-theme-runtime.js` and `git commit -m "fix: restart local Nuxt theme runtime serially"`.

### Task 4: Align Project Knowledge

**Files:**
- Modify: `knowledge/index.md:287-294`
- Modify: `knowledge/modules/frontend.md:88-106`
- Modify: `knowledge/modules/extensions.md:152-170`
- Create: `knowledge/sessions/2026-07-10-dev-theme-runtime-serial-restart.md`

- [ ] **Step 1: Update the module and index notes**

Record these exact architectural facts:

```text
Production runtime.mjs keeps blue-green Nitro switching and preserves the old
server when a candidate fails. Local dev-theme-runtime.mjs intentionally owns
one Nuxt dev process: current.json changes clear the proxy target, stop and
wait for the old process group, then start the latest layer. This local switch
has a brief development-only outage because parallel Nuxt dev instances share
lock, generated output, cache, and HMR resources.
```

Remove the claim in `knowledge/modules/extensions.md` that both production and
local development are zero-downtime. Keep `current.json` as the single shared
selection signal.

- [ ] **Step 2: Create the session handoff**

Create `knowledge/sessions/2026-07-10-dev-theme-runtime-serial-restart.md`:

```markdown
# 2026-07-10 Dev Theme Runtime Serial Restart

## Changed

- Replaced local blue-green Nuxt dev switching with one serial process lifecycle.
- Added explicit selection state, process-group waiting, latest-change convergence,
  identity-safe exits, crash recovery, and last-known-good rollback.
- Kept production Nitro blue-green switching and the current.json contract unchanged.

## Root Cause

- Nuxt 4.4.8 rejected the parallel candidate with
  `Another Nuxt dev server is already running`; bypassing the lock would still
  share generated files, caches, and HMR ports.

## Decisions

- Local theme activation may have a short Nuxt restart outage.
- Do not set NUXT_IGNORE_LOCK and do not create parallel local build slots.

## Verification

- `cd apps/web && bun test tests/devThemeLifecycle.test.ts tests/devRuntimeStartup.test.ts tests/themeProxy.test.ts`
- `node tests/validate-theme-runtime.js`
- `cd apps/web && bun run typecheck`
- Isolated default -> uploaded -> default smoke on an alternate port.
- `./scripts/test.sh`
```

- [ ] **Step 3: Commit Task 4**

Run `git add knowledge/index.md knowledge/modules/frontend.md knowledge/modules/extensions.md knowledge/sessions/2026-07-10-dev-theme-runtime-serial-restart.md` and `git commit -m "docs: record local theme runtime restart semantics"`.

### Task 5: Real Nuxt Smoke Test And Full Verification

**Files:**
- Verify only; do not modify the user's running port 3000 process.

- [ ] **Step 1: Run all focused automated tests fresh**

```bash
cd apps/web && bun test tests/devThemeLifecycle.test.ts tests/devRuntimeStartup.test.ts tests/themeProxy.test.ts
node tests/validate-theme-runtime.js
```

Expected: 0 failures and `Theme runtime validation passed.`

- [ ] **Step 2: Run frontend-wide tests and typecheck**

```bash
cd apps/web && bun test
cd apps/web && bun run typecheck
```

Expected: 0 failures and typecheck exit 0.

- [ ] **Step 3: Start an isolated real-Nuxt supervisor**

Confirm port 4317 is unused, create a temporary default selection atomically, and start a second supervisor with an isolated build directory. Disable Vite HMR only in this smoke instance to avoid competing with the user's HMR socket:

```bash
lsof -nP -iTCP:4317 -sTCP:LISTEN
mkdir -p /private/tmp/sforum-theme-serial-smoke
node -e 'const fs=require("node:fs");const root="/private/tmp/sforum-theme-serial-smoke";const value={extensionId:"sforum.default-theme",mode:"default",activatedAt:new Date().toISOString()};fs.writeFileSync(root+"/current.json.tmp",JSON.stringify(value,null,2));fs.renameSync(root+"/current.json.tmp",root+"/current.json")'
cd apps/web && SFORUM_THEME_RELEASE_ROOT=/private/tmp/sforum-theme-serial-smoke NUXT_BUILD_DIR=.nuxt/.serial-smoke NUXI_DISABLE_VITE_HMR=1 PORT=4317 WEB_PORT=4317 HOST=127.0.0.1 node --env-file=../../.env scripts/dev-theme-runtime.mjs
```

Expected: `http://127.0.0.1:4317/` becomes ready and port 3000 remains owned by the user's process.

- [ ] **Step 4: Switch default -> uploaded -> default atomically**

In a separate shell, write the uploaded selection:

```bash
node -e 'const fs=require("node:fs");const root="/private/tmp/sforum-theme-serial-smoke";const value={extensionId:"sforum.signal-garden",mode:"uploaded",layerPath:"/Users/inkedus/Code/SForum/extensions/dev/themes/sforum-signal-garden/layer",activatedAt:new Date().toISOString()};fs.writeFileSync(root+"/current.json.tmp",JSON.stringify(value,null,2));fs.renameSync(root+"/current.json.tmp",root+"/current.json")'
```

Wait for the uploaded `switched nuxt dev` log, verify the response with `curl -fsS http://127.0.0.1:4317/ -o /private/tmp/sforum-theme-uploaded.html`, and confirm the Nuxt lock error is absent. Then restore default atomically:

```bash
node -e 'const fs=require("node:fs");const root="/private/tmp/sforum-theme-serial-smoke";const value={extensionId:"sforum.default-theme",mode:"default",activatedAt:new Date().toISOString()};fs.writeFileSync(root+"/current.json.tmp",JSON.stringify(value,null,2));fs.renameSync(root+"/current.json.tmp",root+"/current.json")'
```

Wait for the second `switched nuxt dev` log, then run `curl -fsS http://127.0.0.1:4317/ -o /private/tmp/sforum-theme-default.html`.

Expected: both transitions stop before spawning; the final default response is HTTP 200. Stop only the 4317 supervisor with its terminal `Ctrl-C`, then confirm port 3000 still belongs to the user's process.

- [ ] **Step 5: Run the complete repository gate**

```bash
./scripts/test.sh
```

Expected: all Go tests, OpenAPI validation, Nuxt typecheck, and repository validators pass.

- [ ] **Step 6: Review the final diff against the approved spec**

```bash
git diff 838db76..HEAD --check
git diff 838db76..HEAD -- apps/web/scripts apps/web/tests tests/validate-theme-runtime.js knowledge
git status --short
```

Confirm every spec requirement has evidence: serial stop-before-spawn, explicit default state, latest-selection convergence, stale-exit protection, last-known-good rollback, crash recovery, unchanged production runtime, knowledge updates, and an untouched port 3000 process.

## Self-Review

- Spec coverage: Tasks 1-3 cover explicit state, process-group waiting, serial restart, latest-selection convergence, identity-safe exits, rollback, crash recovery, startup health, and shutdown. Task 4 covers all required knowledge changes. Task 5 covers real Nuxt behavior and all automated gates.
- Placeholder scan: no placeholder markers or open-ended error-handling steps remain.
- Type consistency: every coordinator caller uses `{ mode, layerPath }`; `launchChild` always returns `{ child, ready }`; `ready` resolves a proxy target; `stopChild` always receives the child object; `requestRestart` and `shutdown` are the only public lifecycle commands.
