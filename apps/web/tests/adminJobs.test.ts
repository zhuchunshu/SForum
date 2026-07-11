import { describe, expect, test } from 'bun:test'
import { readFileSync } from 'node:fs'
import { ALL_ADMIN_JOBS_FILTER, adminJobFilterValue } from '../app/utils/adminJobs'

const page = readFileSync(new URL('../app/pages/admin/jobs.vue', import.meta.url), 'utf8')
const composable = readFileSync(new URL('../app/composables/useAdminJobs.ts', import.meta.url), 'utf8')

describe('admin jobs filters', () => {
  test('never renders an empty Select item value', () => {
    expect(page).not.toContain("value: ''")
    expect(page).toContain('ALL_ADMIN_JOBS_FILTER')
  })

  test('maps the all-filter sentinel back to an empty API filter', () => {
    expect(ALL_ADMIN_JOBS_FILTER).not.toBe('')
    expect(adminJobFilterValue(ALL_ADMIN_JOBS_FILTER)).toBe('')
    expect(adminJobFilterValue('running')).toBe('running')
    expect(composable).toContain('adminJobFilterValue(filters.queue)')
    expect(composable).toContain('adminJobFilterValue(filters.state)')
  })
})
