import { describe, expect, test } from 'bun:test'
import { fileURLToPath } from 'node:url'
import { compileVueSfc, mount, testVue } from '../helpers/vueSfc'

Object.assign(globalThis, { ref: testVue.ref })

const componentPath = fileURLToPath(
  new URL('../../app/components/SFSearch.vue', import.meta.url)
)
const SFSearch = await compileVueSfc(componentPath, 'sf-search-submit')

describe('SFSearch submission', () => {
  test('submits the current trimmed input through Enter and the accessible command', async () => {
    const wrapper = mount(SFSearch, {
      props: {
        modelValue: '',
        ariaLabel: 'Search forum'
      },
      global: {
        stubs: {
          UIcon: true
        }
      }
    })

    try {
      const input = wrapper.get('input[type="search"]')
      await input.setValue('  launch  ')

      expect(wrapper.emitted('update:modelValue')).toEqual([['  launch  ']])

      await input.trigger('keydown', { key: 'Enter', isComposing: true })
      expect(wrapper.emitted('submit')).toBeUndefined()

      await input.trigger('keydown', { key: 'Enter' })
      expect(wrapper.emitted('submit')).toEqual([['launch']])

      const submit = wrapper.get('button[type="button"]')
      expect(submit.attributes('aria-label')).toBe('Search forum')
      expect(submit.attributes('title')).toBe('Search forum')

      await submit.trigger('click')
      expect(wrapper.emitted('submit')).toEqual([['launch'], ['launch']])
    } finally {
      wrapper.unmount()
    }
  })
})
