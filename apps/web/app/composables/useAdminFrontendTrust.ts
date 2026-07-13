import type { AdminExtension } from '~/utils/adminExtensions'

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

  watch(() => extension.value.id, load, { immediate: true })
  return { status, error, busy, load, mutate, challenge }
}
