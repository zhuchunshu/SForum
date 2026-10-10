import { describe, expect, test } from 'bun:test'

import { readFileSync } from 'node:fs'
import { fileURLToPath } from 'node:url'

import { modelOptionsForSelect } from '../../app/utils/admin/adminAI'

function source(relative: string): string {
  return readFileSync(fileURLToPath(new URL(relative, import.meta.url)), 'utf8')
}

function locale(name: string): Record<string, unknown> {
  return JSON.parse(source(`../../i18n/locales/${name}.json`)) as Record<string, unknown>
}

function lookup(catalog: Record<string, unknown>, key: string): unknown {
  return key.split('.').reduce<unknown>((node, part) => {
    if (!node || typeof node !== 'object') return undefined
    return (node as Record<string, unknown>)[part]
  }, catalog)
}

const modelField = source('../../app/components/admin/ai/blocks/SFAdminAIModelField.vue')
const providersTab = source('../../app/components/admin/ai/tabs/SFAdminAIProvidersTab.vue')

describe('admin AI model options', () => {
  test('keeps the provider order and drops duplicates and blanks', () => {
    const { options } = modelOptionsForSelect(['deepseek-reasoner', ' deepseek-chat ', '', 'deepseek-reasoner'], '')
    expect(options.map(option => option.value)).toEqual(['deepseek-reasoner', 'deepseek-chat'])
  })

  test('keeps a current model that the provider no longer lists selectable', () => {
    // 供应商可能已下线站点正在使用的模型：把它过滤掉，下拉框就会显示「没选中」，
    // 而配置里明明写着一个模型。
    const { options, currentListed } = modelOptionsForSelect(['deepseek-chat'], 'retired-model')
    expect(currentListed).toBe(false)
    expect(options.map(option => option.value)).toEqual(['deepseek-chat', 'retired-model'])
  })

  test('reports a current model that is present in the list', () => {
    const { options, currentListed } = modelOptionsForSelect(['a', 'b'], 'b')
    expect(currentListed).toBe(true)
    expect(options).toHaveLength(2)
  })

  test('handles empty input without inventing an option', () => {
    expect(modelOptionsForSelect([], '').options).toEqual([])
    expect(modelOptionsForSelect([], '   ').currentListed).toBe(false)
  })
})

describe('admin AI model field wiring', () => {
  test('the providers tab mounts the model field with the probe inputs', () => {
    expect(providersTab).toContain('SFAdminAIModelField')
    expect(providersTab).toContain('v-model="selectedProfile.model"')
    expect(providersTab).toContain(':base-url="selectedProfile.baseUrl"')
    // 未保存的密钥也要能用于探测：探测参数取自当前表单，而不是已保存状态。
    expect(providersTab).toContain(':api-key="credentials[selectedProfile.id] || \'\'"')
    expect(providersTab).toContain(':credential-configured=')
  })

  test('the field switches between a select and free text', () => {
    expect(modelField).toContain('<USelect')
    expect(modelField).toContain('<UInput')
    expect(modelField).toContain("t('admin.ai.modelManual')")
    expect(modelField).toContain("t('admin.ai.modelUseList')")
  })

  test('never probes without an address and a credential', () => {
    // 缺参数时按钮必须禁用并说明缺什么，而不是点下去才失败。
    expect(modelField).toContain("const missing = computed(() => {")
    expect(modelField).toContain("return 'baseUrl'")
    expect(modelField).toContain("return 'credential'")
    expect(modelField).toContain(':disabled="!canFetch"')
  })

  test('keeps the previous profile list out of the current profile', () => {
    expect(modelField).toContain('fetched.value.profileId === props.profileId')
  })

  test('explains a provider-side failure next to the field', () => {
    expect(modelField).toContain('admin.ai.modelFetchReasons.')
    expect(modelField).toContain('t(\'admin.ai.modelFetchFailed\')')
  })
})

describe('admin AI model field localization', () => {
  const keys = [...modelField.matchAll(/t\('(admin\.ai\.[A-Za-z0-9_.]+)'/g)].map(match => match[1])
  const zh = locale('zh-CN')
  const en = locale('en-US')

  test('the component references translatable keys', () => {
    expect(keys.length).toBeGreaterThan(5)
  })

  test('every referenced key exists in both locales', () => {
    for (const key of new Set(keys)) {
      expect(`${key}:${typeof lookup(zh, key)}`).toBe(`${key}:string`)
      expect(`${key}:${typeof lookup(en, key)}`).toBe(`${key}:string`)
    }
  })

  test('localizes every provider failure reason the backend can return', () => {
    // 出站守卫拒绝（unsafe_target）与供应商失败（provider_failed）是两种结论：
    // 前者要改地址。缺文案时界面会退回原因码，运营者就读不懂该改什么。
    for (const reason of ['provider_failed', 'unsafe_target']) {
      expect(lookup(zh, `admin.ai.modelFetchReasons.${reason}`)).toBeTruthy()
      expect(lookup(en, `admin.ai.modelFetchReasons.${reason}`)).toBeTruthy()
      // 诊断页与模型列表共用同一套原因码。
      expect(lookup(zh, `admin.ai.diagnose.reasons.${reason}`)).toBeTruthy()
      expect(lookup(en, `admin.ai.diagnose.reasons.${reason}`)).toBeTruthy()
    }
  })
})
