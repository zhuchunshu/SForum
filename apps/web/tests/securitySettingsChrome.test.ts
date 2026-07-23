import { describe, expect, test } from 'bun:test'
import { readFileSync } from 'node:fs'

const source = (path: string) => readFileSync(new URL(path, import.meta.url), 'utf8')
const page = () => source('../app/components/SFSecuritySettingsPage.vue')
const route = () => source('../app/pages/settings/security.vue')
const themeTemplate = () => source('../../../extensions/builtin/themes/sforum-default/templates/settings-security.html')

describe('security settings chrome', () => {
  test('keeps Page Registry ownership boundaries intact', () => {
    expect(route()).toContain('SFPageOutlet')
    expect(route()).toContain('page="forum.settings.security"')
    expect(route()).toContain('<SFSecuritySettingsPage')
    expect(themeTemplate()).toContain('data-theme-owned="presentation"')
    expect(themeTemplate()).toContain('data-page="forum.settings.security"')
    expect(themeTemplate()).toContain('<sf-security-settings')
  })

  test('shares the public three-column settings chrome with profile', () => {
    const src = page()
    expect(src).toContain('data-layout="fullwidth-3col"')
    expect(src).toContain('sforum-settings__layout')
    expect(src).toContain('<SFHomeNavigation')
    expect(src).toContain('navigation-mode="route"')
    expect(src).toContain(':show-categories="false"')
    expect(src).toContain('<SFSettingsAccountNav')
    expect(src).toContain('active="security"')
    expect(src).toContain("useState<boolean>('forum-mobile-menu-open'")
    expect(src).toContain("useState<boolean>('forum-mobile-info-open'")
    expect(src).toContain('accountSecurity.rail.devicesTitle')
    expect(src).toContain('listSessions()')
    expect(src).toContain('listAPITokens()')
    expect(src).not.toContain('sf-public-page__container--narrow')
  })
})
