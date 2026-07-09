// M7: 前端 scheme 白名单兜底，防止 javascript:/data: 等危险协议经 :href 触发 XSS。
// 服务端 Profile 已校验 http/https，这里作为渲染层纵深防御。

/**
 * 返回安全的 URL：仅允许 http/https/mailto 协议，否则返回 'about:blank'。
 */
export function safeUrl(url: string | undefined | null): string {
  if (!url) {
    return ''
  }
  const trimmed = String(url).trim()
  if (/^(https?:|mailto:)/i.test(trimmed)) {
    return trimmed
  }
  // 相对路径（如 /profile）允许；其余（javascript:、data:、vbscript: 等）置空。
  if (trimmed.startsWith('/')) {
    return trimmed
  }
  return ''
}
