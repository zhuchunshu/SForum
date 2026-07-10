import { afterEach, describe, expect, test } from 'bun:test'
import { EventEmitter } from 'node:events'
import fs from 'node:fs'
import os from 'node:os'
import path from 'node:path'

import {
  clearNuxtRouteCache,
  createDevThemeLifecycle,
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

afterEach(() => {
  for (const root of tempRoots.splice(0)) {
    fs.rmSync(root, { recursive: true, force: true })
  }
})

describe('clearNuxtRouteCache', () => {
  test('removes Nitro route responses while preserving other Nuxt caches', () => {
    const buildDir = path.join(tempRoot(), '.nuxt')
    const routesDir = path.join(buildDir, 'cache', 'nitro', 'routes')
    const staleRoute = path.join(routesDir, 'stale.json')
    const viteMarker = path.join(buildDir, 'cache', 'vite', 'keep.txt')
    fs.mkdirSync(routesDir, { recursive: true })
    fs.mkdirSync(path.dirname(viteMarker), { recursive: true })
    fs.writeFileSync(staleRoute, 'stale')
    fs.writeFileSync(viteMarker, 'keep')

    clearNuxtRouteCache(buildDir)

    expect(fs.existsSync(routesDir)).toBe(false)
    expect(fs.readFileSync(viteMarker, 'utf8')).toBe('keep')
  })
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

  test('derives release theme and registry inputs from immutable release.json', () => {
    const root = tempRoot()
    const releaseDir = path.join(root, 'releases', '12')
    const artifact = path.join(releaseDir, 'artifact')
    const serverEntry = path.join(artifact, 'server', 'index.mjs')
    fs.mkdirSync(path.dirname(serverEntry), { recursive: true })
    fs.writeFileSync(serverEntry, 'export {}')
    const desired = {
      schemaVersion: 1, releaseId: 12, compositionHash: 'composition',
      artifactPath: artifact, artifactDigest: 'artifact-digest', serverEntry,
      themeId: 'demo.theme', themeVersion: '1.0.0', reloadMode: 'prompt'
    }
    fs.writeFileSync(path.join(root, 'current.json'), JSON.stringify(desired))
    fs.writeFileSync(path.join(releaseDir, 'release.json'), JSON.stringify({
      ...desired,
      themeLayer: path.join(releaseDir, 'dev-input', 'theme', 'layer'),
      devInput: path.join(releaseDir, 'dev-input'),
      registryRoot: path.join(releaseDir, 'dev-input', 'registry')
    }))

    const selection = readThemeSelection(path.join(root, 'current.json'), { repoRoot: root })
    expect(selection).toMatchObject({
      mode: 'uploaded', releaseId: '12', compositionHash: 'composition',
      registryRoot: path.join(releaseDir, 'dev-input', 'registry')
    })
    expect(themeSelectionKey(selection)).toContain('12:composition')
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

  test('treats EPERM existence probes as alive while waiting for group exit', async () => {
    const signals: string[] = []
    let probes = 0
    let polls = 0

    await stopProcessGroup({ pid: 43 }, {
      graceMs: Number.POSITIVE_INFINITY,
      pollMs: 1,
      signalGroup: (_pid, signal) => { signals.push(signal) },
      groupExists: () => {
        probes += 1
        if (probes === 1) {
          throw Object.assign(new Error('operation not permitted'), { code: 'EPERM' })
        }
        return false
      },
      sleepFn: async () => { polls += 1 },
    })

    expect(probes).toBe(2)
    expect(polls).toBe(1)
    expect(signals).toEqual(['SIGTERM'])
  })
})

describe('createDevThemeLifecycle', () => {
  test('fails initial startup without retrying when the first child is unhealthy', async () => {
    const selection = { mode: 'default', layerPath: '' }
    const startupError = new Error('initial child unhealthy')
    const children: FakeChild[] = []
    const stopped: FakeChild[] = []
    const scheduled: Array<() => void> = []
    const lifecycle = createDevThemeLifecycle({
      readSelection: () => selection,
      launchChild: () => {
        const child = new FakeChild(90 + children.length)
        children.push(child)
        return { child, ready: Promise.reject(startupError) }
      },
      stopChild: async (child) => { stopped.push(child) },
      setTarget: () => {},
      logger: { error: () => {} },
      setTimeoutFn: (callback) => {
        scheduled.push(callback)
        return callback
      },
      clearTimeoutFn: () => {},
    })

    let rejected: unknown
    try {
      await lifecycle.requestRestart('startup')
    } catch (error) {
      rejected = error
    }

    expect(rejected).toBe(startupError)
    expect(children).toHaveLength(1)
    expect(stopped).toEqual(children)
    expect(scheduled).toHaveLength(0)
  })

  test('stops the active child before launching its replacement', async () => {
    let selection = { mode: 'default', layerPath: '' }
    const launches: Array<{
      child: FakeChild
      ready: ReturnType<typeof deferred<ReturnType<typeof target>>>
    }> = []
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
    launches[1].ready.resolve(target(4102))
    await switching
  })

  test('converges on default when default is requested during an uploaded start', async () => {
    let selection = { mode: 'default', layerPath: '' }
    const launches: Array<{
      selection: typeof selection
      child: FakeChild
      ready: ReturnType<typeof deferred<ReturnType<typeof target>>>
    }> = []
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
    const launches: Array<{
      child: FakeChild
      ready: ReturnType<typeof deferred<ReturnType<typeof target>>>
    }> = []
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

  test('rejects candidate cleanup failure without launching last-known-good rollback', async () => {
    let selection = { mode: 'default', layerPath: '' }
    const readinessError = new Error('candidate readiness failed')
    const cleanupError = new Error('candidate cleanup failed')
    const launches: Array<{ selection: typeof selection, child: FakeChild }> = []
    const scheduled: Array<() => void> = []
    const lifecycle = createDevThemeLifecycle({
      readSelection: () => selection,
      launchChild: (nextSelection) => {
        const child = new FakeChild(480 + launches.length)
        launches.push({ selection: nextSelection, child })
        return {
          child,
          ready: launches.length === 2
            ? Promise.reject(readinessError)
            : Promise.resolve(target(4480 + launches.length)),
        }
      },
      stopChild: async (child) => {
        if (child === launches[1]?.child) throw cleanupError
      },
      setTarget: () => {},
      logger: { error: () => {} },
      setTimeoutFn: (callback) => {
        scheduled.push(callback)
        return callback
      },
      clearTimeoutFn: () => {},
    })

    await lifecycle.requestRestart('startup')
    selection = { mode: 'uploaded', layerPath: '/tmp/broken-theme' }

    let rejected: unknown
    try {
      await lifecycle.requestRestart('switch')
    } catch (error) {
      rejected = error
    }

    expect(rejected).toBeInstanceOf(AggregateError)
    expect((rejected as AggregateError).errors).toEqual([readinessError, cleanupError])
    expect(launches.map((item) => item.selection.mode)).toEqual(['default', 'uploaded'])
    expect(scheduled).toHaveLength(0)
  })

  test('rejects when both the requested theme and last-known-good rollback fail', async () => {
    let selection = { mode: 'default', layerPath: '' }
    const requestedError = new Error('requested theme unhealthy')
    const rollbackError = new Error('rollback unhealthy')
    const launches: Array<{ selection: typeof selection, child: FakeChild }> = []
    const lifecycle = createDevThemeLifecycle({
      readSelection: () => selection,
      launchChild: (nextSelection) => {
        const child = new FakeChild(490 + launches.length)
        launches.push({ selection: nextSelection, child })
        const ready = launches.length === 1
          ? Promise.resolve(target(4491))
          : Promise.reject(launches.length === 2 ? requestedError : rollbackError)
        return { child, ready }
      },
      stopChild: async () => {},
      setTarget: () => {},
      logger: { error: () => {} },
    })

    await lifecycle.requestRestart('startup')
    selection = { mode: 'uploaded', layerPath: '/tmp/broken-theme' }

    let rejected: unknown
    try {
      await lifecycle.requestRestart('switch')
    } catch (error) {
      rejected = error
    }

    expect(rejected).toBeInstanceOf(AggregateError)
    expect((rejected as AggregateError).message).toBe(
      'requested theme and last-known-good theme both failed to start',
    )
    expect((rejected as AggregateError).errors).toEqual([requestedError, rollbackError])
    expect(launches.map((item) => item.selection.mode)).toEqual([
      'default',
      'uploaded',
      'default',
    ])
  })

  test('awaits unexpected active-child cleanup before launching recovery', async () => {
    const selection = { mode: 'default', layerPath: '' }
    const children: FakeChild[] = []
    const stopped: FakeChild[] = []
    const scheduled: Array<() => void> = []
    const stopGate = deferred<void>()
    const lifecycle = createDevThemeLifecycle({
      readSelection: () => selection,
      launchChild: () => {
        const child = new FakeChild(500 + children.length)
        children.push(child)
        return { child, ready: Promise.resolve(target(4500 + children.length)) }
      },
      stopChild: async (child) => {
        stopped.push(child)
        await stopGate.promise
      },
      setTarget: () => {},
      logger: { error: () => {} },
      recoveryDelayMs: 0,
      setTimeoutFn: (callback) => {
        scheduled.push(callback)
        return callback
      },
      clearTimeoutFn: () => {},
    })

    await lifecycle.requestRestart('startup')
    children[0].emit('exit', 1, null)
    await waitFor(() => stopped.length === 1)

    expect(stopped).toEqual([children[0]])
    expect(children).toHaveLength(1)
    expect(scheduled).toHaveLength(0)

    stopGate.resolve()
    await waitFor(() => scheduled.length === 1)
    expect(children).toHaveLength(1)

    scheduled[0]()
    await waitFor(() => children.length === 2)
    await lifecycle.shutdown()
  })

  test('serializes external restart requests behind in-flight crash cleanup', async () => {
    let selection = { mode: 'default', layerPath: '' }
    const launches: Array<{ selection: typeof selection, child: FakeChild }> = []
    const scheduled: Array<() => void> = []
    const stopGate = deferred<void>()
    let cleanupStarted = false
    const lifecycle = createDevThemeLifecycle({
      readSelection: () => selection,
      launchChild: (nextSelection) => {
        const child = new FakeChild(525 + launches.length)
        launches.push({ selection: nextSelection, child })
        return { child, ready: Promise.resolve(target(4525 + launches.length)) }
      },
      stopChild: async (child) => {
        if (child !== launches[0].child) return
        cleanupStarted = true
        await stopGate.promise
      },
      setTarget: () => {},
      logger: { error: () => {} },
      recoveryDelayMs: 0,
      setTimeoutFn: (callback) => {
        scheduled.push(callback)
        return callback
      },
      clearTimeoutFn: () => {},
    })

    await lifecycle.requestRestart('startup')
    launches[0].child.emit('exit', 1, null)
    await waitFor(() => cleanupStarted)

    selection = { mode: 'uploaded', layerPath: '/tmp/theme-after-crash' }
    const switching = lifecycle.requestRestart('current.json changed')
    await Promise.resolve()
    expect(launches).toHaveLength(1)
    expect(scheduled).toHaveLength(0)

    stopGate.resolve()
    await switching
    expect(launches.map((item) => item.selection.mode)).toEqual(['default', 'uploaded'])
    expect(scheduled).toHaveLength(1)

    scheduled[0]()
    await Promise.resolve()
    await Promise.resolve()
    expect(launches).toHaveLength(2)
    await lifecycle.shutdown()
  })

  test('reports crash cleanup failure and rejects a waiting restart without relaunching', async () => {
    const selection = { mode: 'default', layerPath: '' }
    const cleanupError = new Error('descendants survived leader exit')
    const children: FakeChild[] = []
    const fatalErrors: Error[] = []
    const scheduled: Array<() => void> = []
    const cleanupGate = deferred<void>()
    let cleanupStarted = false
    const lifecycle = createDevThemeLifecycle({
      readSelection: () => selection,
      launchChild: () => {
        const child = new FakeChild(550 + children.length)
        children.push(child)
        return { child, ready: Promise.resolve(target(4550 + children.length)) }
      },
      stopChild: async () => {
        cleanupStarted = true
        await cleanupGate.promise
      },
      setTarget: () => {},
      logger: { error: () => {} },
      onFatal: (error) => { fatalErrors.push(error) },
      recoveryDelayMs: 0,
      setTimeoutFn: (callback) => {
        scheduled.push(callback)
        return callback
      },
      clearTimeoutFn: () => {},
    })

    await lifecycle.requestRestart('startup')
    children[0].emit('exit', 1, null)
    await waitFor(() => cleanupStarted)

    const restarting = lifecycle.requestRestart('current.json changed')
    await Promise.resolve()
    expect(children).toHaveLength(1)
    expect(scheduled).toHaveLength(0)

    cleanupGate.reject(cleanupError)
    let rejected: unknown
    try {
      await restarting
    } catch (error) {
      rejected = error
    }
    await waitFor(() => fatalErrors.length === 1)
    await Promise.resolve()
    await Promise.resolve()

    expect(rejected).toBe(cleanupError)
    expect(fatalErrors).toEqual([cleanupError])
    expect(children).toHaveLength(1)
    expect(scheduled).toHaveLength(0)
    await lifecycle.shutdown()
  })

  test('shutdown awaits in-flight crash cleanup and prevents recovery launch', async () => {
    const selection = { mode: 'default', layerPath: '' }
    const children: FakeChild[] = []
    const stopGate = deferred<void>()
    let cleanupStarted = false
    let shutdownSettled = false
    const lifecycle = createDevThemeLifecycle({
      readSelection: () => selection,
      launchChild: () => {
        const child = new FakeChild(600 + children.length)
        children.push(child)
        return { child, ready: Promise.resolve(target(4600 + children.length)) }
      },
      stopChild: async () => {
        cleanupStarted = true
        await stopGate.promise
      },
      setTarget: () => {},
      logger: { error: () => {} },
      recoveryDelayMs: 0,
    })

    await lifecycle.requestRestart('startup')
    children[0].emit('exit', 1, null)
    await waitFor(() => cleanupStarted)

    const shutdown = lifecycle.shutdown().then(() => { shutdownSettled = true })
    await new Promise((resolve) => setTimeout(resolve, 1))
    expect(shutdownSettled).toBe(false)
    expect(children).toHaveLength(1)

    stopGate.resolve()
    await shutdown
    await new Promise((resolve) => setTimeout(resolve, 1))
    expect(children).toHaveLength(1)
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
