import type { AdminExtension, AdminExtensionOperation } from '~/utils/adminExtensions'

export type AdminFrontendStatus = {
  extensionId: string
  kind: 'none' | 'legacy_web_release' | 'prebuilt_component'
  trustState: string
  digest?: string
  artifactActive?: boolean
  buildRequired?: boolean
  declaration?: { root: string, apiVersion: number, components: Record<string, string>, locales: Record<string, string> }
  component?: { id: string, apiVersion: number, entry: string, css?: string }
  dependencies?: { direct?: Array<{ name: string, version: string }>, resolved?: Array<{ name: string, version: string }> }
}

export type AdminFrontendTrustConfirmation = {
  challengeId: string
  code: string
  extensionId: string
  version: string
  digest: string
  apiVersion: number
  componentId: string
  phrase: string
  acknowledged: boolean
}

export type AdminFrontendTrustChallenge = Omit<AdminFrontendTrustConfirmation, 'phrase' | 'acknowledged'> & {
  expiresAt: string
}

export function useAdminFrontendTrust(extension: Ref<AdminExtension>) {
  const { request } = useApiClient()
  const { t } = useI18n()
  const toast = useToast()
  const status = ref<AdminFrontendStatus | null>(null)
  const error = ref('')
  const busy = ref(false)

  async function load() {
    error.value = ''
    try {
      status.value = await request(`/admin/extensions/${extension.value.id}/frontend`)
    } catch (cause) {
      error.value = `${cause}`
    }
  }

  async function mutate(action: 'grant' | 'revoke', confirmation?: AdminFrontendTrustConfirmation) {
    if (!status.value) return
    busy.value = true
    error.value = ''
    try {
      const operation = action === 'grant'
        ? await request<AdminExtensionOperation>(`/admin/extensions/${extension.value.id}/frontend/trust`, {
            method: 'POST',
            body: { packageDigest: status.value.digest, confirmation }
          })
        : await request<AdminExtensionOperation>(`/admin/extensions/${extension.value.id}/frontend/trust`, { method: 'DELETE' })
      if (operation.queued) {
        // 信任变更后会排队 Web Release，构建日志在「Web 发布」页查看。
        toast.add({
          color: 'info',
          icon: 'i-lucide-hourglass',
          title: t(action === 'grant' ? 'admin.extensions.frontend.grantQueued' : 'admin.extensions.frontend.revokeQueued'),
          description: operation.webRelease?.id
            ? t('admin.extensions.webReleaseQueuedHint', { id: operation.webRelease.id })
            : t('admin.extensions.webReleaseQueuedHintNoId'),
          duration: 10000
        })
      } else {
        toast.add({ color: 'success', icon: 'i-lucide-shield-check', title: t(`admin.extensions.frontend.${action}Success`), duration: 10000 })
      }
      if (operation.frontend) status.value = operation.frontend as AdminFrontendStatus
      else await load()
    } catch (cause) {
      error.value = `${cause}`
      toast.add({ color: 'error', icon: 'i-lucide-triangle-alert', title: error.value, duration: 0 })
    } finally {
      busy.value = false
    }
  }

  async function challenge() {
    error.value = ''
    try {
      return await request<AdminFrontendTrustChallenge>(`/admin/extensions/${extension.value.id}/frontend/confirmation`, {
        method: 'POST',
        body: {}
      })
    } catch (cause) {
      error.value = `${cause}`
      throw cause
    }
  }

  watch(() => extension.value.id, load, { immediate: true })
  return { status, error, busy, load, mutate, challenge }
}
