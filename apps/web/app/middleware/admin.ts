export default defineNuxtRouteMiddleware(async (to) => {
  const localePath = useLocalePath()
  const { user, refresh, can } = useAuthSession()

  if (!user.value) {
    await refresh({ timeout: import.meta.dev ? 800 : 2000 })
  }

  if (!user.value) {
    // 认证服务短暂不可用时不要升级成 Nuxt 错误页；没有可用会话则退回登录页。
    return navigateTo({
      path: localePath('/login'),
      query: { redirect: to.fullPath }
    })
  }

  if (!can('admin.access')) {
    return navigateTo(localePath('/'))
  }
})
