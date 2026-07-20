import { describe, expect, test } from 'bun:test'
import { readFileSync } from 'node:fs'

const accountSecurityPage = () => readFileSync(
  new URL('../../../apps/web/app/components/SFSecuritySettingsPage.vue', import.meta.url),
  'utf8'
)

const adminSettingsPage = () => readFileSync(
  new URL('../app/pages/admin/settings/index.vue', import.meta.url),
  'utf8'
)

const homepage = () => readFileSync(
  new URL('../../../apps/web/app/components/SFHomePage.vue', import.meta.url),
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

  test('admin account security settings track and reset session options', () => {
    const source = adminSettingsPage()

    expect(source).toContain('initialSessionsMaxDevices')
    expect(source).toContain('initialSessionsKeepDays')
    expect(source).toContain('form.sessionsMaxDevices !== initialSessionsMaxDevices.value')
    expect(source).toContain('form.sessionsKeepDays !== initialSessionsKeepDays.value')
    expect(source).toContain('form.sessionsMaxDevices = initialSessionsMaxDevices.value')
    expect(source).toContain('form.sessionsKeepDays = initialSessionsKeepDays.value')
  })

  test('homepage exposes only the implemented latest feed filter', () => {
    const source = homepage()

    expect(source).toContain("t('home.filter.latest')")
    expect(source).not.toContain("t('home.filter.hot')")
    expect(source).not.toContain("t('home.filter.unread')")
    expect(source).not.toContain("t('home.filter.ranking')")
  })
})
