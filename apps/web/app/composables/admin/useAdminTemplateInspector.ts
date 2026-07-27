import { apiErrorMessage, apiErrorReason } from '../useApiClient'

export type TemplateInspectorItem = {
  extensionId: string
  extensionVersion: string
  packageDigest: string
  kind: 'theme' | 'plugin' | string
  contributionIds: string[]
  overrideTargets: string[]
  active: boolean
  default: boolean
}

export type TemplateInspectorSnapshot = {
  schemaVersion: string
  revision: number
  activeTheme: string
  defaultTheme: string
  snapshotCount: number
  overrideCount: number
  snapshots: TemplateInspectorItem[]
}

const TEMPLATE_INSPECTOR_SCHEMA = 'sforum.template-inspector@1'

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

function parseItem(raw: unknown): TemplateInspectorItem | null {
  const data = asObject(raw)
  if (!data) return null
  if (typeof data.extensionId !== 'string' || typeof data.extensionVersion !== 'string') return null
  if (typeof data.packageDigest !== 'string' || typeof data.kind !== 'string') return null
  return {
    extensionId: data.extensionId,
    extensionVersion: data.extensionVersion,
    packageDigest: data.packageDigest,
    kind: data.kind,
    contributionIds: asStringArray(data.contributionIds),
    overrideTargets: asStringArray(data.overrideTargets),
    active: asBoolean(data.active),
    default: asBoolean(data.default)
  }
}

/** 解析模板检查器响应；形状非法时返回 null（不抛异常）。 */
export function parseTemplateInspectorSnapshot(raw: unknown): TemplateInspectorSnapshot | null {
  const data = asObject(raw)
  if (!data) return null
  if (data.schemaVersion !== TEMPLATE_INSPECTOR_SCHEMA) return null
  if (typeof data.revision !== 'number' || !Number.isFinite(data.revision)) return null
  if (typeof data.snapshotCount !== 'number' || !Number.isFinite(data.snapshotCount)) return null
  if (typeof data.overrideCount !== 'number' || !Number.isFinite(data.overrideCount)) return null
  if (!Array.isArray(data.snapshots)) return null
  const snapshots: TemplateInspectorItem[] = []
  for (const item of data.snapshots) {
    const parsed = parseItem(item)
    if (!parsed) return null
    snapshots.push(parsed)
  }
  return {
    schemaVersion: TEMPLATE_INSPECTOR_SCHEMA,
    revision: data.revision,
    activeTheme: asString(data.activeTheme),
    defaultTheme: asString(data.defaultTheme),
    snapshotCount: data.snapshotCount,
    overrideCount: data.overrideCount,
    snapshots
  }
}

function clampInspectorLimit(limit: number): number {
  if (!Number.isInteger(limit) || limit < 1 || limit > 200) {
    throw new Error('inspector limit must be an integer from 1 to 200')
  }
  return limit
}

export function useAdminTemplateInspector() {
  const { request } = useApiClient()

  const inspect = async (limit = 50) => {
    const safeLimit = clampInspectorLimit(limit)
    try {
      const raw = await request<unknown>(`/admin/extensions/template-inspector?limit=${safeLimit}`)
      const snapshot = parseTemplateInspectorSnapshot(raw)
      if (!snapshot) {
        throw new Error('template inspector response invalid')
      }
      return snapshot
    } catch (error) {
      const message =
        apiErrorMessage(error)
        || apiErrorReason(error)
        || (error instanceof Error && error.message.trim() ? error.message : '')
        || 'template inspector failed'
      throw new Error(message)
    }
  }

  return { inspect }
}
