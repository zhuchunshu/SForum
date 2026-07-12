export type AdminExtensionType = 'plugin' | 'theme'
export type AdminExtensionStatus = 'installed' | 'enabled' | 'disabled'
export type AdminExtensionSource = 'builtin' | 'uploaded'
export type AdminThemeReleaseStatus = 'queued' | 'building' | 'built' | 'activating' | 'active' | 'failed' | 'rolled_back'
export type AdminThemeActionState = 'active' | 'activateDefault' | 'activate' | 'queued' | 'building' | 'activating' | 'failed'
export type AdminRuntimeState = 'stopped' | 'starting' | 'running' | 'failed'
export type AdminExtensionEventKind = 'observe' | 'validate' | 'filter'
export type AdminExtensionDeliveryStatus = 'queued' | 'running' | 'succeeded' | 'failed' | 'skipped'
export type AdminContributionPayloadType = 'extensionRoute'

export type AdminExtensionSetting = {
  key: string
  label: string
  description?: string
  type: string
  default?: string
  placeholder?: string
  recommendedValue?: string
  group?: string
  options?: Array<{ value: string, label: string, description?: string }>
}

export type AdminExtensionAuthor = {
  name: string
  url?: string
  email?: string
}

// 可选本地化覆盖；字段有值才覆盖顶层默认英文。
export type AdminExtensionLocale = {
  name?: string
  description?: string
  url?: string
  author?: Partial<AdminExtensionAuthor>
}

export type AdminExtensionAdminPage = {
  path: string
  label: string
  description?: string
  icon?: string
  view?: 'about' | 'settings'
  menu?: boolean
  order?: number
  permission?: string
}

export type AdminExtensionAdmin = {
  entry?: string
  pages?: AdminExtensionAdminPage[]
}

export type AdminExtensionManifest = {
  id: string
  name: string
  description: string
  url: string
  author: AdminExtensionAuthor
  version: string
  type: AdminExtensionType
  sforumVersion: string
  // 可选；未声明时直接使用顶层英文文案，无需实现翻译。
  langs?: Record<string, AdminExtensionLocale>
  permissions?: string[]
  /** Host capability keys declared by the plugin (F2.1). */
  capabilities?: string[]
  settings?: AdminExtensionSetting[]
  migrations?: Array<{ path: string }>
  backend?: { entry?: string, rpc?: string, protocolVersion?: number }
  frontend?: {
    layer?: string
    admin?: { root: string, apiVersion: number, components: Record<string, string>, locales: Record<string, string> }
  }
  admin?: AdminExtensionAdmin
  adminPages?: AdminExtensionAdminPage[]
  routes?: Array<{ path: string, methods?: string[], access?: 'public' | 'login' | 'permission', permission?: string, timeoutMs?: number }>
  hooks?: Array<{ name: string }>
  events?: Array<{ name: string, kind?: AdminExtensionEventKind, timeoutMs?: number }>
  jobs?: Array<{ name: string }>
  providers?: Array<{ slot: string, label: string, timeoutMs?: number }>
  contributions?: AdminManifestContribution[]
}

export type AdminManifestContribution = {
  point: string
  id: string
  order?: number
  label?: Record<string, string>
  icon?: string
  payload?: AdminContributionPayload
}

export type AdminContributionPayload = {
  type?: AdminContributionPayloadType | string
  method?: string
  path?: string
  confirm?: boolean
  [key: string]: unknown
}

export type AdminExtensionNavigationItem = {
  extensionId: string
  extensionName: string
  extensionType: AdminExtensionType
  extensionStatus: AdminExtensionStatus
  path: string
  label: string
  description: string
  icon: string
  view: 'about' | 'settings'
  order: number
}

export type AdminExtensionSettingValue = {
  key: string
  label: string
  description?: string
  type: string
  default: string
  value: string
  secretSet?: boolean
  placeholder?: string
  recommendedValue?: string
  group?: string
  options?: Array<{ value: string, label: string, description?: string }>
}

export type AdminExtensionSettings = {
  extensionId: string
  items: AdminExtensionSettingValue[]
}

export function recommendedExtensionSettingValues(items: AdminExtensionSettingValue[]) {
  return Object.fromEntries(items
    .filter(item => item.type !== 'secret')
    .map(item => [item.key, item.recommendedValue ?? item.default]))
}

export type AdminExtensionRuntime = {
  state: AdminRuntimeState
  lastError?: string
  startedAt?: string
  routeCount: number
  hookCount: number
  eventCount?: number
  providerCount: number
}

export type AdminThemeRelease = {
  id: number
  extensionId: string
  extensionVersion: string
  status: AdminThemeReleaseStatus
  message: string
  buildLog?: string
  createdAt: string
  updatedAt: string
  activatedAt?: string
}

export type AdminWebReleaseSummary = {
  id: number
  status: string
  compositionHash: string
  reloadMode: string
  triggerKind?: string
  triggerExtensionId?: string
  publicReason?: string
  publicMessage?: string
  /** 与主题 themeRelease.buildLog 同用途，供行内展开。 */
  buildLog?: string
}

export type AdminCapabilityRisk = 'low' | 'medium' | 'high'

/** 启用审查用的有效 Host 能力（显式 + 宿主推断）。 */
export type AdminCapabilityGrant = {
  key: string
  risk: AdminCapabilityRisk
  labelZh: string
  labelEn: string
  description: string
  implied?: boolean
}

export type AdminExtension = {
  id: string
  name: string
  version: string
  type: AdminExtensionType
  status: AdminExtensionStatus
  source?: AdminExtensionSource
  isSystem?: boolean
  isDeletable?: boolean
  manifest: AdminExtensionManifest
  /** F2.1 有效能力列表，启用前需运营确认。 */
  capabilityGrants?: AdminCapabilityGrant[]
  runtime?: AdminExtensionRuntime
  themeRelease?: AdminThemeRelease
  /** 插件启停/信任变更排队的 Web 发布进度（主题用 themeRelease）。 */
  webRelease?: AdminWebReleaseSummary
  packagePath: string
  installedAt: string
  updatedAt: string
}

export type AdminExtensionOperation = {
  extension: AdminExtension
  frontend?: unknown
  webRelease?: AdminWebReleaseSummary
  queued: boolean
}

export type AdminExtensionEvent = {
  id: number
  extensionId: string
  actorUserId: number
  action: string
  message: string
  createdAt: string
}

export type AdminExtensionEventDefinition = {
  name: string
  kind: AdminExtensionEventKind
  description: string
  payloadFields?: string[]
  patchFields?: string[]
  timeoutMs: number
}

export type AdminExtensionEventDelivery = {
  id: number
  extensionId: string
  eventName: string
  eventKind: AdminExtensionEventKind
  status: AdminExtensionDeliveryStatus
  reason: string
  message: string
  correlationId: string
  attemptCount: number
  createdAt: string
  updatedAt: string
  completedAt?: string
}

export type AdminContributionPointDefinition = {
  id: string
  owner: string
  kind: string
  description: string
  payloadType: string
}

export type AdminEffectiveContribution = {
  extensionId: string
  extensionName: string
  extensionType: AdminExtensionType
  point: string
  id: string
  order: number
  label?: Record<string, string>
  icon?: string
  payload?: AdminContributionPayload
}

export type AdminExtensionStats = {
  pluginCount: number
  themeCount: number
  enabledPluginCount: number
  activeThemeId: string
}

export const EXTENSION_EVENT_PAGE_SIZE = 8
export const EXTENSION_EVENT_FETCH_LIMIT = 100
export const EXTENSION_DELIVERY_FETCH_LIMIT = 100

export type AdminExtensionSettingDeclaration = {
  extensionId: string
  extensionName: string
  extensionType: AdminExtensionType
  setting: AdminExtensionSetting
}

export function filterExtensionsByType(items: AdminExtension[], type: AdminExtensionType) {
  return items.filter(item => item.type === type)
}

export function extensionStats(items: AdminExtension[]): AdminExtensionStats {
  return {
    pluginCount: filterExtensionsByType(items, 'plugin').length,
    themeCount: filterExtensionsByType(items, 'theme').length,
    enabledPluginCount: items.filter(item => item.type === 'plugin' && item.status === 'enabled').length,
    activeThemeId: activeTheme(items)?.id || ''
  }
}

export function activeTheme(items: AdminExtension[]) {
  return items.find(item => item.type === 'theme' && item.status === 'enabled')
}

export function themeStatusLabelKey(item: AdminExtension) {
  return item.type === 'theme' && item.status === 'enabled'
    ? 'admin.extensions.status.activeTheme'
    : `admin.extensions.status.${item.status}`
}

export function themeActionState(item: AdminExtension): AdminThemeActionState {
  if (item.status === 'enabled') {
    return 'active'
  }
  if (item.id === 'sforum.default-theme' && item.source === 'builtin') {
    return 'activateDefault'
  }
  if (item.themeRelease?.status === 'queued') {
    return 'queued'
  }
  if (item.themeRelease?.status === 'building') {
    return 'building'
  }
  if (item.themeRelease?.status === 'activating') {
    return 'activating'
  }
  if (item.themeRelease?.status === 'failed') {
    return 'failed'
  }
  return 'activate'
}

export function themeActivationProgress(release?: AdminThemeRelease | null) {
  if (!release) {
    return null
  }

  switch (release.status) {
    case 'queued':
      return themeProgressState(release.status, 10, 'info', 'i-lucide-hourglass', true)
    case 'building':
      return themeProgressState(release.status, 45, 'warning', 'i-lucide-hammer', true)
    case 'built':
      return themeProgressState(release.status, 70, 'warning', 'i-lucide-package-check', true)
    case 'activating':
      return themeProgressState(release.status, 85, 'warning', 'i-lucide-refresh-cw', true)
    case 'active':
      return themeProgressState(release.status, 100, 'success', 'i-lucide-check-circle-2', false)
    case 'failed':
      return themeProgressState(release.status, 100, 'error', 'i-lucide-triangle-alert', false)
    case 'rolled_back':
      return themeProgressState(release.status, 100, 'neutral', 'i-lucide-undo-2', false)
  }
}

export function hasThemeActivationInProgress(items: AdminExtension[]) {
  return items.some(item => themeActivationProgress(item.themeRelease)?.active)
}

/** 插件 Web Release 进度（与主题 progress 同形，便于共用进度条 UI）。 */
export function pluginWebReleaseProgress(release?: AdminWebReleaseSummary | null) {
  if (!release?.status) {
    return null
  }
  switch (release.status) {
    case 'queued':
      return webReleaseProgressState(release.status, 8, 'info', 'i-lucide-hourglass', true)
    case 'resolving':
      return webReleaseProgressState(release.status, 18, 'info', 'i-lucide-search', true)
    case 'installing':
      return webReleaseProgressState(release.status, 28, 'warning', 'i-lucide-download', true)
    case 'building':
      return webReleaseProgressState(release.status, 48, 'warning', 'i-lucide-hammer', true)
    case 'verifying':
      return webReleaseProgressState(release.status, 62, 'warning', 'i-lucide-shield-check', true)
    case 'ready':
      return webReleaseProgressState(release.status, 75, 'warning', 'i-lucide-package-check', true)
    case 'activating':
      return webReleaseProgressState(release.status, 88, 'warning', 'i-lucide-refresh-cw', true)
    case 'failed':
      return webReleaseProgressState(release.status, 100, 'error', 'i-lucide-triangle-alert', false)
    default:
      // active / inactive 等终态不在插件行上常驻展示。
      return null
  }
}

export function hasPluginWebReleaseInProgress(items: AdminExtension[]) {
  return items.some(item => pluginWebReleaseProgress(item.webRelease)?.active)
}

export function hasExtensionReleaseInProgress(items: AdminExtension[]) {
  return hasThemeActivationInProgress(items) || hasPluginWebReleaseInProgress(items)
}

function themeProgressState(
  status: AdminThemeReleaseStatus,
  percent: number,
  color: 'info' | 'success' | 'warning' | 'error' | 'neutral',
  icon: string,
  active: boolean
) {
  return {
    percent,
    status,
    labelKey: `admin.extensions.themeRelease.${status}`,
    detailKey: `admin.extensions.themeProgress.${status}`,
    icon,
    color,
    active
  }
}

function webReleaseProgressState(
  status: string,
  percent: number,
  color: 'info' | 'success' | 'warning' | 'error' | 'neutral',
  icon: string,
  active: boolean
) {
  return {
    percent,
    status,
    labelKey: `admin.extensions.releases.statusLabels.${status}`,
    detailKey: `admin.extensions.webReleaseProgress.${status}`,
    icon,
    color,
    active
  }
}

export function capabilityCount(item: AdminExtension) {
  const manifest = item.manifest
  return [
    manifest.permissions?.length || 0,
    manifest.settings?.length || 0,
    manifest.migrations?.length || 0,
    effectiveManifestAdminPages(manifest).length,
    manifest.routes?.length || 0,
    manifest.hooks?.length || 0,
    manifest.events?.length || 0,
    manifest.jobs?.length || 0,
    manifest.providers?.length || 0,
    manifest.contributions?.length || 0
  ].reduce((total, count) => total + count, 0)
}

function effectiveManifestAdminPages(manifest: AdminExtensionManifest) {
  return manifest.admin?.pages?.length ? manifest.admin.pages : manifest.adminPages || []
}

export type ExtensionLocalizedDisplay = {
  name: string
  description: string
  url: string
  author: AdminExtensionAuthor
}

function normalizeExtensionLocaleKey(value?: string | null) {
  const raw = `${value || ''}`.trim().replace(/_/g, '-')
  if (!raw) {
    return ''
  }
  const parts = raw.split('-').filter(Boolean)
  const first = parts[0]
  if (!first) {
    return ''
  }
  // 语言主码小写；两位区域码大写（zh-CN），其余段小写。
  parts[0] = first.toLowerCase()
  for (let index = 1; index < parts.length; index += 1) {
    const part = parts[index]
    if (!part) {
      continue
    }
    parts[index] = part.length === 2 ? part.toUpperCase() : part.toLowerCase()
  }
  return parts.join('-')
}

function extensionLocaleLookupCandidates(locale?: string | null) {
  const code = normalizeExtensionLocaleKey(locale)
  if (!code) {
    return [] as string[]
  }
  const candidates = [code]
  const primary = code.split('-')[0]
  if (primary && primary !== code) {
    candidates.push(primary)
  }
  return candidates
}

function lookupExtensionLocale(langs: Record<string, AdminExtensionLocale> | undefined, locale?: string | null) {
  if (!langs) {
    return undefined
  }
  const entries = Object.entries(langs)
  if (!entries.length) {
    return undefined
  }
  // 键做规范化后再匹配，兼容 zh / zh-CN / zh_CN。
  const normalized = new Map(
    entries
      .map(([key, value]) => [normalizeExtensionLocaleKey(key), value] as const)
      .filter(([key]) => Boolean(key))
  )
  for (const candidate of extensionLocaleLookupCandidates(locale)) {
    const match = normalized.get(candidate)
    if (match) {
      return match
    }
  }
  return undefined
}

function resolveLocaleCode(locale?: unknown) {
  // 兼容直接传入 string / Ref / ComputedRef，避免模板误传对象导致匹配失败。
  if (locale == null) {
    return ''
  }
  if (typeof locale === 'string') {
    return locale
  }
  if (typeof locale === 'object' && locale !== null && 'value' in locale) {
    const value = (locale as { value: unknown }).value
    return typeof value === 'string' ? value : `${value ?? ''}`
  }
  return `${locale}`
}

// 按当前 UI locale 解析展示文案；无 langs 或未命中时回退顶层默认英文。
export function extensionLocalizedDisplay(item: AdminExtension, locale?: unknown): ExtensionLocalizedDisplay {
  const manifest = item.manifest
  const display: ExtensionLocalizedDisplay = {
    name: item.name || manifest.name || '',
    description: manifest.description || '',
    url: manifest.url || '',
    author: {
      name: manifest.author?.name || '',
      url: manifest.author?.url,
      email: manifest.author?.email
    }
  }
  const override = lookupExtensionLocale(manifest.langs, resolveLocaleCode(locale))
  if (!override) {
    return display
  }
  if (override.name?.trim()) {
    display.name = override.name.trim()
  }
  if (override.description?.trim()) {
    display.description = override.description.trim()
  }
  if (override.url?.trim()) {
    display.url = override.url.trim()
  }
  if (override.author?.name?.trim()) {
    display.author.name = override.author.name.trim()
  }
  if (override.author?.url?.trim()) {
    display.author.url = override.author.url.trim()
  }
  if (override.author?.email?.trim()) {
    display.author.email = override.author.email.trim()
  }
  return display
}

export function extensionDisplayName(item: AdminExtension, locale?: unknown) {
  return extensionLocalizedDisplay(item, locale).name
}

export function extensionDisplayDescription(item: AdminExtension, locale?: unknown) {
  return extensionLocalizedDisplay(item, locale).description
}

export function extensionAuthorName(item: AdminExtension, locale?: unknown) {
  return extensionLocalizedDisplay(item, locale).author.name?.trim() || ''
}

export function extensionAuthorWebsite(item: AdminExtension, locale?: unknown) {
  const display = extensionLocalizedDisplay(item, locale)
  return display.author.url || display.url || ''
}

export function runtimeStatusLabelKey(item: AdminExtension) {
  return `admin.extensions.runtime.${item.runtime?.state || 'stopped'}`
}

export function runtimeCapabilitySummary(item: AdminExtension) {
  return {
    routes: item.runtime?.routeCount ?? item.manifest.routes?.length ?? 0,
    hooks: item.runtime?.hookCount ?? declaredEventCount(item),
    events: item.runtime?.eventCount ?? declaredEventCount(item),
    providers: item.runtime?.providerCount ?? item.manifest.providers?.length ?? 0
  }
}

export function canRestartPlugin(item: AdminExtension) {
  return item.type === 'plugin' && item.status === 'enabled' && Boolean(item.runtime)
}

export function mergeExtensionEvents(eventsByExtension: Record<string, AdminExtensionEvent[]>) {
  return Object.values(eventsByExtension)
    .flat()
    .slice()
    .sort((left, right) => {
      const timeDiff = Date.parse(right.createdAt) - Date.parse(left.createdAt)
      return timeDiff || right.id - left.id
    })
}

export function mergeExtensionDeliveries(items: AdminExtensionEventDelivery[]) {
  return items.slice().sort((left, right) => {
    const timeDiff = Date.parse(right.createdAt) - Date.parse(left.createdAt)
    return timeDiff || right.id - left.id
  })
}

export function extensionEventPage(items: AdminExtensionEvent[], page: number, pageSize = EXTENSION_EVENT_PAGE_SIZE) {
  return extensionItemsPage(items, page, pageSize)
}

export function extensionDeliveryPage(items: AdminExtensionEventDelivery[], page: number, pageSize = EXTENSION_EVENT_PAGE_SIZE) {
  return extensionItemsPage(items, page, pageSize)
}

export function extensionDefinitionPage(items: AdminExtensionEventDefinition[], page: number, pageSize = EXTENSION_EVENT_PAGE_SIZE) {
  return extensionItemsPage(items, page, pageSize)
}

export function extensionContributionPage(items: AdminEffectiveContribution[], page: number, pageSize = EXTENSION_EVENT_PAGE_SIZE) {
  return extensionItemsPage(items, page, pageSize)
}

export function extensionContributionLabel(item: AdminEffectiveContribution, locale = 'zh-CN') {
  const labels = item.label || {}
  return labels[locale] || labels['zh-CN'] || labels['en-US'] || Object.values(labels).find(Boolean) || item.id
}

export function extensionContributionPayloadSummary(item: AdminEffectiveContribution) {
  const payload = item.payload || {}
  const parts = [payload.type, payload.method, payload.path]
    .map(value => `${value || ''}`.trim())
    .filter(Boolean)
  return parts.length ? parts.join(' ') : '-'
}

function extensionItemsPage<T>(items: T[], page: number, pageSize = EXTENSION_EVENT_PAGE_SIZE) {
  const safePageSize = Math.max(1, Math.floor(pageSize))
  const totalPages = Math.max(1, Math.ceil(items.length / safePageSize))
  const currentPage = Math.min(totalPages, Math.max(1, Math.floor(page) || 1))
  const start = items.length === 0 ? 0 : (currentPage - 1) * safePageSize + 1
  const end = items.length === 0 ? 0 : Math.min(items.length, currentPage * safePageSize)

  return {
    items: items.slice(start === 0 ? 0 : start - 1, end),
    page: currentPage,
    pageSize: safePageSize,
    totalPages,
    start,
    end,
    total: items.length
  }
}

export function extensionSettingDeclarations(items: AdminExtension[], locale?: string | null) {
  return items.flatMap((item): AdminExtensionSettingDeclaration[] => (item.manifest.settings || []).map(setting => ({
    extensionId: item.id,
    extensionName: extensionDisplayName(item, locale),
    extensionType: item.type,
    setting
  })))
}

export function defaultExtensionIcon(type: AdminExtensionType) {
  return type === 'theme' ? 'i-lucide-palette' : 'i-lucide-plug'
}

export function extensionAdminPages(item: AdminExtension, locale?: string | null): AdminExtensionAdminPage[] {
  const display = extensionLocalizedDisplay(item, locale)
  const pages: AdminExtensionAdminPage[] = [
    {
      path: '/about',
      label: display.name,
      description: display.description,
      icon: defaultExtensionIcon(item.type),
      view: 'about',
      order: 0
    }
  ]

  for (const page of effectiveManifestAdminPages(item.manifest)) {
    pages.push({
      path: normalizeExtensionPagePath(page.path),
      label: page.label,
      description: page.description || display.description,
      icon: page.icon || defaultExtensionIcon(item.type),
      view: page.view || 'about',
      menu: page.menu === true,
      order: page.order || 0,
      permission: page.permission
    })
  }

  return pages.sort((left, right) => {
    const orderDiff = (left.order || 0) - (right.order || 0)
    return orderDiff || left.path.localeCompare(right.path)
  })
}

export function findExtensionAdminPage(item: AdminExtension, path: string, locale?: string | null) {
  const normalized = normalizeExtensionPagePath(path)
  return extensionAdminPages(item, locale).find(page => normalizeExtensionPagePath(page.path) === normalized)
}

export function extensionManagePagePath(item: AdminExtension, locale?: string | null) {
  const pages = extensionAdminPages(item, locale)
  const entry = normalizeExtensionPagePath(item.manifest.admin?.entry)
  if (item.manifest.admin?.entry && pages.some(page => normalizeExtensionPagePath(page.path) === entry)) {
    return entry
  }

  const settings = pages.find(page => normalizeExtensionPagePath(page.path) === '/settings')
  if (settings) {
    return '/settings'
  }

  const declared = pages.find(page => normalizeExtensionPagePath(page.path) !== '/about')
  return declared?.path || '/about'
}

export function extensionManageRoute(item: AdminExtension) {
  return extensionAdminPageRoute(item.id, extensionManagePagePath(item))
}

export function extensionAdminPageRoute(extensionId: string, pagePath = '/about') {
  return `/extensions/${extensionId}/pages${normalizeExtensionPagePath(pagePath)}`
}

export function normalizeExtensionPagePath(path?: string | string[] | null) {
  const raw = Array.isArray(path) ? path.join('/') : `${path || ''}`
  const normalized = raw
    .trim()
    .replace(/\\/g, '/')
    .replace(/^\/+|\/+$/g, '')

  if (!normalized) {
    return '/about'
  }

  return `/${normalized.split('/').filter(Boolean).join('/')}`
}

export function declaredEventCount(item: AdminExtension) {
  const names = new Set<string>()
  for (const event of item.manifest.events || []) {
    names.add(`${event.name}:${event.kind || ''}`)
  }
  for (const hook of item.manifest.hooks || []) {
    names.add(`${hook.name}:hook`)
  }
  return names.size
}
