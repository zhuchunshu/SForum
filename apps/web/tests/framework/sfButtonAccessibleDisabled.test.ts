import { describe, expect, test } from 'bun:test'
import { fileURLToPath } from 'node:url'
import { compileVueSfc, mount, testVue } from '../helpers/vueSfc'

Object.assign(globalThis, { computed: testVue.computed })

const componentPath = fileURLToPath(
  new URL('../../app/components/SFButton.vue', import.meta.url)
)
const SFButton = await compileVueSfc(componentPath, 'sf-button-accessible-disabled')

describe('SFButton accessible disabled state', () => {
  test('keeps aria-disabled validation actions clickable', async () => {
    const wrapper = mount(SFButton, {
      attrs: { 'aria-disabled': 'true' },
      slots: { default: 'Save' }
    })

    try {
      expect(wrapper.attributes('aria-disabled')).toBe('true')
      expect(wrapper.attributes('disabled')).toBeUndefined()

      await wrapper.trigger('click')
      expect(wrapper.emitted('click')).toHaveLength(1)
    } finally {
      wrapper.unmount()
    }
  })

  test('keeps true disabled actions non-interactive', async () => {
    const wrapper = mount(SFButton, {
      props: { disabled: true },
      slots: { default: 'Save' }
    })

    try {
      expect(wrapper.attributes('disabled')).toBe('')

      await wrapper.trigger('click')
      expect(wrapper.emitted('click')).toBeUndefined()
    } finally {
      wrapper.unmount()
    }
  })
})
