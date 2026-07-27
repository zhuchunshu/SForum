import { computed, watch } from 'vue'

export const COLOR_MODE_PREFERENCES = ['system', 'light', 'dark'] as const

export type ColorModePreference = typeof COLOR_MODE_PREFERENCES[number]
export type ResolvedColorMode = Exclude<ColorModePreference, 'system'>

export type ColorModeOptionDefinition = Readonly<{
  value: ColorModePreference
  icon: string
  labelKey: string
  descriptionKey?: string
}>

export const COLOR_MODE_OPTION_DEFINITIONS: readonly ColorModeOptionDefinition[] = Object.freeze([
  Object.freeze({
    value: 'system',
    icon: 'i-lucide-monitor',
    labelKey: 'appearance.colorMode.system',
    descriptionKey: 'appearance.colorMode.systemDescription'
  }),
  Object.freeze({
    value: 'light',
    icon: 'i-lucide-sun',
    labelKey: 'appearance.colorMode.light'
  }),
  Object.freeze({
    value: 'dark',
    icon: 'i-lucide-moon',
    labelKey: 'appearance.colorMode.dark'
  })
])

export function normalizeColorModePreference(value: unknown): ColorModePreference {
  return COLOR_MODE_PREFERENCES.includes(value as ColorModePreference)
    ? value as ColorModePreference
    : 'system'
}

export function useColorModePreference() {
  const colorMode = useColorMode()

  // 模块启动脚本会原样读取 storage；在共享 authority 内收口历史或损坏值。
  watch(
    () => colorMode.preference,
    (value) => {
      const normalized = normalizeColorModePreference(value)
      if (value !== normalized) {
        colorMode.preference = normalized
      }
    },
    { immediate: true }
  )

  const preference = computed<ColorModePreference>(() =>
    normalizeColorModePreference(colorMode.preference)
  )
  const resolvedMode = computed<ResolvedColorMode>(() =>
    colorMode.value === 'dark' ? 'dark' : 'light'
  )

  function setPreference(value: unknown) {
    colorMode.preference = normalizeColorModePreference(value)
  }

  return {
    preference,
    resolvedMode,
    options: COLOR_MODE_OPTION_DEFINITIONS,
    setPreference
  }
}
