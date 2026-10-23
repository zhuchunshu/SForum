import { Editor } from '@tiptap/core'
import { TextSelection } from '@tiptap/pm/state'
import { createSFEditorExtensions } from '../../app/utils/sfEditor'

type TestEditorOptions = {
  content?: string
  preset?: 'full' | 'basic-field'
}

type ClipboardPayload = {
  text?: string
  html?: string
  files?: File[]
}

export function createTestEditor(contentOrOptions: string | TestEditorOptions = {}) {
  const options = typeof contentOrOptions === 'string'
    ? { content: contentOrOptions }
    : contentOrOptions
  return new Editor({
    content: options.content || '',
    contentType: 'markdown',
    extensions: createSFEditorExtensions({
      placeholder: 'p',
      maxCharacters: 100000,
      preset: options.preset || 'full',
      resourceKind: 'topic'
    }) as never[]
  })
}

/**
 * 构造与浏览器粘贴事件同形的 clipboardData。happy-dom 的 ClipboardEvent
 * 不接受 clipboardData 入参，这里用 defineProperty 注入，走 ProseMirror 的
 * 真实 paste handler。
 */
export function pasteClipboard(editor: Editor, payload: ClipboardPayload) {
  const event = new Event('paste', { bubbles: true, cancelable: true })
  const getData = (type: string) => {
    if (type === 'text/plain' || type === 'Text') {
      return payload.text || ''
    }
    if (type === 'text/html') {
      return payload.html || ''
    }
    return ''
  }
  Object.defineProperty(event, 'clipboardData', {
    value: {
      files: payload.files || [],
      types: [
        payload.text !== undefined ? 'text/plain' : '',
        payload.html !== undefined ? 'text/html' : ''
      ].filter(Boolean),
      getData
    }
  })
  editor.view.dom.dispatchEvent(event)
  return event
}

export function typeText(editor: Editor, text: string) {
  for (const character of text) {
    const { from } = editor.state.selection
    editor.view.dispatch(editor.state.tr.insertText(character, from))
  }
}

export function setSelection(editor: Editor, from: number, to = from) {
  editor.view.dispatch(editor.state.tr.setSelection(TextSelection.create(editor.state.doc, from, to)))
}

export function selectFirstCodeBlockEnd(editor: Editor) {
  const codeBlock = editor.state.doc.firstChild
  if (!codeBlock || codeBlock.type.name !== 'codeBlock') {
    throw new Error('expected the document to start with a codeBlock')
  }
  setSelection(editor, 1 + codeBlock.textContent.length)
}

export function collectBlocks(editor: Editor) {
  const blocks: Array<{ type: string, text: string, language: string | null }> = []
  editor.state.doc.forEach(node => {
    blocks.push({
      type: node.type.name,
      text: node.textContent,
      language: (node.attrs.language as string | null | undefined) ?? null
    })
  })
  return blocks
}

export function collectTextMarks(editor: Editor) {
  const spans: Array<{ text: string, marks: string[] }> = []
  editor.state.doc.descendants(node => {
    if (node.isText) {
      spans.push({
        text: node.text || '',
        marks: node.marks.map(mark => mark.type.name)
      })
    }
    return true
  })
  return spans
}
