import { describe, expect, test } from 'bun:test'
import { readFileSync } from 'node:fs'

const topicPage = () => readFileSync(
  new URL('../../../extensions/builtin/themes/sforum-default/layer/app/pages/t/[...path].vue', import.meta.url),
  'utf8'
)

describe('default theme topic page contract', () => {
  test('declares edit mode before immediate effects read it', () => {
    const source = topicPage()
    const isEditingDeclaration = source.indexOf('const isEditing = computed(')
    const canonicalRedirectEffect = source.indexOf('watchEffect(() => {')
    const seoRegistration = source.indexOf('useSForumSeo({')

    expect(isEditingDeclaration).toBeGreaterThan(-1)
    expect(canonicalRedirectEffect).toBeGreaterThan(-1)
    expect(seoRegistration).toBeGreaterThan(-1)
    expect(isEditingDeclaration).toBeLessThan(canonicalRedirectEffect)
    expect(isEditingDeclaration).toBeLessThan(seoRegistration)
  })
})
