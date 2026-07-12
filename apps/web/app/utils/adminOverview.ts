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
  extensionWidgets?: AdminOverviewExtensionWidget[]
}

export type AdminOverviewExtensionWidget = {
  extensionId: string
  id: string
  order: number
  label?: Record<string, string>
  icon?: string
  route: string
  severity: string
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

/** 趋势三列迷你图使用的字段 */
export type AdminOverviewTrendField = 'topicCount' | 'commentCount' | 'userCount'

export type AdminOverviewTrendSpark = {
  line: string
  area: string
  points: Array<{ x: number, y: number }>
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

/** 单系列最大值，供独立 sparkline 刻度使用 */
export function overviewTrendFieldMax(days: AdminOverviewTrendDay[], field: AdminOverviewTrendField) {
  if (!days.length) {
    return 1
  }
  return Math.max(1, ...days.map(day => Math.max(0, Number(day[field]) || 0)))
}

export function overviewTrendSum(days: AdminOverviewTrendDay[], field: AdminOverviewTrendField) {
  return days.reduce((total, day) => total + Math.max(0, Number(day[field]) || 0), 0)
}

/** 较前一日变化百分比；prev 为 0 且 today > 0 时视为 +100% */
export function overviewTrendDeltaPercent(today: number, previous: number) {
  const current = Math.max(0, Number(today) || 0)
  const prev = Math.max(0, Number(previous) || 0)
  if (prev === 0) {
    return current > 0 ? 100 : 0
  }
  return Math.round(((current - prev) / prev) * 100)
}

export function overviewTrendPeakDate(days: AdminOverviewTrendDay[], field: AdminOverviewTrendField) {
  if (!days.length) {
    return ''
  }
  return days.reduce((best, day) => (
    (Number(day[field]) || 0) >= (Number(best[field]) || 0) ? day : best
  )).date
}

export function overviewTrendDateLabel(date: string) {
  const value = `${date || ''}`.trim()
  if (value.length >= 10) {
    return value.slice(5, 10)
  }
  return value
}

/**
 * 生成平滑 sparkline 的 line / area path 与数据点坐标。
 * 各系列独立 max，避免用户被回复数量级压扁。
 */
export function overviewTrendSparkPath(
  values: number[],
  width = 280,
  height = 72,
  pad = 8
): AdminOverviewTrendSpark {
  const safe = values.map(value => Math.max(0, Number(value) || 0))
  if (safe.length === 0) {
    return { line: '', area: '', points: [] }
  }

  const max = Math.max(1, ...safe)
  const lastIndex = Math.max(safe.length - 1, 1)
  const points = safe.map((value, index) => {
    const x = pad + (index / lastIndex) * (width - pad * 2)
    const y = pad + (1 - value / max) * (height - pad * 2)
    return { x, y }
  })

  let line = `M ${points[0].x} ${points[0].y}`
  for (let index = 1; index < points.length; index += 1) {
    const prev = points[index - 1]
    const curr = points[index]
    const midX = (prev.x + curr.x) / 2
    line += ` C ${midX} ${prev.y}, ${midX} ${curr.y}, ${curr.x} ${curr.y}`
  }

  const bottom = height - 2
  const area = safe.length === 1
    ? `${line} L ${points[0].x} ${bottom} Z`
    : `${line} L ${points[points.length - 1].x} ${bottom} L ${points[0].x} ${bottom} Z`

  return { line, area, points }
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
