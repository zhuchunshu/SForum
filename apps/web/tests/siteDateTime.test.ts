import { describe, expect, it } from 'vitest'
import {
  formatSiteDateTime,
  normalizeSiteDateFormat,
  normalizeSiteStartOfWeek,
  normalizeSiteTimeFormat,
  normalizeSiteTimezone,
  previewSiteDateTime,
  recommendedSiteDateTimeSettings,
  resolveSiteDateTimeSettings
} from '../app/utils/siteDateTime'

describe('siteDateTime', () => {
  it('normalizes invalid values to recommended defaults', () => {
    expect(normalizeSiteTimezone('')).toBe('UTC')
    expect(normalizeSiteDateFormat('nope')).toBe('Y-m-d')
    expect(normalizeSiteTimeFormat('nope')).toBe('H:i')
    expect(normalizeSiteStartOfWeek(99)).toBe(1)
    expect(normalizeSiteStartOfWeek('0')).toBe(0)
  })

  it('resolves settings from option map', () => {
    const settings = resolveSiteDateTimeSettings({
      'site.timezone': 'Asia/Shanghai',
      'site.date_format': 'Y/m/d',
      'site.time_format': 'g:i A',
      'site.start_of_week': '0'
    })
    expect(settings).toEqual({
      timezone: 'Asia/Shanghai',
      dateFormat: 'Y/m/d',
      timeFormat: 'g:i A',
      startOfWeek: 0
    })
  })

  it('formats a fixed UTC instant in Asia/Shanghai as Y-m-d H:i', () => {
    // 2026-07-12T06:30:00Z → Asia/Shanghai +8 → 2026-07-12 14:30
    const text = formatSiteDateTime('2026-07-12T06:30:00.000Z', {
      settings: {
        timezone: 'Asia/Shanghai',
        dateFormat: 'Y-m-d',
        timeFormat: 'H:i',
        startOfWeek: 1
      },
      locale: 'zh-CN'
    })
    expect(text).toBe('2026-07-12 14:30')
  })

  it('hides time when time format is hidden', () => {
    const text = formatSiteDateTime('2026-07-12T06:30:00.000Z', {
      settings: {
        timezone: 'UTC',
        dateFormat: 'Y-m-d',
        timeFormat: 'hidden',
        startOfWeek: 1
      }
    })
    expect(text).toBe('2026-07-12')
  })

  it('supports relative date format', () => {
    const now = new Date('2026-07-12T08:00:00.000Z')
    const text = formatSiteDateTime('2026-07-12T06:00:00.000Z', {
      settings: {
        timezone: 'UTC',
        dateFormat: 'relative',
        timeFormat: 'H:i',
        startOfWeek: 1
      },
      locale: 'en-US',
      now
    })
    expect(text).toMatch(/2 hours ago|2 hr\. ago/i)
  })

  it('previews recommended defaults', () => {
    const preview = previewSiteDateTime(recommendedSiteDateTimeSettings, 'en-US')
    expect(preview).toContain('2026-07-12')
  })

  it('returns empty string for invalid input', () => {
    expect(formatSiteDateTime('not-a-date', {
      settings: recommendedSiteDateTimeSettings
    })).toBe('')
    expect(formatSiteDateTime(null, {
      settings: recommendedSiteDateTimeSettings
    })).toBe('')
  })
})
