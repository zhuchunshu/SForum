import { apiErrorReason } from '../useApiClient'
import type { RouteProviderArtifact, RouteProviderKey, RouteProviderSelection } from './useAdminRouteProviders'

/** HTTP methods accepted by the Route Inspector UI (wildcard rejected server-side). */
export const ROUTE_INSPECTOR_METHODS = [
  'GET',
  'POST',
  'PUT',
  'PATCH',
  'DELETE',
  'HEAD',
  'OPTIONS'
] as const

export type RouteInspectorMethod = (typeof ROUTE_INSPECTOR_METHODS)[number]

export type RouteInspectorPhase = 'global' | 'before' | 'filter' | 'wrap' | 'handler' | 'after'

export type RouteInspectorInvocationStage = 'request' | 'handler' | 'response'

export type RouteInspectorResolution = 'resolved' | 'not_found' | 'ambiguous' | 'stale'

export type RouteInspectorProviderStatus = 'not_required' | 'selected' | 'unselected' | 'stale'

export type RouteInspectorConflictKind = 'path_method' | 'route_identity' | 'provider_selection'

export type RouteInspectorSelectionStatus = 'selected' | 'unselected' | 'stale'

export type RouteInspectorTraceOutcome =
  | 'succeeded'
  | 'denied'
  | 'schema_rejected'
  | 'transport_failed'
  | 'fallback_used'
  | 'committed'

export type RouteInspectorCommitState =
  | 'pristine'
  | 'response_started'
  | 'side_effect_started'
  | 'committed'

export type RouteInspectorProvider = {
  kind: 'core' | 'plugin'
  artifact?: RouteProviderArtifact
}

export type RouteInspectorPluginGuard = {
  id: string
  contractVersion: string
  kind: 'custom' | 'raw_request'
  entry: string
  digest: string
  permissions?: string[]
}

export type RouteInspectorStep = {
  index: number
  phase: RouteInspectorPhase
  action: string
  routeId: string
  contractVersion: string
  targetRouteId?: string
  method: string
  /** Declared route template — never the raw request path. */
  path: string
  pathSignature: string
  provider: RouteInspectorProvider
  guard: string
  access: string
  pluginGuard?: RouteInspectorPluginGuard
  permission?: string
  handler?: string
  destination?: string
  requestSchema?: string
  responseSchema?: string
  mutableRequestFields?: string[]
  mutableResponseFields?: string[]
  mode: string
  fallback: string
  timeoutMs: number
  priority: number
}

export type RouteInspectorConflict = {
  kind: RouteInspectorConflictKind
  routeId?: string
  contractVersion?: string
  method: string
  pathSignature: string
  candidates: RouteInspectorStep[]
  selectionKey?: RouteProviderKey
  desired?: RouteProviderSelection
  selectionStatus?: RouteInspectorSelectionStatus
}

export type RouteInspectorProviderResolution = {
  status: RouteInspectorProviderStatus
  desired?: RouteProviderSelection
  live?: RouteInspectorProvider
}

/** Bounded redacted trace — no request/response/header/query/body/secret fields. */
export type RouteInspectorTrace = {
  sequence: number
  observedAt: string
  revision: number
  stepIndex: number
  phase: RouteInspectorPhase
  invocationStage: RouteInspectorInvocationStage
  action: string
  routeId: string
  contractVersion: string
  method: string
  pathSignature: string
  mode: string
  fallback: string
  outcome: RouteInspectorTraceOutcome
  durationMicros: number
  commitState: RouteInspectorCommitState
  provider: RouteInspectorProvider
}

export type RouteInspectorSnapshot = {
  revision: number
  safeMode: boolean
  method: string
  resolution: RouteInspectorResolution
  chain: RouteInspectorStep[]
  provider: RouteInspectorProviderResolution
  conflicts: RouteInspectorConflict[]
  traces: RouteInspectorTrace[]
}

export type RouteInspectorLookup = {
  method: string
  path: string
}

export type RouteInspectorLookupValidation =
  | { ok: true, method: RouteInspectorMethod, path: string }
  | { ok: false, reason: 'empty' | 'method' | 'path' }

const PHASES = new Set<string>(['global', 'before', 'filter', 'wrap', 'handler', 'after'])
const INVOCATION_STAGES = new Set<string>(['request', 'handler', 'response'])
const RESOLUTIONS = new Set<string>(['resolved', 'not_found', 'ambiguous', 'stale'])
const PROVIDER_STATUSES = new Set<string>(['not_required', 'selected', 'unselected', 'stale'])
const CONFLICT_KINDS = new Set<string>(['path_method', 'route_identity', 'provider_selection'])
const SELECTION_STATUSES = new Set<string>(['selected', 'unselected', 'stale'])
const TRACE_OUTCOMES = new Set<string>([
  'succeeded',
  'denied',
  'schema_rejected',
  'transport_failed',
  'fallback_used',
  'committed'
])
const COMMIT_STATES = new Set<string>(['pristine', 'response_started', 'side_effect_started', 'committed'])
const PROVIDER_KINDS = new Set<string>(['core', 'plugin'])
const GUARD_KINDS = new Set<string>(['custom', 'raw_request'])
const METHOD_SET = new Set<string>(ROUTE_INSPECTOR_METHODS)
const MUTABLE_FIELD_MAX_COUNT = 64
const MUTABLE_FIELD_MAX_BYTES = 256
const MUTABLE_FIELD_MAX_TOKENS = 32
const UTF8_ENCODER = new TextEncoder()
const MUTABLE_REQUEST_ACTIONS = new Set<string>(['global_middleware', 'before', 'filter', 'wrap'])
const MUTABLE_RESPONSE_ACTIONS = new Set<string>(['filter', 'wrap', 'after'])

function isRecord(value: unknown): value is Record<string, unknown> {
  return Boolean(value) && typeof value === 'object' && !Array.isArray(value)
}

function asString(value: unknown): string | undefined {
  return typeof value === 'string' ? value : undefined
}

function asFiniteNumber(value: unknown): number | undefined {
  return typeof value === 'number' && Number.isFinite(value) ? value : undefined
}

function asNonNegativeInt(value: unknown): number | undefined {
  const n = asFiniteNumber(value)
  if (n === undefined || !Number.isInteger(n) || n < 0) return undefined
  return n
}

function asPositiveInt(value: unknown): number | undefined {
  const n = asNonNegativeInt(value)
  return n !== undefined && n >= 1 ? n : undefined
}

function asBoolean(value: unknown): boolean | undefined {
  return typeof value === 'boolean' ? value : undefined
}

function validMutableFieldPointer(value: string): boolean {
  if (
    value !== value.trim()
    || !value.startsWith('/')
    || UTF8_ENCODER.encode(value).byteLength > MUTABLE_FIELD_MAX_BYTES
  ) {
    return false
  }
  let tokens = 1
  for (let index = 1; index < value.length; index++) {
    const character = value[index]
    if (character === '/') {
      tokens++
      if (tokens > MUTABLE_FIELD_MAX_TOKENS) return false
      continue
    }
    if (character !== '~') continue
    index++
    if (index >= value.length || value[index] !== '0' && value[index] !== '1') return false
  }
  return true
}

// undefined means omitted by the backend; null means a present but invalid value.
function parseMutableFieldList(value: unknown): string[] | null | undefined {
  if (value === undefined) return undefined
  if (!Array.isArray(value) || value.length > MUTABLE_FIELD_MAX_COUNT) return null
  const fields: string[] = []
  const seen = new Set<string>()
  for (const item of value) {
    if (typeof item !== 'string' || !validMutableFieldPointer(item) || seen.has(item)) return null
    seen.add(item)
    fields.push(item)
  }
  return fields
}

function parseArtifact(value: unknown): RouteProviderArtifact | undefined {
  if (!isRecord(value)) return undefined
  const extensionId = asString(value.extensionId)
  const extensionVersion = asString(value.extensionVersion)
  const packageDigest = asString(value.packageDigest)
  const runtimeInstanceId = asString(value.runtimeInstanceId)
  if (!extensionId || !extensionVersion || !packageDigest || !runtimeInstanceId) return undefined
  return { extensionId, extensionVersion, packageDigest, runtimeInstanceId }
}

function parseProvider(value: unknown): RouteInspectorProvider | undefined {
  if (!isRecord(value)) return undefined
  const kind = asString(value.kind)
  if (!kind || !PROVIDER_KINDS.has(kind)) return undefined
  const artifact = value.artifact === undefined ? undefined : parseArtifact(value.artifact)
  // 后端只会为插件提供者携带 exact artifact；接受其他组合会掩盖损坏的快照。
  if (kind === 'plugin' && !artifact || kind === 'core' && value.artifact !== undefined) return undefined
  return { kind: kind as RouteInspectorProvider['kind'], artifact }
}

function parsePluginGuard(value: unknown): RouteInspectorPluginGuard | undefined {
  if (!isRecord(value)) return undefined
  const id = asString(value.id)
  const contractVersion = asString(value.contractVersion)
  const kind = asString(value.kind)
  const entry = asString(value.entry)
  const digest = asString(value.digest)
  if (!id || !contractVersion || !kind || !GUARD_KINDS.has(kind) || !entry || !digest) return undefined
  let permissions: string[] | undefined
  if (value.permissions !== undefined) {
    if (!Array.isArray(value.permissions) || !value.permissions.every(item => typeof item === 'string')) {
      return undefined
    }
    permissions = value.permissions as string[]
  }
  return {
    id,
    contractVersion,
    kind: kind as RouteInspectorPluginGuard['kind'],
    entry,
    digest,
    permissions
  }
}

function parseSelectionKey(value: unknown): RouteProviderKey | undefined {
  if (!isRecord(value)) return undefined
  const targetRouteId = asString(value.targetRouteId)
  const targetContractVersion = asString(value.targetContractVersion)
  const method = asString(value.method)
  const pathSignature = asString(value.pathSignature)
  if (!targetRouteId || !targetContractVersion || !method || !pathSignature) return undefined
  return { targetRouteId, targetContractVersion, method, pathSignature }
}

function parseSelection(value: unknown): RouteProviderSelection | undefined {
  if (!isRecord(value)) return undefined
  const key = parseSelectionKey(value.key)
  const routeId = asString(value.routeId)
  const contractVersion = asString(value.contractVersion)
  const extensionId = asString(value.extensionId)
  // OpenAPI: extensionVersionId/selectionAuditEventId/revision ≥ 1；selectedByUserId ≥ 0。
  const extensionVersionId = asPositiveInt(value.extensionVersionId)
  const extensionVersion = asString(value.extensionVersion)
  const packageDigest = asString(value.packageDigest)
  const selectedByUserId = asNonNegativeInt(value.selectedByUserId)
  const selectionAuditEventId = asPositiveInt(value.selectionAuditEventId)
  const revision = asPositiveInt(value.revision)
  const selectedAt = asString(value.selectedAt)
  const updatedAt = asString(value.updatedAt)
  if (
    !key
    || !routeId
    || !contractVersion
    || !extensionId
    || extensionVersionId === undefined
    || !extensionVersion
    || !packageDigest
    || selectedByUserId === undefined
    || selectionAuditEventId === undefined
    || revision === undefined
    || !selectedAt
    || !updatedAt
  ) {
    return undefined
  }
  return {
    key,
    routeId,
    contractVersion,
    extensionId,
    extensionVersionId,
    extensionVersion,
    packageDigest,
    selectedByUserId,
    selectionAuditEventId,
    revision,
    selectedAt,
    updatedAt
  }
}

function parseStep(value: unknown): RouteInspectorStep | undefined {
  if (!isRecord(value)) return undefined
  const index = asNonNegativeInt(value.index)
  const phase = asString(value.phase)
  const action = asString(value.action)
  const routeId = asString(value.routeId)
  const contractVersion = asString(value.contractVersion)
  const method = asString(value.method)
  const path = asString(value.path)
  const pathSignature = asString(value.pathSignature)
  const provider = parseProvider(value.provider)
  const guard = asString(value.guard)
  const access = asString(value.access)
  const mode = asString(value.mode)
  const fallback = asString(value.fallback)
  const timeoutMs = asNonNegativeInt(value.timeoutMs)
  const priority = asFiniteNumber(value.priority)
  if (
    index === undefined
    || !phase
    || !PHASES.has(phase)
    || !action
    || !routeId
    || !contractVersion
    || !method
    || path === undefined
    || pathSignature === undefined
    || !provider
    || guard === undefined
    || access === undefined
    || mode === undefined
    || fallback === undefined
    || timeoutMs === undefined
    || priority === undefined
    || !Number.isInteger(priority)
  ) {
    return undefined
  }

  const targetRouteId = asString(value.targetRouteId)
  const permission = asString(value.permission)
  const handler = asString(value.handler)
  const destination = asString(value.destination)
  const requestSchema = asString(value.requestSchema)
  const responseSchema = asString(value.responseSchema)
  const mutableRequestFields = parseMutableFieldList(value.mutableRequestFields)
  const mutableResponseFields = parseMutableFieldList(value.mutableResponseFields)
  if (mutableRequestFields === null || mutableResponseFields === null) return undefined
  if (mutableRequestFields?.length && !MUTABLE_REQUEST_ACTIONS.has(action)) return undefined
  if (mutableResponseFields?.length && !MUTABLE_RESPONSE_ACTIONS.has(action)) return undefined
  let pluginGuard: RouteInspectorPluginGuard | undefined
  if (value.pluginGuard !== undefined) {
    pluginGuard = parsePluginGuard(value.pluginGuard)
    if (!pluginGuard) return undefined
  }

  return {
    index,
    phase: phase as RouteInspectorPhase,
    action,
    routeId,
    contractVersion,
    targetRouteId,
    method,
    path,
    pathSignature,
    provider,
    guard,
    access,
    pluginGuard,
    permission,
    handler,
    destination,
    requestSchema,
    responseSchema,
    mutableRequestFields,
    mutableResponseFields,
    mode,
    fallback,
    timeoutMs,
    priority
  }
}

function parseConflict(value: unknown): RouteInspectorConflict | undefined {
  if (!isRecord(value)) return undefined
  const kind = asString(value.kind)
  const method = asString(value.method)
  const pathSignature = asString(value.pathSignature)
  if (!kind || !CONFLICT_KINDS.has(kind) || !method || pathSignature === undefined) return undefined
  if (!Array.isArray(value.candidates) || value.candidates.length < 2) return undefined
  const candidates: RouteInspectorStep[] = []
  for (const item of value.candidates) {
    const step = parseStep(item)
    if (!step) return undefined
    candidates.push(step)
  }

  let selectionKey: RouteProviderKey | undefined
  if (value.selectionKey !== undefined) {
    selectionKey = parseSelectionKey(value.selectionKey)
    if (!selectionKey) return undefined
  }
  let desired: RouteProviderSelection | undefined
  if (value.desired !== undefined) {
    desired = parseSelection(value.desired)
    if (!desired) return undefined
  }
  let selectionStatus: RouteInspectorSelectionStatus | undefined
  if (value.selectionStatus !== undefined) {
    const status = asString(value.selectionStatus)
    if (!status || !SELECTION_STATUSES.has(status)) return undefined
    selectionStatus = status as RouteInspectorSelectionStatus
  }

  return {
    kind: kind as RouteInspectorConflictKind,
    routeId: asString(value.routeId),
    contractVersion: asString(value.contractVersion),
    method,
    pathSignature,
    candidates,
    selectionKey,
    desired,
    selectionStatus
  }
}

function parseProviderResolution(value: unknown): RouteInspectorProviderResolution | undefined {
  if (!isRecord(value)) return undefined
  const status = asString(value.status)
  if (!status || !PROVIDER_STATUSES.has(status)) return undefined
  let desired: RouteProviderSelection | undefined
  if (value.desired !== undefined) {
    desired = parseSelection(value.desired)
    if (!desired) return undefined
  }
  let live: RouteInspectorProvider | undefined
  if (value.live !== undefined) {
    live = parseProvider(value.live)
    if (!live) return undefined
  }
  return { status: status as RouteInspectorProviderStatus, desired, live }
}

function parseTrace(value: unknown): RouteInspectorTrace | undefined {
  if (!isRecord(value)) return undefined
  const sequence = asPositiveInt(value.sequence)
  const observedAt = asString(value.observedAt)
  const revision = asPositiveInt(value.revision)
  const stepIndex = asNonNegativeInt(value.stepIndex)
  const phase = asString(value.phase)
  const invocationStage = asString(value.invocationStage)
  const action = asString(value.action)
  const routeId = asString(value.routeId)
  const contractVersion = asString(value.contractVersion)
  const method = asString(value.method)
  const pathSignature = asString(value.pathSignature)
  const mode = asString(value.mode)
  const fallback = asString(value.fallback)
  const outcome = asString(value.outcome)
  const durationMicros = asNonNegativeInt(value.durationMicros)
  const commitState = asString(value.commitState)
  const provider = parseProvider(value.provider)
  if (
    sequence === undefined
    || !observedAt
    || revision === undefined
    || stepIndex === undefined
    || !phase
    || !PHASES.has(phase)
    || !invocationStage
    || !INVOCATION_STAGES.has(invocationStage)
    || !action
    || !routeId
    || !contractVersion
    || !method
    || pathSignature === undefined
    || mode === undefined
    || fallback === undefined
    || !outcome
    || !TRACE_OUTCOMES.has(outcome)
    || durationMicros === undefined
    || !commitState
    || !COMMIT_STATES.has(commitState)
    || !provider
  ) {
    return undefined
  }
  return {
    sequence,
    observedAt,
    revision,
    stepIndex,
    phase: phase as RouteInspectorPhase,
    invocationStage: invocationStage as RouteInspectorInvocationStage,
    action,
    routeId,
    contractVersion,
    method,
    pathSignature,
    mode,
    fallback,
    outcome: outcome as RouteInspectorTraceOutcome,
    durationMicros,
    commitState: commitState as RouteInspectorCommitState,
    provider
  }
}

/**
 * Strict runtime validation at the API boundary. Rejects unknown shapes instead
 * of trusting server data with `as` casts.
 */
export function parseRouteInspectorSnapshot(value: unknown): RouteInspectorSnapshot | null {
  if (!isRecord(value)) return null
  const revision = asPositiveInt(value.revision)
  const safeMode = asBoolean(value.safeMode)
  const method = asString(value.method)
  const resolution = asString(value.resolution)
  if (
    revision === undefined
    || safeMode === undefined
    || !method
    || !resolution
    || !RESOLUTIONS.has(resolution)
    || !Array.isArray(value.chain)
    || !Array.isArray(value.conflicts)
    || !Array.isArray(value.traces)
  ) {
    return null
  }
  if (value.traces.length > 50) return null

  const chain: RouteInspectorStep[] = []
  for (const item of value.chain) {
    const step = parseStep(item)
    if (!step) return null
    chain.push(step)
  }

  const provider = parseProviderResolution(value.provider)
  if (!provider) return null

  const conflicts: RouteInspectorConflict[] = []
  for (const item of value.conflicts) {
    const conflict = parseConflict(item)
    if (!conflict) return null
    conflicts.push(conflict)
  }

  const traces: RouteInspectorTrace[] = []
  for (const item of value.traces) {
    const trace = parseTrace(item)
    if (!trace) return null
    traces.push(trace)
  }

  return {
    revision,
    safeMode,
    method,
    resolution: resolution as RouteInspectorResolution,
    chain,
    provider,
    conflicts,
    traces
  }
}

/**
 * Normalize form/query input; does not call the API.
 * Path rules mirror backend normalizeRequestPath: absolute path, no //, no \, #, .., control chars.
 * Query strings are stripped client-side the same way the server discards them.
 */
export function validateRouteInspectorLookup(input: RouteInspectorLookup): RouteInspectorLookupValidation {
  const method = String(input.method || '').trim().toUpperCase()
  let path = String(input.path || '').trim()
  if (!method && !path) return { ok: false, reason: 'empty' }
  if (!METHOD_SET.has(method)) return { ok: false, reason: 'method' }
  if (!path) return { ok: false, reason: 'empty' }

  const queryIndex = path.indexOf('?')
  if (queryIndex >= 0) path = path.slice(0, queryIndex)
  if (
    !path.startsWith('/')
    || path.startsWith('//')
    || path.includes('\\')
    || path.includes('#')
    || path.includes('..')
    || /[\u0000-\u001f]/.test(path)
  ) {
    return { ok: false, reason: 'path' }
  }
  return { ok: true, method: method as RouteInspectorMethod, path }
}

export function routeInspectorQueryFromRoute(query: Record<string, unknown> | undefined): RouteInspectorLookup {
  const methodRaw = query?.method
  const pathRaw = query?.path
  return {
    method: typeof methodRaw === 'string' ? methodRaw : Array.isArray(methodRaw) ? String(methodRaw[0] || '') : '',
    path: typeof pathRaw === 'string' ? pathRaw : Array.isArray(pathRaw) ? String(pathRaw[0] || '') : ''
  }
}

export function routeInspectorQueryParams(method: string, path: string): { method: string, path: string } {
  const validation = validateRouteInspectorLookup({ method, path })
  if (validation.ok) return { method: validation.method, path: validation.path }
  return { method: method.trim().toUpperCase(), path: path.trim() }
}

export function formatDurationMicros(micros: number): string {
  if (!Number.isFinite(micros) || micros < 0) return '—'
  if (micros < 1000) return `${micros} µs`
  if (micros < 1_000_000) return `${(micros / 1000).toFixed(micros < 10_000 ? 2 : 1)} ms`
  return `${(micros / 1_000_000).toFixed(2)} s`
}

/** Terminal / matched step: last chain entry when resolved. */
export function routeInspectorMatchedStep(snapshot: RouteInspectorSnapshot): RouteInspectorStep | undefined {
  if (snapshot.resolution !== 'resolved' || snapshot.chain.length === 0) return undefined
  return snapshot.chain[snapshot.chain.length - 1]
}

export function routeInspectorErrorKind(cause: unknown):
  | 'permission'
  | 'validation'
  | 'unavailable'
  | 'conflict'
  | 'generic' {
  const reason = apiErrorReason(cause)
  if (reason === 'permission.denied') return 'permission'
  if (reason === 'extensions.route_inspector_invalid') return 'validation'
  if (reason === 'extensions.route_inspector_unavailable') return 'unavailable'
  // 检查器 revision CAS 冲突复用 route_provider_conflict reason。
  if (reason === 'extensions.route_provider_conflict') return 'conflict'
  const status = apiStatusCode(cause)
  if (status === 403) return 'permission'
  if (status === 422) return 'validation'
  if (status === 409) return 'conflict'
  if (status === 503) return 'unavailable'
  return 'generic'
}

function apiStatusCode(error: unknown): number | null {
  if (!error || typeof error !== 'object') return null
  const candidate = error as {
    statusCode?: unknown
    status?: unknown
    response?: { status?: unknown }
    data?: { code?: unknown }
  }
  for (const value of [
    candidate.statusCode,
    candidate.status,
    candidate.response?.status,
    candidate.data?.code
  ]) {
    const n = Number(value)
    if (Number.isInteger(n) && n >= 100 && n < 600) return n
  }
  return null
}

export function useAdminRouteInspector() {
  const { request } = useApiClient()
  const basePath = '/admin/extensions/route-inspector'

  const inspect = async (method: string, path: string) => {
    const validation = validateRouteInspectorLookup({ method, path })
    if (!validation.ok) {
      throw Object.assign(new Error('route inspector lookup invalid'), {
        data: {
          code: 422,
          message: 'invalid lookup',
          data: { reason: 'extensions.route_inspector_invalid' }
        }
      })
    }
    const query = new URLSearchParams({
      method: validation.method,
      path: validation.path
    })
    const raw = await request<unknown>(`${basePath}?${query.toString()}`)
    const snapshot = parseRouteInspectorSnapshot(raw)
    if (!snapshot) {
      throw Object.assign(new Error('route inspector response invalid'), {
        data: {
          code: 503,
          message: 'invalid inspector response',
          data: { reason: 'extensions.route_inspector_unavailable' }
        }
      })
    }
    return snapshot
  }

  return { inspect }
}
