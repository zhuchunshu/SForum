import { afterEach, describe, expect, test } from 'bun:test'
import { fileURLToPath } from 'node:url'
import { compileVueSfc, mount } from '../helpers/vueSfc'

async function compile(relative: string, id: string) {
  return compileVueSfc(fileURLToPath(new URL(`../../packages/plugin-ui/src/components/${relative}`, import.meta.url)), id)
}

const SPluginAlert = await compile('feedback/SPluginAlert.vue', 'plugin-ui-alert')
const SPluginButton = await compile('forms/SPluginButton.vue', 'plugin-ui-button')
const SPluginField = await compile('forms/SPluginField.vue', 'plugin-ui-field')
const SPluginInput = await compile('forms/SPluginInput.vue', 'plugin-ui-input')
const SPluginSelect = await compile('forms/SPluginSelect.vue', 'plugin-ui-select')
const SPluginSection = await compile('layout/SPluginSection.vue', 'plugin-ui-section')
const SPluginTable = await compile('data/SPluginTable.vue', 'plugin-ui-table')

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
