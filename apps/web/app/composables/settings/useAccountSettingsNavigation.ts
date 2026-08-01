export type AccountSettingsNavItem = {
  id: string
  extensionId?: string
  label: string
  href: string
  order: number
  icon?: string
}

type AccountSettingsNavigationResponse = {
  items?: AccountSettingsNavItem[]
}

export function useAccountSettingsNavigation() {
  const { request } = useApiClient()

  function list() {
    return request<AccountSettingsNavigationResponse>('/site/account-navigation')
  }

  return { list }
}
