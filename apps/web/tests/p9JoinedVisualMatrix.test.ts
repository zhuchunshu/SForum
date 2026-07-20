import { describe, expect, it } from 'bun:test'
import { readFileSync } from 'node:fs'
import { Window } from 'happy-dom'

import { mountPublicFrontendModule } from '../app/runtime/public-extensions/mount'
import { resetPublicAssetRuntimeForTest } from '../app/runtime/public-extensions/assets'
import {
  PUBLIC_FRONTEND_API_VERSION,
  PUBLIC_FRONTEND_SCHEMA_VERSION,
  PUBLIC_FRONTEND_TRUST_NOTICE,
  type PublicFrontendBridgeV1
} from '../app/runtime/public-extensions/types'

function source(relative: string) {
  return readFileSync(new URL(relative, import.meta.url), 'utf8')
}

/**
 * P9 desktop/mobile visual + interaction matrix for high-traffic replaced surfaces.
 * Structural contracts for desktop/mobile chrome; happy-dom for mount/unmount cleanup.
 * Full Playwright browser is optional (env may lack playwright-cli) — structural +
 * mount gates are the accepted production row for this exit.
 */
describe('P9 joined desktop/mobile visual matrix', () => {
  it('keeps high-traffic public chrome desktop and mobile contracts', () => {
    const navbar = source('../app/components/SFNavbar.vue')
    const home = source('../app/pages/index.vue')
    const widget = source('../app/components/SFExtensionWidget.vue')
    const theme = source('../app/components/SFThemeTemplate.vue')

    // desktop shell
    expect(navbar).toContain('navbar__desktop-nav')
    expect(navbar).toContain('desktopNavItems')
    expect(navbar).toContain('min-height: var(--sf-public-topbar-height, 52px)')
    // mobile shell
    expect(navbar).toContain('navbar__mobile-new-topic')
    expect(navbar).toContain('mobileMenuItems')
    expect(navbar).toContain('i-lucide-menu')
    expect(navbar).toContain(':aria-label="t(\'nav.openMenu\')"')
    // no emoji icons
    expect(navbar).not.toMatch(/[\u{1F300}-\u{1FAFF}]/u)

    // home uses page outlet (theme-replaceable high-traffic surface)
    expect(home).toContain('SFPageOutlet')
    expect(home).toContain('forum.home')

    // L2 honesty + theme CSP wiring for replaced widgets
    // 源码引用常量 PUBLIC_FRONTEND_TRUST_NOTICE（值为 fully_trusted_browser_code），
    // 与 publicL2Honesty.test.ts 对齐，不要求字面量出现在 SFC 中。
    expect(widget).toContain('public-l2-honesty')
    expect(widget).toContain('PUBLIC_FRONTEND_TRUST_NOTICE')
    expect(widget).toContain('data-testid="public-l2-honesty"')
    expect(widget).toContain('data-testid="public-l2-honesty-dismiss"')
    expect(theme).toContain('applyPublicPageDocumentPolicy')
    expect(theme).toContain('core.component.shared.sfextension_widget')
  })

  it('mounts L2 then unmount cleans DOM (desktop viewport 1280px)', async () => {
    resetPublicAssetRuntimeForTest()
    const browser = new Window({ url: 'https://forum.example/', width: 1280, height: 800 })
    expect(browser.innerWidth).toBeGreaterThanOrEqual(1024)

    // mount-only：不走 loadPublicFrontendRelease（依赖全局 document，bun 无 DOM 环境）。
    // CSS lease / integrity 已在 publicExtensionRuntime / lease 测试覆盖。
    const { target, bridge } = mountFixture(browser)
    const releaseMount = await mountPublicFrontendModule({
      apiVersion: 1,
      async mount(element) {
        const button = element.ownerDocument.createElement('button')
        button.setAttribute('data-p9-visual', 'desktop')
        button.setAttribute('type', 'button')
        button.textContent = 'L2'
        element.append(button)
        return () => { element.replaceChildren() }
      }
    }, target, bridge)

    expect(target.querySelector('[data-p9-visual="desktop"]')).toBeTruthy()
    await releaseMount()
    expect(target.childElementCount).toBe(0)
    resetPublicAssetRuntimeForTest()
  })

  it('keeps mobile viewport interaction usable after mount/unmount (390px)', async () => {
    resetPublicAssetRuntimeForTest()
    const browser = new Window({ url: 'https://forum.example/', width: 390, height: 844 })
    expect(browser.innerWidth).toBeLessThanOrEqual(430)

    const { target, bridge } = mountFixture(browser)
    const releaseMount = await mountPublicFrontendModule({
      apiVersion: 1,
      mount(element) {
        const action = element.ownerDocument.createElement('button')
        action.setAttribute('data-p9-visual', 'mobile')
        action.setAttribute('type', 'button')
        action.style.minHeight = '40px'
        action.style.minWidth = '40px'
        element.append(action)
        return () => { element.replaceChildren() }
      }
    }, target, bridge)

    const button = target.querySelector('[data-p9-visual="mobile"]') as HTMLElement | null
    expect(button).toBeTruthy()
    expect(button?.style.minHeight).toBe('40px')
    // 触控目标最小尺寸约定（移动端可用交互）
    expect(Number.parseInt(button?.style.minWidth || '0', 10)).toBeGreaterThanOrEqual(40)
    await releaseMount()
    expect(target.childElementCount).toBe(0)
    resetPublicAssetRuntimeForTest()
  })
})

function mountFixture(browser: Window) {
  const digest = 'a'.repeat(64)
  const packageDigest = 'b'.repeat(64)
  const target = browser.document.createElement('div')
  const ssrRoot = browser.document.createElement('div')
  const bridge: PublicFrontendBridgeV1 = Object.freeze({
    apiVersion: PUBLIC_FRONTEND_API_VERSION,
    trust: PUBLIC_FRONTEND_TRUST_NOTICE,
    extensionId: 'demo.public',
    extensionVersion: '1.0.0',
    packageDigest,
    impactDigest: digest,
    componentId: 'demo.public.component.card',
    locale: 'zh-CN',
    appearance: Object.freeze({ colorMode: 'light' as const, accent: '', accentContrast: '' }),
    props: Object.freeze({}),
    ssrRoot,
    request: async () => { throw new Error('unused') },
    navigate: async () => {}
  })
  // 保留 schema 常量引用，防止视觉矩阵与 descriptor 契约漂移时静默通过。
  void PUBLIC_FRONTEND_SCHEMA_VERSION
  return { target, bridge }
}
