#!/usr/bin/env node

import { execFileSync, spawnSync } from 'node:child_process'
import { mkdtempSync, readdirSync, rmSync } from 'node:fs'
import { tmpdir } from 'node:os'
import { join, resolve } from 'node:path'
import { fileURLToPath } from 'node:url'

import { npmRegistry, packSDKPackages, readPackedManifest, readTarballMemberDigests } from './web-sdk-packages.mjs'

function npmVersionSupportsTrustedPublishing() {
  const version = execFileSync('npm', ['--version'], { encoding: 'utf8' }).trim()
  const match = /^(\d+)\.(\d+)\.(\d+)/.exec(version)
  if (!match || Number(match[1]) < 11 || (Number(match[1]) === 11 && Number(match[2]) < 5)) {
    throw new Error(`npm ${version} cannot use Trusted Publishing; npm 11.5.1 or newer is required`)
  }
  return version
}

function readRemoteIntegrity(name, version) {
  const result = spawnSync('npm', ['view', `${name}@${version}`, 'dist.integrity', '--json', `--registry=${npmRegistry}`], {
    encoding: 'utf8'
  })
  if (result.status === 0) {
    const integrity = JSON.parse(result.stdout)
    if (typeof integrity !== 'string' || !integrity.startsWith('sha512-')) {
      throw new Error(`npm returned an invalid integrity for ${name}@${version}`)
    }
    return integrity
  }
  const details = `${result.stdout}\n${result.stderr}`
  if (/E404|404 Not Found/i.test(details)) return null
  throw new Error(`npm view failed for ${name}@${version}: ${details.trim()}`)
}

function publishArchive(archive) {
  execFileSync('npm', ['publish', archive, '--provenance', '--access', 'public', `--registry=${npmRegistry}`], {
    stdio: 'inherit'
  })
}

// 已发布版本的内容指纹。必须比较解包后的成员摘要，而不是 tarball 完整性：
// 同一份 tar 载荷在不同 Node/npm 的 zlib 实现下会压出不同的字节，交互式首次
// 发布与 CI 打包的工具链并不一致，字节级比较会把「内容完全相同」误判成变更。
function readPublishedMembers(name, version) {
  const directory = mkdtempSync(join(tmpdir(), 'sforum-published-sdk-'))
  try {
    execFileSync(
      'npm',
      ['pack', `${name}@${version}`, '--ignore-scripts', '--pack-destination', directory, `--registry=${npmRegistry}`],
      { stdio: 'pipe' }
    )
    const archives = readdirSync(directory).filter((entry) => entry.endsWith('.tgz'))
    if (archives.length !== 1) {
      throw new Error(`npm pack fetched ${archives.length} archives for published ${name}@${version}`)
    }
    return readTarballMemberDigests(join(directory, archives[0]))
  } finally {
    rmSync(directory, { recursive: true, force: true })
  }
}

export const defaultNpmClient = {
  readRemoteIntegrity,
  readPublishedMembers,
  readLocalMembers: readTarballMemberDigests,
  publishArchive
}

function sameMembers(left, right) {
  return left.length === right.length && left.every((member, index) => member === right[index])
}

export function publishSDKPackages(root, packages, npmClient = defaultNpmClient, log = console.log) {
  for (const sdk of packages) {
    const remoteIntegrity = npmClient.readRemoteIntegrity(sdk.name, sdk.version)
    if (remoteIntegrity === null) {
      npmClient.publishArchive(join(root, sdk.filename))
      log(`published ${sdk.name}@${sdk.version}`)
      continue
    }
    const publishedMembers = npmClient.readPublishedMembers(sdk.name, sdk.version)
    if (sameMembers(publishedMembers, npmClient.readLocalMembers(join(root, sdk.filename)))) {
      log(`${sdk.name}@${sdk.version} already published with identical content (registry integrity ${remoteIntegrity}); skipping`)
      continue
    }
    throw new Error(
      `${sdk.name}@${sdk.version} already exists with different content; bump the SDK version before releasing ` +
        `(local ${sdk.integrity}, registry ${remoteIntegrity})`
    )
  }
}

function main() {
  const npmVersion = npmVersionSupportsTrustedPublishing()
  const output = mkdtempSync(join(tmpdir(), 'sforum-publish-web-sdks-'))
  try {
    const packed = packSDKPackages(output)
    const { root, manifest } = readPackedManifest(packed.manifestPath)
    console.log(`using npm ${npmVersion} with Trusted Publishing`)
    publishSDKPackages(root, manifest.packages)
  } finally {
    rmSync(output, { recursive: true, force: true })
  }
}

if (process.argv[1] && resolve(process.argv[1]) === fileURLToPath(import.meta.url)) main()
