import type { Editor } from '@tiptap/core'
import { mergeAttributes } from '@tiptap/core'
import {
  Table,
  TableCell,
  TableHeader,
  TableRow
} from '@tiptap/extension-table'

/**
 * 表格节点契约（与 Go 端 EditorDocument 渲染保持一致）：
 * table / tableRow / tableHeader / tableCell；单元格属性只有 colspan、rowspan、align。
 * 对齐必须输出 `align` 属性而不是内联 style —— 前端 DOMPurify 与服务端 bluemonday
 * 都会剥掉 style，只有属性才能在两端净化后存活。
 * 不做单元格合并/拆分，也不持久化列宽（resizable: false）。
 */
export type SFTableCellAlign = 'left' | 'center' | 'right'

export const SF_EDITOR_TABLE_LIMITS = {
  maxRows: 8,
  maxCols: 8,
  defaultRows: 3,
  defaultCols: 3,
  /** 工具栏 picker 的可选网格上限（比节点上限保守，避免误建大表）。 */
  pickerRows: 6,
  pickerCols: 6
} as const

export function normalizeSFTableCellAlign(value: unknown): SFTableCellAlign | null {
  return value === 'left' || value === 'center' || value === 'right' ? value : null
}

/**
 * 读取元素上的对齐：先看历史内联 style（旧数据 / 其它编辑器产物），再看 align 属性。
 * 写回时只输出 align 属性。
 */
function parseSFTableCellAlign(element: HTMLElement): SFTableCellAlign | null {
  const styleAlign = (element.style?.textAlign || '').trim().toLowerCase()
  const attrAlign = (element.getAttribute('align') || '').trim().toLowerCase()
  return normalizeSFTableCellAlign(styleAlign || attrAlign)
}

function createSFTableCellAlignAttribute() {
  return {
    default: null,
    parseHTML: (element: HTMLElement) => parseSFTableCellAlign(element),
    renderHTML: (attributes: Record<string, unknown>) => {
      const align = normalizeSFTableCellAlign(attributes.align)
      return align ? { align } : {}
    }
  }
}

// 覆盖父类属性：去掉 colwidth（不做列宽持久化），对齐渲染成 align 属性。
export const SForumTableCell = TableCell.extend({
  addAttributes() {
    return {
      colspan: { default: 1 },
      rowspan: { default: 1 },
      align: createSFTableCellAlignAttribute()
    }
  }
})

export const SForumTableHeader = TableHeader.extend({
  addAttributes() {
    return {
      colspan: { default: 1 },
      rowspan: { default: 1 },
      align: createSFTableCellAlignAttribute()
    }
  }
})

export const SForumTableRow = TableRow

// 父类 renderHTML 会写入 colgroup 与内联 style（min-width），与 align 属性契约冲突，
// 也会被两端净化器剥掉。这里只输出裸 table + tbody。
export const SForumTable = Table.extend({
  renderHTML({ HTMLAttributes }) {
    return [
      'table',
      mergeAttributes(this.options.HTMLAttributes, HTMLAttributes),
      ['tbody', 0]
    ]
  }
})

export function createSForumTableExtensions() {
  return [
    SForumTable.configure({
      resizable: false,
      allowTableNodeSelection: false
    }),
    SForumTableRow,
    SForumTableHeader,
    SForumTableCell
  ]
}

export type SFEditorTableSize = {
  rows: number
  cols: number
}

function clampTableDimension(value: unknown, fallback: number, max: number) {
  const parsed = Number(value)
  if (!Number.isFinite(parsed)) return fallback
  return Math.min(max, Math.max(1, Math.trunc(parsed)))
}

export function normalizeSFEditorTableSize(size?: Partial<SFEditorTableSize>): SFEditorTableSize {
  return {
    rows: clampTableDimension(size?.rows, SF_EDITOR_TABLE_LIMITS.defaultRows, SF_EDITOR_TABLE_LIMITS.maxRows),
    cols: clampTableDimension(size?.cols, SF_EDITOR_TABLE_LIMITS.defaultCols, SF_EDITOR_TABLE_LIMITS.maxCols)
  }
}

/**
 * 表格后面必须还有可输入的块。ProseMirror 不允许把光标放到文档末尾的 table 之后，
 * 因此在表格成为最后一个节点时补一个空段落（不移动当前选区，光标仍留在表格内）。
 */
function ensureTrailingParagraph(editor: Editor) {
  const { doc, schema } = editor.state
  const last = doc.lastChild

  if (!last || last.type.name !== 'table') return false

  const paragraph = schema.nodes.paragraph?.createAndFill()
  if (!paragraph) return false

  const inserted = editor
    .chain()
    .insertContentAt(doc.content.size, paragraph, { updateSelection: false })
    .run()

  return inserted
}

export function insertSForumTable(editor: Editor, size?: Partial<SFEditorTableSize>) {
  // 不支持嵌套表格：光标已在表格内时直接忽略。
  if (editor.isActive('table')) return false

  const { rows, cols } = normalizeSFEditorTableSize(size)
  const inserted = editor
    .chain()
    .focus()
    .insertTable({ rows, cols, withHeaderRow: true })
    .run()

  if (!inserted) return false

  ensureTrailingParagraph(editor)
  return true
}

/** 当前选区（或单元格）的对齐；无对齐返回 null。 */
export function getSForumTableAlign(editor: Editor): SFTableCellAlign | null {
  if (!editor.isActive('table')) return null
  const attrs = editor.getAttributes('tableCell')
  const headerAttrs = editor.getAttributes('tableHeader')
  return normalizeSFTableCellAlign(attrs.align ?? headerAttrs.align)
}

export function setSForumTableAlign(editor: Editor, align: SFTableCellAlign | null) {
  if (!editor.isActive('table')) return false
  return editor.chain().focus().setCellAttribute('align', align).run()
}
