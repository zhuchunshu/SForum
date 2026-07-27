import { describe, expect, test } from 'bun:test'
import { Window } from 'happy-dom'
import { fileURLToPath } from 'node:url'
import { join } from 'node:path'
import { compileScript, compileTemplate, parse, rewriteDefault } from '@vue/compiler-sfc'

import { asExternalIdentityList } from '../app/composables/useAccountSecurityApi'
import { authProviderDisplayMeta, providerSupportsOperation, type PublicAuthProvider } from '../app/composables/useAuthProviders'
import { resolveExternalAuthFeedback } from '../app/utils/externalAuthFeedback'

const securityWindow = new Window({ url: 'http://localhost/settings/security' })
Object.assign(globalThis, { window: securityWindow, document: securityWindow.document, navigator: securityWindow.navigator, Element: securityWindow.Element, HTMLElement: securityWindow.HTMLElement, SVGElement: securityWindow.SVGElement, Node: securityWindow.Node, Event: securityWindow.Event, MouseEvent: securityWindow.MouseEvent })
const securityRoot = fileURLToPath(new URL('..', import.meta.url))
const securityVue = await import('vue')
const { mount, flushPromises } = await import('@vue/test-utils')

async function compileSecurityComponent(relativePath: string) {
  const path = join(securityRoot, relativePath)
  const parsed = parse(await Bun.file(path).text(), { filename: path })
  const script = compileScript(parsed.descriptor, { id: `security-${relativePath}` })
  const template = compileTemplate({ source: parsed.descriptor.template?.content || '', filename: path, id: `security-${relativePath}`, compilerOptions: { bindingMetadata: script.bindings } })
  if (template.errors.length) throw new Error(String(template.errors[0]))
  const scriptCode = `const { withAsyncContext: _withAsyncContext, defineComponent: _defineComponent } = Vue\n${script.content.replace(/^import(?:[\s\S]*?from\s+)?['"][^'"]+['"];?\s*$/gm, '')}`
  const templateCode = template.code.replace(/import \{([^}]+)\} from \"vue\"/, (_match, names) => `const {${names.replace(/(\w+) as (\w+)/g, '$1: $2')}} = Vue`).replace('export function render', 'function render')
  const source = `${rewriteDefault(scriptCode, '__sfc__', ['typescript'])}\n${templateCode}\n__sfc__.render = render\nreturn __sfc__`
  return new Function('Vue', new Bun.Transpiler({ loader: 'ts', target: 'esnext' }).transformSync(source))(securityVue)
}

const SFLinkedAccountsSection = await compileSecurityComponent('app/components/SFLinkedAccountsSection.vue')
const SFSecuritySettingsPage = await compileSecurityComponent('app/components/SFSecuritySettingsPage.vue')

describe('linked account data contracts', () => {
  test('projects only the public redacted identity shape', () => {
    const identities = asExternalIdentityList([{
      linkId: 12, providerId: 'example.provider.auth', status: 'active', linkedAt: '2026-07-27T00:00:00Z',
      providerSubject: 'subject-must-not-cross-client-boundary', subjectDigest: 'digest-must-not-cross-client-boundary'
    }])
    expect(identities).toEqual([{
      linkId: 12, providerId: 'example.provider.auth', status: 'active', linkedAt: '2026-07-27T00:00:00Z'
    }])
    expect(JSON.stringify(identities)).not.toContain('subject-must-not-cross-client-boundary')
    expect(JSON.stringify(identities)).not.toContain('digest-must-not-cross-client-boundary')
  })

  test('uses the generic link operation and stable Host errors', () => {
    const provider: PublicAuthProvider = {
      id: 'example.provider.auth', kind: 'auth', contractVersion: 'example.auth@1', priority: 1,
      operations: ['link.start'], activatedOperations: ['link'], label: 'Example Login', icon: 'i-lucide-key-round'
    }
    expect(providerSupportsOperation(provider, 'link')).toBe(true)
    expect(authProviderDisplayMeta(provider, 'third-party account').label).toBe('Example Login')
    expect(resolveExternalAuthFeedback('auth.last_login_method_required')?.messageKey)
      .toBe('auth.external.reasons.lastLoginMethodRequired')
  })

  test('mounts production security and linked-account components with redacted data and real sensitive actions', async () => {
    const calls: string[] = []
    const identities = [{ linkId: 12, providerId: 'example.provider.auth', status: 'active', linkedAt: '2026-07-27T00:00:00Z', providerSubject: 'must-not-render', subjectDigest: 'must-not-render' }]
    const api = {
      listSessions: async () => ({ items: [], total: 0, page: 1, perPage: 20 }), listAPITokens: async () => ({ items: [] }),
      listExternalIdentities: async () => asExternalIdentityList(identities), unlinkExternalIdentity: async () => { calls.push('unlink'); throw { reason: 'auth.last_login_method_required' } },
      setupPassword: async () => { calls.push('password') }, revokeSession: async () => {}, revokeOtherSessions: async () => ({ revoked: 0 }), createAPIToken: async () => ({ token: '' }), revokeAPIToken: async () => {}, rotateAPIToken: async () => ({ token: '' })
    }
    Object.assign(globalThis, {
      computed: securityVue.computed, ref: securityVue.ref, reactive: securityVue.reactive, watch: securityVue.watch, onMounted: securityVue.onMounted,
      useI18n: () => ({ t: (key: string) => key }), useToast: () => ({ add: () => {} }), useLocalePath: () => (path: string) => path,
      useWebOptions: () => ({ siteName: securityVue.ref('SForum'), passwordPolicy: securityVue.ref({ minLength: 1, maxLength: 64 }) }), useAccountSecurityApi: () => api,
      useAuthProviders: () => ({ linkProviders: securityVue.ref([]), providers: securityVue.ref([{ id: 'example.provider.auth', label: 'Example Login', icon: 'i-lucide-key-round', activatedOperations: ['link'] }]), pending: securityVue.ref(false), redirectToProvider: async () => {}, refresh: async () => {} }),
      useSiteDateTime: () => ({ format: (value: string) => value }), useSForumSeo: () => {}, useRoute: () => ({ query: {} }),
      useAsyncData: async (_key: string, handler: () => Promise<unknown>, options?: { default?: () => unknown }) => ({ data: securityVue.ref(await handler().catch(() => options?.default?.())), pending: securityVue.ref(false), refresh: async () => {} }),
      apiErrorMessage: () => '', apiErrorReason: (error: { reason?: string }) => error?.reason || '', buildAuthPageLink: (path: string) => path,
      authProviderDisplayMeta: (provider: { label?: string, icon?: string }, fallback: string) => ({ label: provider.label || fallback, icon: provider.icon || 'i-lucide-key-round' }),
      passwordPolicyProgress: () => 100, passwordPolicyProgressLevel: () => 'strong', passwordPolicyRequirements: () => [{ key: 'length', met: true }]
    })
    const wrapper = mount({ components: { SFSecuritySettingsPage }, template: '<Suspense><SFSecuritySettingsPage /></Suspense>' }, {
      global: { stubs: { SFSettingsShell: { template: '<section><slot /><slot name="rail" /><slot name="head-actions" /></section>' }, SFCard: { template: '<div><slot /></div>' }, SFButton: { emits: ['click'], template: '<button @click="$emit(\'click\', $event)"><slot /></button>' }, SFAlert: { props: ['title', 'description'], template: '<div>{{ title }}{{ description }}<slot /></div>' }, SFSkeleton: true, SFEmptyState: true, UIcon: true, NuxtLink: { template: '<a><slot /></a>' }, SFLinkedAccountsSection } }
    })
    try {
      await flushPromises()
      expect(wrapper.get('[data-sforum-island-body="forum.component.settings_security"]').exists()).toBe(true)
      expect(wrapper.get('[data-testid="linked-accounts-section"]').text()).not.toContain('must-not-render')
      const unlink = wrapper.get('[data-testid="unlink-external-identity"]')
      await unlink.trigger('click')
      await flushPromises()
      expect(calls).toContain('unlink')
      expect(wrapper.text()).toContain('accountSecurity.linkedAccounts.lastMethodBlocked')
    } finally {
      wrapper.unmount()
    }
  })
})
