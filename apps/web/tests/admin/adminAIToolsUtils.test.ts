import { describe, expect, test } from 'bun:test'

import { readFileSync } from 'node:fs'
import { fileURLToPath } from 'node:url'

import type { AIComparableSettings } from '../../app/utils/admin/adminAI'
import type { AIToolDisplayLike, AIToolPurposeLike } from '../../app/utils/admin/adminAITools'
import {
  normalizedAISettingsForComparison,
  pruneToolDisplayDescriptions
} from '../../app/utils/admin/adminAI'
import {
  filterTools,
  purposeToolExposure,
  purposeToolSupportState,
  referencedButUnregistered,
  removeToolFromAllPurposes,
  setPurposeTool,
  toolDisplayEnglish,
  toolOriginLabel
} from '../../app/utils/admin/adminAITools'

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

function canonical(settings: AIComparableSettings, defaultPrompt: string): string {
  return JSON.stringify(normalizedAISettingsForComparison(settings, defaultPrompt))
}

const DEFAULT_PROMPT = '你是助手。'

describe('purpose tool auto-convergence', () => {
  test('unticking a declared tool materialises the remaining allowlist', () => {
    // 取消一个声明默认的工具＝运营者接管，必须写成显式 allowed；空列表也有意义
    // （这个用途一个工具都不给），不能被当成「没配置」。
    expect(setPurposeTool(['forum-search'], undefined, 'forum-search', false))
      .toEqual({ tools: { allowed: [] } })
    expect(setPurposeTool(['forum-search', 'time-now'], undefined, 'forum-search', false))
      .toEqual({ tools: { allowed: ['time-now'] } })
  })

  test('ticking the last missing tool back converges to no override at all', () => {
    // 勾上再取消必须回到「未覆盖」：留下一个与声明同值的覆盖，会让面板一直显示
    // 「已自定义」，并且掩盖后续声明变更。
    const declared = ['forum-search']
    const unticked = setPurposeTool(declared, undefined, 'forum-search', false)
    expect(setPurposeTool(declared, unticked, 'forum-search', true)).toBeUndefined()

    const added = setPurposeTool(declared, undefined, 'vendor-tool', true)
    expect(added).toEqual({ tools: { allowed: ['forum-search', 'vendor-tool'] } })
    expect(setPurposeTool(declared, added, 'vendor-tool', false)).toBeUndefined()
  })

  test('failurePosture survives convergence instead of being dropped with tools', () => {
    const declared = ['forum-search']
    const posture = 'fail_closed'
    const unticked = setPurposeTool(declared, { failurePosture: posture }, 'forum-search', false)
    expect(unticked).toEqual({ tools: { allowed: [] }, failurePosture: posture })
    expect(setPurposeTool(declared, unticked, 'forum-search', true)).toEqual({ failurePosture: posture })
  })

  test('ticking an already-declared tool is a no-op, not a new override', () => {
    expect(setPurposeTool(['forum-search'], undefined, 'forum-search', true)).toBeUndefined()
    // 覆盖里已经是声明集合时，重复勾选同样收敛掉。
    const same = { tools: { allowed: ['forum-search'] }, failurePosture: 'skip_ai' }
    expect(setPurposeTool(['forum-search'], same, 'forum-search', true))
      .toEqual({ failurePosture: 'skip_ai' })
  })

  test('selection order is not a change: the allowlist is always sorted', () => {
    const result = setPurposeTool(['time-now'], undefined, 'forum-search', true)
    expect(result?.tools?.allowed).toEqual(['forum-search', 'time-now'])
  })

  test('a blank tool name never rewrites the override', () => {
    expect(setPurposeTool(['forum-search'], undefined, '   ', true)).toBeUndefined()
    expect(setPurposeTool(['forum-search'], { failurePosture: 'fail_open' }, '', false))
      .toEqual({ failurePosture: 'fail_open' })
  })
})

describe('removing a tool from every purpose', () => {
  test('converges each purpose independently and keeps other overrides', () => {
    const purposes: AIToolPurposeLike[] = [
      { id: 'forum.reply', declaredTools: ['forum-search', 'time-now'] },
      { id: 'moderation.review', declaredTools: ['moderation-hide'] }
    ]
    const overrides = {
      'forum.reply': { tools: { allowed: ['forum-search', 'time-now', 'vendor-tool'] }, failurePosture: 'fail_closed' },
      'moderation.review': { tools: { allowed: ['moderation-hide', 'vendor-tool'] } }
    }
    // vendor-tool 只是运营者额外加进来的：移除后两个用途都回到声明默认。
    expect(removeToolFromAllPurposes(purposes, overrides, 'vendor-tool'))
      .toEqual({ 'forum.reply': { failurePosture: 'fail_closed' } })
  })

  test('returns undefined when nothing is left to override', () => {
    const purposes: AIToolPurposeLike[] = [{ id: 'forum.reply', declaredTools: ['forum-search'] }]
    expect(removeToolFromAllPurposes(
      purposes,
      { 'forum.reply': { tools: { allowed: ['forum-search', 'vendor-tool'] } } },
      'vendor-tool'
    )).toBeUndefined()
    // 名字根本不在任何生效集合里：没有任何覆盖需要改写。
    expect(removeToolFromAllPurposes(purposes, undefined, 'vendor-tool')).toBeUndefined()
  })

  test('removing a declared tool leaves an explicit allowlist behind', () => {
    const purposes: AIToolPurposeLike[] = [{ id: 'forum.reply', declaredTools: ['forum-search', 'time-now'] }]
    expect(removeToolFromAllPurposes(purposes, undefined, 'forum-search'))
      .toEqual({ 'forum.reply': { tools: { allowed: ['time-now'] } } })
  })

  test('overrides for purposes outside the catalog are preserved, not silently dropped', () => {
    const purposes: AIToolPurposeLike[] = [{ id: 'forum.reply', declaredTools: ['forum-search'] }]
    expect(removeToolFromAllPurposes(
      purposes,
      { 'legacy.purpose': { failurePosture: 'skip_ai' } },
      'vendor-tool'
    )).toEqual({ 'legacy.purpose': { failurePosture: 'skip_ai' } })
  })
})

describe('reverse index: which purposes expose which tools', () => {
  test('materialises the override when present and the declaration otherwise', () => {
    const purposes: AIToolPurposeLike[] = [
      { id: 'forum.reply', declaredTools: ['forum-search', 'time-now'] },
      { id: 'moderation.review', declaredTools: ['moderation-hide'] },
      { id: 'site.summary', declaredTools: ['forum-search'] }
    ]
    const exposure = purposeToolExposure(purposes, {
      'moderation.review': { tools: { allowed: ['moderation-hide', 'forum-search'] } },
      'site.summary': { tools: { allowed: [] } }
    })
    expect(exposure.get('forum.reply')).toEqual(['forum-search', 'time-now'])
    expect(exposure.get('moderation.review')).toEqual(['forum-search', 'moderation-hide'])
    // 显式空列表＝这个用途一个工具都不用，不能回退到声明。
    expect(exposure.get('site.summary')).toEqual([])
    expect(exposure.size).toBe(3)
  })

  test('ignores blank purpose ids so one bad row cannot create a phantom entry', () => {
    const exposure = purposeToolExposure([{ id: '  ', declaredTools: ['forum-search'] }], undefined)
    expect(exposure.size).toBe(0)
  })
})

describe('referenced but unregistered tools', () => {
  const purposes: AIToolPurposeLike[] = [
    { id: 'moderation.review', declaredTools: ['moderation-hide', 'ghost-tool'] },
    { id: 'forum.reply', declaredTools: ['forum-search'] }
  ]
  const overrides = {
    'forum.reply': { tools: { allowed: ['forum-search', 'ghost-tool', 'another-ghost'] } }
  }

  test('lists every unknown name once, sorted, with its referencing purposes', () => {
    expect(referencedButUnregistered(purposes, overrides, ['forum-search', 'moderation-hide']))
      .toEqual([
        { name: 'another-ghost', purposes: ['forum.reply'] },
        { name: 'ghost-tool', purposes: ['forum.reply', 'moderation.review'] }
      ])
  })

  test('accepts a Set and returns nothing when every reference is registered', () => {
    expect(referencedButUnregistered(purposes, overrides, new Set(['forum-search', 'moderation-hide', 'ghost-tool', 'another-ghost'])))
      .toEqual([])
  })

  test('trims registry entries so whitespace cannot hide a registration', () => {
    expect(referencedButUnregistered(purposes, overrides, [' forum-search ', 'moderation-hide', 'ghost-tool', 'another-ghost']))
      .toEqual([])
  })
})

describe('provider tool support', () => {
  const settings = {
    costClassProfiles: { economy: 'cheap', standard: 'missing-profile' },
    profiles: [
      { id: 'cheap', supportsTools: true },
      { id: 'plain', supportsTools: false }
    ]
  }

  test('reports ok when the bound provider supports tools', () => {
    expect(purposeToolSupportState({ costClass: 'economy' }, settings)).toBe('ok')
  })

  test('reports no-profile for an unbound cost class or a dangling binding', () => {
    expect(purposeToolSupportState({ costClass: 'premium' }, settings)).toBe('no-profile')
    expect(purposeToolSupportState({ costClass: 'standard' }, settings)).toBe('no-profile')
    expect(purposeToolSupportState({ costClass: '  ' }, settings)).toBe('no-profile')
  })

  test('reports no-tool-support only when the provider declares false', () => {
    const plainBound = { costClassProfiles: { economy: 'plain' }, profiles: [{ id: 'plain', supportsTools: false }] }
    expect(purposeToolSupportState({ costClass: 'economy' }, plainBound)).toBe('no-tool-support')
    // 缺省（undefined）视为支持：只有明确的 false 才降级。
    const unset = { costClassProfiles: { economy: 'plain' }, profiles: [{ id: 'plain' }] }
    expect(purposeToolSupportState({ costClass: 'economy' }, unset)).toBe('ok')
  })
})

describe('tool catalog filtering', () => {
  const tools: AIToolDisplayLike[] = [
    { name: 'forum-search', description: 'Search topics and posts', origin: { kind: 'core' } },
    {
      name: 'moderation-hide',
      description: 'Hide a topic',
      descriptionI18n: { 'en-US': 'Hide a topic or comment', 'zh-CN': '隐藏主题或评论' },
      origin: { kind: 'core' }
    },
    {
      name: 'vendor-tool',
      description: 'Acme vendor helper',
      origin: { kind: 'plugin', extensionId: 'acme-suite' }
    }
  ]

  test('filters by origin without touching order', () => {
    expect(filterTools(tools, '', 'all').map(tool => tool.name))
      .toEqual(['forum-search', 'moderation-hide', 'vendor-tool'])
    expect(filterTools(tools, '', 'core').map(tool => tool.name))
      .toEqual(['forum-search', 'moderation-hide'])
    expect(filterTools(tools, '', 'plugin').map(tool => tool.name))
      .toEqual(['vendor-tool'])
  })

  test('matches name, description, localized copy and extension id, case-insensitively', () => {
    expect(filterTools(tools, 'HIDE', 'all').map(tool => tool.name)).toEqual(['moderation-hide'])
    expect(filterTools(tools, 'search topics', 'all').map(tool => tool.name)).toEqual(['forum-search'])
    expect(filterTools(tools, '隐藏主题', 'all').map(tool => tool.name)).toEqual(['moderation-hide'])
    expect(filterTools(tools, 'acme', 'all').map(tool => tool.name)).toEqual(['vendor-tool'])
    expect(filterTools(tools, '  ', 'all')).toHaveLength(3)
    expect(filterTools(tools, 'nothing-matches', 'all')).toEqual([])
  })

  test('keeps the origin filter while searching', () => {
    expect(filterTools(tools, 'acme', 'core')).toEqual([])
    expect(filterTools(tools, 'hide', 'plugin')).toEqual([])
  })
})

describe('tool display description pruning and dirty detection', () => {
  test('drops blank text, blank entries and blank names', () => {
    expect(pruneToolDisplayDescriptions(undefined)).toBeUndefined()
    expect(pruneToolDisplayDescriptions({})).toBeUndefined()
    expect(pruneToolDisplayDescriptions({ 'forum-search': {} })).toBeUndefined()
    expect(pruneToolDisplayDescriptions({ 'forum-search': { 'en-US': '   ' } })).toBeUndefined()
    expect(pruneToolDisplayDescriptions({ '   ': { 'en-US': 'Search topics' } })).toBeUndefined()
    expect(pruneToolDisplayDescriptions({ 'forum-search': { '  ': 'Search topics' } })).toBeUndefined()
  })

  test('trims surviving names, locales and text', () => {
    expect(pruneToolDisplayDescriptions({
      ' forum-search ': { ' en-US ': ' Search topics ', 'zh-CN': '  ' }
    })).toEqual({ 'forum-search': { 'en-US': 'Search topics' } })
  })

  test('whitespace-only differences are not a dirty panel', () => {
    const stored = { reply: {}, toolDisplayDescriptions: { 'forum-search': { 'en-US': 'Search topics' } } }
    const form = {
      reply: { systemPrompt: '' },
      toolDisplayDescriptions: { ' forum-search ': { ' en-US ': '  Search topics  ' } }
    }
    expect(canonical(form, DEFAULT_PROMPT)).toBe(canonical(stored, DEFAULT_PROMPT))
  })

  test('clearing a description compares equal to never having set one', () => {
    // 服务端会把空值归一成「没有这个字段」；界面若保留空串，清空之后会一直脏。
    const never = { reply: {} }
    const cleared = { reply: {}, toolDisplayDescriptions: { 'forum-search': { 'en-US': '  ' } } }
    expect(canonical(cleared, DEFAULT_PROMPT)).toBe(canonical(never, DEFAULT_PROMPT))
  })

  test('a real edit is dirty, and so is removing one of two tools', () => {
    const stored = { reply: {}, toolDisplayDescriptions: { 'forum-search': { 'en-US': 'Search topics' } } }
    const edited = { reply: {}, toolDisplayDescriptions: { 'forum-search': { 'en-US': 'Search everything' } } }
    expect(canonical(edited, DEFAULT_PROMPT)).not.toBe(canonical(stored, DEFAULT_PROMPT))

    const twoTools = {
      reply: {},
      toolDisplayDescriptions: {
        'forum-search': { 'en-US': 'Search topics' },
        'time-now': { 'en-US': 'Read the clock' }
      }
    }
    expect(canonical(twoTools, DEFAULT_PROMPT)).not.toBe(canonical(stored, DEFAULT_PROMPT))
  })
})

describe('English description provenance', () => {
  const declared = {
    name: 'forum-search',
    descriptionI18n: { 'en-US': 'Declared English', en: 'Legacy English' }
  }

  test('an operator override wins over the declared text', () => {
    expect(toolDisplayEnglish(declared, { 'forum-search': { 'en-US': 'Operator English' } }))
      .toEqual({ text: 'Operator English', source: 'operator' })
  })

  test('falls back to descriptionI18n en-US, then en', () => {
    expect(toolDisplayEnglish(declared, undefined)).toEqual({ text: 'Declared English', source: 'declared' })
    expect(toolDisplayEnglish({ name: 'x', descriptionI18n: { en: 'Legacy only' } }, {}))
      .toEqual({ text: 'Legacy only', source: 'declared' })
  })

  test('a blank operator value is not a source; nothing anywhere is none', () => {
    expect(toolDisplayEnglish(declared, { 'forum-search': { 'en-US': '   ' } }))
      .toEqual({ text: 'Declared English', source: 'declared' })
    expect(toolDisplayEnglish({ name: 'x' }, undefined)).toEqual({ text: '', source: 'none' })
    expect(toolDisplayEnglish({ name: 'x' }, { x: { 'zh-CN': '运营者中文' } }))
      .toEqual({ text: '', source: 'none' })
  })

  test('the operator map is keyed by the exact tool name', () => {
    expect(toolDisplayEnglish(declared, { 'forum-search-other': { 'en-US': 'Wrong tool' } }).source)
      .toBe('declared')
  })
})

describe('tool origin labels', () => {
  test('labels Core built-ins and plugin contributions', () => {
    expect(toolOriginLabel({ kind: 'core' })).toBe('Core')
    expect(toolOriginLabel({ kind: 'plugin', extensionId: 'acme-suite' })).toBe('@acme-suite')
    expect(toolOriginLabel({ kind: 'plugin' })).toBe('@plugin')
    expect(toolOriginLabel({ kind: 'plugin', extensionId: '   ' })).toBe('@plugin')
    expect(toolOriginLabel(undefined)).toBe('Core')
  })
})

describe('admin AI tools copy', () => {
  const keys = [
    'admin.ai.tabs.tools',
    'admin.ai.tools.title',
    'admin.ai.tools.intro',
    'admin.ai.tools.empty',
    'admin.ai.tools.emptyHint',
    'admin.ai.tools.searchPlaceholder',
    'admin.ai.tools.filterAll',
    'admin.ai.tools.filterCore',
    'admin.ai.tools.filterPlugin',
    'admin.ai.tools.noResults',
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
    'admin.ai.tools.overrideMark',
    'admin.ai.tools.removeEverywhere',
    'admin.ai.tools.removeEverywhereDone',
    'admin.ai.tools.copyName',
    'admin.ai.tools.copied',
    'admin.ai.tools.copyFailed',
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
  ]

  test('both locales ship every key with non-empty copy', () => {
    const zh = locale('zh-CN')
    const en = locale('en-US')
    for (const key of keys) {
      const zhText = lookup(zh, key)
      const enText = lookup(en, key)
      expect(typeof zhText).toBe('string')
      expect(typeof enText).toBe('string')
      expect((zhText as string).trim().length).toBeGreaterThan(0)
      expect((enText as string).trim().length).toBeGreaterThan(0)
    }
  })

  test('the tab name is the agreed wording', () => {
    expect(lookup(locale('zh-CN'), 'admin.ai.tabs.tools')).toBe('模型工具')
    expect(lookup(locale('en-US'), 'admin.ai.tabs.tools')).toBe('Model tools')
  })

  test('the two catalogs agree on the exact tool key set', () => {
    const flatten = (node: unknown, prefix: string): string[] => {
      if (!node || typeof node !== 'object') return []
      return Object.entries(node as Record<string, unknown>).flatMap(([key, value]) => {
        const path = `${prefix}.${key}`
        return value && typeof value === 'object' ? flatten(value, path) : [path]
      })
    }
    const zhKeys = flatten(lookup(locale('zh-CN'), 'admin.ai.tools'), 'admin.ai.tools').sort()
    const enKeys = flatten(lookup(locale('en-US'), 'admin.ai.tools'), 'admin.ai.tools').sort()
    expect(zhKeys).toEqual(enKeys)
    expect(zhKeys).toEqual([...keys].filter(key => key !== 'admin.ai.tabs.tools').sort())
  })
})
