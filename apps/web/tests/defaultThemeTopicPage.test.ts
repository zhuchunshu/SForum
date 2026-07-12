import { describe, expect, test } from 'bun:test'
import { existsSync, readFileSync } from 'node:fs'

const topicPage = () => readFileSync(
  new URL('../../../apps/web/app/pages/t/[...path].vue', import.meta.url),
  'utf8'
)

const sourceFile = (path: string) => readFileSync(new URL(path, import.meta.url), 'utf8')
const themeFile = (path: string) => readFileSync(
  new URL(`../../../apps/web/${path}`, import.meta.url),
  'utf8'
)

const topicComponentNames = [
  'SFTopicHeading',
  'SFTopicSideCard',
  'SFTopicActionMenu',
  'SFCommentStreamControls',
  'SFReportDialog'
] as const

describe('default theme V32 topic page contract', () => {
  test('declares edit mode before immediate effects read it', () => {
    const source = topicPage()
    const isEditingDeclaration = source.indexOf('const isEditing = computed(')
    const canonicalRedirectEffect = source.indexOf('watchEffect(() => {')
    const seoRegistration = source.indexOf('useSForumSeo(computed(')

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
      expect(existsSync(new URL(`../../../apps/web/${path}`, import.meta.url))).toBe(true)
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
    expect(source).toContain('useSForumSeo(computed(')
    expect(source).toContain('sanitizeHtml(topic.content.htmlContent)')
    expect(source).toContain('v-highlight')
    expect(source).toContain('<SFTopicEditor')
    expect(source).toContain('applyTopicExtensionAction')
    // E2.2：评论行扩展动作与主题动作同一代理边界
    expect(source).toContain('applyCommentExtensionAction')
    expect(source).toContain('extensionActions')
  })

  test('uses a dual-column reading shell with topic info side card', () => {
    const source = topicPage()

    expect(source).toContain('sforum-topic-page__shell')
    expect(source).toContain('sforum-topic-page__reading')
    expect(source).toContain('sforum-topic-page__post-card')
    expect(source).toContain('sforum-topic-page__actions')
    expect(source).toContain('<SFTopicSideCard')
    expect(source).toContain('shareTopic')
    expect(source).not.toContain('SFTopicProgressRail')
    expect(source).not.toContain('sforum-topic-page__action-rail')
    expect(source).not.toContain('statsTitle')
    // SFPageOutlet wrap adds a few lines; keep the page scannable (hard warning ~1000).
    expect(source.split('\n').length).toBeLessThan(1010)
  })

  test('keeps mutation errors persistent and passes explicit comment presentation', () => {
    const source = topicPage()

    expect(source).not.toContain('watch(showActionError')
    expect(source).toContain(':presentation="commentView"')
    expect(source).toContain(':depth="0"')
    expect(source).toContain(':collapse-from-depth="2"')
    const commentRequest = source.slice(source.indexOf('const commentQuery'), source.indexOf('watch(commentView'))
    expect(commentRequest).not.toContain('perPage')
    expect(source).toContain('commentData.value.perPage')
  })

  test('uses API edit marks and the complete author lock policy', () => {
    const source = topicPage()
    const heading = themeFile('app/components/SFTopicHeading.vue')
    const commentMeta = source.slice(source.indexOf('function commentMeta'), source.indexOf('// 主题生命周期动作'))

    expect(source).toContain('const suffix = comment.edited ?')
    expect(commentMeta).not.toContain('updatedAt')
    expect(heading).toContain('v-if="topic.edited"')
    expect(source).toContain("webOption('forum.topics.allow_author_close_replies', 'enabled')")
    expect(source).toContain('can(FORUM_PERMISSIONS.topicEditOwn)')
    expect(source).toContain('topic.value?.authorUserId === reportUser.value?.id')
  })

  test('registers the V32 dual-column topic stylesheet', () => {
    const config = themeFile('nuxt.config.ts')
    const css = themeFile('app/assets/css/sforum-topic.css')

    expect(config).toContain('sforum-topic.css')
    expect(css).toContain('grid-template-columns: minmax(0, 1fr) 280px')
    expect(css).toContain('.sforum-topic-page__post-card')
    expect(css).toContain('.sforum-topic-page .sf-topic-heading__title')
    expect(css).toContain('.sf-topic-side-card')
    expect(css).toContain('background: var(--sf-public-surface)')
    expect(css).toContain('overflow-wrap: anywhere')
    expect(css).not.toContain('.sforum-topic-page__action-rail')
    expect(css).not.toContain('.sforum-topic-page__summary')
  })
})
