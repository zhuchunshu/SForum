import { afterEach, beforeEach, describe, expect, test } from 'bun:test'
import { effectScope, nextTick, reactive, type EffectScope } from 'vue'

type MockColorMode = {
  preference: string
  value: string
  unknown: boolean
  forced: boolean
}

let colorMode: MockColorMode
let scope: EffectScope

Object.assign(globalThis, {
  useColorMode: () => colorMode
})

const {
  COLOR_MODE_OPTION_DEFINITIONS,
  nextColorModePreference,
  normalizeColorModePreference,
  useColorModePreference
} = await import('../../app/composables/appearance/useColorModePreference')

beforeEach(() => {
  colorMode = reactive({
    preference: 'system',
    value: 'light',
    unknown: false,
    forced: false
  })
  scope = effectScope()
})

afterEach(() => {
  scope.stop()
})

function usePreference() {
  return scope.run(() => useColorModePreference())!
}

describe('color-mode preference authority', () => {
  test('keeps the three native values and normalizes unsupported input', () => {
    expect(normalizeColorModePreference('system')).toBe('system')
    expect(normalizeColorModePreference('light')).toBe('light')
    expect(normalizeColorModePreference('dark')).toBe('dark')
    expect(normalizeColorModePreference('sepia')).toBe('system')
    expect(normalizeColorModePreference(undefined)).toBe('system')
  })

  test('publishes the frozen Automatic, Light, Dark catalog', () => {
    expect(COLOR_MODE_OPTION_DEFINITIONS.map(option => option.value)).toEqual([
      'system',
      'light',
      'dark'
    ])
    expect(COLOR_MODE_OPTION_DEFINITIONS.map(option => option.icon)).toEqual([
      'i-tabler-brightness-filled',
      'i-tabler-sun-high',
      'i-tabler-moon-stars'
    ])
    expect(Object.isFrozen(COLOR_MODE_OPTION_DEFINITIONS)).toBe(true)
  })

  test('writes every selection as preference rather than resolution', () => {
    const model = usePreference()

    model.setPreference('dark')
    expect(colorMode.preference).toBe('dark')
    expect(model.preference.value).toBe('dark')

    model.setPreference('light')
    expect(colorMode.preference).toBe('light')

    colorMode.value = 'dark'
    model.setPreference('system')
    expect(colorMode.preference).toBe('system')
    expect(model.preference.value).toBe('system')
    expect(model.resolvedMode.value).toBe('dark')

    model.setPreference('unsupported')
    expect(colorMode.preference).toBe('system')
  })

  test('cycles preferences in the fixed Automatic, Light, Dark order', () => {
    expect(nextColorModePreference('system')).toBe('light')
    expect(nextColorModePreference('light')).toBe('dark')
    expect(nextColorModePreference('dark')).toBe('system')
    expect(nextColorModePreference('invalid')).toBe('light')

    const model = usePreference()
    model.cyclePreference()
    expect(colorMode.preference).toBe('light')
    model.cyclePreference()
    expect(colorMode.preference).toBe('dark')
    model.cyclePreference()
    expect(colorMode.preference).toBe('system')
  })

  test('tracks live resolved changes without replacing Automatic', () => {
    const model = usePreference()

    expect(model.preference.value).toBe('system')
    expect(model.resolvedMode.value).toBe('light')

    colorMode.value = 'dark'
    expect(model.resolvedMode.value).toBe('dark')
    expect(colorMode.preference).toBe('system')
  })

  test('normalizes a damaged stored preference through the native writer', async () => {
    colorMode.preference = 'damaged'
    const model = usePreference()
    await nextTick()

    expect(colorMode.preference).toBe('system')
    expect(model.preference.value).toBe('system')
  })

  test('keeps explicit preference authoritative when system resolution changes', () => {
    colorMode.preference = 'light'
    const model = usePreference()

    colorMode.value = 'dark'
    expect(model.preference.value).toBe('light')
    expect(model.resolvedMode.value).toBe('dark')
    expect(colorMode.preference).toBe('light')
  })
})
