import {
  joinAdminRoutePath,
  normalizeAdminRoutePrefix
} from '~/utils/adminRoutePrefix'

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

  return {
    prefix,
    path
  }
}
