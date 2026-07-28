import { formatSiteDateTime, type SiteDateTimeSettings } from '../siteDateTime'

const SECOND = 1000
const MINUTE = 60 * SECOND
const HOUR = 60 * MINUTE
const DAY = 24 * HOUR
const MONTH = 30 * DAY
const MONTH_LABEL_FROM = 28 * DAY

export type ForumContentTimeOptions = {
  settings: SiteDateTimeSettings
  locale?: string
  now?: Date | number
}

/**
 * 帖子详情时间规则：一个月内显示相对时间，更早的时间固定为 Y-m-d H:i:s。
 * 28-30 天归为「1 个月前」，使月单位与「超过一月显示具体日期」边界一致。
 */
export function formatForumContentTime(
  value: string | number | Date | null | undefined,
  options: ForumContentTimeOptions
): string {
  if (value == null || value === '') {
    return ''
  }
  const date = value instanceof Date ? value : new Date(value)
  if (Number.isNaN(date.getTime())) {
    return ''
  }

  const now = options.now instanceof Date
    ? options.now
    : new Date(options.now ?? Date.now())
  const age = Math.max(0, now.getTime() - date.getTime())

  if (age > MONTH) {
    return formatSiteDateTime(date, {
      settings: {
        ...options.settings,
        dateFormat: 'Y-m-d',
        timeFormat: 'H:i:s'
      },
      locale: options.locale
    })
  }

  const relative = new Intl.RelativeTimeFormat(options.locale || 'zh-CN', { numeric: 'always' })
  if (age < MINUTE) {
    return relative.format(-Math.max(1, Math.floor(age / SECOND)), 'second')
  }
  if (age < HOUR) {
    return relative.format(-Math.max(1, Math.floor(age / MINUTE)), 'minute')
  }
  if (age < DAY) {
    return relative.format(-Math.max(1, Math.floor(age / HOUR)), 'hour')
  }
  if (age < MONTH_LABEL_FROM) {
    return relative.format(-Math.max(1, Math.floor(age / DAY)), 'day')
  }
  return relative.format(-1, 'month')
}
