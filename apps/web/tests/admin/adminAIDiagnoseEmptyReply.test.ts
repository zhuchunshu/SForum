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

const diagnoseTab = source('../../app/components/admin/ai/tabs/SFAdminAIDiagnoseTab.vue')
const composable = source('../../app/composables/admin/useAdminAI.ts')
const zh = locale('zh-CN')
const en = locale('en-US')

// 推理类模型会先把输出预算消耗在思考上，于是「调用成功但正文为空」是一种常见
// 结果。它既不是连接正常也不是连接失败，界面必须单独表达，否则运营者只会看到
// 一条空白的回复条。
describe('admin AI diagnose empty reply', () => {
  test('localizes the empty reply outcome in both locales', () => {
    for (const catalog of [zh, en]) {
      expect(lookup(catalog, 'admin.ai.diagnose.emptyReply')).toBeTruthy()
      expect(lookup(catalog, 'admin.ai.diagnose.emptyReplyToast')).toBeTruthy()
      expect(lookup(catalog, 'admin.ai.diagnose.reasons.empty_reply')).toBeTruthy()
    }
  })

  test('distinguishes an empty reply from a normal success', () => {
    // 空正文必须有独立的判定与渲染分支：不能沿用「ok 就显示 reply」的老路径。
    expect(diagnoseTab).toContain('emptyReply')
    expect(diagnoseTab).toContain("t('admin.ai.diagnose.emptyReply')")
    // 正文条只在确实有正文时渲染，避免留下一条空白块。
    expect(diagnoseTab).toContain('v-if="ai.diagnoseResult.value.ok && !emptyReply"')
  })

  test('treats reply as a string the result always carries', () => {
    // 空字符串与字段缺失必须可区分：后端去掉了 reply 的 omitempty。
    expect(composable).toContain('reply: string')
    expect(composable).toContain('stopReason?: string')
  })

  test('toasts a warning instead of claiming success when the reply is empty', () => {
    expect(composable).toContain("result.reason === 'empty_reply'")
    expect(composable).toContain("t('admin.ai.diagnose.emptyReplyToast')")
  })
})
