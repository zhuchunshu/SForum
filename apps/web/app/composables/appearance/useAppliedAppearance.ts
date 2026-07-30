import { toValue, type MaybeRefOrGetter } from 'vue'
import {
  useUserAppearancePreference,
  type UserAppearancePreference
} from '~/composables/appearance/useUserAppearancePreference'
import { resolveAppearanceTheme } from '~/utils/settings/appearance'

export function useAppliedAppearance(
  override?: MaybeRefOrGetter<UserAppearancePreference | null | undefined>
) {
  const {
    lightBackground: siteLightBackground,
    resolvedAppearanceTheme: siteAppearanceTheme
  } = useWebOptions()
  const {
    saved: savedUserAppearance,
    preview: userAppearancePreview
  } = useUserAppearancePreference()

  const appearanceOverride = computed(() =>
    override === undefined ? null : toValue(override) || null
  )
  const effectiveAppearance = computed(() =>
    appearanceOverride.value || userAppearancePreview.value || savedUserAppearance.value
  )
  const appliedAppearanceTheme = computed(() => effectiveAppearance.value
    ? resolveAppearanceTheme(effectiveAppearance.value.theme)
    : siteAppearanceTheme.value)
  const appliedLightBackground = computed(() =>
    effectiveAppearance.value?.lightBackground || siteLightBackground.value
  )

  return {
    appliedAppearanceTheme,
    appliedLightBackground,
    savedUserAppearance,
    userAppearancePreview
  }
}
