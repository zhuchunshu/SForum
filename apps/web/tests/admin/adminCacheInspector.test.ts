import { describe, expect, test } from 'bun:test'
import { readFileSync } from 'node:fs'

import {
  cacheInspectorErrorKind,
  formatCacheDuration,
  parseCacheInspectorSnapshot,
  useAdminCacheInspector
} from '../../app/composables/admin/useAdminCacheInspector'

const source = (path: string) => readFileSync(new URL(path, import.meta.url), 'utf8')
const digest = 'a'.repeat(64)

function artifact(overrides: Record<string, unknown> = {}) {
  return {
    extensionId: 'demo.cache',
    extensionVersion: '1.2.3',
    packageDigest: digest,
    versionId: 7,
    runtimeInstanceId: 'runtime-demo-cache-1',
    ...overrides
  }
}

function declaration(overrides: Record<string, unknown> = {}) {
  return {
    id: 'demo.cache.topics',
    contractVersion: 'demo.cache.topics@1',
    namespace: 'demo.cache.topics',
    policy: 'permission',
    provider: 'core.redis',
    invalidators: ['demo.cache.topic.changed'],
    ...overrides
  }
}

function metrics(overrides: Record<string, unknown> = {}) {
  return {
    samples: 4,
    hits: 1,
    misses: 1,
    allowed: 1,
    denied: 1,
    stale: 0,
    conflicts: 0,
    errors: 0,
    canceled: 0,
    deadlines: 0,
    slow: 1,
    affected: 2,
    totalDurationMicros: 1000,
    averageDurationMicros: 250,
    p95DurationMicros: 500,
    ...overrides
  }
}

function trace(overrides: Record<string, unknown> = {}) {
  return {
    sequence: 8,
    extensionId: 'demo.cache',
    extensionVersion: '1.2.3',
    artifactDigest: digest,
    runtimeInstanceId: 'runtime-demo-cache-1',
    versionId: 7,
    cacheId: 'demo.cache.topics',
    contractVersion: 'demo.cache.topics@1',
    registryRevision: 3,
    registryCurrent: true,
    providerRevision: 2,
    providerId: 'core.redis',
    operation: 'get',
    tagDigest: 'b'.repeat(64),
    tagCount: 2,
    outcome: 'hit',
    durationMicros: 420,
    attempts: 1,
    hit: true,
    ...overrides
  }
}

function snapshot(overrides: Record<string, unknown> = {}) {
  return {
    schemaVersion: 'sforum.cache-inspector@1',
    registry: {
      schemaVersion: 'sforum.cache-registry@1',
      revision: 3,
      digest,
      safeMode: false,
      publications: [{ artifact: artifact(), caches: [declaration()] }],
      caches: [{ ...declaration(), artifact: artifact() }]
    },
    retainedFromSequence: 4,
    retainedThroughSequence: 8,
    metrics: metrics(),
    operations: [metrics({ operation: 'get' })],
    traces: [trace()],
    invalidations: [],
    ...overrides
  }
}

describe('cache inspector snapshot parser', () => {
  test('parses the exact redacted registry, metrics, traces, and retention window', () => {
    const parsed = parseCacheInspectorSnapshot(snapshot())
    expect(parsed).not.toBeNull()
    expect(parsed?.registry.revision).toBe(3)
    expect(parsed?.registry.caches[0]?.artifact.runtimeInstanceId).toBe('runtime-demo-cache-1')
    expect(parsed?.operations[0]?.p95DurationMicros).toBe(500)
    expect(parsed?.traces[0]?.registryCurrent).toBe(true)
    expect(parsed?.retainedFromSequence).toBe(4)
    expect(JSON.stringify(parsed)).not.toContain('topic:secret')
  })

  test('accepts an empty registry and omitempty trace fields', () => {
    const raw = snapshot({
      registry: {
        schemaVersion: 'sforum.cache-registry@1',
        revision: 0,
        digest,
        publications: [],
        caches: []
      },
      retainedFromSequence: undefined,
      retainedThroughSequence: undefined,
      metrics: metrics({ samples: 0, hits: 0, misses: 0, allowed: 0, denied: 0, slow: 0,
        affected: 0, totalDurationMicros: 0, averageDurationMicros: 0, p95DurationMicros: 0 }),
      operations: [],
      traces: [],
      invalidations: []
    })
    const parsed = parseCacheInspectorSnapshot(raw)
    expect(parsed?.registry.safeMode).toBe(false)
    expect(parsed?.retainedFromSequence).toBe(0)
  })

  test('rejects tag names, cache material, forged shapes, and impossible retention windows', () => {
    const withTags = snapshot()
    const publication = (withTags.registry as { publications: Array<{ caches: Array<Record<string, unknown>> }> })
      .publications[0]!
    publication.caches[0]!.tags = ['topic:secret']
    expect(parseCacheInspectorSnapshot(withTags)).toBeNull()

    expect(parseCacheInspectorSnapshot(snapshot({ traces: [trace({ key: 'private-key' })] }))).toBeNull()
    expect(parseCacheInspectorSnapshot(snapshot({ traces: [trace({ value: 'private-value' })] }))).toBeNull()
    expect(parseCacheInspectorSnapshot(snapshot({ traces: [trace({ lockToken: 'secret' })] }))).toBeNull()
    expect(parseCacheInspectorSnapshot(snapshot({ schemaVersion: 'sforum.cache-inspector@2' }))).toBeNull()
    expect(parseCacheInspectorSnapshot(snapshot({ retainedFromSequence: 9, retainedThroughSequence: 8 }))).toBeNull()
    expect(parseCacheInspectorSnapshot(snapshot({ operations: [metrics({ samples: -1 })] }))).toBeNull()
    expect(parseCacheInspectorSnapshot(snapshot({ traces: [trace({ outcome: 'executed' })] }))).toBeNull()
  })
})

describe('cache inspector formatting and API boundary', () => {
  test('formats dense duration values and classifies typed errors', () => {
    expect(formatCacheDuration(-1)).toBe('—')
    expect(formatCacheDuration(420)).toBe('420 µs')
    expect(formatCacheDuration(1250)).toBe('1.25 ms')
    expect(formatCacheDuration(2_500_000)).toBe('2.50 s')
    expect(cacheInspectorErrorKind({
      data: { code: 409, message: 'changed', data: { reason: 'extensions.cache_inspector_conflict' } }
    })).toBe('conflict')
    expect(cacheInspectorErrorKind({ statusCode: 403 })).toBe('permission')
    expect(cacheInspectorErrorKind({ statusCode: 422 })).toBe('validation')
    expect(cacheInspectorErrorKind({ statusCode: 503 })).toBe('unavailable')
    expect(cacheInspectorErrorKind({ statusCode: 500 })).toBe('generic')
  })

  test('calls the exact read-only endpoint and rejects invalid limits or response shapes', async () => {
    const calls: Array<{ path: string, options?: unknown }> = []
    ;(globalThis as typeof globalThis & { useApiClient: () => unknown }).useApiClient = () => ({
      request: async (path: string, options?: unknown) => {
        calls.push({ path, options })
        return snapshot()
      }
    })
    const api = useAdminCacheInspector()
    expect((await api.inspect(50)).registry.revision).toBe(3)
    expect(calls).toEqual([{ path: '/admin/extensions/cache-inspector?limit=50', options: undefined }])
    await expect(api.inspect(0)).rejects.toMatchObject({
      data: { data: { reason: 'extensions.cache_inspector_invalid' } }
    })
    expect(calls).toHaveLength(1)

    ;(globalThis as typeof globalThis & { useApiClient: () => unknown }).useApiClient = () => ({
      request: async () => ({ schemaVersion: 'forged' })
    })
    await expect(useAdminCacheInspector().inspect()).rejects.toMatchObject({
      data: { data: { reason: 'extensions.cache_inspector_unavailable' } }
    })
  })
})

describe('cache inspector page and navigation contracts', () => {
  const page = source('../../app/pages/admin/extensions/cache-inspector.vue')
  const modules = source('../../app/config/adminModules.ts')
  const plugins = source('../../app/pages/admin/extensions/plugins.vue')
  const en = source('../../i18n/locales/en-US.json')
  const zh = source('../../i18n/locales/zh-CN.json')

  test('registers the extension.view read-only operations page', () => {
    expect(modules).toContain("id: '/extensions/cache-inspector'")
    expect(modules).toContain("componentName: 'AdminExtensionCacheInspector'")
    expect(modules).toContain("pageId: '/extensions/cache-inspector'")
    expect(plugins).toContain("adminRoutes.path('/extensions/cache-inspector')")
    expect(page).toContain("name: 'AdminExtensionCacheInspector'")
    expect(page).toContain("useAdminPage('/extensions/cache-inspector')")
    expect(page).toContain('cache-inspector-loading')
    expect(page).toContain('cache-inspector-error')
    expect(page).toContain('cache-inspector-safe-mode')
    expect(page).toContain('cache-inspector-empty-traces')
    expect(page).not.toContain('selectProvider')
    expect(page).not.toContain('cache key')
    expect(page).not.toContain('<UCard')
  })

  test('ships matching navigation and UI copy without emoji', () => {
    for (const locale of [en, zh]) {
      expect(locale).toContain('"extensionCacheInspector"')
      expect(locale).toContain('"cacheInspector"')
      expect(locale).toContain('"redactionHint"')
      expect(locale).toContain('"retention"')
    }
    expect(page).not.toMatch(/[\u{1F300}-\u{1FAFF}]/u)
    expect(page).toContain('i-lucide-database-zap')
    expect(page).toContain('overflow-x-auto')
    expect(page).toContain('break-all')
  })
})
