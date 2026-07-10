export default defineNuxtRouteMiddleware(async (to) => {
  if (!to.meta.requiresAuth) {
    return
  }

  const localePath = useLocalePath()
  const { user, refresh } = useAuthSession()

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
