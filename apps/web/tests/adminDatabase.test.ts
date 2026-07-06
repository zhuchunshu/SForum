import { describe, expect, test } from 'bun:test'

import {
  buildDatabaseRowsQuery,
  databaseSensitiveMask,
  databaseValueText,
  isDatabaseCellRevealable,
  replaceDatabaseCellValue,
  type DatabaseRow
} from '../app/utils/adminDatabase'

describe('admin database helpers', () => {
  test('builds row query parameters for paging, sorting, and filtering', () => {
    const query = buildDatabaseRowsQuery({
      page: 2,
      perPage: 25,
      sort: 'username',
      direction: 'desc',
      filterColumn: 'email',
      filterOperator: 'contains',
      filterValue: 'example.com'
    })

    expect(query).toBe('page=2&perPage=25&sort=username&direction=desc&filterColumn=email&filterOperator=contains&filterValue=example.com')
  })

  test('omits empty filter values while keeping null filters', () => {
    const query = buildDatabaseRowsQuery({
      page: 1,
      perPage: 50,
      filterColumn: 'deleted_at',
      filterOperator: 'is_null'
    })

    expect(query).toBe('page=1&perPage=50&filterColumn=deleted_at&filterOperator=is_null')
  })

  test('identifies revealable masked sensitive cells', () => {
    expect(isDatabaseCellRevealable({ value: databaseSensitiveMask, sensitive: true, masked: true })).toBe(true)
    expect(isDatabaseCellRevealable({ value: 'plain', sensitive: false, masked: false })).toBe(false)
  })

  test('replaces one revealed cell without mutating the original row', () => {
    const row: DatabaseRow = {
      rowKey: 'key',
      values: {
        id: { value: 1, sensitive: false, masked: false },
        api_token: { value: databaseSensitiveMask, sensitive: true, masked: true }
      }
    }

    const updated = replaceDatabaseCellValue(row, 'api_token', 'secret-token')

    expect(updated.values.api_token).toEqual({ value: 'secret-token', sensitive: true, masked: false })
    expect(row.values.api_token).toEqual({ value: databaseSensitiveMask, sensitive: true, masked: true })
  })

  test('formats database values for compact display', () => {
    expect(databaseValueText(null)).toBe('NULL')
    expect(databaseValueText(undefined)).toBe('NULL')
    expect(databaseValueText(true)).toBe('true')
    expect(databaseValueText({ nested: true })).toBe('{"nested":true}')
  })
})
