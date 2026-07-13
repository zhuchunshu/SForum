import type { SForumAdminToastInput } from './host'

export const ADMIN_SDK_API_VERSION = 1 as const
export const ADMIN_MICRO_FRONTEND_API_VERSION = 1 as const

export { useSForumAdminHost } from './host'
export type { SForumAdminHost, SForumAdminToastInput } from './host'

export type AdminJobContext = Readonly<{ job: Record<string, unknown> }>

export type AdminExtensionSettingItem = Readonly<{
  key: string
  label: string
  description?: string
  type: string
  default: string
  value: string
  secretSet?: boolean
  placeholder?: string
  recommendedValue?: string
  group?: string
  options?: ReadonlyArray<{ value: string, label: string, description?: string }>
}>

/** 扩展设置页插槽上下文：宿主提供读写与保存，插件只负责 UI 与文案。 */
export type AdminExtensionSettingsContext = Readonly<{
  extensionId: string
  extensionVersion?: string
  items: readonly AdminExtensionSettingItem[]
  values: Readonly<Record<string, string>>
  loading: boolean
  saving: boolean
  recommendedApplied: boolean
  updateValue: (key: string, value: string) => void
  save: () => Promise<void>
  reset: () => Promise<void>
  openMailCenter?: () => Promise<void>
}>

export type AdminMicroFrontendAppearance = Readonly<{
  colorMode: 'light' | 'dark'
  accent: string
  accentContrast: string
}>

export type AdminMicroFrontendBridgeV1 = Readonly<{
  apiVersion: typeof ADMIN_MICRO_FRONTEND_API_VERSION
  extensionId: string
  extensionVersion: string
  locale: string
  appearance: AdminMicroFrontendAppearance
  settings: Readonly<{
    items: readonly AdminExtensionSettingItem[]
    values: () => Readonly<Record<string, string>>
    updateValue: (key: string, value: string) => void
    save: () => Promise<void>
    reset: () => Promise<void>
  }>
  request: <T>(path: string, options?: Record<string, unknown>) => Promise<T>
  toast: (input: SForumAdminToastInput) => void
  t: (key: string, params?: Record<string, unknown>) => string
  navigate: (adminPath: string) => Promise<void>
}>

export type AdminMicroFrontendCleanup = () => void | Promise<void>

export type AdminMicroFrontendModuleV1 = Readonly<{
  apiVersion: typeof ADMIN_MICRO_FRONTEND_API_VERSION
  mount: (
    target: HTMLElement,
    bridge: AdminMicroFrontendBridgeV1
  ) => void | AdminMicroFrontendCleanup | Promise<void | AdminMicroFrontendCleanup>
}>

export interface AdminSlotContextMap {
  'admin.jobs.table.columns': AdminJobContext
  'admin.jobs.row.actions': AdminJobContext
  'admin.jobs.detail.sections': AdminJobContext
  'admin.extension.settings.page': AdminExtensionSettingsContext
  'admin.extension.settings.header': AdminExtensionSettingsContext
  'admin.extension.settings.footer': AdminExtensionSettingsContext
}

export interface AdminSlotOptionsMap {
  'admin.jobs.table.columns': Readonly<{ width?: number }>
  'admin.jobs.row.actions': Readonly<Record<string, never>>
  'admin.jobs.detail.sections': Readonly<Record<string, never>>
  'admin.extension.settings.page': Readonly<Record<string, never>>
  'admin.extension.settings.header': Readonly<Record<string, never>>
  'admin.extension.settings.footer': Readonly<Record<string, never>>
}

export type AdminSlotPoint = keyof AdminSlotContextMap & string

export type AdminSlotProps<P extends AdminSlotPoint> = Readonly<{
  context: AdminSlotContextMap[P]
  options: AdminSlotOptionsMap[P]
  extensionId: string
  contributionId: string
}>
