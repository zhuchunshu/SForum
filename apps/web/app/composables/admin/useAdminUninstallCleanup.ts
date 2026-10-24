import { ref, watch, type Ref } from 'vue'
import { apiErrorDetails, apiErrorMessage } from '~/composables/useApiClient'
import {
  isLifecycleV2Plugin,
  RECOMMENDED_LIFECYCLE_REMOVAL_MODE,
  type AdminExtension,
  type AdminLifecycleRemovalMode
} from '~/utils/admin/adminExtensions'

// useAdminUninstallCleanup 是卸载确认与清理恢复的聚焦协作对象：幂等键的
// 生成/持久化、确认弹窗状态、503 结构化失败详情与免键收尾入口都收在这里，
// 管理器组合式函数只做装配与列表投影。
export function useAdminUninstallCleanup(deps: {
  t: (key: string, params?: Record<string, unknown>) => string
  toast: ReturnType<typeof useToast>
  request: ReturnType<typeof useApiClient>['request']
  busyId: Ref<string>
  refresh: () => Promise<unknown>
  /** 卸载成功后的列表投影回调；由管理器维护 data/selectedId。 */
  onUninstalled: (item: AdminExtension) => void
}) {
  const { t, toast, request, busyId, refresh, onUninstalled } = deps
  const uninstallConfirmOpen = ref(false)
  const uninstallConfirmItem = ref<AdminExtension | null>(null)
  const uninstallRemovalMode = ref<AdminLifecycleRemovalMode>(RECOMMENDED_LIFECYCLE_REMOVAL_MODE)
  const uninstallError = ref('')
  const uninstallKeys = new Map<string, string>()
  const uninstallCleanupDetails = ref<Record<string, unknown> | null>(null)
  const finalizeCleanupBusyId = ref('')

  function storedUninstallKey(extensionId: string): string {
    try {
      return globalThis.sessionStorage.getItem(`${UNINSTALL_KEY_STORAGE_PREFIX}${extensionId}`) || ''
    } catch {
      return ''
    }
  }

  function persistUninstallKey(extensionId: string, key: string) {
    try {
      globalThis.sessionStorage.setItem(`${UNINSTALL_KEY_STORAGE_PREFIX}${extensionId}`, key)
    } catch {
      // 隐私模式等 storage 不可用场景静默降级为纯内存键。
    }
  }

  function forgetUninstallKey(extensionId: string) {
    uninstallKeys.delete(extensionId)
    try {
      globalThis.sessionStorage.removeItem(`${UNINSTALL_KEY_STORAGE_PREFIX}${extensionId}`)
    } catch {
      // 同上：storage 异常不影响卸载流程本身。
    }
  }

  function openUninstallExtension(item: AdminExtension) {
    if (!item.isDeletable || item.source === 'builtin' || item.isSystem) {
      return
    }
    uninstallConfirmItem.value = item
    uninstallRemovalMode.value = RECOMMENDED_LIFECYCLE_REMOVAL_MODE
    uninstallError.value = ''
    uninstallCleanupDetails.value = null
    if (!uninstallKeys.has(item.id)) {
      const stored = storedUninstallKey(item.id)
      uninstallKeys.set(item.id, stored || lifecycleIdempotencyKey())
      persistUninstallKey(item.id, uninstallKeys.get(item.id) as string)
    }
    uninstallConfirmOpen.value = true
  }

  function cancelUninstallExtension() {
    uninstallConfirmOpen.value = false
    uninstallConfirmItem.value = null
    uninstallRemovalMode.value = RECOMMENDED_LIFECYCLE_REMOVAL_MODE
    uninstallError.value = ''
    uninstallCleanupDetails.value = null
  }

  async function confirmUninstallExtension() {
    const item = uninstallConfirmItem.value
    if (!item) {
      return
    }
    busyId.value = item.id
    uninstallError.value = ''
    try {
      await request(`/admin/extensions/${item.id}`, {
        method: 'DELETE',
        body: isLifecycleV2Plugin(item) ? { removalMode: uninstallRemovalMode.value } : {},
        headers: { 'Idempotency-Key': uninstallKeys.get(item.id) || lifecycleIdempotencyKey() }
      })
      onUninstalled(item)
      toast.add({
        color: 'success',
        icon: 'i-lucide-trash-2',
        title: t('admin.extensions.uninstalled'),
        duration: 5000
      })
      forgetUninstallKey(item.id)
      uninstallCleanupDetails.value = null
      cancelUninstallExtension()
    } catch (error) {
      uninstallError.value = apiErrorMessage(error) || t('admin.extensions.actionFailed')
      uninstallCleanupDetails.value = apiErrorDetails(error) || null
      toast.add({
        color: 'error',
        icon: 'i-lucide-triangle-alert',
        title: uninstallError.value,
        duration: 0
      })
    } finally {
      busyId.value = ''
    }
  }

  // 免键恢复入口：原幂等键已丢失（如刷新后重开确认框）时，按扩展维度直接
  // 收尾该扩展全部 pending tombstone；端点幂等，已清理完成时同样返回成功。
  async function finalizeUninstallCleanup(item: AdminExtension) {
    finalizeCleanupBusyId.value = item.id
    try {
      await request(`/admin/extensions/${item.id}/uninstall-cleanup/finalize`, { method: 'POST' })
      await refresh()
      toast.add({
        color: 'success',
        icon: 'i-lucide-check-circle-2',
        title: t('admin.extensions.uninstallCleanup.finalizeSuccess'),
        duration: 5000
      })
      return true
    } catch (error) {
      toast.add({
        color: 'error',
        icon: 'i-lucide-triangle-alert',
        title: apiErrorMessage(error) || t('admin.extensions.actionFailed'),
        duration: 0
      })
      return false
    } finally {
      finalizeCleanupBusyId.value = ''
    }
  }

  watch([uninstallConfirmOpen, busyId], ([open, busy]) => {
    if (!open && busy !== uninstallConfirmItem.value?.id) {
      uninstallConfirmItem.value = null
      uninstallRemovalMode.value = RECOMMENDED_LIFECYCLE_REMOVAL_MODE
      uninstallError.value = ''
      uninstallCleanupDetails.value = null
    }
  })

  return {
    openUninstallExtension,
    confirmUninstallExtension,
    cancelUninstallExtension,
    finalizeUninstallCleanup,
    uninstallConfirmOpen,
    uninstallConfirmItem,
    uninstallRemovalMode,
    uninstallError,
    uninstallCleanupDetails,
    finalizeCleanupBusyId
  }
}

export function lifecycleIdempotencyKey() {
  return globalThis.crypto.randomUUID()
}

const UNINSTALL_KEY_STORAGE_PREFIX = 'sforum:uninstall-key:'
