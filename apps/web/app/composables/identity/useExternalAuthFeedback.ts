/**
 * 消费 Host 回调带回的 `ext_auth` query：展示反馈并清理地址栏，避免刷新重复提示。
 * 使用 useState 做会话内单次消费，登录壳与根布局可共享同一结果。
 */

import {
  externalAuthFeedbackDelivery,
  externalAuthFeedbackToastDuration,
  externalAuthFeedbackUsesInlineSurface,
  resolveExternalAuthFeedback,
  stripExtAuthQuery,
  type ExternalAuthFeedback,
  type ExternalAuthFeedbackSurface
} from '~/utils/identity/externalAuthFeedback'

export function useExternalAuthFeedback(
  options: { surface?: ExternalAuthFeedbackSurface } = {}
) {
  const route = useRoute()
  const router = useRouter()
  const toast = useToast()
  const { t, te } = useI18n()
  const surface = options.surface || 'auth'

  const feedback = useState<ExternalAuthFeedback | null>(`ext-auth:feedback:${surface}`, () => null)
  const alertMessage = useState<string>(`ext-auth:alert-message:${surface}`, () => '')
  const alertVariant = useState<'danger' | 'warning' | 'success' | 'info'>(
    `ext-auth:alert-variant:${surface}`,
    () => 'danger'
  )
  const consumedKey = useState<string>(`ext-auth:consumed-key:${surface}`, () => '')
  let clearTimer: ReturnType<typeof setTimeout> | null = null

  const routeAlertItem = computed(() => {
    if (surface !== 'global' || externalAuthFeedbackUsesInlineSurface(route.path)) {
      return null
    }
    if (import.meta.client && !new URL(window.location.href).searchParams.has('ext_auth')) {
      return null
    }
    const item = resolveExternalAuthFeedback(route.query.ext_auth)
    return item && item.kind !== 'success' ? item : null
  })
  const displayAlertMessage = computed(() =>
    alertMessage.value || (routeAlertItem.value ? messageFor(routeAlertItem.value) : '')
  )
  const displayAlertVariant = computed(() =>
    alertMessage.value
      ? alertVariant.value
      : routeAlertItem.value
        ? variantFor(routeAlertItem.value)
        : 'danger'
  )

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

  function scheduleAlertClear(item: ExternalAuthFeedback) {
    if (clearTimer) {
      clearTimeout(clearTimer)
      clearTimer = null
    }
    if (!import.meta.client || item.kind === 'error') {
      return
    }
    clearTimer = setTimeout(() => {
      alertMessage.value = ''
      clearTimer = null
    }, externalAuthFeedbackToastDuration(item))
  }

  function initializeAlert(item: ExternalAuthFeedback) {
    consumedKey.value = consumeKeyFor(route.path, item.reason)
    feedback.value = item
    alertMessage.value = messageFor(item)
    alertVariant.value = variantFor(item)
  }

  function stripCallbackQuery() {
    if (!import.meta.client || !route.query.ext_auth) {
      return
    }
    if (!new URL(window.location.href).searchParams.has('ext_auth')) {
      return
    }
    const nextQuery = stripExtAuthQuery({ ...route.query })
    void router.replace({
      path: route.path,
      query: nextQuery as Record<string, string | string[] | null | undefined>
    })
  }

  function replaceCallbackQuery() {
    if (!import.meta.client || !route.query.ext_auth) {
      return
    }
    const nextQuery = stripExtAuthQuery({ ...route.query })
    void router.replace({
      path: route.path,
      query: nextQuery as Record<string, string | string[] | null | undefined>
    })
  }

  function stripHydratedCallbackQuery() {
    if (!import.meta.client || !route.query.ext_auth) {
      return
    }
    if (surface === 'global' && externalAuthFeedbackUsesInlineSurface(route.path)) {
      return
    }
    // hydration 出错时 Vue mounted hook 可能不执行；这里只同步清理地址栏，不改变路由状态。
    const url = new URL(window.location.href)
    url.searchParams.delete('ext_auth')
    window.history.replaceState(
      window.history.state,
      '',
      `${url.pathname}${url.search}${url.hash}`
    )
  }

  function present(item: ExternalAuthFeedback, options?: { stripQuery?: boolean }) {
    const key = consumeKeyFor(route.path, item.reason)
    if (consumedKey.value === key) {
      if (options?.stripQuery !== false) {
        stripCallbackQuery()
      }
      return
    }
    consumedKey.value = key
    feedback.value = item
    const title = messageFor(item)

    const delivery = externalAuthFeedbackDelivery(item, surface)
    if (delivery === 'toast') {
      if (import.meta.client) {
        toast.add({
          color: item.kind === 'error' ? 'error' : item.kind === 'success' ? 'success' : 'neutral',
          icon: item.kind === 'error'
            ? 'i-lucide-triangle-alert'
            : item.kind === 'success'
              ? 'i-lucide-check'
              : 'i-lucide-info',
          title,
          duration: externalAuthFeedbackToastDuration(item)
        })
      }
      if (surface === 'auth') {
        // 成功落到登录壳时也展示 status，避免 Toast 被忽略。
        alertMessage.value = title
        alertVariant.value = 'success'
        scheduleAlertClear(item)
      }
    } else {
      alertMessage.value = title
      alertVariant.value = variantFor(item)
      scheduleAlertClear(item)
    }

    if (options?.stripQuery !== false) {
      stripCallbackQuery()
    }
  }

  async function consumeFromRoute() {
    if (surface === 'global' && externalAuthFeedbackUsesInlineSurface(route.path)) {
      return
    }
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
    const shouldRenderAlert = surface === 'auth'
      ? !item?.preferToast
      : false
    if (item && shouldRenderAlert) {
      const key = consumeKeyFor(route.path, item.reason)
      if (consumedKey.value !== key) {
        initializeAlert(item)
      }
    }
  }

  if (import.meta.client) {
    const item = resolveExternalAuthFeedback(route.query.ext_auth)
    const shouldRenderAlert = surface === 'auth'
      ? !item?.preferToast
      : item?.kind !== 'success' && !externalAuthFeedbackUsesInlineSurface(route.path)
    if (item && shouldRenderAlert) {
      const key = consumeKeyFor(route.path, item.reason)
      if (consumedKey.value !== key) {
        initializeAlert(item)
      }
    }
    if (!(surface === 'global' && externalAuthFeedbackUsesInlineSurface(route.path))) {
      // mounted hook may be skipped by an unrelated hydration warning; router cleanup is harmless here.
      stripCallbackQuery()
    }
    stripHydratedCallbackQuery()
    // 根布局与登录壳都可调用；consumeKey 保证只处理一次。
    onMounted(() => {
      if (feedback.value && alertMessage.value) {
        // SSR 已渲染的非错误提示在 hydration 后开始自动关闭计时。
        scheduleAlertClear(feedback.value)
      }
      void consumeFromRoute()
    })
    onScopeDispose(() => {
      if (clearTimer) clearTimeout(clearTimer)
    })
  }

  function clearAlert() {
    if (clearTimer) {
      clearTimeout(clearTimer)
      clearTimer = null
    }
    alertMessage.value = ''
    replaceCallbackQuery()
  }

  return {
    feedback,
    alertMessage,
    alertVariant,
    displayAlertMessage,
    displayAlertVariant,
    clearAlert,
    consumeFromRoute
  }
}
