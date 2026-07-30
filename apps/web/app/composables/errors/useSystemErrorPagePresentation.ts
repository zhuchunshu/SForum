import { useActiveThemeSkin } from '~/composables/themes/useActiveThemeSkin'
import { useErrorPageStartupState } from '~/composables/errors/useErrorPageStartupState'
import { useSystemErrorPageResolve } from '~/composables/errors/useSystemErrorPageResolve'
import type { MaybeRefOrGetter } from 'vue'
import {
  exactThemeIdentityForPageResolve,
  PAGE_RESOLVE_REASON,
  type PageResolvePayload,
  type PageResolveReason
} from '~/utils/pageResolve'

export function useSystemErrorPagePresentation(pageId: MaybeRefOrGetter<string>) {
  const systemResolve = useSystemErrorPageResolve(pageId)
  const startupState = useErrorPageStartupState()
  const themeSkin = useActiveThemeSkin()

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
      const startupTask = startupState.refresh()
      const resolved = await systemResolve.refresh({ deferCommit: true })
      if (resolved.provider === 'core' || resolved.fallback) {
        await startupTask
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
      const [skinResult, startupResult] = await Promise.all([
        skinTask,
        startupTask
      ])
      if (skinResult.status !== 'success') {
        return enterCoreEmergency(
          skinResult.status === 'failed' && skinResult.reason === 'request_failed'
            ? PAGE_RESOLVE_REASON.transportUnavailable
            : PAGE_RESOLVE_REASON.artifactMismatch,
          skinResult.error
        )
      }
      if (!startupResult.ok) {
        return enterCoreEmergency(PAGE_RESOLVE_REASON.transportUnavailable, startupResult.error)
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
