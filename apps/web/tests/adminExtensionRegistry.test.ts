import { describe, expect, test } from 'bun:test'

import {
  ADMIN_EXTENSION_SLOT_CATALOG,
  loaderKey,
  lookupAdminComponentLoader,
  sortAdminComponentMetadata,
  translateAdminExtensionMessage
} from '../app/runtime/admin-extensions/catalog'
import {
  clearContributionFailures,
  contributionFailureKey,
  recordContributionFailure
} from '../app/runtime/admin-extensions/quarantine'
import { extensionRequestPath } from '../app/runtime/admin-extensions/types'

describe('admin extension registry', () => {
  test('publishes the Jobs and extension settings component slots', () => {
    expect(Object.keys(ADMIN_EXTENSION_SLOT_CATALOG).sort()).toEqual([
      'admin.extension.settings.footer',
      'admin.extension.settings.header',
      'admin.extension.settings.page',
      'admin.jobs.detail.sections',
      'admin.jobs.row.actions',
      'admin.jobs.table.columns'
    ])
  })

  test('sorts metadata by order, extension, then contribution', () => {
    const input = [
      metadata('z.plugin', 'b', 10),
      metadata('a.plugin', 'z', 10),
      metadata('a.plugin', 'a', 10),
      metadata('z.plugin', 'a', 5)
    ]

    expect(sortAdminComponentMetadata(input).map(item => `${item.extensionId}:${item.contributionId}`)).toEqual([
      'z.plugin:a',
      'a.plugin:a',
      'a.plugin:z',
      'z.plugin:b'
    ])
  })

  test('looks up only the exact lazy loader', async () => {
    const expected = { default: { name: 'ExpectedComponent' } }
    const registry = {
      [loaderKey('demo.plugin', 'latency')]: async () => expected
    }

    expect(await lookupAdminComponentLoader(registry, 'demo.plugin', 'latency')?.()).toBe(expected)
    expect(lookupAdminComponentLoader(registry, 'other.plugin', 'latency')).toBeUndefined()
  })

  test('maps en to en-US and keeps translations inside the owner namespace', () => {
    const messages = {
      'demo.plugin': {
        'zh-CN': { action: { run: '\u8fd0\u884c' } },
        'en-US': { action: { run: 'Run' } }
      },
      'other.plugin': {
        'en-US': { action: { run: 'Wrong owner' } }
      }
    }

    expect(translateAdminExtensionMessage(messages, 'demo.plugin', 'en', 'action.run')).toBe('Run')
    expect(translateAdminExtensionMessage(messages, 'demo.plugin', 'zh-CN', 'action.run')).toBe('\u8fd0\u884c')
    expect(translateAdminExtensionMessage(messages, 'demo.plugin', 'en', 'missing.key')).toBe('missing.key')
  })

  // 主题/插件设置项 id 常含点号，locale 用字面量键 fields["home.notice.zh-CN"]，
  // 若按 '.' 硬拆会得到 fields.home.notice... 的假路径。
  test('resolves locale keys whose object keys contain dots (settings field ids)', () => {
    const messages = {
      'sforum.default-theme': {
        'zh-CN': {
          fields: {
            'home.notice.zh-CN': {
              label: '首页提示条（中文）',
              description: '显示在首页话题列表上方。留空则隐藏。'
            },
            host: {
              label: '嵌套写法仍可用'
            }
          }
        },
        'en-US': {
          fields: {
            'home.notice.zh-CN': {
              label: 'Homepage notice (Chinese)'
            }
          }
        }
      }
    }

    expect(translateAdminExtensionMessage(
      messages,
      'sforum.default-theme',
      'zh-CN',
      'fields.home.notice.zh-CN.label'
    )).toBe('首页提示条（中文）')
    expect(translateAdminExtensionMessage(
      messages,
      'sforum.default-theme',
      'zh-CN',
      'fields.home.notice.zh-CN.description'
    )).toBe('显示在首页话题列表上方。留空则隐藏。')
    expect(translateAdminExtensionMessage(
      messages,
      'sforum.default-theme',
      'en',
      'fields.home.notice.zh-CN.label'
    )).toBe('Homepage notice (Chinese)')
    // 纯嵌套路径仍兼容（SMTP 的 fields.host.label）。
    expect(translateAdminExtensionMessage(
      messages,
      'sforum.default-theme',
      'zh-CN',
      'fields.host.label'
    )).toBe('嵌套写法仍可用')
    expect(translateAdminExtensionMessage(
      messages,
      'sforum.default-theme',
      'zh-CN',
      'fields.home.notice.zh-CN.missing'
    )).toBe('fields.home.notice.zh-CN.missing')
  })

  test('prefixes requests with the owning extension route', () => {
    expect(extensionRequestPath('demo.plugin', '/jobs/42')).toBe('/extensions/demo.plugin/jobs/42')
    expect(extensionRequestPath('demo/plugin', 'jobs')).toBe('/extensions/demo%2Fplugin/jobs')
    expect(() => extensionRequestPath('demo.plugin', 'https://example.com')).toThrow()
    expect(() => extensionRequestPath('demo.plugin', '../admin/users')).toThrow()
    expect(() => extensionRequestPath('demo.plugin', '%2e%2e/admin/users')).toThrow()
  })

  test('quarantines the third failure per release, extension, and contribution', () => {
    const storage = memoryStorage()
    const key = contributionFailureKey('release-1', 'demo.plugin', 'latency')

    expect(recordContributionFailure(storage, key)).toEqual({ count: 1, quarantined: false })
    expect(recordContributionFailure(storage, key)).toEqual({ count: 2, quarantined: false })
    expect(recordContributionFailure(storage, key)).toEqual({ count: 3, quarantined: true })
    expect(recordContributionFailure(storage, contributionFailureKey('release-2', 'demo.plugin', 'latency'))).toEqual({ count: 1, quarantined: false })

    clearContributionFailures(storage, key)
    expect(recordContributionFailure(storage, key)).toEqual({ count: 1, quarantined: false })
  })
})

function metadata(extensionId: string, contributionId: string, order: number) {
  return {
    point: 'admin.test.fixture',
    extensionId,
    contributionId,
    componentId: contributionId,
    order,
    label: {},
    options: {}
  }
}

function memoryStorage(): Storage {
  const values = new Map<string, string>()
  return {
    get length() { return values.size },
    clear: () => values.clear(),
    getItem: key => values.get(key) ?? null,
    key: index => [...values.keys()][index] ?? null,
    removeItem: key => values.delete(key),
    setItem: (key, value) => values.set(key, value)
  }
}
