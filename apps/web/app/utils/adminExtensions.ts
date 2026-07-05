export type AdminExtensionType = 'plugin' | 'theme'
export type AdminExtensionStatus = 'installed' | 'enabled' | 'disabled'

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
  backend?: { entry?: string, rpc?: string }
  frontend?: { layer?: string }
  adminPages?: Array<{ path: string, label: string, permission?: string }>
  routes?: Array<{ path: string, methods?: string[] }>
  hooks?: Array<{ name: string }>
  jobs?: Array<{ name: string }>
}

export type AdminExtension = {
  id: string
  name: string
  version: string
  type: AdminExtensionType
  status: AdminExtensionStatus
  manifest: AdminExtensionManifest
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
  enabledCount: number
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
    enabledCount: items.filter(item => item.status === 'enabled').length
  }
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
    manifest.jobs?.length || 0
  ].reduce((total, count) => total + count, 0)
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
