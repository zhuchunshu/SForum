export const ADMIN_MICRO_FRONTEND_API_VERSION = 1 as const

export type SForumAdminToastInput = Readonly<{
  title: string
  description?: string
  kind?: 'success' | 'error'
}>

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
