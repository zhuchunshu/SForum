import { describe, expect, test } from 'bun:test'
import { userDisplayName } from '../../app/utils/forum/userDisplay'

describe('userDisplayName', () => {
  test('prefers displayName over username', () => {
    expect(userDisplayName({ displayName: '张三', username: 'zhangsan' })).toBe('张三')
  })

  test('falls back to username when displayName is empty or blank', () => {
    expect(userDisplayName({ displayName: '', username: 'zhangsan' })).toBe('zhangsan')
    expect(userDisplayName({ displayName: '   ', username: 'zhangsan' })).toBe('zhangsan')
  })

  test('uses the caller-provided fallback when both are missing', () => {
    expect(userDisplayName({ displayName: '', username: '' }, '#42')).toBe('#42')
    expect(userDisplayName(undefined, 'unknown')).toBe('unknown')
    expect(userDisplayName(null)).toBe('')
  })

  test('accepts PublicProfile-like objects with required fields', () => {
    expect(userDisplayName({ displayName: '李四', username: 'lisi', userId: 1 } as never)).toBe('李四')
  })

  test('ignores whitespace-only usernames too', () => {
    expect(userDisplayName({ displayName: '', username: '  ' }, 'fallback')).toBe('fallback')
  })
})
