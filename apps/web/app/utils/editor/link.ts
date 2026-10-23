import type { Editor } from '@tiptap/core'
import Link from '@tiptap/extension-link'

/**
 * Host 链接 mark。
 *
 * Tiptap v3 的 Link 默认把 `inclusive` 绑定到 `autolink`：开启自动链接后，
 * 光标停在链接末尾继续输入时 link mark 会一路续写下去（粘贴 URL 后输入空格和
 * 普通文字也会被并入链接）。编辑器需要「链接在边界处结束」的语义，因此固定
 * `inclusive: false`；自动链接本身仍由扩展的 appendTransaction 负责，不受影响。
 */
export const SForumLink = Link.extend({
  inclusive: () => false
})

/**
 * 空选区插入链接：把 URL 本身作为链接文字，光标停在链接之后。
 * 不使用 `setMark`，避免 stored mark 把后续输入全部吞进链接。
 */
export function insertSForumLink(editor: Editor, href: string) {
  return editor.chain().focus().insertContent({
    type: 'text',
    text: href,
    marks: [{ type: 'link', attrs: { href } }]
  }).run()
}

