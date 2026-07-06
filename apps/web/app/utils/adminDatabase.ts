export const databaseSensitiveMask = '••••••••'

export type DatabaseTableSummary = {
  schema: string
  name: string
  kind: string
  estimatedRows: number
  sizeBytes: number
  comment: string
}

export type DatabaseColumn = {
  name: string
  dataType: string
  nullable: boolean
  defaultValue: string
  isPrimaryKey: boolean
  isSensitive: boolean
}

export type DatabaseIndex = {
  name: string
  unique: boolean
  primary: boolean
  definition: string
}

export type DatabaseConstraint = {
  name: string
  type: string
  definition: string
}

export type DatabaseTableDetail = Pick<DatabaseTableSummary, 'schema' | 'name' | 'kind'> & {
  columns: DatabaseColumn[]
  primaryKey: string[]
  indexes: DatabaseIndex[]
  constraints: DatabaseConstraint[]
}

export type DatabaseCell = {
  value: unknown
  sensitive: boolean
  masked: boolean
}

export type DatabaseRow = {
  rowKey?: string
  values: Record<string, DatabaseCell>
}

export type DatabaseRows = {
  columns: DatabaseColumn[]
  rows: DatabaseRow[]
  page: number
  perPage: number
  hasNext: boolean
}

export type DatabaseReveal = {
  schema: string
  table: string
  column: string
  value: unknown
}

export type DatabaseRowsQuery = {
  page?: number
  perPage?: number
  sort?: string
  direction?: 'asc' | 'desc' | ''
  filterColumn?: string
  filterOperator?: 'eq' | 'contains' | 'is_null' | 'not_null' | ''
  filterValue?: string
}

export function buildDatabaseRowsQuery(input: DatabaseRowsQuery) {
  const params = new URLSearchParams()
  if (input.page) {
    params.set('page', String(input.page))
  }
  if (input.perPage) {
    params.set('perPage', String(input.perPage))
  }
  appendIfPresent(params, 'sort', input.sort)
  appendIfPresent(params, 'direction', input.direction)
  appendIfPresent(params, 'filterColumn', input.filterColumn)
  appendIfPresent(params, 'filterOperator', input.filterOperator)
  if (input.filterOperator !== 'is_null' && input.filterOperator !== 'not_null') {
    appendIfPresent(params, 'filterValue', input.filterValue)
  }
  return params.toString()
}

export function isDatabaseCellRevealable(cell?: DatabaseCell) {
  return Boolean(cell?.sensitive && cell.masked)
}

export function replaceDatabaseCellValue(row: DatabaseRow, column: string, value: unknown): DatabaseRow {
  const current = row.values[column]
  if (!current) {
    return row
  }

  return {
    ...row,
    values: {
      ...row.values,
      [column]: {
        ...current,
        value,
        masked: false
      }
    }
  }
}

export function databaseValueText(value: unknown) {
  if (value === null || value === undefined) {
    return 'NULL'
  }
  if (typeof value === 'string') {
    return value
  }
  if (typeof value === 'number' || typeof value === 'boolean' || typeof value === 'bigint') {
    return String(value)
  }
  return JSON.stringify(value)
}

export function databaseTableKey(table: Pick<DatabaseTableSummary, 'schema' | 'name'>) {
  return `${table.schema}.${table.name}`
}

function appendIfPresent(params: URLSearchParams, key: string, value?: string) {
  const normalized = String(value || '').trim()
  if (normalized) {
    params.set(key, normalized)
  }
}
