import { describe, expect, test } from 'bun:test'

import { readFileSync } from 'node:fs'
import { fileURLToPath } from 'node:url'

import { MAX_TOOLS_PER_REQUEST } from '../../app/utils/admin/adminAI'

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

const pageShell = source('../../app/pages/admin/ai.vue')
const tab = source('../../app/components/admin/ai/tabs/SFAdminAIToolsTab.vue')
const composable = source('../../app/composables/admin/useAdminAI.ts')
const zh = locale('zh-CN')
const en = locale('en-US')

// tools 是第六个固定页签：页签只渲染，数据与草稿归页面壳。工具清单是「全局事实」，
// 用途白名单是「某个用途的生效配置」——两者必须在同一屏里对得上，否则运营者看到的
// 勾选状态和模型实际拿到的工具是两回事。
describe('admin AI tools tab registration', () => {
  test('registers the sixth tab in the page shell', () => {
    expect(pageShell).toContain("import SFAdminAIToolsTab from '~/components/admin/ai/tabs/SFAdminAIToolsTab.vue'")
    expect(pageShell).toContain("'providers', 'replyPrompt', 'purposes', 'tools', 'diagnose', 'observability'")
    expect(pageShell).toContain("type AITab = 'providers' | 'replyPrompt' | 'purposes' | 'tools' | 'diagnose' | 'observability'")
    expect(pageShell).toContain('tools: SFAdminAIToolsTab')
  })

  test('gives the tab a label key and a library icon instead of an emoji', () => {
    expect(pageShell).toContain("{ id: 'tools', label: t('admin.ai.tabs.tools'), icon: 'i-lucide-wrench' }")
    expect(lookup(zh, 'admin.ai.tabs.tools')).toBe('模型工具')
    expect(lookup(en, 'admin.ai.tabs.tools')).toBe('Model tools')
  })
})

describe('admin AI tools tab renders only', () => {
  test('reads the shared draft and never fetches or owns page chrome', () => {
    expect(tab).toContain('inject(ADMIN_AI_DRAFT_KEY)!')
    // 页签不自带标题/工具栏：几何归页面壳所有，与其它 AI 页签一致。
    expect(tab).not.toContain('UDashboardToolbar')
    expect(tab).not.toContain('<h2')
    expect(tab).not.toContain('SFAdminFixedTabNav')
    expect(tab).not.toContain('useAsyncData')
  })

  test('guards on the draft and renders one dense table instead of cards', () => {
    expect(tab).toContain('v-if="form"')
    expect(tab).toContain('<table')
    expect(tab).toContain('<thead')
    expect(tab).toContain('<tbody')
    expect(tab).toContain('overflow-x-auto')
    expect(tab).toContain('sticky left-0')
    expect(tab).toContain('line-clamp-2')
    // 大卡片布局已被表格取代：工具行不再拆成独立的卡片组件。
    expect(tab).not.toContain('SFAdminAIToolCard')
  })

  test('exposes the same columns as the permissions matrix', () => {
    for (const key of [
      'admin.ai.tools.columns.tool',
      'admin.ai.tools.columns.description',
      'admin.ai.tools.columns.parameters',
      'admin.ai.tools.columns.timeout'
    ]) {
      expect(tab).toContain(key)
    }
    // 操作列已去掉：详情与移除都在弹窗里，表格只保留矩阵本身。
    expect(tab).not.toContain('columns.actions')
    // 用途列由登记表驱动，不硬编码任何用途 id；回复用途排第一列。
    expect(tab).toContain('v-for="purpose in displayPurposes"')
    expect(tab).toContain("id === 'forum.reply' ? 0")
  })

  test('shows the overview, the diagnostics, the search and the dirty hint', () => {
    expect(tab).toContain('coreCount')
    expect(tab).toContain('pluginCount')
    expect(tab).toContain('unusedCount')
    expect(tab).toContain('admin.ai.tools.diagnosticsTitle')
    expect(tab).toContain('admin.ai.tools.searchPlaceholder')
    expect(tab).toContain('admin.ai.tools.empty')
    expect(tab).toContain('admin.ai.tools.dirtyHint')
  })

  test('reads the tool budget from the draft, treating nil as the default', () => {
    // nil 表示「按服务端默认生效」（默认不是 0）：只有显式写下的 0 才关闭工具循环。
    // 把 nil 当 0 会让每个新站点一进页面就看到一条假诊断。
    expect(tab).toContain('gates?.toolCallsPerReply === 0')
    expect(tab).not.toContain('!settings.gates.toolCallsPerReply')
    expect(tab).toContain('purposeToolSupportState')
    expect(tab).toContain('admin.ai.tools.diagnosticsNoProfile')
    expect(tab).toContain('admin.ai.tools.diagnosticsNoToolSupport')
  })

  test('filters with the shared helper instead of reimplementing the match', () => {
    expect(tab).toContain('filterTools(tools.value, query.value, origin.value)')
    expect(tab).toContain('admin.ai.tools.filterAll')
    expect(tab).toContain('admin.ai.tools.filterCore')
    expect(tab).toContain('admin.ai.tools.filterPlugin')
    expect(tab).toContain('admin.ai.tools.noResults')
  })

  test('delegates unregistered references and manual naming to the shared block', () => {
    // 白名单比工具活得久：清理入口与「先配策略、后装插件」的手动点名都在新区块里，
    // 页签本体不再持有这段交互。
    expect(tab).toContain('import SFAdminAIToolReferencesBlock')
    expect(tab).toContain('<SFAdminAIToolReferencesBlock />')
    expect(tab).not.toContain('referencedButUnregistered')
    expect(tab).not.toContain('removeUnregistered')
  })
})

describe('admin AI tools matrix', () => {
  test('toggles each purpose through setPurposeTool and caps new additions', () => {
    expect(tab).toContain('setPurposeTool(purpose.declaredTools, overrideFor(purpose.id), tool.name, checked)')
    expect(tab).toContain('MAX_TOOLS_PER_REQUEST')
    expect(tab).toContain('admin.ai.tools.tooMany')
    expect(MAX_TOOLS_PER_REQUEST).toBe(16)
    // 上限只拦新增：取消勾选永远允许，否则超限配置会把运营者锁在界面里。
    expect(tab).toContain('!effective.includes(tool.name)')
  })

  test('marks overridden purpose columns and resets them to the declared list', () => {
    expect(tab).toContain('purposeOverridden(purpose)')
    expect(tab).toContain('purposeToolsAreOverridden(overrideFor(purpose.id))')
    expect(tab).toContain('resetPurposeTools(overrideFor(purpose.id))')
    expect(tab).toContain('admin.ai.tools.resetPurpose')
    expect(tab).toContain('admin.ai.tools.resetPurposeDone')
    expect(tab).toContain('removeToolFromAllPurposes(catalog.value, current.purposes, tool.name)')
    expect(tab).toContain('admin.ai.tools.removeEverywhereDone')
  })

  test('writes the settings-document shape through the shared adapter', () => {
    expect(tab).toContain('toSettingsOverride(next)')
    expect(tab).toContain('toSettingsOverrides(')
  })

  test('honours the read-only permission gate on every editing control', () => {
    expect(tab).toContain(':disabled="readOnly"')
    expect(tab).toContain('if (!current || readOnly.value) return')
  })
})

describe('admin AI tool detail modal', () => {
  test('keeps the full description and the English editor out of the table row', () => {
    expect(tab).toContain('<UModal')
    expect(tab).toContain('detailOpen')
    expect(tab).toContain('detailTool')
    expect(tab).toContain('toolDisplayEnglish(detailTool.value, form.value?.toolDisplayDescriptions)')
    expect(tab).toContain('admin.ai.tools.detailClose')
  })

  test('writes only the en-US console copy into the draft', () => {
    expect(tab).toContain("locales['en-US'] = text")
    expect(tab).toContain("delete locales['en-US']")
    expect(tab).toContain('current.toolDisplayDescriptions =')
    expect(tab).toContain('admin.ai.tools.descriptionEnClear')
    expect(tab).toContain('admin.ai.tools.descriptionEnPlaceholder')
  })

  test('labels origin through the shared helper and copies through the clipboard', () => {
    expect(tab).toContain('toolOriginLabel(tool.origin)')
    expect(tab).toContain('navigator.clipboard?.writeText')
    expect(tab).toContain('admin.ai.tools.copyName')
    expect(tab).toContain('admin.ai.tools.copied')
    // 复制失败必须说清楚，且错误 Toast 不自动关闭。
    expect(tab).toContain('admin.ai.tools.copyFailed')
    expect(tab).toContain("color: 'error'")
  })

  test('summarises parameters and the plugin timeout', () => {
    expect(tab).toContain('requiredNames(tool).has(name)')
    expect(tab).toContain('admin.ai.tools.parametersRequired')
    expect(tab).toContain('admin.ai.tools.parametersTruncated')
    expect(tab).toContain('admin.ai.tools.timeout')
    expect(tab).toContain('tool.timeoutMs')
  })

  test('renders text through interpolation only', () => {
    expect(tab).not.toContain('v-html')
    // 无 emoji 图标：图标一律走 i-lucide-*，文本里不夹带表情符号。
    expect(/[\p{Extended_Pictographic}]/u.test(tab)).toBe(false)
  })
})

describe('admin AI tool plugin settings jump', () => {
  test('links only when the tool carries a declared settings path', () => {
    expect(composable).toContain('settingsPath?: string')
    expect(tab).toContain('useAdminRoutes()')
    expect(tab).toContain('v-if="tool.settingsPath"')
    expect(tab).toContain('adminRoutes.path(tool.settingsPath)')
    expect(tab).toContain('admin.attachments.openPluginSettings')
    expect(tab).toContain('i-lucide-arrow-up-right')
  })
})

describe('admin AI tools tab localization', () => {
  test('ships every admin.ai.tools.* key in both locales', () => {
    for (const key of [
      'admin.ai.tools.title',
      'admin.ai.tools.intro',
      'admin.ai.tools.empty',
      'admin.ai.tools.emptyHint',
      'admin.ai.tools.searchPlaceholder',
      'admin.ai.tools.filterAll',
      'admin.ai.tools.filterCore',
      'admin.ai.tools.filterPlugin',
      'admin.ai.tools.originCore',
      'admin.ai.tools.originPlugin',
      'admin.ai.tools.descriptionModel',
      'admin.ai.tools.descriptionEn',
      'admin.ai.tools.descriptionEnMissing',
      'admin.ai.tools.descriptionEnEdit',
      'admin.ai.tools.descriptionEnPlaceholder',
      'admin.ai.tools.descriptionEnClear',
      'admin.ai.tools.descriptionEnOperator',
      'admin.ai.tools.descriptionEnDeclared',
      'admin.ai.tools.parameters',
      'admin.ai.tools.parametersRequired',
      'admin.ai.tools.parametersEmpty',
      'admin.ai.tools.parametersTruncated',
      'admin.ai.tools.timeout',
      'admin.ai.tools.timeoutHint',
      'admin.ai.tools.usedBy',
      'admin.ai.tools.usedByNone',
      'admin.ai.tools.usedByNoneHint',
      'admin.ai.tools.unusedCount',
      'admin.ai.tools.tooMany',
      'admin.ai.tools.resetPurpose',
      'admin.ai.tools.resetPurposeDone',
      'admin.ai.tools.unregisteredCount',
      'admin.ai.tools.manualAdd.title',
      'admin.ai.tools.manualAdd.hint',
      'admin.ai.tools.manualAdd.purpose',
      'admin.ai.tools.manualAdd.purposeMissing',
      'admin.ai.tools.manualAdd.namePlaceholder',
      'admin.ai.tools.manualAdd.add',
      'admin.ai.tools.manualAdd.added',
      'admin.ai.tools.manualAdd.alreadyPresent',
      'admin.ai.tools.manualAdd.nameInvalid',
      'admin.ai.tools.manualAdd.registered',
      'admin.ai.tools.manualAdd.empty',
      'admin.ai.tools.removeEverywhere',
      'admin.ai.tools.removeEverywhereDone',
      'admin.ai.tools.copyName',
      'admin.ai.tools.copied',
      'admin.ai.tools.copyFailed',
      'admin.ai.tools.noResults',
      'admin.ai.tools.unregisteredTitle',
      'admin.ai.tools.unregisteredHint',
      'admin.ai.tools.unregisteredRemove',
      'admin.ai.tools.diagnosticsTitle',
      'admin.ai.tools.diagnosticsGateOff',
      'admin.ai.tools.diagnosticsNoProfile',
      'admin.ai.tools.diagnosticsNoToolSupport',
      'admin.ai.tools.dirtyHint',
      'admin.ai.tools.loadHint',
      'admin.ai.tools.columns.tool',
      'admin.ai.tools.columns.description',
      'admin.ai.tools.columns.parameters',
      'admin.ai.tools.columns.timeout',
      'admin.ai.tools.detailClose'
    ]) {
      expect(`${key}:${typeof lookup(zh, key)}`).toBe(`${key}:string`)
      expect(`${key}:${typeof lookup(en, key)}`).toBe(`${key}:string`)
    }
  })

  test('every key the tab references exists in both locales', () => {
    // 静态扫一遍组件里所有 t('…') 调用：少一个键就会在界面上渲染出裸键名。
    const keys = new Set<string>()
    for (const match of tab.matchAll(/\bt\('([A-Za-z0-9_.]+)'/g)) {
      keys.add(match[1]!)
    }
    expect(keys.size).toBeGreaterThan(30)
    for (const key of keys) {
      expect(`${key}:${typeof lookup(zh, key)}`).toBe(`${key}:string`)
      expect(`${key}:${typeof lookup(en, key)}`).toBe(`${key}:string`)
    }
  })
})
