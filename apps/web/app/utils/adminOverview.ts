export type AdminOverview = {
  generatedAt: string
  windowDays: number
  runtime: AdminOverviewRuntime
  community: AdminOverviewCommunity
  attachments: AdminOverviewAttachments
  moderation: AdminOverviewModeration
  extensions: AdminOverviewExtensions
  trends: {
    days: AdminOverviewTrendDay[]
  }
  topCategories: AdminOverviewCategoryActivity[]
  actions: AdminOverviewAction[]
}

export type AdminOverviewWorkerRuntime = {
  lastSeenAt?: string
  ageSeconds?: number
  stale: boolean
  status: 'ok' | 'stale' | 'unknown'
}

export type AdminOverviewQueueLag = {
  waiting: number
  running: number
  failed: number
}

export type AdminOverviewRuntime = {
  startedAt: string
  uptimeSeconds: number
  memoryBytes: number
  heapAllocBytes: number
  heapSysBytes: number
  goroutineCount: number
  gcCount: number
  lastGcPauseNs: number
  database: {
    maxConnections: number
    totalConnections: number
    acquiredConnections: number
    idleConnections: number
  }
  worker?: AdminOverviewWorkerRuntime
  queueLag?: AdminOverviewQueueLag
}

export type AdminOverviewCommunity = {
  userCount: number
  activeUserCount: number
  disabledUserCount: number
  bannedUserCount: number
  topicCount: number
  activeTopicCount: number
  lockedTopicCount: number
  hiddenTopicCount: number
  deletedTopicCount: number
  commentCount: number
  postCount: number
  categoryCount: number
  tagCount: number
  pendingTagCount: number
  totalViews: number
}

export type AdminOverviewAttachments = {
  totalCount: number
  activeCount: number
  disabledCount: number
  deletedCount: number
  orphanCount: number
  totalBytes: number
}

export type AdminOverviewModeration = {
  openCount: number
  reviewingCount: number
  resolvedCount: number
  rejectedCount: number
}

export type AdminOverviewExtensions = {
  totalCount: number
  enabledCount: number
  pluginCount: number
  themeCount: number
  installedPluginRuntimeCount: number
  failedEventCount: number
  pendingThemeReleaseCount: number
  failedThemeReleaseCount: number
  activeThemeReleaseCount: number
}

export type AdminOverviewTrendDay = {
  date: string
  topicCount: number
  commentCount: number
  userCount: number
}

export type AdminOverviewCategoryActivity = {
  id: number
  slug: string
  name: string
  topicCount: number
  commentCount: number
}

export type AdminOverviewAction = {
  key: string
  count: number
  severity: string
  route: string
}

export type OverviewTone = 'success' | 'info' | 'warning' | 'danger' | 'neutral'

export function formatOverviewBytes(bytes: number) {
  const value = Math.max(0, Number(bytes) || 0)
  const mib = Math.round(value / 1024 / 1024)
  return `${formatInteger(mib)} MiB`
}

export function formatOverviewCount(value: number) {
  const count = Math.max(0, Number(value) || 0)
  if (count >= 1_000_000) {
    return `${trimOneDecimal(count / 1_000_000)}m`
  }
  if (count >= 1_000) {
    return `${trimOneDecimal(count / 1_000)}k`
  }
  return formatInteger(count)
}

export function overviewPercent(value: number, total: number) {
  const normalizedTotal = Number(total) || 0
  if (normalizedTotal <= 0) {
    return 0
  }
  return Math.round((Math.max(0, Number(value) || 0) / normalizedTotal) * 100)
}

export function overviewTrendMax(days: AdminOverviewTrendDay[]) {
  const max = days.reduce((current, day) => {
    return Math.max(current, day.topicCount + day.commentCount + day.userCount)
  }, 0)
  return Math.max(max, 1)
}

export function overviewActionTone(severity: string): OverviewTone {
  switch (severity) {
    case 'success':
    case 'info':
    case 'warning':
    case 'danger':
      return severity
    default:
      return 'neutral'
  }
}

export function formatOverviewUptime(seconds: number) {
  const total = Math.max(0, Math.floor(Number(seconds) || 0))
  const days = Math.floor(total / 86400)
  const hours = Math.floor((total % 86400) / 3600)
  const minutes = Math.floor((total % 3600) / 60)
  if (days > 0) {
    return `${days}d ${hours}h`
  }
  if (hours > 0) {
    return `${hours}h ${minutes}m`
  }
  return `${minutes}m`
}

export function formatOverviewDate(value: string) {
  const date = new Date(value)
  if (Number.isNaN(date.getTime())) {
    return ''
  }
  const year = date.getFullYear()
  const month = padDatePart(date.getMonth() + 1)
  const day = padDatePart(date.getDate())
  const hours = padDatePart(date.getHours())
  const minutes = padDatePart(date.getMinutes())
  const seconds = padDatePart(date.getSeconds())
  return `${year}-${month}-${day} ${hours}:${minutes}:${seconds}`
}

function trimOneDecimal(value: number) {
  return value.toFixed(1).replace(/\.0$/, '')
}

function formatInteger(value: number) {
  return Math.round(value).toString().replace(/\B(?=(\d{3})+(?!\d))/g, ',')
}

function padDatePart(value: number) {
  return value.toString().padStart(2, '0')
}
