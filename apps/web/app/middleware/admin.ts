export default defineNuxtRouteMiddleware(async () => {
  const localePath = useLocalePath()
  const { user, status, refresh, can } = useAuthSession()

  if (!user.value) {
    await refresh({ timeout: import.meta.dev ? 800 : 2000 })
  }

  if (!user.value) {
    if (status.value === 'unavailable') {
      return abortNavigation(createError({
        statusCode: 503,
        statusMessage: 'Auth service temporarily unavailable'
      }))
    }

    return navigateTo(localePath('/login'))
  }

  if (!can('admin.access')) {
    return navigateTo(localePath('/'))
  }
})
