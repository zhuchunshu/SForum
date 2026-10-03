import { afterEach, describe, expect, test } from 'bun:test'
import { defineComponent, h, nextTick } from 'vue'
import { mount } from '@vue/test-utils'

import {
  closeForumMentionPreview,
  openForumMentionPreview,
  useForumMentionPreviewHost
} from '../../app/composables/forum/useForumMentionPreview'

// 宿主组件：真实 composable + 一个承载 layerRef 的浮层节点，用于验证单例交互契约。
const Harness = defineComponent({
  name: 'MentionPreviewHarness',
  setup() {
    const { state, layerRef, previewAvailable } = useForumMentionPreviewHost()
    return { state, layerRef, previewAvailable }
  },
  render() {
    return h('div', [
      this.state.open && this.state.target
        ? h('div', { ref: 'layerRef', class: 'layer' }, 'card')
        : null
    ])
  }
})

function testAnchor(top = 100) {
  const anchor = document.createElement('a')
  anchor.className = 'sf-mention'
  anchor.setAttribute('data-sf-mention', 'alice')
  anchor.setAttribute('aria-expanded', 'false')
  anchor.setAttribute('href', '/u/alice')
  anchor.getBoundingClientRect = () => ({
    top,
    bottom: top + 20,
    left: 4,
    width: 40,
    right: 44,
    height: 20,
    x: 4,
    y: top,
    toJSON: () => ({})
  }) as DOMRect
  document.body.appendChild(anchor)
  return anchor
}

const target = {
  username: 'alice',
  displayName: 'alice',
  avatar: null,
  profilePath: '/u/alice'
}

afterEach(() => {
  closeForumMentionPreview()
  document.body.innerHTML = ''
})

describe('forum mention preview host', () => {
  test('requires a mounted host before intercepting mention clicks', async () => {
    const wrapper = mount(Harness)
    expect(wrapper.vm.previewAvailable).toBe(true)

    const anchor = testAnchor()
    openForumMentionPreview(target, anchor)
    expect(wrapper.vm.state.open).toBe(true)
    expect(wrapper.vm.state.target?.username).toBe('alice')
    expect(anchor.getAttribute('aria-expanded')).toBe('true')
    expect(wrapper.vm.state.style?.width).toBe('320px')
    await nextTick()
    expect(wrapper.find('.layer').exists()).toBe(true)

    wrapper.unmount()
    expect(wrapper.vm.previewAvailable).toBe(false)
    // 卸载后不得留下悬挂状态。
    expect(wrapper.vm.state.open).toBe(false)
  })

  test('flips above the anchor when the mention sits near the viewport bottom', () => {
    const wrapper = mount(Harness)
    const anchor = testAnchor(window.innerHeight - 40)

    openForumMentionPreview(target, anchor)

    expect(wrapper.vm.state.style?.transform).toBe('translateY(-100%)')
  })

  test('closes on Escape with focus restored, and on outside pointerdown', () => {
    const wrapper = mount(Harness)
    const anchor = testAnchor()
    openForumMentionPreview(target, anchor)

    document.dispatchEvent(new KeyboardEvent('keydown', { key: 'Escape' }))
    expect(wrapper.vm.state.open).toBe(false)
    expect(anchor.getAttribute('aria-expanded')).toBe('false')
    expect(document.activeElement).toBe(anchor)

    openForumMentionPreview(target, anchor)
    expect(wrapper.vm.state.open).toBe(true)
    document.body.dispatchEvent(new MouseEvent('pointerdown', { bubbles: true }))
    expect(wrapper.vm.state.open).toBe(false)
  })

  test('keeps the card open when the pointer goes down inside it, and replaces the target', async () => {
    const wrapper = mount(Harness)
    const anchor = testAnchor()
    openForumMentionPreview(target, anchor)
    await nextTick()

    wrapper.find('.layer').element.dispatchEvent(new MouseEvent('pointerdown', { bubbles: true }))
    expect(wrapper.vm.state.open).toBe(true)

    const second = testAnchor(200)
    openForumMentionPreview({ ...target, username: 'bob' }, second)
    expect(wrapper.vm.state.target?.username).toBe('bob')
    expect(anchor.getAttribute('aria-expanded')).toBe('false')
    expect(second.getAttribute('aria-expanded')).toBe('true')
  })
})
