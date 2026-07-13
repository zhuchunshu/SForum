import { describe, expect, test } from 'bun:test'
import { createApp } from 'vue'

import * as publicSdk from '../packages/admin-sdk/src/index'
import { ADMIN_HOST_INJECTION_KEY } from '../packages/admin-sdk/src/internal'
import type { AdminSlotProps } from '@sforum/admin-sdk'

declare module '@sforum/admin-sdk' {
  interface AdminSlotContextMap {
    'admin.test.fixture': { jobId: number }
  }
  interface AdminSlotOptionsMap {
    'admin.test.fixture': { width: number }
  }
}

const fixtureProps = {
  context: { jobId: 42 },
  options: { width: 120 },
  extensionId: 'demo.owner',
  contributionId: 'latency'
} satisfies AdminSlotProps<'admin.test.fixture'>

describe('@sforum/admin-sdk', () => {
  test('exposes only the supported runtime API', () => {
    expect(Object.keys(publicSdk).sort()).toEqual([
      'ADMIN_MICRO_FRONTEND_API_VERSION',
      'ADMIN_SDK_API_VERSION',
      'useSForumAdminHost'
    ])
    expect(publicSdk.ADMIN_MICRO_FRONTEND_API_VERSION).toBe(1)
    expect(publicSdk.ADMIN_SDK_API_VERSION).toBe(1)
    expect('ADMIN_HOST_INJECTION_KEY' in publicSdk).toBe(false)
    expect(fixtureProps.context.jobId).toBe(42)
  })

  test('resolves the host provided by the owning contribution boundary', () => {
    const ownerHost = {
      extensionId: 'demo.owner',
      locale: { value: 'zh-CN' },
      t: (key: string) => key,
      navigate: async () => {},
      toast: () => {},
      extensionRequest: async () => ({})
    }
    const app = createApp({})
    app.provide(ADMIN_HOST_INJECTION_KEY, ownerHost)

    expect(app.runWithContext(() => publicSdk.useSForumAdminHost().extensionId)).toBe('demo.owner')
  })
})
