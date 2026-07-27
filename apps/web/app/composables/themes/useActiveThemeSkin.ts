import { useActiveThemeIdentity } from '~/composables/themes/useActiveThemeIdentity'
import {
  canUseActiveThemeSkinRecord,
  clearStoredSkinRecord,
  normalizeActiveThemeSkinPayload,
  readStoredSkinRecord,
  sameActiveThemeIdentity,
  writeStoredSkinRecord,
  type ActiveThemeIdentity,
  type ActiveThemeSkinCacheRecord,
  type ActiveThemeSkinResponse
} from '~/utils/themes/activeThemeClientCache'

/**
 * L0 皮肤：从 API 拉取当前主题 CSS URL，交给 app.vue 的 useHead 输出。
 * SSR 首屏必须已经包含 link，避免 hydration 后注入导致整页重排。
 *
 * 瞬时失败只允许恢复同一 exact theme artifact 的短 TTL last-good；
 * 权威空响应会清掉旧 CSS，防止主题撤销/切换后旧 L0 复活。
 */
const SKIN_TIMEOUT_MS = import.meta.dev ? 5000 : 8000

type ActiveThemeSkinRefreshOptions = {
  allowRestore?: boolean
  apply?: boolean
  expectedIdentity?: ActiveThemeIdentity
  requireLinks?: boolean
}

export type ActiveThemeSkinRefreshResult =
  | { status: 'success', identity: ActiveThemeIdentity | null, links: string[] }
  | { status: 'restored', identity: ActiveThemeIdentity, links: string[], error: unknown }
  | {
      status: 'failed'
      reason: 'request_failed' | 'invalid_payload' | 'artifact_mismatch' | 'superseded'
      error?: unknown
    }

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
  // refresh 可能在 error.vue 的预取 await 之后执行；请求函数必须在当前 Nuxt 上下文内捕获。
  const { request } = useApiClient()
  const links = useState<string[]>('sforum-active-theme-css', () => [])
  const lastPublicRecord = useState<ActiveThemeSkinCacheRecord | null>(
    'sforum-active-theme-css-last',
    () => null
  )
  const activeTheme = useActiveThemeIdentity()
  let revision = 0

  function applySkin(normalized: ReturnType<typeof normalizeActiveThemeSkinPayload>) {
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

  async function refresh(
    options: ActiveThemeSkinRefreshOptions = {}
  ): Promise<ActiveThemeSkinRefreshResult> {
    const requestedRevision = ++revision
    try {
      const skin = await request<ActiveThemeSkinResponse>('/site/active-theme/skin', {
        timeout: SKIN_TIMEOUT_MS,
        serverInternal: import.meta.server
      })
      if (requestedRevision !== revision) {
        return { status: 'failed', reason: 'superseded' }
      }
      const normalized = normalizeActiveThemeSkinPayload(skin)
      if (options.requireLinks && (!normalized.identity || !normalized.links.length)) {
        return { status: 'failed', reason: 'invalid_payload' }
      }
      if (options.expectedIdentity && !sameExactThemeIdentity(normalized.identity, options.expectedIdentity)) {
        return { status: 'failed', reason: 'artifact_mismatch' }
      }
      if (options.apply !== false) {
        applySkin(normalized)
      }
      return {
        status: 'success',
        identity: normalized.identity,
        links: [...normalized.links]
      }
    } catch (error) {
      if (requestedRevision !== revision) {
        return { status: 'failed', reason: 'superseded', error }
      }
      if (options.allowRestore !== false && !links.value.length && restoreLastPublic()) {
        const identity = activeTheme.identity.value
        if (identity) {
          return { status: 'restored', identity, links: [...links.value], error }
        }
      }
      return { status: 'failed', reason: 'request_failed', error }
    }
  }

  function clear(options: { resetIdentity?: boolean } = {}) {
    revision++
    // 管理端只卸下当前 head 链；last-good 仍需 exact identity + TTL 才能恢复。
    links.value = []
    if (options.resetIdentity) {
      activeTheme.update(null)
    }
  }

  function commit(result: ActiveThemeSkinRefreshResult) {
    if (result.status !== 'success' || !result.identity) {
      return false
    }
    applySkin(normalizeActiveThemeSkinPayload({
      extensionId: result.identity.extensionId,
      version: result.identity.version,
      packageDigest: result.identity.packageDigest,
      nodeRevision: result.identity.nodeRevision,
      css: result.links
    }))
    return true
  }

  return { links, refresh, commit, clear, restoreLastPublic }
}

function sameExactThemeIdentity(
  left: ActiveThemeIdentity | null | undefined,
  right: ActiveThemeIdentity | null | undefined
) {
  return Boolean(
    left?.version
    && right?.version
    && left.version === right.version
    && sameActiveThemeIdentity(left, right, { requireRevision: true })
  )
}
