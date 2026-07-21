import { describe, expect, test } from 'bun:test'
import { readFileSync } from 'node:fs'

const source = (path: string) => readFileSync(new URL(path, import.meta.url), 'utf8')

describe('unified avatar rendering contract', () => {
  test('public and admin user chrome render through SFAvatar', () => {
    const navbar = source('../../../apps/web/app/components/SFNavbar.vue')
    const adminLayout = source('../app/layouts/admin.vue')

    expect(navbar).toContain('<SFAvatar')
    expect(navbar).toContain(':avatar="user.avatar"')
    expect(navbar).not.toContain('navbar__avatar">{{ avatarLetter }}')

    expect(adminLayout).toContain('<SFAvatar')
    expect(adminLayout).toContain(':avatar="user?.avatar"')
    expect(adminLayout).not.toContain('<UAvatar')
  })

  test('forum surfaces that show avatars pass AvatarView into SFAvatar', () => {
    const avatar = source('../app/components/SFAvatar.vue')
    const homepageRow = source('../../../apps/web/app/components/SFHomeTopicRow.vue')
    const topicPage = source('../../../apps/web/app/components/SFTopicShowPage.vue')
    const topicHeading = source('../../../apps/web/app/components/SFTopicHeading.vue')
    const feedRow = source('../app/components/SFFeedRow.vue')
    const comment = source('../app/components/SFComment.vue')

    expect(homepageRow).toContain('<SFAvatar')
    expect(homepageRow).toContain(':avatar="topic.author?.avatar"')
    expect(homepageRow).toContain('size="list"')
    // 列表不得绕过 AvatarView（禁止 prefer-initials 强制字头、局部改尺寸）
    expect(homepageRow).not.toContain('prefer-initials')
    expect(homepageRow).not.toContain('preferInitials')
    expect(homepageRow).not.toContain('!size-9')
    expect(homepageRow).not.toContain('topic.excerpt')
    expect(homepageRow).not.toContain('participants')
    // 尺寸/字头色板/远程探测集中在全局 SFAvatar；URL 策略由后端 AvatarView 决定
    expect(avatar).toContain("'list'")
    expect(avatar).toContain('forumAvatarToneClass')
    expect(avatar).toContain('resolveImage')
    expect(avatar).toContain('isRemoteImage')
    expect(avatar).toContain('imageReady')
    expect(avatar).toContain('props.alt ?? props.avatar?.alt ?? props.name')
    expect(avatar).toContain(":aria-hidden=\"isDecorative ? 'true' : undefined\"")
    expect(avatar).not.toContain('preferInitials')
    expect(avatar).not.toContain('identicon')
    // 详情页 byline 不得用 text ellipsis 误伤 .sf-avatar
    const topicCss = source('../app/assets/css/sforum-topic.css')
    expect(topicCss).toContain('.sf-topic-heading__author > span:not(.sf-avatar)')
    expect(topicCss).not.toMatch(/\.sf-topic-heading__author > span\s*\{/)
    // 详情页标题 / 评论 / 侧卡同样只走 SFAvatar + AvatarView
    expect(topicHeading).toContain('<SFAvatar')
    expect(topicHeading).toContain(':avatar="topic.author?.avatar"')
    expect(topicPage).toContain(':avatar="comment.author?.avatar"')
    expect(topicPage).toContain('<SFTopicSideCard')
    expect(feedRow).toContain('avatar?: AvatarView | null')
    expect(feedRow).toContain(':avatar="avatar"')
    expect(comment).toContain('<SFAvatar')
    expect(comment).toContain('avatar?: AvatarView | null')
    expect(comment).toContain(':avatar="avatar"')
    expect(comment).toContain(':avatar="child.author?.avatar"')
  })
})
