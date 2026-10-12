import { describe, expect, test } from 'bun:test'
import { readFileSync } from 'node:fs'
import { forumMentionPattern, forumMentionPatternSource } from '../../app/utils/forum/forumMentions'

// 前后端提及正则的契约测试：前端跑与 Go 侧 mentions_test.go 相同的输入串，
// 断言双方对"哪些 token 算提及"的语义一致。
// 已知差异（有意保留，见 Go TestMentionedUsernamesLengthBoundary 注释）：
// 超出 64 上限的连续串，RE2 侧截断成前缀，前端用尾部否定前瞻整段放弃高亮；
// 两种行为都保证"不会为不存在的用户名触发通知/高亮承诺"。

function extract(text: string): string[] {
  const pattern = forumMentionPattern()
  const result: string[] = []
  let match = pattern.exec(text)
  while (match) {
    if (match[1]) {
      result.push(match[1])
    }
    match = pattern.exec(text)
  }
  return result
}

describe('mention regex cross-implementation contract', () => {
  test('markdown text + dedup semantics align with the Go case', () => {
    // Go: hello @Alice and @张三, @approval_parent, again @alice + 代码块忽略
    // 前端契约作用在"渲染后的可见文本"上（后端 goldmark 已剥掉代码块，DOM linkify
    // 再跳过 pre/code 元素），所以这里用 goldmark 输出的可见文本对照。
    const visibleText = 'hello @Alice and @张三, @approval_parent, again @alice'
    // 前端不做大小写去重（那是后端通知扇出的职责），只断言解析出的 token 集合（保持出现顺序）。
    expect(extract(visibleText).map(token => token.toLowerCase()))
      .toEqual(['alice', '张三', 'approval_parent', 'alice'])
    // 渲染层语义：代码块/行内代码经 DOM 跳过规则不产生链接（见 linkifyForumMentions SKIPPED_TAGS）。
    const source = forumMentionPatternSource()
    expect(source).toContain('\\p{L}\\p{N}_-]')
  })

  test('hyphenated/underscored usernames and email exclusion align with the Go case', () => {
    const input = 'ping @zhang-san and @li_wei, email user@example.com stays out'
    expect(extract(input)).toEqual(['zhang-san', 'li_wei'])
  })

  test('64-length boundary: exact ok, overlong abandoned (documented divergence)', () => {
    const max = 'a'.repeat(64)
    expect(extract('@' + max)).toEqual([max])
    expect(extract('@' + max + 'b')).toEqual([])
  })

  test('the Go test file keeps the same input strings (drift guard)', () => {
    const goTest = readFileSync(
      new URL('../../../api/app/Models/Forum/mentions_test.go', import.meta.url),
      'utf8'
    )
    expect(goTest).toContain('hello @Alice and @张三, @approval_parent, again @alice')
    expect(goTest).toContain('ping @zhang-san and @li_wei, email user@example.com stays out')
  })

  test('pattern source keeps the documented token charset and 64 cap', () => {
    const source = forumMentionPatternSource()
    expect(source).toContain('{1,64}')
    expect(source).toContain('\\p{L}\\p{N}_-]')
  })
})
