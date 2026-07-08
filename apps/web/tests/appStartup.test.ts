import { describe, expect, test } from 'bun:test'
import { readFileSync } from 'node:fs'
import { parse, compileScript } from '@vue/compiler-sfc'
import { ref } from 'vue'

describe('app startup rendering', () => {
  test('does not block client setup while startup refresh is still pending', async () => {
    const page = loadAppComponentForStartupTest()
    const setupPromise = page.component.setup({}, { expose: () => {} })

    const result = await Promise.race([
      setupPromise.then(() => 'resolved'),
      new Promise(resolve => setTimeout(() => resolve('pending'), 25))
    ])

    expect(page.loaderStarted()).toBe(true)
    expect(page.webOptionsRefreshStarted()).toBe(true)
    expect(page.authRefreshStarted()).toBe(true)
    expect(result).toBe('resolved')
  })
})

function loadAppComponentForStartupTest() {
  const source = readFileSync(new URL('../app/app.vue', import.meta.url), 'utf8')
  const { descriptor } = parse(source, { filename: 'app.vue' })
  const compiled = compileScript(descriptor, { id: 'app-startup-test' }).content
  const transpiler = new Bun.Transpiler({ loader: 'ts', target: 'browser' })
  const executable = transpiler.transformSync(compiled)
    .replace(
      /import\s+\{\s*withAsyncContext\s+as\s+_withAsyncContext,\s*defineComponent\s+as\s+_defineComponent\s*\}\s+from\s+['"]vue['"];?/,
      'const { withAsyncContext: _withAsyncContext, defineComponent: _defineComponent } = __vue;'
    )
    .replace(/import\s+\{\s*useAdminTabs\s*\}\s+from\s+['"]~\/composables\/useAdminTabs['"];?\n*/, '')
    .replaceAll('import.meta.dev', 'true')
    .replaceAll('import.meta.server', 'false')
    .replace(/export default /, 'return ')

  let loaderStarted = false
  let webOptionsRefreshStarted = false
  let authRefreshStarted = false
  const never = new Promise(() => {})

  const factory = new Function(
    '__vue',
    'useLocaleHead',
    'useWebOptions',
    'useAuthSession',
    'useAdminTabs',
    'useAsyncData',
    'useHead',
    'applySEOTitleTemplate',
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
      refresh: () => {
        webOptionsRefreshStarted = true
        return never
      }
    }),
    () => ({
      refresh: () => {
        authRefreshStarted = true
        return never
      }
    }),
    () => ({ cachedTabNames: ref([]) }),
    (_key: string, loader: () => Promise<unknown>) => {
      loaderStarted = true
      return loader()
    },
    () => {},
    () => ''
  )

  return {
    component,
    loaderStarted: () => loaderStarted,
    webOptionsRefreshStarted: () => webOptionsRefreshStarted,
    authRefreshStarted: () => authRefreshStarted
  }
}
