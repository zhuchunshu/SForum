export type AdminExtensionType = 'plugin' | 'theme'
export type AdminExtensionStatus = 'installed' | 'enabled' | 'disabled'
export type AdminExtensionSource = 'builtin' | 'uploaded'
export type AdminThemeActionState = 'active' | 'activateDefault' | 'verifyOnly'
export type AdminRuntimeState = 'stopped' | 'starting' | 'running' | 'failed'

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
  jobs?: Array<{ name: string }>
  providers?: Array<{ slot: string, label: string, timeoutMs?: number }>
}

export type AdminExtensionRuntime = {
  state: AdminRuntimeState
  lastError?: string
  startedAt?: string
  routeCount: number
  hookCount: number
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

export type AdminExtensionStats = {
  pluginCount: number
  themeCount: number
  enabledPluginCount: number
  activeThemeId: string
}

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
    hooks: item.runtime?.hookCount ?? item.manifest.hooks?.length ?? 0,
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

export function extensionSettingDeclarations(items: AdminExtension[]) {
  return items.flatMap((item): AdminExtensionSettingDeclaration[] => (item.manifest.settings || []).map(setting => ({
    extensionId: item.id,
    extensionName: item.name,
    extensionType: item.type,
    setting
  })))
}
