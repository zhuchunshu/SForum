import { useAuthSession } from '~/composables/identity/useAuthSession'
export default defineNuxtRouteMiddleware(async (to) => {
  const localePath = useLocalePath()
  const { user, refresh, can } = useAuthSession()
  if (!user.value) await refresh({ timeout: import.meta.dev ? 800 : 2000 })
  if (!user.value) return navigateTo({ path: localePath('/login'), query: { redirect: to.fullPath } })
  if (!can('moderation.review')) return navigateTo(localePath('/'))
})
