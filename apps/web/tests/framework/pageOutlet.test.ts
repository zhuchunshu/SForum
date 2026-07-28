import { describe, expect, it } from 'bun:test'
import { readFileSync } from 'node:fs'
import { join } from 'node:path'

const root = join(import.meta.dir, '../..')

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
    ['app/pages/settings/notifications.vue', 'forum.settings.notifications'],
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

  it('routes themeable system errors through the dedicated themed boundary', () => {
    const errorPage = read('app/error.vue')
    const systemError = read('app/components/errors/SFSystemErrorPage.vue')
    expect(errorPage).toContain('SFSystemErrorPage')
    expect(errorPage).toContain('const nuxtError = computed(() => props.error)')
    expect(errorPage).toContain("robots: 'noindex,nofollow'")
    expect(systemError).toContain('SFSystemThemeTemplate')
    expect(systemError).toContain(':data-page="page.pageId"')
    expect(systemError).toContain(':render-output="resolvedPage?.renderOutput"')
    expect(systemError).toContain('SFSystemErrorEmergencyPage')
    expect(errorPage).toContain('useSystemErrorPagePresentation(systemPageId)')
    expect(errorPage).toContain('isThemeableSystemErrorStatus')
    expect(errorPage).toContain('systemErrorPageIdForStatus')
    expect(errorPage).toContain('useActiveThemeSkin()')
    expect(errorPage).toContain("'data-sforum-theme-skin': '1'")
    expect(errorPage).toContain("'data-sforum-theme': resolvedAppearanceTheme.value.dataTheme")
    expect(errorPage).toContain("'data-sforum-error': String(systemPage.value.statusCode)")
    expect(read('app/assets/css/sforum-theme.css')).toContain('body[data-sforum-error="404"] > nuxt-error-overlay.pip-hidden')
    const errorPanel = read('app/components/errors/SFErrorPagePanel.vue')
    expect(errorPanel).toContain('function clearResolvedDevErrorOverlay()')
    expect(errorPanel).toContain("document.querySelector('nuxt-error-overlay.pip-hidden')?.remove()")
    expect(errorPanel).toContain('await clearError({ redirect: localePath(\'/\') })')
    expect(errorPanel.indexOf('await clearError({ redirect: localePath(\'/\') })'))
      .toBeLessThan(errorPanel.indexOf('clearResolvedDevErrorOverlay()', errorPanel.indexOf('async function goHome()')))
    expect(errorPage).toContain('const resolvedSystemPage = shallowRef(')
    expect(errorPage).not.toContain('await prepareNotFoundPage')
    expect(errorPage).not.toContain('onServerPrefetch')
    expect(errorPage).not.toContain('<script setup lang="ts" async>')

    const presentation = read('app/composables/errors/useSystemErrorPagePresentation.ts')
    expect(presentation).toContain('themeSkin.refresh({')
    expect(presentation).toContain('serverInternal: import.meta.server')
    expect(presentation).toContain('serverInternal: true')
    expect(presentation).toContain("if (resolved.provider === 'core' || resolved.fallback)")
    expect(presentation).toContain('exactThemeIdentityForPageResolve(resolved)')
    expect(presentation).toContain('enterCoreEmergency(')
    expect(presentation).toContain('Promise.all([')
    expect(presentation).not.toContain('Promise.allSettled')

    const systemResolve = read('app/composables/errors/useSystemErrorPageResolve.ts')
    expect(systemResolve).not.toContain('actorKey')
    expect(systemResolve).toContain('system-error-page-resolve:${resolvedPageId.value}:${String(locale.value || \'zh-CN\')}:${route.path}')
    expect(systemResolve).toContain('useState<PageResolvePayload | null>')
    expect(systemResolve).toContain('clearNuxtState(stateKey)')
    expect(systemResolve).not.toContain('return useAsyncData')
    expect(systemResolve).toContain('async function refresh(')
    expect(systemResolve).toContain('serverInternal: import.meta.server')
    expect(systemResolve).toContain('failure = shallowRef<unknown>(null)')
    expect(systemResolve).toContain('useCoreFallback')
    expect(systemResolve).toContain('deferCommit')
    expect(presentation).toContain('apply: false')
    expect(presentation).toContain('themeSkin.commit(skinResult)')
    expect(systemError).not.toContain('class="min-h-screen"')
    expect(systemResolve).toContain('timeout: import.meta.dev ? 800 : 1000, maxAttempts: 1')

    const serverPlugin = read('app/plugins/not-found-theme.server.ts')
    expect(serverPlugin).toContain("name: 'sforum-system-error-theme'")
    expect(serverPlugin).toContain('await useSystemErrorPagePresentation(pageId).prepare()')
    expect(read('app/components/SFPageOutletResolver.vue')).toContain('await notFoundPresentation.prepare()')
    expect(read('app/pages/[...sfRegistryPage].vue')).toContain('await notFoundPresentation.prepare()')

    const activeThemeSkin = read('app/composables/themes/useActiveThemeSkin.ts')
    expect(activeThemeSkin.indexOf('const { request } = useApiClient()'))
      .toBeLessThan(activeThemeSkin.indexOf('async function refresh('))
    expect(activeThemeSkin.slice(activeThemeSkin.indexOf('async function refresh(')))
      .not.toContain('const { request } = useApiClient()')

    const systemTemplate = read('app/components/SFSystemThemeTemplate.vue')
    const systemNode = read('app/components/SFSystemThemeNode.vue')
    expect(systemTemplate).toContain('renderThemeRenderNodes')
    expect(systemTemplate).toContain('const ThemeNodes = () =>')
    expect(systemTemplate).toContain('L0/L1')
    expect(systemTemplate).toContain('SFSystemErrorEmergencyPage')
    expect(systemTemplate).toContain('<SFSystemErrorEmergencyPage v-else')
    expect(systemTemplate).not.toContain('SFAlert')
    expect(systemTemplate).toContain("'system.component.error_details': SFSystemErrorDetails")
    expect(systemTemplate).toContain("'system.component.error_actions': SFSystemErrorActions")
    expect(systemTemplate).toContain("'system.component.error_recovery': SFSystemErrorRecovery")
    expect(systemTemplate).toContain("'system.component.error_sidebar': SFSystemErrorSidebar")
    expect(systemTemplate).toContain("'system.component.error_rail': SFSystemErrorRail")
    expect(systemNode).toContain('<SFNotFoundPageContent')
    expect(systemNode).toContain('<SFNavbar')
    expect(systemNode).toContain('<SFFooter')
    expect(systemNode).toContain('const self = getCurrentInstance()!.type')
    expect(systemNode).not.toContain('<SFSystemThemeNode')
    expect(systemNode).toContain("import SFNavbar from './SFNavbar.vue'")
    expect(systemNode).toContain("import SFNotFoundPageContent from './SFNotFoundPageContent.vue'")
    expect(systemNode).toContain("import SFFooter from './SFFooter.vue'")
    expect(read('app/components/SFNavbar.vue')).not.toContain('await useAsyncData')
    expect(read('app/components/SFNavbar.vue')).toContain('fetchRemoteChrome')
    expect(read('app/components/SFFooter.vue')).not.toContain("await useAsyncData('site-public-friend-links'")
    expect(read('app/components/SFFooter.vue')).toContain('fetchRemoteChrome')
    expect(read('app/components/errors/SFSystemErrorEmergencyPage.vue')).toContain('emergency')
    const errorContent = read('app/components/errors/SFErrorPageContent.vue')
    expect(errorContent).toContain(':fetch-remote-chrome="!emergency"')
    expect(errorContent).toContain('!props.emergency && normalizeErrorStatus')
    const recoveryBody = read('app/components/errors/SFSystemErrorRecovery.vue')
    expect(recoveryBody).toContain('method="get"')
    expect(recoveryBody).toContain(':action="searchAction"')
    expect(recoveryBody).toContain('name="q"')
    expect(recoveryBody).not.toContain('@submit.prevent')
    const recoveryData = read('app/composables/errors/useSystemErrorRecoveryData.ts')
    expect(recoveryData).toContain('RECOVERY_DATA_TIMEOUT_MS')
    expect(recoveryData).toContain('serverInternal: import.meta.server')
    expect(recoveryData).toContain('forumApi.listCategoryGroups(readOptions)')
    expect(recoveryData).toContain('forumApi.listTags(readOptions)')
    expect(recoveryData).toContain('forumApi.listTopics({ page: 1 }, readOptions)')
    expect(read('server/plugins/not-found-document-policy.ts')).toContain("headers['cache-control'] = 'no-store'")
    expect(read('server/plugins/not-found-document-policy.ts')).toContain("headers['x-robots-tag'] = 'noindex,nofollow'")
    expect(read('server/plugins/not-found-document-policy.ts')).toContain("path.startsWith('/api/')")
    expect(read('server/plugins/not-found-document-policy.ts')).toContain("contentType.includes('application/json')")
    expect(systemError).not.toContain('setResponseHeader(')
    const systemDetails = read('app/components/errors/SFSystemErrorDetails.vue')
    expect(systemDetails).toContain('sforum-system-error__status-code')
    expect(systemDetails).toContain("t('errors.page.statusCaption')")
    expect(read('i18n/locales/zh-CN.json')).toContain('"statusCaption": "HTTP STATUS"')

    const adminPages = read('app/pages/admin/extensions/pages.vue')
    expect(adminPages).toContain('virtual?: boolean')
    expect(adminPages).toContain('function pathLabel(row: PageRow)')
    expect(adminPages).toContain('admin.extensions.pages.virtual.${row.page.id.replaceAll')
    expect(adminPages).toContain('i-lucide-file-warning')
    expect(read('i18n/locales/zh-CN.json')).toContain('"system_rate_limited": "429 访问过频错误页"')
    expect(read('i18n/locales/en-US.json')).toContain('"system_server_error": "5xx service error page"')
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
    expect(template).toContain("'identity.component.login_form': defineAsyncComponent(() => import('./identity/SFLoginFormPage.vue'))")
    expect(template).toContain("'forum.component.topic_composer': defineAsyncComponent(() => import('./forum/SFTopicComposerPage.vue'))")
    expect(template).toContain("'forum.component.topic_reply': defineAsyncComponent(() => import('./forum/SFTopicReplyPage.vue'))")
    expect(template).toContain("'forum.component.topic_editor': defineAsyncComponent(() => import('./forum/SFTopicEditPage.vue'))")
    expect(template).toContain("'forum.component.home_page': defineAsyncComponent(() => import('./forum/SFHomePage.vue'))")
    expect(template).toContain("'system.component.not_found': SFNotFoundPageContent")
    expect(read('app/components/SFSystemThemeTemplate.vue')).toContain("'system.component.error_details': SFSystemErrorDetails")
    expect(template).not.toContain('resolveComponent(')
    expect(template).toContain('slots.default?.()')
    // fail-closed 公开页走宿主 chrome；主题成功路径不套 Host chrome。
    expect(outlet).toContain('SFHostPublicChrome')
    expect(outlet).toContain('useHostPublicChrome')
    const hostChrome = read('app/components/SFHostPublicChrome.vue')
    expect(hostChrome).toContain('sf-host-public-chrome--fullwidth-3col')
    expect(hostChrome).toContain('data-layout="fullwidth-3col"')
    expect(hostChrome).toContain('sf-host-public-chrome__body')
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

  it('keeps non-themeable failures outside the replaceable system-error surface', () => {
    const src = read('app/error.vue')
    expect(src).toContain('isThemeableSystemErrorStatus(nuxtError.value?.statusCode)')
    expect(src).toContain('<UApp v-else>')
    expect(src).toContain('<SFErrorPageContent :error="nuxtError" />')
    expect(src.indexOf('<SFSystemErrorPage')).toBeLessThan(src.indexOf('<UApp v-else>'))
  })

  it('lets the active theme present system error pages while Host keeps the error islands', () => {
    const outlet = [
      read('app/components/SFPageOutlet.vue'),
      read('app/components/SFPageOutletRender.vue')
    ].join('\n')
    const errorPage = read('app/error.vue')
    const catalog = read('../../apps/api/app/Support/Pages/catalog.go')

    expect(outlet).not.toContain('forceDefaultTheme')
    expect(errorPage).not.toContain('forceDefaultTheme')
    expect(errorPage).toContain('SFSystemErrorPage')
    expect(catalog).toMatch(/ID: "system\.forbidden"[^\n]+Replaceable: true[^\n]+Virtual: true/)
    expect(catalog).toMatch(/ID: "system\.not_found"[^\n]+Replaceable: true/)
    expect(catalog).toMatch(/ID: "system\.rate_limited"[^\n]+Replaceable: true[^\n]+Virtual: true/)
    expect(catalog).toMatch(/ID: "system\.server_error"[^\n]+Replaceable: true[^\n]+Virtual: true/)
  })
})
