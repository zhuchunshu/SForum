#!/usr/bin/env node

import { mkdtempSync, rmSync } from 'node:fs'
import { tmpdir } from 'node:os'
import { join } from 'node:path'

import { packSDKPackages, verifyPackedSDKConsumer } from './web-sdk-packages.mjs'

const output = mkdtempSync(join(tmpdir(), 'sforum-web-sdks-'))
try {
  const packed = packSDKPackages(output)
  verifyPackedSDKConsumer(output, packed.packages)
  for (const sdk of packed.packages) console.log(`verified ${sdk.name}@${sdk.version} ${sdk.integrity}`)
} finally {
  rmSync(output, { recursive: true, force: true })
}
