import { describe, expect, test } from 'bun:test'
import { readFileSync } from 'node:fs'
import {
  accountSecurityPayload,
  normalizeAccountSecurityForm,
  normalizeAccountSecurityForSave,
  recommendedAccountSecurityForm
} from '../../app/components/admin/settings/site/models/accountSecurity'

const accountSecurityPage = () => readFileSync(
  new URL('../../app/components/settings/SFSecuritySettingsPage.vue', import.meta.url),
  'utf8'
)

const homepage = () => readFileSync(
  new URL('../../app/components/forum/SFHomePage.vue', import.meta.url),
  'utf8'
)

describe('account security page contracts', () => {
  test('renders active sessions from the paginated API result items', () => {
    const source = accountSecurityPage()

    expect(source).toContain('activeSessions')
    expect(source).toContain('sessions.value?.items')
    expect(source).toContain('v-for="session in activeSessions"')
    expect(source).toContain('activeSessions.length <= 1')
    expect(source).not.toContain('v-for="session in sessions"')
    expect(source).not.toContain('sessions.length <= 1')
  })

  test('admin account security normalizes, restores and submits only its owned options', () => {
    const normalized = normalizeAccountSecurityForm([
      { name: 'identity.password.min_length', value: '16' },
      { name: 'identity.sessions.max_devices', value: '7' },
      { name: 'identity.sessions.keep_days', value: '90' },
      { name: 'unrelated.option', value: 'must-not-submit' }
    ])
    expect(normalized.passwordMinLength).toBe(16)
    expect(normalized.sessionsMaxDevices).toBe(7)
    expect(normalized.sessionsKeepDays).toBe(90)

    const restored = { ...recommendedAccountSecurityForm }
    expect(restored.sessionsMaxDevices).toBe(5)
    expect(restored.sessionsKeepDays).toBe(30)

    const prepared = normalizeAccountSecurityForSave({ ...normalized, passwordMinLength: 100, passwordMaxLength: 64 })
    expect(prepared.passwordMaxLength).toBe(100)
    const payload = accountSecurityPayload(prepared)
    expect(payload).toContainEqual({ name: 'identity.sessions.max_devices', value: '7' })
    expect(payload).toContainEqual({ name: 'identity.sessions.keep_days', value: '90' })
    expect(payload.some(item => item.name === 'unrelated.option')).toBe(false)
  })

  test('homepage exposes only the implemented latest feed filter', () => {
    const source = homepage()

    expect(source).toContain("t('home.filter.latest')")
    expect(source).not.toContain("t('home.filter.hot')")
    expect(source).not.toContain("t('home.filter.unread')")
    expect(source).not.toContain("t('home.filter.ranking')")
  })
})
