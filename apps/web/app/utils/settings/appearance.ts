export type AppearanceThemePreset = 'pine_teal' | 'ocean_blue' | 'violet' | 'rose' | 'amber'
export type AppearanceTheme = AppearanceThemePreset | `custom:${string}`
export type ResolvedAppearanceTheme = {
  theme: AppearanceTheme
  dataTheme: AppearanceThemePreset | 'custom'
  customColor: string
  cssVars: Record<string, string>
  style: string
}

export type LightBackgroundPreset =
  | 'pure_white'
  | 'porcelain'
  | 'paper'
  | 'parchment'
  | 'mist_gray'
  | 'cool_frost'
  | 'cloud_blue'
  | 'mint_mist'
  | 'sage'
  | 'sakura'
  | 'lilac_mist'
  | 'morning_apricot'
export type LightBackgroundPalette = {
  background: string
  surface: string
  muted: string
  border: string
}

export const appearanceThemes: AppearanceThemePreset[] = ['pine_teal', 'ocean_blue', 'violet', 'rose', 'amber']
export const recommendedAppearanceTheme: AppearanceThemePreset = 'pine_teal'
export const defaultCustomThemeColor = '#2563eb'
export const lightBackgroundPresets: LightBackgroundPreset[] = [
  'pure_white',
  'porcelain',
  'paper',
  'parchment',
  'mist_gray',
  'cool_frost',
  'cloud_blue',
  'mint_mist',
  'sage',
  'sakura',
  'lilac_mist',
  'morning_apricot'
]
export const recommendedLightBackground: LightBackgroundPreset = 'pure_white'
export const lightBackgroundPalettes: Record<LightBackgroundPreset, LightBackgroundPalette> = {
  pure_white: { background: '#ffffff', surface: '#ffffff', muted: '#f5f5f8', border: '#e9e8ed' },
  porcelain: { background: '#f8faf9', surface: '#ffffff', muted: '#eff3f1', border: '#dce5e1' },
  paper: { background: '#f7f5ef', surface: '#fffdf8', muted: '#f1ede4', border: '#ded8cb' },
  parchment: { background: '#f4eedf', surface: '#fcf8ef', muted: '#ebe2cf', border: '#d8ccb5' },
  mist_gray: { background: '#f3f5f7', surface: '#ffffff', muted: '#eaedf1', border: '#dde1e6' },
  cool_frost: { background: '#f4f7fa', surface: '#fbfcfe', muted: '#eaf0f5', border: '#d9e2ea' },
  cloud_blue: { background: '#f1f6fb', surface: '#f9fcff', muted: '#e6eff8', border: '#d3e0ec' },
  mint_mist: { background: '#f1f8f5', surface: '#fbfffd', muted: '#e4f0eb', border: '#cfe1d9' },
  sage: { background: '#f2f5ef', surface: '#fbfcfa', muted: '#e6ebe1', border: '#d2dccb' },
  sakura: { background: '#fcf4f5', surface: '#fffbfb', muted: '#f7e7e9', border: '#ebd4d8' },
  lilac_mist: { background: '#f7f4fa', surface: '#fdfbff', muted: '#ede7f3', border: '#dfd4e8' },
  morning_apricot: { background: '#fcf4eb', surface: '#fffaf5', muted: '#f6e7d7', border: '#ebd5bd' }
}

const customThemePrefix = 'custom:'

export function normalizeAppearanceThemeValue(value: string | undefined): AppearanceTheme {
  const raw = value?.trim().toLowerCase() || ''
  if (isAppearanceThemePreset(raw)) {
    return raw
  }

  if (raw.startsWith(customThemePrefix)) {
    const color = normalizeHexColor(raw.slice(customThemePrefix.length))
    return color ? (`${customThemePrefix}${color}` as AppearanceTheme) : recommendedAppearanceTheme
  }

  const color = normalizeHexColor(raw)
  return color ? (`${customThemePrefix}${color}` as AppearanceTheme) : recommendedAppearanceTheme
}

export function isAppearanceThemePreset(value: string): value is AppearanceThemePreset {
  return appearanceThemes.includes(value as AppearanceThemePreset)
}

export function buildCustomAppearanceThemeValue(color: string): AppearanceTheme {
  return `${customThemePrefix}${normalizeHexColor(color) || defaultCustomThemeColor}` as AppearanceTheme
}

export function customColorFromAppearanceTheme(value: string): string | null {
  const normalized = normalizeAppearanceThemeValue(value)
  if (!normalized.startsWith(customThemePrefix)) {
    return null
  }
  return normalizeHexColor(normalized.slice(customThemePrefix.length))
}

export function resolveAppearanceTheme(value: string): ResolvedAppearanceTheme {
  const theme = normalizeAppearanceThemeValue(value)
  const customColor = customColorFromAppearanceTheme(theme) || defaultCustomThemeColor

  if (isAppearanceThemePreset(theme)) {
    return { theme, dataTheme: theme, customColor, cssVars: {}, style: '' }
  }

  const cssVars = buildCustomThemeVars(customColor)
  return { theme, dataTheme: 'custom', customColor, cssVars, style: cssVarsToStyle(cssVars) }
}

export function normalizeHexColor(value: string | undefined): string | null {
  const raw = value?.trim().toLowerCase().replace(/^#/, '') || ''
  return /^[0-9a-f]{6}$/.test(raw) ? `#${raw}` : null
}

export function normalizeLightBackground(value: string | undefined): LightBackgroundPreset {
  const normalized = value?.trim().toLowerCase()
  return lightBackgroundPresets.find(preset => preset === normalized) || recommendedLightBackground
}

function buildCustomThemeVars(color: string): Record<string, string> {
  const accent = normalizeHexColor(color) || defaultCustomThemeColor
  const rgb = hexToRgb(accent)
  const hover = mixHex(accent, '#000000', 0.16)
  const dark = mixHex(accent, '#ffffff', 0.32)

  return {
    '--sf-accent': accent,
    '--sf-accent-hover': hover,
    '--sf-accent-soft': mixHex(accent, '#ffffff', 0.92),
    '--sf-accent-soft-border': mixHex(accent, '#ffffff', 0.68),
    '--sf-accent-dark': dark,
    '--sf-accent-rgb': `${rgb.r} ${rgb.g} ${rgb.b}`,
    '--sf-accent-contrast': relativeLuminance(rgb) > 0.55 ? '#111827' : '#ffffff',
    '--sf-primary-50': mixHex(accent, '#ffffff', 0.94),
    '--sf-primary-100': mixHex(accent, '#ffffff', 0.88),
    '--sf-primary-200': mixHex(accent, '#ffffff', 0.72),
    '--sf-primary-300': mixHex(accent, '#ffffff', 0.52),
    '--sf-primary-400': mixHex(accent, '#ffffff', 0.28),
    '--sf-primary-500': accent,
    '--sf-primary-600': accent,
    '--sf-primary-700': hover,
    '--sf-primary-800': mixHex(accent, '#000000', 0.3),
    '--sf-primary-900': mixHex(accent, '#000000', 0.45),
    '--sf-primary-950': mixHex(accent, '#000000', 0.62),
    // 成功 Toast 跟随站点强调色，避免回落为 Nuxt UI 默认绿色。
    '--ui-color-success-50': 'var(--sf-primary-50)',
    '--ui-color-success-100': 'var(--sf-primary-100)',
    '--ui-color-success-200': 'var(--sf-primary-200)',
    '--ui-color-success-300': 'var(--sf-primary-300)',
    '--ui-color-success-400': 'var(--sf-primary-400)',
    '--ui-color-success-500': 'var(--sf-primary-500)',
    '--ui-color-success-600': 'var(--sf-primary-600)',
    '--ui-color-success-700': 'var(--sf-primary-700)',
    '--ui-color-success-800': 'var(--sf-primary-800)',
    '--ui-color-success-900': 'var(--sf-primary-900)',
    '--ui-color-success-950': 'var(--sf-primary-950)'
  }
}

function cssVarsToStyle(vars: Record<string, string>) {
  return Object.entries(vars).map(([name, value]) => `${name}: ${value}`).join('; ')
}

type RGB = { r: number, g: number, b: number }

function hexToRgb(hex: string): RGB {
  const value = (normalizeHexColor(hex) || defaultCustomThemeColor).slice(1)
  return {
    r: Number.parseInt(value.slice(0, 2), 16),
    g: Number.parseInt(value.slice(2, 4), 16),
    b: Number.parseInt(value.slice(4, 6), 16)
  }
}

function mixHex(from: string, to: string, amount: number) {
  const start = hexToRgb(from)
  const end = hexToRgb(to)
  const mix = (a: number, b: number) => Math.round(a + (b - a) * amount)
  return rgbToHex({ r: mix(start.r, end.r), g: mix(start.g, end.g), b: mix(start.b, end.b) })
}

function rgbToHex(rgb: RGB) {
  return `#${[rgb.r, rgb.g, rgb.b].map(value => value.toString(16).padStart(2, '0')).join('')}`
}

function relativeLuminance(rgb: RGB) {
  const channel = (value: number) => {
    const normalized = value / 255
    return normalized <= 0.03928 ? normalized / 12.92 : ((normalized + 0.055) / 1.055) ** 2.4
  }
  return 0.2126 * channel(rgb.r) + 0.7152 * channel(rgb.g) + 0.0722 * channel(rgb.b)
}
