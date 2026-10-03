// Host 流式 API 路径判定：这些路径必须走裸管道透传（node:http pipe），
// 不能落到 Nitro 的 sendProxy，否则响应会被缓冲，SSE 客户端一个事件都收不到。
//
// 与 apps/api 侧保持一致：
//   - /api/v1/notifications/stream（收件人修订信号）
//   - /api/v1/topics/:topicID/comments/stream（主题评论修订信号）
//
// path 参数是 Nitro catch-all 已经分段编码后的相对路径（无前导斜杠），
// 例如 "topics/42/comments/stream"。
export function isStreamingAPIPath(path: string): boolean {
  const normalized = path.replace(/^\/+/, '').replace(/\/+$/, '')
  if (normalized === 'notifications/stream') {
    return true
  }
  return /^topics\/[^/]+\/comments\/stream$/.test(normalized)
}
