import { describe, expect, test } from 'bun:test'

import { useErrorPageShell } from '~/composables/errors/useErrorPageShell'
import {
  createFallbackErrorPageTranslator,
  interpolateErrorPageText,
  isErrorPageI18nReady
} from '~/utils/errors/errorPageShell'

const unused = () => {
  throw new Error('i18n composable must not be called when the plugin did not finish installing')
}

describe('error page i18n readiness', () => {
  test('requires both the composer and the locale head provider', () => {
    expect(isErrorPageI18nReady({})).toBe(false)
    expect(isErrorPageI18nReady({ i18n: { t: () => '' } })).toBe(false)
    expect(isErrorPageI18nReady({ localeHead: () => ({}) })).toBe(false)
    expect(isErrorPageI18nReady({ i18n: { t: () => '' }, localeHead: () => ({}) })).toBe(true)
  })

  test('degrades without touching the i18n composables', () => {
    const shell = useErrorPageShell({}, { useI18n: unused, useLocaleHead: unused })

    expect(shell.i18nReady).toBe(false)
    expect(shell.localeHead.value).toEqual({ htmlAttrs: {}, link: [], meta: [] })
    expect(typeof shell.t('errors.page.notFound.title')).toBe('string')
    expect(shell.t('errors.page.notFound.title')).not.toBe('')
  })

  test('treats a half-installed i18n plugin as not ready', () => {
    const shell = useErrorPageShell(
      { $i18n: { t: () => 'translated' } },
      { useI18n: unused, useLocaleHead: unused }
    )

    expect(shell.i18nReady).toBe(false)
  })

  test('delegates to the real composables once i18n is installed', () => {
    const localeHeadCalls: Array<{ dir: boolean, lang: boolean, seo: boolean }> = []
    const shell = useErrorPageShell(
      { $i18n: { t: () => 'translated' }, $localeHead: () => ({}) },
      {
        useI18n: () => ({ t: (key: string) => `t:${key}` }),
        useLocaleHead: (options) => {
          localeHeadCalls.push(options)
          return { value: { htmlAttrs: { lang: 'zh-CN' }, link: [], meta: [] } }
        }
      }
    )

    expect(shell.i18nReady).toBe(true)
    expect(shell.t('errors.page.notFound.title')).toBe('t:errors.page.notFound.title')
    expect(localeHeadCalls).toEqual([{ dir: true, lang: true, seo: false }])
    expect(shell.localeHead.value.htmlAttrs.lang).toBe('zh-CN')
  })
})

describe('error page fallback translator', () => {
  test('interpolates named values and keeps unknown placeholders', () => {
    expect(interpolateErrorPageText('{siteName} 不可用', { siteName: 'SForum' })).toBe('SForum 不可用')
    expect(interpolateErrorPageText('{siteName} 不可用', {})).toBe('{siteName} 不可用')
    expect(interpolateErrorPageText('纯文本')).toBe('纯文本')
  })

  test('warns once and never throws', () => {
    const warnings: string[] = []
    const t = createFallbackErrorPageTranslator(message => warnings.push(message))

    expect(t('errors.page.notFound.title')).toBe('title')
    expect(t('errors.page.forbidden.description')).toBe('description')
    expect(warnings).toHaveLength(1)
  })
})
