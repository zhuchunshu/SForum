import {
  buildAuthPageLink,
  requiresDocumentNavigation,
  resolveAuthReturnPath
} from '~/utils/identity/authReturn'

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

  // 回跳目标是 API / 插件端点时必须整页导航：客户端路由只会命中
  // `[...sfRegistryPage]` catch-all，端点永远收不到请求（例如 OAuth2 的
  // `/api/oauth/authorize`，登录后回跳会停在 “Failed to load extension page”）。
  const returnFromAuth = async () => {
    const target = destination.value
    if (requiresDocumentNavigation(target)) {
      if (import.meta.client) {
        window.location.assign(target)
        return
      }
      return navigateTo(target, { external: true })
    }
    return navigateTo(target, { replace: true })
  }
  const authPageLink = (path: string) =>
    buildAuthPageLink(localePath(path), redirect.value)

  return { destination, returnFromAuth, authPageLink }
}
