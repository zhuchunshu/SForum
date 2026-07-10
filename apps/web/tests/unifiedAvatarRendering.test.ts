import { describe, expect, test } from 'bun:test'
import { readFileSync } from 'node:fs'

const source = (path: string) => readFileSync(new URL(path, import.meta.url), 'utf8')

describe('unified avatar rendering contract', () => {
  test('public and admin user chrome render through SFAvatar', () => {
    const navbar = source('../../../extensions/builtin/themes/sforum-default/layer/app/components/SFNavbar.vue')
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
    const homepageRow = source('../../../extensions/builtin/themes/sforum-default/layer/app/components/SFHomeTopicRow.vue')
    const topicPage = source('../../../extensions/builtin/themes/sforum-default/layer/app/pages/t/[...path].vue')
    const topicHeading = source('../../../extensions/builtin/themes/sforum-default/layer/app/components/SFTopicHeading.vue')
    const feedRow = source('../app/components/SFFeedRow.vue')
    const comment = source('../app/components/SFComment.vue')

    expect(homepageRow).toContain('topic.excerpt')
    expect(homepageRow).not.toContain('<SFAvatar')
    expect(homepageRow).not.toContain(':aria-label="t(\'home.feed.repliesColumn\')"')
    expect(homepageRow).not.toContain('participants')
    expect(avatar).toContain('props.alt ?? props.avatar?.alt ?? props.name')
    expect(avatar).toContain(":aria-hidden=\"isDecorative ? 'true' : undefined\"")
    expect(topicHeading).toContain(':avatar="topic.author?.avatar"')
    expect(topicPage).toContain(':avatar="comment.author?.avatar"')
    expect(feedRow).toContain('avatar?: AvatarView | null')
    expect(feedRow).toContain(':avatar="avatar"')
    expect(comment).toContain('avatar?: AvatarView | null')
    expect(comment).toContain(':avatar="avatar"')
    expect(comment).toContain(':avatar="child.author?.avatar"')
  })
})
