import { afterEach, describe, expect, it } from 'bun:test'
import type { Editor } from '@tiptap/core'
import { NodeSelection } from '@tiptap/pm/state'
import {
  collectBlocks,
  createTestEditor,
  pasteClipboard,
  setSelection
} from '../helpers/editorClipboard'
import { clipboardHtmlToText, collapseCodeOnlyHtml } from '../../app/utils/editor/clipboard'

let editor: Editor | undefined

afterEach(() => {
  editor?.destroy()
  editor = undefined
})

function createEmptyCodeBlock(language = 'shell') {
  const instance = createTestEditor()
  instance.chain().focus().toggleCodeBlock({ language }).run()
  return instance
}

describe('editor code block paste', () => {
  it('纯文本多行粘贴进代码块保持单块与空行', () => {
    editor = createEmptyCodeBlock()
    pasteClipboard(editor, { text: 'line1\n\nline2\n    indented\nline4\n' })

    expect(collectBlocks(editor)).toEqual([
      { type: 'codeBlock', text: 'line1\n\nline2\n    indented\nline4\n', language: 'shell' },
      { type: 'paragraph', text: '', language: null }
    ])
  })

  it('text/plain + text/html 同时存在时仍按纯文本插入', () => {
    editor = createEmptyCodeBlock()
    pasteClipboard(editor, {
      text: 'line1\nline2',
      html: '<pre><code>line1</code></pre><pre><code>line2</code></pre>'
    })

    expect(collectBlocks(editor)[0]).toEqual({
      type: 'codeBlock',
      text: 'line1\nline2',
      language: 'shell'
    })
  })

  it('只有 text/html 时也保持单个代码块', () => {
    editor = createEmptyCodeBlock()
    pasteClipboard(editor, {
      html: '<pre><code>line1<br>line2<br><br>line3</code></pre>'
    })

    expect(collectBlocks(editor)[0]).toEqual({
      type: 'codeBlock',
      text: 'line1\nline2\n\nline3',
      language: 'shell'
    })
  })

  it('pre 内每行一个 div 时按行还原', () => {
    editor = createEmptyCodeBlock()
    pasteClipboard(editor, {
      html: '<pre><code><div>line1</div><div>line2</div><div>line3</div></code></pre>'
    })

    expect(collectBlocks(editor)[0]).toEqual({
      type: 'codeBlock',
      text: 'line1\nline2\nline3',
      language: 'shell'
    })
  })

  it('代码块整体被 NodeSelection 选中时替换为单个代码块', () => {
    editor = createTestEditor('```shell\na\nb\n```')
    editor.view.dispatch(editor.state.tr.setSelection(NodeSelection.create(editor.state.doc, 0)))
    pasteClipboard(editor, {
      html: '<pre><code>line1<br>line2</code></pre>'
    })

    expect(collectBlocks(editor)[0]).toEqual({
      type: 'codeBlock',
      text: 'line1\nline2',
      language: 'shell'
    })
  })

  it('每行一个 pre 的纯代码剪贴板合并成一个带语言的代码块', () => {
    editor = createTestEditor()
    pasteClipboard(editor, {
      html: '<pre><code class="language-shell">line1</code></pre><pre><code>line2</code></pre><pre><code>line3</code></pre>'
    })

    expect(collectBlocks(editor)).toEqual([
      { type: 'codeBlock', text: 'line1\nline2\nline3', language: 'shell' },
      { type: 'paragraph', text: '', language: null }
    ])
  })

  it('正文与代码混合的剪贴板不合并正文', () => {
    editor = createTestEditor()
    pasteClipboard(editor, {
      text: 'hello\n\ncode',
      html: '<p>hello</p><pre><code>code</code></pre>'
    })

    expect(collectBlocks(editor)).toEqual([
      { type: 'paragraph', text: 'hello', language: null },
      { type: 'codeBlock', text: 'code', language: null },
      { type: 'paragraph', text: '', language: null }
    ])
  })

  it('多段选区点击代码块只产生一个代码块', () => {
    editor = createTestEditor('line1\n\nline2\n\nline3')
    setSelection(editor, 1, editor.state.doc.content.size - 1)
    editor.chain().focus().toggleCodeBlock().run()

    const blocks = collectBlocks(editor)
    expect(blocks.filter(block => block.type === 'codeBlock')).toHaveLength(1)
    expect(blocks[0]).toEqual({
      type: 'codeBlock',
      text: 'line1\nline2\nline3',
      language: null
    })
  })

  it('单段选区仍保持原有代码块切换行为', () => {
    editor = createTestEditor('line1\n\nline2')
    setSelection(editor, 1, 6)
    editor.chain().focus().toggleCodeBlock().run()

    expect(collectBlocks(editor).filter(block => block.type === 'codeBlock')).toHaveLength(1)
    expect(collectBlocks(editor)[0]).toEqual({
      type: 'codeBlock',
      text: 'line1',
      language: null
    })
  })

  it('已有代码块仍然可以取消', () => {
    editor = createTestEditor('```shell\na\nb\n```')
    setSelection(editor, 1, 4)
    editor.chain().focus().toggleCodeBlock().run()

    expect(collectBlocks(editor).some(block => block.type === 'codeBlock')).toBe(false)
  })

  it('粘贴后的 Markdown 往返仍然是一个代码块', () => {
    editor = createEmptyCodeBlock()
    pasteClipboard(editor, { text: 'line1\n\nline2' })
    const markdown = editor.getMarkdown()

    const reloaded = createTestEditor(markdown)
    const codeBlocks = collectBlocks(reloaded).filter(block => block.type === 'codeBlock')
    expect(codeBlocks).toHaveLength(1)
    expect(codeBlocks[0]?.text).toBe('line1\n\nline2')
    expect(codeBlocks[0]?.language).toBe('shell')
    reloaded.destroy()
  })

  it('basic-field 不注册代码块节点', () => {
    editor = createTestEditor({ preset: 'basic-field' })
    expect(editor.extensionManager.extensions.some(extension => extension.name === 'codeBlock')).toBe(false)
  })
})

describe('clipboard code normalization helpers', () => {
  it('忽略 meta 包装并保留语言', () => {
    const collapsed = collapseCodeOnlyHtml(
      '<meta charset="utf-8"><pre><code class="language-shell">line1\nline2</code></pre>'
    )
    expect(collapsed).toContain('<pre><code class="language-shell">')
    expect(collapsed).toContain('line1\nline2')
  })

  it('转义代码文本并清洗语言属性', () => {
    const collapsed = collapseCodeOnlyHtml(
      '<pre class="language-sh&quot; onclick=alert(1)"><code>&lt;img src=x onerror=alert(1)&gt;</code></pre>'
    )
    expect(collapsed).toContain('class="language-sh"')
    expect(collapsed).not.toContain('onclick')
    expect(collapsed).toContain('&lt;img src=x onerror=alert(1)&gt;')
  })

  it('把 pre 内的逐行 div 还原为换行文本', () => {
    expect(clipboardHtmlToText('<pre><code><div>line1</div><div>line2</div></code></pre>'))
      .toBe('line1\nline2')
  })

  it('正文与代码混合时原样返回', () => {
    const html = '<p>hello</p><pre><code>code</code></pre>'
    expect(collapseCodeOnlyHtml(html)).toBe(html)
  })
})
