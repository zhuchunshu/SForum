import { describe, expect, test } from 'bun:test'
import { readFileSync } from 'node:fs'

const themeCss = readFileSync(
  new URL('../../../../extensions/builtin/themes/sforum-default/assets/hybrid-forum.css', import.meta.url),
  'utf8'
)
const hostTopicCss = readFileSync(new URL('../../app/assets/css/sforum-topic.css', import.meta.url), 'utf8')
const hostMobileCss = hostTopicCss.slice(hostTopicCss.lastIndexOf('@media (max-width: 640px)'))
const themeMobileCss = themeCss.slice(themeCss.lastIndexOf('@media (max-width: 640px)'))

describe('default theme topic typography', () => {
  test('keeps readable content and metadata sizes', () => {
    expect(themeCss).toContain('--sf-public-text-size-caption: 12px')
    expect(themeCss).toContain('--sf-public-text-size-meta: 14px')
    expect(themeCss).toContain('--sf-public-text-size-control: 14px')
    expect(themeCss).toContain('--sf-public-text-size-body: 16px')
    expect(hostTopicCss).toContain('font-size: var(--sf-public-text-size-body, 14px)')

    expect(themeCss).toMatch(/\.sf-theme--default \.sf-topic-heading__byline time,[\s\S]*?color: var\(--sf-public-text-secondary\);[\s\S]*?font-size: var\(--sf-public-text-size-meta\);/)
    expect(themeCss).toMatch(/\.sf-theme--default \.sforum-topic-page__prose,[\s\S]*?font-size: var\(--sf-public-text-size-body\);/)
    expect(themeCss).toMatch(/\.sf-theme--default \.sf-comment__content \{[\s\S]*?font-size: var\(--sf-public-text-size-body\);/)
    expect(themeCss).toMatch(/\.sf-theme--default \.sf-topic-side-card--page-nav a \{[\s\S]*?font-size: var\(--sf-public-text-size-meta\);/)
  })

  test('keeps the mobile discussion heading as a compact desktop-style row', () => {
    expect(hostMobileCss).toMatch(/\.sf-comment-stream-controls \{[\s\S]*?align-items: center;[\s\S]*?flex-direction: row;/)
    expect(themeMobileCss).toMatch(/\.sf-theme--default \.sforum-topic-comments \{[\s\S]*?margin-top: 24px;/)
    expect(themeMobileCss).toMatch(/\.sf-theme--default \.sf-comment-stream-controls \{[\s\S]*?min-height: 44px;[\s\S]*?flex-direction: row;[\s\S]*?padding-bottom: 0;/)
  })
})
