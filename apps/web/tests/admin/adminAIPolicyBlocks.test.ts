import { describe, expect, test } from 'bun:test'

import { readFileSync } from 'node:fs'
import { fileURLToPath } from 'node:url'

function source(relative: string): string {
  return readFileSync(fileURLToPath(new URL(relative, import.meta.url)), 'utf8')
}

const providersTab = source('../../app/components/admin/ai/tabs/SFAdminAIProvidersTab.vue')
const autoActionBlock = source('../../app/components/admin/ai/blocks/SFAdminAIAutoActionBlock.vue')
const redactionBlock = source('../../app/components/admin/ai/blocks/SFAdminAIRedactionBlock.vue')

// autoAction 与 redaction 在类型、默认值与后端校验里都存在，但曾经没有任何 UI
// 入口：运营者在提交自己看不见也改不了的策略。这两个区块必须真实挂到页面上。
describe('admin AI policy blocks are reachable', () => {
  test('providers tab mounts both policy blocks', () => {
    expect(providersTab).toContain('SFAdminAIAutoActionBlock')
    expect(providersTab).toContain('SFAdminAIRedactionBlock')
  })

  test('blocks share the page form so edits land in the same payload', () => {
    expect(providersTab).toContain('v-model:form="form"')
    expect(autoActionBlock).toContain("defineModel<AdminAISettings | null>('form'")
    expect(redactionBlock).toContain("defineModel<AdminAISettings | null>('form'")
  })

  test('blocks respect the read-only state', () => {
    expect(providersTab).toContain('v-model:disabled="readOnly"')
    expect(autoActionBlock).toContain("defineModel<boolean>('disabled'")
    expect(redactionBlock).toContain("defineModel<boolean>('disabled'")
  })
})

describe('admin AI auto action block', () => {
  test('offers only the three backend-allowed actions, never delete', () => {
    // ValidAutoAction 只接受 hide/tag/reject；删除类是硬边界，界面不得提供。
    expect(autoActionBlock).toContain("['hide', 'tag', 'reject']")
    expect(autoActionBlock).not.toContain("'delete'")
  })

  test('clamps the confidence threshold to the validated 0..1 range', () => {
    expect(autoActionBlock).toContain('Math.min(1, Math.max(0, parsed))')
  })

  test('warns before the backend would reject an enabled empty action list', () => {
    // 后端在 enabled 且 allowedActions 为空时返回 422，界面必须提前说明。
    expect(autoActionBlock).toContain('needsActions')
    expect(autoActionBlock).toContain("t('admin.ai.autoActionNeedsActions')")
  })
})

describe('admin AI redaction block', () => {
  test('turning the master switch off clears the three sub-options', () => {
    // 「脱敏已启用但什么都没脱」是自相矛盾状态，不应可保存。
    expect(redactionBlock).toContain('function toggleEnabled')
    expect(redactionBlock).toContain('current.redaction.redactEmail = false')
    expect(redactionBlock).toContain('current.redaction.redactPhone = false')
    expect(redactionBlock).toContain('current.redaction.redactIp = false')
  })

  test('sub-options are unreachable while redaction is off', () => {
    expect(redactionBlock).toContain(':disabled="disabled || !form.redaction.enabled"')
  })
})
