import { describe, expect, test } from 'bun:test'
import { readFileSync } from 'node:fs'

import {
  formatDurationMicros,
  parseRouteInspectorSnapshot,
  ROUTE_INSPECTOR_METHODS,
  routeInspectorErrorKind,
  routeInspectorMatchedStep,
  routeInspectorQueryFromRoute,
  routeInspectorQueryParams,
  useAdminRouteInspector,
  validateRouteInspectorLookup,
  type RouteInspectorSnapshot
} from '../app/composables/useAdminRouteInspector'

const source = (path: string) => readFileSync(new URL(path, import.meta.url), 'utf8')

const longRouteId = `plugin.${'very-long-extension-id-segment-'.repeat(6)}route.handler`
const longPath = `/api/v1/${'nested-segment/'.repeat(20)}resource/{id}`
const packageDigest = 'a'.repeat(64)

function coreStep(overrides: Record<string, unknown> = {}) {
  return {
    index: 0,
    phase: 'handler',
    action: 'add',
    routeId: 'core.route.topic.list',
    contractVersion: 'sforum.route.topic.list@1',
    method: 'GET',
    path: '/api/v1/topics',
    pathSignature: '/api/v1/topics',
    provider: { kind: 'core' },
    guard: 'core.guard.public',
    access: 'public',
    mode: 'http',
    fallback: '',
    timeoutMs: 0,
    priority: 0,
    ...overrides
  }
}

function validSnapshot(overrides: Record<string, unknown> = {}): Record<string, unknown> {
  return {
    revision: 3,
    safeMode: false,
    method: 'GET',
    resolution: 'resolved',
    chain: [coreStep()],
    provider: { status: 'not_required', live: { kind: 'core' } },
    conflicts: [],
    traces: [],
    ...overrides
  }
}

describe('route inspector lookup validation', () => {
  test('accepts concrete methods and absolute paths, strips query strings', () => {
    expect(validateRouteInspectorLookup({ method: 'get', path: '/api/v1/topics?token=secret' })).toEqual({
      ok: true,
      method: 'GET',
      path: '/api/v1/topics'
    })
    expect(ROUTE_INSPECTOR_METHODS).toContain('PATCH')
    expect(validateRouteInspectorLookup({ method: 'HEAD', path: longPath })).toMatchObject({
      ok: true,
      method: 'HEAD',
      path: longPath
    })
  })

  test('rejects empty, wildcard, relative, traversal, and unsafe path shapes', () => {
    expect(validateRouteInspectorLookup({ method: '', path: '' }).ok).toBe(false)
    expect(validateRouteInspectorLookup({ method: 'GET', path: '' })).toEqual({ ok: false, reason: 'empty' })
    expect(validateRouteInspectorLookup({ method: '*', path: '/api/v1/topics' })).toEqual({ ok: false, reason: 'method' })
    expect(validateRouteInspectorLookup({ method: 'TRACE', path: '/api/v1/topics' })).toEqual({ ok: false, reason: 'method' })
    expect(validateRouteInspectorLookup({ method: 'GET', path: 'relative' })).toEqual({ ok: false, reason: 'path' })
    expect(validateRouteInspectorLookup({ method: 'GET', path: '//evil' })).toEqual({ ok: false, reason: 'path' })
    expect(validateRouteInspectorLookup({ method: 'GET', path: '/api/../secret' })).toEqual({ ok: false, reason: 'path' })
    expect(validateRouteInspectorLookup({ method: 'GET', path: '/api\\topics' })).toEqual({ ok: false, reason: 'path' })
    expect(validateRouteInspectorLookup({ method: 'GET', path: '/api#frag' })).toEqual({ ok: false, reason: 'path' })
    expect(validateRouteInspectorLookup({ method: 'GET', path: '/api\u0000topics' })).toEqual({ ok: false, reason: 'path' })
  })

  test('syncs method and path query params from the route and normalized lookup', () => {
    expect(routeInspectorQueryFromRoute({ method: 'post', path: '/controller/topics' })).toEqual({
      method: 'post',
      path: '/controller/topics'
    })
    expect(routeInspectorQueryFromRoute({ method: ['PATCH'], path: ['/x'] })).toEqual({
      method: 'PATCH',
      path: '/x'
    })
    expect(routeInspectorQueryFromRoute(undefined)).toEqual({ method: '', path: '' })
    expect(routeInspectorQueryParams(' post ', ' /api/v1/topics?drop=1 ')).toEqual({
      method: 'POST',
      path: '/api/v1/topics'
    })
  })
})

describe('route inspector snapshot parser', () => {
  test('accepts omitted, empty, and bounded mutable field allowlists', () => {
    const maxTokens = `/${Array.from({ length: 32 }, () => 'a').join('/')}`
    const maxBytes = `/${'界'.repeat(85)}`
    const requestFields = [
      '/',
      '/body/title',
      '/meta/a~1b',
      '/~0private',
      maxTokens,
      maxBytes,
      ...Array.from({ length: 58 }, (_, index) => `/extra-${index}`)
    ]
    expect(requestFields).toHaveLength(64)
    expect(new TextEncoder().encode(maxBytes)).toHaveLength(256)

    const omitted = parseRouteInspectorSnapshot(validSnapshot())
    expect(omitted?.chain[0]?.mutableRequestFields).toBeUndefined()
    expect(omitted?.chain[0]?.mutableResponseFields).toBeUndefined()

    const empty = parseRouteInspectorSnapshot(validSnapshot({
      chain: [coreStep({ mutableRequestFields: [], mutableResponseFields: [] })]
    }))
    expect(empty?.chain[0]?.mutableRequestFields).toEqual([])
    expect(empty?.chain[0]?.mutableResponseFields).toEqual([])

    const parsed = parseRouteInspectorSnapshot(validSnapshot({
      chain: [coreStep({
        phase: 'filter',
        action: 'filter',
        mutableRequestFields: requestFields,
        mutableResponseFields: ['/status/code', '/payload/items/0']
      })]
    }))
    expect(parsed?.chain[0]?.mutableRequestFields).toEqual(requestFields)
    expect(parsed?.chain[0]?.mutableResponseFields).toEqual(['/status/code', '/payload/items/0'])
    requestFields[0] = '/changed-after-parse'
    expect(parsed?.chain[0]?.mutableRequestFields?.[0]).toBe('/')
  })

  test('rejects malformed or over-limit mutable field allowlists', () => {
    const invalidLists: Array<[string, unknown]> = [
      ['non-array', '/body/title'],
      ['null', null],
      ['root pointer', ['']],
      ['relative pointer', ['body/title']],
      ['trailing whitespace', ['/body/title ']],
      ['invalid escape', ['/body/~2title']],
      ['trailing escape', ['/body/title~']],
      ['duplicate', ['/body/title', '/body/title']],
      ['non-string member', ['/body/title', 7]],
      ['too many fields', Array.from({ length: 65 }, (_, index) => `/field-${index}`)],
      ['too many tokens', [`/${Array.from({ length: 33 }, () => 'field').join('/')}`]],
      ['too many UTF-8 bytes', [`/${'界'.repeat(85)}a`]]
    ]

    for (const [name, value] of invalidLists) {
      for (const field of ['mutableRequestFields', 'mutableResponseFields']) {
        expect(parseRouteInspectorSnapshot(validSnapshot({
          chain: [coreStep({ [field]: value })]
        })), `${name}: ${field}`).toBeNull()
      }
    }
  })

  test('rejects non-empty mutable field allowlists outside their action matrix', () => {
    const invalidActions: Array<[string, 'mutableRequestFields' | 'mutableResponseFields']> = [
      ['after', 'mutableRequestFields'],
      ['before', 'mutableResponseFields'],
      ['global_middleware', 'mutableResponseFields'],
      ['handler', 'mutableRequestFields'],
      ['handler', 'mutableResponseFields'],
      ['add', 'mutableRequestFields'],
      ['add', 'mutableResponseFields'],
      ['replace', 'mutableRequestFields'],
      ['replace', 'mutableResponseFields'],
      ['alias', 'mutableRequestFields'],
      ['alias', 'mutableResponseFields'],
      ['redirect', 'mutableRequestFields'],
      ['redirect', 'mutableResponseFields'],
      ['rewrite', 'mutableRequestFields'],
      ['rewrite', 'mutableResponseFields']
    ]

    for (const [action, field] of invalidActions) {
      expect(parseRouteInspectorSnapshot(validSnapshot({
        chain: [coreStep({ action, [field]: ['/body/title'] })]
      })), `${action}: ${field}`).toBeNull()
    }

    for (const action of ['global_middleware', 'before', 'filter', 'wrap']) {
      expect(parseRouteInspectorSnapshot(validSnapshot({
        chain: [coreStep({ action, mutableRequestFields: ['/body/title'] })]
      })), `${action}: mutableRequestFields`).not.toBeNull()
    }
    for (const action of ['filter', 'wrap', 'after']) {
      expect(parseRouteInspectorSnapshot(validSnapshot({
        chain: [coreStep({ action, mutableResponseFields: ['/payload/title'] })]
      })), `${action}: mutableResponseFields`).not.toBeNull()
    }

    const empty = parseRouteInspectorSnapshot(validSnapshot({
      chain: [coreStep({
        action: 'add',
        mutableRequestFields: [],
        mutableResponseFields: []
      })]
    }))
    expect(empty?.chain[0]?.mutableRequestFields).toEqual([])
    expect(empty?.chain[0]?.mutableResponseFields).toEqual([])
  })

  test('parses a resolved snapshot with chain, provider, conflict, and redacted traces', () => {
    const pluginStep = coreStep({
      index: 1,
      phase: 'handler',
      action: 'replace',
      routeId: longRouteId,
      contractVersion: 'demo.route.replace@1',
      targetRouteId: 'core.route.topic.create',
      method: 'POST',
      path: longPath,
      pathSignature: longPath,
      provider: {
        kind: 'plugin',
        artifact: {
          extensionId: 'demo.plugin',
          extensionVersion: '1.2.3',
          packageDigest,
          runtimeInstanceId: 'runtime-instance-very-long-id-0001'
        }
      },
      guard: 'demo.plugin.guard.owner',
      access: 'permission',
      permission: 'topic.create',
      pluginGuard: {
        id: 'demo.plugin.guard.owner',
        contractVersion: 'demo.guard.owner@1',
        kind: 'custom',
        entry: 'guards/owner',
        digest: packageDigest,
        permissions: ['topic.create']
      },
      handler: 'route.replace',
      requestSchema: 'demo.req@1',
      responseSchema: 'demo.res@1',
      mode: 'http',
      fallback: 'readonly_core',
      timeoutMs: 1500,
      priority: 10
    })

    const raw = validSnapshot({
      method: 'POST',
      safeMode: true,
      chain: [pluginStep],
      provider: {
        status: 'selected',
        live: pluginStep.provider,
        desired: {
          key: {
            targetRouteId: 'core.route.topic.create',
            targetContractVersion: 'sforum.route.topic.create@1',
            method: 'POST',
            pathSignature: '/api/v1/topics'
          },
          routeId: longRouteId,
          contractVersion: 'demo.route.replace@1',
          extensionId: 'demo.plugin',
          extensionVersionId: 9,
          extensionVersion: '1.2.3',
          packageDigest,
          selectedByUserId: 0,
          selectionAuditEventId: 44,
          revision: 2,
          selectedAt: '2026-07-16T01:00:00Z',
          updatedAt: '2026-07-16T02:00:00Z'
        }
      },
      conflicts: [{
        kind: 'provider_selection',
        routeId: 'core.route.topic.create',
        contractVersion: 'sforum.route.topic.create@1',
        method: 'POST',
        pathSignature: '/api/v1/topics',
        candidates: [coreStep({ action: 'add', method: 'POST' }), pluginStep],
        selectionKey: {
          targetRouteId: 'core.route.topic.create',
          targetContractVersion: 'sforum.route.topic.create@1',
          method: 'POST',
          pathSignature: '/api/v1/topics'
        },
        selectionStatus: 'selected'
      }],
      traces: [{
        sequence: 1,
        observedAt: '2026-07-16T03:00:00Z',
        revision: 3,
        stepIndex: 1,
        phase: 'handler',
        invocationStage: 'handler',
        action: 'replace',
        routeId: longRouteId,
        contractVersion: 'demo.route.replace@1',
        method: 'POST',
        pathSignature: longPath,
        mode: 'http',
        fallback: 'readonly_core',
        outcome: 'succeeded',
        durationMicros: 1250,
        commitState: 'committed',
        provider: pluginStep.provider
      }]
    })

    const snapshot = parseRouteInspectorSnapshot(raw)
    expect(snapshot).not.toBeNull()
    expect(snapshot?.safeMode).toBe(true)
    expect(snapshot?.chain[0]?.routeId).toBe(longRouteId)
    expect(snapshot?.chain[0]?.path).toBe(longPath)
    expect(snapshot?.chain[0]?.pluginGuard?.kind).toBe('custom')
    expect(snapshot?.provider.desired?.selectedByUserId).toBe(0)
    expect(snapshot?.conflicts[0]?.candidates).toHaveLength(2)
    expect(snapshot?.traces[0]?.durationMicros).toBe(1250)
    expect(snapshot?.traces[0]?.invocationStage).toBe('handler')
    expect(JSON.stringify(snapshot)).not.toContain('token=')
    expect(JSON.stringify(snapshot)).not.toContain('Authorization')
  })

  test('rejects invented fields, incomplete artifacts, short conflict candidate lists, and oversized traces', () => {
    expect(parseRouteInspectorSnapshot(null)).toBeNull()
    expect(parseRouteInspectorSnapshot({ ...validSnapshot(), resolution: 'maybe' })).toBeNull()
    expect(parseRouteInspectorSnapshot({ ...validSnapshot(), revision: 0 })).toBeNull()
    expect(parseRouteInspectorSnapshot({
      ...validSnapshot(),
      chain: [coreStep({ provider: { kind: 'plugin', artifact: { extensionId: 'x' } } })]
    })).toBeNull()
    expect(parseRouteInspectorSnapshot({
      ...validSnapshot(),
      chain: [coreStep({ provider: { kind: 'plugin' } })]
    })).toBeNull()
    expect(parseRouteInspectorSnapshot({
      ...validSnapshot(),
      chain: [coreStep({ provider: {
        kind: 'core',
        artifact: {
          extensionId: 'forged.core',
          extensionVersion: '1.0.0',
          packageDigest,
          runtimeInstanceId: 'forged-runtime'
        }
      } })]
    })).toBeNull()
    expect(parseRouteInspectorSnapshot({
      ...validSnapshot(),
      chain: [coreStep({ phase: 'middleware' })]
    })).toBeNull()
    expect(parseRouteInspectorSnapshot({
      ...validSnapshot(),
      conflicts: [{
        kind: 'path_method',
        method: 'GET',
        pathSignature: '/x',
        candidates: [coreStep()]
      }]
    })).toBeNull()
    expect(parseRouteInspectorSnapshot({
      ...validSnapshot(),
      provider: {
        status: 'selected',
        desired: {
          key: {
            targetRouteId: 'core.route.topic.create',
            targetContractVersion: 'sforum.route.topic.create@1',
            method: 'POST',
            pathSignature: '/api/v1/topics'
          },
          routeId: 'demo.route',
          contractVersion: 'demo@1',
          extensionId: 'demo',
          extensionVersionId: 0,
          extensionVersion: '1.0.0',
          packageDigest,
          selectedByUserId: 1,
          selectionAuditEventId: 1,
          revision: 1,
          selectedAt: '2026-07-16T01:00:00Z',
          updatedAt: '2026-07-16T01:00:00Z'
        }
      }
    })).toBeNull()
    expect(parseRouteInspectorSnapshot({
      ...validSnapshot(),
      traces: Array.from({ length: 51 }, (_, index) => ({
        sequence: index + 1,
        observedAt: '2026-07-16T03:00:00Z',
        revision: 3,
        stepIndex: 0,
        phase: 'handler',
        invocationStage: 'handler',
        action: 'add',
        routeId: 'core.route.topic.list',
        contractVersion: 'sforum.route.topic.list@1',
        method: 'GET',
        pathSignature: '/api/v1/topics',
        mode: 'http',
        fallback: '',
        outcome: 'succeeded',
        durationMicros: 1,
        commitState: 'committed',
        provider: { kind: 'core' }
      }))
    })).toBeNull()
    // 快照不得接受请求体 / 查询串等敏感字段伪装成合法 trace。
    expect(parseRouteInspectorSnapshot({
      ...validSnapshot(),
      traces: [{
        sequence: 1,
        observedAt: '2026-07-16T03:00:00Z',
        revision: 3,
        stepIndex: 0,
        phase: 'handler',
        invocationStage: 'handler',
        action: 'add',
        routeId: 'core.route.topic.list',
        contractVersion: 'sforum.route.topic.list@1',
        method: 'GET',
        pathSignature: '/api/v1/topics',
        mode: 'http',
        fallback: '',
        outcome: 'leaked',
        durationMicros: 1,
        commitState: 'committed',
        provider: { kind: 'core' },
        requestBody: 'secret'
      }]
    })).toBeNull()
    for (const invocationStage of [undefined, '', 'forged']) {
      expect(parseRouteInspectorSnapshot({
        ...validSnapshot(),
        traces: [{
          sequence: 1,
          observedAt: '2026-07-16T03:00:00Z',
          revision: 3,
          stepIndex: 0,
          phase: 'handler',
          invocationStage,
          action: 'add',
          routeId: 'core.route.topic.list',
          contractVersion: 'sforum.route.topic.list@1',
          method: 'GET',
          pathSignature: '/api/v1/topics',
          mode: 'http',
          fallback: '',
          outcome: 'succeeded',
          durationMicros: 1,
          commitState: 'committed',
          provider: { kind: 'core' }
        }]
      })).toBeNull()
    }
  })

  test('matched step is only the terminal chain entry for resolved snapshots', () => {
    const resolved = parseRouteInspectorSnapshot(validSnapshot({
      chain: [coreStep({ index: 0 }), coreStep({ index: 1, routeId: 'core.route.topic.list.terminal' })]
    })) as RouteInspectorSnapshot
    expect(routeInspectorMatchedStep(resolved)?.routeId).toBe('core.route.topic.list.terminal')

    const notFound = parseRouteInspectorSnapshot(validSnapshot({
      resolution: 'not_found',
      chain: [],
      provider: { status: 'not_required' }
    })) as RouteInspectorSnapshot
    expect(routeInspectorMatchedStep(notFound)).toBeUndefined()
  })
})

describe('route inspector formatting and errors', () => {
  test('formats duration micros for dense UI labels', () => {
    expect(formatDurationMicros(-1)).toBe('—')
    expect(formatDurationMicros(420)).toBe('420 µs')
    expect(formatDurationMicros(1250)).toBe('1.25 ms')
    expect(formatDurationMicros(12_500)).toBe('12.5 ms')
    expect(formatDurationMicros(2_500_000)).toBe('2.50 s')
  })

  test('classifies API error reasons and HTTP status fallbacks', () => {
    expect(routeInspectorErrorKind({
      data: { code: 403, message: 'denied', data: { reason: 'permission.denied' } }
    })).toBe('permission')
    expect(routeInspectorErrorKind({
      data: { code: 422, message: 'invalid', data: { reason: 'extensions.route_inspector_invalid' } }
    })).toBe('validation')
    expect(routeInspectorErrorKind({
      data: { code: 503, message: 'down', data: { reason: 'extensions.route_inspector_unavailable' } }
    })).toBe('unavailable')
    expect(routeInspectorErrorKind({
      data: { code: 409, message: 'cas', data: { reason: 'extensions.route_provider_conflict' } }
    })).toBe('conflict')
    expect(routeInspectorErrorKind({ statusCode: 403 })).toBe('permission')
    expect(routeInspectorErrorKind({ statusCode: 422 })).toBe('validation')
    expect(routeInspectorErrorKind({ statusCode: 409 })).toBe('conflict')
    expect(routeInspectorErrorKind({ statusCode: 503 })).toBe('unavailable')
    expect(routeInspectorErrorKind({ statusCode: 500 })).toBe('generic')
  })
})

describe('route inspector inspect request', () => {
  test('calls the exact GET endpoint with method and path query, then parses the detached snapshot', async () => {
    const calls: Array<{ path: string, options?: unknown }> = []
    ;(globalThis as typeof globalThis & { useApiClient: () => unknown }).useApiClient = () => ({
      request: async (path: string, options?: unknown) => {
        calls.push({ path, options })
        return validSnapshot({ method: 'POST', chain: [coreStep({ method: 'POST', action: 'add' })] })
      }
    })

    const api = useAdminRouteInspector()
    const snapshot = await api.inspect('post', '/controller/topics?token=secret')
    expect(calls).toHaveLength(1)
    expect(calls[0]?.path).toBe('/admin/extensions/route-inspector?method=POST&path=%2Fcontroller%2Ftopics')
    expect(calls[0]?.options).toBeUndefined()
    expect(snapshot.method).toBe('POST')
    expect(snapshot.resolution).toBe('resolved')
  })

  test('fails closed on invalid lookup and unparseable responses without mutating providers', async () => {
    const calls: string[] = []
    ;(globalThis as typeof globalThis & { useApiClient: () => unknown }).useApiClient = () => ({
      request: async (path: string) => {
        calls.push(path)
        return { revision: 1 }
      }
    })

    const api = useAdminRouteInspector()
    await expect(api.inspect('*', '/api/v1/topics')).rejects.toMatchObject({
      data: { data: { reason: 'extensions.route_inspector_invalid' } }
    })
    expect(calls).toEqual([])

    await expect(api.inspect('GET', '/api/v1/topics')).rejects.toMatchObject({
      data: { data: { reason: 'extensions.route_inspector_unavailable' } }
    })
    expect(calls).toEqual(['/admin/extensions/route-inspector?method=GET&path=%2Fapi%2Fv1%2Ftopics'])
  })
})

describe('route inspector page, nav, and i18n contracts', () => {
  const page = source('../app/pages/admin/extensions/route-inspector.vue')
  const composable = source('../app/composables/useAdminRouteInspector.ts')
  const modules = source('../app/config/adminModules.ts')
  const plugins = source('../app/pages/admin/extensions/plugins.vue')
  const en = source('../i18n/locales/en-US.json')
  const zh = source('../i18n/locales/zh-CN.json')

  test('registers a read-only extension.view admin page and plugin workbench link', () => {
    expect(modules).toContain("id: '/extensions/route-inspector'")
    expect(modules).toContain("labelKey: 'admin.nav.extensionRouteInspector'")
    expect(modules).toContain("componentName: 'AdminExtensionRouteInspector'")
    expect(modules).toContain("requiredPermissions: ['extension.view']")
    expect(modules).toContain("pageId: '/extensions/route-inspector'")
    expect(plugins).toContain("adminRoutes.path('/extensions/route-inspector')")
    expect(plugins).toContain('admin.nav.extensionRouteInspector')
    expect(plugins).toContain('i-lucide-scan-search')

    expect(page).toContain("name: 'AdminExtensionRouteInspector'")
    expect(page).toContain("useAdminPage('/extensions/route-inspector')")
    expect(page).toContain('onInspectClick')
    expect(page).toContain('hydrateFromQuery')
    expect(page).toContain('syncQuery')
    expect(page).toContain('routeInspectorQueryParams')
    expect(page).toContain("data-testid=\"route-inspector-submit\"")
    expect(page).toContain('route-inspector-matched-request-fields')
    expect(page).toContain('route-inspector-matched-response-fields')
    expect(page).toContain('route-inspector-step-${step.index}-request-fields')
    expect(page).toContain('route-inspector-step-${step.index}-response-fields')
    expect(page).toContain('matched.mutableRequestFields?.length')
    expect(page).toContain('step.mutableResponseFields?.length')
    expect(page).toContain('labelInvocationStage(trace.invocationStage)')
    expect(page).toContain('admin.extensions.routeInspector.readOnlyHint')
    expect(page).not.toContain('selectProvider')
    expect(page).not.toContain('resetProvider')
    expect(page).not.toContain('expectedRevision')
    expect(composable).toContain("const basePath = '/admin/extensions/route-inspector'")
    expect(composable).toContain('return { inspect }')
    expect(composable).not.toContain('route-providers/selection')
    expect(composable).not.toContain('selectProvider')
    expect(composable).not.toContain('resetProvider')
  })

  test('covers empty, error, safe-mode, not-found, ambiguous, and stale UI states', () => {
    expect(page).toContain('route-inspector-first-use')
    expect(page).toContain('route-inspector-validation')
    expect(page).toContain('route-inspector-error')
    expect(page).toContain('route-inspector-loading')
    expect(page).toContain('route-inspector-safe-mode')
    expect(page).toContain('route-inspector-not-found')
    expect(page).toContain('route-inspector-ambiguous')
    expect(page).toContain('route-inspector-stale')
    expect(page).toContain('route-inspector-traces-empty')
    expect(page).toContain('safeModeTitle')
    expect(page).toContain('notFoundTitle')
    expect(page).toContain('ambiguousTitle')
    expect(page).toContain('staleTitle')
  })

  test('keeps a dense mobile-safe layout without nested cards or emoji icons', () => {
    expect(page).toContain('min-w-0 w-full max-w-full')
    expect(page).toContain('break-all')
    expect(page).toContain('overflow-x-auto')
    expect(page).toContain('sm:flex-row')
    expect(page).toContain('sm:grid-cols-2')
    expect(page).toContain('sm:w-36')
    expect(page).toContain('w-full shrink-0 sm:w-auto')
    expect(page).not.toContain('<UCard')
    expect(page).not.toContain('nested card')
    expect(page).toContain('i-lucide-search')
    expect(page).toContain('i-lucide-info')
    expect(page).not.toMatch(/[\u{1F300}-\u{1FAFF}]/u)
    expect(page).toContain('font-mono')
  })

  test('ships matching en-US and zh-CN routeInspector copy for UI keys', () => {
    for (const locale of [en, zh]) {
      expect(locale).toContain('"extensionRouteInspector"')
      expect(locale).toContain('"routeInspector"')
      expect(locale).toContain('"readOnlyHint"')
      expect(locale).toContain('"firstUseTitle"')
      expect(locale).toContain('"safeModeTitle"')
      expect(locale).toContain('"notFoundTitle"')
      expect(locale).toContain('"ambiguousTitle"')
      expect(locale).toContain('"staleTitle"')
      expect(locale).toContain('"provider_selection"')
      expect(locale).toContain('"schema_rejected"')
      expect(locale).toContain('"side_effect_started"')
      expect(locale).toContain('"invocationStage"')
      expect(locale).toContain('"request"')
      expect(locale).toContain('"response"')
      expect(locale).toContain('"packageDigest"')
      expect(locale).toContain('"declaredPath"')
      expect(locale).toContain('"mutableRequestFields"')
      expect(locale).toContain('"mutableResponseFields"')
    }
    expect(en).toContain('never selects, resets, or mutates route providers')
    expect(zh).toContain('不会选择、重置或变更路由提供者')
  })
})
