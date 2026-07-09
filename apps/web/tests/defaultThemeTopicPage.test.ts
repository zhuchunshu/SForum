import { describe, expect, test } from 'bun:test'
import { existsSync, readFileSync } from 'node:fs'

const topicPage = () => readFileSync(
  new URL('../../../extensions/builtin/themes/sforum-default/layer/app/pages/t/[...path].vue', import.meta.url),
  'utf8'
)

const sourceFile = (path: string) => readFileSync(new URL(path, import.meta.url), 'utf8')
const themeCss = () => readFileSync(
  new URL('../../../extensions/builtin/themes/sforum-default/layer/app/assets/css/sforum-theme.css', import.meta.url),
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

  test('registers the highlight directive during SSR for rendered forum content', () => {
    const renderedContentSources = [
      topicPage(),
      sourceFile('../app/components/SFComment.vue'),
      sourceFile('../app/components/SFEditor.vue')
    ]
    const serverPluginPath = new URL('../app/plugins/highlight.server.ts', import.meta.url)

    expect(renderedContentSources.some(source => source.includes('v-highlight'))).toBe(true)
    expect(existsSync(serverPluginPath)).toBe(true)

    const serverPluginSource = readFileSync(serverPluginPath, 'utf8')
    expect(serverPluginSource).toContain("vueApp.directive('highlight'")
    expect(serverPluginSource).toContain('getSSRProps')
  })

  test('uses the fused reading layout, action rail, summary dock, and redesigned comments', () => {
    const source = topicPage()
    const css = themeCss()

    expect(source).toContain('sforum-topic-page')
    expect(source).toContain('sforum-topic-page__shell')
    expect(source).toContain('sforum-topic-page__action-rail')
    expect(source).toContain('sforum-topic-page__article')
    expect(source).toContain('sforum-topic-page__summary')
    expect(source).toContain('sforum-topic-comments')
    expect(source).toContain('sforum-topic-comments__card')

    expect(css).toContain('.sforum-topic-page__shell')
    expect(css).toContain('.sforum-topic-page__action-rail')
    expect(css).toContain('.sforum-topic-page__article')
    expect(css).toContain('.sforum-topic-page__summary')
    expect(css).toContain('.sforum-topic-comments__card')
    expect(css).toContain('.sf-comment__reply-to')
  })
})
