import { describe, expect, test } from 'bun:test'

import { readFileSync } from 'node:fs'
import { fileURLToPath } from 'node:url'

const providersTabSource = readFileSync(
  fileURLToPath(new URL('../../app/components/admin/ai/tabs/SFAdminAIProvidersTab.vue', import.meta.url)),
  'utf8'
)

// 密钥保存必须先被加密能力拦住：后端已经 fail closed，前端若不同步禁用，
// 运营者就会在填完密钥之后才看到 422。
describe('admin AI credential encryption gate', () => {
  test('disables the credential input and save button without encryption', () => {
    expect(providersTabSource).toContain(":disabled=\"!canManage || !ai.encryptionEnabled.value\"")
    expect(providersTabSource).toContain("!canManage || !ai.encryptionEnabled.value || !(credentials[selectedProfile.id] || '').trim()")
  })

  test('blocks submission even if the button is bypassed', () => {
    expect(providersTabSource).toContain("if (!ai.encryptionEnabled.value) return")
  })

  test('explains the blocked state next to the field instead of only in a toast', () => {
    expect(providersTabSource).toContain("t('admin.ai.credentialBlockedHint')")
  })
})
