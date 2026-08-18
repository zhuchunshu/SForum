import { describe, expect, test } from 'bun:test'
import { readFileSync } from 'node:fs'

const source = (relative: string) => readFileSync(new URL(relative, import.meta.url), 'utf8')
const coreThemeCss = source('../../app/assets/css/sforum-theme.css')
const defaultThemeCss = source('../../../../extensions/builtin/themes/sforum-default/assets/hybrid-forum.css')

function expectFullViewportDrawer(css: string) {
  expect(css).toMatch(/\.sforum-mobile-drawer__backdrop\s*\{[^}]*\binset:\s*0;/s)
  expect(css).toMatch(/\.sforum-mobile-drawer\s*\{[^}]*\btop:\s*0;/s)
  expect(css).not.toMatch(/\.sforum-mobile-drawer__backdrop\s*\{[^}]*\binset:\s*var\(--sf-public-topbar-height\)/s)
  expect(css).not.toMatch(/\.sforum-mobile-drawer\s*\{[^}]*\btop:\s*var\(--sf-public-topbar-height\)/s)
}

describe('public mobile drawer geometry', () => {
  test('covers the full viewport in Core fallback and the default runtime theme', () => {
    expectFullViewportDrawer(coreThemeCss)
    expectFullViewportDrawer(defaultThemeCss)
  })
})
