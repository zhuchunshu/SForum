export type AdminExtensionType = 'plugin' | 'theme'
export type AdminExtensionStatus = 'installed' | 'enabled' | 'disabled'
export type AdminExtensionSource = 'builtin' | 'uploaded'
export type AdminThemeActionState = 'active' | 'activateDefault' | 'verifyOnly'
export type AdminRuntimeState = 'stopped' | 'starting' | 'running' | 'failed'
export type AdminExtensionEventKind = 'observe' | 'validate' | 'filter'
export type AdminExtensionDeliveryStatus = 'queued' | 'running' | 'succeeded' | 'failed' | 'skipped'

export type AdminExtensionSetting = {
  key: string
  label: string
  type: string
}

export type AdminExtensionManifest = {
  id: string
  name: string
  version: string
  type: AdminExtensionType
  sforumVersion: string
  permissions?: string[]
  settings?: AdminExtensionSetting[]
  migrations?: Array<{ path: string }>
  backend?: { entry?: string, rpc?: string, protocolVersion?: number }
  frontend?: { layer?: string }
  adminPages?: Array<{ path: string, label: string, permission?: string }>
  routes?: Array<{ path: string, methods?: string[], access?: 'public' | 'login' | 'permission', permission?: string, timeoutMs?: number }>
  hooks?: Array<{ name: string }>
  events?: Array<{ name: string, kind?: AdminExtensionEventKind, timeoutMs?: number }>
  jobs?: Array<{ name: string }>
  providers?: Array<{ slot: string, label: string, timeoutMs?: number }>
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
  runtime?: AdminExtensionRuntime
  packagePath: string
  installedAt: string
  updatedAt: string
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
  return 'verifyOnly'
}

export function capabilityCount(item: AdminExtension) {
  const manifest = item.manifest
  return [
    manifest.permissions?.length || 0,
    manifest.settings?.length || 0,
    manifest.migrations?.length || 0,
    manifest.adminPages?.length || 0,
    manifest.routes?.length || 0,
    manifest.hooks?.length || 0,
    manifest.events?.length || 0,
    manifest.jobs?.length || 0,
    manifest.providers?.length || 0
  ].reduce((total, count) => total + count, 0)
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

export function extensionDeliveryPage(items: AdminExtensionEventDelivery[], page: number, pageSize = EXTENSION_EVENT_PAGE_SIZE) {
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

export function extensionSettingDeclarations(items: AdminExtension[]) {
  return items.flatMap((item): AdminExtensionSettingDeclaration[] => (item.manifest.settings || []).map(setting => ({
    extensionId: item.id,
    extensionName: item.name,
    extensionType: item.type,
    setting
  })))
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
