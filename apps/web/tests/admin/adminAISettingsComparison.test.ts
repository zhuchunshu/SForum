import { describe, expect, test } from 'bun:test'

import { normalizedAISettingsForComparison, promptUsesDefault } from '../../app/utils/admin/adminAI'

const DEFAULT_PROMPT = '你是助手。\n\n要求：\n- 简短\n- 用中文'

describe('admin AI prompt comparison', () => {
  test('treats empty and default-equal prompts as using the default', () => {
    expect(promptUsesDefault('', DEFAULT_PROMPT)).toBe(true)
    expect(promptUsesDefault(undefined, DEFAULT_PROMPT)).toBe(true)
    expect(promptUsesDefault(DEFAULT_PROMPT, DEFAULT_PROMPT)).toBe(true)
  })

  test('ignores whitespace differences the editor introduces', () => {
    // 编辑器会在段落与列表之间补空行、去掉行尾空白：这些不算自定义。
    const editorRoundTrip = `${DEFAULT_PROMPT.replace('\n- ', '\n\n- ')}\n`
    expect(promptUsesDefault(editorRoundTrip, DEFAULT_PROMPT)).toBe(true)
  })

  test('keeps a real customization distinguishable', () => {
    expect(promptUsesDefault('你是助手。\n\n要求：\n- 简短', DEFAULT_PROMPT)).toBe(false)
    expect(promptUsesDefault('完全不同的提示词', DEFAULT_PROMPT)).toBe(false)
  })

  test('canonicalizes default prompts so empty-vs-omitted compares equal', () => {
    // 服务端在提示词为空时省略字段，表单侧可能持有显式空串。
    const stored = { reply: {}, gates: { toolCallsPerReply: 3 } }
    const form = { reply: { systemPrompt: '' }, gates: { toolCallsPerReply: 3 } }
    expect(JSON.stringify(normalizedAISettingsForComparison(form, DEFAULT_PROMPT)))
      .toBe(JSON.stringify(normalizedAISettingsForComparison(stored, DEFAULT_PROMPT)))
  })

  test('does not collapse a customized prompt into the default', () => {
    const stored = { reply: {} }
    const customized = { reply: { systemPrompt: `${DEFAULT_PROMPT}\n- 另外补充一句` } }
    expect(JSON.stringify(normalizedAISettingsForComparison(customized, DEFAULT_PROMPT)))
      .not.toBe(JSON.stringify(normalizedAISettingsForComparison(stored, DEFAULT_PROMPT)))
  })
})
