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
  type AdminExtensionVersion,
  type AdminExtensionStatus
} from '~/utils/adminExtensions'
import type {
  ExecutableTrustChallenge,
  ExecutableTrustStatus,
  ExtensionEnableTrustMode
} from '~/utils/extensionTrust'
import { apiErrorMessage, apiErrorReason } from '~/composables/useApiClient'

export const useAdminExtensionsManager = async () => {
  const { t } = useI18n()
  const toast = useToast()
  const { request } = useApiClient()
  const { user } = useAuthSession()

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
    activationPending?: boolean
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
          title: result.activationPending
            ? t('admin.extensions.upgradeStaged')
            : t('admin.extensions.upgraded'),
          description: result.activationPending
            ? t('admin.extensions.upgradeStagedHint')
            : result.trustRevoked
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

  // V3 优先读取服务端 canonical impact；开关关闭时才回落 F2.1 capability review。
  const enableConfirmOpen = ref(false)
  const enableConfirmItem = ref<AdminExtension | null>(null)
  const enableTrustMode = ref<ExtensionEnableTrustMode>('legacy')
  const enableTrustStatus = ref<ExecutableTrustStatus | null>(null)
  const enableTrustChallenge = ref<ExecutableTrustChallenge | null>(null)
  const enableTrustError = ref('')
  const enableTrustBusy = ref(false)
  const isSuperAdmin = computed(() => user.value?.roleKeys?.includes('super_admin') === true)

  function resetEnableTrust() {
    enableTrustStatus.value = null
    enableTrustChallenge.value = null
    enableTrustError.value = ''
    enableTrustBusy.value = false
  }

  function openLegacyEnable(item: AdminExtension) {
    enableTrustMode.value = 'legacy'
    enableConfirmItem.value = item
    enableConfirmOpen.value = true
  }

  async function enableExtension(item: AdminExtension) {
    resetEnableTrust()
    enableTrustBusy.value = true
    try {
      const status = await request<ExecutableTrustStatus>(`/admin/extensions/${item.id}/trust`)
      enableTrustMode.value = 'exact'
      enableTrustStatus.value = status
      if (!status.trustRequired) {
        await lifecycle(item, 'enable')
        return
      }
      enableConfirmItem.value = item
      enableConfirmOpen.value = true
    } catch (error) {
      if (apiErrorReason(error) === 'extension.trust_not_required') {
        // V3 migration gate is off; preserve the existing boolean confirmation flow.
        if (item.status !== 'enabled' && (item.capabilityGrants?.length ?? 0) > 0) {
          openLegacyEnable(item)
        } else {
          await lifecycle(item, 'enable', { confirmCapabilities: true })
        }
        return
      }
      enableTrustMode.value = 'exact'
      enableTrustError.value = apiErrorMessage(error) || t('admin.extensions.trust.previewFailed')
      enableConfirmItem.value = item
      enableConfirmOpen.value = true
    } finally {
      enableTrustBusy.value = false
    }
  }

  async function confirmEnableExtension() {
    const item = enableConfirmItem.value
    if (!item) {
      return
    }
    const body: Record<string, unknown> = enableTrustMode.value === 'exact'
      ? { confirmationToken: enableTrustChallenge.value?.token || undefined }
      : { confirmCapabilities: true }
    enableTrustError.value = ''
    const result = await lifecycle(item, 'enable', body)
    if (result.ok) {
      enableConfirmOpen.value = false
      // 保留 impact 到关闭过渡结束，避免弹窗在离场动画中闪成空内容；一次性 token 立即丢弃。
      enableTrustChallenge.value = null
      enableTrustError.value = ''
      enableTrustBusy.value = false
      return
    }
    enableTrustError.value = apiErrorMessage(result.error) || t('admin.extensions.actionFailed')
    enableTrustChallenge.value = null
    if (enableTrustMode.value === 'exact') {
      try {
        enableTrustStatus.value = await request<ExecutableTrustStatus>(`/admin/extensions/${item.id}/trust`)
      } catch {
        // Keep the actionable enable failure visible; the next challenge request rechecks impact.
      }
    }
  }

  async function issueEnableTrustChallenge() {
    const item = enableConfirmItem.value
    if (!item || !isSuperAdmin.value) return
    enableTrustBusy.value = true
    enableTrustError.value = ''
    try {
      const challenge = await request<ExecutableTrustChallenge>(`/admin/extensions/${item.id}/trust/challenge`, {
        method: 'POST',
        body: {}
      })
      enableTrustChallenge.value = challenge
      enableTrustStatus.value = {
        impact: challenge.impact,
        trustRequired: true,
        trusted: false
      }
    } catch (error) {
      enableTrustError.value = apiErrorMessage(error) || t('admin.extensions.trust.challengeFailed')
      toast.add({
        color: 'error',
        icon: 'i-lucide-triangle-alert',
        title: enableTrustError.value,
        duration: 0
      })
    } finally {
      enableTrustBusy.value = false
    }
  }

  function cancelEnableExtension() {
    enableConfirmOpen.value = false
    enableTrustChallenge.value = null
    enableTrustError.value = ''
    enableTrustBusy.value = false
  }

  async function disableExtension(item: AdminExtension) {
    await lifecycle(item, 'disable')
  }

  async function upgradeExtension(item: AdminExtension, confirmationToken?: string) {
    return lifecycle(item, 'upgrade', { confirmationToken })
  }

  async function rollbackExtension(item: AdminExtension, target: AdminExtensionVersion) {
    return lifecycle(item, 'rollback', {
      targetVersion: target.version,
      targetPackageDigest: target.packageDigest
    })
  }

  async function restartExtension(item: AdminExtension) {
	if (item.stagedVersion) {
		// Enabled + staged is an upgrade transaction, so it must use the same
		// exact-artifact challenge flow as enable instead of a blind restart.
		await enableExtension(item)
		return
	}
    busyId.value = item.id
	const idempotencyKey = lifecycleIdempotencyKey()
    try {
      // 已启用插件重启不要求再次确认 capabilities。
		const updated = await request<AdminExtension>(`/admin/extensions/${item.id}/enable`, {
			method: 'POST',
			body: { confirmCapabilities: true },
			headers: { 'Idempotency-Key': idempotencyKey }
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
    action: 'enable' | 'disable' | 'upgrade' | 'rollback',
    body: Record<string, unknown> = {}
  ) {
    busyId.value = item.id
	const idempotencyKey = lifecycleIdempotencyKey()
    try {
		const updated = await request<AdminExtension>(`/admin/extensions/${item.id}/${action}`, {
			method: 'POST',
			body,
			headers: { 'Idempotency-Key': idempotencyKey }
		})
		replaceExtension(updated)
		await loadEvents(updated.id)
		toast.add({
			color: 'success',
			icon: lifecycleSuccessIcon(action),
			title: t(lifecycleSuccessMessage(action)),
			duration: 10000
		})
		return { ok: true as const, updated }
    } catch (error) {
		toast.add({
			color: 'error',
			icon: 'i-lucide-triangle-alert',
			title: apiErrorMessage(error) || t('admin.extensions.actionFailed'),
			duration: 0
		})
		return { ok: false as const, error }
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
    issueEnableTrustChallenge,
    cancelEnableExtension,
    enableConfirmOpen,
    enableConfirmItem,
    enableTrustMode,
    enableTrustStatus,
    enableTrustChallenge,
    enableTrustError,
    enableTrustBusy,
    isSuperAdmin,
    openUninstallExtension,
    confirmUninstallExtension,
    cancelUninstallExtension,
    uninstallConfirmOpen,
    uninstallConfirmItem,
    disableExtension,
    upgradeExtension,
    rollbackExtension,
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

function lifecycleIdempotencyKey() {
  return globalThis.crypto.randomUUID()
}

function lifecycleSuccessIcon(action: 'enable' | 'disable' | 'upgrade' | 'rollback') {
  switch (action) {
    case 'enable': return 'i-lucide-play'
    case 'disable': return 'i-lucide-pause'
    case 'upgrade': return 'i-lucide-package-check'
    case 'rollback': return 'i-lucide-history'
  }
}

function lifecycleSuccessMessage(action: 'enable' | 'disable' | 'upgrade' | 'rollback') {
  switch (action) {
    case 'enable': return 'admin.extensions.enabled'
    case 'disable': return 'admin.extensions.disabled'
    case 'upgrade': return 'admin.extensions.upgraded'
    case 'rollback': return 'admin.extensions.rolledBack'
  }
}
