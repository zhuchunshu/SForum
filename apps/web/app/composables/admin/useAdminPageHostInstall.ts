import { ref, computed, watch } from 'vue'
import type { AdminExtensionInstallResult } from '~/utils/admin/adminExtensions'
import {
  ADMIN_HOST_INSTALL_TIMEOUT_MS,
  AdminHostInstallError,
  adminPageSourceLabel,
  adminExtensionInstallErrorMessage,
  validateAdminHostInstallRequest
} from '~/utils/admin/extensions/hostActions'
import type {
  AdminHostInstallFailureCode,
  AdminHostInstallRequest,
  AdminHostInstallResult,
  AdminPageInstallState
} from '~/utils/admin/extensions/hostActions'
import type { AdminInstallFromUrlOptions } from '~/composables/admin/useAdminExtensionInstall'

export type AdminPageHostInstallDeps = {
  /** Owning extension id; used for the audit source label. */
  extensionId: () => string
  /** Permission checker; a gate only — the API policy check stays authoritative. */
  can: (permission: string) => boolean
  isSuperAdmin: () => boolean
  /** Shared install transport (`useAdminExtensionInstall().installFromUrl`). */
  install: (
    url: string,
    sha256: string,
    options?: AdminInstallFromUrlOptions
  ) => Promise<AdminExtensionInstallResult>
  /** Host result normalizer, injected so tests can observe the raw response. */
  toResult: (result: AdminExtensionInstallResult) => AdminHostInstallResult
  t: (key: string) => string
  notify: (input: { title: string, description?: string }) => void
  navigate: (path: string) => Promise<unknown> | unknown
}

/**
 * State machine behind `bridge.installExtension`.
 *
 * Contract: the returned promise settles when the package is staged or the
 * caller gives up (cancel/close). A failed attempt keeps the dialog open so the
 * operator can retry; only success, cancel, or close settle it. One dialog and
 * one install at a time per page (`install_in_progress`).
 */
export function useAdminPageHostInstall(deps: AdminPageHostInstallDeps) {
  const dialogOpen = ref(false)
  const request = ref<AdminHostInstallRequest | null>(null)
  const state = ref<AdminPageInstallState>('idle')
  const result = ref<AdminHostInstallResult | null>(null)
  const errorMessage = ref('')
  let settler: {
    resolve: (value: AdminHostInstallResult) => void
    reject: (reason: AdminHostInstallError) => void
  } | null = null

  const permission = computed(() => request.value?.expectedType === 'theme'
    ? 'extension.theme.manage'
    : 'extension.plugin.manage')
  const busy = computed(() => state.value === 'installing')

  function settle(error: AdminHostInstallError | null, value?: AdminHostInstallResult) {
    const current = settler
    settler = null
    if (!current) return
    if (error) current.reject(error)
    else if (value) current.resolve(value)
  }

  function reset() {
    dialogOpen.value = false
    request.value = null
    state.value = 'idle'
    result.value = null
    errorMessage.value = ''
  }

  function start(input: AdminHostInstallRequest) {
    return new Promise<AdminHostInstallResult>((resolve, reject) => {
      if (state.value === 'confirming' || state.value === 'installing') {
        reject(new AdminHostInstallError('install_in_progress'))
        return
      }
      const validation = validateAdminHostInstallRequest(input)
      if (!validation.ok) {
        reject(new AdminHostInstallError('install_invalid_request', `invalid ${validation.reason}`))
        return
      }
      const next = validation.request
      const required = next.expectedType === 'theme' ? 'extension.theme.manage' : 'extension.plugin.manage'
      if (!deps.can(required)) {
        reject(new AdminHostInstallError('install_permission_denied'))
        return
      }
      settler = { resolve, reject }
      request.value = next
      result.value = null
      errorMessage.value = ''
      // 确认框本身就是确认动作，因此不要求调用瞬间处于用户手势。
      state.value = 'confirming'
      dialogOpen.value = true
    })
  }

  // 外部关闭（ESC/父级重置）等同于取消：promise 不能因为对话框消失而永远悬着。
  watch(dialogOpen, visible => {
    if (visible || state.value === 'idle' || state.value === 'installing') return
    const code = cancelCode()
    const detail = errorMessage.value
    reset()
    settle(new AdminHostInstallError(code, detail || undefined))
  })

  function cancelCode(): AdminHostInstallFailureCode {
    return state.value === 'error' ? 'install_failed' : 'install_cancelled'
  }

  async function confirm() {
    const current = request.value
    if (!current || state.value !== 'confirming') return
    state.value = 'installing'
    errorMessage.value = ''
    try {
      const response = await deps.install(current.url, current.sha256, {
        expectations: {
          expectedExtensionId: current.expectedExtensionId,
          expectedType: current.expectedType,
          ...(current.expectedVersion ? { expectedVersion: current.expectedVersion } : {}),
          source: adminPageSourceLabel(deps.extensionId())
        },
        timeout: ADMIN_HOST_INSTALL_TIMEOUT_MS
      })
      const installed = deps.toResult(response)
      result.value = installed
      state.value = 'success'
      deps.notify({
        title: deps.t(response.upgraded
          ? 'admin.extensionInstall.result.upgradeStaged'
          : 'admin.extensionInstall.result.installed'),
        description: deps.t('admin.extensionInstall.hostAction.stagedOnlyBody')
      })
      settle(null, installed)
    } catch (error) {
      errorMessage.value = adminExtensionInstallErrorMessage(
        deps.t,
        error,
        'admin.extensionInstall.hostAction.failedTitle'
      )
      state.value = 'error'
    }
  }

  function close() {
    if (busy.value) return
    const code = cancelCode()
    const detail = errorMessage.value
    reset()
    settle(new AdminHostInstallError(code, detail || undefined))
  }

  function retry() {
    if (state.value !== 'error') return
    errorMessage.value = ''
    state.value = 'confirming'
  }

  async function openManagement() {
    const type = result.value?.type === 'theme' ? 'theme' : 'plugin'
    reset()
    await deps.navigate(type === 'theme' ? '/extensions/themes' : '/extensions/plugins')
  }

  /** Page teardown aborts the in-flight install; the caller retries. */
  function abortOnUnmount() {
    settle(new AdminHostInstallError('install_cancelled', 'admin page unmounted'))
    reset()
  }

  return {
    dialogOpen,
    request,
    state,
    result,
    errorMessage,
    permission,
    busy,
    isSuperAdmin: computed(() => deps.isSuperAdmin()),
    start,
    confirm,
    close,
    retry,
    openManagement,
    abortOnUnmount
  }
}
