import { adminSurfacePlacements, type AdminSurfacePlacement } from '~/config/adminSurfaceCatalog.gen'

export const adminSurfaceKinds = [
  'navigation',
  'dashboard',
  'list_column',
  'list_filter',
  'row_action',
  'bulk_action',
  'form',
  'notice',
  'editor_panel',
  'detail_region',
  'importer',
  'exporter'
] as const

export type AdminSurfaceKind = typeof adminSurfaceKinds[number]
export type AdminSurfaceAction = 'add' | 'before' | 'after' | 'wrap' | 'replace' | 'hide' | 'filter'
export type AdminSurfaceOperation = 'query' | 'command'
export type AdminSurfaceTone = 'neutral' | 'primary' | 'success' | 'warning' | 'error'
export type AdminSurfacePrimitive = string | number | boolean | null

export type AdminSurfaceContract = {
  id: string
  contractVersion: string
  extensionId: string
  extensionVersion: string
  kind: AdminSurfaceKind
  action: AdminSurfaceAction
  targetId?: string
  placementId?: string
  placementContractVersion?: string
  label: string
  propsSchema?: string
  propsSchemaDigest?: string
  resultSchema?: string
  resultSchemaDigest?: string
  operation: AdminSurfaceOperation
  schema?: string
  schemaDigest?: string
  priority: number
  invokable: boolean
}

export type AdminSurfaceCatalog = {
  revision: number
  surfaces: AdminSurfaceContract[]
}

export type AdminSurfaceInvocation = {
  surface: AdminSurfaceContract
  output: Record<string, unknown>
}

export type AdminSurfaceOption = {
  label: string
  value: string
}

export type AdminSurfaceField = {
  key: string
  label: string
  type: 'text' | 'textarea' | 'number' | 'boolean' | 'select'
  required: boolean
  placeholder?: string
  options: AdminSurfaceOption[]
}

export type AdminSurfaceViewModel = {
  title?: string
  description?: string
  message?: string
  value?: AdminSurfacePrimitive
  tone: AdminSurfaceTone
  icon?: string
  pageId?: string
  commandSurfaceId?: string
  items: Array<{ label: string, value: AdminSurfacePrimitive }>
  cells: Record<string, AdminSurfacePrimitive>
  options: AdminSurfaceOption[]
  visibleResourceIds: string[]
  visibleResourceIdsDeclared: boolean
  fields: AdminSurfaceField[]
  values: Record<string, AdminSurfacePrimitive>
  refresh: boolean
  download?: { url: string, filename?: string }
}

const iconPattern = /^i-(?:lucide|tabler)-[a-z0-9]+(?:-[a-z0-9]+)*$/
const fieldKeyPattern = /^[a-z0-9][a-z0-9._-]{0,79}$/
const surfaceIdPattern = /^[a-z0-9][a-z0-9._-]{1,160}$/
const tones = new Set<AdminSurfaceTone>(['neutral', 'primary', 'success', 'warning', 'error'])
const fieldTypes = new Set<AdminSurfaceField['type']>(['text', 'textarea', 'number', 'boolean', 'select'])

export function resolveAdminSurfacePlacement(pageId: string): AdminSurfacePlacement | undefined {
  const normalized = normalizeAdminSurfaceRoute(pageId)
  const exact = adminSurfacePlacements.find(item => item.route === normalized)
  if (exact) return exact

  return adminSurfacePlacements.find(item => routePatternMatches(item.route, normalized))
}

export function adminSurfacePlacementPageId(placementId?: string) {
  const placement = adminSurfacePlacements.find(item => item.id === placementId)
  if (!placement || placement.route.includes(':')) return undefined
  return placement.route === '/admin' ? '/' : placement.route.replace(/^\/admin/, '') || '/'
}

export function normalizeAdminSurfaceOutput(output: unknown): AdminSurfaceViewModel | null {
  if (!isRecord(output)) return null
  const tone = typeof output.tone === 'string' && tones.has(output.tone as AdminSurfaceTone)
    ? output.tone as AdminSurfaceTone
    : 'neutral'
  const icon = boundedString(output.icon, 80)
  const pageId = safeAdminPageId(output.pageId)
  const result: AdminSurfaceViewModel = {
    title: boundedString(output.title, 160),
    description: boundedString(output.description, 1000),
    message: boundedString(output.message, 1000),
    value: primitive(output.value),
    tone,
    icon: iconPattern.test(icon || '') ? icon : undefined,
    pageId,
    commandSurfaceId: safeSurfaceId(output.commandSurfaceId),
    items: normalizeItems(output.items),
    cells: normalizePrimitiveRecord(output.cells, 1000),
    options: normalizeOptions(output.options),
    visibleResourceIds: normalizeResourceIds(output.visibleResourceIds),
    visibleResourceIdsDeclared: Array.isArray(output.visibleResourceIds),
    fields: normalizeFields(output.fields),
    values: normalizePrimitiveRecord(output.values, 100),
    refresh: output.refresh === true,
    download: normalizeDownload(output.download)
  }
  if (result.title === undefined && result.description === undefined && result.message === undefined && result.value === undefined &&
    result.icon === undefined && result.pageId === undefined && result.commandSurfaceId === undefined &&
    result.items.length === 0 && Object.keys(result.cells).length === 0 &&
    result.options.length === 0 && !result.visibleResourceIdsDeclared && result.fields.length === 0 &&
    Object.keys(result.values).length === 0 && !result.refresh && result.download === undefined) {
    return null
  }
  return result
}

export function adminSurfaceIdempotencyKey(surfaceId: string) {
  const prefix = surfaceId.replace(/[^a-zA-Z0-9._-]/g, '').slice(0, 48) || 'admin-surface'
  const cryptoValue = globalThis.crypto?.randomUUID?.()
  if (cryptoValue) return `${prefix}:${cryptoValue}`.slice(0, 128)
  return `${prefix}:${Date.now().toString(36)}:${Math.random().toString(36).slice(2)}`.slice(0, 128)
}

export function adminSurfaceKindIcon(kind: AdminSurfaceKind) {
  const icons: Record<AdminSurfaceKind, string> = {
    navigation: 'i-lucide-panel-left',
    dashboard: 'i-lucide-layout-dashboard',
    list_column: 'i-lucide-columns-3',
    list_filter: 'i-lucide-list-filter',
    row_action: 'i-lucide-mouse-pointer-click',
    bulk_action: 'i-lucide-list-checks',
    form: 'i-lucide-notebook-pen',
    notice: 'i-lucide-info',
    editor_panel: 'i-lucide-panel-right',
    detail_region: 'i-lucide-rows-3',
    importer: 'i-lucide-file-up',
    exporter: 'i-lucide-file-down'
  }
  return icons[kind]
}

function normalizeAdminSurfaceRoute(pageId: string) {
  const path = String(pageId || '').trim().split(/[?#]/, 1)[0] || '/'
  if (path === '/admin' || path.startsWith('/admin/')) return path.replace(/\/$/, '') || '/admin'
  const suffix = path === '/' ? '' : `/${path.replace(/^\/+|\/+$/g, '')}`
  return `/admin${suffix}`
}

function routePatternMatches(pattern: string, route: string) {
  if (!pattern.includes(':')) return false
  const expression = pattern
    .split('/')
    .map((segment) => {
      if (segment.startsWith(':') && segment.endsWith('*')) return '.+'
      if (segment.startsWith(':')) return '[^/]+'
      return segment.replace(/[.*+?^${}()|[\]\\]/g, '\\$&')
    })
    .join('/')
  return new RegExp(`^${expression}$`).test(route)
}

function isRecord(value: unknown): value is Record<string, unknown> {
  return Boolean(value) && typeof value === 'object' && !Array.isArray(value)
}

function boundedString(value: unknown, maximum: number) {
  if (typeof value !== 'string') return undefined
  const normalized = value.trim()
  return normalized && normalized.length <= maximum ? normalized : undefined
}

function primitive(value: unknown): AdminSurfacePrimitive | undefined {
  if (value === null || typeof value === 'boolean') return value
  if (typeof value === 'number' && Number.isFinite(value)) return value
  return boundedString(value, 1000)
}

function safeAdminPageId(value: unknown) {
  const pageId = boundedString(value, 512)
  if (!pageId || !pageId.startsWith('/') || pageId.startsWith('//') || pageId.includes('\\') || /[\u0000-\u001f]/.test(pageId)) {
    return undefined
  }
  return pageId
}

function safeSurfaceId(value: unknown) {
  const id = boundedString(value, 160)
  return id && surfaceIdPattern.test(id) ? id : undefined
}

function normalizeItems(value: unknown) {
  if (!Array.isArray(value)) return []
  return value.slice(0, 100).flatMap((item) => {
    if (!isRecord(item)) return []
    const label = boundedString(item.label, 160)
    const itemValue = primitive(item.value)
    return label && itemValue !== undefined ? [{ label, value: itemValue }] : []
  })
}

function normalizePrimitiveRecord(value: unknown, maximum: number) {
  if (!isRecord(value)) return {}
  const result: Record<string, AdminSurfacePrimitive> = {}
  for (const [key, item] of Object.entries(value).slice(0, maximum)) {
    const normalized = primitive(item)
    if (fieldKeyPattern.test(key) && normalized !== undefined) result[key] = normalized
  }
  return result
}

function normalizeOptions(value: unknown): AdminSurfaceOption[] {
  if (!Array.isArray(value)) return []
  return value.slice(0, 100).flatMap((item) => {
    if (!isRecord(item)) return []
    const label = boundedString(item.label, 160)
    const optionValue = boundedString(item.value, 160)
    return label && optionValue ? [{ label, value: optionValue }] : []
  })
}

function normalizeResourceIds(value: unknown) {
  if (!Array.isArray(value)) return []
  return [...new Set(value.slice(0, 1000).flatMap(item => boundedString(item, 160) || []))]
}

function normalizeFields(value: unknown): AdminSurfaceField[] {
  if (!Array.isArray(value)) return []
  const seen = new Set<string>()
  return value.slice(0, 50).flatMap((item) => {
    if (!isRecord(item)) return []
    const key = boundedString(item.key, 80)
    const label = boundedString(item.label, 160)
    const type = boundedString(item.type, 20) as AdminSurfaceField['type'] | undefined
    if (!key || !fieldKeyPattern.test(key) || seen.has(key) || !label || !type || !fieldTypes.has(type)) return []
    seen.add(key)
    return [{
      key,
      label,
      type,
      required: item.required === true,
      placeholder: boundedString(item.placeholder, 240),
      options: type === 'select' ? normalizeOptions(item.options) : []
    }]
  })
}

function normalizeDownload(value: unknown) {
  if (!isRecord(value)) return undefined
  const url = boundedString(value.url, 2048)
  if (!url || !url.startsWith('/') || url.startsWith('//') || url.includes('\\') || /[\u0000-\u001f]/.test(url)) return undefined
  return { url, filename: boundedString(value.filename, 255) }
}
