import { afterEach, describe, expect, test } from 'bun:test'
import fs from 'node:fs'
import os from 'node:os'
import path from 'node:path'

import { resolvePlainReleaseRoot, startPlainReleaseAck } from '../scripts/dev-plain-release-ack.mjs'

function tempRoot() {
  return fs.mkdtempSync(path.join(os.tmpdir(), 'sforum-plain-ack-'))
}

describe('dev plain release ack', () => {
  const cleanups: Array<() => void> = []

  afterEach(() => {
    while (cleanups.length) {
      cleanups.pop()?.()
    }
  })

  test('resolvePlainReleaseRoot prefers env then storage/theme-releases', () => {
    expect(resolvePlainReleaseRoot({
      cwd: '/repo/apps/web',
      env: { SFORUM_WEB_RELEASE_ROOT: '/custom/releases' },
    })).toBe('/custom/releases')
    expect(resolvePlainReleaseRoot({
      cwd: '/repo/apps/web',
      env: {},
    })).toBe(path.resolve('/repo/storage/theme-releases'))
  })

  test('acknowledges desired web release without requiring theme restart', async () => {
    const root = tempRoot()
    cleanups.push(() => fs.rmSync(root, { recursive: true, force: true }))
    const logs: string[] = []

    fs.writeFileSync(path.join(root, 'current.json'), `${JSON.stringify({
      schemaVersion: 1,
      releaseId: 42,
      compositionHash: 'composition-42',
      artifactPath: '/artifact',
      artifactDigest: 'digest-42',
      serverEntry: '/artifact/server/index.mjs',
      themeId: 'sforum.default-theme',
      themeVersion: '1.0.0',
      reloadMode: 'prompt',
    }, null, 2)}\n`)

    const ack = startPlainReleaseAck({
      releaseRoot: root,
      log: (message) => logs.push(message),
      error: () => {},
    })
    cleanups.push(() => ack.close())

    await ack.sync('test')

    const active = JSON.parse(fs.readFileSync(path.join(root, 'active.json'), 'utf8'))
    expect(active).toMatchObject({
      releaseId: 42,
      compositionHash: 'composition-42',
      artifactDigest: 'digest-42',
      themeId: 'sforum.default-theme',
    })
    expect(logs.some(line => line.includes('#42'))).toBe(true)
  })

  test('ignores non-release current pointers', async () => {
    const root = tempRoot()
    cleanups.push(() => fs.rmSync(root, { recursive: true, force: true }))

    fs.writeFileSync(path.join(root, 'current.json'), `${JSON.stringify({ mode: 'default' }, null, 2)}\n`)

    const ack = startPlainReleaseAck({
      releaseRoot: root,
      log: () => {},
      error: () => {},
    })
    cleanups.push(() => ack.close())
    await ack.sync('test')

    expect(fs.existsSync(path.join(root, 'active.json'))).toBe(false)
  })
})
