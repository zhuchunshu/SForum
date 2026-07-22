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
    ['app/pages/topics/reply.vue', 'forum.topic.reply'],
    ['app/pages/u/[username].vue', 'forum.profile.show'],
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
    ['app/pages/components.vue', 'dev.components']
  ]

  for (const [file, pageId] of catalogPages) {
    it(`${file} uses SFPageOutlet page="${pageId}"`, () => {
      const src = read(file)
      expect(src).toContain('SFPageOutlet')
      expect(src).toContain(`page="${pageId}"`)
    })
  }

  it('routes 404 errors through the dedicated themed not-found boundary', () => {
    const errorPage = read('app/error.vue')
    const notFound = read('app/components/SFNotFoundPage.vue')
    expect(errorPage).toContain('SFNotFoundPage')
    expect(errorPage).toContain('const nuxtError = computed(() => props.error)')
    expect(errorPage).toContain("robots: 'noindex,nofollow'")
    expect(notFound).toContain('SFPageOutlet')
    expect(notFound).toContain('page="system.not_found"')
    expect(notFound).toContain(':resolved-payload="resolvedPage"')
    expect(notFound).toContain('NotFoundEmergencyPage')
    expect(errorPage).toContain('useNotFoundPageResolve()')
    expect(errorPage).toContain('useActiveThemeSkin()')
    expect(errorPage).toContain('themeSkin.refresh()')
    expect(errorPage).toContain("'data-sforum-theme-skin': '1'")
    expect(errorPage).toContain("'data-sforum-theme': resolvedAppearanceTheme.value.dataTheme")
    expect(errorPage).toContain('refreshWebOptions({ timeout: startupTimeout })')
    expect(errorPage).toContain('refreshAuthSession({ timeout: startupTimeout })')
    expect(errorPage).toContain("if (resolved.provider === 'core' || resolved.fallback)")
    expect(errorPage).toContain('Promise.allSettled')
    expect(errorPage).toContain('onServerPrefetch(() => notFoundStartup)')
    expect(errorPage).not.toContain('<script setup lang="ts" async>')
    const notFoundResolve = read('app/composables/useNotFoundPageResolve.ts')
    expect(notFoundResolve).not.toContain('actorKey')
    expect(notFoundResolve).toContain('not-found-page-resolve:${String(locale.value || \'zh-CN\')}:${route.path}')
    expect(notFoundResolve).toContain('useState<PageResolvePayload | null>')
    expect(notFoundResolve).toContain('clearNuxtState(stateKey)')
    expect(notFoundResolve).not.toContain('return useAsyncData')
    expect(notFoundResolve).toContain('async function refresh()')
    expect(notFoundResolve).toContain('process.env.NUXT_API_INTERNAL_BASE_URL')
    expect(notFoundResolve).toContain('headers: apiHeaders()')
    expect(notFoundResolve).toContain('timeout: import.meta.dev ? 800 : 1000, maxAttempts: 1')
    const systemTemplate = read('app/components/SFSystemThemeTemplate.vue')
    const systemNode = read('app/components/SFSystemThemeNode.vue')
    expect(systemTemplate).toContain('SFSystemThemeNode')
    expect(systemTemplate).toContain('L0/L1')
    expect(systemTemplate).not.toContain('renderThemeRenderNodes')
    expect(systemNode).toContain('<SFNotFoundPageContent')
    expect(systemNode).toContain('<SFNavbar')
    expect(systemNode).toContain('<SFFooter')
    expect(read('app/components/SFNavbar.vue')).not.toContain('await useAsyncData')
    expect(read('app/components/SFNavbar.vue')).toContain('fetchRemoteChrome')
    expect(read('app/components/SFFooter.vue')).not.toContain("await useAsyncData('site-public-friend-links'")
    expect(read('app/components/SFFooter.vue')).toContain('fetchRemoteChrome')
    expect(read('app/components/SFNotFoundEmergencyPage.vue')).toContain('emergency')
    expect(read('app/components/SFErrorPageContent.vue')).toContain(':fetch-remote-chrome="!emergency"')
    const notFoundBody = read('app/components/SFNotFoundPageContent.vue')
    expect(notFoundBody).toContain('sforum-not-found-page__layout sforum-home__layout')
    expect(notFoundBody).toContain('sforum-not-found-page__sidebar sforum-home__sidebar')
    expect(notFoundBody).toContain('can(FORUM_PERMISSIONS.topicCreate)')
    expect(notFoundBody).toContain(':can-create-topic="canCreateTopic"')
    expect(read('server/plugins/not-found-document-policy.ts')).toContain("headers['cache-control'] = 'no-store'")
    expect(read('server/plugins/not-found-document-policy.ts')).toContain("headers['x-robots-tag'] = 'noindex,nofollow'")
    expect(read('server/plugins/not-found-document-policy.ts')).toContain("path.startsWith('/api/')")
    expect(read('server/plugins/not-found-document-policy.ts')).toContain("contentType.includes('application/json')")
  })

  it('resolves protected pages and supplies the core page as a Host island slot', () => {
    const outlet = [
      read('app/components/SFPageOutlet.vue'),
      read('app/components/SFPageOutletResolver.vue'),
      read('app/components/SFPageOutletRender.vue')
    ].join('\n')
    const template = read('app/components/SFThemeTemplate.vue')
    expect(outlet).not.toContain('CONSTRAINED_PAGES')
    expect(outlet).not.toContain('isConstrained')
    expect(outlet).toContain('<slot />')
    // Auth forms are Host body islands (Host code, not theme-executable).
    expect(template).toContain("'identity.component.login_form': resolveComponent('LazySFLoginFormPage')")
    expect(template).toContain("'forum.component.topic_composer': resolveComponent('LazySFTopicComposerPage')")
    expect(template).toContain("'forum.component.topic_reply': resolveComponent('LazySFTopicReplyPage')")
    expect(template).toContain("'forum.component.home_page': resolveComponent('LazySFHomePage')")
    expect(template).toContain("'system.component.not_found': SFNotFoundPageContent")
    expect(template).toContain('slots.default?.()')
    // fail-closed 公开页走宿主 chrome；主题成功路径不套 Host chrome。
    expect(outlet).toContain('SFHostPublicChrome')
    expect(outlet).toContain('useHostPublicChrome')
  })

  it('binds core theme resolution to the current path and query', () => {
    const src = read('app/components/SFPageOutletResolver.vue')
    expect(src).toContain("path: route.path")
    expect(src).toContain("query.set('query', requestQuery.value)")
    expect(src).toContain('resolveLocale.value')
    expect(src).toContain('resolveActorKey.value')
    expect(src).toContain('${route.path}?${requestQuery.value}')
  })

  it('does not cache rendered Page Registry output across paths or actors', () => {
    const src = read('app/components/SFPageOutletResolver.vue')
    const util = read('app/utils/pageResolve.ts')
    expect(src).not.toContain('usePageResolveShellCache')
    expect(src).not.toContain('rememberShell')
    expect(src).not.toContain('recallShell')
    expect(src).not.toContain('data-stale')
    expect(src).not.toContain('default: () =>')
    expect(src).toContain('requestPageResolveWithRetry')
    expect(src).toContain('PAGE_RESOLVE_TIMEOUT_MS')
    expect(src).toContain('PAGE_RESOLVE_REASON.transportUnavailable')
    expect(src).toContain('disableSharedPageCacheForPageResolve')
    expect(src).toContain('disableSharedDocumentCache()')
    expect(src).toContain('requestEvent?.context')
    expect(util).toContain('routeRules.cache = false')
    expect(util).toContain('routeRules.swr = false')
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
    expect(src).toContain('Number(nuxtError.value?.statusCode) === 404')
    expect(src).toContain('<UApp v-else>')
    expect(src).toContain('<SFErrorPageContent :error="nuxtError" />')
    expect(src.indexOf('<SFNotFoundPage')).toBeLessThan(src.indexOf('<UApp v-else>'))
  })

  it('lets the active theme present 404 pages while Host keeps the error island', () => {
    const outlet = [
      read('app/components/SFPageOutlet.vue'),
      read('app/components/SFPageOutletRender.vue')
    ].join('\n')
    const errorPage = read('app/error.vue')
    const catalog = read('../../apps/api/app/Support/Pages/catalog.go')

    expect(outlet).not.toContain('forceDefaultTheme')
    expect(errorPage).not.toContain('forceDefaultTheme')
    expect(errorPage).toContain('SFNotFoundPage')
    expect(catalog).toMatch(/ID: "system\.not_found"[^\n]+Replaceable: true/)
  })
})
