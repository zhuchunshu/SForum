import { describe, expect, test } from 'bun:test'
import { fileURLToPath } from 'node:url'
import { compileVueSfc, flushPromises, mount, testVue } from '../helpers/vueSfc'
import {
  sforumReferenceKind,
  sforumReferenceNumericID,
  sforumReferenceSelection,
  sforumProtectedKind,
  sforumProtectedSelection,
  inspectSForumShortcodeDocument
} from '../../app/utils/editor/shortcodes'

const tiptapVue = await import('@tiptap/vue-3')

let requestImpl: (path: string) => Promise<unknown> = async () => ({ items: [], hasMore: false })
const messages: Record<string, string> = {
  'common.back': 'Back', 'common.close': 'Close', 'common.retry': 'Retry',
  'composer.shortcodes.title': 'Insert reference',
  'composer.shortcodes.chooseDescription': 'Choose type',
  'composer.shortcodes.searchDescription': 'Search visible content',
  'composer.shortcodes.searchPlaceholder': 'Search',
  'composer.shortcodes.loading': 'Loading',
  'composer.shortcodes.empty': 'Empty',
  'composer.shortcodes.denied': 'Denied',
  'composer.shortcodes.loadError': 'Load failed',
  'composer.shortcodes.referenceUnavailable': 'Unavailable',
  'composer.shortcodes.unavailable': 'Unavailable',
  'composer.shortcodes.edit': 'Edit',
  'composer.shortcodes.delete': 'Delete',
  'composer.shortcodes.kinds.user': 'User',
  'composer.shortcodes.kinds.topic': 'Topic',
  'composer.shortcodes.kinds.comment': 'Comment',
  'composer.shortcodes.kinds.category': 'Category',
  'composer.shortcodes.kinds.friend-links': 'Friend links'
  , 'composer.shortcodes.kinds.login': 'Log in',
  'composer.shortcodes.kinds.reply': 'After replying',
  'composer.shortcodes.kinds.only-author': 'Authors only',
  'composer.shortcodes.protectedNodeLabel': '{kind} protected block',
  'composer.shortcodes.changeProtection': 'Change protection',
  'composer.shortcodes.unwrap': 'Unwrap',
  'composer.shortcodes.invalidProtectedBody': 'Protected content cannot be empty',
  'composer.shortcodes.selectProtectedBody': 'Select content first',
  'composer.shortcodes.inserted': 'Inserted',
  'composer.shortcodes.updated': 'Updated',
  'composer.shortcodes.unwrapped': 'Unwrapped'
}

Object.assign(globalThis, {
  ...testVue,
  useI18n: () => ({ t: (key: string, params?: Record<string, unknown>) => {
    if (key === 'composer.shortcodes.nodeLabel') return `${String(params?.kind)} preview`
    if (key === 'composer.shortcodes.searchLabel') return `Search ${String(params?.kind)}`
    return messages[key] || key
  } }),
  useApiClient: () => ({ request: (path: string) => requestImpl(path) }),
  apiErrorStatusCode: (error: unknown) => Number((error as { status?: number })?.status || 0),
  sforumReferenceKind,
  sforumReferenceNumericID,
    sforumReferenceSelection,
  sforumProtectedKind,
  sforumProtectedSelection,
  nodeViewProps: tiptapVue.nodeViewProps
})

const modalStub = testVue.defineComponent({
  props: { open: Boolean },
  template: '<div v-if="open"><slot name="content" /></div>'
})
const buttonStub = testVue.defineComponent({
  inheritAttrs: false,
  emits: ['click'],
  template: '<button v-bind="$attrs" @click="$emit(\'click\', $event)"><slot /></button>'
})
const inputStub = testVue.defineComponent({
  props: { modelValue: { type: String, default: '' } },
  emits: ['update:modelValue'],
  template: '<input :value="modelValue" @input="$emit(\'update:modelValue\', $event.target.value)">'
})
const commonStubs = { UModal: modalStub, UButton: buttonStub, UInput: inputStub, UIcon: true, SFAvatar: true }

const dialogPath = fileURLToPath(new URL(
  '../../app/components/editor/shortcodes/SFShortcodeReferenceDialog.vue', import.meta.url
))
const toolbarPath = fileURLToPath(new URL('../../app/components/editor/SFEditorToolbar.vue', import.meta.url))
const nodeViewPath = fileURLToPath(new URL(
  '../../app/components/editor/shortcodes/SFShortcodeReferenceNodeView.vue', import.meta.url
))
const Dialog = await compileVueSfc(dialogPath, 'shortcode-reference-dialog')
const Toolbar = await compileVueSfc(toolbarPath, 'shortcode-toolbar')

describe('shortcode editor UI', () => {
  test('renders the admitted shortcode action inside the existing toolbar', async () => {
    const wrapper = mount(Toolbar, {
      props: {
        preset: 'full', disabled: false, canUndo: false, canRedo: false,
        active: {}, blockFormat: 'paragraph', viewMode: 'write',
        extensionTools: [{ id: 'shortcodes', label: 'Insert reference', icon: 'i-lucide-brackets' }]
      },
      global: { stubs: { UIcon: true } }
    })
    try {
      expect(wrapper.findAll('.sf-editor__topbar')).toHaveLength(1)
      const action = wrapper.get('button[aria-label="Insert reference"]')
      await action.trigger('click')
      expect(wrapper.emitted('extension-action')).toEqual([['shortcodes']])
    } finally {
      wrapper.unmount()
    }
  })

  test('shows loading, searchable results, empty, denied, and error states inline', async () => {
    let resolveRequest: ((value: unknown) => void) | undefined
    requestImpl = async () => new Promise(resolve => { resolveRequest = resolve })
    const wrapper = mount(Dialog, {
      props: { open: false, resources: ['user', 'topic', 'comment', 'category', 'friend-links'] },
      global: { stubs: commonStubs }
    })
    try {
      await wrapper.setProps({ open: true })
      await wrapper.findAll('.sf-shortcode-dialog__resource')[0]!.trigger('click')
      expect(wrapper.text()).toContain('Loading')
      resolveRequest?.({ items: [{ id: 7, label: 'Alice', secondaryLabel: '@alice' }], hasMore: false })
      await flushPromises()
      expect(wrapper.text()).toContain('Alice')
      await wrapper.get('.sf-shortcode-dialog__option').trigger('click')
      expect(wrapper.emitted('resolve')?.at(-1)).toEqual([{ action: 'upsert', kind: 'user', id: 7 }])
    } finally {
      wrapper.unmount()
    }

    for (const state of ['empty', 'denied', 'error'] as const) {
      requestImpl = state === 'empty'
        ? async () => ({ items: [], hasMore: false })
        : async () => { throw { status: state === 'denied' ? 403 : 500 } }
      const stateWrapper = mount(Dialog, {
        props: { open: false, resources: ['topic'] },
        global: { stubs: commonStubs }
      })
      try {
        await stateWrapper.setProps({ open: true })
        await stateWrapper.get('.sf-shortcode-dialog__resource').trigger('click')
        await flushPromises()
        expect(stateWrapper.text()).toContain(state === 'empty' ? 'Empty' : state === 'denied' ? 'Denied' : 'Load failed')
        if (state !== 'empty') expect(stateWrapper.get('[role="alert"]').exists()).toBe(true)
      } finally {
        stateWrapper.unmount()
      }
    }
  })

  test('inserts friend links directly from the compact reference menu', async () => {
    const wrapper = mount(Dialog, {
      props: { open: false, resources: ['friend-links'] },
      global: { stubs: commonStubs }
    })
    try {
      await wrapper.setProps({ open: true })
      await wrapper.get('.sf-shortcode-dialog__resource').trigger('click')
      expect(wrapper.emitted('resolve')).toEqual([[{ action: 'upsert', kind: 'friend-links' }]])
    } finally {
      wrapper.unmount()
    }
  })

  test('offers protected kinds according to topic/comment placement', async () => {
    const topic = mount(Dialog, {
      props: { open: false, resources: ['login', 'reply', 'only-author'] },
      global: { stubs: commonStubs }
    })
    const comment = mount(Dialog, {
      props: { open: false, resources: ['login', 'reply', 'only-author'] },
      global: { stubs: commonStubs }
    })
    try {
      await topic.setProps({ open: true })
      expect(topic.findAll('.sf-shortcode-dialog__resource')).toHaveLength(3)
      await topic.get('.sf-shortcode-dialog__resource').trigger('click')
      expect(topic.emitted('resolve')).toEqual([[{ action: 'upsert', kind: 'login' }]])

      await comment.setProps({ open: true })
      expect(comment.findAll('.sf-shortcode-dialog__resource')).toHaveLength(3)
      await comment.findAll('.sf-shortcode-dialog__resource')[2]!.trigger('click')
      expect(comment.emitted('resolve')).toEqual([[{ action: 'upsert', kind: 'only-author' }]])
    } finally {
      topic.unmount()
      comment.unmount()
    }
  })
})

describe('shortcode reference NodeView', () => {
  test('renders the selected node without inline actions and opens edit on keyboard activation', async () => {
    const edits: unknown[] = []
    const deletions: unknown[] = []
    const NodeViewWrapper = testVue.defineComponent({ template: '<div><slot /></div>' })
    const NodeViewContent = testVue.defineComponent({ template: '<div><slot /></div>' })
    Object.assign(globalThis, { NodeViewWrapper, NodeViewContent })
    const NodeView = await compileVueSfc(nodeViewPath, 'shortcode-reference-node-view')
    const host = {
      disabled: () => false,
      resolveReference: async () => ({ label: 'Visible topic', secondaryLabel: 'General', icon: 'i-lucide-message-square-text' }),
      editReference: (value: unknown) => edits.push(value),
      deleteReference: (value: unknown) => deletions.push(value)
    }
    const wrapper = mount(NodeView, {
      props: {
        node: { attrs: { id: 'sforum-shortcodes.topic', contractVersion: 'sforum-shortcodes.topic@1', arguments: { topicId: 42 } } },
        extension: { options: { nodeViewHost: host } }, selected: true, getPos: () => 5,
        editor: {}, view: {}, decorations: [], innerDecorations: {}, HTMLAttributes: {}, updateAttributes: () => {}, deleteNode: () => {}
      },
      global: { stubs: { NodeViewWrapper, SFAvatar: true, UIcon: true } }
    })
    try {
      await flushPromises()
      expect(wrapper.text()).toContain('Visible topic')
      expect(wrapper.get('.sf-shortcode-node').attributes('role')).toBe('group')
      expect(wrapper.findAll('button')).toHaveLength(0)
      expect(wrapper.get('.sf-shortcode-node__contract').text()).toContain('1')
      await wrapper.get('.sf-shortcode-node').trigger('keydown.enter')
      expect(edits).toHaveLength(1)
      expect(deletions).toHaveLength(0)
    } finally {
      wrapper.unmount()
    }
  })

  test('renders user avatar metadata and category icon/color metadata', async () => {
    const NodeViewWrapper = testVue.defineComponent({ template: '<div><slot /></div>' })
    const AvatarStub = testVue.defineComponent({
      props: { name: String, avatar: Object },
      template: '<span data-avatar-preview>{{ name }}</span>'
    })
    const IconStub = testVue.defineComponent({
      props: { name: String },
      template: '<span data-icon-preview>{{ name }}</span>'
    })
    Object.assign(globalThis, { NodeViewWrapper })
    const NodeView = await compileVueSfc(nodeViewPath, 'shortcode-reference-node-view-metadata')
    const host = {
      disabled: () => false,
      resolveReference: async (kind: string) => kind === 'user'
        ? { label: 'Alice', secondaryLabel: '@alice', avatar: { kind: 'initials', url: '', alt: 'Alice' } }
        : { label: 'Design', secondaryLabel: 'Public', icon: 'i-lucide-palette', iconColor: '#7c3aed' },
      editReference: () => undefined,
      deleteReference: () => undefined
    }
    const baseProps = {
      extension: { options: { nodeViewHost: host } }, selected: false, getPos: () => 5,
      editor: {}, view: {}, decorations: [], innerDecorations: {}, HTMLAttributes: {}, updateAttributes: () => {}, deleteNode: () => {}
    }
    const user = mount(NodeView, {
      props: { ...baseProps, node: { attrs: { id: 'sforum-shortcodes.user', contractVersion: 'sforum-shortcodes.user@1', arguments: { userId: 42 } } } },
      global: { stubs: { NodeViewWrapper, SFAvatar: AvatarStub, UIcon: IconStub } }
    })
    const category = mount(NodeView, {
      props: { ...baseProps, node: { attrs: { id: 'sforum-shortcodes.category', contractVersion: 'sforum-shortcodes.category@1', arguments: { categoryId: 7 } } } },
      global: { stubs: { NodeViewWrapper, SFAvatar: AvatarStub, UIcon: IconStub } }
    })
    try {
      await flushPromises()
      expect(user.find('[data-avatar-preview]').text()).toBe('Alice')
      expect(category.get('.sf-shortcode-node__label-mark').attributes('style') || '').toContain('color: #7c3aed')
      expect(category.text()).toContain('Design')
    } finally {
      user.unmount()
      category.unmount()
    }
  })
})

describe('protected shortcode NodeView and document rules', () => {
  test('renders protected block chrome without inline action buttons', async () => {
    const NodeViewWrapper = testVue.defineComponent({ template: '<div><slot /></div>' })
    Object.assign(globalThis, { NodeViewWrapper })
    const NodeView = await compileVueSfc(fileURLToPath(new URL(
      '../../app/components/editor/shortcodes/SFShortcodeProtectedNodeView.vue', import.meta.url
    )), 'shortcode-protected-node-view')
    const host = {
      disabled: () => false,
      editProtected: () => undefined,
      unwrapProtected: () => undefined,
      deleteProtected: () => undefined
    }
    const wrapper = mount(NodeView, {
      props: {
        node: { attrs: { id: 'sforum-shortcodes.login', contractVersion: 'sforum-shortcodes.login@1', arguments: {} } },
        extension: { options: { nodeViewHost: host } }, selected: true, getPos: () => 5,
        editor: {}, view: {}, decorations: [], innerDecorations: {}, HTMLAttributes: {}, updateAttributes: () => {}, deleteNode: () => {}
      },
      global: { stubs: { NodeViewWrapper, NodeViewContent: true, UIcon: true } }
    })
    try {
      expect(wrapper.text()).toContain('Log in')
      expect(wrapper.findAll('button')).toHaveLength(0)
      expect(wrapper.get('.sf-protected-node__label').text()).toContain('Log in')
    } finally {
      wrapper.unmount()
    }
  })

  test('hides protected block actions while the node is not selected', async () => {
    const NodeViewWrapper = testVue.defineComponent({ template: '<div><slot /></div>' })
    Object.assign(globalThis, { NodeViewWrapper })
    const NodeView = await compileVueSfc(fileURLToPath(new URL(
      '../../app/components/editor/shortcodes/SFShortcodeProtectedNodeView.vue', import.meta.url
    )), 'shortcode-protected-node-view-unselected')
    const wrapper = mount(NodeView, {
      props: {
        node: { attrs: { id: 'sforum-shortcodes.login', contractVersion: 'sforum-shortcodes.login@1', arguments: {} } },
        extension: { options: { nodeViewHost: { disabled: () => false } } }, selected: false, getPos: () => 5,
        editor: {}, view: {}, decorations: [], innerDecorations: {}, HTMLAttributes: {}, updateAttributes: () => {}, deleteNode: () => {}
      },
      global: { stubs: { NodeViewWrapper, NodeViewContent: true, UIcon: true } }
    })
    try {
      expect(wrapper.find('.sf-protected-node--selected').exists()).toBe(false)
      expect(wrapper.findAll('button')).toHaveLength(0)
    } finally {
      wrapper.unmount()
    }
  })

  test('rejects empty protected bodies and only-author on topics at the Host document boundary', () => {
    expect(inspectSForumShortcodeDocument({
      type: 'doc', content: [{ type: 'sforumShortcodeBlock', attrs: {
        id: 'sforum-shortcodes.login', contractVersion: 'sforum-shortcodes.login@1', arguments: {}
      }, content: [{ type: 'paragraph', content: [{ type: 'text', text: '   ' }] }] }]
    }, 'topic').valid).toBe(false)
    expect(inspectSForumShortcodeDocument({
      type: 'doc', content: [{ type: 'sforumShortcodeBlock', attrs: {
        id: 'sforum-shortcodes.only-author', contractVersion: 'sforum-shortcodes.only-author@1', arguments: {}
      }, content: [{ type: 'paragraph', content: [{ type: 'text', text: 'secret' }] }] }]
    }, 'topic').valid).toBe(false)
    expect(inspectSForumShortcodeDocument({
      type: 'doc', content: [{ type: 'sforumShortcodeBlock', attrs: {
        id: 'sforum-shortcodes.only-author', contractVersion: 'sforum-shortcodes.only-author@1', arguments: {}
      }, content: [{ type: 'paragraph', content: [{ type: 'text', text: 'secret' }] }] }]
    }, 'comment').valid).toBe(true)
  })
})
