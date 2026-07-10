import { describe, expect, test } from 'bun:test'
import { readFileSync } from 'node:fs'

describe('dev runtime startup', () => {
  test('dev script loads the root env before starting the public proxy', () => {
    const packageJson = JSON.parse(readFileSync(new URL('../package.json', import.meta.url), 'utf8'))

    expect(packageJson.scripts.dev).toContain('--env-file=../../.env')
    expect(packageJson.scripts.dev).toContain('scripts/dev-theme-runtime.mjs')
  })

  test('dev supervisor uses serial lifecycle instead of the production blue-green helper', () => {
    const runtime = readFileSync(new URL('../scripts/dev-theme-runtime.mjs', import.meta.url), 'utf8')
    const lifecycle = readFileSync(new URL('../scripts/dev-theme-lifecycle.mjs', import.meta.url), 'utf8')

    expect(runtime).toContain('createDevThemeLifecycle')
    expect(runtime).toContain('stopProcessGroup')
    expect(runtime).not.toContain('replaceTarget')
    expect(lifecycle).toContain("signalGroup(pid, 'SIGTERM')")
    expect(lifecycle).toContain("signalGroup(pid, 'SIGKILL')")

    const startup = runtime.indexOf("await lifecycle.requestRestart('startup')")
    const listen = runtime.indexOf('await proxy.listen()')
    expect(startup).toBeGreaterThan(-1)
    expect(listen).toBeGreaterThan(startup)
  })
})
