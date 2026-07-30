export const MAX_SELECTION_QUOTE_LENGTH = 500

export type SelectionQuoteTarget =
  | { kind: 'topic' }
  | { kind: 'comment', commentId: number }

export type SelectionQuoteRequest = {
  target: SelectionQuoteTarget
  markdown: string
}

type SelectionQuotePositionInput = {
  selectionRect: Pick<DOMRect, 'left' | 'top' | 'width' | 'height'>
  hostRect: Pick<DOMRect, 'left' | 'top'>
  hostScrollLeft: number
  hostScrollTop: number
  hostClientWidth: number
  toolbarWidth: number
  toolbarHeight: number
  edgeInset?: number
  gap?: number
}

export type SelectionQuotePosition = {
  left: number
  top: number
  placement: 'above' | 'below'
}

function truncateUnicode(value: string, maxLength: number) {
  return Array.from(value).slice(0, maxLength).join('')
}

function escapeMarkdownHtml(value: string) {
  return value
    .replaceAll('&', '&amp;')
    .replaceAll('<', '&lt;')
    .replaceAll('>', '&gt;')
}

export function normalizeSelectionQuote(value: string, maxLength = MAX_SELECTION_QUOTE_LENGTH) {
  const normalized = value
    .replace(/\r\n?/g, '\n')
    .replace(/\u00A0/g, ' ')
    .replace(/[\u200B-\u200D\uFEFF]/g, '')
    .split('\n')
    .map(line => line.replace(/[\t\f\v ]+/g, ' ').trim())
    .join('\n')
    .replace(/\n{3,}/g, '\n\n')
    .trim()

  return truncateUnicode(normalized, Math.max(0, maxLength)).trim()
}

export function buildSelectionQuoteMarkdown(value: string) {
  const normalized = normalizeSelectionQuote(value)
  if (!normalized) return ''

  const quoted = normalized
    .split('\n')
    .map(line => line ? `> ${escapeMarkdownHtml(line)}` : '>')
    .join('\n')

  return `${quoted}\n\n`
}

export function parseSelectionQuoteTarget(source: string | undefined, commentId: string | undefined): SelectionQuoteTarget | null {
  if (source === 'topic') return { kind: 'topic' }
  if (source !== 'comment') return null

  const parsedCommentId = Number(commentId)
  return Number.isInteger(parsedCommentId) && parsedCommentId > 0
    ? { kind: 'comment', commentId: parsedCommentId }
    : null
}

export function selectionQuoteToolbarPosition(input: SelectionQuotePositionInput): SelectionQuotePosition {
  const edgeInset = input.edgeInset ?? 12
  const gap = input.gap ?? 9
  const selectionLeft = input.selectionRect.left - input.hostRect.left + input.hostScrollLeft
  const selectionTop = input.selectionRect.top - input.hostRect.top + input.hostScrollTop
  const visibleLeft = input.hostScrollLeft + edgeInset
  const visibleRight = input.hostScrollLeft + input.hostClientWidth - edgeInset
  const desiredLeft = selectionLeft + input.selectionRect.width / 2 - input.toolbarWidth / 2
  const maxLeft = Math.max(visibleLeft, visibleRight - input.toolbarWidth)
  const left = Math.min(maxLeft, Math.max(visibleLeft, desiredLeft))
  const aboveTop = selectionTop - input.toolbarHeight - gap
  const visibleTop = input.hostScrollTop + edgeInset

  if (aboveTop >= visibleTop) {
    return { left, top: aboveTop, placement: 'above' }
  }

  return {
    left,
    top: selectionTop + input.selectionRect.height + gap,
    placement: 'below'
  }
}
