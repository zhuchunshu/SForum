import { apiErrorMessage, apiErrorReason } from './useApiClient'

export type ComponentCompositionInspectorSnapshot = {
  revision: number
  safeMode: boolean
  targetCount: number
  contributionCount: number
  conflicts: Array<Record<string, unknown>>
  traces: Array<Record<string, unknown>>
}

export type NavigationInspectorSnapshot = {
  revision: number
  digest: string
  safeMode: boolean
  navigationCount: number
  regionCount: number
  providerConflicts: number
  traces: Array<Record<string, unknown>>
}

function asObject(value: unknown): Record<string, unknown> | null {
  return value && typeof value === 'object' && !Array.isArray(value)
    ? value as Record<string, unknown>
    : null
}

function asNumber(value: unknown): number {
  return typeof value === 'number' && Number.isFinite(value) ? value : 0
}

function asBoolean(value: unknown): boolean {
  return value === true
}

function asString(value: unknown): string {
  return typeof value === 'string' ? value : ''
}

function asObjectArray(value: unknown): Array<Record<string, unknown>> {
  if (!Array.isArray(value)) return []
  return value.filter((item): item is Record<string, unknown> => !!item && typeof item === 'object' && !Array.isArray(item))
}

/** 接受 [] 或 null/缺省（兼容旧后端把空 slice 编成 null）。 */
function asOptionalObjectArray(value: unknown): Array<Record<string, unknown>> | null {
  if (value == null) return []
  if (!Array.isArray(value)) return null
  return asObjectArray(value)
}

/** 解析组件组合检查器响应；形状非法时返回 null（不抛异常）。 */
export function parseComponentCompositionSnapshot(raw: unknown): ComponentCompositionInspectorSnapshot | null {
  const data = asObject(raw)
  if (!data) return null
  // 数值字段必须为有限数字，避免把字符串 revision 当成 0 误报成功。
  if (typeof data.revision !== 'number' || !Number.isFinite(data.revision)) return null
  if (typeof data.targetCount !== 'number' || !Number.isFinite(data.targetCount)) return null
  if (typeof data.contributionCount !== 'number' || !Number.isFinite(data.contributionCount)) return null
  if (typeof data.safeMode !== 'boolean') return null
  const conflicts = asOptionalObjectArray(data.conflicts)
  const traces = asOptionalObjectArray(data.traces)
  if (conflicts == null || traces == null) return null
  return {
    revision: asNumber(data.revision),
    safeMode: asBoolean(data.safeMode),
    targetCount: asNumber(data.targetCount),
    contributionCount: asNumber(data.contributionCount),
    conflicts,
    traces
  }
}

/** 解析导航检查器响应；形状非法时返回 null。 */
export function parseNavigationInspectorSnapshot(raw: unknown): NavigationInspectorSnapshot | null {
  const data = asObject(raw)
  if (!data) return null
  if (typeof data.revision !== 'number' || !Number.isFinite(data.revision)) return null
  if (typeof data.digest !== 'string') return null
  if (typeof data.safeMode !== 'boolean') return null
  if (typeof data.navigationCount !== 'number' || !Number.isFinite(data.navigationCount)) return null
  if (typeof data.regionCount !== 'number' || !Number.isFinite(data.regionCount)) return null
  if (typeof data.providerConflicts !== 'number' || !Number.isFinite(data.providerConflicts)) return null
  const traces = asOptionalObjectArray(data.traces)
  if (traces == null) return null
  return {
    revision: asNumber(data.revision),
    digest: asString(data.digest),
    safeMode: asBoolean(data.safeMode),
    navigationCount: asNumber(data.navigationCount),
    regionCount: asNumber(data.regionCount),
    providerConflicts: asNumber(data.providerConflicts),
    traces
  }
}

function clampInspectorLimit(limit: number): number {
  if (!Number.isInteger(limit) || limit < 1 || limit > 200) {
    throw new Error('inspector limit must be an integer from 1 to 200')
  }
  return limit
}

export function useAdminCompositionInspectors() {
  const { request } = useApiClient()

  const inspectComponents = async (limit = 50) => {
    const safeLimit = clampInspectorLimit(limit)
    try {
      const raw = await request<unknown>(`/admin/extensions/component-inspector?limit=${safeLimit}`)
      const snapshot = parseComponentCompositionSnapshot(raw)
      if (!snapshot) {
        throw new Error('component inspector response invalid')
      }
      return snapshot
    } catch (error) {
      // 保留 plain Error.message；不要用空 envelope 覆盖成无意义的 failed。
      const message =
        apiErrorMessage(error)
        || apiErrorReason(error)
        || (error instanceof Error && error.message.trim() ? error.message : '')
        || 'component inspector failed'
      throw new Error(message)
    }
  }

  const inspectNavigation = async (limit = 50) => {
    const safeLimit = clampInspectorLimit(limit)
    try {
      const raw = await request<unknown>(`/admin/extensions/navigation-inspector?limit=${safeLimit}`)
      const snapshot = parseNavigationInspectorSnapshot(raw)
      if (!snapshot) {
        throw new Error('navigation inspector response invalid')
      }
      return snapshot
    } catch (error) {
      const message =
        apiErrorMessage(error)
        || apiErrorReason(error)
        || (error instanceof Error && error.message.trim() ? error.message : '')
        || 'navigation inspector failed'
      throw new Error(message)
    }
  }

  return { inspectComponents, inspectNavigation }
}
