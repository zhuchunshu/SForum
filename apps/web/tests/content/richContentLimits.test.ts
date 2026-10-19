import { describe, expect, test } from 'bun:test'
import {
  RICH_CHART_LIMITS,
  RICH_FENCE_LIMITS,
  RICH_MATH_LIMITS
} from '../../app/utils/content/richContentLimits'

// 权威数据是仓库根的 fixture；本用例逐字段比对 TS 镜像，改一侧不改另一侧直接红。
const fixtureUrl = new URL('../../../../contracts/fixtures/rich-content-v1.json', import.meta.url)
const fixture = await Bun.file(fixtureUrl).json() as {
  schemaVersion: string
  fences: Array<{ language: string, kind: string, maxSourceBytes: number, maxPerDocument: number }>
  math: { maxSourceBytes: number, maxPerDocument: number, maxExpand: number }
  chart: {
    types: string[]
    maxSeries: number
    maxLabels: number
    maxLabelChars: number
    maxTitleChars: number
    maxAbsValue: number
    maxSourceBytes: number
  }
  fallbackReasons: string[]
}

describe('rich content limits mirror the frozen contract', () => {
  test('loads the shared fixture', () => {
    expect(fixture.schemaVersion).toBe('sforum.rich-content@1')
    expect(fixture.fences.length).toBe(RICH_FENCE_LIMITS.length)
  })

  test('mirrors every fence entry field by field', () => {
    expect(fixture.fences.map(fence => fence.language)).toEqual(
      RICH_FENCE_LIMITS.map(limit => limit.language)
    )

    for (const fence of fixture.fences) {
      const limit = RICH_FENCE_LIMITS.find(entry => entry.language === fence.language)
      expect(limit).toBeDefined()
      expect(limit!.kind).toBe(fence.kind)
      expect(limit!.maxSourceBytes).toBe(fence.maxSourceBytes)
      expect(limit!.maxPerDocument).toBe(fence.maxPerDocument)
    }
  })

  test('mirrors the math limits', () => {
    expect(RICH_MATH_LIMITS.maxSourceBytes).toBe(fixture.math.maxSourceBytes)
    expect(RICH_MATH_LIMITS.maxPerDocument).toBe(fixture.math.maxPerDocument)
    expect(RICH_MATH_LIMITS.maxExpand).toBe(fixture.math.maxExpand)
  })

  test('mirrors the chart limits', () => {
    expect([...RICH_CHART_LIMITS.types]).toEqual(fixture.chart.types)
    expect(RICH_CHART_LIMITS.maxSeries).toBe(fixture.chart.maxSeries)
    expect(RICH_CHART_LIMITS.maxLabels).toBe(fixture.chart.maxLabels)
    expect(RICH_CHART_LIMITS.maxLabelChars).toBe(fixture.chart.maxLabelChars)
    expect(RICH_CHART_LIMITS.maxTitleChars).toBe(fixture.chart.maxTitleChars)
    expect(RICH_CHART_LIMITS.maxAbsValue).toBe(fixture.chart.maxAbsValue)
    expect(RICH_CHART_LIMITS.maxSourceBytes).toBe(fixture.chart.maxSourceBytes)
    expect(RICH_CHART_LIMITS.maxSourceBytes).toBe(
      RICH_FENCE_LIMITS.find(limit => limit.language === 'chart')!.maxSourceBytes
    )
  })

  test('keeps the frozen fallback reason codes', () => {
    expect(fixture.fallbackReasons).toEqual(['invalid', 'too_large', 'unsupported', 'csp', 'error'])
  })
})
