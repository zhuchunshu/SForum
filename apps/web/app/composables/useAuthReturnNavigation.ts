export function useAuthReturnNavigation(options?: { explicitRedirect: unknown }) {
  // 路由中间件必须使用传入的 to；只有页面组件调用时才读取 Nuxt 当前路由。
  const route = options ? null : useRoute()
  const localePath = useLocalePath()
  const referrerPath = ref<string>()
  const redirect = computed(() => options ? options.explicitRedirect : route?.query.redirect)

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
      redirect.value,
      referrerPath.value,
      localePath('/')
    )
  )

  const returnFromAuth = () => navigateTo(destination.value, { replace: true })
  const authPageLink = (path: string) =>
    buildAuthPageLink(localePath(path), redirect.value)

  return { destination, returnFromAuth, authPageLink }
}
