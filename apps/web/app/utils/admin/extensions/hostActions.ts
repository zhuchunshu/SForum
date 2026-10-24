/**
 * Host-action vocabulary shared by manifest disclosure, the admin-page bridge,
 * and the confirmation dialog. Keeping the constants and predicates in one file
 * means a page cannot "declare" an action with one spelling and have the bridge
 * look for another.
 */

import { apiErrorReason } from '~/composables/useApiClient'
import { extensionAdminPages } from '~/utils/admin/adminExtensions'
import type { AdminExtension, AdminExtensionAdminPage } from '~/utils/admin/adminExtensions'
import { normalizeRemoteArchiveUrl, normalizeSha256 } from './extensionInstall'

export const ADMIN_HOST_ACTION_EXTENSION_INSTALL = 'extension.install'

/**
 * The host-mediated install is a synchronous download+stage. `useApiClient`
 * defaults POST to 30 s, which would abort a slow but healthy download, so the
 * single call that performs the install passes this budget explicitly.
 */
export const ADMIN_HOST_INSTALL_TIMEOUT_MS = 120000

export type AdminHostInstallType = 'plugin' | 'theme'

/** Bridge-side state machine for one host-mediated install. */
export type AdminPageInstallState = 'idle' | 'confirming' | 'installing' | 'success' | 'error' | 'cancelled'

export type AdminHostInstallRequest = {
  url: string
  sha256: string
  expectedExtensionId: string
  expectedType: AdminHostInstallType
  expectedVersion?: string
}

export type AdminHostInstallResult = {
  extensionId: string
  version: string
  type: AdminHostInstallType
  packageDigest: string
  upgraded: boolean
  requiresTrust: boolean
}

/** Host actions a page declares, normalized to a deduplicated list. */
export function adminPageHostActions(page: Pick<AdminExtensionAdminPage, 'hostActions'> | null | undefined) {
  if (!page?.hostActions?.length) return [] as string[]
  const seen = new Set<string>()
  for (const action of page.hostActions) {
    const value = String(action || '').trim().toLowerCase()
    if (value) seen.add(value)
  }
  return [...seen]
}

/**
 * Host actions are only ever attached to `view: component` pages; the API
 * rejects any other combination, and the client mirrors that rule so a stale
 * bootstrap payload cannot inject the bridge on a plain about/settings page.
 */
export function adminPageDeclaresHostAction(
  page: Pick<AdminExtensionAdminPage, 'view' | 'hostActions'> | null | undefined,
  action: string
) {
  if (!page || page.view !== 'component') return false
  return adminPageHostActions(page).includes(action)
}

export function adminPageCanInstallExtensions(page: Pick<AdminExtensionAdminPage, 'view' | 'hostActions'> | null | undefined) {
  return adminPageDeclaresHostAction(page, ADMIN_HOST_ACTION_EXTENSION_INSTALL)
}

/**
 * True when any admin page of this extension declares the install action.
 * Disclosure UI and the bridge both read the manifest through this helper, so
 * they cannot disagree about what a plugin can ask for.
 */
export function extensionCanRequestInstall(item: AdminExtension | null | undefined) {
  if (!item) return false
  return extensionAdminPages(item).some(page => adminPageCanInstallExtensions(page))
}

export type InstallRequestValidation =
  | { ok: true, request: AdminHostInstallRequest }
  | { ok: false, reason: 'url' | 'sha256' | 'extensionId' | 'type' | 'version' }

/**
 * Validates a plugin-supplied install request before the Host spends a network
 * round trip on it. The Host still re-validates and re-checks the manifest
 * identity server-side; this is a fast, honest rejection, not the authority.
 */
export function validateAdminHostInstallRequest(input: unknown): InstallRequestValidation {
  const value = (input || {}) as Partial<AdminHostInstallRequest>
  const url = normalizeRemoteArchiveUrl(String(value.url ?? ''))
  if (!url.ok) return { ok: false, reason: 'url' }

  const digest = normalizeSha256(String(value.sha256 ?? ''))
  // The bridge requires a pinned digest: an unpinned download cannot be tied to
  // the artifact the catalog advertised.
  if (!digest.ok || !digest.value) return { ok: false, reason: 'sha256' }

  const extensionId = String(value.expectedExtensionId ?? '').trim().toLowerCase()
  if (!/^[a-z0-9][a-z0-9._-]{1,80}$/.test(extensionId)) return { ok: false, reason: 'extensionId' }

  const expectedType = String(value.expectedType ?? '').trim().toLowerCase()
  if (expectedType !== 'plugin' && expectedType !== 'theme') return { ok: false, reason: 'type' }

  const version = String(value.expectedVersion ?? '').trim()
  if (version && !/^[0-9A-Za-z._+-]{1,64}$/.test(version)) return { ok: false, reason: 'version' }

  return {
    ok: true,
    request: {
      url: url.url,
      sha256: digest.value,
      expectedExtensionId: extensionId,
      expectedType,
      ...(version ? { expectedVersion: version } : {})
    }
  }
}

/** Stable bridge failure codes. A plugin can branch on `error.code` without
 * parsing text; the text is only for humans. */
export type AdminHostInstallFailureCode =
  | 'install_permission_denied'
  | 'install_cancelled'
  | 'install_in_progress'
  | 'install_invalid_request'
  | 'install_failed'

export class AdminHostInstallError extends Error {
  readonly code: AdminHostInstallFailureCode

  constructor(code: AdminHostInstallFailureCode, message?: string) {
    super(message || code)
    this.name = 'AdminHostInstallError'
    this.code = code
  }
}

/** Audit label for a host-mediated install: `admin-page:<extension id>`. */
export function adminPageSourceLabel(extensionId: string) {
  const base = `admin-page:${String(extensionId || '').trim().toLowerCase()}`
  return base.length <= 64 ? base : base.slice(0, 64).replace(/[.:-]+$/, '')
}

/** Stable install failure reason → i18n key under `admin.extensionInstall.errors`. */
const INSTALL_ERROR_KEYS: Record<string, string> = {
  'extension.remote_url_invalid': 'remoteUrlInvalid',
  'extension.remote_fetch_failed': 'remoteFetchFailed',
  'extension.remote_archive_too_large': 'remoteArchiveTooLarge',
  'extension.remote_digest_mismatch': 'remoteDigestMismatch',
  'extension.remote_install_unavailable': 'installUnavailable',
  'extension.expected_mismatch': 'expectedMismatch',
  'extension.archive_invalid': 'archiveInvalid',
  'extension.manifest_invalid': 'manifestInvalid',
  'permission.denied': 'permissionDenied',
  'validation.invalid': 'invalidRequest'
}

/**
 * Maps a transport failure to the shared install error message. The admin
 * install panel and the bridge dialog both call this, so one backend code can
 * never grow two different explanations.
 */
export function adminExtensionInstallErrorMessage(
  t: (key: string) => string,
  error: unknown,
  fallbackKey: string
) {
  const reason = apiErrorReason(error)
  const mapped = reason ? INSTALL_ERROR_KEYS[reason] : ''
  if (mapped) return t(`admin.extensionInstall.errors.${mapped}`)
  return t(fallbackKey)
}
