import { describe, expect, test } from 'bun:test'

const manager = await Bun.file(new URL('../app/composables/useAdminExtensionsManager.ts', import.meta.url)).text()
const dialog = await Bun.file(new URL('../app/components/SFAdminExtensionEnableDialog.vue', import.meta.url)).text()
const impact = await Bun.file(new URL('../app/components/SFAdminExecutableTrustImpact.vue', import.meta.url)).text()
const overview = await Bun.file(new URL('../app/pages/admin/extensions/index.vue', import.meta.url)).text()
const plugins = await Bun.file(new URL('../app/pages/admin/extensions/plugins.vue', import.meta.url)).text()
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
      'services', 'commands', 'adminSurfaces', 'queries', 'identity', 'permissionDefinitions', 'media',
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

  test('reuses one dialog in both admin entry points and retires frontend-only grant in V3', () => {
    expect(overview).toContain('<SFAdminExtensionEnableDialog')
    expect(plugins).toContain('<SFAdminExtensionEnableDialog')
    expect(frontendPanel).toContain('isSuperAdmin && !exactTrustManaged')
    expect(frontendPanel).toContain('exactTrustDescription')
    expect(frontendTrust).toContain('extension.value.status')
    expect(frontendTrust).toContain('extension.value.packageDigest')
  })
})
