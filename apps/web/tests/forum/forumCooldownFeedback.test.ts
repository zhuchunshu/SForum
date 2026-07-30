import { describe, expect, test } from 'bun:test'
import { readFileSync } from 'node:fs'

const source = (path: string) => readFileSync(new URL(path, import.meta.url), 'utf8')

describe('forum cooldown feedback', () => {
  test('keeps topic and comment cooldown handling independent', () => {
    const cooldown = source('../../app/composables/forum/useForumCooldownError.ts')
    expect(cooldown).toContain("kind === 'topic' ? 'forum.topic_cooldown' : 'forum.comment_cooldown'")
    expect(cooldown).toContain('retryAfterSeconds')
    expect(cooldown).toContain('setInterval(tick, 1000)')
  })

  test('wires countdown feedback into all create surfaces without locking drafts', () => {
    const topic = source('../../app/components/forum/SFTopicComposerPage.vue')
    const comments = source('../../app/components/forum/SFTopicShowPage.vue')
    const quickReply = source('../../app/components/forum/SFTopicReplyComposer.vue')
    const drawer = source('../../app/composables/forum/useTopicCommentComposerDrawer.ts')
    const commentSubmission = source('../../app/composables/forum/useTopicCommentSubmission.ts')
    const editor = source('../../app/components/SFEditor.vue')

    expect(topic).toContain("useForumCooldownError('topic')")
    expect(quickReply).toContain('useTopicCommentSubmission')
    expect(drawer).toContain('useTopicCommentSubmission')
    expect(commentSubmission).toContain("useForumCooldownError('comment')")
    expect(topic).toContain(':submit-disabled="topicCooldownActive"')
    expect(quickReply).toContain(':submit-disabled="commentCooldownActive"')
    expect(comments).toContain(':submit-disabled="composerMode === \'edit\' ? false : commentCooldownActive"')
    expect(editor).toContain('disabled || submitDisabled || currentPayload.isEmpty')
  })
})
