export type SPluginSelectOption = Readonly<{
  value: string
  label: string
  disabled?: boolean
}>

export type SPluginTableColumn = Readonly<{
  key: string
  label: string
  align?: 'start' | 'center' | 'end'
  /** 表头图标：宿主图标名，例如 `i-lucide-hash`；解析失败时静默隐藏。 */
  icon?: string
}>

export type SPluginTableRow = Readonly<Record<string, unknown>>
