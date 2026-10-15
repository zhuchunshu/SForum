/**
 * 正文外链呈现契约（与后端 `apps/api/app/Support/ContentLinks` 对齐）。
 *
 * 服务端净化后已把正文链接统一成最终形态：
 * - 所有链接 `target="_blank"`，`rel="nofollow noreferrer noopener"`；
 * - 绝对 http(s) 链接改写成 `/outbound?u=<base64url>` 站内中转页。
 *
 * 本模块承担三件事：
 * 1. 中转页 `?u=` 的唯一解码实现（页面与指令共用，避免两套解析语义）；
 * 2. 为历史内容与插件渲染出的旧 HTML 补齐同样的形态（`v-outbound` 指令）；
 * 3. 「站内 / 白名单 / 运营开关」的点击时判定——判定只发生在点击时，
 *    所以运营改配置不需要重渲染任何历史内容。
 */

export const OUTBOUND_INTERSTITIAL_PATH = '/outbound'
export const OUTBOUND_TARGET_PARAM = 'u'
export const OUTBOUND_TARGET_MAX_LENGTH = 2048
/** 与后端 canonical rel 顺序一致：nofollow → noreferrer → noopener。 */
export const OUTBOUND_REL_TOKENS = ['nofollow', 'noreferrer', 'noopener'] as const
export const OUTBOUND_REL_VALUE = OUTBOUND_REL_TOKENS.join(' ')

/** 不参与外链呈现的 href 前缀（与后端跳过规则对齐）。 */
const UNMANAGED_HREF_PREFIXES = ['#', 'mailto:', 'tel:', 'javascript:', 'data:', 'blob:']

const MENTION_ANCHOR_SELECTOR = 'a.sf-mention, a[data-sf-mention]'
/** 图片灯箱触发器：点击由查看器接管，不能改成新标签页跳转。 */
const IMAGE_VIEWER_ANCHOR_SELECTOR = 'a[data-sforum-image-viewer]'

function toBase64Url(value: string) {
  const bytes = new TextEncoder().encode(value)
  let binary = ''
  for (const byte of bytes) {
    binary += String.fromCharCode(byte)
  }
  return btoa(binary).replace(/\+/g, '-').replace(/\//g, '_').replace(/=+$/, '')
}

function fromBase64Url(payload: string) {
  const normalized = payload.replace(/-/g, '+').replace(/_/g, '/')
  const padded = normalized + '='.repeat((4 - (normalized.length % 4)) % 4)
  const binary = atob(padded)
  const bytes = new Uint8Array(binary.length)
  for (let index = 0; index < binary.length; index += 1) {
    bytes[index] = binary.charCodeAt(index)
  }
  return new TextDecoder().decode(bytes)
}

/** 编码中转目标（页面侧不使用，仅供指令与测试构造链接）。 */
export function encodeOutboundTarget(target: string) {
  return toBase64Url(target)
}

/** 中转目标是否可安全跳转：仅 http(s)、无 userinfo、无控制字符、长度受限。 */
export function isSafeOutboundTarget(target: string) {
  const value = String(target || '').trim()
  if (!value || value.length > OUTBOUND_TARGET_MAX_LENGTH) {
    return false
  }
  // 控制字符会让浏览器解析出的地址与展示不一致。
  if (/[\u0000-\u001f\u007f]/.test(value)) {
    return false
  }
  let parsed: URL
  try {
    parsed = new URL(value)
  } catch {
    return false
  }
  if (parsed.protocol !== 'http:' && parsed.protocol !== 'https:') {
    return false
  }
  if (!parsed.hostname) {
    return false
  }
  // 带凭据的地址一律拒绝：中转页不转发也不展示任何凭据。
  return !parsed.username && !parsed.password
}

/**
 * 解码 `?u=`；非法或非 http(s) 目标返回 null。
 * 入参刻意放宽到 unknown：路由 query 的值可能是数组（`?u=a&u=b`），
 * 只接受其中的第一个字符串，其余一律按非法处理。
 */
export function decodeOutboundTarget(raw: unknown) {
  const candidate = Array.isArray(raw) ? raw[0] : raw
  const payload = typeof candidate === 'string' ? candidate.trim() : ''
  if (!payload) {
    return null
  }
  let decoded = ''
  try {
    decoded = fromBase64Url(payload)
  } catch {
    return null
  }
  return isSafeOutboundTarget(decoded) ? decoded.trim() : null
}

/** 目标链接的 host（含端口，小写）；无法解析时返回空串。 */
export function outboundTargetHost(target: string) {
  try {
    return new URL(String(target || '').trim()).host.toLowerCase()
  } catch {
    return ''
  }
}

/** 构造中转链接。 */
export function outboundInterstitialHref(target: string) {
  return `${OUTBOUND_INTERSTITIAL_PATH}?${OUTBOUND_TARGET_PARAM}=${encodeOutboundTarget(target)}`
}

/** 是否已经是中转链接（幂等判断）。 */
export function isInterstitialHref(href: string) {
  const value = String(href || '').trim().toLowerCase()
  return value.startsWith(`${OUTBOUND_INTERSTITIAL_PATH}?${OUTBOUND_TARGET_PARAM}=`)
}

function hostFromCandidate(candidate: string) {
  const value = String(candidate || '').trim()
  if (!value) {
    return ''
  }
  try {
    const parsed = new URL(value.includes('://') ? value : `https://${value}`)
    return parsed.host.toLowerCase()
  } catch {
    return ''
  }
}

/**
 * 汇总「站内域名」集合：站点配置 URL 的 host、site.domain、当前页面 host，
 * 以及去掉端口后的裸域名（site.url 带端口而 site.domain 不带的情况很常见）。
 */
export function contentLinkInternalHosts(input: { siteUrl?: string | null, siteDomain?: string | null, currentHost?: string | null } = {}) {
  const hosts = new Set<string>()
  for (const candidate of [input.siteUrl, input.siteDomain, input.currentHost]) {
    const host = hostFromCandidate(String(candidate || ''))
    if (!host) {
      continue
    }
    hosts.add(host)
    const bare = host.split(':')[0]
    if (bare) {
      hosts.add(bare)
    }
  }
  return Array.from(hosts)
}

/** 解析免中转域名白名单（逐行或逗号分隔，归一为 host[:port]）。 */
export function parseTrustedDomains(value: string | null | undefined) {
  const hosts = new Set<string>()
  for (const rawEntry of String(value || '').split(/[\n,]/)) {
    const host = hostFromCandidate(rawEntry)
    if (host) {
      hosts.add(host)
    }
  }
  return Array.from(hosts)
}

/** host 命中白名单：支持精确 host[:port] 与裸域名两种写法。 */
export function isTrustedOutboundHost(host: string, trustedDomains: string[]) {
  const value = String(host || '').trim().toLowerCase()
  if (!value) {
    return false
  }
  const bare = value.split(':')[0] || ''
  return trustedDomains.some((entry) => {
    const normalized = String(entry || '').trim().toLowerCase()
    if (!normalized) {
      return false
    }
    if (normalized === value) {
      return true
    }
    // 白名单不带端口时按 hostname 匹配（例如 example.com 覆盖 example.com:8443）。
    return !normalized.includes(':') && normalized === bare
  })
}

/** 绝对 http(s) 且 host 不在站内集合内 → 站外。相对路径一律视为站内。 */
export function isOffsiteContentHref(href: string, internalHosts: string[]) {
  const value = String(href || '').trim()
  if (!value) {
    return false
  }
  if (value.startsWith('//')) {
    const host = hostFromCandidate(`https:${value}`)
    return Boolean(host) && !internalHosts.includes(host) && !internalHosts.includes(host.split(':')[0] || '')
  }
  let parsed: URL
  try {
    parsed = new URL(value)
  } catch {
    return false
  }
  if (parsed.protocol !== 'http:' && parsed.protocol !== 'https:') {
    return false
  }
  const host = parsed.host.toLowerCase()
  return !internalHosts.includes(host) && !internalHosts.includes(host.split(':')[0] || '')
}

export type ContentLinkPolicy = {
  /** 运营是否启用站外中转页（关闭时站外链接直接新标签打开）。 */
  interstitial: boolean
  /** 免中转域名白名单（已归一 host[:port]）。 */
  trustedDomains: string[]
  /** 站内域名集合（site.url / site.domain / 当前 host）。 */
  internalHosts: string[]
}

function ensureBlankTargetAndRel(anchor: HTMLAnchorElement) {
  anchor.setAttribute('target', '_blank')
  const tokens = new Set((anchor.getAttribute('rel') || '').split(/\s+/).filter(Boolean))
  for (const token of OUTBOUND_REL_TOKENS) {
    tokens.add(token)
  }
  anchor.setAttribute('rel', Array.from(tokens).join(' '))
}

/**
 * 为容器内的正文链接补齐「新标签页 + 站外中转」形态，返回被改写的锚点数。
 * 幂等：已是中转链接、提及链接、图片灯箱、页内锚点、mailto/tel 都跳过。
 */
export function decorateContentLinks(root: Element | null, policy: ContentLinkPolicy) {
  if (!root || typeof root.querySelectorAll !== 'function') {
    return 0
  }
  let rewritten = 0
  for (const anchor of Array.from(root.querySelectorAll<HTMLAnchorElement>('a[href]'))) {
    if (anchor.matches(IMAGE_VIEWER_ANCHOR_SELECTOR) || anchor.matches(MENTION_ANCHOR_SELECTOR)) {
      continue
    }
    const href = (anchor.getAttribute('href') || '').trim()
    if (!href) {
      continue
    }
    const lowered = href.toLowerCase()
    if (UNMANAGED_HREF_PREFIXES.some(prefix => lowered.startsWith(prefix))) {
      continue
    }
    if (
      policy.interstitial
      && !isInterstitialHref(href)
      && !href.startsWith('//')
      && isOffsiteContentHref(href, policy.internalHosts)
      && isSafeOutboundTarget(href)
    ) {
      const host = outboundTargetHost(href)
      if (host && !isTrustedOutboundHost(host, policy.trustedDomains)) {
        anchor.setAttribute('href', outboundInterstitialHref(href))
        rewritten += 1
      }
    }
    ensureBlankTargetAndRel(anchor)
  }
  return rewritten
}
