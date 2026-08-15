import { describe, expect, test } from 'bun:test'
// Static import intentionally mirrors tests/framework/editorL2Load.test.ts:
// sfEditor pulls in @tiptap/vue-3 -> vue before any test body runs. In a shared
// `bun test` module registry this is the first Vue import in the process. If the
// bunfig test preload has not installed a `document` first, @vue/runtime-dom
// freezes its internal `doc` binding to `null` and the mount below throws
// `TypeError: null is not an object (evaluating 'doc.createElement')`.
import { createSFEditorExtensions } from '../../app/utils/sfEditor'
import { installTestDom } from '../helpers/dom'

installTestDom()

const { mount } = await import('@vue/test-utils')
const { defineComponent } = await import('vue')

describe('editor module + Vue mount load-order regression', () => {
  test('mounts a component after the editor statically imported Vue first', () => {
    expect(createSFEditorExtensions({
      placeholder: 'x',
      maxCharacters: 100,
      trustedExtensions: []
    }).length).toBeGreaterThan(3)

    const Component = defineComponent({ template: '<div data-load-order="ok">mounted</div>' })
    const wrapper = mount(Component, { attachTo: document.body })
    expect(wrapper.get('[data-load-order="ok"]').text()).toBe('mounted')
    wrapper.unmount()
  })
})
