import { describe, expect, test } from 'bun:test'
import { readFileSync } from 'node:fs'

import {
  parseComponentCompositionSnapshot,
  parseNavigationInspectorSnapshot,
  useAdminCompositionInspectors
} from '../../app/composables/admin/useAdminCompositionInspectors'

const source = (path: string) => readFileSync(new URL(path, import.meta.url), 'utf8')

function componentSnapshot(overrides: Record<string, unknown> = {}) {
  return {
    revision: 3,
    safeMode: false,
    targetCount: 12,
    contributionCount: 4,
    conflicts: [{ targetId: 'core.component.demo', providers: ['a', 'b'] }],
    traces: [{ targetId: 'core.component.demo', outcome: 'composed' }],
    ...overrides
  }
}

function navigationSnapshot(overrides: Record<string, unknown> = {}) {
  return {
    revision: 5,
    digest: 'a'.repeat(64),
    safeMode: false,
    navigationCount: 2,
    regionCount: 1,
    providerConflicts: 0,
    traces: [{ targetId: 'core.nav.primary', outcome: 'composed' }],
    ...overrides
  }
}

describe('composition inspector snapshot parsing', () => {
  test('accepts valid component and navigation snapshots', () => {
    const component = parseComponentCompositionSnapshot(componentSnapshot())
    expect(component?.revision).toBe(3)
    expect(component?.contributionCount).toBe(4)
    expect(component?.conflicts).toHaveLength(1)

    const navigation = parseNavigationInspectorSnapshot(navigationSnapshot())
    expect(navigation?.revision).toBe(5)
    expect(navigation?.digest).toHaveLength(64)
    expect(navigation?.navigationCount).toBe(2)
  })

  test('rejects forged or incomplete shapes', () => {
    expect(parseComponentCompositionSnapshot(null)).toBeNull()
    expect(parseComponentCompositionSnapshot(componentSnapshot({ revision: '3' }))).toBeNull()
    expect(parseComponentCompositionSnapshot(componentSnapshot({ safeMode: 'no' }))).toBeNull()
    expect(parseComponentCompositionSnapshot(componentSnapshot({ conflicts: 'x' }))).toBeNull()
    expect(parseComponentCompositionSnapshot(componentSnapshot({ traces: 'x' }))).toBeNull()

    expect(parseNavigationInspectorSnapshot({})).toBeNull()
    expect(parseNavigationInspectorSnapshot(navigationSnapshot({ digest: 1 }))).toBeNull()
    expect(parseNavigationInspectorSnapshot(navigationSnapshot({ regionCount: '1' }))).toBeNull()
    expect(parseNavigationInspectorSnapshot(navigationSnapshot({ traces: {} }))).toBeNull()
  })

  test('accepts null conflicts/traces as empty arrays (legacy empty-slice JSON)', () => {
    const component = parseComponentCompositionSnapshot(componentSnapshot({ conflicts: null, traces: null }))
    expect(component?.conflicts).toEqual([])
    expect(component?.traces).toEqual([])

    const navigation = parseNavigationInspectorSnapshot(navigationSnapshot({ traces: null }))
    expect(navigation?.traces).toEqual([])
  })
})

describe('composition inspector API boundary', () => {
  test('calls exact read-only endpoints and rejects invalid limits', async () => {
    const calls: string[] = []
    ;(globalThis as typeof globalThis & { useApiClient: () => unknown }).useApiClient = () => ({
      request: async (path: string) => {
        calls.push(path)
        if (path.includes('component-inspector')) return componentSnapshot()
        return navigationSnapshot()
      }
    })

    const api = useAdminCompositionInspectors()
    expect((await api.inspectComponents(50)).revision).toBe(3)
    expect((await api.inspectNavigation(25)).navigationCount).toBe(2)
    expect(calls).toEqual([
      '/admin/extensions/component-inspector?limit=50',
      '/admin/extensions/navigation-inspector?limit=25'
    ])

    await expect(api.inspectComponents(0)).rejects.toThrow(/limit/)
    await expect(api.inspectNavigation(201)).rejects.toThrow(/limit/)
    expect(calls).toHaveLength(2)

    ;(globalThis as typeof globalThis & { useApiClient: () => unknown }).useApiClient = () => ({
      request: async () => ({ revision: 'forged' })
    })
    await expect(useAdminCompositionInspectors().inspectComponents()).rejects.toThrow(/invalid|failed/i)
  })
})

describe('composition inspector pages and navigation contracts', () => {
  const componentPage = source('../../app/pages/admin/extensions/component-inspector.vue')
  const navigationPage = source('../../app/pages/admin/extensions/navigation-inspector.vue')
  const modules = source('../../app/config/adminModules.ts')
  const plugins = source('../../app/pages/admin/extensions/plugins.vue')
  const en = source('../../i18n/locales/en-US.json')
  const zh = source('../../i18n/locales/zh-CN.json')

  test('registers extension.view read-only inspector pages', () => {
    expect(modules).toContain("id: '/extensions/component-inspector'")
    expect(modules).toContain("componentName: 'AdminExtensionComponentInspector'")
    expect(modules).toContain("pageId: '/extensions/component-inspector'")
    expect(modules).toContain("id: '/extensions/navigation-inspector'")
    expect(modules).toContain("componentName: 'AdminExtensionNavigationInspector'")
    expect(modules).toContain("pageId: '/extensions/navigation-inspector'")

    expect(plugins).toContain("adminRoutes.path('/extensions/component-inspector')")
    expect(plugins).toContain("adminRoutes.path('/extensions/navigation-inspector')")

    expect(componentPage).toContain("name: 'AdminExtensionComponentInspector'")
    expect(componentPage).toContain("useAdminPage('/extensions/component-inspector')")
    expect(componentPage).toContain('component-inspector-loading')
    expect(componentPage).toContain('component-inspector-error')
    expect(componentPage).toContain('component-inspector-safe-mode')
    expect(componentPage).toContain('component-inspector-empty-traces')

    expect(navigationPage).toContain("name: 'AdminExtensionNavigationInspector'")
    expect(navigationPage).toContain("useAdminPage('/extensions/navigation-inspector')")
    expect(navigationPage).toContain('navigation-inspector-loading')
    expect(navigationPage).toContain('navigation-inspector-error')
    expect(navigationPage).toContain('navigation-inspector-safe-mode')
    expect(navigationPage).toContain('navigation-inspector-empty-traces')
  })

  test('follows cache-inspector admin UX shell and fail-closed load errors', () => {
    for (const page of [componentPage, navigationPage]) {
      expect(page).toContain('UDashboardToolbar')
      expect(page).toContain('SFAlert')
      expect(page).toContain('SFEmptyState')
      expect(page).toContain('USkeleton')
      expect(page).toContain('adminPage.icon')
      expect(page).toContain('i-lucide-rotate-cw')
      expect(page).toContain('void load()')
      expect(page).toContain('mapLoadError')
      expect(page).not.toContain('UAlert')
      expect(page).not.toContain('onMounted(load)')
      expect(page).not.toContain('i-tabler-refresh')
      expect(page).not.toContain('JSON.stringify')
    }
    expect(componentPage).toContain("t('admin.extensions.componentInspector.loadFailed')")
    expect(navigationPage).toContain("t('admin.extensions.navigationInspector.loadFailed')")
  })

  test('ships matching navigation and UI copy without emoji', () => {
    for (const locale of [en, zh]) {
      expect(locale).toContain('"extensionComponentInspector"')
      expect(locale).toContain('"extensionNavigationInspector"')
      expect(locale).toContain('"componentInspector"')
      expect(locale).toContain('"navigationInspector"')
      expect(locale).toContain('"safeModeDescription"')
      expect(locale).toContain('"loadFailed"')
      expect(locale).toContain('"summaryTitle"')
      expect(locale).toContain('"tracesEmptyTitle"')
    }
    expect(componentPage).not.toMatch(/[\u{1F300}-\u{1FAFF}]/u)
    expect(navigationPage).not.toMatch(/[\u{1F300}-\u{1FAFF}]/u)
    expect(componentPage).toContain('break-all')
    expect(navigationPage).toContain('break-all')
  })
})
