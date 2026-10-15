import { describe, expect, test } from 'bun:test'
import { readFileSync } from 'node:fs'
import {
  createDefaultForumSettings,
  forumSettingsManageKeys,
  forumSettingsValidationError,
  normalizeForumSettings,
  trustedDomainEntries
} from '~/utils/admin/adminForum'

const source = (path: string) => readFileSync(new URL(`../../../${path}`, import.meta.url), 'utf8')
const locale = (name: string) => JSON.parse(source(`i18n/locales/${name}.json`))

describe('forum outbound link settings', () => {
  test('ships the interstitial switch and allowlist in the forum reading tab', () => {
    const tab = source('app/components/admin/settings/forum/tabs/SFAdminForumReadingTab.vue')
    expect(tab).toContain("'excerptRuneLimit', 'externalInterstitial', 'trustedDomains'")
    expect(tab).toContain('tab.form.externalInterstitial')
    expect(tab).toContain('tab.form.trustedDomains')
    expect(tab).toContain("t('admin.forum.settings.externalInterstitialHelp')")
    expect(tab).toContain("t('admin.forum.settings.trustedDomainsHelp')")
  })

  test('writes both options through the shared forum settings payload', () => {
    expect(forumSettingsManageKeys).toContain('externalInterstitial')
    expect(forumSettingsManageKeys).toContain('trustedDomains')
    const defaults = createDefaultForumSettings()
    expect(defaults.externalInterstitial).toBe(true)
    expect(defaults.trustedDomains).toBe('')
  })

  test('normalizes the allowlist to lowercase host[:port] lines', () => {
    const normalized = normalizeForumSettings({
      externalInterstitial: false,
      trustedDomains: 'https://GitHub.com/a/b\nExample.ORG:8443\n  github.com  \nnot a host entry'
    })
    expect(normalized.externalInterstitial).toBe(false)
    // 非法条目在客户端丢弃，服务端仍会对最终提交值 fail closed。
    expect(normalized.trustedDomains).toBe('github.com\nexample.org:8443')
  })

  test('rejects an allowlist above the reviewed entry limit and localizes the message', () => {
    const entries = Array.from({ length: 51 }, (_, index) => `host${index}.example.com`)
    expect(trustedDomainEntries(entries.join('\n')).length).toBe(51)
    expect(forumSettingsValidationError({
      ...createDefaultForumSettings(),
      trustedDomains: entries.join('\n')
    })).toBe('trustedDomainsTooMany')
    expect(forumSettingsValidationError({
      ...createDefaultForumSettings(),
      trustedDomains: entries.slice(0, 50).join('\n')
    })).toBeNull()

    for (const name of ['zh-CN', 'en-US']) {
      const settings = locale(name).admin.forum.settings
      expect(settings.validation.trustedDomainsTooMany.length).toBeGreaterThan(0)
      expect(settings.externalInterstitial.length).toBeGreaterThan(0)
      expect(settings.externalInterstitialHelp.length).toBeGreaterThan(0)
      expect(settings.trustedDomains.length).toBeGreaterThan(0)
      expect(settings.trustedDomainsHelp.length).toBeGreaterThan(0)
      expect(settings.trustedDomainsPlaceholder.length).toBeGreaterThan(0)
    }
  })
})
