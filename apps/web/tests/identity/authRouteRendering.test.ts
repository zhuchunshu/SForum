import { describe, expect, test } from 'bun:test'
import { Window } from 'happy-dom'
import { fileURLToPath } from 'node:url'
import { join } from 'node:path'
import { compileScript, compileTemplate, parse, rewriteDefault } from '@vue/compiler-sfc'

import { buildAuthPageLink, resolveAuthReturnPath } from '../../app/utils/identity/authReturn'
import {
  passwordPolicyProgress,
  passwordPolicyProgressLevel,
  passwordPolicyRequirements,
  type PasswordPolicy
} from '../../app/composables/useWebOptions'

const zhCN = JSON.parse(await Bun.file(new URL('../../i18n/locales/zh-CN.json', import.meta.url)).text())
const enUS = JSON.parse(await Bun.file(new URL('../../i18n/locales/en-US.json', import.meta.url)).text())
const authShellSource = await Bun.file(new URL('../../app/components/identity/auth/SFAuthShell.vue', import.meta.url)).text()

const mountWindow = new Window({ url: 'http://localhost/login' })
Object.assign(globalThis, {
  window: mountWindow, document: mountWindow.document, navigator: mountWindow.navigator,
  Element: mountWindow.Element, HTMLElement: mountWindow.HTMLElement, SVGElement: mountWindow.SVGElement,
  Node: mountWindow.Node, Event: mountWindow.Event, MouseEvent: mountWindow.MouseEvent
})
const mountRoot = fileURLToPath(new URL('../..', import.meta.url))
const mountVue = await import('vue')
const { mount, flushPromises } = await import('@vue/test-utils')

function vueImportToConst(names: string) {
  return `const {${names.replace(/(\w+) as (\w+)/g, '$1: $2')}} = Vue`
}

async function compileMountedComponent(
  relativePath: string,
  bindings: Record<string, unknown> = {}
) {
  const path = join(mountRoot, relativePath)
  const parsed = parse(await Bun.file(path).text(), { filename: path })
  const script = compileScript(parsed.descriptor, { id: `mount-${relativePath}` })
  const template = compileTemplate({
    source: parsed.descriptor.template?.content || '', filename: path, id: `mount-${relativePath}`,
    compilerOptions: { bindingMetadata: script.bindings }
  })
  if (template.errors.length) throw new Error(String(template.errors[0]))
  const scriptCode = `const { withAsyncContext: _withAsyncContext, defineComponent: _defineComponent } = Vue\n${script.content.replace(/^import[^\n]*\n/gm, '')}`
  const templateCode = template.code
    .replace(/import \{([^}]+)\} from \"vue\"/, (_match, names) => vueImportToConst(names))
    .replace('export function render', 'function render')
  const source = `${rewriteDefault(scriptCode, '__sfc__', ['typescript'])}\n${templateCode}\n__sfc__.render = render\nreturn __sfc__`
  const executable = new Bun.Transpiler({ loader: 'ts', target: 'esnext' }).transformSync(source)
  const names = Object.keys(bindings)
  return new Function('Vue', ...names, executable)(mountVue, ...Object.values(bindings))
}

const SFAuthProviderButtons = await compileMountedComponent('app/components/identity/SFAuthProviderButtons.vue')
const SFAuthShell = mountVue.defineComponent({
  template: '<main data-testid="auth-shell"><slot /></main>'
})
const SFLoginFormPage = await compileMountedComponent(
  'app/components/identity/SFLoginFormPage.vue',
  { SFAuthProviderButtons, SFAuthShell }
)
const SFRegisterFormPage = await compileMountedComponent(
  'app/components/identity/SFRegisterFormPage.vue',
  { SFAuthProviderButtons, SFAuthShell }
)

describe('auth route support contracts', () => {
  test('shares the persisted language menu through the authentication shell', () => {
    expect(authShellSource).toContain("from '~/composables/navigation/useNavbarLanguageMenu'")
    expect(authShellSource).toContain('const { currentLocaleName, languageMenuItems } = useNavbarLanguageMenu()')
    expect(authShellSource).toContain('<UDropdownMenu :items="languageMenuItems"')
    expect(authShellSource).toContain(":aria-label=\"t('nav.language')\"")
    expect(authShellSource).toContain(':title="currentLocaleName"')
    expect(authShellSource).toContain('i-tabler-language')
    expect(authShellSource).toContain('data-ssr-fallback="auth-language"')
    expect(authShellSource).toContain('data-ssr-fallback="auth-appearance"')
    expect(authShellSource).toContain('i-tabler-brightness-filled')
    expect(authShellSource).not.toContain('sf-auth-shell__utility-placeholder')
    expect(authShellSource.match(/aria-hidden="true"[\s\S]*?tabindex="-1"[\s\S]*?data-ssr-fallback="auth-/g)?.length).toBe(2)
    expect(authShellSource).toContain('[data-ssr-fallback] { pointer-events: none; }')
  })

  test('preserves local return targets and rejects auth-page or external redirects', () => {
    expect(buildAuthPageLink('/register', '/topics/123?page=2')).toEqual({
      path: '/register', query: { redirect: '/topics/123?page=2' }
    })
    expect(buildAuthPageLink('/login', '/register')).toBe('/login')
    expect(resolveAuthReturnPath('/login', '/settings/security', '/')).toBe('/settings/security')
    expect(resolveAuthReturnPath('https://evil.example', '', '/')).toBe('/')
  })

  test('evaluates the shared password policy without rendering a parallel form', () => {
    const policy: PasswordPolicy = {
      minLength: 8, maxLength: 64,
      requireLowercase: true, requireUppercase: true, requireNumber: true, requireSymbol: true
    }
    expect(passwordPolicyProgressLevel(passwordPolicyProgress('short', policy))).toBe('weak')
    expect(passwordPolicyProgress('Str0ng-pass!', policy)).toBe(100)
    expect(passwordPolicyRequirements('Str0ng-pass!', policy).every(item => item.met)).toBe(true)
    expect(zhCN.auth.passwordRequirementSymbol).toBeTruthy()
    expect(enUS.auth.passwordRequirementSymbol).toBeTruthy()
  })

  test('mounts the production login form with password fallback and generic provider start', async () => {
    const starts: Array<[string, string]> = []
    const locale = mountVue.ref('en-US')
    const providers = mountVue.ref([{
      id: 'example.provider.auth', kind: 'auth', contractVersion: 'example.auth@1', priority: 1,
      operations: ['login.start'], activatedOperations: ['login'], label: 'Example Login', icon: 'i-lucide-key-round'
    }])
    Object.assign(globalThis, {
      computed: mountVue.computed, ref: mountVue.ref, reactive: mountVue.reactive, watch: mountVue.watch,
      useI18n: () => ({ locale, t: (key: string, values?: Record<string, string>) => key === 'auth.providers.continueWith' ? `Continue with ${values?.name}` : key }),
      useToast: () => ({ add: () => {} }), useLocalePath: () => (path: string) => path,
      useApiClient: () => ({ apiBaseUrl: 'http://api.test', request: async (path: string) => path === '/auth/registration-status' ? { nextUserIsInitialSuperAdmin: false, registrationEnabled: true } : { id: 1 } }),
      useAuthSession: () => ({ setUser: () => {} }),
      useAuthReturnNavigation: () => ({ returnFromAuth: async () => {}, authPageLink: (path: string) => path, destination: mountVue.ref('/') }),
      useWebOptions: () => ({ siteName: mountVue.ref('SForum'), siteTagline: mountVue.ref(''), altchaWidgetSettings: mountVue.ref({ hideLogo: true, hideFooter: true, minDuration: 0 }) }),
      useAuthProviders: () => ({ loginProviders: providers, redirectToProvider: async (id: string, operation: string) => { starts.push([id, operation]) } }),
      useExternalAuthFeedback: () => ({ alertMessage: mountVue.ref(''), alertVariant: mountVue.ref('danger') }),
      useAsyncData: async (_key: string, handler: () => Promise<unknown>) => ({ data: mountVue.ref(await handler()), pending: mountVue.ref(false) }),
      useSeoMeta: () => {}, apiErrorMessage: () => '', apiErrorReason: () => '',
      authProviderDisplayMeta: (provider: { label?: string, icon?: string }, fallback: string) => ({ label: provider.label || fallback, icon: provider.icon || 'i-lucide-key-round' })
    })
    const wrapper = mount({
      components: { SFLoginFormPage },
      template: '<Suspense><SFLoginFormPage /></Suspense>'
    }, {
      global: { stubs: { NuxtLink: { template: '<a><slot /></a>' }, UIcon: true, SFAlert: true, ClientOnly: true, 'altcha-widget': true, SFAuthProviderButtons } }
    })
    try {
      await flushPromises()
      expect(wrapper.get('#login-input').exists()).toBe(true)
      expect(wrapper.get('#password-input').exists()).toBe(true)
      const provider = wrapper.get('[data-provider-id="example.provider.auth"]')
      await provider.trigger('click')
      expect(starts).toEqual([['example.provider.auth', 'login']])
      providers.value = []
      await mountVue.nextTick()
      expect(wrapper.find('[data-testid="auth-provider-buttons"]').exists()).toBe(false)
      expect(wrapper.get('#login-input').exists()).toBe(true)
    } finally {
      wrapper.unmount()
    }
  })

  test('mounts the production registration form with an independent catalog operation and fallback after start failure', async () => {
    const starts: Array<[string, string]> = []
    const providers = mountVue.ref([{
      id: 'example.provider.auth', kind: 'auth', contractVersion: 'example.auth@1', priority: 1,
      operations: ['registration.start'], activatedOperations: ['registration'], label: 'Example Register', icon: 'i-lucide-key-round'
    }])
    Object.assign(globalThis, {
      computed: mountVue.computed, ref: mountVue.ref, reactive: mountVue.reactive, watch: mountVue.watch,
      useI18n: () => ({ locale: mountVue.ref('en-US'), t: (key: string, values?: Record<string, string>) => key === 'auth.providers.continueWith' ? `Continue with ${values?.name}` : key }),
      useToast: () => ({ add: () => {} }), useLocalePath: () => (path: string) => path,
      useRoute: () => ({ path: '/register', query: {} }), useRouter: () => ({ replace: async () => {} }),
      useApiClient: () => ({ apiBaseUrl: 'http://api.test', request: async (path: string) => path === '/auth/registration-status' ? { nextUserIsInitialSuperAdmin: false, registrationEnabled: true } : { id: 1 } }),
      useAuthSession: () => ({ setUser: () => {} }),
      useAuthReturnNavigation: () => ({ returnFromAuth: async () => {}, authPageLink: (path: string) => path, destination: mountVue.ref('/') }),
      useWebOptions: () => ({ siteName: mountVue.ref('SForum'), siteTagline: mountVue.ref(''), humanVerificationEnabledFor: () => false, altchaWidgetSettings: mountVue.ref({ hideLogo: true, hideFooter: true, minDuration: 0, type: 'checkbox', auto: 'off', display: 'default', workers: 1 }), passwordPolicy: mountVue.ref({ minLength: 1, maxLength: 64 }) }),
      useAuthProviders: () => ({ registrationProviders: providers, redirectToProvider: async (id: string, operation: string) => { starts.push([id, operation]); throw new Error('provider start failed') } }),
      useExternalAuthFeedback: () => ({ alertMessage: mountVue.ref(''), alertVariant: mountVue.ref('danger') }),
      useAsyncData: async (_key: string, handler: () => Promise<unknown>) => ({ data: mountVue.ref(await handler()), pending: mountVue.ref(false) }),
      useSeoMeta: () => {}, apiErrorFields: () => ({}), apiErrorMessage: (error: Error) => error.message, apiErrorReason: () => '',
      registerErrorMessage: () => '', passwordPolicyProgress: () => 100, passwordPolicyProgressLevel: () => 'strong', passwordPolicyRequirements: () => [],
      authProviderDisplayMeta: (provider: { label?: string, icon?: string }, fallback: string) => ({ label: provider.label || fallback, icon: provider.icon || 'i-lucide-key-round' })
    })
    const wrapper = mount({ components: { SFRegisterFormPage }, template: '<Suspense><SFRegisterFormPage /></Suspense>' }, {
      global: { stubs: { NuxtLink: { template: '<a><slot /></a>' }, UIcon: true, SFAlert: { props: ['title'], template: '<div>{{ title }}<slot /></div>' }, ClientOnly: true, 'altcha-widget': true, SFAuthProviderButtons } }
    })
    try {
      await flushPromises()
      expect(wrapper.get('#reg-password-input').exists()).toBe(true)
      await wrapper.get('[data-provider-id="example.provider.auth"]').trigger('click')
      await flushPromises()
      expect(starts).toEqual([['example.provider.auth', 'registration']])
      expect(wrapper.text()).toContain('provider start failed')
      providers.value = []
      await mountVue.nextTick()
      expect(wrapper.find('[data-testid="auth-provider-buttons"]').exists()).toBe(false)
      expect(wrapper.get('#reg-password-input').exists()).toBe(true)
    } finally {
      wrapper.unmount()
    }
  })

  test('reuses the registration form and autofills only verified external hints', async () => {
    const submitted: Array<{ path: string, body: Record<string, unknown> }> = []
    Object.assign(globalThis, {
      computed: mountVue.computed, ref: mountVue.ref, reactive: mountVue.reactive, watch: mountVue.watch,
      useI18n: () => ({ locale: mountVue.ref('en-US'), t: (key: string) => key }),
      useToast: () => ({ add: () => {} }), useLocalePath: () => (path: string) => path,
      useRoute: () => ({ path: '/register', query: { ticket: 'opaque-ticket', redirect: '/topics' } }),
      useRouter: () => ({ replace: async () => {} }),
      useApiClient: () => ({
        apiBaseUrl: 'http://api.test',
        request: async (path: string, options?: { body?: Record<string, unknown> }) => {
          if (path === '/auth/external-registration/prepare') {
            expect(options?.body).toEqual({ ticket: 'opaque-ticket' })
            return { usernameHint: 'octocat', emailHint: 'octocat@example.com', displayName: 'The Octocat', emailVerified: true }
          }
          if (path === '/auth/registration-status') {
            return { nextUserIsInitialSuperAdmin: false, registrationEnabled: true }
          }
          if (path === '/auth/external-registration') {
            submitted.push({ path, body: options?.body || {} })
            return { id: 42, username: 'octocat' }
          }
          throw new Error(`unexpected request ${path}`)
        }
      }),
      useAuthSession: () => ({ setUser: () => {} }),
      useAuthReturnNavigation: () => ({ returnFromAuth: async () => {}, authPageLink: (path: string) => path, destination: mountVue.ref('/topics') }),
      useWebOptions: () => ({ humanVerificationEnabledFor: () => false, altchaWidgetSettings: mountVue.ref({ hideLogo: true, hideFooter: true, minDuration: 0, type: 'checkbox', auto: 'off', display: 'default', workers: 1 }), passwordPolicy: mountVue.ref({ minLength: 1, maxLength: 64 }) }),
      useAuthProviders: () => ({ registrationProviders: mountVue.ref([]), redirectToProvider: async () => {} }),
      useExternalAuthFeedback: () => ({ alertMessage: mountVue.ref(''), alertVariant: mountVue.ref('danger') }),
      useAsyncData: async (_key: string, handler: () => Promise<unknown>) => ({ data: mountVue.ref(await handler()), pending: mountVue.ref(false) }),
      useSeoMeta: () => {}, apiErrorFields: () => ({}), apiErrorMessage: () => '', apiErrorReason: () => '',
      registerErrorMessage: () => '', passwordPolicyProgress: () => 100, passwordPolicyProgressLevel: () => 'strong', passwordPolicyRequirements: () => []
    })
    const wrapper = mount({ components: { SFRegisterFormPage }, template: '<Suspense><SFRegisterFormPage /></Suspense>' }, {
      global: { stubs: { NuxtLink: { template: '<a><slot /></a>' }, UIcon: true, SFAlert: { props: ['title'], template: '<div>{{ title }}<slot /></div>' }, ClientOnly: true, 'altcha-widget': true, SFAuthProviderButtons } }
    })
    try {
      await flushPromises()
      expect((wrapper.get('#username-input').element as HTMLInputElement).value).toBe('octocat')
      expect((wrapper.get('#email-input').element as HTMLInputElement).value).toBe('octocat@example.com')
      expect((wrapper.get('#displayname-input').element as HTMLInputElement).value).toBe('The Octocat')
      expect(wrapper.find('#reg-password-input').exists()).toBe(false)
      await wrapper.get('form').trigger('submit')
      await flushPromises()
      expect(submitted).toHaveLength(1)
      expect(submitted[0]?.body).toMatchObject({
        ticket: 'opaque-ticket', username: 'octocat', email: 'octocat@example.com', displayName: 'The Octocat'
      })
    } finally {
      wrapper.unmount()
    }
  })
})
