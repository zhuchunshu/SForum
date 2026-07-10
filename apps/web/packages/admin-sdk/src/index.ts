export const ADMIN_SDK_API_VERSION = 1 as const

export { useSForumAdminHost } from './host'
export type { SForumAdminHost, SForumAdminToastInput } from './host'

export type AdminJobContext = Readonly<{ job: Record<string, unknown> }>

export interface AdminSlotContextMap {
  'admin.jobs.table.columns': AdminJobContext
  'admin.jobs.row.actions': AdminJobContext
  'admin.jobs.detail.sections': AdminJobContext
}

export interface AdminSlotOptionsMap {
  'admin.jobs.table.columns': Readonly<{ width?: number }>
  'admin.jobs.row.actions': Readonly<Record<string, never>>
  'admin.jobs.detail.sections': Readonly<Record<string, never>>
}

export type AdminSlotPoint = keyof AdminSlotContextMap & string

export type AdminSlotProps<P extends AdminSlotPoint> = Readonly<{
  context: AdminSlotContextMap[P]
  options: AdminSlotOptionsMap[P]
  extensionId: string
  contributionId: string
}>
