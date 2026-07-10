import { afterEach, describe, expect, test } from 'bun:test'
import fs from 'node:fs'
import os from 'node:os'
import path from 'node:path'

import {
  digestArtifactTree,
  readDesiredRelease,
  verifyReleaseArtifact,
  watchableReleaseFile,
  writeActiveAcknowledgement,
  writeFailureAcknowledgement
} from '../scripts/web-release-contract.mjs'

const roots: string[] = []
function tempRoot() {
  const root = fs.mkdtempSync(path.join(os.tmpdir(), 'sforum-web-release-'))
  roots.push(root)
  return root
}
afterEach(() => {
  for (const root of roots.splice(0)) fs.rmSync(root, { recursive: true, force: true })
})

describe('web release file contract', () => {
  test('reads the new desired pointer and resolves relative paths', () => {
    const root = tempRoot()
    fs.writeFileSync(path.join(root, 'current.json'), JSON.stringify({
      schemaVersion: 1,
      releaseId: 42,
      compositionHash: 'composition',
      artifactPath: 'releases/42/artifact',
      artifactDigest: 'digest',
      serverEntry: 'releases/42/artifact/server/index.mjs',
      themeId: 'demo.theme',
      themeVersion: '1.0.0',
      reloadMode: 'prompt'
    }))

    expect(readDesiredRelease({ releaseRoot: root, fallback: { serverEntry: '/fallback/server/index.mjs' } })).toMatchObject({
      kind: 'release',
      releaseId: 42,
      artifactPath: path.join(root, 'releases/42/artifact'),
      serverEntry: path.join(root, 'releases/42/artifact/server/index.mjs')
    })
  })

  test('keeps legacy uploaded and default theme pointers compatible', () => {
    const root = tempRoot()
    fs.writeFileSync(path.join(root, 'current.json'), JSON.stringify({ mode: 'uploaded', server: '/legacy/server.mjs', layerPath: '/legacy/layer' }))
    expect(readDesiredRelease({ releaseRoot: root, fallback: { serverEntry: '/fallback.mjs' } })).toMatchObject({
      kind: 'legacy', serverEntry: '/legacy/server.mjs', themeLayer: '/legacy/layer'
    })
    fs.writeFileSync(path.join(root, 'current.json'), JSON.stringify({ mode: 'default' }))
    expect(readDesiredRelease({ releaseRoot: root, fallback: { serverEntry: '/fallback.mjs' } })).toMatchObject({
      kind: 'fallback', serverEntry: '/fallback.mjs'
    })
  })

  test('verifies release manifest identity and the complete artifact digest', async () => {
    const root = tempRoot()
    const releaseDir = path.join(root, 'releases', '7')
    const artifact = path.join(releaseDir, 'artifact')
    const server = path.join(artifact, 'server', 'index.mjs')
    fs.mkdirSync(path.dirname(server), { recursive: true })
    fs.writeFileSync(server, 'export {}\n', { mode: 0o644 })
    fs.chmodSync(server, 0o644)
    const digest = await digestArtifactTree(artifact)
    expect(digest).toBe('4a8de05aa42aabf8deb3312c215f18c3beab4875eb35e2101cf784403f3d3c2c')
    const manifest = {
      schemaVersion: 1, releaseId: 7, compositionHash: 'hash', artifactPath: artifact,
      artifactDigest: digest, serverEntry: server, themeId: 'demo.theme', themeVersion: '1.0.0',
      themeLayer: '/theme/layer', devInput: '/dev-input', registryRoot: '/registry', reloadMode: 'prompt'
    }
    fs.writeFileSync(path.join(releaseDir, 'release.json'), JSON.stringify(manifest))
    const desired = { kind: 'release', ...manifest }

    await expect(verifyReleaseArtifact(desired)).resolves.toMatchObject({ releaseId: 7, themeLayer: '/theme/layer' })
    fs.writeFileSync(server, 'tampered')
    await expect(verifyReleaseArtifact(desired)).rejects.toThrow('digest')
  })

  test('writes active and failure acknowledgements atomically', async () => {
    const root = tempRoot()
    await writeActiveAcknowledgement(root, { releaseId: 9, compositionHash: 'hash' })
    await writeFailureAcknowledgement(root, { releaseId: 10, reason: 'web_release.start_failed', message: 'candidate failed\nwith details' })

    expect(JSON.parse(fs.readFileSync(path.join(root, 'active.json'), 'utf8'))).toMatchObject({ releaseId: 9 })
    expect(JSON.parse(fs.readFileSync(path.join(root, 'failures', '10.json'), 'utf8'))).toMatchObject({
      releaseId: 10, reason: 'web_release.start_failed', message: 'candidate failed with details'
    })
    expect(fs.existsSync(path.join(root, 'active.json.tmp'))).toBe(false)
  })

  test('watches only desired pointer writes', () => {
    expect(watchableReleaseFile('current.json')).toBe(true)
    expect(watchableReleaseFile('current.json.tmp')).toBe(true)
    expect(watchableReleaseFile('active.json')).toBe(false)
    expect(watchableReleaseFile('failures')).toBe(false)
  })
})
