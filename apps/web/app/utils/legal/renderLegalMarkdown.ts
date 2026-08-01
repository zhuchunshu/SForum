import { marked } from 'marked'
import { parseFragment, type DefaultTreeAdapterTypes } from 'parse5'

const ALLOWED_TAGS = new Set([
  'a',
  'abbr',
  'b',
  'blockquote',
  'br',
  'code',
  'del',
  'em',
  'h1',
  'h2',
  'h3',
  'h4',
  'h5',
  'h6',
  'hr',
  'img',
  'input',
  'ins',
  'kbd',
  'li',
  'ol',
  'p',
  'pre',
  'q',
  's',
  'small',
  'strong',
  'sub',
  'sup',
  'table',
  'tbody',
  'td',
  'tfoot',
  'th',
  'thead',
  'tr',
  'ul'
])

const VOID_TAGS = new Set(['br', 'hr', 'img', 'input'])
const FORBIDDEN_TAGS = new Set(['script', 'style', 'iframe', 'object', 'embed', 'form', 'template', 'svg', 'math'])

/**
 * 将法律/指南 Markdown 转为安全 HTML。
 * 允许常规 Markdown 结构，剥离原始 HTML 和危险 URL。
 */
export function renderLegalMarkdown(source: string | null | undefined): string {
  const value = typeof source === 'string' ? source.trim() : ''
  if (!value) return ''

  const html = String(marked.parse(value))
  return serializeSanitizedNodes(parseFragment(html).childNodes)
}

function serializeSanitizedNodes(nodes: DefaultTreeAdapterTypes.ChildNode[]): string {
  return nodes.map(node => serializeSanitizedNode(node)).join('')
}

function serializeSanitizedNode(node: DefaultTreeAdapterTypes.ChildNode): string {
  if (node.nodeName === '#text' && 'value' in node) {
    return escapeHtml(node.value)
  }
  if (node.nodeName === '#comment' || node.nodeName === '#documentType') {
    return ''
  }
  if (!('tagName' in node) || !('attrs' in node) || !('childNodes' in node)) {
    return ''
  }

  const tag = node.tagName.toLowerCase()
  if (FORBIDDEN_TAGS.has(tag)) {
    return ''
  }

  const children = serializeSanitizedNodes(node.childNodes)
  if (!ALLOWED_TAGS.has(tag)) {
    return children
  }

  const attrs = sanitizeAttributes(tag, node.attrs)
  if (tag === 'a' && !attrs.href) {
    return children
  }
  if (tag === 'img' && !attrs.src) {
    return ''
  }

  const serializedAttrs = Object.entries(attrs)
    .map(([name, value]) => ` ${name}="${escapeHtml(value)}"`)
    .join('')

  if (VOID_TAGS.has(tag)) {
    return `<${tag}${serializedAttrs}>`
  }
  return `<${tag}${serializedAttrs}>${children}</${tag}>`
}

function sanitizeAttributes(
  tag: string,
  attrs: DefaultTreeAdapterTypes.Element['attrs']
): Record<string, string> {
  const result: Record<string, string> = {}

  for (const attr of attrs) {
    const name = attr.name.toLowerCase()
    if (!name || name.startsWith('on') || name.startsWith('v-') || name.startsWith(':') || name.startsWith('@') || name.startsWith('#')) {
      continue
    }

    switch (tag) {
      case 'a':
        if (name === 'href' && isSafeUrl(attr.value)) {
          result.href = attr.value
        } else if (name === 'title') {
          result.title = attr.value
        }
        break
      case 'img':
        if (name === 'src' && isSafeUrl(attr.value)) {
          result.src = attr.value
        } else if (name === 'alt' || name === 'title') {
          result[name] = attr.value
        } else if ((name === 'width' || name === 'height') && /^\d+$/.test(attr.value.trim())) {
          result[name] = attr.value.trim()
        }
        break
      case 'input':
        if (name === 'type' && attr.value === 'checkbox') {
          result.type = 'checkbox'
        } else if ((name === 'checked' || name === 'disabled') && attr.value !== 'false') {
          result[name] = name
        }
        break
      case 'ol':
        if (name === 'start' && /^-?\d+$/.test(attr.value.trim())) {
          result.start = attr.value.trim()
        }
        break
      case 'td':
      case 'th':
        if ((name === 'colspan' || name === 'rowspan') && /^\d+$/.test(attr.value.trim())) {
          result[name] = attr.value.trim()
        } else if (name === 'align' && /^(left|right|center|justify)$/.test(attr.value.trim().toLowerCase())) {
          result.align = attr.value.trim().toLowerCase()
        }
        break
      default:
        break
    }
  }

  return result
}

function isSafeUrl(value: string): boolean {
  const canonical = value.replace(/[\u0000-\u0020\u007f]/g, '').toLowerCase()
  return !['javascript:', 'vbscript:', 'data:', 'file:'].some(scheme => canonical.includes(scheme))
}

function escapeHtml(value: string): string {
  return value
    .replaceAll('&', '&amp;')
    .replaceAll('<', '&lt;')
    .replaceAll('>', '&gt;')
    .replaceAll('"', '&quot;')
    .replaceAll("'", '&#39;')
}
