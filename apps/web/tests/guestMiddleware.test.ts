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
      returnFromAuth: async () => {
        actions.returns += 1
        return { type: 'redirect' }
      }
    })

    const result = await middleware()

    expect(actions.refreshes).toEqual([{ timeout: 800 }])
    expect(actions.returns).toBe(1)
    expect(result).toEqual({ type: 'redirect' })
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

    expect(await middleware()).toBeUndefined()
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

    expect(await middleware()).toBeUndefined()
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
      returnFromAuth: async () => {
        actions.returns += 1
        return { type: 'redirect' }
      }
    })

    expect(await middleware()).toEqual({ type: 'redirect' })
    expect(actions.refreshes).toEqual([])
    expect(actions.returns).toBe(1)
  })
})

function createActions() {
  return {
    refreshes: [] as unknown[],
    returns: 0
  }
}

function loadGuestMiddleware(globals: {
  useAuthSession: () => unknown
  returnFromAuth: () => unknown
}) {
  const middlewarePath = new URL('../app/middleware/guest.ts', import.meta.url)
  const source = existsSync(middlewarePath)
    ? readFileSync(middlewarePath, 'utf8')
    : 'export default defineNuxtRouteMiddleware(async () => {})'
  const transpiler = new Bun.Transpiler({ loader: 'ts', target: 'browser' })
  const executable = transpiler.transformSync(source)
    .replaceAll('import.meta.dev', 'true')
    .replace(/export default /, 'return ')

  const factory = new Function(
    'defineNuxtRouteMiddleware',
    'useAuthSession',
    'useAuthReturnNavigation',
    executable
  )

  return factory(
    (middleware: () => Promise<unknown>) => middleware,
    globals.useAuthSession,
    () => ({ returnFromAuth: globals.returnFromAuth })
  ) as () => Promise<unknown>
}
