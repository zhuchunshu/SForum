import type { AppearanceTheme, LightBackgroundPreset } from '~/utils/settings/appearance'

export type AdminAppearancePreview = {
  theme: AppearanceTheme
  lightBackground: LightBackgroundPreset
}

export function useAdminAppearancePreview() {
  const preview = useState<AdminAppearancePreview | null>('admin-appearance-preview', () => null)

  function show(value: AdminAppearancePreview) {
    preview.value = value
  }

  function clear() {
    preview.value = null
  }

  return { preview, show, clear }
}
