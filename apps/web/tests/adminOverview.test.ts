import { describe, expect, test } from 'bun:test'

import {
  formatOverviewBytes,
  formatOverviewCount,
  formatOverviewDate,
  overviewActionTone,
  overviewPercent,
  overviewTrendMax,
  type AdminOverviewTrendDay
} from '../app/utils/adminOverview'

describe('admin overview helpers', () => {
  test('formats memory bytes as compact MiB labels', () => {
    expect(formatOverviewBytes(0)).toBe('0 MiB')
    expect(formatOverviewBytes(1024 * 1024)).toBe('1 MiB')
    expect(formatOverviewBytes(1536 * 1024 * 1024)).toBe('1,536 MiB')
  })

  test('formats large counts for dashboard cards', () => {
    expect(formatOverviewCount(999)).toBe('999')
    expect(formatOverviewCount(1200)).toBe('1.2k')
    expect(formatOverviewCount(1_250_000)).toBe('1.3m')
  })

  test('formats generated timestamps without locale-dependent output', () => {
    const localDate = new Date(2026, 6, 8, 2, 54, 29)
    expect(formatOverviewDate(localDate.toISOString())).toBe('2026-07-08 02:54:29')
    expect(formatOverviewDate('not-a-date')).toBe('')
  })

  test('calculates safe percentages', () => {
    expect(overviewPercent(5, 10)).toBe(50)
    expect(overviewPercent(1, 3)).toBe(33)
    expect(overviewPercent(5, 0)).toBe(0)
  })

  test('finds a non-zero trend maximum for bar charts', () => {
    const days: AdminOverviewTrendDay[] = [
      { date: '2026-07-02', topicCount: 0, commentCount: 0, userCount: 0 },
      { date: '2026-07-03', topicCount: 3, commentCount: 8, userCount: 1 },
      { date: '2026-07-04', topicCount: 2, commentCount: 5, userCount: 4 }
    ]

    expect(overviewTrendMax(days)).toBe(12)
    expect(overviewTrendMax([])).toBe(1)
  })

  test('maps action severity to visual tone', () => {
    expect(overviewActionTone('danger')).toBe('danger')
    expect(overviewActionTone('warning')).toBe('warning')
    expect(overviewActionTone('info')).toBe('info')
    expect(overviewActionTone('unknown')).toBe('neutral')
  })
})
