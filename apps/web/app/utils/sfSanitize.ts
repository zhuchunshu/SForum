// H3: 前端二次净化 HTML 内容，作为服务端 bluemonday 之外的纵深防御。
// 仅在 client 端运行 DOMPurify（依赖 DOM）；SSR 阶段原样返回（信任服务端已净化的内容），
// 避免引入依赖 window 的逻辑到服务端渲染。
import DOMPurify from 'dompurify'

// 与服务端 bluemonday UGCPolicy 大致对齐的白名单：
// 允许常见富文本标签，禁用脚本/事件处理器/危险属性。
const PURIFY_CONFIG = {
  ALLOWED_TAGS: [
    'a', 'abbr', 'b', 'blockquote', 'br', 'caption', 'cite', 'code', 'col',
    'colgroup', 'dd', 'del', 'details', 'div', 'dl', 'dt', 'em', 'figcaption',
    'figure', 'h1', 'h2', 'h3', 'h4', 'h5', 'h6', 'hr', 'i', 'img', 'ins',
    'kbd', 'li', 'mark', 'ol', 'p', 'pre', 'q', 's', 'samp', 'small', 'span',
    'strong', 'sub', 'summary', 'sup', 'table', 'tbody', 'td', 'tfoot', 'th',
    'thead', 'tr', 'u', 'ul', 'var', 'input'
  ],
  ALLOWED_ATTR: [
    'href', 'title', 'alt', 'src', 'width', 'height', 'class', 'id',
    'target', 'rel', 'colspan', 'rowspan', 'start', 'type', 'checked',
    'disabled', 'open'
  ],
  // 禁止任何脚本/样式/表单提交相关内容。
  FORBID_TAGS: ['script', 'style', 'form', 'iframe', 'object', 'embed'],
  FORBID_ATTR: ['onerror', 'onload', 'onclick', 'onmouseover', 'style'],
  RETURN_TRUSTED_TYPE: false
}

/**
 * 净化 HTML 字符串，client 端用 DOMPurify，SSR 端原样返回。
 * 用法：v-html="sanitizeHtml(content)"
 */
export function sanitizeHtml(html: string | undefined | null): string {
  if (!html) {
    return ''
  }
  if (import.meta.server) {
    // SSR 阶段无 DOM，信任服务端 bluemonday 已净化。
    return html
  }
  return String(DOMPurify.sanitize(html, PURIFY_CONFIG))
}
