import type { MaybeRefOrGetter } from 'vue'
import {
  exactThemeIdentityForPageResolve,
  PAGE_RESOLVE_REASON,
  type PageResolvePayload,
  type PageResolveReason
} from '~/utils/pageResolve'

export function useSystemErrorPagePresentation(pageId: MaybeRefOrGetter<string>) {
  const systemResolve = useSystemErrorPageResolve(pageId)
  const { refresh: refreshWebOptions } = useWebOptions()
  const {
    refresh: refreshAuthSession,
    status: authStatus,
    lastRefreshError: authRefreshError
  } = useAuthSession()
  const themeSkin = useActiveThemeSkin()
  const startupTimeout = import.meta.dev ? 1000 : 1200
  const hasServerSession = import.meta.server
    && /(?:^|;\s*)sforum_session=/.test(useRequestHeaders(['cookie']).cookie || '')

  function enterCoreEmergency(reason: PageResolveReason, error?: unknown): PageResolvePayload {
    themeSkin.clear({ resetIdentity: true })
    const fallback = systemResolve.useCoreFallback(reason)
    if (error) {
      console.error('[SForum] system error theme shell unavailable; using Core emergency page', error)
    }
    return fallback
  }

  async function prepare(): Promise<PageResolvePayload> {
    if (systemResolve.data.value) {
      return systemResolve.data.value
    }
    if (import.meta.client) {
      themeSkin.clear({ resetIdentity: true })
    }

    try {
      const resolved = await systemResolve.refresh({ deferCommit: true })
      if (resolved.provider === 'core' || resolved.fallback) {
        themeSkin.clear({ resetIdentity: true })
        const fallback = systemResolve.commit(resolved)
        if (systemResolve.failure.value) {
          console.error(
            '[SForum] system error page resolve unavailable; using Core emergency page',
            systemResolve.failure.value
          )
        }
        return fallback
      }

      const expectedIdentity = exactThemeIdentityForPageResolve(resolved)
      if (!expectedIdentity) {
        return enterCoreEmergency(PAGE_RESOLVE_REASON.artifactMismatch)
      }

      const skinTask = themeSkin.refresh({
        allowRestore: false,
        apply: false,
        expectedIdentity,
        requireLinks: true
      })
      const optionsTask = refreshWebOptions({
        timeout: startupTimeout,
        serverInternal: import.meta.server
      }).then(
        () => ({ ok: true as const }),
        error => ({ ok: false as const, error })
      )
      const authTask = hasServerSession
        ? refreshAuthSession({
            timeout: startupTimeout,
            serverInternal: true
          }).then(() => authStatus.value === 'unavailable'
            ? { ok: false as const, error: authRefreshError.value }
            : { ok: true as const })
        : Promise.resolve({ ok: true as const })

      const [skinResult, optionsResult, authResult] = await Promise.all([
        skinTask,
        optionsTask,
        authTask
      ])
      if (skinResult.status !== 'success') {
        return enterCoreEmergency(
          skinResult.status === 'failed' && skinResult.reason === 'request_failed'
            ? PAGE_RESOLVE_REASON.transportUnavailable
            : PAGE_RESOLVE_REASON.artifactMismatch,
          skinResult.error
        )
      }
      if (!optionsResult.ok) {
        return enterCoreEmergency(PAGE_RESOLVE_REASON.transportUnavailable, optionsResult.error)
      }
      if (!authResult.ok) {
        return enterCoreEmergency(PAGE_RESOLVE_REASON.transportUnavailable, authResult.error)
      }
      if (!themeSkin.commit(skinResult)) {
        return enterCoreEmergency(PAGE_RESOLVE_REASON.artifactMismatch)
      }
      return systemResolve.commit(resolved)
    } catch (error) {
      return enterCoreEmergency(PAGE_RESOLVE_REASON.transportUnavailable, error)
    }
  }

  return {
    resolvedPage: systemResolve.data,
    failure: systemResolve.failure,
    prepare
  }
}
