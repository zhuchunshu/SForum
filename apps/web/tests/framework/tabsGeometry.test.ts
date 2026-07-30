import { describe, expect, test } from 'bun:test'
import { readFileSync } from 'node:fs'

const componentsCss = readFileSync(
  new URL('../../app/assets/css/sforum-components.css', import.meta.url),
  'utf8'
)

describe('SFTabs geometry', () => {
  test('does not collapse inside scrollable flex columns', () => {
    expect(componentsCss).toMatch(/\.sf-tabs\s*\{[^}]*flex:\s*0 0 auto;/s)
    expect(componentsCss).toMatch(/\.sf-tabs__item\s*\{[^}]*flex:\s*0 0 auto;/s)
  })
})
