import { describe, expect, test } from 'bun:test'
import { readFileSync } from 'node:fs'
import { ALL_ADMIN_JOBS_FILTER, adminJobFilterValue, formatScheduleDateTime, formatScheduleInterval } from '../app/utils/adminJobs'

const page = readFileSync(new URL('../app/pages/admin/jobs.vue', import.meta.url), 'utf8')
const schedulesPage = readFileSync(new URL('../app/pages/admin/schedules.vue', import.meta.url), 'utf8')
const composable = readFileSync(new URL('../app/composables/useAdminJobs.ts', import.meta.url), 'utf8')
const modules = readFileSync(new URL('../app/config/adminModules.ts', import.meta.url), 'utf8')

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

  test('jobs page links to the dedicated schedules workbench', () => {
    expect(page).toContain("adminRoutes.path('/schedules')")
    expect(page).toContain('admin.jobs.openSchedules')
    expect(page).not.toContain('manager.schedules.data.value')
  })
})

describe('admin schedules workbench', () => {
  test('registers a dedicated admin page under operations', () => {
    expect(modules).toContain("id: '/schedules'")
    expect(modules).toContain('admin.nav.schedules')
    expect(modules).toContain("pageId: '/schedules'")
  })

  test('loads schedules and exposes enable/disable/trigger actions', () => {
    expect(composable).toContain('/admin/jobs/schedules')
    expect(composable).toContain("enabled ? 'enable' : 'disable'")
    expect(composable).toContain('/trigger')
    expect(composable).toContain('setScheduleEnabled')
    expect(composable).toContain('triggerSchedule')
    expect(schedulesPage).toContain('admin.schedules.title')
    expect(schedulesPage).toContain('formatScheduleDateTime')
    expect(schedulesPage).toContain('manager.triggerSchedule')
    expect(schedulesPage).toContain('manager.setScheduleEnabled')
  })
})

describe('formatScheduleInterval', () => {
  test('formats common maintenance intervals', () => {
    expect(formatScheduleInterval(undefined)).toBe('—')
    expect(formatScheduleInterval(86400)).toBe('24h')
    expect(formatScheduleInterval(3600)).toBe('1h')
    expect(formatScheduleInterval(900)).toBe('15m')
    expect(formatScheduleInterval(45)).toBe('45s')
  })
})

describe('formatScheduleDateTime', () => {
  test('renders missing values as dash', () => {
    expect(formatScheduleDateTime(undefined)).toBe('—')
    expect(formatScheduleDateTime('')).toBe('—')
  })
})
