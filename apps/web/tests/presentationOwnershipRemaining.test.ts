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
    ['app/pages/notifications.vue', 'forum.notifications', 'SFNotificationsPage', 'forum.component.notifications', 'notifications.html'],
    ['app/pages/settings/profile.vue', 'forum.settings.profile', 'SFProfileSettingsPage', 'profile.component.settings_form', 'settings-profile.html'],
    ['app/pages/settings/security.vue', 'forum.settings.security', 'SFSecuritySettingsPage', 'identity.component.security_settings', 'settings-security.html'],
    ['app/pages/login.vue', 'auth.login', 'SFLoginFormPage', 'identity.component.login_form', 'login.html'],
    ['app/pages/register.vue', 'auth.register', 'SFRegisterFormPage', 'identity.component.register_form', 'register.html'],
    ['app/pages/forgot-password.vue', 'auth.forgot_password', 'SFRecoveryRequestPage', 'identity.component.recovery_request_form', 'forgot-password.html'],
    ['app/pages/reset-password.vue', 'auth.reset_password', 'SFRecoveryConfirmPage', 'identity.component.recovery_confirm_form', 'reset-password.html'],
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
      expect(templateSrc).toContain(`'${componentId}': resolveComponent('Lazy${island}')`)
    })

    test(`${pageId} theme shells mark presentation ownership`, () => {
      for (const theme of ['sforum-default', 'sforum-nocturne']) {
        const tpl = read(`../../extensions/builtin/themes/${theme}/templates/${template}`)
        expect(tpl).toContain('data-theme-owned="presentation"')
        expect(tpl).toContain(`data-page="${pageId}"`)
        // auth 使用 auth layout；非 auth 公开页必须挂导航/页脚岛。
        if (!pageId.startsWith('auth.')) {
          expect(tpl).toContain('<sf-navbar')
          expect(tpl).toContain('<sf-footer')
        }
      }
    })
  }

  test('auth credential forms are Host body islands (not theme-executable)', () => {
    const template = read('app/components/SFThemeTemplate.vue')
    expect(template).toContain("'identity.component.login_form': resolveComponent('LazySFLoginFormPage')")
    expect(template).toContain("'identity.component.register_form': resolveComponent('LazySFRegisterFormPage')")
    expect(template).toContain("'identity.component.recovery_request_form': resolveComponent('LazySFRecoveryRequestPage')")
    expect(template).toContain("'identity.component.recovery_confirm_form': resolveComponent('LazySFRecoveryConfirmPage')")
    // system.not_found 仍走 HostPageIsland：error.vue 需把 NuxtError 注入 SFErrorPageContent。
    expect(template).toContain("'system.component.not_found': HostPageIsland")
  })

  test('system.not_found theme shells mark presentation ownership', () => {
    for (const theme of ['sforum-default', 'sforum-nocturne']) {
      const tpl = read(`../../extensions/builtin/themes/${theme}/templates/not-found.html`)
      expect(tpl).toContain('data-theme-owned="presentation"')
      expect(tpl).toContain('data-page="system.not_found"')
      expect(tpl).toContain('<sf-not-found-page')
    }
  })

  test('moderation.review stays Host-owned (non-replaceable workbench)', () => {
    const route = read('app/pages/moderation/index.vue')
    expect(route).toContain('SFPageOutlet')
    expect(route).toContain('page="moderation.review"')
    expect(route).toContain('<SFModerationReviewPage')
    // 不可主题替换：theme.json 不应声明 moderation.review replace。
    for (const theme of ['sforum-default', 'sforum-nocturne']) {
      const themeJson = read(`../../extensions/builtin/themes/${theme}/theme.json`)
      expect(themeJson).not.toContain('"target": "moderation.review"')
    }
  })
})
