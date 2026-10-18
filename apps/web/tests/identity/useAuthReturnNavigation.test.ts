import { describe, expect, test } from 'bun:test'
import { readFileSync } from 'node:fs'
import { computed, ref } from 'vue'

import {
  buildAuthPageLink,
  requiresDocumentNavigation,
  resolveAuthReturnPath
} from '../../app/utils/identity/authReturn'

/**
 * 行为级回归：认证完成后的回跳目标决定用哪种导航。
 * API / 插件端点（如 OAuth2 的 `/api/oauth/authorize`）只存在于服务端，
 * 客户端路由会被 `[...sfRegistryPage]` catch-all 接住，必须整页导航。
 */

const sourcePath = new URL(
  '../../app/composables/identity/useAuthReturnNavigation.ts',
  import.meta.url
)

type Harness = {
  returnFromAuth: () => Promise<unknown>
  documentAssignments: string[]
  navigateCalls: Array<{ target: string, options?: Record<string, unknown> }>
}

function loadComposable(redirect: unknown): Harness {
  const source = readFileSync(sourcePath, 'utf8')
  const transpiler = new Bun.Transpiler({ loader: 'ts', target: 'browser' })
  const executable = transpiler
    .transformSync(source.replace(/^import(?:[\s\S]*?from\s+)?['"][^'"]+['"];?\s*$/gm, ''))
    .replaceAll('import.meta.client', 'true')
    .replace('export function useAuthReturnNavigation', 'return function useAuthReturnNavigation')

  const documentAssignments: string[] = []
  const navigateCalls: Harness['navigateCalls'] = []
  const fakeWindow = {
    location: {
      origin: 'http://sforum.local',
      assign: (url: string) => {
        documentAssignments.push(url)
      }
    }
  }

  const factory = new Function(
    'buildAuthPageLink',
    'requiresDocumentNavigation',
    'resolveAuthReturnPath',
    'useRoute',
    'useLocalePath',
    'ref',
    'computed',
    'navigateTo',
    'window',
    'document',
    executable
  )

  const useAuthReturnNavigation = factory(
    buildAuthPageLink,
    requiresDocumentNavigation,
    resolveAuthReturnPath,
    () => ({ query: { redirect } }),
    () => (path: string) => path,
    ref,
    computed,
    (target: string, options?: Record<string, unknown>) => {
      navigateCalls.push({ target, options })
      return Promise.resolve()
    },
    fakeWindow,
    { referrer: '' }
  ) as () => { returnFromAuth: () => Promise<unknown> }

  return { ...useAuthReturnNavigation(), documentAssignments, navigateCalls }
}

describe('useAuthReturnNavigation returnFromAuth', () => {
  test('loads API endpoints as a full document navigation', async () => {
    const target = '/api/oauth/authorize?client_id=sfo_client_x&state=s'
    const harness = loadComposable(target)

    await harness.returnFromAuth()

    expect(harness.documentAssignments).toEqual([target])
    expect(harness.navigateCalls).toEqual([])
  })

  test('loads plugin routes as a full document navigation', async () => {
    const harness = loadComposable('/plugin/echo?next=1')

    await harness.returnFromAuth()

    expect(harness.documentAssignments).toEqual(['/plugin/echo?next=1'])
    expect(harness.navigateCalls).toEqual([])
  })

  test('keeps Nuxt pages on client-side navigation', async () => {
    const harness = loadComposable('/settings/security')

    await harness.returnFromAuth()

    expect(harness.documentAssignments).toEqual([])
    expect(harness.navigateCalls).toEqual([
      { target: '/settings/security', options: { replace: true } }
    ])
  })
})
