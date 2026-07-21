import { describe, expect, test } from 'bun:test'

import { executableDatabaseGrants, executableDatabaseMigrationRisk } from '../app/utils/extensionTrust'

const manager = await Bun.file(new URL('../app/composables/useAdminExtensionsManager.ts', import.meta.url)).text()
const dialog = await Bun.file(new URL('../app/components/SFAdminExtensionEnableDialog.vue', import.meta.url)).text()
const impact = await Bun.file(new URL('../app/components/SFAdminExecutableTrustImpact.vue', import.meta.url)).text()
const overview = await Bun.file(new URL('../app/pages/admin/extensions/index.vue', import.meta.url)).text()
const plugins = await Bun.file(new URL('../app/pages/admin/extensions/plugins.vue', import.meta.url)).text()
const themesPage = await Bun.file(new URL('../app/pages/admin/extensions/themes.vue', import.meta.url)).text()
const frontendPanel = await Bun.file(new URL('../app/components/SFAdminFrontendTrustPanel.vue', import.meta.url)).text()
const frontendTrust = await Bun.file(new URL('../app/composables/useAdminFrontendTrust.ts', import.meta.url)).text()

describe('V3 exact-artifact trust operator flow', () => {
  test('loads the canonical preview and consumes a server-issued one-use token', () => {
    expect(manager).toContain('/admin/extensions/${item.id}/trust')
    expect(manager).toContain('/admin/extensions/${item.id}/trust/challenge')
    expect(manager).toContain('confirmationToken: enableTrustChallenge.value?.token')
    expect(manager).toContain("apiErrorReason(error) === 'extension.trust_not_required'")
    expect(manager).toContain("duration: 10000")
    expect(manager).toContain("duration: 0")
    expect(dialog).not.toContain('challenge.token')
  })

  test('keeps delegated managers in preview-only mode until an exact grant exists', () => {
    expect(manager).toContain("roleKeys?.includes('super_admin')")
    expect(dialog).toContain('needsChallenge && !isSuperAdmin')
    expect(dialog).toContain('delegatedPreviewOnly')
    expect(dialog).toContain('needsChallenge && isSuperAdmin && !challenge')
    expect(dialog).toContain(':disabled="!canConfirm"')
  })

  test('renders every canonical impact category and exact digests', () => {
    for (const field of [
      'artifactDigests', 'requestedAuthority', 'contracts', 'binaries', 'backend', 'routes', 'guards',
      'guardDeclarations', 'hooks', 'events', 'migrations', 'migrationDeclarations', 'providers', 'jobs',
      'schedules', 'components', 'registryComponents', 'templates', 'assets', 'content', 'database', 'cache',
      'services', 'commands', 'adminSurfaces', 'queries', 'queryResultFilters', 'identity', 'permissionDefinitions', 'media',
      'navigation', 'regions', 'contributions', 'capabilities', 'permissions', 'requiredFeatures',
      'dependencies', 'lifecycle', 'openapi', 'packageFiles'
    ]) {
      expect(impact).toContain(`key: '${field}'`)
    }
    expect(impact).toContain('impact.manifestContract')
    expect(impact).toContain('impact.packageDigest')
    expect(impact).toContain('impact.digest')
    expect(impact).toContain('JSON.stringify(value, null, 2)')
  })

  test('promotes database migration and recovery risk before confirmation', () => {
    expect(impact).toContain('data-testid="extension-database-risk"')
    expect(impact).toContain('database.coreCompatibility')
    expect(impact).toContain('databaseGrantSummary')
    expect(impact).toContain('database.backup.required')
    expect(impact).toContain('database.backup.strategy')
    expect(impact).toContain('database.retention.onDisable')
    expect(impact).toContain('database.retention.onUninstall')
    expect(impact).toContain('migration.digest')
    expect(impact).toContain('migrationTransaction(migration)')
    expect(impact).toContain('data-testid="extension-database-high-risk"')
    expect(impact).toContain('data-testid="extension-database-backup-guidance"')
    expect(impact).toContain('}, 10000)')

    const risk = executableDatabaseMigrationRisk({
      database: {
        contractVersion: 'fixture.database@1',
        authority: 'raw_core',
        coreCompatibility: '>=3.0.0 <4.0.0',
        backup: { required: false },
        retention: { onDisable: 'retain', onUninstall: 'export' }
      },
      migrationDeclarations: [{
        id: 'fixture.migration.1',
        contractVersion: 'fixture.migration@1',
        path: 'migrations/001.sql',
        digest: 'a'.repeat(64),
        transaction: 'forbidden'
      }]
    })
    expect(risk.hasDatabaseChanges).toBe(true)
    expect(risk.missingRequiredBackup).toBe(true)
    expect(risk.nonTransactional).toHaveLength(1)
    expect(executableDatabaseGrants({
      contractVersion: 'fixture.database@1',
      authority: 'raw_core',
      backup: { required: false },
      retention: { onDisable: 'retain', onUninstall: 'retain' }
    })).toEqual(['own_schema', 'core_views', 'host_commands', 'raw_core'])
    expect(executableDatabaseGrants({
      contractVersion: 'fixture.database@1',
      grants: ['raw_core', 'core_views'],
      backup: { required: false },
      retention: { onDisable: 'retain', onUninstall: 'retain' }
    })).toEqual(['core_views', 'raw_core'])
  })

  test('reuses one dialog in both admin entry points and retires frontend-only grant in V3', () => {
    expect(overview).toContain('<SFAdminExtensionEnableDialog')
    expect(plugins).toContain('<SFAdminExtensionEnableDialog')
    expect(frontendPanel).toContain('isSuperAdmin && !exactTrustManaged')
    expect(frontendPanel).toContain('exactTrustDescription')
    expect(frontendTrust).toContain('extension.value.status')
    expect(frontendTrust).toContain('extension.value.packageDigest')
  })

  test('executable uploaded themes reuse exact trust challenge for activate and send confirmationToken with preview tuple', () => {
    // Ordinary L0/L1 (or V3 gate off) stays operator-buildless: page preview Modal, no challenge issue step.
    expect(manager).toContain("apiErrorReason(err) === 'extension.trust_not_required'")
    expect(manager).toContain('!trustStatus.trustRequired')
    expect(manager).toContain('openThemePreviewConfirm(item, preview)')
    expect(manager).toContain('await performActivateTheme(item, preview)')
    expect(manager).not.toContain('globalThis.confirm')
    // L2 path: open exact dialog only when trustRequired; challenge is super_admin-gated.
    expect(manager).toContain('themeActivateConfirmOpen.value = true')
    expect(manager).toContain('if (!item || !isSuperAdmin.value) return')
    expect(manager).toContain('/admin/extensions/${item.id}/trust/challenge')
    // Activate POST carries the full ThemeActivationRequest tuple and optional one-use token.
    expect(manager).toContain('function themeActivationRequestBody')
    for (const field of [
      'version: preview.version',
      'packageDigest: preview.packageDigest',
      'currentThemeId: preview.currentThemeId',
      'currentThemeVersion: preview.currentThemeVersion',
      'currentThemeDigest: preview.currentThemeDigest',
      'approveCoreReplacements: preview.requiresCoreReplacementApproval && preview.canApproveCoreReplacements'
    ]) {
      expect(manager).toContain(field)
    }
    expect(manager).toContain('body.confirmationToken = token')
    expect(manager).toContain('postActivateTheme(item, preview, themeActivateTrustChallenge.value?.token)')
    // stale/expired/replayed/denied: blocking error stays, token discarded, trust status refreshed for retry.
    expect(manager).toMatch(
      /themeActivateTrustError\.value = apiErrorMessage\(result\.error\)[\s\S]*?themeActivateTrustChallenge\.value = null[\s\S]*?await refreshThemeActivateRetryContext\(item\)/
    )
    expect(manager).toContain('request<ExecutableTrustStatus>(`/admin/extensions/${item.id}/trust`)')
    expect(manager).toContain('request<ThemeActivationPreview>(`/admin/pages/activate-preview/${item.id}`)')
    expect(manager).toContain('duration: 0')
    // Dialog surfaces: activate purpose, no raw token render, super_admin gate, disabled confirm until ready.
    expect(overview).toContain('purpose="activate"')
    expect(overview).toContain('<SFAdminThemeActivateDialog')
    expect(overview).toContain('reactivateTheme')
    expect(themesPage).toContain('purpose="activate"')
    expect(themesPage).toContain('<SFAdminThemeActivateDialog')
    expect(themesPage).toContain('reactivateTheme')
    expect(themesPage).toContain('confirmThemeActivate')
    expect(themesPage).toContain('issueThemeActivateTrustChallenge')
    expect(themesPage).toContain('confirmThemePreviewActivate')
    expect(dialog).toContain("purpose?: 'enable' | 'activate'")
    expect(dialog).toContain('blockingErrorActivate')
    expect(dialog).not.toContain('challenge.token')
    expect(dialog).toContain('needsChallenge && !isSuperAdmin')
    expect(dialog).toContain(':disabled="!canConfirm"')
    expect(dialog).toContain('role="alert"')
  })
})
