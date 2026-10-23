import { describe, expect, test } from 'bun:test'

import { readFileSync } from 'node:fs'
import { fileURLToPath } from 'node:url'

import {
  isValidPurposeToolName,
  normalizedAISettingsForComparison,
  normalizePurposeToolNames,
  prunePurposeOverrides,
  purposeOverrideIsEmpty,
  purposeToolsAreOverridden,
  selectedPurposeTools
} from '../../app/utils/admin/adminAI'

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

const DEFAULT_PROMPT = '你是助手。'

describe('purpose override shapes', () => {
  test('only a present tools field means the operator took the allowlist over', () => {
    // 字段存在（哪怕空列表）＝运营者接管；字段缺失＝沿用用途声明。两者不能混同，
    // 否则「一个工具都不给」会被当成「没配置过」。
    expect(purposeToolsAreOverridden(undefined)).toBe(false)
    expect(purposeToolsAreOverridden({})).toBe(false)
    expect(purposeToolsAreOverridden({ failurePosture: 'fail_closed' })).toBe(false)
    expect(purposeToolsAreOverridden({ tools: { allowed: [] } })).toBe(true)
    expect(purposeToolsAreOverridden({ tools: { allowed: ['forum-search'] } })).toBe(true)
    // 与后端 PurposeSettings.ToolAllowlist() 对齐：`{"tools":{}}` 也是接管，
    // 语义是「一个工具都不给」，不是「沿用声明」。
    expect(purposeToolsAreOverridden({ tools: {} })).toBe(true)
  })

  test('an override that overrides nothing is empty', () => {
    expect(purposeOverrideIsEmpty(undefined)).toBe(true)
    expect(purposeOverrideIsEmpty({})).toBe(true)
    expect(purposeOverrideIsEmpty({ failurePosture: '   ' })).toBe(true)
    expect(purposeOverrideIsEmpty({ failurePosture: 'fail_open' })).toBe(false)
    expect(purposeOverrideIsEmpty({ tools: { allowed: [] } })).toBe(false)
  })

  test('effective tools follow the override, including an empty list', () => {
    const declared = ['forum-search', 'time-now']
    expect(selectedPurposeTools(declared, undefined)).toEqual(['forum-search', 'time-now'])
    expect(selectedPurposeTools(declared, { tools: { allowed: [] } })).toEqual([])
    expect(selectedPurposeTools(declared, { tools: {} })).toEqual([])
    expect(selectedPurposeTools(declared, { tools: { allowed: ['vendor-tool'] } })).toEqual(['vendor-tool'])
  })

  test('normalizes tool names so selection order is not a change', () => {
    expect(normalizePurposeToolNames([' b ', 'a', 'a', ''])).toEqual(['a', 'b'])
    expect(normalizePurposeToolNames(undefined)).toEqual([])
  })

  test('validates tool names against the same charset the server accepts', () => {
    expect(isValidPurposeToolName('forum-search')).toBe(true)
    expect(isValidPurposeToolName('vendor_tool_2')).toBe(true)
    expect(isValidPurposeToolName('Forum Search')).toBe(false)
    expect(isValidPurposeToolName('forum.search')).toBe(false)
    expect(isValidPurposeToolName('   ')).toBe(false)
    expect(isValidPurposeToolName('x'.repeat(65))).toBe(false)
  })
})

describe('purpose overrides survive the settings round trip', () => {
  test('drops empty overrides and omits the whole field when nothing is overridden', () => {
    expect(prunePurposeOverrides({ 'forum.reply': {} })).toBeUndefined()
    expect(prunePurposeOverrides(undefined)).toBeUndefined()
    expect(prunePurposeOverrides({ 'forum.reply': { failurePosture: '  ' } })).toBeUndefined()
  })

  test('keeps a real override with normalized tool names', () => {
    const pruned = prunePurposeOverrides({
      'forum.reply': { tools: { allowed: ['time-now', ' forum-search '] }, failurePosture: ' fail_closed ' }
    })
    expect(pruned).toEqual({
      'forum.reply': { tools: { allowed: ['forum-search', 'time-now'] }, failurePosture: 'fail_closed' }
    })
  })

  test('normalizes a bare tools object into an explicit empty allowlist', () => {
    // 后端会把 `{"tools":{}}` 归一成 `{"tools":{"allowed":[]}}`；前端提交前保持同形，
    // 保存后再读回来才是同一份文档，脏判定也不会因为形状差异误报。
    expect(prunePurposeOverrides({ 'forum.reply': { tools: {} } })).toEqual({
      'forum.reply': { tools: { allowed: [] } }
    })
  })

  test('stored defaults and an untouched form compare equal', () => {
    // 服务端会丢弃空覆盖并省略字段；界面上一打开就持有空对象，两者必须等价，
    // 否则面板一进来就显示「有未保存的改动」。
    const stored = { reply: {}, purposes: undefined }
    const form = { reply: { systemPrompt: '' }, purposes: {} }
    expect(JSON.stringify(normalizedAISettingsForComparison(form, DEFAULT_PROMPT)))
      .toBe(JSON.stringify(normalizedAISettingsForComparison(stored, DEFAULT_PROMPT)))
  })

  test('an explicit empty allowlist stays distinguishable from unset', () => {
    const unset = normalizedAISettingsForComparison({ purposes: {} }, DEFAULT_PROMPT)
    const empty = normalizedAISettingsForComparison({ purposes: { 'forum.reply': { tools: { allowed: [] } } } }, DEFAULT_PROMPT)
    expect(JSON.stringify(empty)).not.toBe(JSON.stringify(unset))
    expect(empty.purposes?.['forum.reply']?.tools?.allowed).toEqual([])
  })
})

describe('admin AI purposes console surface', () => {
  const pageShell = source('../../app/pages/admin/ai.vue')
  const tab = source('../../app/components/admin/ai/tabs/SFAdminAIPurposesTab.vue')
  const block = source('../../app/components/admin/ai/blocks/SFAdminAIPurposesBlock.vue')
  const composable = source('../../app/composables/admin/useAdminAI.ts')

  test('registers the tab in the page shell and the query deep link', () => {
    expect(pageShell).toContain('SFAdminAIPurposesTab')
    expect(pageShell).toContain("'providers', 'replyPrompt', 'purposes', 'tools', 'diagnose', 'observability'")
    expect(pageShell).toContain('purposes: SFAdminAIPurposesTab')
  })

  test('syncs the active tab with the query so deep links switch panes', () => {
    // router.replace 只改地址栏，不重新挂载页面；没有这个 watch，从用途页签跳到
    // ?tab=tools 只会更新 URL，内容仍停在原页签。
    expect(pageShell).toContain('watch(() => route.query.tab')
    expect(pageShell).toContain('activeTab.value = normalizeTab(value)')
  })

  test('the tab renders the block and shares the page draft', () => {
    expect(tab).toContain('SFAdminAIPurposesBlock')
    expect(tab).toContain('ADMIN_AI_DRAFT_KEY')
    // 页签不自带标题/工具栏：几何归页面壳所有。
    expect(tab).not.toContain('UDashboardToolbar')
    expect(tab).not.toContain('<h2')
  })

  test('the block edits the shared draft instead of owning a copy', () => {
    expect(block).toContain('ADMIN_AI_DRAFT_KEY')
    expect(block).toContain('form.value?.purposes')
    expect(block).toContain('writeOverride')
  })

  test('shows the effective tool count without owning an editor', () => {
    // 工具可见性只有一个编辑面（模型工具页签）：这里保留声明/生效两份事实，
    // 但只读——计数、徽标与深链足够让运营者知道该去哪里改。
    expect(block).toContain('selectedPurposeTools')
    expect(block).toContain('admin.ai.purposes.tools.summary')
    expect(block).toContain('admin.ai.purposes.tools.managedHint')
    expect(block).toContain('admin.ai.purposes.tools.manage')
    expect(block).toContain("query: { tab: 'tools' }")
    expect(block).toContain("adminRoutes.path('ai')")
    expect(block).toContain('admin.ai.purposes.tools.declaredBadge')
    expect(block).toContain('admin.ai.purposes.tools.overriddenBadge')
  })

  test('no longer edits tool visibility; the model tools tab owns it', () => {
    expect(block).not.toContain('setPurposeTool')
    expect(block).not.toContain('UCheckbox')
    expect(block).not.toContain('availableTools')
    expect(block).not.toContain('MAX_TOOLS_PER_REQUEST')
  })

  test('keeps the failure posture selectable and restorable', () => {
    expect(block).toContain('AI_FAILURE_POSTURES')
    expect(block).toContain('onPostureChange')
    expect(block).toContain('restorePurpose')
  })

  test('never offers an empty-string option value to the select', () => {
    // reka-ui 的 Select 对空 value 的处理是未定义行为：选项渲染为空、选中态无法
    // 成立（SelectItem missing render function 就是它）。「跟随声明」必须用显式
    // 哨兵值表达，写入时再翻译回「删除字段」。
    expect(block).toContain("POSTURE_DECLARED = '__declared__'")
    expect(block).toContain('value: POSTURE_DECLARED')
    expect(block).toContain('postureValue(purpose) || POSTURE_DECLARED')
    // 翻译必须发生在写入侧：哨兵不得进配置文档。
    expect(block).toContain("value === POSTURE_DECLARED ? '' : value")
  })

  test('honours the read-only permission gate', () => {
    expect(block).toContain('readOnly')
    expect(block).toContain(':disabled="readOnly"')
  })

  test('consumes the payload catalog and its types', () => {
    expect(composable).toContain('AdminAIValidatedPurpose')
    expect(composable).toContain('purposes: AdminAIValidatedPurpose[]')
    expect(composable).toContain('payload.data.value?.purposes')
    expect(block).toContain('ai.purposes.value')
  })

  test('ships complete zh-CN and en-US copy', () => {
    const zh = locale('zh-CN')
    const en = locale('en-US')
    for (const key of [
      'admin.ai.tabs.purposes',
      'admin.ai.purposes.title',
      'admin.ai.purposes.intro',
      'admin.ai.purposes.empty',
      'admin.ai.purposes.restore',
      'admin.ai.purposes.restoreDone',
      'admin.ai.purposes.dirtyHint',
      'admin.ai.purposes.columns.posture',
      'admin.ai.purposes.columns.tools',
      'admin.ai.purposes.cost.economy',
      'admin.ai.purposes.cost.standard',
      'admin.ai.purposes.cost.premium',
      'admin.ai.purposes.posture.followDeclared',
      'admin.ai.purposes.posture.fail_open',
      'admin.ai.purposes.posture.fail_closed',
      'admin.ai.purposes.posture.skip_ai',
      'admin.ai.purposes.postureValue.fail_open',
      'admin.ai.purposes.postureValue.fail_closed',
      'admin.ai.purposes.postureValue.skip_ai',
      'admin.ai.purposes.blastRadius.single',
      'admin.ai.purposes.blastRadius.batch',
      'admin.ai.purposes.blastRadius.site',
      'admin.ai.purposes.tools.declaredBadge',
      'admin.ai.purposes.tools.overriddenBadge',
      'admin.ai.purposes.tools.summary',
      'admin.ai.purposes.tools.managedHint',
      'admin.ai.purposes.tools.manage',
      'admin.ai.purposes.restoreHint',
      'admin.ai.purposes.resourceType',
      'admin.ai.purposes.promptVersion'
    ]) {
      expect(lookup(zh, key)).toBeTruthy()
      expect(lookup(en, key)).toBeTruthy()
    }
  })
})

describe('tool catalog payload', () => {
  const composable = source('../../app/composables/admin/useAdminAI.ts')

  test('the payload carries the registered tool catalog', () => {
    expect(composable).toContain('AdminAIToolAvailability')
    expect(composable).toContain('availableTools?: AdminAIToolAvailability[]')
    expect(composable).toContain('payload.data.value?.availableTools')
  })
})
