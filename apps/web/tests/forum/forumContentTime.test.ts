import { describe, expect, test } from 'bun:test'

import { formatForumContentTime } from '../../app/utils/forum/forumContentTime'
import type { SiteDateTimeSettings } from '../../app/utils/siteDateTime'

const settings: SiteDateTimeSettings = {
  timezone: 'Asia/Shanghai',
  dateFormat: 'Y/m/d',
  timeFormat: 'hidden',
  startOfWeek: 1
}
const now = new Date('2026-07-29T12:00:00.000Z')

describe('forum content time', () => {
  test('uses second, minute, hour, day, and month labels within one month', () => {
    expect(formatForumContentTime('2026-07-29T11:59:55.000Z', { settings, locale: 'zh-CN', now })).toContain('5')
    expect(formatForumContentTime('2026-07-29T11:55:00.000Z', { settings, locale: 'zh-CN', now })).toContain('5')
    expect(formatForumContentTime('2026-07-29T07:00:00.000Z', { settings, locale: 'zh-CN', now })).toContain('5')
    expect(formatForumContentTime('2026-07-24T12:00:00.000Z', { settings, locale: 'zh-CN', now })).toContain('5')
    expect(formatForumContentTime('2026-06-30T12:00:00.000Z', { settings, locale: 'zh-CN', now })).toMatch(/1\s*个月前/)
  })

  test('forces Y-m-d H:i:s in the site timezone after one month', () => {
    expect(formatForumContentTime('2026-06-01T00:00:00.000Z', {
      settings,
      locale: 'zh-CN',
      now
    })).toBe('2026-06-01 08:00:00')
  })

  test('clamps future clock skew and rejects invalid values', () => {
    expect(formatForumContentTime('2026-07-29T12:00:05.000Z', {
      settings,
      locale: 'en-US',
      now
    })).toMatch(/1 second ago/i)
    expect(formatForumContentTime('invalid', { settings, now })).toBe('')
  })
})
