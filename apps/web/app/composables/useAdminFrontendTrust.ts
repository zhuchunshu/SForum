import type { AdminExtension, AdminExtensionOperation } from '~/utils/adminExtensions'

export type AdminFrontendStatus = {
  extensionId: string
  trustState: string
  digest?: string
  declaration?: { root: string, apiVersion: number, components: Record<string, string>, locales: Record<string, string> }
  dependencies?: { direct?: Array<{ name: string, version: string }>, resolved?: Array<{ name: string, version: string }> }
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

  async function mutate(action: 'grant' | 'revoke') {
    if (!status.value) return
    busy.value = true
    error.value = ''
    try {
      const operation = action === 'grant'
        ? await request<AdminExtensionOperation>(`/admin/extensions/${extension.value.id}/frontend/trust`, { method: 'POST', body: { packageDigest: status.value.digest } })
        : await request<AdminExtensionOperation>(`/admin/extensions/${extension.value.id}/frontend/trust`, { method: 'DELETE' })
      toast.add({ color: 'success', icon: 'i-lucide-shield-check', title: t(`admin.extensions.frontend.${action}Success`), duration: 10000 })
      if (operation.frontend) status.value = operation.frontend as AdminFrontendStatus
      else await load()
    } catch (cause) {
      error.value = `${cause}`
      toast.add({ color: 'error', icon: 'i-lucide-triangle-alert', title: error.value, duration: 0 })
    } finally {
      busy.value = false
    }
  }

  watch(() => extension.value.id, load, { immediate: true })
  return { status, error, busy, load, mutate }
}
