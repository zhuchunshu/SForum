import { describe, expect, test } from 'bun:test'
import { readFileSync } from 'node:fs'
import { join } from 'node:path'
import { replyOutcomeColor, replyOutcomeHref } from '../../app/components/admin/ai/replyOutcomes'

const localeDir = join(import.meta.dir, '../../i18n/locales')

function localeKeys(name: string): Record<string, unknown> {
  return JSON.parse(readFileSync(join(localeDir, name), 'utf8')) as Record<string, unknown>
}

// 「回复有没有出现」是这次故障的核心问题：控制台此前只有执行记录（每次模型调用），
// 于是一列 succeeded 与「社区没有回复」可以同时为真。这些断言钉住那半块缺失的面：
// 每个结果码都有中文文案、颜色只把 posted 当成功、链接永远指向真实楼层。
describe('AI reply outcomes', () => {
  test('treats only posted as success', () => {
    expect(replyOutcomeColor('posted')).toBe('success')
    expect(replyOutcomeColor('empty_reply')).toBe('warning')
    expect(replyOutcomeColor('tool_loop_stuck')).toBe('warning')
    expect(replyOutcomeColor('unavailable')).toBe('error')
    expect(replyOutcomeColor('publish_failed')).toBe('error')
    expect(replyOutcomeColor('failed')).toBe('error')
    // 未知取值不得伪装成成功：历史行或新结果码都应落在中性色上。
    expect(replyOutcomeColor('something-new')).toBe('neutral')
  })

  test('links to the topic and only anchors a real comment', () => {
    expect(replyOutcomeHref({ topicId: 12 })).toBe('/t/12')
    expect(replyOutcomeHref({ topicId: 12, commentId: 501 })).toBe('/t/12#comment-501')
    // 未发布时 commentId 是 0（Go 侧 omitempty 也可能整个字段缺失）：不得拼出
    // 一个指向 #comment-0 的死锚点。
    expect(replyOutcomeHref({ topicId: 12, commentId: 0 })).toBe('/t/12')
    expect(replyOutcomeHref({ topicId: 12, commentId: null })).toBe('/t/12')
    // 没有主题就没有可点的落点，调用方渲染纯文本。
    expect(replyOutcomeHref({ topicId: 0 })).toBe('')
  })

  test('ships localized text for every outcome code in both locales', () => {
    const zh = localeKeys('zh-CN.json') as { admin: { ai: Record<string, unknown> } }
    const en = localeKeys('en-US.json') as { admin: { ai: Record<string, unknown> } }
    const codes = ['posted', 'empty_reply', 'tool_loop_stuck', 'unavailable', 'publish_failed', 'failed']
    for (const [name, locale] of [['zh-CN', zh], ['en-US', en]] as const) {
      const block = locale.admin.ai.replyOutcome as Record<string, string> | undefined
      expect(block, `${name}: admin.ai.replyOutcome`).toBeDefined()
      for (const code of codes) {
        expect(block?.[code], `${name}: ${code}`).toBeTruthy()
      }
      // 区块自身的标题与列名同样必须有：少一个键会在界面上渲染出原始 key。
      for (const key of ['replyOutcomes', 'replyOutcomesIntro', 'replyOutcomesEmpty', 'colTrigger', 'colOutcome', 'colReason', 'colCalls', 'cacheBadge', 'cacheHitHint', 'filterCacheHit']) {
        expect(locale.admin.ai[key], `${name}: admin.ai.${key}`).toBeTruthy()
      }
    }
  })
})
