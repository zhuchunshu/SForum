import { describe, expect, test } from 'bun:test'
import { existsSync, readFileSync } from 'node:fs'

const source = (path: string) => readFileSync(new URL(`../../${path}`, import.meta.url), 'utf8')

const comment = () => source('app/components/forum/SFComment.vue')
const topicPage = () => source('app/components/forum/SFTopicShowPage.vue')
const mentionContent = () => source('app/components/forum/SFMentionContent.vue')
const mentionPreview = () => source('app/components/forum/SFUserMentionPreview.vue')
const mentionStyles = () => source('app/assets/css/sforum-mention.css')

describe('forum mention content contract', () => {
  test('routes comment and topic bodies through the shared mention container', () => {
    expect(comment()).toContain("import SFMentionContent from '~/components/forum/SFMentionContent.vue'")
    expect(comment()).toContain('<SFMentionContent')
    expect(comment()).toContain(':html="sanitizeHtml(htmlContent)"')
    expect(comment()).toContain('v-highlight')

    expect(topicPage()).toContain("import SFMentionContent from '~/components/forum/SFMentionContent.vue'")
    expect(topicPage()).toContain("import SFUserMentionPreview from '~/components/forum/SFUserMentionPreview.vue'")
    expect(topicPage()).toContain('<SFMentionContent')
    expect(topicPage()).toContain(':html="sanitizeHtml(topic.content.htmlContent)"')
    expect(topicPage()).toContain('v-highlight')
    // 提及预览宿主是整页单例：话题正文与评论区的 @ 共用同一张卡片。
    expect(topicPage()).toContain('<SFUserMentionPreview />')
  })

  test('declares the mention content responsibilities in one component', () => {
    const content = mentionContent()

    expect(content).toContain('v-mention="mentionHref"')
    expect(content).toContain('v-html="html"')
    expect(content).toContain('mentionUsernameFromEvent(event.target)')
    expect(content).toContain('openForumMentionPreview(')
    expect(content).toContain('event.preventDefault()')
    // 修饰键与中键保留浏览器默认的「新标签打开资料页」行为。
    expect(content).toContain('event.metaKey || event.ctrlKey || event.shiftKey || event.altKey')
    expect(content).toContain('forumUserProfilePath(username)')
    expect(content).toContain('previewAvailable')
  })

  test('reuses the comment avatar preview card for mentions', () => {
    const preview = mentionPreview()

    expect(preview).toContain("import SFCommentUserPreview from '~/components/forum/SFCommentUserPreview.vue'")
    expect(preview).toContain('<SFCommentUserPreview')
    expect(preview).toContain('useForumMentionPreviewHost')
    expect(preview).toContain(':profile-path="state.target.profilePath"')
    expect(preview).toContain('<Teleport to="body">')
  })

  test('registers the mention directive for client decoration with an SSR placeholder', () => {
    expect(existsSync(new URL('../../app/plugins/mention.client.ts', import.meta.url))).toBe(true)
    expect(existsSync(new URL('../../app/plugins/mention.server.ts', import.meta.url))).toBe(true)

    const client = source('app/plugins/mention.client.ts')
    expect(client).toContain("vueApp.directive('mention'")
    expect(client).toContain('linkifyForumMentions(el')
    expect(client).toContain('getSSRProps')

    const server = source('app/plugins/mention.server.ts')
    expect(server).toContain("vueApp.directive('mention'")
    expect(server).toContain('getSSRProps')
  })

  test('keeps the mention token styling theme-token driven and loaded by Nuxt', () => {
    const styles = mentionStyles()

    expect(styles).toContain('.sf-mention {')
    expect(styles).toContain('background: var(--sf-accent-soft')
    expect(styles).toContain('color: var(--sf-accent)')
    expect(styles).toContain('.sf-mention-preview-layer {')
    expect(styles).toContain('position: fixed')
    expect(source('nuxt.config.ts')).toContain("'~/assets/css/sforum-mention.css'")
  })

  test('keeps action-menu label tables with the menu model', () => {
    const presentation = source('app/utils/forum/forumTopicPresentation.ts')

    expect(presentation).toContain('export function buildTopicActionLabels(')
    expect(presentation).toContain('export function buildCommentActionLabels(')
    expect(topicPage()).toContain('buildTopicActionLabels(t)')
    expect(topicPage()).toContain('buildCommentActionLabels(t,')
    expect(topicPage()).not.toContain("edit: t('topicDetail.edit')")
  })
})
