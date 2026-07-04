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
  const { request } = useApiClient()

  const refresh = async () => {
    pending.value = true
    try {
      user.value = await request<CurrentUser>('/auth/session')
    } catch {
      user.value = null
    } finally {
      pending.value = false
    }
  }

  const setUser = (currentUser: CurrentUser | null) => {
    user.value = currentUser
  }

  const can = (permission: string) => {
    return Boolean(user.value?.permissions.includes(permission) || user.value?.roleKeys.includes('super_admin'))
  }

  return { user, pending, refresh, setUser, can }
}
