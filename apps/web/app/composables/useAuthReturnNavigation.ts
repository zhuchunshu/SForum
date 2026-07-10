export function useAuthReturnNavigation() {
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
    resolveAuthReturnPath(route.query.redirect, referrerPath.value, localePath('/'))
  )

  const returnFromAuth = () => navigateTo(destination.value, { replace: true })

  return { destination, returnFromAuth }
}
