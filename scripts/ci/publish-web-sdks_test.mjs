#!/usr/bin/env node

import { publishSDKPackages } from './publish-web-sdks.mjs'

function fail(message) {
  throw new Error(`publish-web-sdks_test: ${message}`)
}

const packages = [
  { name: '@sforum/admin-sdk', version: '1.0.0', filename: 'admin.tgz', integrity: 'sha512-admin' },
  { name: '@sforum/plugin-ui', version: '1.0.0', filename: 'ui.tgz', integrity: 'sha512-ui' }
]

function runCase(remote) {
  const published = []
  const logs = []
  const npmClient = {
    readRemoteIntegrity(name) {
      return remote[name]
    },
    publishArchive(archive) {
      published.push(archive)
    }
  }
  let error = null
  try {
    publishSDKPackages('/tmp/sdk-packages', packages, npmClient, (line) => logs.push(line))
  } catch (caught) {
    error = caught
  }
  return { published, logs, error }
}

const same = runCase({ '@sforum/admin-sdk': 'sha512-admin', '@sforum/plugin-ui': 'sha512-ui' })
if (same.error || same.published.length !== 0 || same.logs.length !== 2) {
  fail('same-content retry was not idempotent')
}

const missing = runCase({ '@sforum/admin-sdk': null, '@sforum/plugin-ui': null })
if (missing.error || missing.published.length !== 2 || !missing.published.every((file) => file.endsWith('.tgz'))) {
  fail('missing versions were not both published')
}

const mismatch = runCase({ '@sforum/admin-sdk': 'sha512-different', '@sforum/plugin-ui': null })
if (!mismatch.error?.message.includes('bump the SDK version') || mismatch.published.length !== 0) {
  fail('different-content retry did not fail closed before publication')
}

console.log('publish-web-sdks_test: all checks passed')
