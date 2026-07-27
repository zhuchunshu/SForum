import type { AdminExtension } from '~/utils/admin/adminExtensions'
import type { ExecutableTrustStatus } from '~/utils/extensions/extensionTrust'
import { apiErrorReason } from '~/composables/useApiClient'

export type AdminFrontendStatus = {
  extensionId: string
  kind: 'none' | 'prebuilt_component'
	trustState: string
	digest?: string
	component?: { id: string, apiVersion: number, entry: string, css?: string }
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
  const exactTrustManaged = ref(false)
  const exactTrustStatus = ref<ExecutableTrustStatus | null>(null)
  const error = ref('')
  const busy = ref(false)

  async function load() {
    error.value = ''
    exactTrustManaged.value = false
    exactTrustStatus.value = null
    try {
      status.value = await request(`/admin/extensions/${extension.value.id}/frontend`)
    } catch (cause) {
      error.value = `${cause}`
      return
    }
    try {
      exactTrustStatus.value = await request<ExecutableTrustStatus>(`/admin/extensions/${extension.value.id}/trust`)
      exactTrustManaged.value = true
    } catch (cause) {
      if (apiErrorReason(cause) !== 'extension.trust_not_required') {
        // 只把 V3 开关关闭视为旧模式；其他错误保留 frontend 状态与 Schema fallback。
        exactTrustManaged.value = true
        exactTrustStatus.value = null
      }
    }
  }

  async function mutate(action: 'grant' | 'revoke', confirmation?: AdminFrontendTrustConfirmation) {
    if (!status.value) return
    busy.value = true
    error.value = ''
    try {
		const nextStatus = action === 'grant'
			? await request<AdminFrontendStatus>(`/admin/extensions/${extension.value.id}/frontend/trust`, {
					method: 'POST',
					body: { digest: status.value.digest, confirmation }
				})
			: await request<AdminFrontendStatus>(`/admin/extensions/${extension.value.id}/frontend/trust`, { method: 'DELETE' })
		status.value = nextStatus
		toast.add({ color: 'success', icon: 'i-lucide-shield-check', title: t(`admin.extensions.frontend.${action}Success`), duration: 10000 })
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

  watch(
    () => [extension.value.id, extension.value.status, extension.value.packageDigest, extension.value.adminFrontendDigest],
    load,
    { immediate: true }
  )
  return { status, exactTrustManaged, exactTrustStatus, error, busy, load, mutate, challenge }
}
