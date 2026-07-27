/**
 * Admin consumer for Host public registry catalogs + entity import/export dry-run.
 * Catalogs are declaration projections only; dry-run never executes store I/O.
 */

const CONTENT_CATALOG_SCHEMA = 'sforum.content-catalog@1'
const ENTITY_CATALOG_SCHEMA = 'sforum.entity-catalog@1'
const MEDIA_CATALOG_SCHEMA = 'sforum.media-catalog@1'
const DRY_RUN_SCHEMA = 'sforum.entity-import-export-dry-run@1'

export type RegistryCatalogSummary = {
  schemaVersion: string
  revision: number
  digest: string
  safeMode: boolean
  entryCount: number
}

export type ContentCatalogView = RegistryCatalogSummary & {
  content: Array<{
    id: string
    kind: string
    extensionId: string
    packageDigest: string
    core?: boolean
  }>
}

export type EntityCatalogView = RegistryCatalogSummary & {
  entities: Array<{
    id: string
    kind: string
    extensionId: string
    packageDigest: string
    importExportPolicy?: string
    canImport?: boolean
    canExport?: boolean
    core?: boolean
  }>
}

export type MediaCatalogView = RegistryCatalogSummary & {
  policies: Array<{ id: string; purpose: string; extensionId: string }>
  processors: Array<{ id: string; stage: string; extensionId: string }>
  variants: Array<{ id: string; name: string; extensionId: string; bindingStatus?: string }>
}

export type EntityImportExportDryRunView = {
  schemaVersion: string
  executes: boolean
  entityId: string
  action: string
  allowed: boolean
  permissionKey?: string
  reason?: string
  plan?: {
    policy?: string
    canImport?: boolean
    canExport?: boolean
  }
}

function asRecord(value: unknown): Record<string, unknown> | null {
  return value && typeof value === 'object' && !Array.isArray(value)
    ? (value as Record<string, unknown>)
    : null
}

function asString(value: unknown): string {
  return typeof value === 'string' ? value : ''
}

function asNumber(value: unknown): number {
  return typeof value === 'number' && Number.isFinite(value) ? value : 0
}

function asBoolean(value: unknown): boolean {
  return value === true
}

function asArray(value: unknown): unknown[] {
  return Array.isArray(value) ? value : []
}

export function parseContentCatalog(raw: unknown): ContentCatalogView | null {
  const root = asRecord(raw)
  if (!root || asString(root.schemaVersion) !== CONTENT_CATALOG_SCHEMA) return null
  const content = asArray(root.content).map((item) => {
    const row = asRecord(item) || {}
    return {
      id: asString(row.id),
      kind: asString(row.kind),
      extensionId: asString(row.extensionId),
      packageDigest: asString(row.packageDigest),
      core: asBoolean(row.core) || undefined
    }
  }).filter(row => row.id)
  return {
    schemaVersion: CONTENT_CATALOG_SCHEMA,
    revision: asNumber(root.revision),
    digest: asString(root.digest),
    safeMode: asBoolean(root.safeMode),
    entryCount: content.length,
    content
  }
}

export function parseEntityCatalog(raw: unknown): EntityCatalogView | null {
  const root = asRecord(raw)
  if (!root || asString(root.schemaVersion) !== ENTITY_CATALOG_SCHEMA) return null
  // entity catalog may project as `entities` list of kind entity/taxonomy/field
  const entities = asArray(root.entities ?? root.content).map((item) => {
    const row = asRecord(item) || {}
    const importExport = asRecord(row.importExport) || {}
    return {
      id: asString(row.id),
      kind: asString(row.kind),
      extensionId: asString(row.extensionId),
      packageDigest: asString(row.packageDigest),
      importExportPolicy: asString(row.importExportPolicy) || asString(importExport.policy) || undefined,
      canImport: typeof importExport.canImport === 'boolean' ? importExport.canImport : undefined,
      canExport: typeof importExport.canExport === 'boolean' ? importExport.canExport : undefined,
      core: asBoolean(row.core) || undefined
    }
  }).filter(row => row.id)
  return {
    schemaVersion: ENTITY_CATALOG_SCHEMA,
    revision: asNumber(root.revision),
    digest: asString(root.digest),
    safeMode: asBoolean(root.safeMode),
    entryCount: entities.length,
    entities
  }
}

export function parseMediaCatalog(raw: unknown): MediaCatalogView | null {
  const root = asRecord(raw)
  if (!root || asString(root.schemaVersion) !== MEDIA_CATALOG_SCHEMA) return null
  const policies = asArray(root.policies).map((item) => {
    const row = asRecord(item) || {}
    return {
      id: asString(row.id),
      purpose: asString(row.purpose),
      extensionId: asString(row.extensionId)
    }
  }).filter(row => row.id)
  const processors = asArray(root.processors).map((item) => {
    const row = asRecord(item) || {}
    return {
      id: asString(row.id),
      stage: asString(row.stage),
      extensionId: asString(row.extensionId)
    }
  }).filter(row => row.id)
  const variants = asArray(root.variants).map((item) => {
    const row = asRecord(item) || {}
    return {
      id: asString(row.id),
      name: asString(row.name),
      extensionId: asString(row.extensionId),
      bindingStatus: asString(row.bindingStatus) || undefined
    }
  }).filter(row => row.id)
  return {
    schemaVersion: MEDIA_CATALOG_SCHEMA,
    revision: asNumber(root.revision),
    digest: asString(root.digest),
    safeMode: asBoolean(root.safeMode),
    entryCount: policies.length + processors.length + variants.length,
    policies,
    processors,
    variants
  }
}

export function parseEntityImportExportDryRun(raw: unknown): EntityImportExportDryRunView | null {
  const root = asRecord(raw)
  if (!root) return null
  const schema = asString(root.schemaVersion)
  if (schema && schema !== DRY_RUN_SCHEMA) return null
  const plan = asRecord(root.plan) || {}
  const decision = asRecord(root.decision) || {}
  return {
    schemaVersion: schema || DRY_RUN_SCHEMA,
    executes: asBoolean(root.executes),
    entityId: asString(root.entityId) || asString(plan.entityId),
    action: asString(root.action),
    allowed: asBoolean(decision.allowed) || asBoolean(root.allowed),
    permissionKey: asString(decision.permissionKey) || asString(root.permissionKey) || undefined,
    reason: asString(decision.reason) || asString(root.reason) || undefined,
    plan: {
      policy: asString(plan.policy) || undefined,
      canImport: typeof plan.canImport === 'boolean' ? plan.canImport : undefined,
      canExport: typeof plan.canExport === 'boolean' ? plan.canExport : undefined
    }
  }
}

export function useAdminRegistryCatalogs() {
  const { request } = useApiClient()

  async function loadContentCatalog() {
    const raw = await request<unknown>('/extensions/runtime/content-catalog')
    const parsed = parseContentCatalog(raw)
    if (!parsed) throw new Error('content catalog response is invalid')
    return parsed
  }

  async function loadEntityCatalog() {
    const raw = await request<unknown>('/extensions/runtime/entity-catalog')
    const parsed = parseEntityCatalog(raw)
    if (!parsed) throw new Error('entity catalog response is invalid')
    return parsed
  }

  async function loadMediaCatalog() {
    const raw = await request<unknown>('/extensions/runtime/media-catalog')
    const parsed = parseMediaCatalog(raw)
    if (!parsed) throw new Error('media catalog response is invalid')
    return parsed
  }

  async function dryRunEntityImportExport(entityId: string, action: 'import' | 'export') {
    const id = entityId.trim()
    if (!id) throw new Error('entity id is required')
    const path = `/admin/extensions/entity-catalog/${encodeURIComponent(id)}/import-export-dry-run?action=${action}`
    const raw = await request<unknown>(path)
    const parsed = parseEntityImportExportDryRun(raw)
    if (!parsed) throw new Error('entity import/export dry-run response is invalid')
    return parsed
  }

  return {
    loadContentCatalog,
    loadEntityCatalog,
    loadMediaCatalog,
    dryRunEntityImportExport
  }
}
