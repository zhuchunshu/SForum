import { describe, expect, test } from 'bun:test'
import { readFileSync } from 'node:fs'
import { join } from 'node:path'

const root = join(import.meta.dir, '../..')
const read = (rel: string) => readFileSync(join(root, rel), 'utf8')

describe('remaining public presentation ownership', () => {
  const surfaces = [
    ['app/pages/t/[...path].vue', 'forum.topic.show', 'SFTopicShowPage', 'forum.component.topic_show', 'topic-show.html'],
    ['app/pages/topics/new.vue', 'forum.topic.create', 'SFTopicComposerPage', 'forum.component.topic_composer', 'topic-create.html'],
    ['app/pages/topics/reply.vue', 'forum.topic.reply', 'SFTopicReplyPage', 'forum.component.topic_reply', 'topic-reply.html'],
    ['app/pages/topics/[topicId]/edit.vue', 'forum.topic.edit', 'SFTopicEditPage', 'forum.component.topic_editor', 'topic-edit.html'],
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
      const componentDomain = pageId.startsWith('auth.')
        ? 'identity'
        : pageId.startsWith('forum.settings.')
          ? 'settings'
          : pageId === 'forum.profile.show'
            ? 'profile'
            : pageId === 'forum.notifications'
              ? 'notifications'
              : 'forum'
      const expectedComponent = pageId === 'forum.topic.show'
        ? `'${componentId}': ${island}`
        : `'${componentId}': defineAsyncComponent(() => import('./${componentDomain}/${island}.vue'))`
      expect(templateSrc).toContain(expectedComponent)
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
    expect(template).toContain("'identity.component.login_form': defineAsyncComponent(() => import('./identity/SFLoginFormPage.vue'))")
    expect(template).toContain("'identity.component.register_form': defineAsyncComponent(() => import('./identity/SFRegisterFormPage.vue'))")
    expect(template).toContain("'identity.component.recovery_request_form': defineAsyncComponent(() => import('./identity/SFRecoveryRequestPage.vue'))")
    expect(template).toContain("'identity.component.recovery_confirm_form': defineAsyncComponent(() => import('./identity/SFRecoveryConfirmPage.vue'))")
    expect(template).toContain("'system.component.not_found': SFNotFoundPageContent")
  })

  test('system error theme shells mark presentation ownership', () => {
    const pages = [
      ['forbidden.html', 'system.forbidden'],
      ['not-found.html', 'system.not_found'],
      ['rate-limited.html', 'system.rate_limited'],
      ['server-error.html', 'system.server_error']
    ] as const
    for (const theme of ['sforum-default', 'sforum-nocturne']) {
      for (const [template, pageId] of pages) {
        const tpl = read(`../../extensions/builtin/themes/${theme}/templates/${template}`)
        expect(tpl).toContain('data-theme-owned="presentation"')
        expect(tpl).toContain(`data-page="${pageId}"`)
        expect(tpl).toContain('<sf-error-details')
        expect(tpl).toContain('<sf-error-actions')
        expect(tpl).toContain('<sf-error-recovery')
        expect(tpl).toContain('<sf-error-sidebar')
        expect(tpl).toContain('<sf-error-rail')
        expect(tpl).toContain('<sf-navbar')
        expect(tpl).toContain('<sf-footer')
      }
    }
    const systemTemplate = read('app/components/SFSystemThemeTemplate.vue')
    expect(systemTemplate).toContain("'system.component.error_details': SFSystemErrorDetails")
    expect(systemTemplate).toContain("'system.component.error_actions': SFSystemErrorActions")
    expect(systemTemplate).toContain("'system.component.error_recovery': SFSystemErrorRecovery")
    expect(systemTemplate).toContain("'system.component.error_sidebar': SFSystemErrorSidebar")
    expect(systemTemplate).toContain("'system.component.error_rail': SFSystemErrorRail")
    const nocturneCSS = read('../../extensions/builtin/themes/sforum-nocturne/assets/theme.css')
    const errorBody = read('app/components/errors/SFSystemErrorPage.vue')
    expect(errorBody).toContain(':data-page="page.pageId"')
    expect(nocturneCSS).toContain('.nh-system-error-shell')
    expect(nocturneCSS).toContain('.nh-system-error-shell .sforum-system-error__layout')
    expect(nocturneCSS).toContain('.nh-system-error-shell .sforum-system-error__details')
    expect(nocturneCSS).toContain('.nh-system-error-shell .sforum-system-error__search-box')
    expect(nocturneCSS).toContain('@media (max-width: 960px)')
    expect(nocturneCSS).not.toContain('.nh-not-found-shell .sforum-not-found-page__layout')
    expect(nocturneCSS).not.toContain('.nh-not-found-shell .sforum-not-found-page__sidebar')
  })

  test('legacy system.not_found compatibility remains closed to L2', () => {
    for (const theme of ['sforum-default', 'sforum-nocturne']) {
      const tpl = read(`../../extensions/builtin/themes/${theme}/templates/not-found.html`)
      expect(tpl).toContain('data-theme-owned="presentation"')
      expect(tpl).toContain('data-page="system.not_found"')
      expect(tpl).toContain('<sf-error-details')
      expect(tpl).toContain('<sf-navbar')
      expect(tpl).toContain('<sf-footer')
    }
    const notFoundBody = read('app/components/errors/SFNotFoundPageContent.vue')
    expect(notFoundBody).toContain('sforum-not-found-page__layout sforum-home__layout')
    expect(notFoundBody).toContain('sforum-not-found-page__sidebar sforum-home__sidebar')
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
