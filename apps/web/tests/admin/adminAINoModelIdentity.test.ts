import { describe, expect, test } from 'bun:test'

import { existsSync, readFileSync } from 'node:fs'
import { fileURLToPath } from 'node:url'

import { normalizedAISettingsForComparison } from '../../app/utils/admin/adminAI'

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

const contract = source('../../../../contracts/openapi/schemas/ai.yaml')
const composable = source('../../app/composables/admin/useAdminAI.ts')
const promptTab = source('../../app/components/admin/ai/tabs/SFAdminAIReplyPromptTab.vue')
const zh = locale('zh-CN')
const en = locale('en-US')

const DEFAULT_PROMPT = '你是助手。'

// 站内 AI 回答「你是哪个模型」时，内容只能来自模型自身：网关不把供应商标识、
// 模型名、协议名或「由本站网关调用」写进 system。这个契约没有开关——任何一层
// 重新引入身份注入都是回归，因此这组断言覆盖后端契约、控制台与文案三层。
describe('model identity is never injected', () => {
  test('the console no longer carries an identity block', () => {
    const block = fileURLToPath(new URL('../../app/components/admin/ai/blocks/SFAdminAIIdentityBlock.vue', import.meta.url))
    expect(existsSync(block)).toBe(false)
    expect(promptTab).not.toContain('SFAdminAIIdentityBlock')
  })

  test('the settings type and both locales carry no identity switch', () => {
    expect(composable).not.toContain('discloseModel')
    for (const key of ['identity', 'identityIntro', 'identityDisclose', 'identityHint']) {
      expect(lookup(zh, `admin.ai.${key}`)).toBeUndefined()
      expect(lookup(en, `admin.ai.${key}`)).toBeUndefined()
    }
  })

  test('the shared contract exposes no identity schema', () => {
    expect(contract).not.toContain('AIIdentitySettings')
    expect(contract).not.toContain('discloseModel')
  })
})

// 旧后端回传或旧草稿里残留的 identity 必须被剥掉：一个已经不存在的开关不该继续
// 让面板显示「有未保存的改动」，也不该被下一次保存原样写回去。
describe('legacy identity values are pruned when comparing settings', () => {
  test('an explicit legacy value compares equal to a document without it', () => {
    const stored = { reply: { systemPrompt: '' } }
    const form = { reply: { systemPrompt: '' }, identity: { discloseModel: false } }
    const normalized = JSON.stringify(normalizedAISettingsForComparison(form, DEFAULT_PROMPT))
    expect(normalized).toBe(JSON.stringify(normalizedAISettingsForComparison(stored, DEFAULT_PROMPT)))
    expect(normalized).not.toContain('discloseModel')
  })
})
