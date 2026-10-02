#!/usr/bin/env node

import { basename } from 'node:path'

import { publishSDKPackages } from './publish-web-sdks.mjs'

function fail(message) {
  throw new Error(`publish-web-sdks_test: ${message}`)
}

const packages = [
  { name: '@sforum/admin-sdk', version: '1.0.0', filename: 'admin.tgz', integrity: 'sha512-admin' },
  { name: '@sforum/plugin-ui', version: '1.0.0', filename: 'ui.tgz', integrity: 'sha512-ui' }
]

// 本地打包产物解包后的成员摘要。
const localMembers = {
  'admin.tgz': ['package/LICENSE:aaaa', 'package/package.json:bbbb', 'package/src/index.ts:cccc'],
  'ui.tgz': ['package/LICENSE:dddd', 'package/package.json:eeee', 'package/src/index.ts:ffff']
}

function published(integrity, members) {
  return { integrity, members }
}

function runCase(remote) {
  const publishedArchives = []
  const logs = []
  const npmClient = {
    readRemoteIntegrity(name) {
      return remote[name] ? remote[name].integrity : null
    },
    readPublishedMembers(name) {
      return remote[name] ? remote[name].members : null
    },
    readLocalMembers(archive) {
      return localMembers[basename(archive)]
    },
    publishArchive(archive) {
      publishedArchives.push(archive)
    }
  }
  let error = null
  try {
    publishSDKPackages('/tmp/sdk-packages', packages, npmClient, (line) => logs.push(line))
  } catch (caught) {
    error = caught
  }
  return { publishedArchives, logs, error }
}

// 同内容、不同工具链压缩出的 tarball 字节不同（交互式首发 vs CI 打包）：必须跳过而不是失败。
const repacked = runCase({
  '@sforum/admin-sdk': published('sha512-other-toolchain', localMembers['admin.tgz']),
  '@sforum/plugin-ui': published('sha512-other-toolchain-ui', localMembers['ui.tgz'])
})
if (repacked.error || repacked.publishedArchives.length !== 0 || repacked.logs.length !== 2) {
  fail('identical content under different tarball bytes must be skipped idempotently')
}
if (!repacked.logs.every((line) => line.includes('identical content'))) {
  fail('idempotent skip must report identical content')
}

// 版本不存在：两个包都要发布。
const missing = runCase({ '@sforum/admin-sdk': null, '@sforum/plugin-ui': null })
if (missing.error || missing.publishedArchives.length !== 2 || !missing.publishedArchives.every((file) => file.endsWith('.tgz'))) {
  fail('missing versions were not both published')
}

// 版本已存在但内容真的变了：fail closed，且不得触发发布。
const changed = runCase({
  '@sforum/admin-sdk': published('sha512-other-toolchain', ['package/LICENSE:aaaa', 'package/package.json:bbbb', 'package/src/index.ts:CHANGED']),
  '@sforum/plugin-ui': null
})
if (!changed.error?.message.includes('bump the SDK version') || !changed.error.message.includes('already exists with different content')) {
  fail('different-content retry did not fail closed before publication')
}
if (changed.publishedArchives.length !== 0) fail('different content must not publish under an existing version')

// 成员集合多一个或少一个同样算内容变更。
const extraMember = runCase({
  '@sforum/admin-sdk': published('sha512-other-toolchain', [...localMembers['admin.tgz'], 'package/src/extra.ts:0000']),
  '@sforum/plugin-ui': published('sha512-other-toolchain-ui', localMembers['ui.tgz'])
})
if (!extraMember.error?.message.includes('bump the SDK version') || extraMember.publishedArchives.length !== 0) {
  fail('member-set drift under an existing version did not fail closed')
}

// 已发布内容一致 + 另一版本缺失：只发布缺失的那个。
const mixed = runCase({
  '@sforum/admin-sdk': published('sha512-other-toolchain', localMembers['admin.tgz']),
  '@sforum/plugin-ui': null
})
if (mixed.error || mixed.publishedArchives.length !== 1 || !mixed.publishedArchives[0].endsWith('ui.tgz')) {
  fail('mixed retry must skip the identical version and publish only the missing one')
}

console.log('publish-web-sdks_test: all checks passed')
