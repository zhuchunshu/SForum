import { afterAll, describe, expect, test } from 'bun:test'
import { readFileSync } from 'node:fs'

/**
 * 后台路径必须跟随运营者的 adminRoutePrefix，本地化交给 i18n。
 * i18n 是 no_prefix：手拼 `/<locale><prefix>` 只在整页加载时被服务端 301 兜住，
 * 客户端导航会落到不存在的路由。
 */

const composableSource = readFileSync(
  new URL('../../app/composables/admin/useAdminRoutes.ts', import.meta.url),
  'utf8'
)

const originalGlobals = {
  useRuntimeConfig: (globalThis as { useRuntimeConfig?: unknown }).useRuntimeConfig,
  useLocalePath: (globalThis as { useLocalePath?: unknown }).useLocalePath
}

type AdminRoutes = {
  prefix: string
  path: (childPath?: string) => string
  routeId: (routePath: string) => string | null
}

async function loadAdminRoutes(options: { prefix?: string } = {}) {
  const localePathCalls: string[] = []
  Object.assign(globalThis, {
    useRuntimeConfig: () => ({ public: { adminRoutePrefix: options.prefix } }),
    useLocalePath: () => (path: string) => {
      localePathCalls.push(path)
      return path
    }
  })
  const { useAdminRoutes } = await import('../../app/composables/admin/useAdminRoutes')

  return { routes: useAdminRoutes() as AdminRoutes, localePathCalls }
}

afterAll(() => {
  Object.assign(globalThis, originalGlobals)
})

describe('admin route paths', () => {
  test('follows the operator prefix and delegates localization to i18n', async () => {
    const { routes, localePathCalls } = await loadAdminRoutes({ prefix: '/manage' })

    expect(routes.prefix).toBe('/manage')
    expect(routes.path()).toBe('/manage')
    expect(routes.path('/roles')).toBe('/manage/roles')
    // 每个后台路径只经 localePath 一次：既不重复加前缀，也不手拼语言段。
    expect(localePathCalls).toEqual(['/manage', '/manage/roles'])
  })

  test('falls back to the default control panel prefix', async () => {
    const { routes } = await loadAdminRoutes({ prefix: '' })

    expect(routes.path('/')).toBe('/control-panel')
    expect(routes.routeId('/control-panel/users')).toBe('/users')
  })

  test('never hand-rolls a locale prefix on top of the i18n authority', () => {
    expect(composableSource).toContain('localePath(joinAdminRoutePath(prefix, childPath))')
    expect(composableSource).not.toContain('${locale.value}')
  })
})
