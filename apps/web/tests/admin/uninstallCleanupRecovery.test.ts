import { describe, expect, test } from 'bun:test'
import { readFileSync } from 'node:fs'

const source = (path: string) => readFileSync(new URL(path, import.meta.url), 'utf8')

describe('uninstall cleanup recovery', () => {
  test('persists the uninstall idempotency key in sessionStorage and forgets it only on success', () => {
    const cleanup = source('../../app/composables/admin/useAdminUninstallCleanup.ts')
    const composable = source('../../app/composables/admin/useAdminExtensionsManager.ts')
    expect(cleanup).toContain("const UNINSTALL_KEY_STORAGE_PREFIX = 'sforum:uninstall-key:'")
    expect(cleanup).toContain('globalThis.sessionStorage.getItem(`${UNINSTALL_KEY_STORAGE_PREFIX}${extensionId}`)')
    expect(cleanup).toContain('globalThis.sessionStorage.setItem(`${UNINSTALL_KEY_STORAGE_PREFIX}${extensionId}`, key)')
    expect(cleanup).toContain('forgetUninstallKey(item.id)')
    // 打开确认框时先恢复 storage 里的原键，再考虑生成新键。
    expect(cleanup).toContain('const stored = storedUninstallKey(item.id)')
    expect(cleanup).toContain('uninstallKeys.set(item.id, stored || lifecycleIdempotencyKey())')
    // 管理器只做装配：卸载成功后的列表投影经回调交回管理器。
    expect(composable).toContain('useAdminUninstallCleanup({')
    expect(composable).toContain('...uninstallCleanup,')
  })

  test('captures structured 503 details and offers the keyless finalize endpoint', () => {
    const cleanup = source('../../app/composables/admin/useAdminUninstallCleanup.ts')
    expect(cleanup).toContain('uninstallCleanupDetails.value = apiErrorDetails(error) || null')
    expect(cleanup).toContain('`/admin/extensions/${item.id}/uninstall-cleanup/finalize`')
    expect(cleanup).toContain('finalizeCleanupBusyId.value = item.id')
    const client = source('../../app/composables/useApiClient.ts')
    expect(client).toContain('export function apiErrorDetails(error: unknown)')
    expect(client).toContain('details?: Record<string, unknown>')
  })

  test('renders the failed step, retry action, and pending-cleanup finalize in the dialog', () => {
    const dialog = source('../../app/components/admin/SFAdminExtensionUninstallDialog.vue')
    expect(dialog).toContain('admin.extensions.uninstallCleanup.failedStep')
    expect(dialog).toContain('admin.extensions.uninstallCleanup.retry')
    expect(dialog).toContain('admin.extensions.uninstallCleanup.pendingBadge')
    expect(dialog).toContain('admin.extensions.uninstallCleanup.finalize')
    expect(dialog).toContain("emit('finalize')")
    // 未知步骤回退而不是渲染原始 i18n key。
    expect(dialog).toContain('admin.extensions.uninstallCleanup.unknownStep')
  })

  test('wires cleanup details, finalize busyness, and the pending badge into admin pages', () => {
    for (const page of [
      '../../app/pages/admin/extensions/plugins.vue',
      '../../app/pages/admin/extensions/themes.vue',
      '../../app/pages/admin/extensions/index.vue'
    ]) {
      const text = source(page)
      expect(text).toContain(':cleanup-details="uninstallCleanupDetails"')
      expect(text).toContain(':finalize-busy=')
      expect(text).toContain('@finalize="uninstallConfirmItem && finalizeUninstallCleanup(uninstallConfirmItem)"')
      expect(text).toContain('item.pendingUninstallCleanup')
      expect(text).toContain('admin.extensions.uninstallCleanup.pendingBadge')
    }
    const types = source('../../app/utils/admin/adminExtensions.ts')
    expect(types).toContain('pendingUninstallCleanup?: boolean')
  })

  test('localizes every cleanup step exposed by the API failure details', () => {
    const dialog = source('../../app/components/admin/SFAdminExtensionUninstallDialog.vue')
    expect(dialog).toContain('admin.extensions.uninstallCleanup.steps.${step}')
    for (const locale of ['zh-CN', 'en-US']) {
      const localeSource = source(`../../i18n/locales/${locale}.json`)
      for (const step of [
        'quarantine_asset_publication',
        'release_attachment_references',
        'clear_provider_selections',
        'physical_purge',
        'not_finalized'
      ]) {
        expect(localeSource).toContain(`"${step}"`)
      }
    }
  })
})
