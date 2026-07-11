/**
 * 站点日期时间展示工具。
 * 与后端 site.timezone / site.date_format / site.time_format / site.start_of_week 对齐。
 * 库内时间仍为 UTC ISO；此处只负责按站点配置格式化为展示字符串。
 */

export type SiteDateFormat =
  | 'Y-m-d'
  | 'Y/m/d'
  | 'd/m/Y'
  | 'm/d/Y'
  | 'M j, Y'
  | 'j M Y'
  | 'relative'

export type SiteTimeFormat =
  | 'H:i'
  | 'H:i:s'
  | 'g:i a'
  | 'g:i A'
  | 'hidden'

export type SiteDateTimeSettings = {
  timezone: string
  dateFormat: SiteDateFormat
  timeFormat: SiteTimeFormat
  /** 0=周日 … 6=周六 */
  startOfWeek: number
}

export const siteDateFormats: SiteDateFormat[] = [
  'Y-m-d',
  'Y/m/d',
  'd/m/Y',
  'm/d/Y',
  'M j, Y',
  'j M Y',
  'relative'
]

export const siteTimeFormats: SiteTimeFormat[] = [
  'H:i',
  'H:i:s',
  'g:i a',
  'g:i A',
  'hidden'
]

/** 后台时区下拉的常用 IANA 列表（非穷尽，仍允许任意合法 IANA 经 API 校验）。 */
export const commonSiteTimezones = [
  'UTC',
  'Asia/Shanghai',
  'Asia/Hong_Kong',
  'Asia/Taipei',
  'Asia/Tokyo',
  'Asia/Seoul',
  'Asia/Singapore',
  'Asia/Bangkok',
  'Asia/Kolkata',
  'Asia/Dubai',
  'Europe/London',
  'Europe/Paris',
  'Europe/Berlin',
  'Europe/Moscow',
  'America/New_York',
  'America/Chicago',
  'America/Denver',
  'America/Los_Angeles',
  'America/Sao_Paulo',
  'Australia/Sydney',
  'Pacific/Auckland'
] as const

export const recommendedSiteDateTimeSettings: SiteDateTimeSettings = {
  timezone: 'UTC',
  dateFormat: 'Y-m-d',
  timeFormat: 'H:i',
  startOfWeek: 1
}

export function normalizeSiteTimezone(value: string | undefined | null, fallback = recommendedSiteDateTimeSettings.timezone): string {
  const trimmed = String(value || '').trim()
  return trimmed || fallback
}

export function normalizeSiteDateFormat(value: string | undefined | null): SiteDateFormat {
  const trimmed = String(value || '').trim()
  return (siteDateFormats as string[]).includes(trimmed)
    ? trimmed as SiteDateFormat
    : recommendedSiteDateTimeSettings.dateFormat
}

export function normalizeSiteTimeFormat(value: string | undefined | null): SiteTimeFormat {
  const trimmed = String(value || '').trim()
  return (siteTimeFormats as string[]).includes(trimmed)
    ? trimmed as SiteTimeFormat
    : recommendedSiteDateTimeSettings.timeFormat
}

export function normalizeSiteStartOfWeek(value: string | number | undefined | null): number {
  const n = typeof value === 'number' ? value : Number.parseInt(String(value ?? ''), 10)
  if (!Number.isFinite(n) || n < 0 || n > 6) {
    return recommendedSiteDateTimeSettings.startOfWeek
  }
  return Math.trunc(n)
}

export function resolveSiteDateTimeSettings(options: Record<string, string | undefined>): SiteDateTimeSettings {
  return {
    timezone: normalizeSiteTimezone(options['site.timezone']),
    dateFormat: normalizeSiteDateFormat(options['site.date_format']),
    timeFormat: normalizeSiteTimeFormat(options['site.time_format']),
    startOfWeek: normalizeSiteStartOfWeek(options['site.start_of_week'])
  }
}

type FormatParts = {
  year: string
  month: string
  day: string
  monthShort: string
  hour24: string
  hour12: string
  minute: string
  second: string
  dayPeriod: string
}

function pad2(value: number): string {
  return String(value).padStart(2, '0')
}

function readParts(date: Date, timeZone: string, locale: string): FormatParts | null {
  try {
    const formatter = new Intl.DateTimeFormat(locale || 'en-US', {
      timeZone,
      year: 'numeric',
      month: '2-digit',
      day: '2-digit',
      hour: '2-digit',
      minute: '2-digit',
      second: '2-digit',
      hourCycle: 'h23',
      weekday: 'short'
    })
    const parts = formatter.formatToParts(date)
    const map = Object.fromEntries(parts.filter(p => p.type !== 'literal').map(p => [p.type, p.value]))

    // 用 en-US 再取一次短月名，保证 M j, Y 类英文风格稳定。
    const enMonth = new Intl.DateTimeFormat('en-US', { timeZone, month: 'short' }).format(date)
    const hour24 = Number.parseInt(map.hour || '0', 10)
    const hour12Raw = hour24 % 12 === 0 ? 12 : hour24 % 12
    const dayPeriod = hour24 >= 12 ? 'pm' : 'am'

    return {
      year: map.year || '',
      month: map.month || '',
      day: map.day || '',
      monthShort: enMonth,
      hour24: pad2(hour24),
      hour12: String(hour12Raw),
      minute: map.minute || '00',
      second: map.second || '00',
      dayPeriod
    }
  } catch {
    return null
  }
}

function formatDatePart(parts: FormatParts, dateFormat: SiteDateFormat): string {
  const y = parts.year
  const m = parts.month
  const d = parts.day
  const mon = parts.monthShort
  // day 去掉前导零用于 j 风格
  const j = String(Number.parseInt(d, 10) || d)

  switch (dateFormat) {
    case 'Y-m-d':
      return `${y}-${m}-${d}`
    case 'Y/m/d':
      return `${y}/${m}/${d}`
    case 'd/m/Y':
      return `${d}/${m}/${y}`
    case 'm/d/Y':
      return `${m}/${d}/${y}`
    case 'M j, Y':
      return `${mon} ${j}, ${y}`
    case 'j M Y':
      return `${j} ${mon} ${y}`
    default:
      return `${y}-${m}-${d}`
  }
}

function formatTimePart(parts: FormatParts, timeFormat: SiteTimeFormat): string {
  switch (timeFormat) {
    case 'H:i':
      return `${parts.hour24}:${parts.minute}`
    case 'H:i:s':
      return `${parts.hour24}:${parts.minute}:${parts.second}`
    case 'g:i a':
      return `${parts.hour12}:${parts.minute} ${parts.dayPeriod}`
    case 'g:i A':
      return `${parts.hour12}:${parts.minute} ${parts.dayPeriod.toUpperCase()}`
    case 'hidden':
      return ''
    default:
      return `${parts.hour24}:${parts.minute}`
  }
}

function formatRelative(date: Date, now: Date, locale: string): string {
  const diffSec = Math.round((date.getTime() - now.getTime()) / 1000)
  const abs = Math.abs(diffSec)
  const rtf = new Intl.RelativeTimeFormat(locale || 'zh-CN', { numeric: 'auto' })

  if (abs < 60) {
    return rtf.format(diffSec, 'second')
  }
  if (abs < 3600) {
    return rtf.format(Math.round(diffSec / 60), 'minute')
  }
  if (abs < 86400) {
    return rtf.format(Math.round(diffSec / 3600), 'hour')
  }
  if (abs < 86400 * 30) {
    return rtf.format(Math.round(diffSec / 86400), 'day')
  }
  if (abs < 86400 * 365) {
    return rtf.format(Math.round(diffSec / (86400 * 30)), 'month')
  }
  return rtf.format(Math.round(diffSec / (86400 * 365)), 'year')
}

export type FormatSiteDateTimeOptions = {
  settings: SiteDateTimeSettings
  /** BCP 47 locale，影响相对时间与数字形态 */
  locale?: string
  /** 是否强制包含时间（忽略 hidden）；默认 true，除非 timeFormat=hidden */
  includeTime?: boolean
  /** relative 模式的参照时刻，默认 now */
  now?: Date
}

/**
 * 将 ISO/可解析时间字符串按站点配置格式化。
 * 无效输入返回空字符串。
 */
export function formatSiteDateTime(
  value: string | number | Date | null | undefined,
  options: FormatSiteDateTimeOptions
): string {
  if (value == null || value === '') {
    return ''
  }
  const date = value instanceof Date ? value : new Date(value)
  if (Number.isNaN(date.getTime())) {
    return ''
  }

  const { settings } = options
  const locale = options.locale || 'zh-CN'
  const timeZone = settings.timezone || 'UTC'

  if (settings.dateFormat === 'relative') {
    return formatRelative(date, options.now || new Date(), locale)
  }

  const parts = readParts(date, timeZone, locale)
  if (!parts) {
    // 时区非法时回退 UTC，避免整页空白
    const fallback = readParts(date, 'UTC', locale)
    if (!fallback) {
      return ''
    }
    return joinDateTime(fallback, settings, options.includeTime)
  }
  return joinDateTime(parts, settings, options.includeTime)
}

function joinDateTime(parts: FormatParts, settings: SiteDateTimeSettings, includeTime?: boolean): string {
  const dateText = formatDatePart(parts, settings.dateFormat)
  const showTime = includeTime !== false && settings.timeFormat !== 'hidden'
  if (!showTime) {
    return dateText
  }
  const timeText = formatTimePart(parts, settings.timeFormat)
  return timeText ? `${dateText} ${timeText}` : dateText
}

/** 生成预览用示例（后台表单实时预览）。 */
export function previewSiteDateTime(settings: SiteDateTimeSettings, locale = 'zh-CN', sample?: Date): string {
  return formatSiteDateTime(sample || new Date('2026-07-12T06:30:00.000Z'), {
    settings,
    locale
  })
}
