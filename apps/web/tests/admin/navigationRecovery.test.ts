import { beforeAll, describe, expect, test } from 'bun:test'
import { fileURLToPath } from 'node:url'
import { compileVueSfc, flushPromises, mount, testVue } from '../helpers/vueSfc'

const defaultsPath = fileURLToPath(new URL('../../app/components/admin/settings/personalization/navigation/SFAdminNavigationDefaultsDialog.vue', import.meta.url))
let DefaultsDialog: Awaited<ReturnType<typeof compileVueSfc>>

beforeAll(async () => {
  Object.assign(globalThis, {
    ref: testVue.ref,
    computed: testVue.computed,
    watch: testVue.watch,
    useI18n: () => ({ t: (key: string) => key }),
    apiErrorMessage: () => '',
    apiErrorStatusCode: () => 0
  })
  DefaultsDialog = await compileVueSfc(defaultsPath, 'navigation-defaults-dialog')
})

describe('navigation recovery UI', () => {
  test('fences recommended-default apply behind a server preview and explicit confirmation', async () => {
    const calls: Array<{ action: string, body: unknown }> = []
    const recovered = { revision: 8, definitions: [], placements: [], themeLocations: [] }
    Object.assign(globalThis, {
      useSiteChromeApi: () => ({
        previewNavigationDefaults: async (body: unknown) => {
          calls.push({ action: 'preview', body })
          return { previewToken: 'preview-7', expectedRevision: 7, mode: 'defaults', changes: ['public.topbar.primary'], warnings: [] }
        },
        applyNavigationDefaults: async (body: unknown) => {
          calls.push({ action: 'apply', body })
          return recovered
        }
      })
    })
    const wrapper = mount(DefaultsDialog, {
      props: {
        revision: 7,
        activeLocation: 'public.topbar.primary',
        locationOptions: [{ label: 'Top bar', value: 'public.topbar.primary' }]
      },
      global: { stubs: uiStubs() }
    })
    try {
      await button(wrapper, 'admin.navigationEditor.recovery.defaults.action').trigger('click')
      await button(wrapper, 'admin.navigationEditor.recovery.preview').trigger('click')
      await flushPromises()
      expect(calls).toEqual([{ action: 'preview', body: { expectedRevision: 7, scope: 'location', location: 'public.topbar.primary' } }])
      const apply = button(wrapper, 'admin.navigationEditor.recovery.defaults.apply')
      expect(apply.attributes('disabled')).toBeDefined()
      await wrapper.get('[data-checkbox]').trigger('click')
      await flushPromises()
      expect(button(wrapper, 'admin.navigationEditor.recovery.defaults.apply').attributes('disabled')).toBeUndefined()
      await button(wrapper, 'admin.navigationEditor.recovery.defaults.apply').trigger('click')
      await flushPromises()
      expect(calls[1]).toEqual({ action: 'apply', body: { expectedRevision: 7, previewToken: 'preview-7', reason: 'operator_restore_defaults:public.topbar.primary' } })
      expect(wrapper.emitted('applied')).toEqual([[recovered]])
    } finally {
      wrapper.unmount()
    }
  })
})

function uiStubs() {
  return {
    UButton: {
      props: ['disabled', 'loading'], emits: ['click'],
      template: '<button :disabled="disabled" @click="$emit(\'click\', $event)"><slot /></button>'
    },
    UModal: { props: ['open'], template: '<div v-if="open"><slot name="content" /></div>' },
    UAlert: { props: ['title', 'description'], template: '<div>{{ title }} {{ description }}</div>' },
    UFormField: { template: '<label><slot /></label>' },
    URadioGroup: true,
    USelect: true,
    UCheckbox: {
      props: ['modelValue', 'label'], emits: ['update:modelValue'],
      template: '<button data-checkbox @click="$emit(\'update:modelValue\', !modelValue)">{{ label }}</button>'
    }
  }
}

function button(wrapper: ReturnType<typeof mount>, label: string) {
  const match = wrapper.findAll('button').find(candidate => candidate.text() === label)
  if (!match) throw new Error(`button not found: ${label}`)
  return match
}
