import { afterEach, describe, expect, test } from 'bun:test'
import { EventEmitter } from 'node:events'
import fs from 'node:fs'
import os from 'node:os'
import path from 'node:path'

import {
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

describe('createDevThemeLifecycle', () => {
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
