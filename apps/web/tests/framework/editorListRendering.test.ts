import { afterEach, describe, expect, it } from 'bun:test'
import { Editor } from '@tiptap/core'
import { createSFEditorExtensions } from '../../app/utils/sfEditor'
import { SF_PURIFY_CONFIG } from '../../app/utils/sfSanitize'

let editor: Editor | undefined

afterEach(() => {
  editor?.destroy()
  editor = undefined
})

function countsFrom(root: ParentNode) {
  return {
    ul: root.querySelectorAll('ul').length,
    ol: root.querySelectorAll('ol').length,
    li: root.querySelectorAll('li').length,
    nestedUl: root.querySelectorAll('li ul').length,
    nestedOl: root.querySelectorAll('li ol').length
  }
}

describe('editor list rendering contract (write & preview)', () => {
  it('keeps ul, ol, and nested lists complete in the write DOM from Markdown', () => {
    editor = new Editor({
      extensions: createSFEditorExtensions({
        placeholder: 'body',
        maxCharacters: 1000
      }),
      content: [
        '- 项目符号一',
        '- 项目符号二',
        '  - 嵌套无序',
        '  - 嵌套无序二',
        '',
        '3. 从三开始的有序列表',
        '4. 第二项',
        '  - 有序中的嵌套无序'
      ].join('\n'),
      contentType: 'markdown'
    })

    const write = editor.view.dom
    expect(countsFrom(write)).toEqual({
      ul: 3,
      ol: 1,
      li: 7,
      nestedUl: 2,
      nestedOl: 0
    })
  })

  it('keeps ul, ol, li, and start complete in the client preview HTML', () => {
    editor = new Editor({
      extensions: createSFEditorExtensions({
        placeholder: 'body',
        maxCharacters: 1000
      }),
      content: [
        '- 项目符号一',
        '- 项目符号二',
        '  - 嵌套无序',
        '',
        '3. 从三开始的有序列表',
        '4. 第二项',
        '  1. 嵌套有序'
      ].join('\n'),
      contentType: 'markdown'
    })

    const preview = document.createElement('div')
    preview.innerHTML = editor.getHTML()

    expect(countsFrom(preview)).toEqual({
      ul: 2,
      ol: 2,
      li: 6,
      nestedUl: 1,
      nestedOl: 1
    })
    expect(preview.innerHTML).toContain('<ol start="3">')
  })

  it('allows list structure and ordered starts through the client sanitizer contract', () => {
    expect(SF_PURIFY_CONFIG.ALLOWED_TAGS).toEqual(expect.arrayContaining(['ul', 'ol', 'li']))
    expect(SF_PURIFY_CONFIG.ALLOWED_ATTR).toContain('start')
  })

  it('keeps orderedList start and nested lists complete from native JSON', () => {
    editor = new Editor({
      extensions: createSFEditorExtensions({
        placeholder: 'body',
        maxCharacters: 1000
      }),
      content: {
        type: 'doc',
        content: [
          {
            type: 'orderedList',
            attrs: { start: 5 },
            content: [
              {
                type: 'listItem',
                content: [
                  { type: 'paragraph', content: [{ type: 'text', text: '第五项' }] },
                  {
                    type: 'bulletList',
                    content: [
                      {
                        type: 'listItem',
                        content: [
                          { type: 'paragraph', content: [{ type: 'text', text: '嵌套项目' }] }
                        ]
                      }
                    ]
                  }
                ]
              },
              {
                type: 'listItem',
                content: [
                  { type: 'paragraph', content: [{ type: 'text', text: '第六项' }] }
                ]
              }
            ]
          }
        ]
      }
    })

    const html = editor.getHTML()
    expect(html).toContain('<ol start="5">')
    expect(html).toContain('<ul>')
    expect(html).toContain('<li>')

    const preview = document.createElement('div')
    preview.innerHTML = html
    expect(countsFrom(preview)).toEqual({
      ul: 1,
      ol: 1,
      li: 3,
      nestedUl: 1,
      nestedOl: 0
    })
    expect(preview.innerHTML).toContain('<ol start="5">')
    // Markdown 导出仍保留列表结构与文本。
    expect(editor.getMarkdown()).toContain('- 嵌套项目')
    expect(editor.getMarkdown()).toContain('5. 第五项')
  })
})
