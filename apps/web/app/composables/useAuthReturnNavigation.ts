export function useAuthReturnNavigation(explicitRedirect?: unknown) {
  const route = useRoute()
  const localePath = useLocalePath()
  const referrerPath = ref<string>()

  if (import.meta.client && document.referrer) {
    try {
      const referrer = new URL(document.referrer)
      if (referrer.origin === window.location.origin) {
        referrerPath.value = `${referrer.pathname}${referrer.search}${referrer.hash}`
      }
    } catch {
      // 非法 referrer 不应阻止认证后的安全回跳。
    }
  }

  const destination = computed(() =>
    resolveAuthReturnPath(
      explicitRedirect === undefined ? route.query.redirect : explicitRedirect,
      referrerPath.value,
      localePath('/')
    )
  )

  const returnFromAuth = () => navigateTo(destination.value, { replace: true })
  const authPageLink = (path: string) =>
    buildAuthPageLink(localePath(path), route.query.redirect)

  return { destination, returnFromAuth, authPageLink }
}
