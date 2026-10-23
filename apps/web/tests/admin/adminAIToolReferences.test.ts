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

const block = source('../../app/components/admin/ai/blocks/SFAdminAIToolReferencesBlock.vue')
const toolsTab = source('../../app/components/admin/ai/tabs/SFAdminAIToolsTab.vue')
const zh = locale('zh-CN')
const en = locale('en-US')

// 引用区承担两件互相关联的事：清理「白名单里有、登记表里没有」的死名字，以及在
// 插件安装前先把工具名写进白名单（后端保存时刻意不校验存在性）。它是工具页签的
// 子区块，只读写页面壳注入的同一份草稿。
describe('admin AI tool references block', () => {
  test('shares the page draft and never owns data fetching or chrome', () => {
    expect(block).toContain('inject(ADMIN_AI_DRAFT_KEY)!')
    expect(block).toContain('const { form, ai, readOnly } = draft')
    expect(block).not.toContain('useAsyncData')
    expect(block).not.toContain('UDashboardToolbar')
    expect(block).not.toContain('<h2')
    expect(block).not.toContain('SFAdminFixedTabNav')
  })

  test('is rendered by the tools tab instead of the tab keeping a copy', () => {
    expect(toolsTab).toContain('import SFAdminAIToolReferencesBlock')
    expect(toolsTab).toContain('<SFAdminAIToolReferencesBlock />')
  })

  test('lists referenced-but-unregistered names and removes them per purpose', () => {
    expect(block).toContain('referencedButUnregistered(catalog.value, form.value?.purposes, tools.value.map(tool => tool.name))')
    expect(block).toContain('setPurposeTool(purpose.declaredTools, purposes[id], entry.name, false)')
    expect(block).toContain('admin.ai.tools.unregisteredTitle')
    expect(block).toContain('admin.ai.tools.unregisteredHint')
    expect(block).toContain('admin.ai.tools.unregisteredRemove')
    expect(block).toContain('admin.ai.tools.removeEverywhereDone')
  })

  test('only names tools that are not registered yet', () => {
    // 已注册工具走矩阵勾选；手动入口只服务「先配策略、后装插件」。
    expect(block).toContain('registeredNames.value.has(name)')
    expect(block).toContain('admin.ai.tools.manualAdd.registered')
    expect(block).toContain('isValidPurposeToolName(name)')
    expect(block).toContain('admin.ai.tools.manualAdd.nameInvalid')
  })

  test('caps additions at what one request can carry but always allows removal', () => {
    expect(block).toContain('effective.length >= MAX_TOOLS_PER_REQUEST')
    expect(block).toContain('admin.ai.tools.tooMany')
    expect(MAX_TOOLS_PER_REQUEST).toBe(16)
    // 移除路径没有上限检查。
    expect(block).toContain('setPurposeTool(purpose.declaredTools, purposes[id], entry.name, false)')
  })

  test('does not rewrite an allowlist that already contains the name', () => {
    expect(block).toContain('effective.includes(name)')
    expect(block).toContain('admin.ai.tools.manualAdd.alreadyPresent')
  })

  test('writes the settings-document shape through the shared adapter', () => {
    expect(block).toContain('toSettingsOverride(next)')
    expect(block).toContain('toSettingsOverride(setPurposeTool(')
  })

  test('honours the read-only permission gate on every editing control', () => {
    expect(block).toContain('if (!current || readOnly.value) return')
    expect(block).toContain(':disabled="readOnly"')
  })

  test('labels compact controls for assistive technology', () => {
    expect(block).toContain(':aria-label="t(\'admin.ai.tools.manualAdd.purpose\')"')
    expect(block).toContain(':aria-label="t(\'admin.ai.tools.manualAdd.add\')"')
    expect(block).toContain(':aria-label="t(\'admin.ai.tools.unregisteredRemove\', { name: entry.name })"')
  })

  test('renders text through interpolation only', () => {
    expect(block).not.toContain('v-html')
    expect(/[\p{Extended_Pictographic}]/u.test(block)).toBe(false)
  })
})

describe('admin AI tool references localization', () => {
  test('ships every key the block references in both locales', () => {
    const keys = new Set<string>()
    for (const match of block.matchAll(/\bt\('([A-Za-z0-9_.]+)'/g)) {
      keys.add(match[1]!)
    }
    expect(keys.size).toBeGreaterThan(15)
    for (const key of keys) {
      expect(`${key}:${typeof lookup(zh, key)}`).toBe(`${key}:string`)
      expect(`${key}:${typeof lookup(en, key)}`).toBe(`${key}:string`)
    }
  })
})
