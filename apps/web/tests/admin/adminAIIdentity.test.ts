import { describe, expect, test } from 'bun:test'

import { readFileSync } from 'node:fs'
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

const block = source('../../app/components/admin/ai/blocks/SFAdminAIIdentityBlock.vue')
const promptTab = source('../../app/components/admin/ai/tabs/SFAdminAIReplyPromptTab.vue')
const composable = source('../../app/composables/admin/useAdminAI.ts')
const contract = source('../../../../contracts/openapi/schemas/ai.yaml')
const zh = locale('zh-CN')
const en = locale('en-US')

const DEFAULT_PROMPT = '你是助手。'

// 模型名只在运行期解析（成本等级 → 供应商绑定），提示词作者写不出它。声明由网关注入，
// 控制台只决定要不要公开——开关缺省必须是「公开」，否则存量站点会原样保留
// 「模型不知道自己是哪个模型」这个缺陷。
describe('admin AI identity block', () => {
  test('is mounted on the reply prompt tab, where the assistant persona lives', () => {
    expect(promptTab).toContain('SFAdminAIIdentityBlock')
    expect(promptTab).toContain('<SFAdminAIIdentityBlock v-model:form="form" :disabled="!canManage" />')
  })

  test('defaults to disclosing, matching the server-side nil semantics', () => {
    // 字段缺失与显式 true 等价：用 ?? true，不能用 === true（那会把缺省读成关闭）。
    expect(block).toContain('?? true')
    expect(block).not.toContain("discloseModel === true")
  })

  test('writes the draft through a getter/setter so mounting is not a change', () => {
    // 直接 v-model 到 form.identity 会在挂载时写入显式值，面板一打开就显示「有未保存
    // 的改动」，而运营者什么都没改。
    expect(block).toContain('computed<boolean>({')
    expect(block).toContain('set: (value: boolean)')
  })

  test('declares the field on the settings type', () => {
    expect(composable).toContain('identity?: { discloseModel?: boolean }')
  })

  test('documents the field in the shared contract', () => {
    expect(contract).toContain('AIIdentitySettings:')
    expect(contract).toContain('discloseModel:')
    // 读与写两个 schema 都要带上它，否则控制台能改、契约里却看不到。
    expect(contract.match(/"\$ref": "#\/AIIdentitySettings"/g)?.length).toBe(2)
  })

  test('ships every key in both locales', () => {
    for (const key of ['identity', 'identityIntro', 'identityDisclose', 'identityHint']) {
      expect(`${key}:${typeof lookup(zh, `admin.ai.${key}`)}`).toBe(`${key}:string`)
      expect(`${key}:${typeof lookup(en, `admin.ai.${key}`)}`).toBe(`${key}:string`)
    }
  })
})

describe('admin AI identity comparison', () => {
  test('treats an explicit "disclose" as equal to the omitted field', () => {
    // 服务端把缺省读成公开；把开关拨回默认后若还判成脏，一次什么都没改的保存
    // 就会把默认值固化成显式配置。
    const stored = { reply: { systemPrompt: '' } }
    const form = { reply: { systemPrompt: '' }, identity: { discloseModel: true } }
    expect(JSON.stringify(normalizedAISettingsForComparison(form, DEFAULT_PROMPT)))
      .toBe(JSON.stringify(normalizedAISettingsForComparison(stored, DEFAULT_PROMPT)))
  })

  test('keeps an explicit "do not disclose" as a real change', () => {
    const stored = { reply: { systemPrompt: '' } }
    const form = { reply: { systemPrompt: '' }, identity: { discloseModel: false } }
    expect(JSON.stringify(normalizedAISettingsForComparison(form, DEFAULT_PROMPT)))
      .not.toBe(JSON.stringify(normalizedAISettingsForComparison(stored, DEFAULT_PROMPT)))
    expect(JSON.stringify(normalizedAISettingsForComparison(form, DEFAULT_PROMPT)))
      .toContain('"discloseModel":false')
  })
})
