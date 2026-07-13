import { computed, ref } from 'vue'
import {
  EXTENSION_DELIVERY_FETCH_LIMIT,
  EXTENSION_EVENT_FETCH_LIMIT,
  extensionStats,
  mergeExtensionDeliveries,
  mergeExtensionEvents,
  type AdminContributionPointDefinition,
  type AdminEffectiveContribution,
  type AdminExtension,
  type AdminExtensionEventDefinition,
  type AdminExtensionEventDelivery,
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
  const loadingEventDefinitions = ref(false)
  const loadingEventDeliveries = ref(false)
  const loadingContributionPoints = ref(false)
  const loadingContributions = ref(false)
  const eventsByExtension = ref<Record<string, AdminExtensionEvent[]>>({})
  const eventDefinitions = ref<AdminExtensionEventDefinition[]>([])
  const eventDeliveries = ref<AdminExtensionEventDelivery[]>([])
  const contributionPoints = ref<AdminContributionPointDefinition[]>([])
  const contributions = ref<AdminEffectiveContribution[]>([])

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
  const aggregatedDeliveries = computed(() => mergeExtensionDeliveries(eventDeliveries.value))

  function openUpload() {
    fileInput.value?.click()
  }

  type AdminExtensionInstallResult = {
    extension: AdminExtension
    upgraded?: boolean
    previousVersion?: string
    previousDigest?: string
    trustRevoked?: boolean
    requiredReEnable?: boolean
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
      // F2.4：响应为 InstallResult；兼容旧形态（直接 Extension）。
      const payload = await request<AdminExtensionInstallResult | AdminExtension>('/admin/extensions', {
        method: 'POST',
        body: form
      })
      const result = 'extension' in payload && payload.extension
        ? payload as AdminExtensionInstallResult
        : { extension: payload as AdminExtension, upgraded: false }
      selectedId.value = result.extension.id
      replaceExtension(result.extension)
      await refresh()
      if (result.upgraded) {
        toast.add({
          color: 'success',
          icon: 'i-lucide-package-plus',
          title: t('admin.extensions.upgraded'),
          description: result.trustRevoked
            ? t('admin.extensions.upgradedTrustRevokedHint')
            : result.requiredReEnable
              ? t('admin.extensions.upgradedReEnableHint')
              : undefined,
          duration: 10000
        })
      } else {
        toast.add({ color: 'success', icon: 'i-lucide-package-check', title: t('admin.extensions.uploaded') })
      }
    } catch (error) {
      toast.add({ color: 'error', icon: 'i-lucide-triangle-alert', title: apiErrorMessage(error) || t('admin.extensions.uploadFailed') })
    } finally {
      uploading.value = false
    }
  }

  // 卸载确认（F2.4）。
  const uninstallConfirmOpen = ref(false)
  const uninstallConfirmItem = ref<AdminExtension | null>(null)

  function openUninstallExtension(item: AdminExtension) {
    if (!item.isDeletable || item.source === 'builtin' || item.isSystem) {
      return
    }
    uninstallConfirmItem.value = item
    uninstallConfirmOpen.value = true
  }

  function cancelUninstallExtension() {
    uninstallConfirmOpen.value = false
    uninstallConfirmItem.value = null
  }

  async function confirmUninstallExtension() {
    const item = uninstallConfirmItem.value
    if (!item) {
      return
    }
    uninstallConfirmOpen.value = false
    uninstallConfirmItem.value = null
    busyId.value = item.id
    try {
      await request(`/admin/extensions/${item.id}`, {
        method: 'DELETE',
        body: {}
      })
      const current = extensions.value.filter(row => row.id !== item.id)
      data.value = current
      if (selectedId.value === item.id) {
        selectedId.value = current[0]?.id || ''
      }
      toast.add({
        color: 'success',
        icon: 'i-lucide-trash-2',
        title: t('admin.extensions.uninstalled'),
        duration: 10000
      })
    } catch (error) {
      toast.add({
        color: 'error',
        icon: 'i-lucide-triangle-alert',
        title: apiErrorMessage(error) || t('admin.extensions.actionFailed')
      })
    } finally {
      busyId.value = ''
    }
  }

  // 启用确认对话框状态（F2.1 capability review）。
  const enableConfirmOpen = ref(false)
  const enableConfirmItem = ref<AdminExtension | null>(null)

  async function enableExtension(item: AdminExtension) {
    // 首次启用且有能力列表时，弹出审查确认；已启用重启走 restart。
    if (item.status !== 'enabled' && (item.capabilityGrants?.length ?? 0) > 0) {
      enableConfirmItem.value = item
      enableConfirmOpen.value = true
      return
    }
    await lifecycle(item, 'enable', { confirmCapabilities: true })
  }

  async function confirmEnableExtension() {
    const item = enableConfirmItem.value
    if (!item) {
      return
    }
    enableConfirmOpen.value = false
    enableConfirmItem.value = null
    await lifecycle(item, 'enable', { confirmCapabilities: true })
  }

  function cancelEnableExtension() {
    enableConfirmOpen.value = false
    enableConfirmItem.value = null
  }

  async function disableExtension(item: AdminExtension) {
    await lifecycle(item, 'disable')
  }

  async function restartExtension(item: AdminExtension) {
    busyId.value = item.id
    try {
      // 已启用插件重启不要求再次确认 capabilities。
		const updated = await request<AdminExtension>(`/admin/extensions/${item.id}/enable`, {
			method: 'POST',
			body: { confirmCapabilities: true }
		})
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
		toast.add({
			color: 'success',
			icon: 'i-lucide-palette',
			title: t('admin.extensions.themeActivated'),
			duration: 10000
		})
    } catch (error) {
      toast.add({ color: 'error', icon: 'i-lucide-triangle-alert', title: apiErrorMessage(error) || t('admin.extensions.themeActivationUnavailable') })
    } finally {
      busyId.value = ''
    }
  }

  async function lifecycle(
    item: AdminExtension,
    action: 'enable' | 'disable',
    body: Record<string, unknown> = {}
  ) {
    busyId.value = item.id
    try {
		const updated = await request<AdminExtension>(`/admin/extensions/${item.id}/${action}`, {
			method: 'POST',
			body
		})
		replaceExtension(updated)
		await loadEvents(updated.id)
		toast.add({
			color: 'success',
			icon: action === 'enable' ? 'i-lucide-play' : 'i-lucide-pause',
			title: action === 'enable' ? t('admin.extensions.enabled') : t('admin.extensions.disabled'),
			duration: 10000
		})
    } catch (error) {
      toast.add({ color: 'error', icon: 'i-lucide-triangle-alert', title: apiErrorMessage(error) || t('admin.extensions.actionFailed') })
    } finally {
      busyId.value = ''
    }
  }

  async function loadEvents(id: string, limit = EXTENSION_EVENT_FETCH_LIMIT) {
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

  async function loadAllEvents(limit = EXTENSION_EVENT_FETCH_LIMIT) {
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

  async function loadEventDefinitions() {
    loadingEventDefinitions.value = true
    try {
      eventDefinitions.value = await request<AdminExtensionEventDefinition[]>('/admin/extensions/event-definitions')
      return eventDefinitions.value
    } catch {
      eventDefinitions.value = []
      return []
    } finally {
      loadingEventDefinitions.value = false
    }
  }

  async function loadEventDeliveries(limit = EXTENSION_DELIVERY_FETCH_LIMIT) {
    loadingEventDeliveries.value = true
    try {
      eventDeliveries.value = await request<AdminExtensionEventDelivery[]>(`/admin/extensions/event-deliveries?limit=${limit}`)
      return eventDeliveries.value
    } catch {
      eventDeliveries.value = []
      return []
    } finally {
      loadingEventDeliveries.value = false
    }
  }

  async function loadContributionPoints() {
    loadingContributionPoints.value = true
    try {
      contributionPoints.value = await request<AdminContributionPointDefinition[]>('/admin/extensions/contribution-points')
      return contributionPoints.value
    } catch {
      contributionPoints.value = []
      return []
    } finally {
      loadingContributionPoints.value = false
    }
  }

  async function loadContributions() {
    loadingContributions.value = true
    try {
      contributions.value = await request<AdminEffectiveContribution[]>('/admin/extensions/contributions')
      return contributions.value
    } catch {
      contributions.value = []
      return []
    } finally {
      loadingContributions.value = false
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
    loadingEventDefinitions,
    loadingEventDeliveries,
    loadingContributionPoints,
    loadingContributions,
    eventsByExtension,
    eventDefinitions,
    eventDeliveries,
    contributionPoints,
    contributions,
    aggregatedEvents,
    aggregatedDeliveries,
    stats,
    openUpload,
    uploadArchive,
    enableExtension,
    confirmEnableExtension,
    cancelEnableExtension,
    enableConfirmOpen,
    enableConfirmItem,
    openUninstallExtension,
    confirmUninstallExtension,
    cancelUninstallExtension,
    uninstallConfirmOpen,
    uninstallConfirmItem,
    disableExtension,
    restartExtension,
    verifyExtension,
    activateTheme,
    loadEvents,
    loadAllEvents,
    loadEventDefinitions,
    loadEventDeliveries,
    loadContributionPoints,
    loadContributions,
    statusColor,
    typeLabel,
    statusLabel
  }
}
