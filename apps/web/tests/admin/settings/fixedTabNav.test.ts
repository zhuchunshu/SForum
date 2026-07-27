import { describe, expect, test } from 'bun:test'
import { fileURLToPath } from 'node:url'
import { compileVueSfc, mount } from '../../helpers/vueSfc'

const componentPath = fileURLToPath(
  new URL('../../../app/components/admin/settings/shared/SFAdminFixedTabNav.vue', import.meta.url)
)
const FixedTabNav = await compileVueSfc(componentPath, 'fixed-tab-nav')

describe('fixed admin tab navigation', () => {
  test('exposes one selected ARIA tab and emits an explicit selection', async () => {
    const wrapper = mount(FixedTabNav, {
      props: {
        items: [
          { id: 'basic', label: 'Basic', icon: 'i-lucide-sliders' },
          { id: 'security', label: 'Security', icon: 'i-lucide-shield' }
        ],
        modelValue: 'basic',
        ariaLabel: 'Settings sections'
      },
      global: {
        stubs: {
          UButton: {
            props: ['disabled'],
            emits: ['click'],
            template: '<button :disabled="disabled" @click="$emit(\'click\', $event)"><slot /></button>'
          }
        }
      }
    })
    try {
      expect(wrapper.get('[role="tablist"]').attributes('aria-label')).toBe('Settings sections')
      const tabs = wrapper.findAll('[role="tab"]')
      expect(tabs.map(tab => tab.attributes('aria-selected'))).toEqual(['true', 'false'])
      await tabs[1]!.trigger('click')
      expect(wrapper.emitted('update:modelValue')).toEqual([['security']])
    } finally {
      wrapper.unmount()
    }
  })
})
