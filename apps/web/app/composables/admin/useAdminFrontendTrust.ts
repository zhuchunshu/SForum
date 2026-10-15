import type { AdminExtension, AdminFrontendStatus } from '~/utils/admin/adminExtensions'
import type { ExecutableTrustStatus } from '~/utils/extensions/extensionTrust'
import { apiErrorReason } from '~/composables/useApiClient'

export type { AdminFrontendStatus }

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

export type AdminFrontendTrustOptions = {
  /**
   * 服务端 page-bootstrap 已解析出的信任结论。
   * 管理端页面是 SSR + 禁缓存，没有它时 status 只能是 null，而 null 会被
   * "!trusted" 渲染成"自定义组件尚未获得信任"，先闪一条警告条再换成真实组件。
   */
  initialStatus?: Ref<AdminFrontendStatus | null | undefined>
}

export function useAdminFrontendTrust(extension: Ref<AdminExtension>, options: AdminFrontendTrustOptions = {}) {
  const { request } = useApiClient()
  const { t } = useI18n()
  const toast = useToast()
  const status = ref<AdminFrontendStatus | null>(null)
  const exactTrustManaged = ref(false)
  const exactTrustStatus = ref<ExecutableTrustStatus | null>(null)
  // resolved 把"尚未读到状态"与"已读到且未授权"分开：只有后者才允许展示授权警告。
  const resolved = ref(false)
  const error = ref('')
  const busy = ref(false)
  let generation = 0
  // bootstrap 初值按 (扩展, 前端 digest) 只消费一次；此后同一实例的 load 一律回源，
  // 否则管理员授权后仍会读到 bootstrap 里的旧结论。
  let consumedSeedKey = ''

  function identityKey(item: AdminExtension) {
    return `${item.id}:${item.adminFrontendDigest || ''}`
  }

  // 初值必须属于当前扩展与当前前端 digest，避免 SPA 导航时把上一个扩展的结论套到新扩展上。
  function seededStatus(item: AdminExtension): AdminFrontendStatus | null {
    const next = options.initialStatus?.value
    if (!next || next.extensionId !== item.id) return null
    if (next.digest && item.adminFrontendDigest && next.digest !== item.adminFrontendDigest) return null
    return next
  }

  // 只决定授权 UI 形态（V3 exact-artifact 还是旧授权表），不参与 trusted 判定，
  // 因此不阻塞首屏，也不放进 load 的关键路径。
  async function loadExactTrust(item: AdminExtension, current: number) {
    try {
      const next = await request<ExecutableTrustStatus>(`/admin/extensions/${item.id}/trust`)
      if (current !== generation) return
      exactTrustStatus.value = next
      exactTrustManaged.value = true
    } catch (cause) {
      if (current !== generation) return
      // 只把 V3 开关关闭视为旧模式；其他错误保留 frontend 状态与 Schema fallback。
      if (apiErrorReason(cause) !== 'extension.trust_not_required') {
        exactTrustManaged.value = true
        exactTrustStatus.value = null
      }
    }
  }

  // force 供授权/撤销后与重试入口使用：必须回源，不能复用 bootstrap 初值。
  async function load(force = false) {
    const item = extension.value
    const current = ++generation
    status.value = null
    resolved.value = false
    error.value = ''
    exactTrustManaged.value = false
    exactTrustStatus.value = null

    const key = identityKey(item)
    if (!force && consumedSeedKey !== key) {
      const seed = seededStatus(item)
      if (seed) {
        consumedSeedKey = key
        status.value = seed
        resolved.value = true
        void loadExactTrust(item, current)
        return
      }
    }

    try {
      const next = await request<AdminFrontendStatus>(`/admin/extensions/${item.id}/frontend`)
      if (current !== generation) return
      status.value = next
    } catch (cause) {
      if (current !== generation) return
      error.value = `${cause}`
      return
    } finally {
      if (current === generation) resolved.value = true
    }
    void loadExactTrust(item, current)
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
		resolved.value = true
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
    () => void load(),
    { immediate: true }
  )
  return { status, resolved, exactTrustManaged, exactTrustStatus, error, busy, load, mutate, challenge }
}
