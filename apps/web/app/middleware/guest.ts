import { useAuthSession } from '~/composables/identity/useAuthSession'
import { useAuthReturnNavigation } from '~/composables/identity/useAuthReturnNavigation'
export default defineNuxtRouteMiddleware(async (to) => {
  const { user, status, refresh } = useAuthSession()

  if (!user.value && status.value === 'unknown') {
    await refresh({ timeout: import.meta.dev ? 800 : 2000 })
  }

  if (!user.value) {
    return
  }

  // 已登录用户访问登录/注册时 guest 会立刻回跳；先暂存成功 reason，避免丢失 Toast。
  // 仅识别 Host 稳定成功码，不依赖插件品牌文案。
  const extAuth = typeof to.query.ext_auth === 'string' ? to.query.ext_auth.trim() : ''
  if (extAuth === 'auth.external_login_ok' || extAuth === 'auth.external_link_ok') {
    const pending = useState<string | null>('ext-auth:pending-toast-reason', () => null)
    pending.value = extAuth
  }

  const { returnFromAuth } = useAuthReturnNavigation({ explicitRedirect: to.query.redirect })
  return returnFromAuth()
})
