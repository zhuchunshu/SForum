import { describe, expect, test } from 'bun:test'
import { topicCommentEditorContentIsMeaningful } from '../../app/composables/forum/useTopicCommentSubmission'

describe('topic comment editor content validation', () => {
  test('accepts an image-only native document when text serializers are empty', () => {
    expect(topicCommentEditorContentIsMeaningful({
      text: '',
      markdown: '',
      native: {
        type: 'doc',
        content: [{
          type: 'image',
          attrs: {
            src: '/media/attachments/0123456789abcdef0123456789abcdef',
            attachmentId: 42,
            attachmentPublicId: '0123456789abcdef0123456789abcdef'
          }
        }]
      },
      attachmentIds: [42]
    })).toBe(true)
  })

  test('keeps text, fallback Markdown, and empty documents distinct', () => {
    expect(topicCommentEditorContentIsMeaningful({ text: '回复' })).toBe(true)
    expect(topicCommentEditorContentIsMeaningful({ text: '', markdown: '' }, 'fallback')).toBe(true)
    expect(topicCommentEditorContentIsMeaningful({
      text: '',
      markdown: '',
      native: { type: 'doc', content: [{ type: 'paragraph' }] }
    })).toBe(false)
  })
})
