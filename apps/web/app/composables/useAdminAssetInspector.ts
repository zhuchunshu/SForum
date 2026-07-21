import { apiErrorMessage, apiErrorReason } from './useApiClient'

export type AssetInspectorHandle = {
  handle: string
  type: string
  path: string
  module: boolean
  loading: string
  integrity: string
  csp: string[]
  scope: string[]
}

export type AssetInspectorPublication = {
  extensionId: string
  extensionVersion: string
  packageDigest: string
  ownerKind: string
  assets: AssetInspectorHandle[]
}

export type AssetInspectorSnapshot = {
  schemaVersion: string
  revision: number
  digest: string
  publicationCount: number
  assetCount: number
  publications: AssetInspectorPublication[]
}

const ASSET_INSPECTOR_SCHEMA = 'sforum.asset-inspector@1'

function asObject(value: unknown): Record<string, unknown> | null {
  return value && typeof value === 'object' && !Array.isArray(value)
    ? value as Record<string, unknown>
    : null
}

function asString(value: unknown): string {
  return typeof value === 'string' ? value : ''
}

function asBoolean(value: unknown): boolean {
  return value === true
}

function asStringArray(value: unknown): string[] {
  if (!Array.isArray(value)) return []
  return value.filter((item): item is string => typeof item === 'string')
}

function parseHandle(raw: unknown): AssetInspectorHandle | null {
  const data = asObject(raw)
  if (!data) return null
  if (typeof data.handle !== 'string' || typeof data.type !== 'string') return null
  return {
    handle: data.handle,
    type: data.type,
    path: asString(data.path),
    module: asBoolean(data.module),
    loading: asString(data.loading),
    integrity: asString(data.integrity),
    csp: asStringArray(data.csp),
    scope: asStringArray(data.scope)
  }
}

function parsePublication(raw: unknown): AssetInspectorPublication | null {
  const data = asObject(raw)
  if (!data) return null
  if (typeof data.extensionId !== 'string' || typeof data.extensionVersion !== 'string') return null
  if (typeof data.packageDigest !== 'string' || typeof data.ownerKind !== 'string') return null
  if (!Array.isArray(data.assets)) return null
  const assets: AssetInspectorHandle[] = []
  for (const item of data.assets) {
    const handle = parseHandle(item)
    if (!handle) return null
    assets.push(handle)
  }
  return {
    extensionId: data.extensionId,
    extensionVersion: data.extensionVersion,
    packageDigest: data.packageDigest,
    ownerKind: data.ownerKind,
    assets
  }
}

/** 解析资源检查器响应；形状非法时返回 null（不抛异常）。 */
export function parseAssetInspectorSnapshot(raw: unknown): AssetInspectorSnapshot | null {
  const data = asObject(raw)
  if (!data) return null
  if (data.schemaVersion !== ASSET_INSPECTOR_SCHEMA) return null
  if (typeof data.revision !== 'number' || !Number.isFinite(data.revision)) return null
  if (typeof data.digest !== 'string') return null
  if (typeof data.publicationCount !== 'number' || !Number.isFinite(data.publicationCount)) return null
  if (typeof data.assetCount !== 'number' || !Number.isFinite(data.assetCount)) return null
  if (!Array.isArray(data.publications)) return null
  const publications: AssetInspectorPublication[] = []
  for (const item of data.publications) {
    const publication = parsePublication(item)
    if (!publication) return null
    publications.push(publication)
  }
  return {
    schemaVersion: ASSET_INSPECTOR_SCHEMA,
    revision: data.revision,
    digest: data.digest,
    publicationCount: data.publicationCount,
    assetCount: data.assetCount,
    publications
  }
}

function clampInspectorLimit(limit: number): number {
  if (!Number.isInteger(limit) || limit < 1 || limit > 200) {
    throw new Error('inspector limit must be an integer from 1 to 200')
  }
  return limit
}

export function useAdminAssetInspector() {
  const { request } = useApiClient()

  const inspect = async (limit = 50) => {
    const safeLimit = clampInspectorLimit(limit)
    try {
      const raw = await request<unknown>(`/admin/extensions/asset-inspector?limit=${safeLimit}`)
      const snapshot = parseAssetInspectorSnapshot(raw)
      if (!snapshot) {
        throw new Error('asset inspector response invalid')
      }
      return snapshot
    } catch (error) {
      const message =
        apiErrorMessage(error)
        || apiErrorReason(error)
        || (error instanceof Error && error.message.trim() ? error.message : '')
        || 'asset inspector failed'
      throw new Error(message)
    }
  }

  return { inspect }
}
