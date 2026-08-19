export type SPluginSelectOption = Readonly<{
  value: string
  label: string
  disabled?: boolean
}>

export type SPluginTableColumn = Readonly<{
  key: string
  label: string
  align?: 'start' | 'center' | 'end'
}>

export type SPluginTableRow = Readonly<Record<string, unknown>>
