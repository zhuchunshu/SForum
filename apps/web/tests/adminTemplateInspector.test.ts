import { describe, expect, test } from 'bun:test'
import { readFileSync } from 'node:fs'

import {
  parseTemplateInspectorSnapshot,
  useAdminTemplateInspector
} from '../app/composables/useAdminTemplateInspector'

const source = (path: string) => readFileSync(new URL(path, import.meta.url), 'utf8')

function templateSnapshot(overrides: Record<string, unknown> = {}) {
  return {
    schemaVersion: 'sforum.template-inspector@1',
    revision: 3,
    activeTheme: 'demo.active-theme',
    defaultTheme: 'demo.default-theme',
    snapshotCount: 2,
    overrideCount: 1,
    snapshots: [
      {
        extensionId: 'demo.active-theme',
        extensionVersion: '1.0.0',
        packageDigest: 'a'.repeat(64),
        kind: 'theme',
        contributionIds: ['demo.active-theme.home'],
        overrideTargets: ['sforum.plugin-page-business-e2e.template.article'],
        active: true,
        default: false
      },
      {
        extensionId: 'demo.plugin-pages',
        extensionVersion: '1.0.0',
        packageDigest: 'b'.repeat(64),
        kind: 'plugin',
        contributionIds: ['demo.plugin-pages.article'],
        overrideTargets: [],
        active: false,
        default: false
      }
    ],
    ...overrides
  }
}

describe('template inspector snapshot parsing', () => {
  test('accepts valid redacted snapshots', () => {
    const snapshot = parseTemplateInspectorSnapshot(templateSnapshot())
    expect(snapshot?.revision).toBe(3)
    expect(snapshot?.snapshotCount).toBe(2)
    expect(snapshot?.overrideCount).toBe(1)
    expect(snapshot?.activeTheme).toBe('demo.active-theme')
    expect(snapshot?.snapshots).toHaveLength(2)
    expect(snapshot?.snapshots[0]?.overrideTargets[0]).toContain('template.article')
  })

  test('rejects forged or incomplete shapes', () => {
    expect(parseTemplateInspectorSnapshot(null)).toBeNull()
    expect(parseTemplateInspectorSnapshot(templateSnapshot({ schemaVersion: 'sforum.template-inspector@2' }))).toBeNull()
    expect(parseTemplateInspectorSnapshot(templateSnapshot({ revision: '3' }))).toBeNull()
    expect(parseTemplateInspectorSnapshot(templateSnapshot({ snapshots: null }))).toBeNull()
    expect(parseTemplateInspectorSnapshot(templateSnapshot({
      snapshots: [{ extensionId: 'x' }]
    }))).toBeNull()
  })
})

describe('template inspector API boundary', () => {
  test('calls exact read-only endpoint and rejects invalid limits', async () => {
    const calls: string[] = []
    ;(globalThis as typeof globalThis & { useApiClient: () => unknown }).useApiClient = () => ({
      request: async (path: string) => {
        calls.push(path)
        return templateSnapshot()
      }
    })

    const api = useAdminTemplateInspector()
    expect((await api.inspect(50)).revision).toBe(3)
    expect(calls).toEqual(['/admin/extensions/template-inspector?limit=50'])

    await expect(api.inspect(0)).rejects.toThrow(/limit/)
    await expect(api.inspect(201)).rejects.toThrow(/limit/)
    expect(calls).toHaveLength(1)

    ;(globalThis as typeof globalThis & { useApiClient: () => unknown }).useApiClient = () => ({
      request: async () => ({ schemaVersion: 'forged' })
    })
    await expect(useAdminTemplateInspector().inspect()).rejects.toThrow(/invalid|failed/i)
  })
})

describe('template inspector page and navigation contracts', () => {
  const page = source('../app/pages/admin/extensions/template-inspector.vue')
  const modules = source('../app/config/adminModules.ts')
  const plugins = source('../app/pages/admin/extensions/plugins.vue')
  const en = source('../i18n/locales/en-US.json')
  const zh = source('../i18n/locales/zh-CN.json')

  test('registers extension.view read-only inspector page', () => {
    expect(modules).toContain("id: '/extensions/template-inspector'")
    expect(modules).toContain("componentName: 'AdminExtensionTemplateInspector'")
    expect(modules).toContain("pageId: '/extensions/template-inspector'")
    expect(plugins).toContain("adminRoutes.path('/extensions/template-inspector')")

    expect(page).toContain("name: 'AdminExtensionTemplateInspector'")
    expect(page).toContain("useAdminPage('/extensions/template-inspector')")
    expect(page).toContain('template-inspector-loading')
    expect(page).toContain('template-inspector-error')
    expect(page).toContain('template-inspector-empty-snapshots')
  })

  test('follows cache-inspector admin UX shell and fail-closed load errors', () => {
    expect(page).toContain('UDashboardToolbar')
    expect(page).toContain('SFAlert')
    expect(page).toContain('SFEmptyState')
    expect(page).toContain('USkeleton')
    expect(page).toContain('adminPage.icon')
    expect(page).toContain('i-lucide-rotate-cw')
    expect(page).toContain('void load()')
    expect(page).toContain('mapLoadError')
    expect(page).toContain("t('admin.extensions.templateInspector.loadFailed')")
    expect(page).not.toContain('UAlert')
    expect(page).not.toContain('onMounted(load)')
    expect(page).not.toContain('i-tabler-refresh')
  })

  test('ships matching navigation and UI copy without emoji', () => {
    for (const locale of [en, zh]) {
      expect(locale).toContain('"extensionTemplateInspector"')
      expect(locale).toContain('"templateInspector"')
      expect(locale).toContain('"redactionHint"')
      expect(locale).toContain('"loadFailed"')
      expect(locale).toContain('"summaryTitle"')
    }
    expect(page).not.toMatch(/[\u{1F300}-\u{1FAFF}]/u)
    expect(page).toContain('break-all')
  })
})
