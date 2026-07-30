import { Extension, type Editor } from '@tiptap/core'
import { Plugin, PluginKey } from '@tiptap/pm/state'
import { Decoration, DecorationSet } from '@tiptap/pm/view'

type PlaceholderAdd = {
  id: string
  pos: number
  label: string
  fileCount: number
}

type PlaceholderAction = {
  add?: PlaceholderAdd
  remove?: { id: string }
}

export const editorImageUploadPlaceholderKey = new PluginKey<DecorationSet>('sforumImageUploadPlaceholder')

function placeholderElement(label: string, fileCount: number) {
  const element = document.createElement('span')
  element.className = 'sf-editor-image-upload'
  element.contentEditable = 'false'
  element.setAttribute('role', 'status')
  element.setAttribute('aria-label', label)

  const spinner = document.createElement('span')
  spinner.className = 'sf-editor-image-upload__spinner'
  spinner.setAttribute('aria-hidden', 'true')
  element.append(spinner)

  const text = document.createElement('span')
  text.textContent = fileCount > 1 ? `${label} (${fileCount})` : label
  element.append(text)
  return element
}

export function createEditorImageUploadPlaceholderExtension() {
  return Extension.create({
    name: 'sforumImageUploadPlaceholder',

    addProseMirrorPlugins() {
      return [new Plugin<DecorationSet>({
        key: editorImageUploadPlaceholderKey,
        state: {
          init: () => DecorationSet.empty,
          apply(transaction, previous) {
            let next = previous.map(transaction.mapping, transaction.doc)
            const action = transaction.getMeta(editorImageUploadPlaceholderKey) as PlaceholderAction | undefined

            if (action?.add) {
              const pos = Math.max(0, Math.min(action.add.pos, transaction.doc.content.size))
              next = next.add(transaction.doc, [Decoration.widget(
                pos,
                () => placeholderElement(action.add!.label, action.add!.fileCount),
                { id: action.add.id, key: action.add.id, side: -1 }
              )])
            }
            if (action?.remove) {
              next = next.remove(next.find(undefined, undefined, spec => spec.id === action.remove!.id))
            }
            return next
          }
        },
        props: {
          decorations(state) {
            return editorImageUploadPlaceholderKey.getState(state)
          }
        }
      })]
    }
  })
}

export function addEditorImageUploadPlaceholder(
  editor: Editor,
  placeholder: PlaceholderAdd
) {
  editor.view.dispatch(editor.state.tr.setMeta(editorImageUploadPlaceholderKey, { add: placeholder }))
}

export function findEditorImageUploadPlaceholder(editor: Editor, id: string) {
  return editorImageUploadPlaceholderKey
    .getState(editor.state)
    ?.find(undefined, undefined, spec => spec.id === id)[0]
    ?.from
}

export function removeEditorImageUploadPlaceholder(editor: Editor, id: string) {
  editor.view.dispatch(editor.state.tr.setMeta(editorImageUploadPlaceholderKey, { remove: { id } }))
}

