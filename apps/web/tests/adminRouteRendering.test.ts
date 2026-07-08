import { describe, expect, test } from 'bun:test'
import { readFileSync } from 'node:fs'

describe('admin route rendering', () => {
  test('does not configure any route as SPA-only to avoid empty-shell white screens', () => {
    const config = readFileSync(new URL('../nuxt.config.ts', import.meta.url), 'utf8')
    // 全站不应有任何 ssr: false 路由——所有页面都 SSR，彻底杜绝空壳白屏。
    expect(config).not.toMatch(/['"`][^'"`]+['"`]\s*:\s*\{[^}]*\bssr\s*:\s*false/)
  })

  test('renders admin and component-preview pages via SSR', () => {
    const config = readFileSync(new URL('../nuxt.config.ts', import.meta.url), 'utf8')
    // admin 路由（模板字符串 key）不再带 ssr: false。
    expect(config).toMatch(/adminRoutePrefix[^}]*\*\*/)
    expect(config).not.toMatch(/adminRoutePrefix[^}]*ssr\s*:\s*false/)
    // 组件预览页不再带 ssr: false。
    expect(config).not.toMatch(/['"`]\/components['"`]\s*:\s*\{[^}]*ssr\s*:\s*false/)
  })

  test('disables route cache for admin pages', () => {
    const config = readFileSync(new URL('../nuxt.config.ts', import.meta.url), 'utf8')
    // admin 路由是动态 key（模板字符串），验证其配置块包含 cache: false。
    expect(config).toMatch(/adminRoutePrefix.*\*\*.*\{[^}]*cache\s*:\s*false/)
  })
})
