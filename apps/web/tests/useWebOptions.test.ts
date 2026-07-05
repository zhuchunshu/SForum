import { describe, expect, test } from 'bun:test'

import {
  buildCustomAppearanceThemeValue,
  normalizeAppearanceThemeValue,
  resolveAppearanceTheme
} from '../app/composables/useWebOptions'

describe('appearance theme helpers', () => {
  test('normalizes preset and custom theme values', () => {
    expect(normalizeAppearanceThemeValue('violet')).toBe('violet')
    expect(normalizeAppearanceThemeValue('custom:#4F46E5')).toBe('custom:#4f46e5')
    expect(normalizeAppearanceThemeValue('custom:not-a-color')).toBe('pine_teal')
  })

  test('builds custom theme CSS variables for the root document', () => {
    const theme = buildCustomAppearanceThemeValue('#4F46E5')
    const resolved = resolveAppearanceTheme(theme)

    expect(theme).toBe('custom:#4f46e5')
    expect(resolved.dataTheme).toBe('custom')
    expect(resolved.cssVars['--sf-accent']).toBe('#4f46e5')
    expect(resolved.cssVars['--sf-accent-rgb']).toBe('79 70 229')
    expect(resolved.style).toContain('--sf-primary-500: #4f46e5')
  })
})
