import { describe, expect, test } from 'bun:test'

import { readFileSync } from 'node:fs'
import { fileURLToPath } from 'node:url'

import {
  cacheHitOnlyRun,
  deniedWithoutOutbound,
  executionExpandable,
  executionStatusColor,
  timelineEntries,
  type RunTimelineRecord
} from '../../app/components/admin/ai/runTimeline'

function source(relative: string): string {
  return readFileSync(fileURLToPath(new URL(relative, import.meta.url)), 'utf8')
}

function locale(name: string): Record<string, unknown> {
  return JSON.parse(source(`../../i18n/locales/${name}.json`)) as Record<string, unknown>
}

function lookup(catalog: Record<string, unknown>, key: string): unknown {
  return key.split('.').reduce<unknown>((node, part) => {
    if (!node || typeof node !== 'object') return undefined
    return (node as Record<string, unknown>)[part]
  }, catalog)
}

const execution: RunTimelineRecord = {
  status: 'succeeded',
  cacheHit: false,
  model: 'gpt-4o-mini',
  latencyMs: 412,
  inputTokens: 1200,
  outputTokens: 300
}

const observabilityTab = source('../../app/components/admin/ai/tabs/SFAdminAIObservabilityTab.vue')
const timelineBlock = source('../../app/components/admin/ai/blocks/SFAdminAIRunTimeline.vue')
const composable = source('../../app/composables/admin/useAdminAI.ts')

describe('run timeline ordering', () => {
  test('numbers records in the order the server returned them', () => {
    // 后端已按 created_at ASC, id ASC 返回。前端再排一次排序就是第二个真相——
    // 同毫秒内只有 id 知道顺序。
    const entries = timelineEntries([execution, execution, execution])
    expect(entries.map(entry => entry.index)).toEqual([1, 2, 3])
  })

  test('does not assume a fixed record count', () => {
    expect(timelineEntries([])).toEqual([])
    expect(timelineEntries([execution])).toHaveLength(1)
    expect(timelineEntries(Array.from({ length: 9 }, () => execution))).toHaveLength(9)
  })
})

describe('run timeline expandability', () => {
  test('only records with a run id get an expand affordance', () => {
    // A3 之前的历史记录没有 run 归属：展开只会得到空列表，入口不该存在。
    expect(executionExpandable({ resourceId: 'abc123' })).toBe(true)
    expect(executionExpandable({ resourceId: '' })).toBe(false)
    expect(executionExpandable({ resourceId: '   ' })).toBe(false)
    expect(executionExpandable({})).toBe(false)
  })
})

describe('run timeline explicit annotations', () => {
  test('a single cache-hit record is recognised as a cached run', () => {
    // 缓存命中的 run 只有一条记录：正确行为，不是「没调工具」。
    expect(cacheHitOnlyRun([{ status: 'succeeded', cacheHit: true }])).toBe(true)
    expect(cacheHitOnlyRun([{ status: 'succeeded', cacheHit: false }])).toBe(false)
  })

  test('more than one record is never reported as a cached run', () => {
    expect(cacheHitOnlyRun([
      { status: 'succeeded', cacheHit: true },
      { status: 'succeeded', cacheHit: false }
    ])).toBe(false)
  })

  test('zero records is "nothing to show", not "cache hit"', () => {
    expect(cacheHitOnlyRun([])).toBe(false)
  })

  test('denied records are flagged as never leaving the server', () => {
    expect(deniedWithoutOutbound({ status: 'denied', cacheHit: false })).toBe(true)
    expect(deniedWithoutOutbound({ status: 'failed', cacheHit: false })).toBe(false)
    expect(deniedWithoutOutbound({ status: 'succeeded', cacheHit: false })).toBe(false)
  })

  test('status colours are shared by the table and the timeline', () => {
    expect(executionStatusColor('succeeded')).toBe('success')
    expect(executionStatusColor('denied')).toBe('warning')
    expect(executionStatusColor('failed')).toBe('error')
  })
})

describe('admin AI run timeline surface', () => {
  test('the composable reads the run endpoint and accepts a null run id', () => {
    expect(composable).toContain('export function useAdminAIRunTimeline(runId: Ref<string | null> | ComputedRef<string | null>)')
    expect(composable).toContain('/admin/ai/runs/${encodeURIComponent(id)}')
    // 空 runId 就地短路：服务端对空 id 也只会返回空列表，不必发这次往返。
    expect(composable).toContain('if (!id) return { items: [] }')
    expect(composable).toContain('watch: [normalizedRunId]')
  })

  test('the payload shape matches listExecutions', () => {
    expect(composable).toContain('useAsyncData<{ items: AdminAIExecution[] }>')
  })

  test('the tab expands a row into its run timeline', () => {
    expect(observabilityTab).toContain('useAdminAIRunTimeline(expandedRunId)')
    expect(observabilityTab).toContain('SFAdminAIRunTimeline')
    expect(observabilityTab).toContain('toggleRun')
  })

  test('expands one row, not every row of the same run', () => {
    // 一次 run 通常占 2–3 行（每条模型调用一行）。展开态按记录 id 记，否则每一行
    // 都渲染一个时间线：同一份内容重复三遍，且三个面板抢同一个 id。
    // 浏览器 QA 实测：按 runId 记时同屏出现 3 个 section。
    expect(observabilityTab).toContain('recordId: number, runId: string')
    expect(observabilityTab).toContain('isExpanded(item)')
    expect(observabilityTab).toContain('v-if="isExpanded(item)"')
    expect(observabilityTab).not.toContain('expandedRunId === item.resourceId')
  })

  test('panel ids are unique per row and match aria-controls exactly', () => {
    // 按钮按记录 id 命名 aria-controls，面板必须用同一个 id——否则 aria-controls
    // 指向一个不存在的元素（浏览器 QA 实测：按钮点 158，面板挂 runId）。
    expect(observabilityTab).toContain('admin-ai-run-timeline-${item.id}')
    expect(observabilityTab).toContain(':panel-id="timelinePanelId(item)"')
    expect(observabilityTab).toContain('aria-controls="isExpanded(item) ? timelinePanelId(item)')
    // 子组件不再自己派生 id。
    expect(timelineBlock).toContain('panelId: string')
    expect(timelineBlock).not.toContain('`admin-ai-run-timeline-${props.runId}`')
  })

  test('records without a run id get no expand affordance and no error', () => {
    expect(observabilityTab).toContain('v-if="executionExpandable(item)"')
    // 行点击与展开按钮都走同一个守卫：没有 run 归属时 toggleRun 直接返回。
    expect(observabilityTab).toContain('if (!executionExpandable(item)) return')
  })

  test('the timeline annotates a cached run instead of implying a lost tool loop', () => {
    expect(timelineBlock).toContain('cacheHitOnly')
    expect(timelineBlock).toContain("t('admin.ai.runTimeline.cacheHitOnly')")
  })

  test('the timeline marks denied records as never outbound and shows gateReason', () => {
    expect(timelineBlock).toContain('deniedWithoutOutbound')
    expect(timelineBlock).toContain("t('admin.ai.runTimeline.deniedBadge')")
    expect(timelineBlock).toContain("t('admin.ai.runTimeline.deniedHint')")
    expect(timelineBlock).toContain('entry.record.gateReason')
  })

  test('the timeline never fabricates tool-call detail', () => {
    // 工具调用明细在日志里，不在执行记录里。界面不得出现工具步/工具名的渲染。
    const block = timelineBlock
    expect(block).toContain("t('admin.ai.runTimeline.traceHonesty')")
    expect(block).not.toContain('toolCalls')
    expect(block).not.toContain('ToolTrace')
    expect(block).not.toContain('argumentsBytes')
  })

  test('new controls carry both an accessible name and a title', () => {
    // 展开/收起在表格行（页签），关闭在时间线块。两处都必须 aria-label + title 成对出现。
    for (const key of ['runTimeline.expand', 'runTimeline.collapse']) {
      expect(observabilityTab).toContain(`t('admin.ai.${key}')`)
    }
    expect(observabilityTab).toMatch(/:aria-label="[^"]*runTimeline\.(expand|collapse)/)
    expect(observabilityTab).toMatch(/:title="[^"]*runTimeline\.(expand|collapse)/)
    expect(timelineBlock).toContain("t('admin.ai.runTimeline.close')")
    expect(timelineBlock).toMatch(/:aria-label="t\('admin\.ai\.runTimeline\.close'\)"/)
    expect(timelineBlock).toMatch(/:title="t\('admin\.ai\.runTimeline\.close'\)"/)
  })

  test('the timeline stays inside a 390px viewport', () => {
    // 表格已由 overflow-x-auto 承担横向滚动，时间线本身必须 min-w-0 且允许换行。
    expect(timelineBlock).toContain('min-w-0')
    expect(timelineBlock).toContain('flex-wrap')
    expect(timelineBlock).not.toContain('whitespace-nowrap')
    // 长 id / 长错误文本靠 truncate 与 break-words 收敛，不撑宽容器。
    expect(timelineBlock).toContain('truncate')
    expect(timelineBlock).toContain('break-words')
    // 面板挂在表格单元格里，而表格有 min-w-full + 横向滚动：不给上限时面板会跟
    // 着表格宽度走（390 屏上实测 788px，只能看到左侧一条）。上限按视口算。
    expect(timelineBlock).toContain('max-w-[calc(100vw-2rem)]')
  })
})

describe('omitted numeric fields never render as NaN', () => {
  test('every numeric column goes through the fallback', () => {
    // 后端 ExecutionRecord 的数值字段是 `omitempty`：0 值不会出现在 JSON 里。
    // 直接插值会渲染出「NaN」「ms」「/」（浏览器 QA 实测），把「本来是 0」显示成
    // 「数据坏了」。三个数值列必须全部经由 num()。
    expect(observabilityTab).toContain('num(item.latencyMs)')
    expect(observabilityTab).toContain('num(item.inputTokens)')
    expect(observabilityTab).toContain('num(item.outputTokens)')
    expect(observabilityTab).toContain('yuan(num(item.spendMicros))')
    // 没有漏网的裸插值。
    expect(observabilityTab).not.toContain('{{ item.latencyMs }}')
    expect(observabilityTab).not.toContain('{{ item.inputTokens }}')
    expect(observabilityTab).not.toContain('{{ item.spendMicros }}')
  })

  test('num treats missing, null, and non-finite values as zero', () => {
    expect(observabilityTab).toContain('Number.isFinite(value) ? value : 0')
  })

  test('an omitted gateScope does not leave a dangling separator', () => {
    expect(observabilityTab).toContain('item.gateScope ?')
    expect(timelineBlock).toContain('v-if="entry.record.gateScope"')
  })
})

describe('run timeline bilingual copy', () => {
  test('ships every new key in both locales under the same names', () => {
    const zh = locale('zh-CN')
    const en = locale('en-US')
    const keys = [
      'admin.ai.runTimeline.expand',
      'admin.ai.runTimeline.collapse',
      'admin.ai.runTimeline.title',
      'admin.ai.runTimeline.intro',
      'admin.ai.runTimeline.runIdLabel',
      'admin.ai.runTimeline.loading',
      'admin.ai.runTimeline.loadFailed',
      'admin.ai.runTimeline.empty',
      'admin.ai.runTimeline.close',
      'admin.ai.runTimeline.step',
      'admin.ai.runTimeline.latency',
      'admin.ai.runTimeline.tokens',
      'admin.ai.runTimeline.cacheHitRecord',
      'admin.ai.runTimeline.cacheHitOnly',
      'admin.ai.runTimeline.deniedBadge',
      'admin.ai.runTimeline.deniedHint',
      'admin.ai.runTimeline.traceHonesty'
    ]
    for (const key of keys) {
      const zhValue = lookup(zh, key)
      const enValue = lookup(en, key)
      expect(typeof zhValue).toBe('string')
      expect(typeof enValue).toBe('string')
      expect((zhValue as string).trim().length).toBeGreaterThan(0)
      expect((enValue as string).trim().length).toBeGreaterThan(0)
    }
  })

  test('uses no emoji as iconography', () => {
    const emoji = /[\u{1F300}-\u{1FAFF}\u{2600}-\u{27BF}]/u
    for (const name of ['zh-CN', 'en-US']) {
      const timeline = JSON.stringify(lookup(locale(name), 'admin.ai.runTimeline'))
      expect(emoji.test(timeline)).toBe(false)
    }
  })
})
