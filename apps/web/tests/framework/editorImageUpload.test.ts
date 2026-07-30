import { afterEach, beforeEach, describe, expect, it } from 'bun:test'
import { Editor } from '@tiptap/core'
import StarterKit from '@tiptap/starter-kit'
import { Window } from 'happy-dom'
import {
  addEditorImageUploadPlaceholder,
  createEditorImageUploadPlaceholderExtension,
  findEditorImageUploadPlaceholder,
  removeEditorImageUploadPlaceholder
} from '../../app/utils/editor/editorImageUpload'
import {
  collectEditorAttachmentIds,
  createSFEditorExtensions
} from '../../app/utils/sfEditor'
import { imageFilesFromList } from '../../app/composables/editor/useEditorImageUpload'

let editor: Editor | undefined

beforeEach(() => {
  const window = new Window({ url: 'http://localhost/' })
  Object.assign(globalThis, {
    window,
    document: window.document,
    navigator: window.navigator,
    Node: window.Node,
    HTMLElement: window.HTMLElement,
    ShadowRoot: window.ShadowRoot,
    MutationObserver: window.MutationObserver,
    requestAnimationFrame: window.requestAnimationFrame.bind(window),
    cancelAnimationFrame: window.cancelAnimationFrame.bind(window),
    getComputedStyle: window.getComputedStyle.bind(window)
  })
})

afterEach(() => {
  editor?.destroy()
  editor = undefined
})

describe('editor image upload position', () => {
  it('maps an upload placeholder through edits before the original target', () => {
    editor = new Editor({
      content: '<p>AB</p>',
      extensions: [StarterKit, createEditorImageUploadPlaceholderExtension()]
    })
    addEditorImageUploadPlaceholder(editor, { id: 'batch', pos: 2, label: 'uploading', fileCount: 1 })

    editor.view.dispatch(editor.state.tr.insertText('X', 1))

    expect(findEditorImageUploadPlaceholder(editor, 'batch')).toBe(3)
  })

  it('keeps side -1 placeholders before text typed at the same position', () => {
    editor = new Editor({
      content: '<p>AB</p>',
      extensions: [StarterKit, createEditorImageUploadPlaceholderExtension()]
    })
    addEditorImageUploadPlaceholder(editor, { id: 'batch', pos: 2, label: 'uploading', fileCount: 2 })

    editor.view.dispatch(editor.state.tr.insertText('X', 2))

    expect(findEditorImageUploadPlaceholder(editor, 'batch')).toBe(2)
    removeEditorImageUploadPlaceholder(editor, 'batch')
    expect(findEditorImageUploadPlaceholder(editor, 'batch')).toBeUndefined()
  })

  it('inserts a completed batch at the mapped target in original file order', () => {
    editor = new Editor({
      content: '<p>before after</p>',
      extensions: createSFEditorExtensions({ placeholder: 'body', maxCharacters: 1000 })
    })
    addEditorImageUploadPlaceholder(editor, { id: 'batch', pos: 8, label: 'uploading', fileCount: 2 })
    editor.view.dispatch(editor.state.tr.insertText('X', 1))

    const pos = findEditorImageUploadPlaceholder(editor, 'batch')
    expect(pos).toBe(9)
    expect(editor.commands.insertContentAt(pos!, [
      { type: 'image', attrs: { src: '/first.png', alt: 'first', attachmentId: 1, attachmentPublicId: 'a'.repeat(32) } },
      { type: 'image', attrs: { src: '/second.png', alt: 'second', attachmentId: 2, attachmentPublicId: 'b'.repeat(32) } }
    ], { updateSelection: false })).toBe(true)
    removeEditorImageUploadPlaceholder(editor, 'batch')

    const content = editor.getJSON().content || []
    expect(content.map(node => node.type)).toEqual(['paragraph', 'image', 'image', 'paragraph'])
    expect(content[0]?.content?.[0]?.text).toBe('Xbefore ')
    expect(content[1]?.attrs?.attachmentId).toBe(1)
    expect(content[2]?.attrs?.attachmentId).toBe(2)
    expect(content[3]?.content?.[0]?.text).toBe('after')
  })
})

describe('editor image upload file selection', () => {
  it('keeps clipboard raster images while rejecting SVG and non-image files', () => {
    const png = { name: 'pasted.png', type: 'image/png' } as File
    const svg = { name: 'unsafe.svg', type: 'image/svg+xml' } as File
    const text = { name: 'notes.txt', type: 'text/plain' } as File
    const files = { 0: png, 1: svg, 2: text, length: 3 } as unknown as FileList

    expect(imageFilesFromList(files)).toEqual([png])
  })
})

describe('editor image attachment identity', () => {
  it('collects unique attachment ids from nested native JSON', () => {
    expect(collectEditorAttachmentIds({
      type: 'doc',
      content: [{
        type: 'paragraph',
        content: [
          { type: 'image', attrs: { attachmentId: 42 } },
          { type: 'image', attrs: { attachmentId: 42 } },
          { type: 'image', attrs: { attachmentId: 7 } },
          { type: 'image', attrs: { src: 'https://example.test/external.png' } }
        ]
      }]
    })).toEqual([42, 7])
  })

  it('keeps attachment identity in native JSON but omits it from HTML', () => {
    editor = new Editor({
      extensions: createSFEditorExtensions({
        placeholder: 'body',
        maxCharacters: 1000
      }),
      content: {
        type: 'doc',
        content: [{
          type: 'image',
          attrs: {
            src: '/media/attachments/0123456789abcdef0123456789abcdef',
            alt: 'example',
            attachmentId: 42,
            attachmentPublicId: '0123456789abcdef0123456789abcdef'
          }
        }]
      }
    })

    expect(editor.getJSON().content?.[0]?.attrs?.attachmentId).toBe(42)
    expect(editor.getHTML()).not.toContain('attachmentId')
    expect(editor.getHTML()).not.toContain('attachmentPublicId')
  })
})
