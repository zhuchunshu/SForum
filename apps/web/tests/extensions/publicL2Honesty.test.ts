import { describe, expect, test } from 'bun:test'
import { readFileSync } from 'node:fs'

const source = (path: string) => readFileSync(new URL(path, import.meta.url), 'utf8')

describe('public L2 honesty disclosure', () => {
  const widget = source('../../app/components/SFExtensionWidget.vue')
  const en = source('../../i18n/locales/en-US.json')
  const zh = source('../../i18n/locales/zh-CN.json')
  const guide = source('../../../../docs/extensions/authoring-guide.md')

  test('widget shows dismissible honesty note for mounted trusted L2', () => {
    expect(widget).toContain('PUBLIC_FRONTEND_TRUST_NOTICE')
    expect(widget).toContain('data-testid="public-l2-honesty"')
    expect(widget).toContain('data-testid="public-l2-honesty-dismiss"')
    expect(widget).toContain('data-l2-trust')
    expect(widget).toContain("t('public.extensions.l2Honesty.title')")
    expect(widget).toContain('i-tabler-shield-exclamation')
    expect(widget).not.toMatch(/[\u{1F300}-\u{1FAFF}]/u)
  })

  test('ships bilingual honesty copy and authoring docs', () => {
    for (const locale of [en, zh]) {
      expect(locale).toContain('"l2Honesty"')
      expect(locale).toContain('"title"')
      expect(locale).toContain('"body"')
      expect(locale).toContain('"dismiss"')
    }
    expect(guide).toContain('Public L2 honesty')
    expect(guide).toContain('fully_trusted_browser_code')
    expect(guide).toContain('not a sandbox')
  })
})
