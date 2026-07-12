import { describe, expect, test } from 'bun:test'
import { readFileSync } from 'node:fs'

describe('dev runtime startup', () => {
  test('dev script loads the root env before starting the public proxy', () => {
    const packageJson = JSON.parse(readFileSync(new URL('../package.json', import.meta.url), 'utf8'))

    expect(packageJson.scripts.dev).toContain('--env-file=../../.env')
    expect(packageJson.scripts.dev).toContain('scripts/dev-theme-runtime.mjs')
  })

  test('dev:plain keeps raw nuxt but still acknowledges web releases', () => {
    const packageJson = JSON.parse(readFileSync(new URL('../package.json', import.meta.url), 'utf8'))
    const plain = readFileSync(new URL('../scripts/dev-plain.mjs', import.meta.url), 'utf8')
    const ack = readFileSync(new URL('../scripts/dev-plain-release-ack.mjs', import.meta.url), 'utf8')

    expect(packageJson.scripts['dev:plain']).toContain('scripts/dev-plain.mjs')
    expect(packageJson.scripts['dev:nuxt']).toContain('nuxt dev')
    expect(plain).toContain("'nuxt', 'dev'")
    expect(plain).toContain('startPlainReleaseAck')
    expect(ack).toContain('writeActiveAcknowledgement')
    expect(ack).toContain('watchableReleaseFile')
  })

  test('dev supervisor uses serial lifecycle instead of the production blue-green helper', () => {
    const runtime = readFileSync(new URL('../scripts/dev-theme-runtime.mjs', import.meta.url), 'utf8')
    const lifecycle = readFileSync(new URL('../scripts/dev-theme-lifecycle.mjs', import.meta.url), 'utf8')

    expect(runtime).toContain('createDevThemeLifecycle')
    expect(runtime).toContain('stopProcessGroup')
    expect(runtime).not.toContain('replaceTarget')
    // supervisor 内层必须是裸 nuxt，避免再套一层 plain ack wrapper。
    expect(runtime).toContain("'dev:nuxt'")
    expect(lifecycle).toContain("signalGroup(pid, 'SIGTERM')")
    expect(lifecycle).toContain("signalGroup(pid, 'SIGKILL')")

    expect(runtime).toContain("await lifecycle.requestRestart('startup')")
  })

  test('clears Nitro route cache on theme switch but not on cold startup', () => {
    const runtime = readFileSync(new URL('../scripts/dev-theme-runtime.mjs', import.meta.url), 'utf8')
    const launchDevChild = runtime.slice(
      runtime.indexOf('function launchDevChild'),
      runtime.indexOf('function createLifecycle'),
    )

    expect(runtime).toContain('clearNuxtRouteCache,')
    expect(runtime).toContain(
      "const nuxtBuildDir = path.resolve(process.cwd(), process.env.NUXT_BUILD_DIR || '.nuxt')",
    )
    expect(launchDevChild).toContain("if (reason !== 'startup')")
    expect(launchDevChild).toContain('clearNuxtRouteCache(nuxtBuildDir)')
    expect(launchDevChild).toContain("spawn(bunPath, ['run', 'dev:nuxt']")
  })

  test('passes immutable release registry identity and ignores acknowledgement writes', () => {
    const runtime = readFileSync(new URL('../scripts/dev-theme-runtime.mjs', import.meta.url), 'utf8')

    expect(runtime).toContain('env.SFORUM_ADMIN_REGISTRY_ROOT = selection.registryRoot')
    expect(runtime).toContain('env.SFORUM_WEB_RELEASE_ID = String(selection.releaseId)')
    expect(runtime).toContain('writeActiveAcknowledgement')
    expect(runtime).toContain('watchableReleaseFile(changed)')
  })

  test('defaults to lightweight dev-compose direct mode; full release keeps proxy', () => {
    const runtime = readFileSync(new URL('../scripts/dev-theme-runtime.mjs', import.meta.url), 'utf8')

    expect(runtime).toContain('watchDevAdminCompose')
    expect(runtime).toContain('composeDevAdmin')
    expect(runtime).toContain('SFORUM_DEV_USE_RELEASE')
    expect(runtime).toContain('dev-compose direct mode')
    expect(runtime).toContain('useProxy')
    expect(runtime).toContain("PORT: useProxy ? '0' : String(externalPort)")
    expect(runtime).toContain('waitForTcpListen')
    expect(runtime).toContain('isNumericReleaseId')
    // 完整 release 确认仍保留，但 dev-local 不得写 active.json
    expect(runtime).toContain('writeActiveAcknowledgement')
  })
})
