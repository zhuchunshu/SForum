import { useAuthSession, type CurrentUser } from '~/composables/identity/useAuthSession'
import { useApiClient } from '~/composables/useApiClient'
import type { AppearanceTheme, LightBackgroundPreset } from '~/utils/settings/appearance'

export type UserAppearancePreference = {
  theme: AppearanceTheme
  lightBackground: LightBackgroundPreset
}

export function useUserAppearancePreference() {
  const { user, setUser } = useAuthSession()
  const { request } = useApiClient()
  const preview = useState<UserAppearancePreference | null>('user-appearance-preview', () => null)
  const saved = computed<UserAppearancePreference | null>(() => user.value?.appearance || null)

  function showPreview(value: UserAppearancePreference) {
    preview.value = value
  }

  function clearPreview() {
    preview.value = null
  }

  async function save(value: UserAppearancePreference | null) {
    const updated = await request<CurrentUser>('/auth/appearance', value
      ? { method: 'PUT', body: value }
      : { method: 'DELETE' })
    setUser(updated)
    clearPreview()
    return updated
  }

  return { saved, preview, showPreview, clearPreview, save }
}
