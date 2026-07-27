import { describe, expect, test } from 'bun:test'

import {
  EXTENSION_EVENT_PAGE_SIZE,
  activeTheme,
  canRestartPlugin,
  capabilityCount,
  extensionContributionLabel,
  extensionContributionPage,
  extensionContributionPayloadSummary,
  extensionAdminPages,
  extensionDeliveryPage,
  extensionDefinitionPage,
  extensionEventPage,
  extensionAuthorName,
  extensionAuthorWebsite,
  extensionDisplayDescription,
  extensionDisplayName,
  extensionLocalizedDisplay,
  extensionManageRoute,
  extensionSettingDeclarations,
  extensionStats,
  filterExtensionsByType,
  findExtensionAdminPage,
  formatPluginMemoryBytes,
  isExtensionArtifactAvailable,
  mergeExtensionDeliveries,
  mergeExtensionEvents,
  missingArtifactCleanupCandidates,
  recommendedExtensionSettingValues,
  runtimeCapabilitySummary,
  runtimeStatusLabelKey,
  themeActionState,
  themeStatusLabelKey,
  type AdminEffectiveContribution,
  type AdminExtension,
  type AdminExtensionEvent,
  type AdminExtensionEventDefinition,
  type AdminExtensionEventDelivery
} from '../../app/utils/admin/adminExtensions'

const baseExtension = {
  version: '1.0.0',
  status: 'installed',
  packagePath: 'storage/extensions/demo',
  installedAt: '2026-07-05T10:00:00Z',
  updatedAt: '2026-07-05T10:00:00Z',
  manifest: {
    sforumVersion: '^1.0.0'
  }
} satisfies Partial<AdminExtension>

describe('admin extension helpers', () => {
  test('restores recommended extension settings without clearing secrets', () => {
    expect(recommendedExtensionSettingValues([
      { key: 'port', label: 'Port', type: 'number', default: '25', recommendedValue: '587', value: '465' },
      { key: 'password', label: 'Password', type: 'secret', default: '', value: '', secretSet: true }
    ])).toEqual({ port: '587' })
  })
  test('filters plugins and themes by manifest type', () => {
    const items = [
      extension({ id: 'demo.plugin', name: 'Demo Plugin', type: 'plugin' }),
      extension({ id: 'demo.theme', name: 'Demo Theme', type: 'theme' })
    ]

    expect(filterExtensionsByType(items, 'plugin').map(item => item.id)).toEqual(['demo.plugin'])
    expect(filterExtensionsByType(items, 'theme').map(item => item.id)).toEqual(['demo.theme'])
  })

  test('counts installed plugins separately from active themes', () => {
    const items = [
      extension({ id: 'one.plugin', name: 'One Plugin', type: 'plugin', status: 'enabled' }),
      extension({ id: 'two.plugin', name: 'Two Plugin', type: 'plugin', status: 'disabled' }),
      extension({ id: 'one.theme', name: 'One Theme', type: 'theme', status: 'enabled' })
    ]

    expect(extensionStats(items)).toEqual({
      pluginCount: 2,
      themeCount: 1,
      enabledPluginCount: 1,
      activeThemeId: 'one.theme'
    })
  })

  test('identifies active theme and theme-specific UI state', () => {
    const items = [
      extension({ id: 'sforum.default-theme', name: 'SForum Default Theme', type: 'theme', status: 'enabled' }),
      extension({ id: 'uploaded.theme', name: 'Uploaded Theme', type: 'theme', status: 'installed' })
    ]
    const inactiveDefaultTheme = extension({
      id: 'sforum.default-theme',
      name: 'SForum Default Theme',
      type: 'theme',
      status: 'disabled',
      source: 'builtin'
    })

    expect(activeTheme(items)?.id).toBe('sforum.default-theme')
    expect(themeStatusLabelKey(items[0])).toBe('admin.extensions.status.activeTheme')
    expect(themeStatusLabelKey(items[1])).toBe('admin.extensions.status.installed')
    expect(themeActionState(items[0])).toBe('active')
    expect(themeActionState(inactiveDefaultTheme)).toBe('activateDefault')
    expect(themeActionState(items[1])).toBe('activate')
  })

  test('counts manifest capability declarations', () => {
    const item = extension({
      id: 'capable.plugin',
      name: 'Capable Plugin',
      type: 'plugin',
      manifest: {
        permissions: ['forum.topic.pin'],
        settings: [{ key: 'enabled', label: 'Enabled', type: 'boolean' }],
        migrations: [{ path: 'migrations/001.sql' }],
        adminPages: [{ path: '/demo', label: 'Demo' }],
        routes: [{ path: '/demo' }, { path: '/demo/:id' }],
        hooks: [{ name: 'topic.created' }],
        events: [{ name: 'topic.before_create', kind: 'filter' }],
        jobs: [{ name: 'demo.sync' }]
      }
    })

    expect(capabilityCount(item)).toBe(9)
  })

  test('resolves v2 admin pages, manage route, and legacy adminPages fallback', () => {
    const item = extension({
      id: 'admin.plugin',
      name: 'Admin Plugin',
      type: 'plugin',
      manifest: {
        admin: {
          entry: '/settings',
          pages: [
            { path: '/settings', label: 'Settings', view: 'settings' },
            { path: '/dashboard', label: 'Dashboard', view: 'about', menu: true, icon: 'i-lucide-layout-dashboard', order: 20 }
          ]
        },
        adminPages: [{ path: '/legacy', label: 'Legacy' }]
      }
    })
    const legacy = extension({
      id: 'legacy.plugin',
      name: 'Legacy Plugin',
      type: 'plugin',
      manifest: {
        adminPages: [{ path: '/settings', label: 'Legacy Settings', view: 'settings', menu: true }]
      }
    })
    const generatedOnly = extension({
      id: 'plain.plugin',
      name: 'Plain Plugin',
      type: 'plugin'
    })

    expect(extensionAdminPages(item).map(page => page.path)).toEqual(['/about', '/settings', '/dashboard'])
    expect(findExtensionAdminPage(item, 'dashboard')?.label).toBe('Dashboard')
    expect(extensionManageRoute(item)).toBe('/extensions/admin.plugin/pages/settings')
    expect(extensionManageRoute(legacy)).toBe('/extensions/legacy.plugin/pages/settings')
    expect(extensionManageRoute(generatedOnly)).toBe('/extensions/plain.plugin/pages/about')
  })

  test('resolves extension author display and website fallback', () => {
    const item = extension({
      id: 'author.plugin',
      name: 'Author Plugin',
      type: 'plugin',
      manifest: {
        url: 'https://example.com/plugins/author',
        author: {
          name: 'Demo Studio',
          url: 'https://studio.example.com'
        }
      }
    })
    const fallback = extension({
      id: 'fallback.theme',
      name: 'Fallback Theme',
      type: 'theme',
      manifest: {
        url: 'https://example.com/themes/fallback',
        author: {
          name: 'Theme Team'
        }
      }
    })

    expect(extensionAuthorName(item)).toBe('Demo Studio')
    expect(extensionAuthorWebsite(item)).toBe('https://studio.example.com')
    expect(extensionAuthorWebsite(fallback)).toBe('https://example.com/themes/fallback')
  })

  test('uses optional langs for display and falls back to top-level english defaults', () => {
    const localized = extension({
      id: 'lang.plugin',
      name: 'Language Plugin',
      type: 'plugin',
      manifest: {
        name: 'Language Plugin',
        description: 'English description.',
        url: 'https://example.com/plugins/lang',
        author: {
          name: 'Demo Studio',
          url: 'https://studio.example.com'
        },
        langs: {
          zh: {
            name: '语言插件',
            description: '中文说明。',
            author: {
              name: '演示工作室'
            }
          }
        }
      }
    })
    const plain = extension({
      id: 'plain.plugin',
      name: 'Plain Plugin',
      type: 'plugin',
      manifest: {
        name: 'Plain Plugin',
        description: 'No langs field.',
        url: 'https://example.com/plugins/plain',
        author: {
          name: 'Plain Studio'
        }
      }
    })

    expect(extensionDisplayName(localized, 'zh-CN')).toBe('语言插件')
    expect(extensionDisplayDescription(localized, 'zh-CN')).toBe('中文说明。')
    expect(extensionAuthorName(localized, 'zh-CN')).toBe('演示工作室')
    expect(extensionAuthorWebsite(localized, 'zh-CN')).toBe('https://studio.example.com')
    expect(extensionLocalizedDisplay(localized, 'en-US')).toMatchObject({
      name: 'Language Plugin',
      description: 'English description.'
    })
    expect(extensionDisplayName(plain, 'zh-CN')).toBe('Plain Plugin')
    expect(extensionDisplayDescription(plain, 'zh-CN')).toBe('No langs field.')
    // 兼容模板里误传 Ref 形态的 locale。
    expect(extensionDisplayName(localized, { value: 'zh-CN' })).toBe('语言插件')
    expect(extensionDisplayDescription(localized, { value: 'en' })).toBe('English description.')
  })

  test('summarizes runtime declarations and running state', () => {
    const item = extension({
      id: 'runtime.plugin',
      name: 'Runtime Plugin',
      type: 'plugin',
      status: 'enabled',
      manifest: {
        routes: [{ path: '/hello', methods: ['GET'], access: 'public' }],
        hooks: [{ name: 'extension.enabled' }],
        events: [{ name: 'topic.created', kind: 'observe' }],
        providers: [{ slot: 'search.provider', label: 'Demo Search' }]
      },
      runtime: {
        state: 'running',
        routeCount: 1,
        hookCount: 1,
        eventCount: 2,
        providerCount: 1
      }
    })

    expect(runtimeStatusLabelKey(item)).toBe('admin.extensions.runtime.running')
    expect(runtimeCapabilitySummary(item)).toEqual({ routes: 1, hooks: 1, events: 2, providers: 1 })
    expect(canRestartPlugin(item)).toBe(true)
  })

  test('blocks runtime and settings helpers for a missing artifact', () => {
    const item = extension({
      id: 'missing.plugin',
      name: 'Missing Plugin',
      type: 'plugin',
      status: 'enabled',
      artifactState: 'missing',
      manifest: {
        settings: [{ key: 'enabled', label: 'Enabled', type: 'boolean' }]
      },
      runtime: {
        state: 'running',
        routeCount: 0,
        hookCount: 0,
        providerCount: 0
      }
    })

    expect(isExtensionArtifactAvailable(item)).toBe(false)
    expect(canRestartPlugin(item)).toBe(false)
    expect(extensionSettingDeclarations([item])).toEqual([])
  })

  test('allows restart recovery for a disabled plugin with an exact staged artifact', () => {
    const item = extension({
      id: 'recoverable.plugin',
      name: 'Recoverable Plugin',
      type: 'plugin',
      status: 'disabled',
      stagedVersion: {
        id: 2,
        version: '2.0.0',
        packageDigest: 'b'.repeat(64),
        packagePath: 'storage/extensions/recoverable/2.0.0',
        installedAt: '2026-07-28T00:00:00Z',
        manifest: {
          id: 'recoverable.plugin',
          name: 'Recoverable Plugin',
          version: '2.0.0',
          type: 'plugin',
          sforumVersion: '^1.0.0'
        }
      },
      runtime: {
        state: 'stopped',
        routeCount: 0,
        hookCount: 0,
        providerCount: 0
      }
    })

    expect(canRestartPlugin(item)).toBe(true)
    expect(canRestartPlugin({ ...item, stagedVersion: undefined })).toBe(false)
  })

  test('selects only deletable disabled uploads whose artifacts are missing', () => {
    const eligible = {
      ...extension({ id: 'missing.plugin', name: 'Missing Plugin', type: 'plugin', status: 'disabled' }),
      artifactState: 'missing' as const,
      source: 'uploaded' as const,
      isSystem: false,
      isDeletable: true
    }
    const items: AdminExtension[] = [
      eligible,
      { ...eligible, id: 'available.plugin', artifactState: 'available' },
      { ...eligible, id: 'enabled.plugin', status: 'enabled' },
      { ...eligible, id: 'builtin.plugin', source: 'builtin' },
      { ...eligible, id: 'system.plugin', isSystem: true },
      { ...eligible, id: 'protected.plugin', isDeletable: false }
    ]

    expect(missingArtifactCleanupCandidates(items).map(item => item.id)).toEqual(['missing.plugin'])
  })

  test('formats plugin process RSS for admin lists', () => {
    expect(formatPluginMemoryBytes(512 * 1024)).toBe('512 KiB')
    expect(formatPluginMemoryBytes(18 * 1024 * 1024)).toBe('18 MiB')
    expect(formatPluginMemoryBytes(128 * 1024 * 1024)).toBe('128 MiB')
  })

  test('formats extension contributions for admin inspection', () => {
    const items: AdminEffectiveContribution[] = [
      contribution({ extensionId: 'beta.plugin', id: 'beta.action', order: 200, label: { 'en-US': 'Beta' } }),
      contribution({ extensionId: 'alpha.plugin', id: 'alpha.action', order: 100, label: { 'zh-CN': '甲' } })
    ]

    const page = extensionContributionPage(items, 1, 1)

    expect(page).toMatchObject({
      page: 1,
      totalPages: 2,
      start: 1,
      end: 1,
      total: 2
    })
    expect(page.items.map(item => item.id)).toEqual(['beta.action'])
    expect(extensionContributionLabel(items[1], 'zh-CN')).toBe('甲')
    expect(extensionContributionLabel(items[0], 'zh-CN')).toBe('Beta')
    expect(extensionContributionPayloadSummary(items[0])).toBe('extensionRoute POST /topic-actions/beta')
  })

  test('merges extension event lists and sorts by newest first', () => {
    const events: Record<string, AdminExtensionEvent[]> = {
      'demo.plugin': [
        event({ id: 1, extensionId: 'demo.plugin', action: 'installed', createdAt: '2026-07-05T10:00:00Z' }),
        event({ id: 2, extensionId: 'demo.plugin', action: 'enabled', createdAt: '2026-07-05T12:00:00Z' })
      ],
      'demo.theme': [
        event({ id: 3, extensionId: 'demo.theme', action: 'installed', createdAt: '2026-07-05T11:00:00Z' })
      ]
    }

    expect(mergeExtensionEvents(events).map(item => `${item.extensionId}:${item.action}`)).toEqual([
      'demo.plugin:enabled',
      'demo.theme:installed',
      'demo.plugin:installed'
    ])
  })

  test('paginates extension events with at most eight rows per page', () => {
    const items = Array.from({ length: 18 }, (_, index) => event({
      id: index + 1,
      extensionId: 'demo.plugin',
      action: `event.${index + 1}`,
      createdAt: '2026-07-05T10:00:00Z'
    }))

    const firstPage = extensionEventPage(items, 1)
    const lastPage = extensionEventPage(items, 99)
    const emptyPage = extensionEventPage([], 1)

    expect(EXTENSION_EVENT_PAGE_SIZE).toBe(8)
    expect(firstPage.items).toHaveLength(8)
    expect(firstPage.start).toBe(1)
    expect(firstPage.end).toBe(8)
    expect(firstPage.totalPages).toBe(3)
    expect(lastPage.page).toBe(3)
    expect(lastPage.items.map(item => item.id)).toEqual([17, 18])
    expect(emptyPage.totalPages).toBe(1)
    expect(emptyPage.start).toBe(0)
    expect(emptyPage.end).toBe(0)
  })

  test('paginateItems is reusable for page registry lists', async () => {
    const { paginateItems } = await import('../../app/utils/admin/adminExtensions')
    const items = Array.from({ length: 10 }, (_, index) => ({ id: index + 1 }))
    const page = paginateItems(items, 2)
    expect(page.items.map(item => item.id)).toEqual([9, 10])
    expect(page.totalPages).toBe(2)
    expect(page.start).toBe(9)
    expect(page.end).toBe(10)
  })

  test('merges and paginates extension event deliveries', () => {
    const items = [
      delivery({ id: 1, extensionId: 'demo.plugin', eventName: 'topic.created', createdAt: '2026-07-05T10:00:00Z' }),
      delivery({ id: 2, extensionId: 'demo.plugin', eventName: 'topic.created', createdAt: '2026-07-05T12:00:00Z' }),
      delivery({ id: 3, extensionId: 'demo.plugin', eventName: 'user.registered', createdAt: '2026-07-05T11:00:00Z' })
    ]

    const merged = mergeExtensionDeliveries(items)
    const page = extensionDeliveryPage(merged, 1, 2)

    expect(merged.map(item => item.id)).toEqual([2, 3, 1])
    expect(page.items.map(item => item.id)).toEqual([2, 3])
    expect(page.totalPages).toBe(2)
  })

  test('paginates extension event definitions', () => {
    const items = Array.from({ length: 19 }, (_, index) => definition(`event.${index + 1}`))

    const firstPage = extensionDefinitionPage(items, 1)
    const lastPage = extensionDefinitionPage(items, 99)

    expect(firstPage.items).toHaveLength(8)
    expect(firstPage.start).toBe(1)
    expect(firstPage.end).toBe(8)
    expect(firstPage.totalPages).toBe(3)
    expect(lastPage.page).toBe(3)
    expect(lastPage.items.map(item => item.name)).toEqual(['event.17', 'event.18', 'event.19'])
  })
})

function extension(input: {
  id: string
  name: string
  type: 'plugin' | 'theme'
  status?: 'installed' | 'enabled' | 'disabled'
  artifactState?: AdminExtension['artifactState']
  source?: AdminExtension['source']
  stagedVersion?: AdminExtension['stagedVersion']
  manifest?: Partial<AdminExtension['manifest']>
  runtime?: Partial<NonNullable<AdminExtension['runtime']>>
}): AdminExtension {
  return {
    ...baseExtension,
    id: input.id,
    name: input.name,
    type: input.type,
    status: input.status || 'installed',
    artifactState: input.artifactState,
    source: input.source,
    stagedVersion: input.stagedVersion,
    manifest: {
      id: input.id,
      name: input.name,
      version: '1.0.0',
      type: input.type,
      sforumVersion: '^1.0.0',
      ...input.manifest
    },
    runtime: input.runtime as AdminExtension['runtime']
  } as AdminExtension
}

function contribution(overrides: Partial<AdminEffectiveContribution>): AdminEffectiveContribution {
  return {
    extensionId: 'demo.plugin',
    extensionName: 'Demo Plugin',
    extensionType: 'plugin',
    point: 'forum.topic.actions',
    id: 'demo.action',
    order: 100,
    label: { 'zh-CN': '动作', 'en-US': 'Action' },
    icon: 'i-lucide-bookmark',
    payload: {
      type: 'extensionRoute',
      method: 'POST',
      path: `/topic-actions/${overrides.id?.split('.')[0] || 'demo'}`
    },
    ...overrides
  }
}

function event(input: {
  id: number
  extensionId: string
  action: string
  createdAt: string
}): AdminExtensionEvent {
  return {
    actorUserId: 1,
    message: '',
    ...input
  }
}

function delivery(input: {
  id: number
  extensionId: string
  eventName: string
  createdAt: string
}): AdminExtensionEventDelivery {
  return {
    eventKind: 'observe',
    status: 'succeeded',
    reason: '',
    message: '',
    correlationId: `corr-${input.id}`,
    attemptCount: 1,
    updatedAt: input.createdAt,
    ...input
  }
}

function definition(name: string): AdminExtensionEventDefinition {
  return {
    name,
    kind: 'observe',
    description: `${name} description`,
    payloadFields: ['extensionId'],
    patchFields: [],
    timeoutMs: 5000
  }
}
