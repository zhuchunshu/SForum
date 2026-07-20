import { describe, expect, test } from 'bun:test'
import { readFileSync } from 'node:fs'
import { join } from 'node:path'

const root = join(import.meta.dir, '..')
const read = (rel: string) => readFileSync(join(root, rel), 'utf8')

describe('remaining public presentation ownership', () => {
  const surfaces = [
    ['app/pages/t/[...path].vue', 'forum.topic.show', 'SFTopicShowPage', 'forum.component.topic_show', 'topic-show.html'],
    ['app/pages/topics/new.vue', 'forum.topic.create', 'SFTopicComposerPage', 'forum.component.topic_composer', 'topic-create.html'],
    ['app/pages/u/[username].vue', 'forum.profile.show', 'SFProfileShowPage', 'forum.component.profile_show', 'profile-show.html'],
    ['app/pages/my/index.vue', 'forum.my.home', 'SFMyHomePage', 'forum.component.my_home', 'my-home.html'],
    ['app/pages/my/content-review.vue', 'forum.my.content_review', 'SFMyContentReviewPage', 'forum.component.my_content_review', 'my-content-review.html'],
    ['app/pages/notifications.vue', 'forum.notifications', 'SFNotificationsPage', 'forum.component.notifications', 'notifications.html'],
    ['app/pages/settings/profile.vue', 'forum.settings.profile', 'SFProfileSettingsPage', 'profile.component.settings_form', 'settings-profile.html'],
    ['app/pages/settings/security.vue', 'forum.settings.security', 'SFSecuritySettingsPage', 'identity.component.security_settings', 'settings-security.html'],
  ] as const

  for (const [route, pageId, island, componentId, template] of surfaces) {
    test(`${pageId} route is thin shell with island fallback`, () => {
      const src = read(route)
      expect(src).toContain('SFPageOutlet')
      expect(src).toContain(`page="${pageId}"`)
      expect(src).toContain(`<${island}`)
      // fat markup should not live on the route shell
      expect(src.split('\n').length).toBeLessThan(25)
    })

    test(`${pageId} ThemeTemplate maps to ${island}`, () => {
      const templateSrc = read('app/components/SFThemeTemplate.vue')
      expect(templateSrc).toContain(`'${componentId}': resolveComponent('${island}')`)
    })

    test(`${pageId} theme shells mark presentation ownership`, () => {
      for (const theme of ['sforum-default', 'sforum-nocturne']) {
        const tpl = read(`../../extensions/builtin/themes/${theme}/templates/${template}`)
        expect(tpl).toContain('data-theme-owned="presentation"')
        expect(tpl).toContain(`data-page="${pageId}"`)
      }
    })
  }

  test('auth credential forms remain HostPageIsland', () => {
    const template = read('app/components/SFThemeTemplate.vue')
    for (const id of [
      'identity.component.login_form',
      'identity.component.register_form',
      'identity.component.recovery_request_form',
      'identity.component.recovery_confirm_form',
    ]) {
      expect(template).toContain(`'${id}': HostPageIsland`)
    }
  })
})
