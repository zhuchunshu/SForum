import { describe, expect, it } from 'bun:test'
import { readFileSync } from 'node:fs'
import { join } from 'node:path'

const root = join(import.meta.dir, '..')

function read(rel: string) {
  return readFileSync(join(root, rel), 'utf8')
}

describe('SFPageOutlet catalog wiring', () => {
  const catalogPages: Array<[string, string]> = [
    ['app/pages/index.vue', 'forum.home'],
    ['app/pages/categories/index.vue', 'forum.category.index'],
    ['app/pages/c/[categorySlug].vue', 'forum.category.show'],
    ['app/pages/tags/index.vue', 'forum.tag.index'],
    ['app/pages/tags/[tagSlug].vue', 'forum.tag.show'],
    ['app/pages/t/[...path].vue', 'forum.topic.show'],
    ['app/pages/topics/new.vue', 'forum.topic.create'],
    ['app/pages/u/[username].vue', 'forum.profile.show'],
    ['app/pages/my/index.vue', 'forum.my.home'],
    ['app/pages/my/content-review.vue', 'forum.my.content_review'],
    ['app/pages/settings/profile.vue', 'forum.settings.profile'],
    ['app/pages/settings/security.vue', 'forum.settings.security'],
    ['app/pages/notifications.vue', 'forum.notifications'],
    ['app/pages/moderation/index.vue', 'moderation.review'],
    ['app/pages/login.vue', 'auth.login'],
    ['app/pages/register.vue', 'auth.register'],
    ['app/pages/forgot-password.vue', 'auth.forgot_password'],
    ['app/pages/reset-password.vue', 'auth.reset_password'],
    ['app/pages/terms.vue', 'site.terms'],
    ['app/pages/privacy.vue', 'site.privacy'],
    ['app/pages/guidelines.vue', 'site.guidelines']
  ]

  for (const [file, pageId] of catalogPages) {
    it(`${file} uses SFPageOutlet page="${pageId}"`, () => {
      const src = read(file)
      expect(src).toContain('SFPageOutlet')
      expect(src).toContain(`page="${pageId}"`)
    })
  }

  it('forces constrained pages to core provider', () => {
    const src = read('app/components/SFPageOutlet.vue')
    expect(src).toContain('CONSTRAINED_PAGES')
    expect(src).toContain('auth.login')
    expect(src).toContain('isConstrained')
  })

  it('disables L2 widget execution', () => {
    const src = read('app/components/SFExtensionWidget.vue')
    expect(src).toContain('data-l2-disabled')
    expect(src).not.toContain('import(/* @vite-ignore */')
    expect(src).not.toMatch(/await import\(/)
  })

  it('dynamic registry catch-all resolves path via API', () => {
    const src = read('app/pages/[...sfRegistryPage].vue')
    expect(src).toContain('/pages/resolve-path')
    expect(src).toContain('createError')
  })
})
