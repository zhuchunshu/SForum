import type { Ref } from 'vue'
import {
  ADMIN_OVERVIEW_KPI_POLL_MS,
  ADMIN_OVERVIEW_RESOURCE_POLL_MS,
  applyOverviewResources,
  type AdminOverview,
  type AdminOverviewResources
} from '~/utils/admin/adminOverview'

type OverviewRequest = <T>(path: string) => Promise<T>

/**
 * 仪表盘双频轮询：资源 5s（轻量 endpoint），KPI/全量 30s。
 * 仅在客户端、页面激活且标签页可见时运行；后台失败静默忽略。
 */
export function useAdminOverviewLive(options: {
  overview: Ref<AdminOverview | null>
  request: OverviewRequest
}) {
  let resourceTimer: ReturnType<typeof setInterval> | null = null
  let kpiTimer: ReturnType<typeof setInterval> | null = null
  let resourceInFlight = false
  let kpiInFlight = false
  let started = false

  async function refreshResources() {
    if (resourceInFlight || !options.overview.value) {
      return
    }
    resourceInFlight = true
    try {
      const patch = await options.request<AdminOverviewResources>('/admin/overview/resources')
      if (!options.overview.value) {
        return
      }
      options.overview.value = applyOverviewResources(options.overview.value, patch)
    } catch {
      // 后台轮询失败不打断页面；手动刷新仍走 useAsyncData error。
    } finally {
      resourceInFlight = false
    }
  }

  async function refreshKpi() {
    if (kpiInFlight) {
      return
    }
    kpiInFlight = true
    try {
      const next = await options.request<AdminOverview>('/admin/overview')
      options.overview.value = next
    } catch {
      // 同上：静默
    } finally {
      kpiInFlight = false
    }
  }

  function pageVisible() {
    return typeof document === 'undefined' || document.visibilityState === 'visible'
  }

  function stop() {
    if (resourceTimer) {
      clearInterval(resourceTimer)
      resourceTimer = null
    }
    if (kpiTimer) {
      clearInterval(kpiTimer)
      kpiTimer = null
    }
    started = false
  }

  function start() {
    if (!import.meta.client || started) {
      return
    }
    started = true
    resourceTimer = setInterval(() => {
      if (!pageVisible()) {
        return
      }
      void refreshResources()
    }, ADMIN_OVERVIEW_RESOURCE_POLL_MS)
    kpiTimer = setInterval(() => {
      if (!pageVisible()) {
        return
      }
      void refreshKpi()
    }, ADMIN_OVERVIEW_KPI_POLL_MS)
  }

  function onVisibilityChange() {
    if (!import.meta.client) {
      return
    }
    if (document.visibilityState === 'visible') {
      // 回到前台立刻补一帧资源，避免长时间挂起后数字陈旧。
      void refreshResources()
    }
  }

  if (import.meta.client) {
    onMounted(() => {
      start()
      document.addEventListener('visibilitychange', onVisibilityChange)
    })
    onActivated(() => {
      start()
    })
    onDeactivated(() => {
      stop()
    })
    onUnmounted(() => {
      stop()
      document.removeEventListener('visibilitychange', onVisibilityChange)
    })
  }

  return {
    refreshResources,
    refreshKpi,
    start,
    stop
  }
}
