import { describe, expect, test } from 'bun:test'

const manager = await Bun.file(new URL('../app/composables/useAdminExtensionsManager.ts', import.meta.url)).text()
const trust = await Bun.file(new URL('../app/composables/useAdminFrontendTrust.ts', import.meta.url)).text()
const releases = await Bun.file(new URL('../app/pages/admin/extensions/releases.vue', import.meta.url)).text()
const zh = JSON.parse(await Bun.file(new URL('../i18n/locales/zh-CN.json', import.meta.url)).text())
const en = JSON.parse(await Bun.file(new URL('../i18n/locales/en-US.json', import.meta.url)).text())

describe('plugin enable/disable lifecycle feedback', () => {
  test('does not reuse theme activation copy for plugin queue toasts', () => {
    // 排队路径必须用插件专用文案，避免“主题构建”误导。
    expect(manager).toContain("pluginEnableQueued")
    expect(manager).toContain("pluginDisableQueued")
    expect(manager).toContain('webReleaseQueuedHint')
    expect(manager).not.toMatch(/operation\.queued \? t\('admin\.extensions\.themeActivationQueued'\)/)
  })

  test('trust grant/revoke surfaces web release log guidance when queued', () => {
    expect(trust).toContain('grantQueued')
    expect(trust).toContain('revokeQueued')
    expect(trust).toContain('webReleaseQueuedHint')
  })

  test('web releases detail always exposes a build log section', () => {
    expect(releases).toContain("t('admin.extensions.releases.buildLog')")
    expect(releases).toContain("t('admin.extensions.releases.emptyBuildLog')")
    expect(releases).toContain("t('admin.extensions.releases.viewDetail')")
  })

  test('web releases page exposes manual rebuild action', () => {
    expect(releases).toContain('rebuild')
    expect(releases).toContain("t('admin.extensions.releases.rebuild')")
    expect(releases).toContain("t('admin.extensions.releases.rebuildTitle')")
    expect(releases).toContain('i-lucide-hammer')
  })

  test('web releases page exposes explicit typecheck policy modes', () => {
    expect(releases).toContain('typecheckMode')
    expect(releases).toContain('setTypecheckMode')
    expect(releases).toContain("t('admin.extensions.releases.typecheckModeTitle')")
    expect(releases).toContain('USelect')
  })

  test('plugin list shows web release progress bars and polls while active', async () => {
    const plugins = await Bun.file(new URL('../app/pages/admin/extensions/plugins.vue', import.meta.url)).text()
    expect(plugins).toContain('pluginWebReleaseProgress')
    expect(plugins).toContain('hasPluginWebReleaseInProgress')
    expect(plugins).toContain('<UProgress')
    expect(plugins).toContain('startReleasePolling')
    // 与主题一致：进度条下方行内展开日志，而不是跳转到 Web 发布页。
    expect(plugins).toContain('toggleBuildLog')
    expect(plugins).toContain("t('admin.extensions.viewBuildLog')")
    expect(plugins).not.toContain("adminRoutes.path('/extensions/releases')")
  })

  test('locale catalogs include plugin queue and release log copy', () => {
    for (const catalog of [zh, en]) {
      expect(catalog.admin.extensions.pluginEnableQueued).toBeTruthy()
      expect(catalog.admin.extensions.pluginDisableQueued).toBeTruthy()
      expect(catalog.admin.extensions.webReleaseQueuedHint).toContain('{id}')
      expect(catalog.admin.extensions.releases.buildLog).toBeTruthy()
      expect(catalog.admin.extensions.releases.emptyBuildLog).toBeTruthy()
      expect(catalog.admin.extensions.frontend.grantQueued).toBeTruthy()
      // 动态页区分「启用中」与「已禁用」，避免 Web Release 期间误导。
      expect(catalog.admin.extensions.dynamic.enablingTitle).toBeTruthy()
      expect(catalog.admin.extensions.dynamic.reloadRequiredTitle).toBeTruthy()
      expect(catalog.admin.extensions.dynamic.plainDevTitle).toBeTruthy()
      // F2.4 升级/卸载文案。
      expect(catalog.admin.extensions.upgraded).toBeTruthy()
      expect(catalog.admin.extensions.uninstalled).toBeTruthy()
      expect(catalog.admin.extensions.confirmUninstallTitle).toBeTruthy()
    }
  })

  test('upload handles InstallResult upgrade metadata and uninstall flow exists', () => {
    expect(manager).toContain('upgraded')
    expect(manager).toContain('trustRevoked')
    expect(manager).toContain('openUninstallExtension')
    expect(manager).toContain("method: 'DELETE'")
  })

  test('dynamic extension page shares list cache and polls while enabling', async () => {
    const dynamicPage = await Bun.file(
      new URL('../app/pages/admin/extensions/[extensionId]/pages/[...pagePath].vue', import.meta.url)
    ).text()
    // 与插件列表共用 admin-extensions，启用/轮询结果能立刻反映到设置页。
    expect(dynamicPage).toContain("useAsyncData<AdminExtension[]>('admin-extensions'")
    expect(dynamicPage).not.toContain('admin-extension-dynamic-list')
    expect(dynamicPage).toContain('isLifecyclePending')
    expect(dynamicPage).toContain('startLifecyclePolling')
    expect(dynamicPage).toContain('enablingTitle')
    expect(dynamicPage).toContain('needsFrontendReload')
    expect(dynamicPage).toContain('registryUnavailable')
    expect(dynamicPage).toContain('plainDevTitle')
  })

  test('admin extension registry exposes whether frontend was injected', async () => {
    const registry = await Bun.file(
      new URL('../app/composables/useAdminExtensionRegistry.ts', import.meta.url)
    ).text()
    expect(registry).toContain('adminFrontendInjected')
    expect(registry).toContain("releaseId !== 'core'")
  })
})
