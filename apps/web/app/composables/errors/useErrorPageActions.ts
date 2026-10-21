/**
 * 错误页动作：一律交给组件渲染成真链接，JS 只做增强。
 *
 * 入口 chunk 被中间层拒载时（QQ 内置浏览器 X5 云加速、CDN、企业代理），页面
 * 只剩 SSR 结果，任何只绑 `@click` 的按钮都会失效，用户没有任何出口。这里同时
 * 交出 href 与事件处理：有 JS 时先 `preventDefault` 走 SPA 恢复，没 JS 时浏览器
 * 自己按 href 导航。
 */
export function useErrorPageActions() {
  const localePath = useLocalePath()
  const requestURL = useRequestURL()
  const canGoBack = ref(false)
  const homeHref = computed(() => localePath('/'))
  // 重试目标就是当前失败地址：整页重新请求，不依赖客户端路由。
  const retryHref = computed(() => `${requestURL.pathname}${requestURL.search}`)

  onMounted(() => {
    canGoBack.value = window.history.length > 1
  })

  function clearResolvedDevErrorOverlay() {
    if (import.meta.dev && import.meta.client) {
      // Nuxt dev 会在 clearError 后留下已折叠入口；错误已恢复时不应遮挡正常页面。
      document.querySelector('nuxt-error-overlay.pip-hidden')?.remove()
    }
  }

  async function goHome(event?: MouseEvent) {
    event?.preventDefault()
    await clearError({ redirect: homeHref.value })
    clearResolvedDevErrorOverlay()
  }

  async function goBack(event?: MouseEvent) {
    if (import.meta.client && window.history.length > 1) {
      event?.preventDefault()
      await clearError()
      clearResolvedDevErrorOverlay()
      window.history.back()
      return
    }
    await goHome(event)
  }

  async function retry(event?: MouseEvent) {
    event?.preventDefault()
    await clearError()
    clearResolvedDevErrorOverlay()
    if (import.meta.client) {
      window.location.reload()
    }
  }

  return { canGoBack, homeHref, retryHref, goHome, goBack, retry }
}
