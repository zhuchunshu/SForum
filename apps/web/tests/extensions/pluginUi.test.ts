import { afterEach, describe, expect, test } from 'bun:test'
import { fileURLToPath } from 'node:url'
import { h, provide } from 'vue'

import { compileVueSfc, flushPromises, mount } from '../helpers/vueSfc'
import {
  PLUGIN_ICON_NAME_PATTERN,
  PLUGIN_ICON_RESOLVER,
  type PluginIconResolver
} from '../../packages/plugin-ui/src/icons/resolver'

const SPluginIcon = await compileVueSfc(
  fileURLToPath(new URL('../../packages/plugin-ui/src/components/icons/SPluginIcon.vue', import.meta.url)),
  'plugin-ui-icon',
  { PLUGIN_ICON_NAME_PATTERN, PLUGIN_ICON_RESOLVER }
)

async function compile(relative: string, id: string) {
  return compileVueSfc(
    fileURLToPath(new URL(`../../packages/plugin-ui/src/components/${relative}`, import.meta.url)),
    id,
    { SPluginIcon }
  )
}

const SPluginAlert = await compile('feedback/SPluginAlert.vue', 'plugin-ui-alert')
const SPluginButton = await compile('forms/SPluginButton.vue', 'plugin-ui-button')
const SPluginField = await compile('forms/SPluginField.vue', 'plugin-ui-field')
const SPluginInput = await compile('forms/SPluginInput.vue', 'plugin-ui-input')
const SPluginSelect = await compile('forms/SPluginSelect.vue', 'plugin-ui-select')
const SPluginSection = await compile('layout/SPluginSection.vue', 'plugin-ui-section')
const SPluginTable = await compile('data/SPluginTable.vue', 'plugin-ui-table')

/** 提供注入键的透明包装组件，用于验证图标解析器的依赖注入路径。 */
function withIconResolver(resolver: PluginIconResolver | undefined) {
  return {
    setup(_props: unknown, { slots }: { slots: { default?: () => unknown } }) {
      if (resolver) {
        provide(PLUGIN_ICON_RESOLVER, resolver)
      }
      return () => h('div', {}, slots.default ? slots.default() as never : [])
    }
  }
}

function iconResolver(icons: Record<string, string>) {
  const calls: string[][] = []
  const resolver: PluginIconResolver = {
    resolve: async (names) => {
      calls.push([...names])
      const result: Record<string, string> = {}
      for (const name of names) {
        if (icons[name]) result[name] = icons[name]
      }
      return result
    }
  }
  return { resolver, calls }
}

const CHECK_SVG = '<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24"><path d="M0 0h4"/></svg>'

const wrappers: Array<{ unmount: () => void }> = []

afterEach(() => {
  while (wrappers.length) wrappers.pop()?.unmount()
})

describe('@sforum/plugin-ui', () => {
  test('renders compact sections and named action slots', () => {
    const wrapper = mount(SPluginSection, {
      props: { title: 'Queue', description: 'Recent jobs' },
      slots: { default: '<p>Body</p>', actions: '<button>Refresh</button>' }
    })
    wrappers.push(wrapper)
    expect(wrapper.get('h2').text()).toBe('Queue')
    expect(wrapper.get('.splugin-section__description').text()).toBe('Recent jobs')
    expect(wrapper.get('.splugin-section__actions button').text()).toBe('Refresh')
  })

  test('supports loading and disabled button states', async () => {
    const wrapper = mount(SPluginButton, {
      props: { loading: true },
      slots: { default: 'Save' }
    })
    wrappers.push(wrapper)
    const button = wrapper.get('button')
    expect(button.attributes('disabled')).toBeDefined()
    expect(button.attributes('aria-busy')).toBe('true')
    await button.trigger('click')
    expect(wrapper.emitted('click')).toBeUndefined()
  })

  test('emits input and select model updates', async () => {
    const input = mount(SPluginInput, { props: { modelValue: '' } })
    wrappers.push(input)
    await input.get('input').setValue('updated')
    expect(input.emitted('update:modelValue')?.[0]).toEqual(['updated'])

    const select = mount(SPluginSelect, {
      props: {
        modelValue: '',
        options: [{ value: 'recommended', label: 'Recommended' }],
        placeholder: 'Choose'
      }
    })
    wrappers.push(select)
    await select.get('select').setValue('recommended')
    expect(select.emitted('update:modelValue')?.[0]).toEqual(['recommended'])
  })

  test('keeps field errors next to the affected control', () => {
    const wrapper = mount(SPluginField, {
      props: { label: 'Endpoint', for: 'endpoint', error: 'Endpoint is required', required: true },
      slots: { default: '<input id="endpoint">' }
    })
    wrappers.push(wrapper)
    expect(wrapper.get('label').attributes('for')).toBe('endpoint')
    expect(wrapper.get('[role="alert"]').text()).toBe('Endpoint is required')
    expect(wrapper.classes()).toContain('splugin-field--invalid')
  })

  test('renders table values, custom cells, and empty content', () => {
    const wrapper = mount(SPluginTable, {
      props: {
        columns: [{ key: 'name', label: 'Name' }, { key: 'status', label: 'Status', align: 'end' }],
        rows: [{ id: 7, name: 'Build', status: 'Ready' }]
      },
      slots: { 'cell-status': ({ value }: { value: unknown }) => `State: ${value}` }
    })
    wrappers.push(wrapper)
    expect(wrapper.findAll('tbody td').map(cell => cell.text())).toEqual(['Build', 'State: Ready'])
    expect(wrapper.findAll('th')[1]?.classes()).toContain('splugin-table__cell--end')

    const empty = mount(SPluginTable, {
      props: { columns: [{ key: 'name', label: 'Name' }], rows: [] },
      slots: { empty: 'Nothing here' }
    })
    wrappers.push(empty)
    expect(empty.get('.splugin-table__message').text()).toBe('Nothing here')
  })

  test('uses alert semantics for blocking failures', () => {
    const wrapper = mount(SPluginAlert, {
      props: { tone: 'danger', title: 'Request failed' },
      slots: { default: 'Try again.' }
    })
    wrappers.push(wrapper)
    expect(wrapper.attributes('role')).toBe('alert')
    expect(wrapper.text()).toContain('Try again.')
  })
})

describe('@sforum/plugin-ui host icons', () => {
  test('renders the resolved host icon inside a section title', async () => {
    const { resolver, calls } = iconResolver({ 'i-lucide-inbox': CHECK_SVG })
    const wrapper = mount(withIconResolver(resolver), {
      slots: {
        default: () => h(SPluginSection, { title: 'Queue', icon: 'i-lucide-inbox' })
      }
    })
    wrappers.push(wrapper)
    await flushPromises()

    expect(calls).toEqual([['i-lucide-inbox']])
    expect(wrapper.get('.splugin-section__title .splugin-icon').html()).toContain('<path d="M0 0h4"')
    expect(wrapper.get('.splugin-section__title').text()).toContain('Queue')
  })

  test('never asks the host for an icon it was not given', async () => {
    const { resolver, calls } = iconResolver({})
    const wrapper = mount(withIconResolver(resolver), {
      slots: { default: () => h(SPluginSection, { title: 'Queue' }) }
    })
    wrappers.push(wrapper)
    await flushPromises()

    expect(calls).toEqual([])
    expect(wrapper.find('.splugin-icon').exists()).toBe(false)
  })

  test('renders button and table header icons', async () => {
    const { resolver } = iconResolver({ 'i-lucide-play': CHECK_SVG, 'i-lucide-hash': CHECK_SVG })
    const wrapper = mount(withIconResolver(resolver), {
      slots: {
        default: () => [
          h(SPluginButton, { icon: 'i-lucide-play' }, () => 'Run'),
          h(SPluginTable, {
            columns: [{ key: 'id', label: 'ID', icon: 'i-lucide-hash' }],
            rows: [{ id: 1 }]
          })
        ]
      }
    })
    wrappers.push(wrapper)
    await flushPromises()

    expect(wrapper.get('.splugin-button__icon').html()).toContain('<path d="M0 0h4"')
    expect(wrapper.get('.splugin-table__head-icon').html()).toContain('<path d="M0 0h4"')
  })

  test('degrades to an empty placeholder when the host provides no resolver', async () => {
    const warnings: string[] = []
    const original = console.warn
    console.warn = (message: string) => { warnings.push(String(message)) }
    try {
      const wrapper = mount(SPluginSection, { props: { title: 'Queue', icon: 'i-lucide-inbox' } })
      wrappers.push(wrapper)
      await flushPromises()

      const icon = wrapper.get('.splugin-icon')
      expect(icon.html()).not.toContain('<svg')
      expect(icon.attributes('aria-hidden')).toBe('true')
      expect(warnings.filter((line) => line.includes('no host icon resolver'))).toHaveLength(1)
    } finally {
      console.warn = original
    }
  })

  test('ignores names outside the host grammar without asking the host', async () => {
    const { resolver, calls } = iconResolver({})
    const warnings: string[] = []
    const original = console.warn
    console.warn = (message: string) => { warnings.push(String(message)) }
    try {
      const wrapper = mount(withIconResolver(resolver), {
        slots: {
          default: () => [
            h(SPluginSection, { title: 'A', icon: 'i-mdi-home' }),
            h(SPluginSection, { title: 'B', icon: 'i-mdi-home' })
          ]
        }
      })
      wrappers.push(wrapper)
      await flushPromises()

      expect(calls).toEqual([])
      expect(warnings.filter((line) => line.includes('unsupported icon name'))).toHaveLength(2)
      expect(wrapper.findAll('.splugin-icon').every((icon) => !icon.html().includes('<svg'))).toBe(true)
    } finally {
      console.warn = original
    }
  })

  test('renders an empty placeholder when the host cannot resolve a valid name', async () => {
    const { resolver, calls } = iconResolver({})
    const wrapper = mount(withIconResolver(resolver), {
      slots: { default: () => h(SPluginSection, { title: 'Queue', icon: 'i-lucide-missing' }) }
    })
    wrappers.push(wrapper)
    await flushPromises()

    expect(calls).toEqual([['i-lucide-missing']])
    expect(wrapper.get('.splugin-icon').html()).not.toContain('<svg')
  })

  test('marks a labelled icon as an image and a bare icon as decorative', async () => {
    const { resolver } = iconResolver({ 'i-lucide-inbox': CHECK_SVG })
    const wrapper = mount(withIconResolver(resolver), {
      slots: {
        default: () => [
          h(SPluginIcon, { name: 'i-lucide-inbox', label: 'Inbox' }),
          h(SPluginIcon, { name: 'i-lucide-inbox' })
        ]
      }
    })
    wrappers.push(wrapper)
    await flushPromises()

    const [labelled, decorative] = wrapper.findAll('.splugin-icon')
    expect(labelled?.attributes('role')).toBe('img')
    expect(labelled?.attributes('aria-label')).toBe('Inbox')
    expect(decorative?.attributes('aria-hidden')).toBe('true')
  })
})
