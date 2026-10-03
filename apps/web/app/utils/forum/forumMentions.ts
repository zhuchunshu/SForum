/**
 * @提及（mention）渲染契约：识别、链接化、预览定位。
 *
 * 语法必须与后端 `apps/api/app/Models/Forum/mentions.go` 的 MentionedUsernames 对齐，
 * 否则前端高亮就在承诺一个不会触发通知的提及：
 * - 边界：`@` 之前不能是字母/数字/下划线，`user@example.com` 不算提及。
 * - token：字母/数字/下划线/连字符，与 identity.UsernamePolicy 允许的字符集一致。
 * - 长度：64，即 identity.username.max_length 的配置硬上限。
 * 代码块与行内代码不参与识别，与后端跳过 ast.KindCodeBlock/KindCodeSpan 的规则一致。
 */

/** identity.username.max_length 的硬上限（Options.usernameMaxLengthMax）。 */
export const FORUM_MENTION_MAX_USERNAME_LENGTH = 64

const MENTION_BOUNDARY_SOURCE = '[^\\p{L}\\p{N}_]'
const MENTION_TOKEN_SOURCE = '[\\p{L}\\p{N}_-]'

/** 提及语法源码；导出让静态契约测试与后端同款语义对照。 */
export function forumMentionPatternSource() {
  // 尾部否定前瞻比后端更严格：token 后面还有用户名字符时整体放弃，
  // 避免把超长串截成半截 @xxx 高亮（后端会截断成不存在的用户名，本就不会通知）。
  return `(?:^|${MENTION_BOUNDARY_SOURCE})@(${MENTION_TOKEN_SOURCE}{1,${FORUM_MENTION_MAX_USERNAME_LENGTH}})(?!${MENTION_TOKEN_SOURCE})`
}

/** 每次调用返回独立实例，避免共享 `lastIndex` 造成的漏匹配。 */
export function forumMentionPattern() {
  return new RegExp(forumMentionPatternSource(), 'gu')
}

export type ForumMentionToken = {
  /** 用户名（不含 @） */
  username: string
  /** 命中的完整文本（含 @） */
  text: string
  /** 在输入文本中的起始下标，指向 @ */
  start: number
}

/** 从纯文本中提取提及 token；顺序与出现顺序一致。 */
export function forumMentionTokens(text: string): ForumMentionToken[] {
  if (!text.includes('@')) {
    return []
  }
  const pattern = forumMentionPattern()
  const tokens: ForumMentionToken[] = []
  let match = pattern.exec(text)
  while (match) {
    const username = match[1] || ''
    if (username) {
      const at = match[0].lastIndexOf('@')
      tokens.push({
        username,
        text: `@${username}`,
        start: match.index + at
      })
    }
    match = pattern.exec(text)
  }
  return tokens
}

export type ForumMentionLinkifyOptions = {
  /** 由用户名生成主页地址；未提供时使用站点默认公开资料路径。 */
  hrefFor?: (username: string) => string
}

const TEXT_NODE = 3
const ELEMENT_NODE = 1
const SKIPPED_TAGS = new Set(['A', 'CODE', 'PRE', 'SCRIPT', 'STYLE', 'TEXTAREA'])
const SKIPPED_ATTRIBUTE = 'data-sf-mention-ignore'

function isSkippedElement(element: Element) {
  return SKIPPED_TAGS.has(element.tagName) || element.hasAttribute(SKIPPED_ATTRIBUTE)
}

function collectTextNodes(node: Node, collected: Text[]) {
  for (let child = node.firstChild; child; child = child.nextSibling) {
    if (child.nodeType === TEXT_NODE) {
      collected.push(child as Text)
      continue
    }
    if (child.nodeType !== ELEMENT_NODE) {
      continue
    }
    const element = child as Element
    if (!isSkippedElement(element)) {
      collectTextNodes(element, collected)
    }
  }
}

function mentionAnchor(document: Document, hrefFor: (username: string) => string, token: ForumMentionToken) {
  const link = document.createElement('a')
  link.className = 'sf-mention'
  link.setAttribute('href', hrefFor(token.username))
  link.setAttribute('data-sf-mention', token.username)
  link.setAttribute('aria-haspopup', 'dialog')
  link.setAttribute('aria-expanded', 'false')
  link.textContent = token.text
  return link
}

function defaultMentionHref(username: string) {
  return `/u/${encodeURIComponent(username)}`
}

/**
 * 把 `root` 内可见文本里的 @提及 链接化，返回新建的锚点数量。
 * 幂等：已链接化的锚点位于 `a` 元素内，递归时会整棵跳过，因此重复调用不产生嵌套链接。
 */
export function linkifyForumMentions(root: Element | null, options: ForumMentionLinkifyOptions = {}) {
  const document = root?.ownerDocument
  if (!root || !document?.createElement) {
    return 0
  }
  const hrefFor = options.hrefFor || defaultMentionHref
  const textNodes: Text[] = []
  collectTextNodes(root, textNodes)

  let created = 0
  for (const textNode of textNodes) {
    const tokens = forumMentionTokens(textNode.data)
    if (!tokens.length) {
      continue
    }
    const fragment = document.createDocumentFragment()
    let cursor = 0
    for (const token of tokens) {
      if (token.start > cursor) {
        fragment.appendChild(document.createTextNode(textNode.data.slice(cursor, token.start)))
      }
      fragment.appendChild(mentionAnchor(document, hrefFor, token))
      cursor = token.start + token.text.length
    }
    if (cursor < textNode.data.length) {
      fragment.appendChild(document.createTextNode(textNode.data.slice(cursor)))
    }
    created += tokens.length
    textNode.parentNode?.replaceChild(fragment, textNode)
  }
  return created
}

/** 提及锚点选择器：类名与 data 属性任一命中即认为是提及链接。 */
export const FORUM_MENTION_ANCHOR_SELECTOR = 'a.sf-mention, a[data-sf-mention]'

/** 从事件目标反解被点击的提及锚点；不是提及链接时返回 null。 */
export function mentionAnchorFromEvent(target: EventTarget | null): HTMLElement | null {
  const element = target as Element | null
  if (!element || typeof element.closest !== 'function') {
    return null
  }
  return element.closest(FORUM_MENTION_ANCHOR_SELECTOR) as HTMLElement | null
}

/** 从事件目标反解被点击的提及用户名；不是提及链接时返回 null。 */
export function mentionUsernameFromEvent(target: EventTarget | null): string | null {
  const link = mentionAnchorFromEvent(target)
  if (!link) {
    return null
  }
  const username = link.getAttribute('data-sf-mention')
  if (username) {
    return username
  }
  // 兜底：锚点属性被净化器剥离时，从 /u/<name> 反解。
  const href = link.getAttribute('href') || ''
  const segment = href.split(/[?#]/)[0]?.split('/').filter(Boolean).pop()
  if (!segment) {
    return null
  }
  try {
    return decodeURIComponent(segment)
  } catch {
    return segment
  }
}

export type ForumMentionPreviewAnchor = {
  top: number
  bottom: number
  left: number
  width: number
}

export type ForumMentionPreviewViewport = {
  width: number
  height: number
}

export type ForumMentionPreviewPlacement = {
  placement: 'above' | 'below'
  /** fixed 定位的 top；placement=above 时由 CSS translateY(-100%) 上移卡片。 */
  top: number
  left: number
  width: number
}

const PREVIEW_WIDTH = 320
const PREVIEW_GAP = 8
const VIEWPORT_MARGIN = 8
// 卡片实际高度取决于简介行数；用估算高度决定向上/向下翻转，避免渲染前测量。
const PREVIEW_ESTIMATED_HEIGHT = 220

/** 计算提及预览卡片的 fixed 定位，靠近视口底部时翻转到锚点上方。 */
export function forumMentionPreviewPosition(
  anchor: ForumMentionPreviewAnchor,
  viewport: ForumMentionPreviewViewport
): ForumMentionPreviewPlacement {
  const available = viewport.width - VIEWPORT_MARGIN * 2
  const width = available <= 0 ? viewport.width : Math.min(PREVIEW_WIDTH, available)
  const spaceBelow = viewport.height - anchor.bottom - PREVIEW_GAP
  const spaceAbove = anchor.top - PREVIEW_GAP
  const placement: ForumMentionPreviewPlacement['placement'] =
    spaceBelow < PREVIEW_ESTIMATED_HEIGHT && spaceAbove > spaceBelow ? 'above' : 'below'
  const top = placement === 'below' ? anchor.bottom + PREVIEW_GAP : anchor.top - PREVIEW_GAP
  const centered = anchor.left + anchor.width / 2 - width / 2
  const maxLeft = Math.max(VIEWPORT_MARGIN, viewport.width - width - VIEWPORT_MARGIN)
  return {
    placement,
    top,
    left: Math.min(Math.max(VIEWPORT_MARGIN, centered), maxLeft),
    width
  }
}
