import { describe, expect, test } from 'bun:test'
import { existsSync, readFileSync } from 'node:fs'
import { ref } from 'vue'

describe('guest middleware', () => {
  test('redirects after an unknown session refresh resolves an authenticated user', async () => {
    const user = ref<null | { id: string }>(null)
    const status = ref('unknown')
    const actions = createActions()
    const middleware = loadGuestMiddleware({
      useAuthSession: () => ({
        user,
        status,
        refresh: async (options: unknown) => {
          actions.refreshes.push(options)
          user.value = { id: 'user-1' }
          status.value = 'authenticated'
        }
      }),
      returnFromAuth: async (explicitRedirect) => {
        actions.returns += 1
        actions.redirects.push(explicitRedirect)
        return { type: 'redirect', target: explicitRedirect }
      }
    })

    const result = await middleware({ query: { redirect: '/settings/security' } })

    expect(actions.refreshes).toEqual([{ timeout: 800 }])
    expect(actions.returns).toBe(1)
    expect(actions.redirects).toEqual(['/settings/security'])
    expect(result).toEqual({ type: 'redirect', target: '/settings/security' })
  })

  test('allows a guest session to continue to the auth page', async () => {
    const actions = createActions()
    const middleware = loadGuestMiddleware({
      useAuthSession: () => ({
        user: ref(null),
        status: ref('guest'),
        refresh: async (options: unknown) => actions.refreshes.push(options)
      }),
      returnFromAuth: async () => {
        actions.returns += 1
      }
    })

    expect(await middleware({ query: {} })).toBeUndefined()
    expect(actions.refreshes).toEqual([])
    expect(actions.returns).toBe(0)
  })

  test('allows an unavailable session to continue to the auth page', async () => {
    const user = ref(null)
    const status = ref('unknown')
    const actions = createActions()
    const middleware = loadGuestMiddleware({
      useAuthSession: () => ({
        user,
        status,
        refresh: async (options: unknown) => {
          actions.refreshes.push(options)
          status.value = 'unavailable'
        }
      }),
      returnFromAuth: async () => {
        actions.returns += 1
      }
    })

    expect(await middleware({ query: {} })).toBeUndefined()
    expect(actions.refreshes).toEqual([{ timeout: 800 }])
    expect(actions.returns).toBe(0)
  })

  test('redirects an existing user without refreshing the session', async () => {
    const actions = createActions()
    const middleware = loadGuestMiddleware({
      useAuthSession: () => ({
        user: ref({ id: 'user-1' }),
        status: ref('authenticated'),
        refresh: async (options: unknown) => actions.refreshes.push(options)
      }),
      returnFromAuth: async (explicitRedirect) => {
        actions.returns += 1
        actions.redirects.push(explicitRedirect)
        return { type: 'redirect', target: explicitRedirect }
      }
    })

    expect(await middleware({ query: { redirect: '/settings/security' } })).toEqual({
      type: 'redirect',
      target: '/settings/security'
    })
    expect(actions.refreshes).toEqual([])
    expect(actions.returns).toBe(1)
    expect(actions.redirects).toEqual(['/settings/security'])
  })

  test('does not reuse a stale current-route redirect when the incoming route has none', async () => {
    const actions = createActions()
    const middleware = loadGuestMiddleware({
      useAuthSession: () => ({
        user: ref({ id: 'user-1' }),
        status: ref('authenticated'),
        refresh: async (options: unknown) => actions.refreshes.push(options)
      }),
      returnFromAuth: async (explicitRedirect) => {
        actions.redirects.push(explicitRedirect)
        return { type: 'redirect', target: explicitRedirect }
      }
    })

    expect(await middleware({ query: {} })).toEqual({ type: 'redirect', target: undefined })
    expect(actions.redirects).toEqual([undefined])
  })
})

function createActions() {
  return {
    refreshes: [] as unknown[],
    redirects: [] as unknown[],
    returns: 0
  }
}

function loadGuestMiddleware(globals: {
  useAuthSession: () => unknown
  returnFromAuth: (explicitRedirect: unknown) => unknown
}) {
  const middlewarePath = new URL('../../app/middleware/guest.ts', import.meta.url)
  const source = existsSync(middlewarePath)
    ? readFileSync(middlewarePath, 'utf8')
    : 'export default defineNuxtRouteMiddleware(async () => {})'
  const transpiler = new Bun.Transpiler({ loader: 'ts', target: 'browser' })
  const executable = transpiler.transformSync(
    source.replace(/^import(?:[\s\S]*?from\s+)?['"][^'"]+['"];?\s*$/gm, '')
  )
    .replaceAll('import.meta.dev', 'true')
    .replace(/export default /, 'return ')

  const factory = new Function(
    'defineNuxtRouteMiddleware',
    'useAuthSession',
    'useAuthReturnNavigation',
    'useState',
    executable
  )

  const state = new Map<string, { value: unknown }>()

  return factory(
    (middleware: (to: { query: Record<string, unknown> }) => Promise<unknown>) => middleware,
    globals.useAuthSession,
    (override?: unknown) => {
      const explicitRedirect = override === undefined
        ? '/old'
        : typeof override === 'object' && override !== null && 'explicitRedirect' in override
          ? (override as { explicitRedirect: unknown }).explicitRedirect
          : override

      return {
        returnFromAuth: () => globals.returnFromAuth(explicitRedirect)
      }
    },
    (key: string, init?: () => unknown) => {
      if (!state.has(key)) {
        state.set(key, { value: init ? init() : null })
      }
      return state.get(key)
    }
  ) as (to: { query: Record<string, unknown> }) => Promise<unknown>
}
