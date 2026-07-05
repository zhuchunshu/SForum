import { describe, expect, test } from 'bun:test'

import {
  activeTheme,
  capabilityCount,
  extensionStats,
  filterExtensionsByType,
  themeActionState,
  themeStatusLabelKey,
  mergeExtensionEvents,
  type AdminExtension,
  type AdminExtensionEvent
} from '../app/utils/adminExtensions'

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
    expect(themeActionState(items[1])).toBe('verifyOnly')
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
        jobs: [{ name: 'demo.sync' }]
      }
    })

    expect(capabilityCount(item)).toBe(8)
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
})

function extension(input: {
  id: string
  name: string
  type: 'plugin' | 'theme'
  status?: 'installed' | 'enabled' | 'disabled'
  source?: AdminExtension['source']
  manifest?: Partial<AdminExtension['manifest']>
}): AdminExtension {
  return {
    ...baseExtension,
    id: input.id,
    name: input.name,
    type: input.type,
    status: input.status || 'installed',
    source: input.source,
    manifest: {
      id: input.id,
      name: input.name,
      version: '1.0.0',
      type: input.type,
      sforumVersion: '^1.0.0',
      ...input.manifest
    }
  } as AdminExtension
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
