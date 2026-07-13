export type AdminExtensionType = 'plugin' | 'theme'
export type AdminExtensionStatus = 'installed' | 'enabled' | 'disabled'
export type AdminExtensionSource = 'builtin' | 'uploaded'
export type AdminThemeActionState = 'active' | 'activateDefault' | 'activate'
export type AdminRuntimeState = 'stopped' | 'starting' | 'running' | 'degraded' | 'failed'
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
  /** Schema UI control width: default (capped) or full (fill column). */
  width?: 'default' | 'full' | string
  group?: string
  groupId?: string
  column?: number
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

export type AdminManifestSettingsDocument = {
  schemaVersion: number
  ui: {
    mode: 'schema' | 'component'
    layout: 'form' | 'tabs'
    component?: { id: string, apiVersion: number, entry?: string, css?: string }
  }
  fields: AdminExtensionSetting[]
  actions?: Array<{ id: string, kind: string }>
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
  settings?: AdminExtensionSetting[] | AdminManifestSettingsDocument
  migrations?: Array<{ path: string }>
  backend?: { entry?: string, rpc?: string, protocolVersion?: number }
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
  /** Schema UI control width: default (capped) or full (fill column). */
  width?: 'default' | 'full' | string
  group?: string
  groupId?: string
  column?: number
  options?: Array<{ value: string, label: string, description?: string }>
}

export type AdminExtensionSettingsRenderer = {
  mode: 'schema' | 'component'
  layout: 'form' | 'tabs'
	source: 'document' | 'legacy_array'
  fallback: 'schema'
  component?: {
    id: string
		kind: 'prebuilt'
    apiVersion: number
    entry?: string
    css?: string
  }
}

export type AdminExtensionSettingsTab = {
  id: string
  label: string
  description?: string
  groups?: string[]
}

export type AdminExtensionSettingsGroup = {
  id: string
  label: string
  description?: string
  columns?: number
}

export type AdminExtensionSettingsCallout = {
  id: string
  tone: string
  title: string
  body?: string
  tab?: string
  group?: string
}

export type AdminExtensionSettingsAction = {
  id: string
  kind: 'provider_probe'
  label: string
  description?: string
  placement: 'header' | 'footer'
  useDraftValues: boolean
  fields?: string[]
  available: boolean
  unavailableReason?: string
}

export type AdminExtensionSettingsActionResult = {
  success: boolean
  reason: string
  message: string
  details?: Record<string, string>
  suggestions?: string[]
  durationMs: number
}

export type AdminExtensionSettings = {
  extensionId: string
  extensionType: AdminExtensionType
  extensionVersion: string
  extensionStatus: AdminExtensionStatus
  renderer: AdminExtensionSettingsRenderer
  tabs?: AdminExtensionSettingsTab[]
  groups?: AdminExtensionSettingsGroup[]
  callouts?: AdminExtensionSettingsCallout[]
  items: AdminExtensionSettingValue[]
  actions?: AdminExtensionSettingsAction[]
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
  /** F2.3 resilience observability */
  circuitOpen?: boolean
  circuitOpenUntil?: string
  consecutiveFailures?: number
  lastFailureReason?: string
  lastFailureAt?: string
  activeRpcCalls?: number
  maxConcurrentRpc?: number
  /** P3 protocol migration telemetry */
  protocolVersion?: 1 | 2
  protocolTransport?: 'net/rpc' | 'grpc'
  protocolDeprecated?: boolean
  protocolStartCount?: number
  protocolCallCount?: number
  protocolLastCallAt?: string
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

export type AdminExtensionVersion = {
  version: string
  manifest: AdminExtensionManifest
  packageDigest: string
  adminFrontendDigest: string
  packagePath: string
  installedAt: string
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
  packageDigest: string
  adminFrontendDigest?: string
  packagePath: string
  stagedVersion?: AdminExtensionVersion
  installedAt: string
  updatedAt: string
}

export type AdminExtensionSettingsProfile = 'none' | 'schema' | 'actions' | 'prebuilt'

export function extensionSettingsProfile(extension: AdminExtension): AdminExtensionSettingsProfile {
  const settings = extension.manifest.settings
  if (settings && !Array.isArray(settings)) {
    if (settings.ui?.mode === 'component' && settings.ui.component?.entry) return 'prebuilt'
    if ((settings.actions?.length ?? 0) > 0) return 'actions'
    return 'schema'
  }
  if (Array.isArray(settings) && settings.length > 0) return 'schema'
  return 'none'
}

const extensionSettingsProfileMeta = {
  none: { color: 'neutral', icon: 'i-lucide-minus', labelKey: 'admin.extensions.settingsProfiles.none' },
  schema: { color: 'primary', icon: 'i-lucide-layout-template', labelKey: 'admin.extensions.settingsProfiles.schema' },
  actions: { color: 'info', icon: 'i-lucide-activity', labelKey: 'admin.extensions.settingsProfiles.actions' },
	prebuilt: { color: 'warning', icon: 'i-lucide-shield-alert', labelKey: 'admin.extensions.settingsProfiles.prebuilt' }
} as const

export function extensionSettingsPresentation(extension: AdminExtension) {
  const profile = extensionSettingsProfile(extension)
  return { profile, ...extensionSettingsProfileMeta[profile] }
}

function manifestSettingFields(manifest: AdminExtensionManifest) {
  const settings = manifest.settings
  return Array.isArray(settings) ? settings : settings?.fields || []
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
	return 'activate'
}

export function capabilityCount(item: AdminExtension) {
  const manifest = item.manifest
  return [
    manifest.permissions?.length || 0,
    manifestSettingFields(manifest).length,
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

/** 通用列表分页（页面注册表等管理列表复用）。 */
export function paginateItems<T>(items: T[], page: number, pageSize = EXTENSION_EVENT_PAGE_SIZE) {
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
  return items.flatMap((item): AdminExtensionSettingDeclaration[] => manifestSettingFields(item.manifest).map(setting => ({
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
