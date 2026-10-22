import type { AdminExtensionInstallResult } from '~/utils/admin/adminExtensions'
import {
  EXTENSION_INSTALL_FROM_URL_PATH,
  EXTENSION_INSTALL_PATH
} from '~/utils/admin/extensions/extensionInstall'

/**
 * Thin transport boundary for the admin install page. Page state stays in the
 * feature component so the same calls can be used by a future async installer.
 */
export const useAdminExtensionInstall = () => {
  const { request } = useApiClient()

  function installArchive(file: File) {
    const form = new FormData()
    form.append('file', file)
    return request<AdminExtensionInstallResult>(EXTENSION_INSTALL_PATH, {
      method: 'POST',
      body: form
    })
  }

  function installFromUrl(url: string, sha256 = '') {
    const body: Record<string, unknown> = { url }
    if (sha256) body.sha256 = sha256
    return request<AdminExtensionInstallResult>(EXTENSION_INSTALL_FROM_URL_PATH, {
      method: 'POST',
      body
    })
  }

  return { installArchive, installFromUrl }
}
