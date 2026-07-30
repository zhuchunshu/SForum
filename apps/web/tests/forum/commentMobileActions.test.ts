import { describe, expect, test } from 'bun:test'
import { readFileSync } from 'node:fs'

const source = (path: string) => readFileSync(new URL(path, import.meta.url), 'utf8')

describe('mobile comment actions', () => {
  test('keeps reply and link inline while moving secondary actions beside the floor', () => {
    const component = source('../../app/components/forum/SFComment.vue')
    const css = source('../../app/assets/css/sforum-comment.css')
    const zh = JSON.parse(source('../../i18n/locales/zh-CN.json'))
    const en = JSON.parse(source('../../i18n/locales/en-US.json'))

    expect(component).toContain("actionItem.value === 'reply' || actionItem.value === 'link'")
    expect(component).toContain("'sf-comment__action--secondary': !isPrimaryAction(actionItem)")
    expect(component).toContain(':items="mobileMenuItems"')
    expect(component).toContain("t('topicDetail.commentMoreActions')")
    expect(css).toMatch(/@media \(max-width: 640px\)[\s\S]*?\.sf-comment__action--secondary \{[\s\S]*?display: none;/)
    expect(css).toMatch(/@media \(max-width: 640px\)[\s\S]*?\.sf-comment__mobile-actions \{[\s\S]*?display: inline-flex;/)
    expect(css).toMatch(/\.sf-comment__floor-actions \.sf-comment__floor \{[\s\S]*?position: static;/)
    expect(zh.topicDetail.commentMoreActions).toBe('更多回复操作')
    expect(en.topicDetail.commentMoreActions).toBe('More reply actions')
  })
})
