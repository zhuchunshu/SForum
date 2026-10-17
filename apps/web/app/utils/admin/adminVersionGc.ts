// Version GC (admin entry, task book stage 5) — focused types/helpers split
// out of adminExtensions.ts to keep that shared file under its size budget.

// ---- Version GC (admin entry, task book stage 5) ----

export type AdminVersionGCBlockedReason =
  | 'publication_history'
  | 'open_operation'
  | 'within_retention'
  | 'disk_missing'

export interface AdminVersionGCCandidate {
  extensionId: string
  extensionType: 'plugin' | 'theme'
  extensionName: string
  versionId: number
  version: string
  packageDigest: string
  packagePath: string
  installedAt: string
  historyRank: number
  reclaimable: boolean
  blockedReason?: AdminVersionGCBlockedReason
}

export interface AdminVersionGCPolicy {
  retain: number
  retainDays: number
}

export interface AdminVersionGCConfirmation {
  token: string
  planDigest: string
  candidateCount: number
  expiresAt: string
}

export interface AdminVersionGCSweepResult {
  removed: AdminVersionGCCandidate[]
  skipped: { candidate: AdminVersionGCCandidate; reason: string }[]
  failed: { candidate: AdminVersionGCCandidate; error: string }[]
  warnings?: string[]
  bytesFreed: number
  candidateCount: number
}

export function versionGcReclaimable(candidates: AdminVersionGCCandidate[]) {
  return candidates.filter(item => item.reclaimable)
}

export function versionGcBlockedSummary(candidates: AdminVersionGCCandidate[]) {
  const summary = new Map<AdminVersionGCBlockedReason, number>()
  for (const item of candidates) {
    if (!item.reclaimable && item.blockedReason) {
      summary.set(item.blockedReason, (summary.get(item.blockedReason) || 0) + 1)
    }
  }
  return summary
}
