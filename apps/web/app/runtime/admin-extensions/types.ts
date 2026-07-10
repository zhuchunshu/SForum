import type { Component } from 'vue'

export type AdminComponentLoader = () => Promise<{ default: Component }>

export type AdminComponentMetadata = {
  point: string
  extensionId: string
  contributionId: string
  componentId: string
  order: number
  label: Record<string, string>
  options: Record<string, unknown>
}

export type AdminComponentRegistry = Record<string, AdminComponentLoader>
export type AdminExtensionLocaleMessages = Record<string, Record<string, Record<string, unknown>>>

export function assertAdminExtensionRelativePath(path: string) {
  const cleanPath = path.trim()
  const routePath = cleanPath.split(/[?#]/, 1)[0] || ''
  if (
    /^(?:[a-z][a-z\d+.-]*:)?\/\//i.test(cleanPath)
    || cleanPath.includes('\\')
    || /%(?:2e|2f|5c)/i.test(routePath)
    || routePath.split('/').some(segment => segment === '.' || segment === '..')
  ) {
    throw new Error('Admin extension paths must stay inside their host route')
  }
  return cleanPath
}

export function extensionRequestPath(extensionId: string, path: string) {
  const cleanPath = assertAdminExtensionRelativePath(path)
  const suffix = cleanPath ? `/${cleanPath.replace(/^\/+/, '')}` : ''
  return `/extensions/${encodeURIComponent(extensionId)}${suffix}`
}
