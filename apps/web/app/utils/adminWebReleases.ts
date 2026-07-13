export type AdminWebReleaseStatus = 'queued' | 'building' | 'verifying' | 'ready' | 'activating' | 'active' | 'inactive' | 'failed' | 'superseded' | 'rolled_back'

export type AdminWebRelease = {
  id: number
  triggerKind: string
  triggerExtensionId?: string
  compositionHash: string
  activeThemeId: string
  themeVersion: string
  status: AdminWebReleaseStatus
  reloadMode: 'prompt' | 'force'
  publicReason?: string
  publicMessage?: string
  previousReleaseId?: number
  createdAt: string
  updatedAt: string
  activatedAt?: string
}

export type AdminWebReleaseEvent = {
  id: number
  previousStatus?: AdminWebReleaseStatus
  nextStatus: AdminWebReleaseStatus
  reason: string
  message?: string
  createdAt: string
}

export type AdminWebReleaseDetail = AdminWebRelease & {
  buildLog?: string
  extensions: Array<{ extensionId: string, extensionVersion: string, packageDigest: string, adminFrontendDigest: string }>
  effects: Array<{ extensionId: string, previousStatus: string, targetStatus: string }>
  events: AdminWebReleaseEvent[]
}

export type AdminWebReleasePage = { items: AdminWebRelease[], total: number, page: number, perPage: number }

export const webReleaseIsFinal = (status: AdminWebReleaseStatus) => ['active', 'inactive', 'failed', 'superseded', 'rolled_back'].includes(status)
export const webReleaseCanRetry = (status: AdminWebReleaseStatus) => status === 'failed' || status === 'superseded'
export const webReleaseCanRollback = (status: AdminWebReleaseStatus) => status === 'inactive' || status === 'rolled_back'

export function webReleaseProgress(status: AdminWebReleaseStatus) {
  return ({ queued: 8, building: 32, verifying: 58, ready: 72, activating: 88, active: 100, inactive: 100, failed: 100, superseded: 100, rolled_back: 100 })[status]
}
