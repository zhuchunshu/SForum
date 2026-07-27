import { describe, expect, test } from 'bun:test'
import { readFileSync } from 'node:fs'
import { parse, compileScript } from '@vue/compiler-sfc'
import { ref } from 'vue'

describe('app startup rendering', () => {
  test('does not embed auth state during root app SSR startup', async () => {
    const page = loadAppComponentForStartupTest({ server: true })

    await page.component.setup({}, { expose: () => {} })

    expect(page.loaderStarted()).toBe(true)
    expect(page.webOptionsRefreshStarted()).toBe(true)
    expect(page.authRefreshStarted()).toBe(false)
    expect(page.themeSkinRefreshStarted()).toBe(true)
  })

  test('restores auth during SSR when the request carries a session', async () => {
    const page = loadAppComponentForStartupTest({
      server: true,
      cookie: 'locale=zh-CN; sforum_session=test-session'
    })

    await page.component.setup({}, { expose: () => {} })

    expect(page.loaderStarted()).toBe(true)
    expect(page.webOptionsRefreshStarted()).toBe(true)
    expect(page.authRefreshStarted()).toBe(true)
    expect(page.themeSkinRefreshStarted()).toBe(true)
  })

  test('does not block client setup while startup refresh is still pending', async () => {
    const page = loadAppComponentForStartupTest({ server: false })
    const setupPromise = page.component.setup({}, { expose: () => {} })

    const result = await Promise.race([
      setupPromise.then(() => 'resolved'),
      new Promise(resolve => setTimeout(() => resolve('pending'), 25))
    ])

    expect(page.loaderStarted()).toBe(false)
    expect(page.webOptionsRefreshStarted()).toBe(false)
    expect(page.authRefreshStarted()).toBe(false)
    expect(result).toBe('resolved')

    await page.runMounted()

    expect(page.webOptionsRefreshStarted()).toBe(true)
    expect(page.authRefreshStarted()).toBe(true)
    expect(page.themeSkinRefreshStarted()).toBe(true)
    expect(page.themeSkinCleared()).toBe(false)
  })

  test('keeps public theme skin outside admin routes', async () => {
    const page = loadAppComponentForStartupTest({ server: false, routePath: '/control-panel' })

    await page.component.setup({}, { expose: () => {} })
    await page.runMounted()

    expect(page.themeSkinRefreshStarted()).toBe(false)
    expect(page.themeSkinCleared()).toBe(true)
  })
})

function loadAppComponentForStartupTest(options: { server: boolean, routePath?: string, cookie?: string }) {
  const source = readFileSync(new URL('../../app/app.vue', import.meta.url), 'utf8')
  const { descriptor } = parse(source, { filename: 'app.vue' })
  const compiled = compileScript(descriptor, { id: 'app-startup-test' }).content
  const transpiler = new Bun.Transpiler({ loader: 'ts', target: 'browser' })
  const executable = transpiler.transformSync(
    compiled.replace(
      /import\s+\{\s*withAsyncContext\s+as\s+_withAsyncContext,\s*defineComponent\s+as\s+_defineComponent\s*\}\s+from\s+['"]vue['"];?/,
      'const { withAsyncContext: _withAsyncContext, defineComponent: _defineComponent } = __vue;'
    )
      .replace(/^import(?:[\s\S]*?from\s+)?['"][^'"]+['"];?\s*$/gm, '')
  )
    .replaceAll('import.meta.dev', 'true')
    .replaceAll('import.meta.server', options.server ? 'true' : 'false')
    .replace(/export default /, 'return ')

  let loaderStarted = false
  let webOptionsRefreshStarted = false
  let authRefreshStarted = false
  let themeSkinRefreshStarted = false
  let themeSkinCleared = false
  const mountedCallbacks: Array<() => void | Promise<void>> = []
  const never = new Promise(() => {})

  const factory = new Function(
    '__vue',
    'useLocaleHead',
    'useWebOptions',
    'useAuthSession',
    'useRoute',
    'useAdminRoutes',
    'useAdminTabs',
    'useAsyncData',
    'useHead',
    'applySEOTitleTemplate',
    'onMounted',
    'watch',
    'useActiveThemeSkin',
    'useRequestHeaders',
    'useExternalAuthFeedback',
    'SFApiConnectionModal',
    executable
  )

  const component = factory(
    {
      withAsyncContext: (fn: () => Promise<unknown>) => [fn(), () => {}],
      defineComponent: (options: unknown) => options
    },
    () => ({ value: { htmlAttrs: {}, link: [], meta: [] } }),
    () => ({
      siteName: ref('SForum'),
      resolvedAppearanceTheme: ref({ dataTheme: 'pine_teal', style: '' }),
      seoSettings: ref({ metaTitleTemplate: '' }),
      siteFaviconUrl: ref(''),
      siteAppleTouchIconUrl: ref(''),
      refresh: () => {
        webOptionsRefreshStarted = true
        return options.server ? Promise.resolve(true) : never
      }
    }),
    () => ({
      refresh: () => {
        authRefreshStarted = true
        return options.server ? Promise.resolve(true) : never
      }
    }),
    () => ({ path: options.routePath || '/' }),
    () => ({ routeId: (path: string) => path.startsWith('/control-panel') ? '/' : null }),
    () => ({ cachedTabNames: ref([]) }),
    (_key: string, loader: () => Promise<unknown>) => {
      loaderStarted = true
      return loader()
    },
    () => {},
    () => '',
    (callback: () => void | Promise<void>) => {
      mountedCallbacks.push(callback)
    },
    () => {},
    () => ({
      links: ref([]),
      restoreLastPublic: () => false,
      refresh: async () => {
        themeSkinRefreshStarted = true
      },
      clear: () => {
        themeSkinCleared = true
      }
    }),
    () => ({ cookie: options.cookie || '' }),
    () => ({
      consumeFromRoute: () => {}
    }),
    {}
  )

  return {
    component,
    loaderStarted: () => loaderStarted,
    webOptionsRefreshStarted: () => webOptionsRefreshStarted,
    authRefreshStarted: () => authRefreshStarted,
    themeSkinRefreshStarted: () => themeSkinRefreshStarted,
    themeSkinCleared: () => themeSkinCleared,
    runMounted: async () => {
      for (const callback of mountedCallbacks) {
        void callback()
      }
      await Promise.resolve()
    }
  }
}
