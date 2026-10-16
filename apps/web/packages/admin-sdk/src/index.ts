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

/**
 * 宿主图标解析器：插件只提供图标名，SVG 标记始终由宿主生成。
 * 名称语法为 `i-(lucide|tabler)-<name>`，与宿主贡献点校验共用同一条规则。
 * 无法解析的图标名会从结果中缺失，调用方必须容忍缺项。
 */
export type AdminIconResolver = Readonly<{
  resolve: (names: readonly string[]) => Promise<Readonly<Record<string, string>>>
}>

export type AdminMicroFrontendBridgeV1 = Readonly<{
  apiVersion: typeof ADMIN_MICRO_FRONTEND_API_VERSION
  extensionId: string
  extensionVersion: string
  locale: string
  appearance: AdminMicroFrontendAppearance
  /** 可选：宿主未提供时为 `undefined`，插件必须优雅降级。 */
  icons?: AdminIconResolver
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

export type AdminPageDescriptor = Readonly<{
  path: string
  label: string
  description?: string
  icon?: string
}>

export type AdminPageBridgeV1 = Readonly<{
  apiVersion: typeof ADMIN_MICRO_FRONTEND_API_VERSION
  extensionId: string
  extensionVersion: string
  locale: string
  appearance: AdminMicroFrontendAppearance
  /** 可选：宿主未提供时为 `undefined`，插件必须优雅降级。 */
  icons?: AdminIconResolver
  page: AdminPageDescriptor
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

export type AdminPageModuleV1 = Readonly<{
  apiVersion: typeof ADMIN_MICRO_FRONTEND_API_VERSION
  mount: (
    target: HTMLElement,
    bridge: AdminPageBridgeV1
  ) => void | AdminMicroFrontendCleanup | Promise<void | AdminMicroFrontendCleanup>
}>
