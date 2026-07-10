import { describe, expect, test } from 'bun:test'
import { existsSync, readFileSync } from 'node:fs'

const topicPage = () => readFileSync(
  new URL('../../../extensions/builtin/themes/sforum-default/layer/app/pages/t/[...path].vue', import.meta.url),
  'utf8'
)

const sourceFile = (path: string) => readFileSync(new URL(path, import.meta.url), 'utf8')
const themeFile = (path: string) => readFileSync(
  new URL(`../../../extensions/builtin/themes/sforum-default/layer/${path}`, import.meta.url),
  'utf8'
)

const topicComponentNames = [
  'SFTopicHeading',
  'SFTopicProgressRail',
  'SFTopicActionMenu',
  'SFCommentStreamControls',
  'SFReportDialog'
] as const

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

  test('composes focused presentational components without moving route policy into them', () => {
    const source = topicPage()

    for (const name of topicComponentNames) {
      const path = `app/components/${name}.vue`
      expect(existsSync(new URL(`../../../extensions/builtin/themes/sforum-default/layer/${path}`, import.meta.url))).toBe(true)
      expect(source).toContain(`<${name}`)

      const component = themeFile(path)
      expect(component).not.toContain('useForumApi(')
      expect(component).not.toContain('useModerationApi(')
      expect(component).not.toContain('usePermissions(')
    }
  })

  test('preserves routing, SEO, rich-content, editor, and extension boundaries', () => {
    const source = topicPage()

    expect(source).toContain('topicPathLookupCandidates(')
    expect(source).toContain('navigateTo(target, { redirectCode: 301 })')
    expect(source).toContain('useSForumSeo({')
    expect(source).toContain('sanitizeHtml(topic.content.htmlContent)')
    expect(source).toContain('v-highlight')
    expect(source).toContain('<SFTopicEditor')
    expect(source).toContain('applyTopicExtensionAction')
  })

  test('uses one reading column and a true post-count progress rail', () => {
    const source = topicPage()

    expect(source).toContain('sforum-topic-page__shell')
    expect(source).toContain('sforum-topic-page__reading')
    expect(source).toContain(':total-posts="topic.commentCount + 1"')
    expect(source).not.toContain('sforum-topic-page__action-rail')
    expect(source).not.toContain('sforum-topic-page__summary')
    expect(source).not.toContain('statsTitle')
    expect(source.split('\n').length).toBeLessThan(1000)
  })

  test('keeps mutation errors persistent and passes explicit comment presentation', () => {
    const source = topicPage()

    expect(source).not.toContain('watch(showActionError')
    expect(source).toContain(':presentation="commentView"')
    expect(source).toContain(':depth="0"')
    expect(source).toContain(':collapse-from-depth="2"')
  })

  test('registers the focused 820px plus 190px responsive topic stylesheet', () => {
    const config = themeFile('nuxt.config.ts')
    const css = themeFile('app/assets/css/sforum-topic.css')

    expect(config).toContain('sforum-topic.css')
    expect(css).toContain('grid-template-columns: minmax(0, 820px) 190px')
    expect(css).toContain('position: sticky')
    expect(css).toContain('overflow-wrap: anywhere')
    expect(css).toContain('min-height: 40px')
    expect(css).not.toContain('.sforum-topic-page__action-rail')
    expect(css).not.toContain('.sforum-topic-page__summary')
  })
})
