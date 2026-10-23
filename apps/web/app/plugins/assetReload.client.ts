import {
  extractErrorChainMessage,
  isDynamicImportFailureMessage,
  isNuxtAssetUrl,
  isWithinAssetReloadCooldown,
  readAssetReloadTimestamp,
  writeAssetReloadTimestamp
} from '~/utils/errors/assetReload'

/**
 * asset-reload：把「旧页面加载已发布替换的 /_nuxt/** chunk 失败」从
 * API 断线感知里分流出去——toast「站点刚发布了新版本，正在刷新」并自动
 * location.reload()，带 10s 冷却防刷新循环（sessionStorage 记时间戳）。
 *
 * 这类失败（路由 chunk 404、ERR_NETWORK_CHANGED、Failed to fetch dynamically
 * imported module）与后端可用性无关，不进 notifyApiConnectionError 的
 * 断线模态；两类监听都可能被同一次失败触发，用 reloadScheduled 去重。
 */
export default defineNuxtPlugin((nuxtApp) => {
  // i18n/toast 必须在插件同步体内捕获：事件回调已脱离 Nuxt 上下文。
  const i18n = nuxtApp.$i18n as { t?: (key: string) => string } | undefined
  const translate = (key: string) => i18n?.t?.(key) || key
  const toast = useToast()

  let reloadScheduled = false

  function scheduleAssetReload() {
    if (reloadScheduled) {
      return
    }

    const now = Date.now()
    if (isWithinAssetReloadCooldown(readAssetReloadTimestamp(window.sessionStorage), now)) {
      // 冷却期内：通常是刷新后新页面仍加载失败（发布异常），保持单次克制，防循环。
      reloadScheduled = true
      return
    }

    reloadScheduled = true
    writeAssetReloadTimestamp(window.sessionStorage, now)
    toast.add({
      color: 'primary',
      icon: 'i-lucide-refresh-cw',
      title: translate('errors.assetReload.toast'),
      duration: 10000
    })
    // 给 toast 留出渲染窗口，立即 reload 会让提示完全不可见。
    setTimeout(() => window.location.reload(), 800)
  }

  function handleResourceError(event: ErrorEvent) {
    const target = event.target
    if (!(target instanceof HTMLScriptElement) && !(target instanceof HTMLLinkElement)) {
      return
    }
    const url = target instanceof HTMLScriptElement ? target.src : target.href
    if (isNuxtAssetUrl(url)) {
      scheduleAssetReload()
    }
  }

  function handleUnhandledRejection(event: PromiseRejectionEvent) {
    if (isDynamicImportFailureMessage(extractErrorChainMessage(event.reason))) {
      scheduleAssetReload()
    }
  }

  // capture 阶段才能收到 <script>/<link> 资源加载失败（不冒泡）。
  window.addEventListener('error', handleResourceError, true)
  window.addEventListener('unhandledrejection', handleUnhandledRejection)

  if (import.meta.hot) {
    import.meta.hot.dispose(() => {
      window.removeEventListener('error', handleResourceError, true)
      window.removeEventListener('unhandledrejection', handleUnhandledRejection)
    })
  }
})
