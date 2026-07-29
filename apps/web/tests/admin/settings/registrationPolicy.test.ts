import { describe, expect, test } from 'bun:test'

import { resolveRegistrationMode } from '../../../app/utils/settings/registrationPolicy'

describe('admin registration policy selection', () => {
  test('defaults a fresh installation to open registration', () => {
    expect(resolveRegistrationMode(undefined, true)).toBe('open')
  })

  test('maps a legacy disabled registration switch to closed when mode is absent', () => {
    expect(resolveRegistrationMode(undefined, false)).toBe('closed')
  })

  test('prefers a valid registration mode over the compatibility switch', () => {
    expect(resolveRegistrationMode('approval', true)).toBe('approval')
  })
})
