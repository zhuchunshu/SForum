import { useAuthSession } from '~/composables/identity/useAuthSession'

type ErrorPageStartupResult =
  | { ok: true }
  | { ok: false, error: unknown }

export function useErrorPageStartupState() {
  const { refresh: refreshWebOptions } = useWebOptions()
  const {
    refresh: refreshAuthSession,
    status: authStatus,
    lastRefreshError: authRefreshError
  } = useAuthSession()
  const startupTimeout = import.meta.dev ? 1000 : 1200
  const hasServerSession = import.meta.server
    && /(?:^|;\s*)sforum_session=/.test(useRequestHeaders(['cookie']).cookie || '')

  async function refresh(): Promise<ErrorPageStartupResult> {
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

    const [optionsResult, authResult] = await Promise.all([optionsTask, authTask])
    if (!optionsResult.ok) {
      return optionsResult
    }
    if (!authResult.ok) {
      return authResult
    }
    return { ok: true }
  }

  return { refresh }
}
