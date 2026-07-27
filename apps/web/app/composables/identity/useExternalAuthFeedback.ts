/**
 * 消费 Host 回调带回的 `ext_auth` query：展示反馈并清理地址栏，避免刷新重复提示。
 * 使用 useState 做会话内单次消费，登录壳与根布局可共享同一结果。
 */

import {
  resolveExternalAuthFeedback,
  stripExtAuthQuery,
  type ExternalAuthFeedback
} from '~/utils/identity/externalAuthFeedback'

export function useExternalAuthFeedback() {
  const route = useRoute()
  const router = useRouter()
  const toast = useToast()
  const { t, te } = useI18n()

  const feedback = useState<ExternalAuthFeedback | null>('ext-auth:feedback', () => null)
  const alertMessage = useState<string>('ext-auth:alert-message', () => '')
  const alertVariant = useState<'danger' | 'warning' | 'success' | 'info'>(
    'ext-auth:alert-variant',
    () => 'danger'
  )
  const consumedKey = useState<string>('ext-auth:consumed-key', () => '')

  function messageFor(item: ExternalAuthFeedback): string {
    if (te(item.messageKey)) {
      return t(item.messageKey)
    }
    return t('auth.external.reasons.generic')
  }

  function variantFor(item: ExternalAuthFeedback): 'danger' | 'warning' | 'success' | 'info' {
    if (item.kind === 'success') return 'success'
    if (item.kind === 'info') return 'info'
    return 'danger'
  }

  function consumeKeyFor(path: string, reason: string) {
    return `${path}?${reason}`
  }

  function present(item: ExternalAuthFeedback, options?: { stripQuery?: boolean }) {
    const key = consumeKeyFor(route.path, item.reason)
    if (consumedKey.value === key) {
      return
    }
    consumedKey.value = key
    feedback.value = item
    const title = messageFor(item)

    // 成功：Toast（含回跳到任意本地页）；错误/提示：登录注册壳用 SFAlert。
    if (item.preferToast) {
      if (import.meta.client) {
        toast.add({
          color: 'success',
          icon: 'i-lucide-check',
          title,
          duration: 10000
        })
      }
      // 成功落到登录壳时也短暂展示 status，避免 Toast 被忽略。
      alertMessage.value = title
      alertVariant.value = 'success'
    } else {
      alertMessage.value = title
      alertVariant.value = variantFor(item)
    }

    if (options?.stripQuery !== false && import.meta.client && route.query.ext_auth) {
      const nextQuery = stripExtAuthQuery({ ...route.query })
      void router.replace({
        path: route.path,
        query: nextQuery as Record<string, string | string[] | null | undefined>
      })
    }
  }

  async function consumeFromRoute() {
    // guest 中间件在已登录回跳前可能暂存成功 reason。
    const pendingReason = useState<string | null>('ext-auth:pending-toast-reason', () => null)
    if (pendingReason.value) {
      const pending = resolveExternalAuthFeedback(pendingReason.value)
      pendingReason.value = null
      if (pending) {
        present(pending, { stripQuery: false })
      }
    }

    const item = resolveExternalAuthFeedback(route.query.ext_auth)
    if (!item) {
      return
    }
    present(item)
  }

  // SSR：首屏即可展示错误/提示（成功 Toast 仅客户端）。
  if (import.meta.server) {
    const item = resolveExternalAuthFeedback(route.query.ext_auth)
    if (item && !item.preferToast) {
      const key = consumeKeyFor(route.path, item.reason)
      if (consumedKey.value !== key) {
        consumedKey.value = key
        feedback.value = item
        alertMessage.value = messageFor(item)
        alertVariant.value = variantFor(item)
      }
    }
  }

  if (import.meta.client) {
    // 根布局与登录壳都可调用；consumeKey 保证只处理一次。
    onMounted(() => {
      void consumeFromRoute()
    })
  }

  function clearAlert() {
    alertMessage.value = ''
  }

  return {
    feedback,
    alertMessage,
    alertVariant,
    clearAlert,
    consumeFromRoute
  }
}
