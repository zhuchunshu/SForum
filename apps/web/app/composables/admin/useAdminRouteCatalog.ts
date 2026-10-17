import type { RouteProviderArtifact } from './useAdminRouteProviders'

export type RouteCatalogConflictKind = 'path_method' | 'route_identity' | 'provider_selection'

export type RouteCatalogGuardRisk = 'host_owned' | 'extension_code' | 'raw_request'

export type RouteCatalogRoute = {
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
  guard: string
  permission?: string
  handler?: string
  destination?: string
  mode: string
  fallback: string
  requestSchema?: string
  responseSchema?: string
  timeoutMs: number
  guardRisk: RouteCatalogGuardRisk
  /** 运行时准入：core 恒 true；plugin 制品缺准入时声明保留但不匹配请求。 */
  admitted: boolean
  conflictKinds: RouteCatalogConflictKind[]
}

export type RouteCatalogSnapshot = {
  revision: number
  safeMode: boolean
  total: number
  routes: RouteCatalogRoute[]
}

export type RouteCatalogGroup = {
  key: string
  kind: 'core' | 'plugin'
  /** 插件组为 extensionId；core 组显示名由组件按 kind 用 i18n 解析。 */
  name: string
  version?: string
  packageDigest?: string
  runtimeInstanceId?: string
  routes: RouteCatalogRoute[]
  unadmittedCount: number
  replaceCount: number
}

export const ROUTE_CATALOG_CONFLICT_KINDS = new Set<string>([
  'path_method',
  'route_identity',
  'provider_selection'
])

export const ROUTE_CATALOG_GUARD_RISKS = new Set<string>(['host_owned', 'extension_code', 'raw_request'])

export function routeCatalogDigestShort(digest: string | undefined) {
  if (!digest) return ''
  return digest.length > 16 ? `${digest.slice(0, 6)}…${digest.slice(-4)}` : digest
}

/** 按提供者来源分组：core 固定第一，插件按 extensionId 字典序。 */
export function groupRouteCatalog(routes: RouteCatalogRoute[]): RouteCatalogGroup[] {
  const coreRoutes: RouteCatalogRoute[] = []
  const pluginRoutes = new Map<string, RouteCatalogRoute[]>()
  for (const route of routes) {
    if (route.providerKind === 'core') {
      coreRoutes.push(route)
      continue
    }
    const key = route.artifact?.extensionId || 'unknown'
    const bucket = pluginRoutes.get(key)
    if (bucket) bucket.push(route)
    else pluginRoutes.set(key, [route])
  }
  const groups: RouteCatalogGroup[] = []
  if (coreRoutes.length) {
    groups.push(buildRouteCatalogGroup('core', 'core', coreRoutes))
  }
  const pluginKeys = [...pluginRoutes.keys()].sort((left, right) => left.localeCompare(right))
  for (const key of pluginKeys) {
    groups.push(buildRouteCatalogGroup(key, 'plugin', pluginRoutes.get(key) ?? []))
  }
  return groups
}

function buildRouteCatalogGroup(key: string, kind: 'core' | 'plugin', routes: RouteCatalogRoute[]): RouteCatalogGroup {
  const first = routes[0]
  return {
    key,
    kind,
    name: kind === 'plugin' ? key : 'core',
    version: first?.artifact?.extensionVersion,
    packageDigest: first?.artifact?.packageDigest,
    runtimeInstanceId: first?.artifact?.runtimeInstanceId,
    routes,
    unadmittedCount: routes.filter(route => !route.admitted).length,
    replaceCount: routes.filter(route => route.action === 'replace').length
  }
}

export function routeCatalogStats(snapshot: RouteCatalogSnapshot) {
  return {
    total: snapshot.routes.length,
    pluginProviders: new Set(
      snapshot.routes.filter(route => route.providerKind === 'plugin').map(route => route.artifact?.extensionId || 'unknown')
    ).size,
    pluginRoutes: snapshot.routes.filter(route => route.providerKind === 'plugin').length,
    unadmitted: snapshot.routes.filter(route => !route.admitted).length,
    conflicts: snapshot.routes.filter(route => route.conflictKinds.length > 0).length
  }
}

function isRecord(value: unknown): value is Record<string, unknown> {
  return Boolean(value) && typeof value === 'object' && !Array.isArray(value)
}

function parseRouteCatalogRoute(value: unknown): RouteCatalogRoute | undefined {
  if (!isRecord(value)) return undefined
  const artifactValue = value.artifact
  let artifact: RouteProviderArtifact | undefined
  if (isRecord(artifactValue)) {
    const { extensionId, extensionVersion, packageDigest, runtimeInstanceId } = artifactValue
    if (typeof extensionId !== 'string' || typeof extensionVersion !== 'string' ||
      typeof packageDigest !== 'string' || typeof runtimeInstanceId !== 'string') return undefined
    artifact = { extensionId, extensionVersion, packageDigest, runtimeInstanceId }
  }
  const providerKind = value.providerKind
  if (providerKind !== 'core' && providerKind !== 'plugin') return undefined
  if (providerKind === 'plugin' && !artifact) return undefined
  const guardRisk = value.guardRisk
  if (typeof guardRisk !== 'string' || !ROUTE_CATALOG_GUARD_RISKS.has(guardRisk)) return undefined
  if (!Array.isArray(value.conflictKinds)) return undefined
  const conflictKinds: RouteCatalogConflictKind[] = []
  for (const kind of value.conflictKinds) {
    if (typeof kind !== 'string' || !ROUTE_CATALOG_CONFLICT_KINDS.has(kind)) return undefined
    conflictKinds.push(kind as RouteCatalogConflictKind)
  }
  const strings = ['routeId', 'contractVersion', 'action', 'method', 'path', 'pathSignature', 'guard', 'mode', 'fallback'] as const
  for (const key of strings) {
    if (typeof value[key] !== 'string') return undefined
  }
  const priority = value.priority
  const timeoutMs = value.timeoutMs
  if (typeof priority !== 'number' || !Number.isInteger(priority)) return undefined
  if (typeof timeoutMs !== 'number' || !Number.isInteger(timeoutMs) || timeoutMs < 0) return undefined
  if (typeof value.admitted !== 'boolean') return undefined
  const optional = ['targetRouteId', 'permission', 'handler', 'destination', 'requestSchema', 'responseSchema'] as const
  for (const key of optional) {
    if (value[key] !== undefined && typeof value[key] !== 'string') return undefined
  }
  return {
    routeId: value.routeId as string,
    contractVersion: value.contractVersion as string,
    action: value.action as string,
    targetRouteId: value.targetRouteId as string | undefined,
    method: value.method as string,
    path: value.path as string,
    pathSignature: value.pathSignature as string,
    priority,
    providerKind,
    artifact,
    guard: value.guard as string,
    permission: value.permission as string | undefined,
    handler: value.handler as string | undefined,
    destination: value.destination as string | undefined,
    mode: value.mode as string,
    fallback: value.fallback as string,
    requestSchema: value.requestSchema as string | undefined,
    responseSchema: value.responseSchema as string | undefined,
    timeoutMs,
    guardRisk: guardRisk as RouteCatalogGuardRisk,
    admitted: value.admitted === true,
    conflictKinds
  }
}

export function parseRouteCatalogSnapshot(value: unknown): RouteCatalogSnapshot | undefined {
  if (!isRecord(value)) return undefined
  const revision = value.revision
  if (typeof revision !== 'number' || !Number.isInteger(revision) || revision < 0) return undefined
  if (typeof value.safeMode !== 'boolean' || typeof value.total !== 'number' || !Array.isArray(value.routes)) {
    return undefined
  }
  const routes: RouteCatalogRoute[] = []
  for (const item of value.routes) {
    const route = parseRouteCatalogRoute(item)
    if (!route) return undefined
    routes.push(route)
  }
  if (value.total !== routes.length) return undefined
  return { revision, safeMode: value.safeMode, total: routes.length, routes }
}

export function useAdminRouteCatalog() {
  const { request } = useApiClient()
  const basePath = '/admin/extensions/route-catalog'

  const load = async (): Promise<RouteCatalogSnapshot> => {
    const raw = await request<unknown>(basePath)
    const snapshot = parseRouteCatalogSnapshot(raw)
    if (!snapshot) {
      throw Object.assign(new Error('route catalog response invalid'), {
        data: {
          code: 503,
          message: 'invalid catalog response',
          data: { reason: 'extensions.route_catalog_unavailable' }
        }
      })
    }
    return snapshot
  }

  return { load }
}
