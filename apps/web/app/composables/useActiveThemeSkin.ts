import {
  canUseActiveThemeSkinRecord,
  clearStoredSkinRecord,
  normalizeActiveThemeSkinPayload,
  readStoredSkinRecord,
  writeStoredSkinRecord,
  type ActiveThemeSkinCacheRecord,
  type ActiveThemeSkinResponse
} from '~/utils/activeThemeClientCache'

/**
 * L0 皮肤：从 API 拉取当前主题 CSS URL，交给 app.vue 的 useHead 输出。
 * SSR 首屏必须已经包含 link，避免 hydration 后注入导致整页重排。
 *
 * 瞬时失败只允许恢复同一 exact theme artifact 的短 TTL last-good；
 * 权威空响应会清掉旧 CSS，防止主题撤销/切换后旧 L0 复活。
 */
const SKIN_TIMEOUT_MS = import.meta.dev ? 5000 : 8000

function browserStorage() {
  if (!import.meta.client) {
    return null
  }
  try {
    return sessionStorage
  } catch {
    return null
  }
}

export function useActiveThemeSkin() {
  const links = useState<string[]>('sforum-active-theme-css', () => [])
  const lastPublicRecord = useState<ActiveThemeSkinCacheRecord | null>(
    'sforum-active-theme-css-last',
    () => null
  )
  const activeTheme = useActiveThemeIdentity()
  let revision = 0

  function applySkin(skin: ActiveThemeSkinResponse | null | undefined) {
    const normalized = normalizeActiveThemeSkinPayload(skin)
    activeTheme.update(normalized.identity)
    links.value = normalized.links

    if (normalized.record) {
      lastPublicRecord.value = normalized.record
      writeStoredSkinRecord(browserStorage(), normalized.record)
      return
    }

    lastPublicRecord.value = null
    clearStoredSkinRecord(browserStorage())
  }

  /** 立即恢复同一 exact artifact 的公开皮肤（不发起网络），用于离开管理端或刷新失败时防闪。 */
  function restoreLastPublic() {
    const identity = activeTheme.identity.value
    const now = Date.now()
    if (canUseActiveThemeSkinRecord(lastPublicRecord.value, identity, now)) {
      links.value = [...lastPublicRecord.value!.links]
      return true
    }

    const stored = readStoredSkinRecord(browserStorage(), identity, now)
    if (stored) {
      lastPublicRecord.value = stored
      links.value = [...stored.links]
      return true
    }
    return false
  }

  async function refresh() {
    const requestedRevision = ++revision
    try {
      const { request } = useApiClient()
      const skin = await request<ActiveThemeSkinResponse>('/site/active-theme/skin', {
        timeout: SKIN_TIMEOUT_MS
      })
      if (requestedRevision !== revision) {
        return
      }
      applySkin(skin)
    } catch {
      if (requestedRevision !== revision) {
        return
      }
      if (!links.value.length) {
        restoreLastPublic()
      }
    }
  }

  function clear() {
    revision++
    // 管理端只卸下当前 head 链；last-good 仍需 exact identity + TTL 才能恢复。
    links.value = []
  }

  return { links, refresh, clear, restoreLastPublic }
}
