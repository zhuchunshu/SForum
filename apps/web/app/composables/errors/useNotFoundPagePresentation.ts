import { useActiveThemeSkin } from '~/composables/themes/useActiveThemeSkin'
import { useErrorPageStartupState } from '~/composables/errors/useErrorPageStartupState'
import { useNotFoundPageResolve } from '~/composables/errors/useNotFoundPageResolve'
import {
  exactThemeIdentityForPageResolve,
  PAGE_RESOLVE_REASON,
  type PageResolvePayload,
  type PageResolveReason
} from '~/utils/pageResolve'

/**
 * 在抛出公开 404 前准备最终 L0/L1 快照；error.vue 只同步消费，避免 Core 闪屏和 SSR 分叉。
 */
export function useNotFoundPagePresentation() {
  const notFoundResolve = useNotFoundPageResolve()
  const startupState = useErrorPageStartupState()
  const themeSkin = useActiveThemeSkin()

  function enterCoreEmergency(reason: PageResolveReason, error?: unknown): PageResolvePayload {
    themeSkin.clear({ resetIdentity: true })
    const fallback = notFoundResolve.useCoreFallback(reason)
    if (error) {
      console.error('[SForum] 404 theme shell unavailable; using Core emergency page', error)
    }
    return fallback
  }

  async function prepare(): Promise<PageResolvePayload> {
    if (notFoundResolve.data.value) {
      return notFoundResolve.data.value
    }
    if (import.meta.client) {
      themeSkin.clear({ resetIdentity: true })
    }

    try {
      const startupTask = startupState.refresh()
      const resolved = await notFoundResolve.refresh({ deferCommit: true })
      if (resolved.provider === 'core' || resolved.fallback) {
        await startupTask
        themeSkin.clear({ resetIdentity: true })
        const fallback = notFoundResolve.commit(resolved)
        if (notFoundResolve.failure.value) {
          console.error(
            '[SForum] 404 page resolve unavailable; using Core emergency page',
            notFoundResolve.failure.value
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
      return notFoundResolve.commit(resolved)
    } catch (error) {
      return enterCoreEmergency(PAGE_RESOLVE_REASON.transportUnavailable, error)
    }
  }

  return {
    resolvedPage: notFoundResolve.data,
    failure: notFoundResolve.failure,
    prepare
  }
}
