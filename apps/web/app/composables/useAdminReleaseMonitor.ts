import { releaseId as embeddedReleaseId } from '#sforum/admin-extension-metadata'

type ActiveAdminRelease = { releaseId: string, reloadMode: 'prompt' | 'force' }

export function useAdminReleaseMonitor() {
  const changed = ref(false)
  const checking = ref(false)
  let timer: ReturnType<typeof setInterval> | null = null

  async function check() {
    if (checking.value || !import.meta.client) return
    checking.value = true
    try {
      const active = await $fetch<ActiveAdminRelease>('/__sforum/admin-release', { headers: { 'cache-control': 'no-cache' } })
      if (`${active.releaseId}` === `${embeddedReleaseId}`) return
      if (active.reloadMode === 'force') {
        window.location.reload()
        return
      }
      changed.value = true
    } finally {
      checking.value = false
    }
  }

  function visibilityCheck() { if (document.visibilityState === 'visible') void check() }
  onMounted(() => {
    timer = setInterval(check, 30000)
    document.addEventListener('visibilitychange', visibilityCheck)
  })
  onUnmounted(() => {
    if (timer) clearInterval(timer)
    document.removeEventListener('visibilitychange', visibilityCheck)
  })
  return { changed, checking, check, reload: () => window.location.reload() }
}
