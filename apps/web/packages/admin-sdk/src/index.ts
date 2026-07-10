export const ADMIN_SDK_API_VERSION = 1 as const

export { useSForumAdminHost } from './host'
export type { SForumAdminHost, SForumAdminToastInput } from './host'

export interface AdminSlotContextMap {}
export interface AdminSlotOptionsMap {}

export type AdminSlotPoint = keyof AdminSlotContextMap & string

export type AdminSlotProps<P extends AdminSlotPoint> = Readonly<{
  context: AdminSlotContextMap[P]
  options: AdminSlotOptionsMap[P]
  extensionId: string
  contributionId: string
}>
