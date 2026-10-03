import { describe, expect, test } from 'bun:test'
import { readFileSync } from 'node:fs'

import { isStreamingAPIPath } from '../../server/utils/apiStreamPath'

const source = (path: string) => readFileSync(new URL(path, import.meta.url), 'utf8')

describe('streaming API path predicate', () => {
  test('matches the host SSE channels only', () => {
    expect(isStreamingAPIPath('notifications/stream')).toBe(true)
    expect(isStreamingAPIPath('topics/42/comments/stream')).toBe(true)
    expect(isStreamingAPIPath('/topics/42/comments/stream/')).toBe(true)
  })

  test('does not capture ordinary API paths', () => {
    expect(isStreamingAPIPath('topics/42/comments')).toBe(false)
    expect(isStreamingAPIPath('topics/42/comments/revision')).toBe(false)
    expect(isStreamingAPIPath('topics/comments/stream')).toBe(false)
    expect(isStreamingAPIPath('topics/42/comments/stream/extra')).toBe(false)
    expect(isStreamingAPIPath('notifications/stream/extra')).toBe(false)
    expect(isStreamingAPIPath('notifications')).toBe(false)
  })
})

describe('Nitro API proxy keeps SSE unbuffered', () => {
  const apiProxy = source('../../server/routes/api/v1/[...path].ts')

  test('routes streaming paths through the raw pipe proxy', () => {
    // sendProxy 会缓冲响应，SSE 客户端会一个事件都收不到。
    expect(apiProxy).toContain('isStreamingAPIPath(path)')
    expect(apiProxy).toContain('proxyNotificationStream(event, target)')
    expect(apiProxy).toContain('proxyRouteRequest(event, target)')
  })

  test('raw pipe proxy disables intermediary buffering headers', () => {
    // 上游（Go API）声明的 no-transform 必须原样透出，Caddy 才不会压缩 SSE。
    const streamProxy = source('../../server/utils/notifications/notificationStreamProxy.ts')
    expect(streamProxy).toContain('response.pipe(event.node.res)')
    expect(streamProxy).not.toContain('sendProxy(event')
  })
})
