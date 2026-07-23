import type {
  ModerationAction,
  ModerationDecision,
  ModerationPendingItem,
  ModerationReportItem,
  ModerationReviewContext,
  ModerationSource,
  ModerationTargetType
} from '~/composables/useModerationApi'

export type ModerationWorkbenchTab = 'pending' | 'reports' | 'history'
export type ModerationWorkbenchTypeFilter = ModerationTargetType | 'all'

export const MODERATION_TABS: ModerationWorkbenchTab[] = ['pending', 'reports', 'history']
export const MODERATION_TYPE_FILTERS: ModerationWorkbenchTypeFilter[] = ['all', 'topic', 'comment']
export const REVIEW_REQUIRED_ACTIONS = new Set<ModerationAction>(['reject', 'hide_and_close', 'delete_and_close'])

export type ModerationReviewSelection = {
  tab: ModerationWorkbenchTab
  source: ModerationSource
  targetType: ModerationTargetType
  targetId: number
  reportId?: number
  decisionId?: number
}

export function firstQueryValue(value: unknown): string {
  return Array.isArray(value) ? String(value[0] || '') : String(value || '')
}

export function parseWorkbenchTab(value: unknown): ModerationWorkbenchTab {
  const raw = firstQueryValue(value)
  return MODERATION_TABS.includes(raw as ModerationWorkbenchTab) ? raw as ModerationWorkbenchTab : 'pending'
}

export function parseTargetType(value: unknown): ModerationWorkbenchTypeFilter {
  const raw = firstQueryValue(value)
  return MODERATION_TYPE_FILTERS.includes(raw as ModerationWorkbenchTypeFilter) ? raw as ModerationWorkbenchTypeFilter : 'all'
}

export function parsePositiveInt(value: unknown, fallback = 1): number {
  const parsed = Number.parseInt(firstQueryValue(value), 10)
  return Number.isFinite(parsed) && parsed > 0 ? parsed : fallback
}

export function tabToSource(tab: ModerationWorkbenchTab): ModerationSource {
  return tab === 'reports' ? 'report' : 'pre_publish'
}

export function parseModerationSource(value: unknown, fallback: ModerationSource): ModerationSource {
  const raw = firstQueryValue(value)
  return raw === 'pre_publish' || raw === 'report' ? raw : fallback
}

export function itemTargetType(item: ModerationPendingItem | ModerationReportItem | ModerationDecision): ModerationTargetType {
  return item.targetType
}

export function itemTargetId(item: ModerationPendingItem | ModerationReportItem | ModerationDecision): number {
  return item.targetId
}

export function itemReportId(item: ModerationPendingItem | ModerationReportItem | ModerationDecision): number | undefined {
  if ('reasonCode' in item) return item.id
  if ('reportId' in item && typeof item.reportId === 'number') return item.reportId
  return undefined
}

export function itemDecisionId(item: ModerationDecision): number {
  return item.id
}

export function queueItemKey(tab: ModerationWorkbenchTab, item: ModerationPendingItem | ModerationReportItem | ModerationDecision): string {
  if (tab === 'history' && 'id' in item && 'action' in item) return `history:${item.id}`
  if ('reasonCode' in item) return `report:${item.id}:${item.targetType}:${item.targetId}`
  return `pending:${item.targetType}:${item.targetId}`
}

export function selectionKey(selection: ModerationReviewSelection | null): string {
  if (!selection) return ''
  return selection.decisionId
    ? `history:${selection.decisionId}`
    : `${selection.source}:${selection.reportId || 0}:${selection.targetType}:${selection.targetId}`
}

export function selectionFromQuery(query: Record<string, unknown>, tab: ModerationWorkbenchTab): ModerationReviewSelection | null {
  const reviewType = firstQueryValue(query.reviewType)
  const reviewId = parsePositiveInt(query.reviewId, 0)
  if ((reviewType !== 'topic' && reviewType !== 'comment') || reviewId <= 0) return null
  const source = tab === 'history'
    ? parseModerationSource(query.source, tabToSource(tab))
    : tabToSource(tab)

  return {
    tab,
    source,
    targetType: reviewType,
    targetId: reviewId,
    reportId: parsePositiveInt(query.reportId, 0) || undefined,
    decisionId: parsePositiveInt(query.decisionId, 0) || undefined
  }
}

export function selectionFromQueueItem(tab: ModerationWorkbenchTab, item: ModerationPendingItem | ModerationReportItem | ModerationDecision): ModerationReviewSelection {
  const source = 'source' in item && tab === 'history' ? item.source : tabToSource(tab)
  return {
    tab,
    source,
    targetType: itemTargetType(item),
    targetId: itemTargetId(item),
    reportId: itemReportId(item),
    decisionId: tab === 'history' && 'action' in item ? itemDecisionId(item) : undefined
  }
}

export function reviewQuery(selection: ModerationReviewSelection): Record<string, string | undefined> {
  return {
    source: selection.source,
    reviewType: selection.targetType,
    reviewId: String(selection.targetId),
    reportId: selection.reportId ? String(selection.reportId) : undefined,
    decisionId: selection.decisionId ? String(selection.decisionId) : undefined
  }
}

export function queueQueryForWorkbench(
  query: Record<string, unknown>,
  overrides: Record<string, string | number | undefined> = {}
): Record<string, string | number | undefined> {
  const next: Record<string, string | number | undefined> = {}
  for (const [key, value] of Object.entries(query)) {
    const normalized = firstQueryValue(value)
    if (normalized) next[key] = normalized
  }

  Object.assign(next, {
    source: undefined,
    reviewType: undefined,
    reviewId: undefined,
    reportId: undefined,
    decisionId: undefined
  }, overrides)

  const normalizedTab = parseWorkbenchTab(next.tab)
  const normalizedType = parseTargetType(next.targetType)
  const normalizedPage = parsePositiveInt(next.page, 1)
  next.tab = normalizedTab === 'pending' ? undefined : normalizedTab
  next.targetType = normalizedType === 'all' ? undefined : normalizedType
  next.page = normalizedPage <= 1 ? undefined : normalizedPage

  return next
}

export function actionListForContext(context: ModerationReviewContext | null, readonly = false): ModerationAction[] {
  if (!context || readonly) return []
  return context.source === 'pre_publish'
    ? ['approve', 'reject']
    : ['keep_and_close', 'hide_and_close', 'delete_and_close']
}

export function isDecisionReadonly(tab: ModerationWorkbenchTab): boolean {
  return tab === 'history'
}
