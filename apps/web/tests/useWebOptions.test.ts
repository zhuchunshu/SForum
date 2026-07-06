import { describe, expect, test } from 'bun:test'

import {
  buildCustomAppearanceThemeValue,
  cloneFooterLinks,
  enabledOptionValue,
  applySEOTitleTemplate,
  isSEOIndexingAllowed,
  normalizeAppearanceThemeValue,
  normalizeEnabledOption,
  recommendedAppearanceTheme,
  recommendedFooterCopyright,
  recommendedFooterLinks,
  resolveAltchaWidgetSettings,
  normalizeSEOVerificationToken,
  parseSEORobotsPathList,
  resolveSEOSettings,
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

  test('exports recommended personalization defaults', () => {
    const links = cloneFooterLinks(recommendedFooterLinks)

    expect(recommendedAppearanceTheme).toBe('pine_teal')
    expect(recommendedFooterCopyright['zh-CN']).toContain('{year}')
    expect(recommendedFooterCopyright['en-US']).toContain('{siteName}')
    expect(links.map(link => link.key)).toEqual(['terms', 'privacy', 'guidelines'])
    links[0].labels['zh-CN'] = 'Changed'
    expect(recommendedFooterLinks[0].labels['zh-CN']).toBe('服务条款')
  })
})

describe('human verification option helpers', () => {
  test('normalizes enabled scenario values', () => {
    expect(normalizeEnabledOption('enabled')).toBe(true)
    expect(normalizeEnabledOption('true')).toBe(true)
    expect(normalizeEnabledOption('0', true)).toBe(false)
    expect(normalizeEnabledOption('unexpected', true)).toBe(true)
    expect(enabledOptionValue(false)).toBe('disabled')
  })

  test('resolves ALTCHA widget settings with safe defaults', () => {
    const settings = resolveAltchaWidgetSettings({
      'human_verification.altcha.widget.type': 'switch',
      'human_verification.altcha.widget.auto': 'onfocus',
      'human_verification.altcha.widget.display': 'floating',
      'human_verification.altcha.widget.hide_logo': 'disabled',
      'human_verification.altcha.widget.hide_footer': 'enabled',
      'human_verification.altcha.widget.workers': '4',
      'human_verification.altcha.widget.min_duration_ms': '1200'
    })

    expect(settings.type).toBe('switch')
    expect(settings.auto).toBe('onfocus')
    expect(settings.display).toBe('floating')
    expect(settings.hideLogo).toBe(false)
    expect(settings.hideFooter).toBe(true)
    expect(settings.workers).toBe(4)
    expect(settings.minDuration).toBe(1200)

    const fallback = resolveAltchaWidgetSettings({
      'human_verification.altcha.widget.type': 'button',
      'human_verification.altcha.widget.auto': 'always',
      'human_verification.altcha.widget.display': 'modal',
      'human_verification.altcha.widget.workers': '99',
      'human_verification.altcha.widget.min_duration_ms': '-1'
    })
    expect(fallback.type).toBe('checkbox')
    expect(fallback.auto).toBe('off')
    expect(fallback.display).toBe('standard')
    expect(fallback.workers).toBe(2)
    expect(fallback.minDuration).toBe(500)
  })
})

describe('seo option helpers', () => {
  test('resolves runtime SEO settings with safe defaults', () => {
    const settings = resolveSEOSettings({
      'seo.twitter_card': 'summary',
      'seo.twitter_site': 'sforum_app',
      'seo.robots.extra_disallow': '/admin\n/private\n/admin',
      'seo.allow_indexing': 'enabled'
    })

    expect(settings.twitterCard).toBe('summary')
    expect(settings.twitterSite).toBe('@sforum_app')
    expect(settings.robotsExtraDisallow).toEqual(['/admin', '/private'])
    expect(isSEOIndexingAllowed(settings, 'http://127.0.0.1:3000')).toBe(false)
    expect(isSEOIndexingAllowed(settings, 'https://forum.example.com')).toBe(true)
  })

  test('normalizes SEO token, robots paths, and title templates', () => {
    expect(normalizeSEOVerificationToken(' google-token ')).toBe('google-token')
    expect(normalizeSEOVerificationToken('<script>')).toBe('')
    expect(parseSEORobotsPathList('/ok\nrelative\n//bad\n/path?q=1')).toEqual(['/ok', '/path?q=1'])
    expect(applySEOTitleTemplate('帖子标题', '{title} · {siteName}', 'SForum')).toBe('帖子标题 · SForum')
    expect(applySEOTitleTemplate('帖子标题', '', 'SForum')).toBe('帖子标题 - SForum')
  })
})
