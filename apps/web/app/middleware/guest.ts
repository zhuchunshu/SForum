export default defineNuxtRouteMiddleware(async () => {
  const { user, status, refresh } = useAuthSession()

  if (!user.value && status.value === 'unknown') {
    await refresh({ timeout: import.meta.dev ? 800 : 2000 })
  }

  if (!user.value) {
    return
  }

  const { returnFromAuth } = useAuthReturnNavigation()
  return returnFromAuth()
})
