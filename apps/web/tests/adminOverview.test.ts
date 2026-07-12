import { describe, expect, test } from 'bun:test'

import {
  formatOverviewBytes,
  formatOverviewCount,
  formatOverviewDate,
  overviewActionTone,
  overviewPercent,
  overviewTrendDateLabel,
  overviewTrendDeltaPercent,
  overviewTrendFieldMax,
  overviewTrendMax,
  overviewTrendPeakDate,
  overviewTrendSparkPath,
  overviewTrendSum,
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

  test('builds independent sparkline series helpers for 01D trend cards', () => {
    const days: AdminOverviewTrendDay[] = [
      { date: '2026-07-02', topicCount: 0, commentCount: 0, userCount: 0 },
      { date: '2026-07-03', topicCount: 3, commentCount: 8, userCount: 1 },
      { date: '2026-07-04', topicCount: 2, commentCount: 5, userCount: 4 }
    ]

    expect(overviewTrendFieldMax(days, 'commentCount')).toBe(8)
    expect(overviewTrendFieldMax([], 'topicCount')).toBe(1)
    expect(overviewTrendSum(days, 'userCount')).toBe(5)
    expect(overviewTrendDeltaPercent(4, 1)).toBe(300)
    expect(overviewTrendDeltaPercent(0, 0)).toBe(0)
    expect(overviewTrendDeltaPercent(5, 0)).toBe(100)
    expect(overviewTrendPeakDate(days, 'commentCount')).toBe('2026-07-03')
    expect(overviewTrendDateLabel('2026-07-03')).toBe('07-03')

    const spark = overviewTrendSparkPath([0, 3, 2], 280, 72, 8)
    expect(spark.points).toHaveLength(3)
    expect(spark.line.startsWith('M ')).toBe(true)
    expect(spark.area.includes('Z')).toBe(true)
    // 独立刻度：用户峰值 4 的 y 应接近顶部 pad，而不是被回复量级压扁
    const userSpark = overviewTrendSparkPath([0, 1, 4], 280, 72, 8)
    expect(userSpark.points[2].y).toBeLessThan(userSpark.points[0].y)
  })

  test('maps action severity to visual tone', () => {
    expect(overviewActionTone('danger')).toBe('danger')
    expect(overviewActionTone('warning')).toBe('warning')
    expect(overviewActionTone('info')).toBe('info')
    expect(overviewActionTone('unknown')).toBe('neutral')
  })
})
