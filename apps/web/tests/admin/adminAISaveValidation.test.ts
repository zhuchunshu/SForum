import { describe, expect, test } from 'bun:test'

import { readFileSync } from 'node:fs'
import { fileURLToPath } from 'node:url'

import {
  DEFAULT_AI_LIMITS,
  coerceInteger,
  coerceNumber,
  isValidEndpointUrl,
  profileIssue,
  utf8ByteLength
} from '../../app/utils/admin/adminAI'

function source(relative: string): string {
  return readFileSync(fileURLToPath(new URL(relative, import.meta.url)), 'utf8')
}

const providersTab = source('../../app/components/admin/ai/tabs/SFAdminAIProvidersTab.vue')
const pageShell = source('../../app/pages/admin/ai.vue')

// 空串与小数是「保存报 422」最直接的来源：`<input type="number">` 被清空时读回 ""，
// Nuxt UI 的 looseToNumber 会把它原样写进模型，PUT 出去就是 `"maxTokens":""`，
// Go 侧 int 解码失败。coerceInteger 的作用是让载荷里永远不出现空串。
describe('admin AI numeric coercion', () => {
  test('empty input falls back to the previous value instead of producing ""', () => {
    expect(coerceInteger('', 2048, 1, 128000)).toBe(2048)
    expect(coerceInteger('   ', 30000, 1000, 120000)).toBe(30000)
    expect(coerceInteger(null, 7, 0, 10)).toBe(7)
    expect(coerceInteger(undefined, 7, 0, 10)).toBe(7)
  })

  test('decimals are rounded into an integer the server can decode', () => {
    expect(coerceInteger('12.5', 0, 0, 100)).toBe(13)
    expect(coerceInteger(12.4, 0, 0, 100)).toBe(12)
  })

  test('out-of-range values clamp to the server ceilings', () => {
    expect(coerceInteger(999999, 0, 0, DEFAULT_AI_LIMITS.maxRateLimitPerMinute)).toBe(DEFAULT_AI_LIMITS.maxRateLimitPerMinute)
    expect(coerceInteger(-5, 0, 0, 100)).toBe(0)
    expect(coerceInteger(1e21, 42, 0, 100)).toBe(42)
  })

  test('non-numeric text falls back rather than sending NaN', () => {
    expect(coerceInteger('abc', 9, 0, 10)).toBe(9)
    expect(coerceNumber('abc', 1.5, 0, 10)).toBe(1.5)
    expect(coerceNumber('2.75', 0, 0, 10)).toBe(2.75)
  })
})

describe('admin AI profile validation mirrors the server', () => {
  test('a URL without a scheme is not "complete"', () => {
    expect(isValidEndpointUrl('api.deepseek.com')).toBe(false)
    expect(profileIssue({ label: 'Custom', baseUrl: 'api.deepseek.com', model: 'gpt' })).toBe('baseUrl')
  })

  test('a URL carrying credentials is rejected before saving', () => {
    expect(isValidEndpointUrl('https://user:key@api.example.com')).toBe(false)
    expect(isValidEndpointUrl('https://api.example.com/v1')).toBe(true)
  })

  test('label limits are measured in bytes, like Go len()', () => {
    expect(utf8ByteLength('模')).toBe(3)
    expect(profileIssue({ label: '模'.repeat(22), baseUrl: 'https://api.example.com', model: 'gpt' })).toBe('label')
    expect(profileIssue({ label: '模'.repeat(21), baseUrl: 'https://api.example.com', model: 'gpt' })).toBe(null)
  })

  test('missing model is reported as model, not as a generic failure', () => {
    expect(profileIssue({ label: 'Custom', baseUrl: 'https://api.example.com', model: '  ' })).toBe('model')
  })
})

// 上下限只有一个权威来源：服务端下发的 limits。前端再硬编码一份数字，就是
// 「界面允许、服务端 422」的成因。
describe('admin AI limits come from the server', () => {
  test('the providers tab reads limits instead of literal ceilings', () => {
    expect(providersTab).toContain('const limits = ai.limits')
    expect(providersTab).toContain('limits.value.maxRateLimitPerMinute')
    expect(providersTab).toContain('limits.value.maxCompletionTokens')
    expect(providersTab).not.toContain('v-model.number="selectedProfile.defaults.maxTokens"')
    expect(providersTab).not.toContain('v-model.number="selectedProfile.defaults.timeoutMs"')
  })

  test('the payload type exposes limits and field errors', () => {
    const composable = source('../../app/composables/admin/useAdminAI.ts')
    expect(composable).toContain('limits?: AdminAILimits')
    expect(composable).toContain('const limits = computed<AdminAILimits>')
    expect(composable).toContain('fieldErrors')
  })
})

// 保存前拦截：这两类载荷一定会被服务端 422，界面必须说清楚该改哪一项，
// 而不是先把请求发出去再显示一个没有指向的错误。
describe('admin AI save gating', () => {
  test('an enabled but invalid profile blocks the request', () => {
    const form = source('../../app/composables/admin/useAdminAISettingsForm.ts')
    expect(form).toContain('profileIssue')
    expect(form).toContain('blockingIssue')
    expect(form).toContain('saveIssue.value = blocking')
  })

  test('enabling auto action without actions fills the safe default', () => {
    const form = source('../../app/composables/admin/useAdminAISettingsForm.ts')
    expect(form).toContain('ensureAutoActionActions')
    expect(form).toContain("current.autoAction.allowedActions = ['hide']")
  })

  test('legacy reply.tools is not echoed back', () => {
    const form = source('../../app/composables/admin/useAdminAISettingsForm.ts')
    expect(form).toContain('delete (payload.reply as { tools?: unknown }).tools')
  })

  test('the shell surfaces local and server-side reasons', () => {
    expect(pageShell).toContain('draft.saveIssue.value')
    expect(pageShell).toContain('ai.fieldErrors.value')
  })
})

// 「恢复默认提示词」曾经整体替换 reply 对象，静默把 autonomous/off 改回 mention。
describe('admin AI reply prompt restore keeps the trigger mode', () => {
  test('restore spreads the existing reply instead of replacing it', () => {
    const tab = source('../../app/components/admin/ai/tabs/SFAdminAIReplyPromptTab.vue')
    expect(tab).toContain('{ ...form.value.reply, systemPrompt: ai.defaultSystemPrompt.value }')
    expect(tab).not.toContain('form.value.reply = { systemPrompt: ai.defaultSystemPrompt.value }')
  })
})
