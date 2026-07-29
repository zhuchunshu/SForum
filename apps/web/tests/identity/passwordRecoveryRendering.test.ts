import { afterEach, describe, expect, test } from 'bun:test'
import { Window } from 'happy-dom'
import { fileURLToPath } from 'node:url'
import { join } from 'node:path'
import { compileScript, compileTemplate, parse, rewriteDefault } from '@vue/compiler-sfc'

import {
  passwordPolicyProgress,
  passwordPolicyProgressLevel,
  passwordPolicyRequirements,
  type PasswordPolicy
} from '../../app/composables/useWebOptions'

const mountWindow = new Window({ url: 'http://localhost/forgot-password' })
Object.assign(globalThis, {
  window: mountWindow,
  document: mountWindow.document,
  navigator: mountWindow.navigator,
  Element: mountWindow.Element,
  HTMLElement: mountWindow.HTMLElement,
  SVGElement: mountWindow.SVGElement,
  Node: mountWindow.Node,
  Event: mountWindow.Event,
  CustomEvent: mountWindow.CustomEvent,
  MouseEvent: mountWindow.MouseEvent
})

const mountRoot = fileURLToPath(new URL('../..', import.meta.url))
const Vue = await import('vue')
const { mount, flushPromises } = await import('@vue/test-utils')

function vueImportToConst(names: string) {
  return `const {${names.replace(/(\w+) as (\w+)/g, '$1: $2')}} = Vue`
}

async function compileMountedComponent(relativePath: string, bindings: Record<string, unknown> = {}) {
  const path = join(mountRoot, relativePath)
  const parsed = parse(await Bun.file(path).text(), { filename: path })
  const script = compileScript(parsed.descriptor, { id: `mount-${relativePath}` })
  const template = compileTemplate({
    source: parsed.descriptor.template?.content || '',
    filename: path,
    id: `mount-${relativePath}`,
    compilerOptions: { bindingMetadata: script.bindings }
  })
  if (template.errors.length) throw new Error(String(template.errors[0]))

  const scriptCode = `const { withAsyncContext: _withAsyncContext, defineComponent: _defineComponent } = Vue\n${script.content.replace(/^import[^\n]*\n/gm, '')}`
  const templateCode = template.code
    .replace(/import \{([^}]+)\} from "vue"/, (_match, names) => vueImportToConst(names))
    .replace('export function render', 'function render')
  const source = `${rewriteDefault(scriptCode, '__sfc__', ['typescript'])}\n${templateCode}\n__sfc__.render = render\nreturn __sfc__`
  const executable = new Bun.Transpiler({ loader: 'ts', target: 'esnext' }).transformSync(source)
  const names = Object.keys(bindings)
  return new Function('Vue', ...names, executable)(Vue, ...Object.values(bindings))
}

const SFRecoveryShell = Vue.defineComponent({
  props: { phase: { type: Number, required: true } },
  template: '<main data-testid="recovery-shell" :data-phase="phase"><slot /></main>'
})
const SFRecoveryRequestPage = await compileMountedComponent(
  'app/components/identity/SFRecoveryRequestPage.vue',
  { SFRecoveryShell }
)
const SFRecoveryConfirmPage = await compileMountedComponent(
  'app/components/identity/SFRecoveryConfirmPage.vue',
  { SFRecoveryShell }
)

const activeWrappers: Array<{ unmount: () => void }> = []
afterEach(() => {
  while (activeWrappers.length) activeWrappers.pop()?.unmount()
})

function t(key: string, values?: Record<string, unknown>) {
  if (key === 'auth.recovery.resendIn') return `${values?.seconds}s`
  if (key === 'auth.recovery.passwordProgress') return `${values?.met}/${values?.total}`
  if (key === 'auth.passwordRequirementLength') return `${values?.min}-${values?.max}`
  return key
}

function commonGlobals() {
  Object.assign(globalThis, {
    computed: Vue.computed,
    ref: Vue.ref,
    watch: Vue.watch,
    nextTick: Vue.nextTick,
    onBeforeUnmount: Vue.onBeforeUnmount,
    useI18n: () => ({ locale: Vue.ref('zh-CN'), t }),
    useLocalePath: () => (path: string) => path,
    useSForumSeo: () => {},
    apiErrorFields: () => ({}),
    apiErrorMessage: () => '',
    apiErrorReason: () => '',
    passwordPolicyProgress,
    passwordPolicyProgressLevel,
    passwordPolicyRequirements
  })
}

const globalComponents = {
  stubs: {
    NuxtLink: { props: ['to'], template: '<a :href="to"><slot /></a>' },
    UIcon: true,
    ClientOnly: { template: '<div><slot /></div>' },
    SFAlert: { props: ['title'], template: '<div role="alert">{{ title }}</div>' },
    'altcha-widget': true
  }
}

describe('password recovery production forms', () => {
  test('submits the non-enumerating request and shows the masked sent state', async () => {
    commonGlobals()
    const calls: Array<{ path: string, options: Record<string, unknown> }> = []
    const toasts: Array<Record<string, unknown>> = []
    Object.assign(globalThis, {
      useToast: () => ({ add: (item: Record<string, unknown>) => toasts.push(item) }),
      useWebOptions: () => ({
        siteName: Vue.ref('SForum'),
        humanVerificationEnabledFor: () => false,
        altchaWidgetSettings: Vue.ref({
          hideLogo: true,
          hideFooter: true,
          minDuration: 0,
          type: 'checkbox',
          auto: 'off',
          display: 'standard',
          workers: 1
        })
      }),
      useApiClient: () => ({
        apiBaseUrl: 'http://api.test',
        request: async (path: string, options: Record<string, unknown>) => {
          calls.push({ path, options })
          return { sent: true }
        }
      })
    })

    const wrapper = mount(SFRecoveryRequestPage, { global: globalComponents })
    activeWrappers.push(wrapper)
    await wrapper.get('#recovery-email').setValue('name@example.com')
    await wrapper.get('form').trigger('submit')
    await flushPromises()

    expect(calls).toEqual([{
      path: '/auth/password-reset/request',
      options: { method: 'POST', body: { email: 'name@example.com' } }
    }])
    expect(wrapper.find('[data-testid="recovery-request-view"]').exists()).toBe(false)
    expect(wrapper.get('[data-testid="recovery-sent-view"]').text()).toContain('n•••@example.com')
    expect(wrapper.get('[data-testid="recovery-shell"]').attributes('data-phase')).toBe('1')
    expect(toasts[0]?.duration).toBe(10000)

    await wrapper.get('.sf-recovery-button.is-secondary').trigger('click')
    await Vue.nextTick()
    expect(wrapper.get('[data-testid="recovery-request-view"]').exists()).toBe(true)
    expect(wrapper.get('[data-testid="recovery-request-view"] [href="/"]').exists()).toBe(true)
  })

  test('enforces runtime password policy, toggles visibility, and reaches completion', async () => {
    commonGlobals()
    const calls: Array<{ path: string, options: Record<string, unknown> }> = []
    const toasts: Array<Record<string, unknown>> = []
    const policy: PasswordPolicy = {
      minLength: 8,
      maxLength: 64,
      requireLowercase: true,
      requireUppercase: true,
      requireNumber: true,
      requireSymbol: true
    }
    Object.assign(globalThis, {
      useRoute: () => ({ query: { token: 'exact-reset-token' } }),
      useToast: () => ({ add: (item: Record<string, unknown>) => toasts.push(item) }),
      useWebOptions: () => ({ siteName: Vue.ref('SForum'), passwordPolicy: Vue.ref(policy) }),
      useApiClient: () => ({
        request: async (path: string, options: Record<string, unknown>) => {
          calls.push({ path, options })
          return { reset: true }
        }
      })
    })

    const wrapper = mount(SFRecoveryConfirmPage, { global: globalComponents })
    activeWrappers.push(wrapper)
    const newPassword = wrapper.get('#new-password')
    const confirmation = wrapper.get('#confirm-password')
    await newPassword.setValue('Strong-pass1!')
    await confirmation.setValue('different')
    expect(wrapper.text()).toContain('auth.passwordsDoNotMatch')
    expect(wrapper.get('button[type="submit"]').attributes('disabled')).toBeDefined()

    expect(newPassword.attributes('type')).toBe('password')
    await wrapper.get('.sf-recovery-input-action').trigger('click')
    expect(wrapper.get('#new-password').attributes('type')).toBe('text')

    await confirmation.setValue('Strong-pass1!')
    await wrapper.get('form').trigger('submit')
    await flushPromises()

    expect(calls).toEqual([{
      path: '/auth/password-reset/confirm',
      options: {
        method: 'POST',
        body: { token: 'exact-reset-token', newPassword: 'Strong-pass1!' }
      }
    }])
    expect(wrapper.get('[data-testid="recovery-complete-view"]').exists()).toBe(true)
    expect(wrapper.get('[data-testid="recovery-shell"]').attributes('data-phase')).toBe('3')
    expect(toasts[0]?.duration).toBe(10000)
  })

  test('shows the approved invalid-link state when the token is missing', () => {
    commonGlobals()
    Object.assign(globalThis, {
      useRoute: () => ({ query: {} }),
      useToast: () => ({ add: () => {} }),
      useWebOptions: () => ({
        siteName: Vue.ref('SForum'),
        passwordPolicy: Vue.ref({
          minLength: 12,
          maxLength: 128,
          requireLowercase: false,
          requireUppercase: false,
          requireNumber: false,
          requireSymbol: false
        })
      }),
      useApiClient: () => ({ request: async () => ({ reset: true }) })
    })

    const wrapper = mount(SFRecoveryConfirmPage, { global: globalComponents })
    activeWrappers.push(wrapper)
    expect(wrapper.get('[data-testid="recovery-invalid-view"]').exists()).toBe(true)
    expect(wrapper.get('[href="/forgot-password"]').exists()).toBe(true)
    expect(wrapper.get('[data-testid="recovery-shell"]').attributes('data-phase')).toBe('1')
  })
})
