import { computed, ref } from 'vue'
import {
  extensionStats,
  mergeExtensionEvents,
  type AdminExtension,
  type AdminExtensionEvent,
  type AdminExtensionStatus
} from '~/utils/adminExtensions'
import { apiErrorMessage } from '~/composables/useApiClient'

export const useAdminExtensionsManager = async () => {
  const { t } = useI18n()
  const toast = useToast()
  const { request } = useApiClient()

  const fileInput = ref<HTMLInputElement | null>(null)
  const selectedId = ref('')
  const uploading = ref(false)
  const busyId = ref('')
  const loadingEvents = ref(false)
  const loadingAllEvents = ref(false)
  const eventsByExtension = ref<Record<string, AdminExtensionEvent[]>>({})

  const {
    data,
    pending,
    error,
    refresh
  } = await useAsyncData<AdminExtension[]>('admin-extensions', () => request<AdminExtension[]>('/admin/extensions'), {
    default: (): AdminExtension[] => []
  })

  const extensions = computed(() => data.value || [])
  const stats = computed(() => extensionStats(extensions.value))
  const selected = computed(() => extensions.value.find(item => item.id === selectedId.value) || extensions.value[0])
  const selectedEvents = computed(() => selected.value ? eventsByExtension.value[selected.value.id] || [] : [])
  const aggregatedEvents = computed(() => mergeExtensionEvents(eventsByExtension.value))

  function openUpload() {
    fileInput.value?.click()
  }

  async function uploadArchive(event: Event) {
    const input = event.target as HTMLInputElement
    const file = input.files?.[0]
    input.value = ''
    if (!file) {
      return
    }

    const form = new FormData()
    form.append('file', file)
    uploading.value = true
    try {
      const installed = await request<AdminExtension>('/admin/extensions', {
        method: 'POST',
        body: form
      })
      selectedId.value = installed.id
      await refresh()
      toast.add({ color: 'success', icon: 'i-lucide-package-check', title: t('admin.extensions.uploaded') })
    } catch (error) {
      toast.add({ color: 'error', icon: 'i-lucide-triangle-alert', title: apiErrorMessage(error) || t('admin.extensions.uploadFailed') })
    } finally {
      uploading.value = false
    }
  }

  async function enableExtension(item: AdminExtension) {
    await lifecycle(item, 'enable')
  }

  async function disableExtension(item: AdminExtension) {
    await lifecycle(item, 'disable')
  }

  async function restartExtension(item: AdminExtension) {
    busyId.value = item.id
    try {
      const updated = await request<AdminExtension>(`/admin/extensions/${item.id}/enable`, { method: 'POST', body: {} })
      replaceExtension(updated)
      await loadEvents(updated.id)
      toast.add({ color: 'success', icon: 'i-lucide-refresh-cw', title: t('admin.extensions.restarted') })
    } catch (error) {
      toast.add({ color: 'error', icon: 'i-lucide-triangle-alert', title: apiErrorMessage(error) || t('admin.extensions.actionFailed') })
    } finally {
      busyId.value = ''
    }
  }

  async function verifyExtension(item: AdminExtension) {
    busyId.value = item.id
    try {
      const verified = await request<AdminExtension>(`/admin/extensions/${item.id}/verify`, { method: 'POST', body: {} })
      replaceExtension(verified)
      await loadEvents(verified.id)
      toast.add({ color: 'success', icon: 'i-lucide-shield-check', title: t('admin.extensions.verified') })
    } catch (error) {
      toast.add({ color: 'error', icon: 'i-lucide-triangle-alert', title: apiErrorMessage(error) || t('admin.extensions.actionFailed') })
    } finally {
      busyId.value = ''
    }
  }

  async function activateTheme(item: AdminExtension) {
    busyId.value = item.id
    try {
      const activated = await request<AdminExtension>(`/admin/extensions/${item.id}/activate`, { method: 'POST', body: {} })
      replaceExtension(activated)
      await refresh()
      await loadEvents(activated.id)
      toast.add({ color: 'success', icon: 'i-lucide-palette', title: t('admin.extensions.themeActivated') })
    } catch (error) {
      toast.add({ color: 'error', icon: 'i-lucide-triangle-alert', title: apiErrorMessage(error) || t('admin.extensions.themeActivationUnavailable') })
    } finally {
      busyId.value = ''
    }
  }

  async function lifecycle(item: AdminExtension, action: 'enable' | 'disable') {
    busyId.value = item.id
    try {
      const updated = await request<AdminExtension>(`/admin/extensions/${item.id}/${action}`, { method: 'POST', body: {} })
      replaceExtension(updated)
      await loadEvents(updated.id)
      toast.add({
        color: 'success',
        icon: action === 'enable' ? 'i-lucide-play' : 'i-lucide-pause',
        title: action === 'enable' ? t('admin.extensions.enabled') : t('admin.extensions.disabled')
      })
    } catch (error) {
      toast.add({ color: 'error', icon: 'i-lucide-triangle-alert', title: apiErrorMessage(error) || t('admin.extensions.actionFailed') })
    } finally {
      busyId.value = ''
    }
  }

  async function loadEvents(id: string, limit = 20) {
    loadingEvents.value = true
    try {
      const items = await request<AdminExtensionEvent[]>(`/admin/extensions/${id}/events?limit=${limit}`)
      eventsByExtension.value = {
        ...eventsByExtension.value,
        [id]: items
      }
      return items
    } catch {
      eventsByExtension.value = {
        ...eventsByExtension.value,
        [id]: []
      }
      return []
    } finally {
      loadingEvents.value = false
    }
  }

  async function loadAllEvents(limit = 20) {
    loadingAllEvents.value = true
    try {
      const entries = await Promise.all(extensions.value.map(async (item) => {
        try {
          const items = await request<AdminExtensionEvent[]>(`/admin/extensions/${item.id}/events?limit=${limit}`)
          return [item.id, items] as const
        } catch {
          return [item.id, []] as const
        }
      }))
      eventsByExtension.value = Object.fromEntries(entries)
    } finally {
      loadingAllEvents.value = false
    }
  }

  function replaceExtension(updated: AdminExtension) {
    const current = extensions.value.slice()
    const index = current.findIndex(item => item.id === updated.id)
    if (index >= 0) {
      current[index] = updated
    } else {
      current.unshift(updated)
    }
    data.value = current
    selectedId.value = updated.id
  }

  function statusColor(status: AdminExtensionStatus) {
    if (status === 'enabled') {
      return 'success'
    }
    if (status === 'disabled') {
      return 'neutral'
    }
    return 'warning'
  }

  function typeLabel(type: AdminExtension['type']) {
    return type === 'theme' ? t('admin.extensions.types.theme') : t('admin.extensions.types.plugin')
  }

  function statusLabel(status: AdminExtensionStatus) {
    return t(`admin.extensions.status.${status}`)
  }

  return {
    extensions,
    pending,
    error,
    refresh,
    fileInput,
    selectedId,
    selected,
    selectedEvents,
    uploading,
    busyId,
    loadingEvents,
    loadingAllEvents,
    eventsByExtension,
    aggregatedEvents,
    stats,
    openUpload,
    uploadArchive,
    enableExtension,
    disableExtension,
    restartExtension,
    verifyExtension,
    activateTheme,
    loadEvents,
    loadAllEvents,
    statusColor,
    typeLabel,
    statusLabel
  }
}
