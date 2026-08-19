#!/usr/bin/env node

import { resolve } from 'node:path'

import { packSDKPackages, verifyPackedSDKConsumer } from './web-sdk-packages.mjs'

if (process.argv.length !== 3) {
  console.error('usage: node scripts/ci/pack-web-sdks.mjs <output-directory>')
  process.exit(2)
}

const output = resolve(process.argv[2])
const packed = packSDKPackages(output)
verifyPackedSDKConsumer(output, packed.packages)
console.log(`wrote ${packed.manifestPath}`)
for (const sdk of packed.packages) console.log(`${sdk.name}@${sdk.version}\t${sdk.filename}\t${sdk.integrity}`)
