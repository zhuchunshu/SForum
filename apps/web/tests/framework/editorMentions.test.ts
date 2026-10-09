import { describe, expect, it } from 'bun:test'
import { Editor } from '@tiptap/core'
import {
  createSFEditorExtensions
} from '../../app/utils/sfEditor'
import {
  createSForumMentionExtensions,
  sforumMentionRanges,
  sforumMentionSuggestionAllowed
} from '../../app/utils/editor/mentions'
import { forumMentionTokens } from '../../app/utils/forum/forumMentions'

function createEditor(content: string, extra: unknown[] = []) {
  return new Editor({
    content,
    contentType: 'markdown',
    extensions: [
      ...createSFEditorExtensions({
        placeholder: 'p',
        maxCharacters: 1000,
        resourceKind: 'topic'
      }) as never[],
      ...extra as never[]
    ]
  })
}

const fakeMentions = {
  fetch: async (query: string) => [{ id: 1, username: `${query}user`, displayName: '测试用户' }],
  labels: { empty: '没有匹配的用户' }
}

describe('mention editor extensions', () => {
  it('suggestion + decoration plugins 仅在 mentions 注入时注册', () => {
    const plain = createEditor('plain')
    expect(plain.extensionManager.extensions.some(ext => ext.name === 'sforumMentionSuggestion')).toBe(false)
    expect(plain.extensionManager.extensions.some(ext => ext.name === 'sforumMentionDecorations')).toBe(false)
    plain.destroy()

    const mentioned = new Editor({
      content: 'plain',
      extensions: [
        ...createSFEditorExtensions({ placeholder: 'p', maxCharacters: 1000, preset: 'full' }) as never[],
        ...createSForumMentionExtensions(fakeMentions)
      ]
    })
    expect(mentioned.extensionManager.extensions.some(ext => ext.name === 'sforumMentionSuggestion')).toBe(true)
    expect(mentioned.extensionManager.extensions.some(ext => ext.name === 'sforumMentionDecorations')).toBe(true)
    mentioned.destroy()
  })

  it('装饰区间只覆盖正文提及，跳过行内代码与代码块', () => {
    const editor = createEditor('hi @alice ok `@bob`\n\n```\n@carol\n```')
    const ranges = sforumMentionRanges(editor.state.doc)
    expect(ranges).toHaveLength(1)
    expect(editor.state.doc.textBetween(ranges[0]!.from, ranges[0]!.to)).toBe('@alice')
    editor.destroy()
  })

  it('@ 左邻是字母/数字/下划线时不识别为提及（邮箱不误伤）', () => {
    const editor = createEditor('mail me at user@example.com please @someone')
    const ranges = sforumMentionRanges(editor.state.doc)
    expect(ranges.map(range => editor.state.doc.textBetween(range.from, range.to))).toEqual(['@someone'])
    editor.destroy()
  })

  it('代码上下文不允许触发候选', () => {
    const editor = createEditor('plain `code`\n\n```\nblock\n```')
    const doc = editor.state.doc
    // 行内代码内部
    doc.descendants((node, pos) => {
      if (node.isText && node.text === 'code') {
        expect(sforumMentionSuggestionAllowed(editor.state, pos)).toBe(false)
      }
      if (node.isText && node.text === 'plain') {
        expect(sforumMentionSuggestionAllowed(editor.state, pos)).toBe(true)
      }
      return true
    })
    doc.descendants((node, pos) => {
      if (node.type.name === 'codeBlock') {
        expect(sforumMentionSuggestionAllowed(editor.state, pos + 1)).toBe(false)
      }
      return true
    })
    editor.destroy()
  })

  it('command 把候选用户名落成纯文本，Markdown 序列化保持 @username', async () => {
    const editor = new Editor({
      content: 'hello ',
      extensions: [
        ...createSFEditorExtensions({ placeholder: 'p', maxCharacters: 1000, preset: 'full' }) as never[],
        ...createSForumMentionExtensions(fakeMentions)
      ]
    })
    const end = editor.state.doc.content.size
    editor.chain().focus().insertContentAt({ from: end, to: end }, '@alice ').run()
    const markdown = editor.getMarkdown()
    expect(markdown).toContain('@alice')
    // 与后端解析同源 token：序列化后的正文仍能提取到 alice。
    expect(forumMentionTokens(markdown).map(token => token.username)).toContain('alice')
    editor.destroy()
  })
})
