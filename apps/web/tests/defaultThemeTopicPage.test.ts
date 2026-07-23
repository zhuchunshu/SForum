import { describe, expect, test } from 'bun:test'
import { existsSync, readFileSync } from 'node:fs'

const topicPage = () => readFileSync(
  new URL('../../../apps/web/app/components/SFTopicShowPage.vue', import.meta.url),
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
  'SFTopicReplyComposer',
  'SFReportDialog'
] as const

describe('default theme V32 topic page contract', () => {
  test('keeps the core reading path eager with global navigation feedback', () => {
    const app = sourceFile('../app/app.vue')
    const themeTemplate = sourceFile('../app/components/SFThemeTemplate.vue')
    const avatar = sourceFile('../app/components/SFAvatar.vue')
    const heading = sourceFile('../app/components/SFTopicHeading.vue')
    const comment = sourceFile('../app/components/SFComment.vue')

    expect(app).toContain('<NuxtLoadingIndicator')
    expect(app).toContain('color="var(--sf-accent)"')
    expect(themeTemplate).toContain("import SFTopicShowPage from './SFTopicShowPage.vue'")
    expect(themeTemplate).toContain("'forum.component.topic_show': SFTopicShowPage")
    expect(themeTemplate).not.toContain("import('./SFTopicShowPage.vue')")
    expect(avatar).toContain("loading: 'lazy'")
    expect(heading).toContain('loading="eager"')
    expect(comment).not.toContain('loading="eager"')
  })

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
    expect(source).toContain("useRequestHeaders(['accept']).accept")
    expect(source).toContain("includes('text/html')")
    expect(source).toContain('if (!topic.value || !canNormalizeTopicURL)')
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

  test('uses a full-width three-column shell with left nav and topic side card', () => {
    const source = topicPage()
    const themeTopicTpl = sourceFile('../../../extensions/builtin/themes/sforum-default/templates/topic-show.html')

    expect(source).toContain('data-layout="fullwidth-3col"')
    expect(source).toContain('sforum-topic-page__layout')
    expect(source).toContain('sforum-topic-page__layout--with-side')
    expect(source).toContain('sforum-topic-page__sidebar')
    expect(source).toContain('<SFHomeNavigation')
    expect(source).toContain('navigation-mode="route"')
    expect(source).toContain('sforum-topic-page__shell')
    expect(source).toContain('sforum-topic-page__reading')
    expect(source).toContain('sforum-topic-page__post-card')
    expect(source).toContain('sforum-topic-page__actions')
    expect(source).toContain('<SFTopicSideCard')
    expect(source).toContain(':first-comment-id="comments[0]?.id"')
    expect(source).toContain(':latest-comment-id="comments[comments.length - 1]?.id"')
    expect(source).toContain('sf-comment-stream-controls__latest')
    const replyComposer = sourceFile('../app/components/SFTopicReplyComposer.vue')
    expect(source).toContain('<SFTopicReplyComposer')
    expect(replyComposer).toContain('<LazySFEditor')
    // 评论输入始终展开，无折叠态 / open 开关
    expect(replyComposer).not.toContain('v-if="open"')
    expect(replyComposer).not.toContain('open: boolean')
    expect(source).not.toContain('replyComposerOpen')
    expect(replyComposer).toContain('compact')
    expect(replyComposer).toContain("t('topicDetail.markdownSupported')")
    // 高级回复：右上角文字链 → 完整编辑器独立页
    expect(replyComposer).toContain("t('topicDetail.advancedReply')")
    expect(replyComposer).toContain('sforum-topic-comments__reply-advanced')
    expect(source).toContain('forumTopicAdvancedReplyPath')
    expect(source).toContain(':advanced-to="advancedReplyTo"')
    expect(sourceFile('../app/pages/topics/reply.vue')).toContain('forum.topic.reply')
    const advancedReplyPage = sourceFile('../app/components/SFTopicReplyPage.vue')
    expect(advancedReplyPage).toContain('<LazySFEditor')
    // 完整编辑器：不传 compact prop
    expect(advancedReplyPage).not.toContain(':compact')
    expect(advancedReplyPage).not.toContain('compact\n')
    expect(sourceFile('../app/components/SFEditor.vue')).toContain('sf-editor--compact')
    expect(sourceFile('../app/components/SFEditor.vue')).toContain("emit('cancel')")
    expect(source).toContain('showTopicSide')
    expect(source).toContain('shareTopic')
    expect(source).toContain('listCategoryGroups')
    expect(source).toContain('lazy: true')
    expect(themeTopicTpl).toContain('data-layout="fullwidth-3col"')
    expect(themeTopicTpl).toContain('sf-theme-shell--fullwidth-3col')
    expect(source).not.toContain('SFTopicProgressRail')
    expect(source).not.toContain('sforum-topic-page__action-rail')
    expect(source).not.toContain('statsTitle')
    // 左栏导航后页面略增；硬警告仍约 1200 行
    expect(source.split('\n').length).toBeLessThan(1200)
  })

  test('keeps mutation errors persistent and fixes the default discussion to a flat stream', () => {
    const source = topicPage()

    expect(source).not.toContain('watch(showActionError')
    expect(source).toContain(':presentation="commentView"')
    expect(source).toContain(':depth="0"')
    expect(source).toContain(':collapse-from-depth="2"')
    expect(source).toContain(':floor-label="commentFloor(comment)"')
    expect(source).toContain('commentFloorLabelsById')
    expect(source).not.toContain('#{{ replyingTo.id }}')
    expect(source).toContain("const commentView = ref<'flat'>('flat')")
    expect(source).not.toContain('<SFCommentStreamControls')
    const commentRequest = source.slice(source.indexOf('const commentQuery'), source.indexOf('function emptyCommentList'))
    expect(commentRequest).not.toContain('perPage')
    expect(source).toContain('commentData.value.perPage')
    expect(source).toContain('replyingTo.value?.id')
    expect(source).toContain('copyCommentLink')
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
    expect(source).toContain('<SFContentColumnFooter')
  })

  test('registers the full-width three-column topic stylesheet', () => {
    const config = themeFile('nuxt.config.ts')
    const css = themeFile('app/assets/css/sforum-topic.css')
    const themePkgCss = sourceFile('../../../extensions/builtin/themes/sforum-default/assets/theme.css')

    expect(config).toContain('sforum-topic.css')
    expect(css).toContain('.sforum-topic-page__layout--with-side')
    expect(css).toContain('var(--sf-public-sidebar-width)')
    expect(css).toContain('var(--sf-public-right-rail-width)')
    expect(css).toContain('.sforum-topic-page__sidebar')
    expect(css).toContain('.sforum-topic-page__post-card')
    expect(css).toContain('padding: 0 18px;')
    expect(css).toContain('.sforum-topic-page .sf-topic-heading__title')
    expect(css).toContain('.sf-topic-side-card')
    expect(css).toContain('background: var(--sf-public-surface)')
    expect(css).toContain('overflow-wrap: anywhere')
    expect(css).toContain('.sforum-topic-page__main {\n    height: 100%;\n    min-height: 0;\n    overflow-y: auto;')
    expect(css).toContain('.sforum-topic-page__sidebar,\n  .sforum-topic-page__side {\n    position: static;')
    expect(css).toContain('overflow: hidden;')
    expect(css).toContain('@media (max-width: 1180px)')
    expect(css).toContain('@media (max-width: 960px)')
    expect(themePkgCss).toContain('.sforum-topic-page__layout--with-side')
    expect(themePkgCss).toContain('fullwidth-3col')
    expect(css).not.toContain('.sforum-topic-page__action-rail')
    expect(css).not.toContain('.sforum-topic-page__summary')
  })
})
