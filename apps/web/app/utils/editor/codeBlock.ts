import type { CommandProps } from '@tiptap/core'
import { CodeBlock } from '@tiptap/extension-code-block'
import type { Node as PMNode } from '@tiptap/pm/model'
import { TextSelection } from '@tiptap/pm/state'

/**
 * Host 代码块。
 *
 * Tiptap 原生的 `toggleCodeBlock` 走 `toggleNode`，会对选区里的每个 textblock
 * 各执行一次。用户先粘贴多段文字再点「代码块」时，就会得到每段一个代码块。
 * 这里在顶层 doc 范围内把连续的 textblock 合并成一个代码块；单段、取消代码块、
 * 列表/引用/表格内的嵌套选区仍走原生行为。
 */

export type SForumCodeBlockAttributes = {
  language?: string | null
}

export const SForumCodeBlock = CodeBlock.extend({
  addCommands() {
    return {
      ...this.parent?.(),
      toggleCodeBlock: (attributes?: SForumCodeBlockAttributes) => (props: CommandProps) => {
        if (mergeSelectedTextblocks(props, attributes)) {
          return true
        }
        return props.commands.toggleNode(this.name, 'paragraph', attributes)
      }
    }
  }
})

function mergeSelectedTextblocks(
  { state, tr }: CommandProps,
  attributes?: SForumCodeBlockAttributes
) {
  // 链式命令里前面的步骤可能已经改过文档，位置不再可信，交回原生逻辑。
  if (tr.docChanged) {
    return false
  }

  const range = state.selection.$from.blockRange(state.selection.$to)
  if (!range || range.depth !== 0) {
    return false
  }

  const count = range.endIndex - range.startIndex
  if (count <= 1) {
    return false
  }

  const blocks: PMNode[] = []
  for (let index = range.startIndex; index < range.endIndex; index += 1) {
    blocks.push(range.parent.child(index))
  }

  // 只合并纯文本块。全部已经是代码块时保留原生「取消代码块」语义。
  if (!blocks.every(block => block.isTextblock)) {
    return false
  }
  if (blocks.every(block => block.type.name === 'codeBlock')) {
    return false
  }

  const existingCodeBlock = blocks.find(block => block.type.name === 'codeBlock')
  const language = attributes?.language !== undefined
    ? attributes.language
    : existingCodeBlock?.attrs.language ?? null
  const text = blocks.map(block => block.textContent).join('\n')
  const codeBlock = state.schema.nodes.codeBlock?.create(
    { language },
    text ? state.schema.text(text) : undefined
  )
  if (!codeBlock) {
    return false
  }

  tr.replaceWith(range.start, range.end, codeBlock)
  tr.setSelection(TextSelection.near(tr.doc.resolve(range.start + codeBlock.nodeSize - 1)))
  return true
}
