import { describe, expect, test } from 'bun:test'
import { readFileSync } from 'node:fs'
import { fileURLToPath } from 'node:url'

import {
  externalAuthFeedbackDelivery,
  externalAuthFeedbackToastDuration,
  externalAuthFeedbackUsesInlineSurface,
  resolveExternalAuthFeedback
} from '../../app/utils/identity/externalAuthFeedback'

const root = fileURLToPath(new URL('../..', import.meta.url))

describe('external auth callback feedback', () => {
  test('keeps auth-page failures inline and makes arbitrary return-page failures persistent', () => {
    const failure = resolveExternalAuthFeedback('auth.provider_callback_invalid')
    expect(failure).not.toBeNull()
    expect(externalAuthFeedbackDelivery(failure!, 'auth')).toBe('alert')
    expect(externalAuthFeedbackDelivery(failure!, 'global')).toBe('alert')
    expect(externalAuthFeedbackToastDuration(failure!)).toBe(0)
  })

  test('uses bounded global feedback for success and cancellation', () => {
    const success = resolveExternalAuthFeedback('auth.external_login_ok')
    const cancellation = resolveExternalAuthFeedback('auth.provider_cancelled')
    expect(externalAuthFeedbackDelivery(success!, 'auth')).toBe('toast')
    expect(externalAuthFeedbackDelivery(cancellation!, 'global')).toBe('alert')
    expect(externalAuthFeedbackToastDuration(success!)).toBe(10000)
    expect(externalAuthFeedbackToastDuration(cancellation!)).toBe(10000)
  })

  test('leaves login and registration callback errors to their inline surfaces', () => {
    expect(externalAuthFeedbackUsesInlineSurface('/login')).toBe(true)
    expect(externalAuthFeedbackUsesInlineSurface('/register/')).toBe(true)
    expect(externalAuthFeedbackUsesInlineSurface('/')).toBe(false)
    expect(externalAuthFeedbackUsesInlineSurface('/settings/login-methods')).toBe(false)
  })

  test('root surface renders non-success feedback during SSR and clears the hydrated query', () => {
    const app = readFileSync(`${root}/app/app.vue`, 'utf8')
    const composable = readFileSync(`${root}/app/composables/identity/useExternalAuthFeedback.ts`, 'utf8')
    expect(app).toContain("useExternalAuthFeedback({ surface: 'global' })")
    expect(composable).toContain("if (import.meta.server)")
    expect(composable).toContain("item?.kind !== 'success' && !externalAuthFeedbackUsesInlineSurface(route.path)")
    expect(composable).toContain('stripCallbackQuery()')
  })
})
