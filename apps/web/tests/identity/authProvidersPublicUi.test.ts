import { describe, expect, test } from 'bun:test'
import { fileURLToPath } from 'node:url'
import { join } from 'node:path'
import { compileScript, compileTemplate, parse, rewriteDefault } from '@vue/compiler-sfc'
import { installTestDom } from '../helpers/dom'

installTestDom()

const root = fileURLToPath(new URL('../..', import.meta.url))
const Vue = await import('vue')
const { mount } = await import('@vue/test-utils')

// Nuxt injects these composables at build time. The production component is
// compiled by Vite here, while the test provides the smallest deterministic
// Host shell required to exercise its rendered DOM.
Object.assign(globalThis, {
  computed: Vue.computed,
  useI18n: () => ({
    t: (key: string, values?: Record<string, string>) => {
      if (key === 'auth.providers.continueWith') return `Continue with ${values?.name || ''}`
      if (key === 'auth.providers.starting') return 'Starting'
      if (key === 'auth.providers.divider') return 'or continue with'
      if (key === 'auth.providers.genericName') return 'third-party account'
      return key
    }
  })
})

function compileProductionComponent(path: string) {
  const source = Bun.file(path).text()
  return source.then((content) => {
    const parsed = parse(content, { filename: path })
    const script = compileScript(parsed.descriptor, { id: 'external-auth-public-ui' })
    const template = compileTemplate({
      source: parsed.descriptor.template?.content || '', filename: path, id: 'external-auth-public-ui',
      compilerOptions: { bindingMetadata: script.bindings }
    })
    if (template.errors.length) throw new Error(String(template.errors[0]))
    const scriptCode = script.content
      .replace("import { defineComponent as _defineComponent } from 'vue'", 'const { defineComponent: _defineComponent } = Vue')
      .replace(/import type[^\n]+\n/, '')
      .replace(/import \{ authProviderDisplayMeta \} from [^\n]+\n/, '')
    const templateCode = template.code
      .replace(/import \{([^}]+)\} from \"vue\"/, (_match, names) => `const {${names.replace(/(\w+) as (\w+)/g, '$1: $2')}} = Vue`)
      .replace('export function render', 'function render')
    const code = `${rewriteDefault(scriptCode, '__sfc__', ['typescript'])}
${templateCode}
__sfc__.render = render
return __sfc__`
    const executable = new Bun.Transpiler({ loader: 'ts', target: 'esnext' }).transformSync(code)
    return new Function('Vue', 'authProviderDisplayMeta', executable)(Vue, (provider: Record<string, unknown>, fallback: string) => ({
      label: String(provider.label || fallback), icon: String(provider.icon || 'i-lucide-key-round')
    }))
  })
}

const SFAuthProviderButtons = await compileProductionComponent(join(root, 'app/components/identity/SFAuthProviderButtons.vue'))

describe('public auth provider buttons', () => {
  test('mounts the production component from catalog metadata and emits its generic start event', async () => {
    const wrapper = mount(SFAuthProviderButtons, {
      attachTo: document.body,
      props: {
        operation: 'login',
        providers: [{
          id: 'example.provider.auth', kind: 'auth', contractVersion: 'example.auth@1', priority: 10,
          operations: ['login.start'], activatedOperations: ['login'],
          label: 'Example Login', icon: 'i-lucide-key-round'
        }]
      },
      global: { stubs: { UIcon: { props: ['name'], template: '<i :data-icon="name" />' } } }
    })
    try {
      const button = wrapper.get('[data-provider-id="example.provider.auth"]')
      expect(button.text()).toContain('Continue with Example Login')
      expect(wrapper.get('[data-icon]').attributes('data-icon')).toBe('i-lucide-key-round')
      await button.trigger('click')
      expect(wrapper.emitted('start')).toEqual([[expect.objectContaining({ id: 'example.provider.auth' })]])
    } finally {
      wrapper.unmount()
    }
  })

  test('keeps password fallback space empty and makes loading/disabled state observable', () => {
    const empty = mount(SFAuthProviderButtons, {
      props: { operation: 'registration', providers: [] },
      global: { stubs: { UIcon: true } }
    })
    const loading = mount(SFAuthProviderButtons, {
      props: {
        operation: 'login', startingId: 'example.provider.auth', providers: [{
          id: 'example.provider.auth', kind: 'auth', contractVersion: 'example.auth@1', priority: 10,
          operations: ['login.start'], activatedOperations: ['login'], label: 'Catalog label', icon: 'i-lucide-key-round'
        }]
      },
      global: { stubs: { UIcon: true } }
    })
    try {
      expect(empty.find('[data-testid="auth-provider-buttons"]').exists()).toBe(false)
      const button = loading.get('[data-provider-id="example.provider.auth"]')
      expect(button.attributes('disabled')).toBeDefined()
      expect(button.attributes('aria-busy')).toBe('true')
      expect(button.text()).toContain('Starting')
    } finally {
      empty.unmount()
      loading.unmount()
    }
  })
})
