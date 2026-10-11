import { describe, expect, test } from 'bun:test'

import { readFileSync } from 'node:fs'
import { fileURLToPath } from 'node:url'

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
const promptTab = source('../../app/components/admin/ai/tabs/SFAdminAIReplyPromptTab.vue')
const providersTab = source('../../app/components/admin/ai/tabs/SFAdminAIProvidersTab.vue')
const draft = source('../../app/composables/admin/useAdminAISettingsForm.ts')
const zh = locale('zh-CN')
const en = locale('en-US')

// 回复提示词从供应商页签搬到独立页签后，两件事必须同时成立：编辑仍然落在同一份
// 草稿上，保存不再要求先切回供应商页签。否则这次搬运会把「改完能保存」弄丢。
describe('admin AI reply prompt tab', () => {
  test('registers the tab in the page shell', () => {
    expect(pageShell).toContain('SFAdminAIReplyPromptTab')
    // 用途页签插在回复提示词与连通性测试之间（见 adminAIPurposes.test.ts）。
    expect(pageShell).toContain("'providers', 'replyPrompt', 'purposes', 'diagnose', 'observability'")
    expect(pageShell).toContain('replyPrompt: SFAdminAIReplyPromptTab')
  })

  test('syncs the tab into the query so a deep link lands on it', () => {
    expect(pageShell).toContain('normalizeTab(route.query.tab)')
    expect(pageShell).toContain('router.replace({')
  })

  test('renders the editor, the restore action, and the fixed safety appendix', () => {
    expect(promptTab).toContain('v-model="form.reply.systemPrompt"')
    expect(promptTab).toContain("t('admin.ai.replyPrompt.restore')")
    expect(promptTab).toContain('ai.safetyAppendix.value')
    expect(promptTab).toContain('restoreDefaultPrompt')
  })

  test('providers tab no longer owns the prompt editor or the save bar', () => {
    expect(providersTab).not.toContain('admin.ai.replyPrompt')
    expect(providersTab).not.toContain('LazySFEditor')
    // 保存按钮搬到了页面壳：留在这里就意味着提示词页签无法保存。
    expect(providersTab).not.toContain("t('admin.ai.save')")
  })

  test('the tab renders content only, never a second page title or toolbar', () => {
    expect(promptTab).not.toContain('UDashboardToolbar')
    expect(promptTab).not.toContain("t('admin.ai.title')")
    expect(promptTab).not.toContain('SFAdminFixedTabNav')
  })
})

describe('admin AI settings draft ownership', () => {
  test('the page shell owns one draft and provides it to every tab', () => {
    expect(pageShell).toContain('useAdminAISettingsForm(ai)')
    expect(pageShell).toContain('provide(ADMIN_AI_DRAFT_KEY, draft)')
  })

  test('both editing tabs read the same injected draft', () => {
    for (const tab of [providersTab, promptTab]) {
      expect(tab).toContain('inject(ADMIN_AI_DRAFT_KEY)!')
    }
  })

  test('the draft keeps the normalization that decides "uses the default"', () => {
    expect(draft).toContain('normalizedAISettingsForComparison')
    expect(draft).toContain('watch(ai.settings')
  })

  test('saving stays reachable from any tab', () => {
    expect(pageShell).toContain('draft.save')
    expect(pageShell).toContain('ai.resetSettings')
    expect(pageShell).toContain("t('admin.ai.dirtyHint')")
  })
})

describe('admin AI reply prompt tab localization', () => {
  test('ships the tab label in both locales', () => {
    expect(lookup(zh, 'admin.ai.tabs.replyPrompt')).toBe('回复提示词')
    expect(lookup(en, 'admin.ai.tabs.replyPrompt')).toBe('Reply prompt')
  })

  test('every key the tab references exists in both locales', () => {
    const keys = [...promptTab.matchAll(/t\('(admin\.ai\.[A-Za-z0-9_.]+)'/g)].map(match => match[1])
    expect(keys.length).toBeGreaterThan(3)
    for (const key of new Set(keys)) {
      expect(`${key}:${typeof lookup(zh, key)}`).toBe(`${key}:string`)
      expect(`${key}:${typeof lookup(en, key)}`).toBe(`${key}:string`)
    }
  })
})
