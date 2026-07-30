import { describe, expect, test } from 'bun:test'
import { readFileSync } from 'node:fs'
import type { ComputedRef } from 'vue'
import { useTopicSelectionQuoteReply } from '../../app/composables/forum/useTopicSelectionQuoteReply'
import {
  buildSelectionQuoteMarkdown,
  normalizeSelectionQuote,
  parseSelectionQuoteTarget,
  selectionQuoteToolbarPosition
} from '../../app/utils/forum/forumSelectionQuote'
import type { ForumComment } from '../../app/utils/forum/forumTaxonomy'

const source = (path: string) => readFileSync(new URL(path, import.meta.url), 'utf8')

describe('selection quote reply', () => {
  test('normalizes and safely inserts selected text as a Markdown blockquote', () => {
    expect(normalizeSelectionQuote('  first\t line\r\n\r\n second\u00A0  line  ')).toBe('first line\n\nsecond line')
    expect(buildSelectionQuoteMarkdown('first line\nsecond <img> & line')).toBe(
      '> first line\n> second &lt;img&gt; &amp; line\n\n'
    )
    expect(normalizeSelectionQuote('好'.repeat(501))).toHaveLength(500)
  })

  test('accepts topic and valid comment targets only', () => {
    expect(parseSelectionQuoteTarget('topic', undefined)).toEqual({ kind: 'topic' })
    expect(parseSelectionQuoteTarget('comment', '42')).toEqual({ kind: 'comment', commentId: 42 })
    expect(parseSelectionQuoteTarget('comment', '0')).toBeNull()
    expect(parseSelectionQuoteTarget('comment', 'unknown')).toBeNull()
  })

  test('positions the action in the scroll host document coordinates', () => {
    expect(selectionQuoteToolbarPosition({
      selectionRect: { left: 300, top: 240, width: 120, height: 24 },
      hostRect: { left: 180, top: 64 },
      hostScrollLeft: 0,
      hostScrollTop: 480,
      hostClientWidth: 640,
      toolbarWidth: 130,
      toolbarHeight: 36
    })).toEqual({ left: 115, top: 611, placement: 'above' })
  })

  test('routes topic and comment quotes without bypassing reply permission', () => {
    const comment = { id: 42, children: [] } as unknown as ForumComment
    const comments = { value: [comment] } as ComputedRef<ForumComment[]>
    const canReply = { value: true } as ComputedRef<boolean>
    const topicDrafts: string[] = []
    const commentDrafts: Array<{ id: number, draft: string }> = []
    const handleQuote = useTopicSelectionQuoteReply({
      comments,
      canReply,
      openTopicReply: draft => topicDrafts.push(draft || ''),
      startReply: (target, draft) => commentDrafts.push({ id: target.id, draft: draft || '' })
    })

    handleQuote({ target: { kind: 'topic' }, markdown: '> topic\n\n' })
    handleQuote({ target: { kind: 'comment', commentId: 42 }, markdown: '> comment\n\n' })
    expect(topicDrafts).toEqual(['> topic\n\n'])
    expect(commentDrafts).toEqual([{ id: 42, draft: '> comment\n\n' }])

    canReply.value = false
    handleQuote({ target: { kind: 'topic' }, markdown: '> denied\n\n' })
    expect(topicDrafts).toHaveLength(1)
  })

  test('wires eligible rendered content to the existing composer paths', () => {
    const page = source('../../app/components/forum/SFTopicShowPage.vue')
    const comment = source('../../app/components/forum/SFComment.vue')
    const action = source('../../app/components/forum/SFSelectionQuoteAction.vue')
    const composer = source('../../app/composables/forum/useTopicCommentComposerDrawer.ts')

    expect(page).toContain('<SFSelectionQuoteAction')
    expect(page).toContain(':enabled="canReplyToComments"')
    expect(page).toContain('data-selection-quote-source="topic"')
    expect(page).toContain('useTopicSelectionQuoteReply({ comments, canReply: canReplyToComments, startReply, openTopicReply: openAdvancedReply })')
    expect(comment).toContain('data-selection-quote-source="comment"')
    expect(comment).toContain(':data-selection-quote-comment-id="comment?.id"')
    expect(composer).toContain("function startReply(comment: ForumComment, initialDraft = '')")
    expect(composer).toContain('replyMarkdown.value = initialDraft')
    const selectionReply = source('../../app/composables/forum/useTopicSelectionQuoteReply.ts')
    expect(selectionReply).toContain('options.openTopicReply(request.markdown)')
    expect(selectionReply).toContain('options.startReply(comment, request.markdown)')
    expect(action).toContain('position: absolute')
    expect(action).not.toContain('position: fixed')
  })
})
