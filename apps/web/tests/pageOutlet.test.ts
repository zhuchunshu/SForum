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
    ['app/pages/guidelines.vue', 'site.guidelines'],
    ['app/error.vue', 'system.not_found'],
    ['app/pages/components.vue', 'dev.components']
  ]

  for (const [file, pageId] of catalogPages) {
    it(`${file} uses SFPageOutlet page="${pageId}"`, () => {
      const src = read(file)
      expect(src).toContain('SFPageOutlet')
      expect(src).toContain(`page="${pageId}"`)
    })
  }

  it('resolves protected pages and supplies the core page as a Host island slot', () => {
    const outlet = read('app/components/SFPageOutlet.vue')
    const template = read('app/components/SFThemeTemplate.vue')
    expect(outlet).not.toContain('CONSTRAINED_PAGES')
    expect(outlet).not.toContain('isConstrained')
    expect(outlet).toContain('<slot />')
    // Auth recovery forms remain HostPageIsland (credential isolation).
    expect(template).toContain("'identity.component.login_form': HostPageIsland")
    expect(template).toContain("'identity.component.register_form': HostPageIsland")
    // Content body islands are self-contained components.
    expect(template).toContain("'forum.component.topic_composer': resolveComponent('SFTopicComposerPage')")
    expect(template).toContain("'forum.component.home_page': resolveComponent('SFHomePage')")
    expect(template).toContain('slots.default?.()')
  })

  it('binds core theme resolution to the current path and query', () => {
    const src = read('app/components/SFPageOutlet.vue')
    expect(src).toContain("path: route.path")
    expect(src).toContain("query.set('query', requestQuery.value)")
    expect(src).toContain('route.path}?${requestQuery.value}')
  })

  it('loads only Host-issued exact-artifact L2 descriptors', () => {
    const src = read('app/components/SFExtensionWidget.vue')
    expect(src).toContain('parsePublicFrontendDescriptor')
    expect(src).toContain('publicComponentPath')
    expect(src).toContain('data-l2-fallback')
    expect(src).not.toContain('entry?: string')
    expect(src).not.toContain('integrity?: string')
  })

  it('dynamic registry catch-all resolves path via API', () => {
    const src = read('app/pages/[...sfRegistryPage].vue')
    expect(src).toContain('/pages/resolve-path')
    expect(src).toContain('createError')
  })

  it('renders Host emergency output for dynamic extension pages', () => {
    for (const file of ['app/pages/[...sfRegistryPage].vue', 'app/pages/x/[...path].vue']) {
      const src = read(file)
      expect(src).toContain('Boolean(data.value?.renderOutput || templateHtml.value)')
      expect(src).not.toContain('&& !data.value?.fallback')
    }
  })

  it('keeps non-404 failures outside the replaceable not-found surface', () => {
    const src = read('app/error.vue')
    expect(src).toContain('v-if="isNotFound"')
    expect(src).toContain('<SFErrorPageContent v-else')
  })
})
