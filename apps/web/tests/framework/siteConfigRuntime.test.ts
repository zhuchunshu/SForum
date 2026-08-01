import { describe, expect, test } from 'bun:test'

const nuxtConfig = await Bun.file(new URL('../../nuxt.config.ts', import.meta.url)).text()
const productionEntrypoint = await Bun.file(new URL('../../scripts/production-entrypoint.sh', import.meta.url)).text()

describe('runtime site URL contract', () => {
  test('declares the runtime URL slots consumed by the container environment', () => {
    expect(nuxtConfig).toMatch(/runtimeConfig:\s*{\s*\/\/[^]*?site:\s*{\s*url: appUrl\s*}/)
    expect(nuxtConfig).toMatch(/public:\s*{[^]*?i18n:\s*{\s*baseUrl: appUrl\s*}/)
    expect(productionEntrypoint).toContain('export NUXT_SITE_URL="${NUXT_SITE_URL:-$canonical_url}"')
    expect(productionEntrypoint).toContain('export NUXT_PUBLIC_I18N_BASE_URL="${NUXT_PUBLIC_I18N_BASE_URL:-$canonical_url}"')
  })

  test('keeps build-time Site Config inputs free of a baked deployment URL', () => {
    const topLevelI18nConfig = nuxtConfig.match(/\n  i18n: \{([^]*?)\n  hooks: \{/)?.[1]

    expect(nuxtConfig).toMatch(/site:\s*{\s*name: appName\s*}/)
    expect(nuxtConfig).not.toMatch(/site:\s*{\s*name: appName,\s*url:/)
    expect(topLevelI18nConfig).toContain("defaultLocale: 'zh-CN'")
    expect(topLevelI18nConfig).not.toContain('baseUrl:')
  })
})
