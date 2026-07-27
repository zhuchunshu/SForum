import type { PublicProfile, ProfileActivity as ApiProfileActivity } from '~/composables/profile/useProfileApi'
import type { SiteDateTimeSettings } from '../siteDateTime'
import { forumTopicPath, type TopicUrlMode } from '../forum/forumTaxonomy'

type ProfileActivity = ApiProfileActivity

export type ProfileActivityGroup = {
  key: string
  label: string
  dateLabel: string
  items: ProfileActivityView[]
}

export type ProfileActivityView = ProfileActivity & {
  to: string
  timeLabel: string
}

type GroupOptions = {
  settings: SiteDateTimeSettings
  locale: string
  topicUrlMode: TopicUrlMode
  labels: {
    today: string
    yesterday: string
  }
  now?: Date
}

type DateParts = {
  year: number
  month: number
  day: number
}

function dateParts(value: Date, timeZone: string, locale: string): DateParts | null {
  try {
    const formatter = new Intl.DateTimeFormat(locale || 'en-US', {
      timeZone: timeZone || 'UTC',
      year: 'numeric',
      month: '2-digit',
      day: '2-digit'
    })
    const parts = Object.fromEntries(
      formatter.formatToParts(value)
        .filter(part => part.type !== 'literal')
        .map(part => [part.type, part.value])
    )
    return {
      year: Number(parts.year),
      month: Number(parts.month),
      day: Number(parts.day)
    }
  } catch {
    return null
  }
}

function serialDay(parts: DateParts): number {
  return Math.floor(Date.UTC(parts.year, parts.month - 1, parts.day) / 86400000)
}

function dateKey(parts: DateParts): string {
  return [
    String(parts.year).padStart(4, '0'),
    String(parts.month).padStart(2, '0'),
    String(parts.day).padStart(2, '0')
  ].join('-')
}

function formatDateLabel(parts: DateParts, settings: SiteDateTimeSettings, locale: string): string {
  const date = new Date(Date.UTC(parts.year, parts.month - 1, parts.day, 12, 0, 0))
  try {
    return new Intl.DateTimeFormat(locale || 'zh-CN', {
      timeZone: settings.timezone || 'UTC',
      year: 'numeric',
      month: 'short',
      day: 'numeric'
    }).format(date)
  } catch {
    return dateKey(parts)
  }
}

function formatTimeLabel(value: Date, settings: SiteDateTimeSettings, locale: string): string {
  if (settings.timeFormat === 'hidden') {
    return ''
  }
  try {
    return new Intl.DateTimeFormat(locale || 'zh-CN', {
      timeZone: settings.timezone || 'UTC',
      hour: '2-digit',
      minute: '2-digit',
      second: settings.timeFormat === 'H:i:s' ? '2-digit' : undefined,
      hour12: settings.timeFormat === 'g:i a' || settings.timeFormat === 'g:i A'
    }).format(value)
  } catch {
    return ''
  }
}

export function profileActivityLink(activity: ProfileActivity, topicUrlMode: TopicUrlMode): string {
  // 回复深链必须带 /page/N：URL fragment 不会发给服务器，仅 #comment-id 时 SSR 无法定位跨页评论。
  const page = activity.kind === 'comment' && activity.commentPage && activity.commentPage > 1
    ? activity.commentPage
    : 1
  const base = forumTopicPath({ id: activity.topic.id, slug: activity.topic.slug }, topicUrlMode, page)
  if (activity.kind === 'comment' && activity.commentId && activity.commentId > 0) {
    return `${base}#comment-${activity.commentId}`
  }
  return base
}

export function groupProfileActivitiesByDate(
  activities: ProfileActivity[] | undefined,
  options: GroupOptions
): ProfileActivityGroup[] {
  const now = options.now || new Date()
  const todayParts = dateParts(now, options.settings.timezone, options.locale)
  const todaySerial = todayParts ? serialDay(todayParts) : 0
  const groups = new Map<string, ProfileActivityGroup>()

  for (const activity of activities || []) {
    const createdAt = new Date(activity.createdAt)
    if (Number.isNaN(createdAt.getTime())) {
      continue
    }
    const parts = dateParts(createdAt, options.settings.timezone, options.locale)
    if (!parts) {
      continue
    }
    const key = dateKey(parts)
    const currentSerial = serialDay(parts)
    let group = groups.get(key)
    if (!group) {
      group = {
        key,
        label: currentSerial === todaySerial
          ? options.labels.today
          : currentSerial === todaySerial - 1
            ? options.labels.yesterday
            : formatDateLabel(parts, options.settings, options.locale),
        dateLabel: formatDateLabel(parts, options.settings, options.locale),
        items: []
      }
      groups.set(key, group)
    }
    group.items.push({
      ...activity,
      to: profileActivityLink(activity, options.topicUrlMode),
      timeLabel: formatTimeLabel(createdAt, options.settings, options.locale)
    })
  }

  return [...groups.values()].sort((left, right) => right.key.localeCompare(left.key))
}

export function profileHasPublicDetails(profile: PublicProfile | null | undefined): boolean {
  return Boolean(
    profile?.profile.bio?.trim()
    || profile?.profile.signature?.trim()
    || profile?.profile.location?.trim()
    || profile?.profile.websiteUrl?.trim()
  )
}
