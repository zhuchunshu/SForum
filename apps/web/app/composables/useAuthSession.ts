export type CurrentUser = {
  id: number
  username: string
  displayName: string
  locale: string
  status: 'active' | 'disabled' | 'banned'
  isInitialSuperAdmin: boolean
  roleKeys: string[]
  permissions: string[]
}

export const useAuthSession = () => {
  const user = useState<CurrentUser | null>('auth:user', () => null)
  const pending = useState<boolean>('auth:pending', () => false)
  const apiBaseUrl = useRuntimeConfig().public.apiBaseUrl as string

  const refresh = async () => {
    pending.value = true
    try {
      user.value = await $fetch<CurrentUser>(`${apiBaseUrl}/auth/session`, {
        credentials: 'include',
        headers: import.meta.server ? useRequestHeaders(['cookie']) : undefined
      })
    } catch {
      user.value = null
    } finally {
      pending.value = false
    }
  }

  const can = (permission: string) => {
    return Boolean(user.value?.permissions.includes(permission) || user.value?.roleKeys.includes('super_admin'))
  }

  return { user, pending, refresh, can }
}
