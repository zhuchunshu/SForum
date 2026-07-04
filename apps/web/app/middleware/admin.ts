export default defineNuxtRouteMiddleware(async () => {
  const localePath = useLocalePath()
  const { user, refresh, can } = useAuthSession()

  if (!user.value) {
    await refresh()
  }

  if (!user.value) {
    return navigateTo(localePath('/login'))
  }

  if (!can('admin.access')) {
    return navigateTo(localePath('/'))
  }
})
