import { apiErrorReason } from '../useApiClient'

export type CacheInspectorPolicy = 'private' | 'actor' | 'permission' | 'public'
export type CacheInspectorOutcome =
  | 'hit'
  | 'miss'
  | 'allowed'
  | 'denied'
  | 'stale'
  | 'conflict'
  | 'error'
  | 'cancel'
  | 'deadline'

export type CacheInspectorArtifact = {
  extensionId: string
  extensionVersion: string
  packageDigest: string
  versionId: number
  runtimeInstanceId: string
  core: boolean
}

export type CacheInspectorDeclaration = {
  id: string
  contractVersion: string
  namespace: string
  policy: CacheInspectorPolicy
  provider: string
  invalidators: string[]
}

export type CacheInspectorContribution = CacheInspectorDeclaration & {
  artifact: CacheInspectorArtifact
}

export type CacheInspectorPublication = {
  artifact: CacheInspectorArtifact
  caches: CacheInspectorDeclaration[]
}

export type CacheInspectorRegistry = {
  schemaVersion: 'sforum.cache-registry@1'
  revision: number
  digest: string
  safeMode: boolean
  publications: CacheInspectorPublication[]
  caches: CacheInspectorContribution[]
}

export type CacheInspectorMetrics = {
  operation: string
  samples: number
  hits: number
  misses: number
  allowed: number
  denied: number
  stale: number
  conflicts: number
  errors: number
  canceled: number
  deadlines: number
  slow: number
  affected: number
  totalDurationMicros: number
  averageDurationMicros: number
  p95DurationMicros: number
}

export type CacheInspectorTrace = {
  sequence: number
  extensionId: string
  extensionVersion: string
  artifactDigest: string
  runtimeInstanceId: string
  versionId: number
  cacheId: string
  contractVersion: string
  registryRevision: number
  registryCurrent: boolean
  providerRevision: number
  providerId: string
  providerExtension: string
  providerArtifact: string
  providerRuntime: string
  providerVersionId: number
  operation: string
  tagDigest: string
  tagCount: number
  invalidatorId: string
  outcome: CacheInspectorOutcome
  durationMicros: number
  attempts: number
  hit: boolean
  affected: number
  slow: boolean
}

export type CacheInspectorSnapshot = {
  schemaVersion: 'sforum.cache-inspector@1'
  registry: CacheInspectorRegistry
  retainedFromSequence: number
  retainedThroughSequence: number
  metrics: CacheInspectorMetrics
  operations: CacheInspectorMetrics[]
  traces: CacheInspectorTrace[]
  invalidations: CacheInspectorTrace[]
}

export type CacheInspectorErrorKind =
  | 'permission'
  | 'validation'
  | 'conflict'
  | 'unavailable'
  | 'generic'

const POLICY_VALUES = new Set<CacheInspectorPolicy>(['private', 'actor', 'permission', 'public'])
const OUTCOME_VALUES = new Set<CacheInspectorOutcome>([
  'hit', 'miss', 'allowed', 'denied', 'stale', 'conflict', 'error', 'cancel', 'deadline'
])
const DIGEST_PATTERN = /^[0-9a-f]{64}$/
const ID_PATTERN = /^[a-z0-9][a-z0-9._-]{1,80}$/
const CONTRACT_PATTERN = /^[a-z0-9][a-z0-9._-]*@[1-9][0-9]*$/
const MAX_PUBLICATIONS = 512
const MAX_CACHES = 4096
const MAX_TRACES = 200

function isRecord(value: unknown): value is Record<string, unknown> {
  return Boolean(value) && typeof value === 'object' && !Array.isArray(value)
}

function hasOnlyKeys(value: Record<string, unknown>, allowed: readonly string[]) {
  const permitted = new Set(allowed)
  return Object.keys(value).every(key => permitted.has(key))
}

function stringValue(value: unknown, maximum: number, pattern?: RegExp): string | undefined {
  if (typeof value !== 'string' || value.length === 0 || value.length > maximum) return undefined
  if (pattern && !pattern.test(value)) return undefined
  return value
}

function optionalString(value: unknown, maximum: number, pattern?: RegExp): string | undefined {
  if (value === undefined) return ''
  if (value === '') return ''
  return stringValue(value, maximum, pattern)
}

function uintValue(value: unknown, fallback?: number): number | undefined {
  if (value === undefined && fallback !== undefined) return fallback
  if (typeof value !== 'number' || !Number.isSafeInteger(value) || value < 0) return undefined
  return value
}

function boolValue(value: unknown, fallback?: boolean): boolean | undefined {
  if (value === undefined && fallback !== undefined) return fallback
  return typeof value === 'boolean' ? value : undefined
}

function parseStringList(value: unknown, maximum: number): string[] | undefined {
  if (value === undefined) return []
  if (!Array.isArray(value) || value.length > maximum) return undefined
  const result: string[] = []
  for (const item of value) {
    const parsed = stringValue(item, 81, ID_PATTERN)
    if (!parsed) return undefined
    result.push(parsed)
  }
  return result
}

function parseArtifact(value: unknown): CacheInspectorArtifact | undefined {
  if (!isRecord(value) || !hasOnlyKeys(value, [
    'extensionId', 'extensionVersion', 'packageDigest', 'versionId', 'runtimeInstanceId', 'core'
  ])) return undefined

  const extensionId = stringValue(value.extensionId, 81, ID_PATTERN)
  const extensionVersion = stringValue(value.extensionVersion, 128)
  const packageDigest = stringValue(value.packageDigest, 64, DIGEST_PATTERN)
  const versionId = uintValue(value.versionId, 0)
  const runtimeInstanceId = optionalString(value.runtimeInstanceId, 512)
  const core = boolValue(value.core, false)
  if (!extensionId || !extensionVersion || !packageDigest || versionId === undefined
    || runtimeInstanceId === undefined || core === undefined) return undefined
  if (core && (versionId !== 0 || runtimeInstanceId !== '')) return undefined
  if (!core && (versionId === 0 || runtimeInstanceId === '')) return undefined

  return { extensionId, extensionVersion, packageDigest, versionId, runtimeInstanceId, core }
}

function parseDeclaration(
  value: unknown,
  contribution: boolean
): CacheInspectorDeclaration | CacheInspectorContribution | undefined {
  const allowed = ['id', 'contractVersion', 'namespace', 'policy', 'provider', 'invalidators']
  if (contribution) allowed.push('artifact')
  if (!isRecord(value) || !hasOnlyKeys(value, allowed)) return undefined

  const id = stringValue(value.id, 81, ID_PATTERN)
  const contractVersion = stringValue(value.contractVersion, 256, CONTRACT_PATTERN)
  const namespace = stringValue(value.namespace, 81, ID_PATTERN)
  const policy = stringValue(value.policy, 16) as CacheInspectorPolicy | undefined
  const provider = optionalString(value.provider, 81, ID_PATTERN)
  const invalidators = parseStringList(value.invalidators, 64)
  if (!id || !contractVersion || !namespace || !policy || !POLICY_VALUES.has(policy)
    || provider === undefined || !invalidators) return undefined

  const declaration: CacheInspectorDeclaration = {
    id, contractVersion, namespace, policy, provider, invalidators
  }
  if (!contribution) return declaration
  const artifact = parseArtifact(value.artifact)
  return artifact ? { ...declaration, artifact } : undefined
}

function parsePublication(value: unknown): CacheInspectorPublication | undefined {
  if (!isRecord(value) || !hasOnlyKeys(value, ['artifact', 'caches']) || !Array.isArray(value.caches)
    || value.caches.length > 512) return undefined
  const artifact = parseArtifact(value.artifact)
  if (!artifact) return undefined
  const caches: CacheInspectorDeclaration[] = []
  for (const item of value.caches) {
    const declaration = parseDeclaration(item, false)
    if (!declaration || 'artifact' in declaration) return undefined
    caches.push(declaration)
  }
  return { artifact, caches }
}

function parseRegistry(value: unknown): CacheInspectorRegistry | undefined {
  if (!isRecord(value) || !hasOnlyKeys(value, [
    'schemaVersion', 'revision', 'digest', 'safeMode', 'publications', 'caches'
  ])) return undefined
  if (value.schemaVersion !== 'sforum.cache-registry@1'
    || !Array.isArray(value.publications) || value.publications.length > MAX_PUBLICATIONS
    || !Array.isArray(value.caches) || value.caches.length > MAX_CACHES) return undefined

  const revision = uintValue(value.revision)
  const digest = stringValue(value.digest, 64, DIGEST_PATTERN)
  const safeMode = boolValue(value.safeMode, false)
  if (revision === undefined || !digest || safeMode === undefined) return undefined

  const publications: CacheInspectorPublication[] = []
  for (const item of value.publications) {
    const publication = parsePublication(item)
    if (!publication) return undefined
    publications.push(publication)
  }
  const caches: CacheInspectorContribution[] = []
  for (const item of value.caches) {
    const contribution = parseDeclaration(item, true)
    if (!contribution || !('artifact' in contribution)) return undefined
    caches.push(contribution)
  }
  return {
    schemaVersion: 'sforum.cache-registry@1', revision, digest, safeMode, publications, caches
  }
}

function parseMetrics(value: unknown): CacheInspectorMetrics | undefined {
  if (!isRecord(value) || !hasOnlyKeys(value, [
    'operation', 'samples', 'hits', 'misses', 'allowed', 'denied', 'stale', 'conflicts',
    'errors', 'canceled', 'deadlines', 'slow', 'affected', 'totalDurationMicros',
    'averageDurationMicros', 'p95DurationMicros'
  ])) return undefined
  const operation = optionalString(value.operation, 32)
  const fields = [
    'samples', 'hits', 'misses', 'allowed', 'denied', 'stale', 'conflicts', 'errors',
    'canceled', 'deadlines', 'slow', 'affected', 'totalDurationMicros',
    'averageDurationMicros', 'p95DurationMicros'
  ] as const
  const numbers = Object.fromEntries(fields.map(field => [field, uintValue(value[field])])) as
    Record<(typeof fields)[number], number | undefined>
  if (operation === undefined || fields.some(field => numbers[field] === undefined)) return undefined
  return { operation, ...numbers } as CacheInspectorMetrics
}

function parseTrace(value: unknown): CacheInspectorTrace | undefined {
  if (!isRecord(value) || !hasOnlyKeys(value, [
    'sequence', 'extensionId', 'extensionVersion', 'artifactDigest', 'runtimeInstanceId', 'versionId',
    'cacheId', 'contractVersion', 'registryRevision', 'registryCurrent', 'providerRevision', 'providerId',
    'providerExtension', 'providerArtifact', 'providerRuntime', 'providerVersionId', 'operation',
    'tagDigest', 'tagCount', 'invalidatorId', 'outcome', 'durationMicros', 'attempts', 'hit',
    'affected', 'slow'
  ])) return undefined

  const sequence = uintValue(value.sequence)
  const extensionId = optionalString(value.extensionId, 255)
  const extensionVersion = optionalString(value.extensionVersion, 128)
  const artifactDigest = optionalString(value.artifactDigest, 128)
  const runtimeInstanceId = optionalString(value.runtimeInstanceId, 255)
  const versionId = uintValue(value.versionId, 0)
  const cacheId = optionalString(value.cacheId, 255)
  const contractVersion = optionalString(value.contractVersion, 128)
  const registryRevision = uintValue(value.registryRevision, 0)
  const registryCurrent = boolValue(value.registryCurrent)
  const providerRevision = uintValue(value.providerRevision, 0)
  const providerId = optionalString(value.providerId, 255)
  const providerExtension = optionalString(value.providerExtension, 255)
  const providerArtifact = optionalString(value.providerArtifact, 128)
  const providerRuntime = optionalString(value.providerRuntime, 255)
  const providerVersionId = uintValue(value.providerVersionId, 0)
  const operation = stringValue(value.operation, 32)
  const tagDigest = optionalString(value.tagDigest, 64, DIGEST_PATTERN)
  const tagCount = uintValue(value.tagCount, 0)
  const invalidatorId = optionalString(value.invalidatorId, 255)
  const outcome = stringValue(value.outcome, 16) as CacheInspectorOutcome | undefined
  const durationMicros = uintValue(value.durationMicros)
  const attempts = uintValue(value.attempts, 0)
  const hit = boolValue(value.hit, false)
  const affected = uintValue(value.affected, 0)
  const slow = boolValue(value.slow, false)
  const parsed = [sequence, extensionId, extensionVersion, artifactDigest, runtimeInstanceId, versionId,
    cacheId, contractVersion, registryRevision, registryCurrent, providerRevision, providerId,
    providerExtension, providerArtifact, providerRuntime, providerVersionId, operation, tagDigest,
    tagCount, invalidatorId, outcome, durationMicros, attempts, hit, affected, slow]
  if (parsed.some(item => item === undefined) || !outcome || !OUTCOME_VALUES.has(outcome)) return undefined

  return {
    sequence: sequence!, extensionId: extensionId!, extensionVersion: extensionVersion!,
    artifactDigest: artifactDigest!, runtimeInstanceId: runtimeInstanceId!, versionId: versionId!,
    cacheId: cacheId!, contractVersion: contractVersion!, registryRevision: registryRevision!,
    registryCurrent: registryCurrent!, providerRevision: providerRevision!, providerId: providerId!,
    providerExtension: providerExtension!, providerArtifact: providerArtifact!,
    providerRuntime: providerRuntime!, providerVersionId: providerVersionId!, operation: operation!,
    tagDigest: tagDigest!, tagCount: tagCount!, invalidatorId: invalidatorId!, outcome,
    durationMicros: durationMicros!, attempts: attempts!, hit: hit!, affected: affected!, slow: slow!
  }
}

function parseMetricsList(value: unknown): CacheInspectorMetrics[] | undefined {
  if (!Array.isArray(value) || value.length > 64) return undefined
  const result: CacheInspectorMetrics[] = []
  for (const item of value) {
    const metrics = parseMetrics(item)
    if (!metrics) return undefined
    result.push(metrics)
  }
  return result
}

function parseTraceList(value: unknown): CacheInspectorTrace[] | undefined {
  if (!Array.isArray(value) || value.length > MAX_TRACES) return undefined
  const result: CacheInspectorTrace[] = []
  for (const item of value) {
    const trace = parseTrace(item)
    if (!trace) return undefined
    result.push(trace)
  }
  return result
}

export function parseCacheInspectorSnapshot(value: unknown): CacheInspectorSnapshot | null {
  if (!isRecord(value) || !hasOnlyKeys(value, [
    'schemaVersion', 'registry', 'retainedFromSequence', 'retainedThroughSequence',
    'metrics', 'operations', 'traces', 'invalidations'
  ]) || value.schemaVersion !== 'sforum.cache-inspector@1') return null

  const registry = parseRegistry(value.registry)
  const retainedFromSequence = uintValue(value.retainedFromSequence, 0)
  const retainedThroughSequence = uintValue(value.retainedThroughSequence, 0)
  const metrics = parseMetrics(value.metrics)
  const operations = parseMetricsList(value.operations)
  const traces = parseTraceList(value.traces)
  const invalidations = parseTraceList(value.invalidations)
  if (!registry || retainedFromSequence === undefined || retainedThroughSequence === undefined
    || !metrics || !operations || !traces || !invalidations) return null
  if ((retainedFromSequence === 0) !== (retainedThroughSequence === 0)
    || retainedFromSequence > retainedThroughSequence) return null

  return {
    schemaVersion: 'sforum.cache-inspector@1', registry, retainedFromSequence,
    retainedThroughSequence, metrics, operations, traces, invalidations
  }
}

export function formatCacheDuration(micros: number) {
  if (!Number.isFinite(micros) || micros < 0) return '—'
  if (micros < 1000) return `${Math.round(micros)} µs`
  if (micros < 1_000_000) return `${(micros / 1000).toFixed(micros < 10_000 ? 2 : 1)} ms`
  return `${(micros / 1_000_000).toFixed(2)} s`
}

function apiStatusCode(error: unknown): number | null {
  if (!error || typeof error !== 'object') return null
  const candidate = error as { statusCode?: unknown, status?: unknown, response?: { status?: unknown } }
  for (const value of [candidate.statusCode, candidate.status, candidate.response?.status]) {
    const status = Number(value)
    if (Number.isInteger(status) && status >= 100 && status < 600) return status
  }
  return null
}

export function cacheInspectorErrorKind(error: unknown): CacheInspectorErrorKind {
  const reason = apiErrorReason(error)
  if (reason === 'permission.denied') return 'permission'
  if (reason === 'extensions.cache_inspector_invalid') return 'validation'
  if (reason === 'extensions.cache_inspector_conflict') return 'conflict'
  if (reason === 'extensions.cache_inspector_unavailable') return 'unavailable'
  const status = apiStatusCode(error)
  if (status === 403) return 'permission'
  if (status === 422) return 'validation'
  if (status === 409) return 'conflict'
  if (status === 503) return 'unavailable'
  return 'generic'
}

export function useAdminCacheInspector() {
  const { request } = useApiClient()

  const inspect = async (limit = 100) => {
    if (!Number.isInteger(limit) || limit < 1 || limit > MAX_TRACES) {
      throw Object.assign(new Error('cache inspector limit invalid'), {
        data: { code: 422, message: 'invalid limit', data: { reason: 'extensions.cache_inspector_invalid' } }
      })
    }
    const raw = await request<unknown>(`/admin/extensions/cache-inspector?limit=${limit}`)
    const snapshot = parseCacheInspectorSnapshot(raw)
    if (!snapshot) {
      throw Object.assign(new Error('cache inspector response invalid'), {
        data: {
          code: 503,
          message: 'invalid inspector response',
          data: { reason: 'extensions.cache_inspector_unavailable' }
        }
      })
    }
    return snapshot
  }

  return { inspect }
}
