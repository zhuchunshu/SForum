import {
  joinAdminRoutePath,
  normalizeAdminRoutePrefix,
  resolveAdminRouteChildPath
} from '~/utils/admin/adminRoutePrefix'

export const useAdminRoutes = () => {
  const config = useRuntimeConfig()
  const { locale, defaultLocale } = useI18n()
  const prefix = normalizeAdminRoutePrefix(config.public.adminRoutePrefix as string)

  const path = (childPath = '/') => {
    const adminPath = joinAdminRoutePath(prefix, childPath)

    if (locale.value && locale.value !== defaultLocale) {
      return `/${locale.value}${adminPath}`
    }

    return adminPath
  }

  const routeId = (routePath: string) => {
    return resolveAdminRouteChildPath(prefix, routePath)
  }

  return {
    prefix,
    path,
    routeId
  }
}
