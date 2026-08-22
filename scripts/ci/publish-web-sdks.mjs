#!/usr/bin/env node

import { execFileSync, spawnSync } from 'node:child_process'
import { existsSync, mkdtempSync, rmSync } from 'node:fs'
import { tmpdir } from 'node:os'
import { dirname, join, resolve } from 'node:path'
import { fileURLToPath } from 'node:url'

import { npmRegistry, packageContentIntegrity, packSDKPackages, readPackedManifest } from './web-sdk-packages.mjs'

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

function readRemoteContentIntegrity(name, version) {
  const output = mkdtempSync(join(tmpdir(), 'sforum-remote-web-sdk-'))
  try {
    const result = spawnSync('npm', [
      'pack', `${name}@${version}`, '--json', '--ignore-scripts',
      '--pack-destination', output, `--registry=${npmRegistry}`
    ], { encoding: 'utf8' })
    if (result.status !== 0) {
      throw new Error(`npm pack failed for ${name}@${version}: ${`${result.stdout}\n${result.stderr}`.trim()}`)
    }
    const metadata = JSON.parse(result.stdout)
    if (!Array.isArray(metadata) || metadata.length !== 1 ||
        metadata[0].name !== name || metadata[0].version !== version) {
      throw new Error(`npm pack returned an unexpected package for ${name}@${version}`)
    }
    const archive = resolve(output, metadata[0].filename)
    if (dirname(archive) !== output || !existsSync(archive)) {
      throw new Error(`npm pack returned an invalid archive path for ${name}@${version}`)
    }
    return packageContentIntegrity(archive, metadata[0].files, name)
  } finally {
    rmSync(output, { recursive: true, force: true })
  }
}

function publishArchive(archive) {
  execFileSync('npm', ['publish', archive, '--provenance', '--access', 'public', `--registry=${npmRegistry}`], {
    stdio: 'inherit'
  })
}

export function publishSDKPackages(root, packages, npmClient = { readRemoteIntegrity, readRemoteContentIntegrity, publishArchive }, log = console.log) {
  for (const sdk of packages) {
    const remoteIntegrity = npmClient.readRemoteIntegrity(sdk.name, sdk.version)
    if (remoteIntegrity === sdk.integrity) {
      log(`${sdk.name}@${sdk.version} already contains the exact SDK artifact`)
      continue
    }
    if (remoteIntegrity !== null) {
      const remoteContentIntegrity = npmClient.readRemoteContentIntegrity(sdk.name, sdk.version)
      if (remoteContentIntegrity === sdk.contentIntegrity) {
        log(`${sdk.name}@${sdk.version} already contains the exact SDK content`)
        continue
      }
      throw new Error(`${sdk.name}@${sdk.version} already exists with different content; bump the SDK version before releasing`)
    }
    npmClient.publishArchive(join(root, sdk.filename))
    log(`published ${sdk.name}@${sdk.version}`)
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
