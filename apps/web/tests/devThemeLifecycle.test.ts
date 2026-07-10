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
