import { describe, expect, test } from 'bun:test'
import { ref } from 'vue'
import { useAdminPageHostInstall } from '../../app/composables/admin/useAdminPageHostInstall'
import { toAdminHostInstallResult } from '../../app/composables/admin/useAdminExtensionInstall'
import {
  ADMIN_HOST_INSTALL_TIMEOUT_MS,
  AdminHostInstallError,
  adminExtensionInstallErrorMessage,
  adminPageCanInstallExtensions,
  adminPageDeclaresHostAction,
  adminPageHostActions,
  adminPageSourceLabel,
  extensionCanRequestInstall,
  validateAdminHostInstallRequest
} from '../../app/utils/admin/extensions/hostActions'
import type { AdminExtensionInstallResult } from '../../app/utils/admin/adminExtensions'

const digest = 'a'.repeat(64)

function validRequest(overrides: Record<string, unknown> = {}) {
  return {
    url: 'https://cdn.example.com/sforum.store-1.4.0.zip',
    sha256: digest,
    expectedExtensionId: 'sforum.store',
    expectedType: 'plugin',
    ...overrides
  } as never
}

function installResult(overrides: Partial<AdminExtensionInstallResult['extension']> = {}): AdminExtensionInstallResult {
  return {
    upgraded: false,
    extension: {
      id: 'sforum.store',
      name: 'App Store',
      version: '1.4.0',
      type: 'plugin',
      status: 'installed',
      manifest: { backend: { entry: 'backend/plugin' } } as never,
      packageDigest: 'b'.repeat(64),
      packagePath: '/tmp/pkg',
      installedAt: '',
      updatedAt: '',
      ...overrides
    }
  } as AdminExtensionInstallResult
}

function harness(options: {
  install?: (url: string, sha256: string, config?: unknown) => Promise<AdminExtensionInstallResult>
  can?: (permission: string) => boolean
} = {}) {
  const calls: Array<{ url: string, sha256: string, config: unknown }> = []
  const toasts: Array<{ title: string, description?: string }> = []
  const navigations: string[] = []
  const hostInstall = useAdminPageHostInstall({
    extensionId: () => 'sforum.store',
    can: options.can ?? (() => true),
    isSuperAdmin: () => true,
    install: (url, sha256, config) => {
      calls.push({ url, sha256, config })
      return options.install
        ? options.install(url, sha256, config)
        : Promise.resolve(installResult())
    },
    toResult: toAdminHostInstallResult,
    t: key => key,
    notify: input => toasts.push(input),
    navigate: path => { navigations.push(path) }
  })
  return { hostInstall, calls, toasts, navigations }
}

describe('host install request validation', () => {
  test('normalizes a declared identity before the request leaves the page', () => {
    const validation = validateAdminHostInstallRequest({
      url: ' https://cdn.example.com/pkg.zip ',
      sha256: digest.toUpperCase(),
      expectedExtensionId: ' SForum.Store ',
      expectedType: 'Plugin',
      expectedVersion: '1.4.0'
    })
    expect(validation.ok).toBeTrue()
    if (!validation.ok) return
    expect(validation.request.expectedExtensionId).toBe('sforum.store')
    expect(validation.request.expectedType).toBe('plugin')
    expect(validation.request.sha256).toBe(digest)
    expect(validation.request.expectedVersion).toBe('1.4.0')
  })

  test('rejects every field that would make the install unbindable', () => {
    const cases: Array<[Record<string, unknown>, string]> = [
      [{ url: 'http://cdn.example.com/pkg.zip' }, 'url'],
      [{ url: 'https://user:pw@cdn.example.com/pkg.zip' }, 'url'],
      [{ sha256: '' }, 'sha256'],
      [{ sha256: 'not-a-digest' }, 'sha256'],
      [{ expectedExtensionId: 'Not An ID' }, 'extensionId'],
      [{ expectedType: 'widget' }, 'type'],
      [{ expectedVersion: '1.0.0/../x' }, 'version']
    ]
    for (const [overrides, reason] of cases) {
      const validation = validateAdminHostInstallRequest(validRequest(overrides))
      expect(validation.ok).toBeFalse()
      expect(validation.ok === false && validation.reason).toBe(reason)
    }
  })
})

describe('host action declaration', () => {
  test('only a component page can carry actions', () => {
    const componentPage = { view: 'component' as const, hostActions: [' Extension.Install ', 'extension.install'] }
    expect(adminPageHostActions(componentPage)).toEqual(['extension.install'])
    expect(adminPageCanInstallExtensions(componentPage)).toBeTrue()
    expect(adminPageDeclaresHostAction({ view: 'about', hostActions: ['extension.install'] }, 'extension.install')).toBeFalse()
    expect(adminPageCanInstallExtensions({ view: 'component', hostActions: ['other.action'] })).toBeFalse()
    expect(adminPageCanInstallExtensions(null)).toBeFalse()
  })

  test('the extension-level predicate reads the same manifest pages the bridge sees', () => {
    const declaring = {
      id: 'sforum.store',
      manifest: {
        admin: {
          pages: [{
            path: '/catalog',
            label: 'Catalog',
            view: 'component',
            component: { id: 'catalog', apiVersion: 1, entry: 'frontend/admin/dist/catalog.mjs' },
            hostActions: ['extension.install']
          }]
        }
      }
    } as never
    const plain = {
      id: 'plain.plugin',
      manifest: {
        admin: {
          pages: [{ path: '/about', label: 'About', view: 'component', component: { id: 'x', apiVersion: 1, entry: 'frontend/admin/dist/x.mjs' } }]
        }
      }
    } as never
    expect(extensionCanRequestInstall(declaring)).toBeTrue()
    expect(extensionCanRequestInstall(plain)).toBeFalse()
    expect(extensionCanRequestInstall(null)).toBeFalse()
  })

  test('audit label is derived from the owning extension and bounded', () => {
    expect(adminPageSourceLabel('SForum.Store')).toBe('admin-page:sforum.store')
    expect(adminPageSourceLabel('a'.repeat(120)).length).toBe(64)
    expect(adminPageSourceLabel('a'.repeat(120)).endsWith(':')).toBeFalse()
  })
})

describe('shared install error mapping', () => {
  test('one backend reason maps to one message for both surfaces', () => {
    const error = { code: 422, message: 'extension.expected_mismatch', data: { reason: 'extension.expected_mismatch' } }
    const bridged = adminExtensionInstallErrorMessage(key => key, error, 'admin.extensionInstall.hostAction.failedTitle')
    const paneled = adminExtensionInstallErrorMessage(key => key, error, 'admin.extensionInstall.url.failed')
    expect(bridged).toBe('admin.extensionInstall.errors.expectedMismatch')
    expect(paneled).toBe(bridged)
  })

  test('unknown reasons fall back per surface instead of leaking transport text', () => {
    const error = { code: 422, message: 'internal_error', data: { reason: 'internal_error' } }
    expect(adminExtensionInstallErrorMessage(key => key, error, 'fallback.key')).toBe('fallback.key')
    expect(adminExtensionInstallErrorMessage(key => key, new Error('boom'), 'fallback.key')).toBe('fallback.key')
  })
})

describe('bridge install state machine', () => {
  test('confirm performs one 120 s install carrying the declared identity', async () => {
    const { hostInstall, calls, toasts } = harness()
    const pending = hostInstall.start(validRequest())
    expect(hostInstall.state.value).toBe('confirming')
    expect(hostInstall.dialogOpen.value).toBeTrue()
    await hostInstall.confirm()
    const installed = await pending

    expect(calls).toHaveLength(1)
    expect(calls[0]?.config).toEqual({
      expectations: {
        expectedExtensionId: 'sforum.store',
        expectedType: 'plugin',
        source: 'admin-page:sforum.store'
      },
      timeout: 120000
    })
    expect(ADMIN_HOST_INSTALL_TIMEOUT_MS).toBe(120000)
    expect(installed.extensionId).toBe('sforum.store')
    expect(installed.requiresTrust).toBeTrue()
    expect(hostInstall.state.value).toBe('success')
    expect(toasts).toHaveLength(1)
  })

  test('cancel sends nothing and rejects with install_cancelled', async () => {
    const { hostInstall, calls } = harness()
    const pending = hostInstall.start(validRequest())
    hostInstall.close()
    await expect(pending).rejects.toMatchObject({ code: 'install_cancelled' })
    expect(calls).toHaveLength(0)
    expect(hostInstall.state.value).toBe('idle')
    expect(hostInstall.dialogOpen.value).toBeFalse()
  })

  test('single flight rejects a second install while one is confirming', async () => {
    const { hostInstall } = harness()
    void hostInstall.start(validRequest())
    await expect(hostInstall.start(validRequest({ expectedExtensionId: 'other.plugin' })))
      .rejects.toMatchObject({ code: 'install_in_progress' })
  })

  test('permission pre-check rejects before any dialog', async () => {
    const { hostInstall } = harness({ can: permission => permission !== 'extension.plugin.manage' })
    await expect(hostInstall.start(validRequest())).rejects.toMatchObject({ code: 'install_permission_denied' })
    expect(hostInstall.dialogOpen.value).toBeFalse()
    expect(hostInstall.state.value).toBe('idle')
  })

  test('an invalid request is rejected before the network', async () => {
    const { hostInstall, calls } = harness()
    await expect(hostInstall.start(validRequest({ expectedType: 'widget' })))
      .rejects.toMatchObject({ code: 'install_invalid_request' })
    expect(calls).toHaveLength(0)
  })

  test('a failure keeps the dialog open for retry, and close settles it', async () => {
    let attempt = 0
    const { hostInstall, calls } = harness({
      install: async () => {
        attempt += 1
        if (attempt === 1) throw { code: 422, message: 'extension.expected_mismatch', data: { reason: 'extension.expected_mismatch' } }
        return installResult()
      }
    })
    const pending = hostInstall.start(validRequest())
    await hostInstall.confirm()

    expect(hostInstall.state.value).toBe('error')
    expect(hostInstall.errorMessage.value).toBe('admin.extensionInstall.errors.expectedMismatch')
    hostInstall.retry()
    expect(hostInstall.state.value).toBe('confirming')
    await hostInstall.confirm()
    await expect(pending).resolves.toMatchObject({ extensionId: 'sforum.store' })
    expect(calls).toHaveLength(2)
  })

  test('closing after a failure rejects with install_failed', async () => {
    const { hostInstall } = harness({ install: async () => { throw new Error('boom') } })
    const pending = hostInstall.start(validRequest())
    await hostInstall.confirm()
    hostInstall.close()
    await expect(pending).rejects.toMatchObject({ code: 'install_failed' })
  })

  test('teardown aborts a pending install', async () => {
    const { hostInstall } = harness()
    const pending = hostInstall.start(validRequest())
    hostInstall.abortOnUnmount()
    await expect(pending).rejects.toMatchObject({ code: 'install_cancelled' })
  })

  test('success navigates to the matching management page', async () => {
    const { hostInstall, navigations } = harness()
    void hostInstall.start(validRequest())
    await hostInstall.confirm()
    await hostInstall.openManagement()
    expect(navigations).toEqual(['/extensions/plugins'])
    expect(hostInstall.state.value).toBe('idle')
  })

  test('errors are typed so plugins can branch on code', () => {
    const error = new AdminHostInstallError('install_permission_denied')
    expect(error).toBeInstanceOf(Error)
    expect(error.name).toBe('AdminHostInstallError')
    expect(error.code).toBe('install_permission_denied')
  })
})

describe('install result normalization', () => {
  test('prefers the staged candidate identity', () => {
    const result = toAdminHostInstallResult({
      ...installResult(),
      upgraded: true,
      extension: {
        ...installResult().extension,
        stagedVersion: {
          version: '1.5.0',
          packageDigest: 'c'.repeat(64),
          manifest: { backend: { entry: 'backend/plugin' } }
        } as never
      }
    })
    expect(result.version).toBe('1.5.0')
    expect(result.packageDigest).toBe('c'.repeat(64))
    expect(result.upgraded).toBeTrue()
    expect(result.requiresTrust).toBeTrue()
  })

  test('a front-end-only package does not claim a trust requirement', () => {
    const result = toAdminHostInstallResult(installResult({ manifest: {} as never }))
    expect(result.requiresTrust).toBeFalse()
  })
})

// ref() 仅用于让本文件在任何测试顺序下都先加载 Vue；行为断言不依赖它。
void ref
