export type QueryCatalogEntry = {
  queryId: string
  hostQueryId: string
  hostPlanVersion: string
  entity: string
  fields: string[]
  filters: string[]
  sort: string[]
  pagination: string
  permissionPolicy: string
  resultSchema: string
  resultSchemaDigest: string
}

export type QueryCatalogSnapshot = {
  extensionId: string
  extensionVersion: string
  artifactDigest: string
  costMaximum: number
  total: number
  queries: QueryCatalogEntry[]
}

export function queryCatalogDigestShort(digest: string | undefined) {
  if (!digest) return ''
  return digest.slice(0, 8)
}

/** shortcode 投影以专用 actor 决策策略标注，只在宿主内部经一次性委托可达。 */
export function isQueryCatalogHostInternal(entry: QueryCatalogEntry) {
  return entry.queryId.startsWith('core.query.shortcode.')
}

export function parseQueryCatalogEntry(value: unknown): QueryCatalogEntry | undefined {
  if (!isRecord(value)) return undefined
  const queryId = value.queryId
  const hostQueryId = value.hostQueryId
  const hostPlanVersion = value.hostPlanVersion
  const entity = value.entity
  const pagination = value.pagination
  const permissionPolicy = value.permissionPolicy
  const resultSchema = value.resultSchema
  const resultSchemaDigest = value.resultSchemaDigest
  if (typeof queryId !== 'string' || typeof hostQueryId !== 'string' ||
    typeof hostPlanVersion !== 'string' || typeof entity !== 'string' ||
    typeof pagination !== 'string' || typeof permissionPolicy !== 'string' ||
    typeof resultSchema !== 'string' || typeof resultSchemaDigest !== 'string') {
    return undefined
  }
  if (!isStringArray(value.fields) || !isStringArray(value.filters) || !isStringArray(value.sort)) {
    return undefined
  }
  return {
    queryId,
    hostQueryId,
    hostPlanVersion,
    entity,
    fields: value.fields,
    filters: value.filters,
    sort: value.sort,
    pagination,
    permissionPolicy,
    resultSchema,
    resultSchemaDigest
  }
}

export function parseQueryCatalogSnapshot(value: unknown): QueryCatalogSnapshot | undefined {
  if (!isRecord(value)) return undefined
  const extensionId = value.extensionId
  const extensionVersion = value.extensionVersion
  const artifactDigest = value.artifactDigest
  const costMaximum = value.costMaximum
  if (typeof extensionId !== 'string' || typeof extensionVersion !== 'string' ||
    typeof artifactDigest !== 'string' || typeof costMaximum !== 'number' ||
    !Number.isInteger(costMaximum) || costMaximum < 1) {
    return undefined
  }
  if (typeof value.total !== 'number' || !Array.isArray(value.queries)) return undefined
  const queries: QueryCatalogEntry[] = []
  for (const item of value.queries) {
    const entry = parseQueryCatalogEntry(item)
    if (!entry) return undefined
    queries.push(entry)
  }
  if (value.total !== queries.length) return undefined
  return {
    extensionId,
    extensionVersion,
    artifactDigest,
    costMaximum,
    total: queries.length,
    queries
  }
}

function isRecord(value: unknown): value is Record<string, unknown> {
  return typeof value === 'object' && value !== null && !Array.isArray(value)
}

function isStringArray(value: unknown): value is string[] {
  return Array.isArray(value) && value.every(item => typeof item === 'string')
}

export function useAdminQueryCatalog() {
  const { request } = useApiClient()
  const basePath = '/admin/extensions/query-catalog'

  const load = async (): Promise<QueryCatalogSnapshot> => {
    const raw = await request<unknown>(basePath)
    const snapshot = parseQueryCatalogSnapshot(raw)
    if (!snapshot) {
      throw Object.assign(new Error('query catalog response invalid'), {
        data: {
          code: 503,
          message: 'invalid catalog response',
          data: { reason: 'extensions.query_catalog_unavailable' }
        }
      })
    }
    return snapshot
  }

  return { load }
}
