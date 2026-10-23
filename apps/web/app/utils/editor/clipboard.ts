import { Extension } from '@tiptap/core'
import { NodeSelection, Plugin, PluginKey } from '@tiptap/pm/state'
import type { Selection } from '@tiptap/pm/state'
import type { EditorView } from '@tiptap/pm/view'

/**
 * 剪贴板粘贴归一化。
 *
 * ProseMirror 只在「当前在代码块内且剪贴板带 text/plain」时保证整段文字作为
 * 一个字面文本节点插入；一旦剪贴板只有 text/html（部分网页/客户端会这样），
 * 代码块会被 HTML 结构拆成多个块。这里补上两条 Host 保证：
 *
 * 1. 光标在代码块内：多行粘贴始终按字面文本插入，绝不走 HTML 解析。
 * 2. 整段剪贴板只有代码（一个或多个 pre，或 pre 内每行一个 div）时：
 *    合并成一个 fenced code block，避免网页复制的逐行结构炸成多个块。
 */

const BLOCK_TAGS = new Set([
  'ADDRESS', 'ARTICLE', 'ASIDE', 'BLOCKQUOTE', 'DD', 'DIV', 'DL', 'DT',
  'FIGCAPTION', 'FIGURE', 'FOOTER', 'FORM', 'H1', 'H2', 'H3', 'H4', 'H5',
  'H6', 'HEADER', 'HR', 'LI', 'MAIN', 'NAV', 'OL', 'P', 'PRE', 'SECTION',
  'TABLE', 'TR', 'UL'
])

const CELL_TAGS = new Set(['TD', 'TH'])

// 剪贴板 HTML 常见的不可见包装；纯代码判定时直接忽略。
const IGNORED_WRAPPER_TAGS = new Set([
  'BASE', 'LINK', 'META', 'NOSCRIPT', 'SCRIPT', 'STYLE', 'TEMPLATE', 'TITLE'
])

const SKIPPED_CONTENT_TAGS = new Set(['NOSCRIPT', 'SCRIPT', 'STYLE', 'TEMPLATE'])

/** 统一剪贴板换行符；不裁剪空行、行尾空白或缩进。 */
export function normalizePastedText(text: string) {
  return text.replace(/\r\n?/g, '\n')
}

/**
 * 把 HTML 读成纯文本，供没有 text/plain 的剪贴板在代码块内使用。
 * 只读取 detached DOM，不插入页面，也不执行任何脚本。
 */
export function clipboardHtmlToText(html: string) {
  if (!html) {
    return ''
  }

  const output = { value: '' }
  appendChildrenText(parseClipboardHtml(html), output)

  return normalizePastedText(output.value)
    .replace(/^\n+/, '')
    .replace(/\n+$/, '')
}

/**
 * 当剪贴板 HTML 只包含代码容器时，合并成一个 `<pre><code>`。
 * 正文与代码混合的剪贴板原样返回，交给 ProseMirror 正常解析。
 */
export function collapseCodeOnlyHtml(html: string) {
  if (!html) {
    return html
  }

  try {
    const fragment = parseClipboardHtml(html)
    const codeBlocks: HTMLElement[] = []

    for (const child of Array.from(fragment.childNodes)) {
      if (child.nodeType === 3) {
        if ((child as Text).data.trim()) {
          return html
        }
        continue
      }
      if (child.nodeType !== 1) {
        continue
      }

      const element = child as HTMLElement
      const tag = element.tagName.toUpperCase()
      if (IGNORED_WRAPPER_TAGS.has(tag)) {
        continue
      }
      if (tag !== 'PRE') {
        return html
      }
      codeBlocks.push(element)
    }

    if (codeBlocks.length === 0) {
      return html
    }

    const text = codeBlocks
      .map(block => collectElementText(block))
      .map(part => part.replace(/^\n+/, '').replace(/\n+$/, ''))
      .filter(Boolean)
      .join('\n')

    if (!text.trim()) {
      return html
    }

    const language = codeBlocks
      .map(codeBlockLanguage)
      .find(Boolean) || ''
    const languageClass = language
      ? ` class="language-${escapeHtmlAttribute(language)}"`
      : ''

    return `<pre><code${languageClass}>${escapeHtmlText(text)}</code></pre>`
  } catch {
    return html
  }
}

export function createSForumClipboardExtensions() {
  return [
    Extension.create({
      name: 'sforumClipboard',
      addProseMirrorPlugins() {
        return [
          new Plugin({
            key: new PluginKey('sforumClipboard'),
            props: {
              transformPastedHTML: (html: string, view: EditorView) => {
                // basic-field 没有 codeBlock 节点，避免把行内字段内容折成代码块。
                return view.state.schema.nodes.codeBlock ? collapseCodeOnlyHtml(html) : html
              },
              handlePaste: (view, event) => {
                const clipboard = event.clipboardData
                if (!clipboard) {
                  return false
                }
                // 图片/文件粘贴保持现有上传流程。
                if (clipboard.files && clipboard.files.length > 0) {
                  return false
                }

                const selection = view.state.selection
                const codeBlock = selectedCodeBlock(selection)
                if (!codeBlock) {
                  return false
                }

                const text = normalizePastedText(clipboard.getData('text/plain') || '')
                  || clipboardHtmlToText(clipboard.getData('text/html') || '')
                if (!text) {
                  return false
                }

                event.preventDefault()
                if (codeBlock.kind === 'node') {
                  const node = codeBlock.selection.node
                  const next = node.type.create(node.attrs, view.state.schema.text(text))
                  view.dispatch(view.state.tr.replaceWith(
                    codeBlock.selection.from,
                    codeBlock.selection.from + node.nodeSize,
                    next
                  ))
                  return true
                }

                view.dispatch(view.state.tr.insertText(text))
                return true
              }
            }
          })
        ]
      }
    })
  ]
}

type SelectedCodeBlock =
  | { kind: 'cursor' }
  | { kind: 'node', selection: NodeSelection }

/**
 * 用 inert `<template>` 解析剪贴板 HTML：脚本不执行，图片也不进入渲染树。
 */
function parseClipboardHtml(html: string) {
  const template = document.createElement('template')
  template.innerHTML = html
  return template.content
}

function selectedCodeBlock(selection: Selection): SelectedCodeBlock | null {
  if (selection instanceof NodeSelection) {
    return selection.node.type.spec.code ? { kind: 'node', selection } : null
  }
  const { $from, $to } = selection
  if (!$from.sameParent($to) || !$from.parent.type.spec.code) {
    return null
  }
  return { kind: 'cursor' }
}

function appendNodeText(node: Node, output: { value: string }) {
  if (node.nodeType === 3) {
    output.value += (node as Text).data
    return
  }
  if (node.nodeType !== 1) {
    return
  }

  const element = node as HTMLElement
  const tag = element.tagName.toUpperCase()
  if (SKIPPED_CONTENT_TAGS.has(tag)) {
    return
  }
  if (tag === 'BR') {
    output.value += '\n'
    return
  }
  if (CELL_TAGS.has(tag)) {
    appendChildrenText(element, output)
    output.value += '\t'
    return
  }
  if (BLOCK_TAGS.has(tag)) {
    ensureTrailingNewline(output)
    appendChildrenText(element, output)
    ensureTrailingNewline(output)
    return
  }

  appendChildrenText(element, output)
}

function appendChildrenText(element: Node, output: { value: string }) {
  for (const child of Array.from(element.childNodes)) {
    appendNodeText(child, output)
  }
}

function ensureTrailingNewline(output: { value: string }) {
  if (output.value && !output.value.endsWith('\n')) {
    output.value += '\n'
  }
}

function collectElementText(element: HTMLElement) {
  const output = { value: '' }
  appendChildrenText(element, output)
  return normalizePastedText(output.value)
}

function codeBlockLanguage(element: HTMLElement) {
  const candidates = [
    element,
    element.querySelector('code'),
    element.firstElementChild
  ].filter((candidate): candidate is HTMLElement => Boolean(candidate))

  for (const candidate of candidates) {
    const dataLanguage = candidate.getAttribute('data-language')
      || candidate.getAttribute('data-lang')
    if (dataLanguage?.trim()) {
      return sanitizeLanguage(dataLanguage)
    }
    for (const className of Array.from(candidate.classList)) {
      const match = /^(?:language|lang)-(.+)$/.exec(className)
      if (match?.[1]) {
        return sanitizeLanguage(match[1])
      }
    }
  }

  return ''
}

function sanitizeLanguage(value: string) {
  return value.trim().replace(/[^A-Za-z0-9_+#.-]/g, '')
}

function escapeHtmlText(value: string) {
  return value
    .replaceAll('&', '&amp;')
    .replaceAll('<', '&lt;')
    .replaceAll('>', '&gt;')
}

function escapeHtmlAttribute(value: string) {
  return escapeHtmlText(value).replaceAll('"', '&quot;')
}
