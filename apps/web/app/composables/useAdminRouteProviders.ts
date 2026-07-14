export type RouteProviderKey = {
  targetRouteId: string
  targetContractVersion: string
  method: string
  pathSignature: string
}

export type RouteProviderArtifact = {
  extensionId: string
  extensionVersion: string
  packageDigest: string
  runtimeInstanceId: string
}

export type RouteProviderCandidate = {
  routeId: string
  contractVersion: string
  action: string
  targetRouteId?: string
  method: string
  path: string
  pathSignature: string
  priority: number
  providerKind: 'core' | 'plugin'
  artifact?: RouteProviderArtifact
  guard?: string
  permission?: string
  fallback?: string
  mode?: string
  destination?: string
  handler?: string
  requestSchema?: string
  responseSchema?: string
  timeoutMs?: number
}

export type RouteProviderSelection = {
  key: RouteProviderKey
  routeId: string
  contractVersion: string
  extensionId: string
  extensionVersionId: number
  extensionVersion: string
  packageDigest: string
  selectedByUserId: number
  selectionAuditEventId: number
  revision: number
  selectedAt: string
  updatedAt: string
}

export type RouteProviderConflict = {
  key: RouteProviderKey
  candidates: RouteProviderCandidate[]
  selectionStatus: 'selected' | 'unselected' | 'stale'
  selection?: RouteProviderSelection
}

export type SelectRouteProviderInput = RouteProviderKey & {
  providerRouteId: string
  providerContractVersion: string
  providerArtifact: RouteProviderArtifact
  expectedRevision: number
}

export type ResetRouteProviderInput = RouteProviderKey & {
  expectedRevision: number
  reasonCode?: string
}

const CORE_GUARDS = new Set([
  'core.guard.public',
  'core.guard.login',
  'core.guard.permission',
  'core.guard.guest',
  'core.guard.inherit'
])

export function routeProviderConflictId(key: RouteProviderKey) {
  return [key.targetRouteId, key.targetContractVersion, key.method, key.pathSignature].join('\u0000')
}

export function routeProviderRisk(candidate: RouteProviderCandidate) {
  const guard = candidate.guard || ''
  return {
    rawRequest: guard === 'core.guard.raw_request',
    customGuard: Boolean(guard && !CORE_GUARDS.has(guard) && guard !== 'core.guard.raw_request'),
    replacementHandler: candidate.action === 'replace' && Boolean(candidate.handler)
  }
}

export function useAdminRouteProviders() {
  const { request } = useApiClient()
  const basePath = '/admin/extensions/route-providers'

  const listConflicts = () => request<RouteProviderConflict[]>(`${basePath}/conflicts`)

  const selectProvider = (input: SelectRouteProviderInput) =>
    request<RouteProviderSelection>(`${basePath}/selection`, {
      method: 'POST',
      body: input
    })

  const resetProvider = (input: ResetRouteProviderInput) =>
    request<{ reset: true, key: RouteProviderKey }>(`${basePath}/selection/reset`, {
      method: 'POST',
      body: input
    })

  return { listConflicts, selectProvider, resetProvider }
}
