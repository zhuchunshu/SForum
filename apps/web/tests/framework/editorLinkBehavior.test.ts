import { afterEach, describe, expect, it } from 'bun:test'
import type { Editor } from '@tiptap/core'
import {
  collectTextMarks,
  createTestEditor,
  pasteClipboard,
  setSelection,
  typeText
} from '../helpers/editorClipboard'
import { insertSForumLink } from '../../app/utils/editor/link'

let editor: Editor | undefined

afterEach(() => {
  editor?.destroy()
  editor = undefined
})

describe('editor link behavior', () => {
  it('粘贴 URL 后输入空格和普通文字，后续文字不属于链接', () => {
    editor = createTestEditor()
    pasteClipboard(editor, { text: 'https://example.com' })
    typeText(editor, ' 普通内容')

    expect(collectTextMarks(editor)).toEqual([
      { text: 'https://example.com', marks: ['link'] },
      { text: ' 普通内容', marks: [] }
    ])
    expect(editor.getMarkdown()).toBe('[https://example.com](https://example.com) 普通内容')
  })

  it('手输 URL 加空格仍然自动链接，空格后的文字不续写', () => {
    editor = createTestEditor()
    typeText(editor, 'see https://example.com 普通内容')

    expect(collectTextMarks(editor)).toEqual([
      { text: 'see ', marks: [] },
      { text: 'https://example.com', marks: ['link'] },
      { text: ' 普通内容', marks: [] }
    ])
  })

  it('空选区插入链接后继续输入，后续文字不带 link mark', () => {
    editor = createTestEditor()
    insertSForumLink(editor, 'https://example.com')
    typeText(editor, ' 普通内容')

    expect(collectTextMarks(editor)).toEqual([
      { text: 'https://example.com', marks: ['link'] },
      { text: ' 普通内容', marks: [] }
    ])
  })

  it('在已有链接内更新 href，链接之后的输入不再续写', () => {
    editor = createTestEditor('[示例](https://old.example) tail')
    setSelection(editor, 1, 3)
    editor.chain().focus().extendMarkRange('link').setLink({ href: 'https://new.example' }).run()

    expect(editor.getMarkdown()).toBe('[示例](https://new.example) tail')

    setSelection(editor, 3)
    typeText(editor, '!')
    expect(collectTextMarks(editor)).toEqual([
      { text: '示例', marks: ['link'] },
      { text: '! tail', marks: [] }
    ])
  })

  it('basic-field 同样不再续写链接', () => {
    editor = createTestEditor({ preset: 'basic-field' })
    pasteClipboard(editor, { text: 'https://example.com' })
    typeText(editor, ' 普通内容')

    expect(collectTextMarks(editor)).toEqual([
      { text: 'https://example.com', marks: ['link'] },
      { text: ' 普通内容', marks: [] }
    ])
  })
})
