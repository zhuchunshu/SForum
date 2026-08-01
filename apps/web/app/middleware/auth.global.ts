import { useAuthSession } from '~/composables/identity/useAuthSession'
import { normalizeEnabledOption, useWebOptions } from '~/composables/useWebOptions'
import { normalizeAdminRoutePrefix, resolveAdminRouteChildPath } from '~/utils/admin/adminRoutePrefix'

const requireEmailVerificationOption = 'identity.registration.require_email_verification'

export default defineNuxtRouteMiddleware(async (to) => {
  const localePath = useLocalePath()
  const runtimeConfig = useRuntimeConfig()
  const adminRoutePrefix = normalizeAdminRoutePrefix(runtimeConfig.public.adminRoutePrefix as string)
  const verificationPath = localePath('/email-verification')
  const isAdminRoute = resolveAdminRouteChildPath(adminRoutePrefix, to.path) !== null
  const isVerificationRoute = normalizeRoutePath(to.path) === normalizeRoutePath(verificationPath)
  const { user, status, refresh } = useAuthSession()

  if (!isAdminRoute && !isVerificationRoute) {
    const {
      loaded: webOptionsLoaded,
      refresh: refreshWebOptions,
      webOption
    } = useWebOptions()

    if (!webOptionsLoaded.value) {
      await refreshWebOptions({ timeout: import.meta.dev ? 800 : 2000 }).catch(() => null)
    }

    const verificationRequired = webOptionsLoaded.value
      && normalizeEnabledOption(webOption(requireEmailVerificationOption), false)

    if (verificationRequired) {
      const hasServerSession = import.meta.server
        && /(?:^|;\s*)sforum_session=/.test(useRequestHeaders(['cookie']).cookie || '')
      const shouldRefreshSession = Boolean(user.value)
        || (status.value !== 'guest'
          && (status.value !== 'unknown' || !import.meta.server || hasServerSession))

      if (shouldRefreshSession) {
        await refresh({ timeout: import.meta.dev ? 800 : 2000 })
      }

      if (user.value && !user.value.emailVerified) {
        return navigateTo({
          path: verificationPath,
          query: { redirect: to.fullPath }
        })
      }
    }
  }

  if (!to.meta.requiresAuth) {
    return
  }

  if (!user.value) {
    await refresh({ timeout: import.meta.dev ? 800 : 2000 })
  }

  if (user.value) {
    return
  }

  // 普通用户页不暴露受保护内容；认证服务短暂不可用时也降级到登录页，避免 SSR 503 或空白体验。
  return navigateTo({
    path: localePath('/login'),
    query: { redirect: to.fullPath }
  })
})

function normalizeRoutePath(path: string) {
  return path.length > 1 ? path.replace(/\/+$/, '') : path
}
