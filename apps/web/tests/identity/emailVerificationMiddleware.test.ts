import { describe, expect, test } from 'bun:test'
import { readFileSync } from 'node:fs'
import { ref } from 'vue'

type TestUser = { emailVerified: boolean }

describe('email verification global middleware', () => {
  test('redirects an unverified user from a public page and preserves the destination', async () => {
    const user = ref<TestUser | null>({ emailVerified: false })
    const actions = createActions()
    const middleware = loadMiddleware({ user, actions })

    const result = await middleware(route('/t/42/example?page=2'))

    expect(actions.sessionRefreshes).toHaveLength(1)
    expect(actions.navigations).toEqual([{
      path: '/email-verification',
      query: { redirect: '/t/42/example?page=2' }
    }])
    expect(result).toEqual({ type: 'navigate', target: actions.navigations[0] })
  })

  test('allows the verification page and admin routes without a redirect loop', async () => {
    const user = ref<TestUser | null>({ emailVerified: false })

    for (const path of ['/email-verification', '/email-verification/', '/control-panel/users']) {
      const actions = createActions()
      const middleware = loadMiddleware({ user, actions })

      expect(await middleware(route(path, path === '/email-verification'))).toBeUndefined()
      expect(actions.navigations).toEqual([])
      expect(actions.optionRefreshes).toEqual([])
      expect(actions.sessionRefreshes).toEqual([])
    }
  })

  test('allows unverified users when the operator setting is disabled', async () => {
    const actions = createActions()
    const middleware = loadMiddleware({
      user: ref<TestUser | null>({ emailVerified: false }),
      required: false,
      actions
    })

    expect(await middleware(route('/categories'))).toBeUndefined()
    expect(actions.sessionRefreshes).toEqual([])
    expect(actions.navigations).toEqual([])
  })

  test('refreshes a verified session so an administrator reset takes effect on the next page', async () => {
    const user = ref<TestUser | null>({ emailVerified: true })
    const actions = createActions()
    const middleware = loadMiddleware({
      user,
      actions,
      refreshSession: async () => {
        user.value = { emailVerified: false }
      }
    })

    await middleware(route('/categories'))

    expect(actions.sessionRefreshes).toHaveLength(1)
    expect(actions.navigations).toEqual([{
      path: '/email-verification',
      query: { redirect: '/categories' }
    }])
  })

  test('loads the public setting and resolves an unknown signed-in session', async () => {
    const user = ref<TestUser | null>(null)
    const status = ref('unknown')
    const actions = createActions()
    const middleware = loadMiddleware({
      user,
      status,
      optionsLoaded: false,
      actions,
      refreshSession: async () => {
        user.value = { emailVerified: false }
        status.value = 'authenticated'
      }
    })

    await middleware(route('/'))

    expect(actions.optionRefreshes).toEqual([{ timeout: 800 }])
    expect(actions.sessionRefreshes).toEqual([{ timeout: 800 }])
    expect(actions.navigations[0]).toEqual({
      path: '/email-verification',
      query: { redirect: '/' }
    })
  })

  test('keeps the existing protected-route login redirect for guests', async () => {
    const actions = createActions()
    const middleware = loadMiddleware({
      user: ref(null),
      status: ref('guest'),
      required: false,
      actions
    })

    await middleware(route('/settings/profile', true))

    expect(actions.navigations).toEqual([{
      path: '/login',
      query: { redirect: '/settings/profile' }
    }])
  })
})

function route(path: string, requiresAuth = false) {
  return { path: path.split('?')[0], fullPath: path, meta: { requiresAuth } }
}

function createActions() {
  return {
    optionRefreshes: [] as unknown[],
    sessionRefreshes: [] as unknown[],
    navigations: [] as unknown[]
  }
}

function loadMiddleware(input: {
  user: ReturnType<typeof ref<TestUser | null>>
  status?: ReturnType<typeof ref<string>>
  required?: boolean
  optionsLoaded?: boolean
  refreshSession?: () => Promise<void>
  actions: ReturnType<typeof createActions>
}) {
  const source = readFileSync(new URL('../../app/middleware/auth.global.ts', import.meta.url), 'utf8')
  const transpiler = new Bun.Transpiler({ loader: 'ts', target: 'browser' })
  const executable = transpiler.transformSync(
    source.replace(/^import(?:[\s\S]*?from\s+)?['"][^'"]+['"];?\s*$/gm, '')
  )
    .replaceAll('import.meta.dev', 'true')
    .replaceAll('import.meta.server', 'false')
    .replace(/export default /, 'return ')

  const factory = new Function(
    'defineNuxtRouteMiddleware',
    'useLocalePath',
    'useRuntimeConfig',
    'normalizeAdminRoutePrefix',
    'resolveAdminRouteChildPath',
    'useAuthSession',
    'useWebOptions',
    'normalizeEnabledOption',
    'useRequestHeaders',
    'navigateTo',
    executable
  )

  const loaded = ref(input.optionsLoaded ?? true)
  const status = input.status ?? ref('authenticated')
  const required = input.required ?? true

  return factory(
    (middleware: (to: ReturnType<typeof route>) => Promise<unknown>) => middleware,
    () => (path: string) => path,
    () => ({ public: { adminRoutePrefix: '/control-panel' } }),
    (prefix: string) => prefix,
    (prefix: string, path: string) => path.startsWith(prefix) ? path : null,
    () => ({
      user: input.user,
      status,
      refresh: async (options: unknown) => {
        input.actions.sessionRefreshes.push(options)
        await input.refreshSession?.()
      }
    }),
    () => ({
      loaded,
      refresh: async (options: unknown) => {
        input.actions.optionRefreshes.push(options)
        loaded.value = true
      },
      webOption: () => required ? 'enabled' : 'disabled'
    }),
    (value: string) => value === 'enabled',
    () => ({}),
    (target: unknown) => {
      input.actions.navigations.push(target)
      return { type: 'navigate', target }
    }
  ) as (to: ReturnType<typeof route>) => Promise<unknown>
}
