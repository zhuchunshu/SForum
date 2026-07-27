import { describe, expect, test } from 'bun:test'

import * as publicSdk from '../../packages/admin-sdk/src/index'

describe('@sforum/admin-sdk', () => {
  test('exposes only the supported runtime API', () => {
    expect(Object.keys(publicSdk)).toEqual(['ADMIN_MICRO_FRONTEND_API_VERSION'])
    expect(publicSdk.ADMIN_MICRO_FRONTEND_API_VERSION).toBe(1)
    expect('ADMIN_SDK_API_VERSION' in publicSdk).toBe(false)
    expect('useSForumAdminHost' in publicSdk).toBe(false)
  })
})
