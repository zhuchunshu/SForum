import { afterEach, describe, expect, it } from 'bun:test'
import { Editor } from '@tiptap/core'
import type { JSONContent } from '@tiptap/core'
import { createSFEditorExtensions } from '../../app/utils/sfEditor'
import { sanitizeHtml } from '../../app/utils/sfSanitize'
import {
  SF_EDITOR_TABLE_LIMITS,
  insertSForumTable,
  normalizeSFEditorTableSize,
  setSForumTableAlign
} from '../../app/utils/editor/table'

let editor: Editor | undefined

afterEach(() => {
  editor?.destroy()
  editor = undefined
})

const pipeTable = [
  '| 名称 | 数量 | 备注 |',
  '| :--- | ---: | :---: |',
  '| 苹果 | 3 | 红 |',
  '| 梨 | 12 | 绿 |'
].join('\n')

function createEditor(content: string | JSONContent, preset: 'full' | 'basic-field' = 'full') {
  return new Editor({
    extensions: createSFEditorExtensions({
      placeholder: 'body',
      maxCharacters: 4000,
      preset
    }),
    content,
    ...(typeof content === 'string' ? { contentType: 'markdown' as const } : {})
  })
}

function tableNode(root: JSONContent | undefined) {
  return (root?.content || []).find(node => node.type === 'table')
}

describe('editor table node contract', () => {
  it('keeps a GFM pipe table in the document instead of dropping it', () => {
    editor = createEditor(pipeTable)

    const table = tableNode(editor.getJSON())
    expect(table).toBeDefined()
    expect(table?.content?.map(row => row.type)).toEqual(['tableRow', 'tableRow', 'tableRow'])

    const header = table?.content?.[0]?.content || []
    expect(header.map(cell => cell.type)).toEqual(['tableHeader', 'tableHeader', 'tableHeader'])
    expect(header.map(cell => cell.attrs?.align)).toEqual(['left', 'right', 'center'])

    const body = table?.content?.[1]?.content || []
    expect(body.map(cell => cell.type)).toEqual(['tableCell', 'tableCell', 'tableCell'])
    expect(body[0]?.content?.[0]?.content?.[0]?.text).toBe('苹果')
  })

  it('round-trips markdown through JSON with alignment markers intact', () => {
    editor = createEditor(pipeTable)

    const markdown = editor.getMarkdown()
    expect(markdown).toContain(':---')
    expect(markdown).toContain('---:')
    expect(markdown).toContain(':---:')

    const reopened = createEditor(markdown)
    const reopenedMarkdown = reopened.getMarkdown()
    expect(reopenedMarkdown).toBe(markdown)
    expect(tableNode(reopened.getJSON())?.content?.length).toBe(3)
    reopened.destroy()
  })

  it('renders alignment as an align attribute and never as inline style', () => {
    editor = createEditor(pipeTable)

    const html = editor.getHTML()
    expect(html).toContain('<table>')
    expect(html).toContain('align="left"')
    expect(html).toContain('align="center"')
    expect(html).toContain('align="right"')
    expect(html).not.toContain('style=')
    expect(html).not.toContain('<colgroup')
  })

  it('parses legacy inline text-align into align and re-renders it as an attribute', () => {
    editor = new Editor({
      extensions: createSFEditorExtensions({ placeholder: 'body', maxCharacters: 4000 }),
      content: '<table><tbody><tr><td style="text-align: center">居中</td><td align="right">右</td></tr></tbody></table>'
    })

    const cells = tableNode(editor.getJSON())?.content?.[0]?.content || []
    expect(cells.map(cell => cell.attrs?.align)).toEqual(['center', 'right'])

    const html = editor.getHTML()
    expect(html).toContain('align="center"')
    expect(html).toContain('align="right"')
    expect(html).not.toContain('style=')
  })

  it('keeps table structure and alignment through the client sanitizer', () => {
    // 预览区拿到的就是编辑器 HTML。happy-dom 的 DOMParser shim 会把「字符串以
    // 裸 <table> 开头」的片段解析丢外层 table（真实浏览器无此问题），所以这里
    // 用带前导段落的真实文档形状验证净化后的结构与对齐。
    editor = createEditor(`前文\n\n${pipeTable}`)

    const sanitized = sanitizeHtml(editor.getHTML())
    const container = document.createElement('div')
    container.innerHTML = sanitized

    expect(container.querySelectorAll('table')).toHaveLength(1)
    expect(container.querySelectorAll('tr')).toHaveLength(3)
    expect(container.querySelectorAll('th')).toHaveLength(3)
    expect(container.querySelectorAll('td')).toHaveLength(6)
    expect(container.querySelector('th[align="left"]')).not.toBeNull()
    expect(container.querySelector('td[align="center"]')).not.toBeNull()
    expect(container.innerHTML).not.toContain('style=')
  })

  it('appends a trailing paragraph when the table lands at the end of the document', () => {
    editor = createEditor('前文')
    editor.commands.setTextSelection(editor.state.doc.content.size)

    expect(insertSForumTable(editor, { rows: 2, cols: 2 })).toBe(true)

    const doc = editor.getJSON()
    expect(doc.content?.map(node => node.type)).toEqual(['paragraph', 'table', 'paragraph'])

    const table = tableNode(doc)
    expect(table?.content).toHaveLength(2)
    expect(table?.content?.[0]?.content).toHaveLength(2)
    expect(table?.content?.[0]?.content?.[0]?.type).toBe('tableHeader')
    expect(table?.content?.[1]?.content?.[0]?.type).toBe('tableCell')
    // 补段落不能把光标踢出表格，否则用户无法继续在单元格里输入。
    expect(editor.isActive('table')).toBe(true)
  })

  it('does not append an extra paragraph when text already follows the inserted table', () => {
    editor = createEditor('前文')
    editor.commands.setTextSelection(1)

    expect(insertSForumTable(editor)).toBe(true)

    const doc = editor.getJSON()
    expect(doc.content?.filter(node => node.type === 'paragraph')).toHaveLength(1)
    expect(doc.content?.[0]?.type).toBe('table')
  })

  it('refuses a nested table while the cursor is inside a table', () => {
    editor = createEditor('前文')
    editor.commands.setTextSelection(editor.state.doc.content.size)
    expect(insertSForumTable(editor, { rows: 2, cols: 2 })).toBe(true)

    expect(insertSForumTable(editor, { rows: 2, cols: 2 })).toBe(false)
    expect(editor.getJSON().content?.filter(node => node.type === 'table')).toHaveLength(1)
  })

  it('clamps requested table sizes to the supported limits', () => {
    expect(normalizeSFEditorTableSize()).toEqual({
      rows: SF_EDITOR_TABLE_LIMITS.defaultRows,
      cols: SF_EDITOR_TABLE_LIMITS.defaultCols
    })
    expect(normalizeSFEditorTableSize({ rows: 99, cols: 0 })).toEqual({ rows: 8, cols: 1 })
  })

  it('sets cell alignment through the table command helper', () => {
    editor = createEditor('前文')
    editor.commands.setTextSelection(editor.state.doc.content.size)
    insertSForumTable(editor, { rows: 1, cols: 2 })

    expect(setSForumTableAlign(editor, 'center')).toBe(true)

    const header = tableNode(editor.getJSON())?.content?.[0]?.content || []
    expect(header[0]?.attrs?.align).toBe('center')
    expect(editor.getHTML()).toContain('align="center"')
  })

  it('does not register table nodes in the basic-field schema', () => {
    editor = createEditor('| 名称 | 数量 |\n| --- | --- |\n| 苹果 | 3 |', 'basic-field')

    expect(editor.schema.nodes.table).toBeUndefined()
    expect(editor.schema.nodes.tableCell).toBeUndefined()
    expect(tableNode(editor.getJSON())).toBeUndefined()
  })

  it('round-trips empty cells and single-column tables without breaking', () => {
    editor = new Editor({
      extensions: createSFEditorExtensions({ placeholder: 'body', maxCharacters: 4000 }),
      content: {
        type: 'doc',
        content: [
          {
            type: 'table',
            content: [
              {
                type: 'tableRow',
                content: [{ type: 'tableHeader', content: [{ type: 'paragraph' }] }]
              },
              {
                type: 'tableRow',
                content: [{ type: 'tableCell', content: [{ type: 'paragraph' }] }]
              }
            ]
          }
        ]
      }
    })

    const markdown = editor.getMarkdown()
    const reopened = createEditor(markdown)
    const table = tableNode(reopened.getJSON())

    expect(table?.content).toHaveLength(2)
    expect(table?.content?.[0]?.content).toHaveLength(1)
    expect(reopened.getMarkdown()).toBe(markdown)
    reopened.destroy()
  })
})
