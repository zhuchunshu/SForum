import { describe, expect, test } from 'bun:test'
import { readFileSync } from 'node:fs'

describe('dev runtime startup', () => {
  test('dev script loads the root env before starting the public proxy', () => {
    const packageJson = JSON.parse(readFileSync(new URL('../package.json', import.meta.url), 'utf8'))

    expect(packageJson.scripts.dev).toContain('--env-file=../../.env')
    expect(packageJson.scripts.dev).toContain('scripts/dev-theme-runtime.mjs')
  })
})
