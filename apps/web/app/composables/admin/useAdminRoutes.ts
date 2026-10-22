import {
  joinAdminRoutePath,
  normalizeAdminRoutePrefix,
  resolveAdminRouteChildPath
} from '~/utils/admin/adminRoutePrefix'

export const useAdminRoutes = () => {
  const config = useRuntimeConfig()
  // 后台路径的本地化交给 i18n：no_prefix 下 localePath 恒等返回，客户端导航
  // 不会落到不存在的 `/<locale><prefix>` 路由；历史带语言前缀的外链仍由
  // server/middleware/locale-prefix-compat.ts 301 剥前缀。
  const localePath = useLocalePath()
  const prefix = normalizeAdminRoutePrefix(config.public.adminRoutePrefix as string)

  const path = (childPath = '/') => localePath(joinAdminRoutePath(prefix, childPath))

  const routeId = (routePath: string) => {
    return resolveAdminRouteChildPath(prefix, routePath)
  }

  return {
    prefix,
    path,
    routeId
  }
}
