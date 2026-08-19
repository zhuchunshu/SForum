import { describe, expect, test } from 'bun:test'

import {
  buildPrivateAssetTarget,
  buildPublicAssetTarget
} from '../../server/utils/sforumAssetProxy'

const digest = 'a'.repeat(64)

describe('SForum immutable asset proxy', () => {
  test('maps public theme paths to the guarded legacy handler with an exact digest', () => {
    const target = buildPublicAssetTarget(
      'http://api:8080/api/v1',
      `/_sforum/assets/themes/sforum.default-theme/${digest}/assets/fonts/font.woff2`
    )
    expect(target.toString()).toBe(
      `http://api:8080/api/v1/site/theme-assets/sforum.default-theme/assets/fonts/font.woff2?v=${digest}`
    )
  })

  test('maps public extension packages without exposing their API route', () => {
    const target = buildPublicAssetTarget(
      'http://api:8080/api/v1',
      `/_sforum/assets/extensions/demo.plugin/${digest}/frontend/public/card.mjs`
    )
    expect(target.pathname).toBe(`/api/v1/extensions/runtime/demo.plugin/packages/${digest}/frontend/public/card.mjs`)
    expect(target.search).toBe('')
  })

  test('maps authenticated admin assets and rejects ambiguous paths', () => {
    expect(buildPrivateAssetTarget(
      'http://api:8080/api/v1',
      `/_sforum/private-assets/extensions/demo.plugin/${digest}/entry`
    ).pathname).toBe(`/api/v1/admin/extensions/demo.plugin/frontend/assets/${digest}/entry`)
    expect(buildPrivateAssetTarget(
      'http://api:8080/api/v1',
      `/_sforum/private-assets/extensions/demo.plugin/${digest}/dashboard/entry`
    ).pathname).toBe(`/api/v1/admin/extensions/demo.plugin/frontend/assets/${digest}/dashboard/entry`)

    expect(() => buildPublicAssetTarget(
      'http://api:8080/api/v1',
      `/_sforum/assets/themes/demo.theme/not-a-digest/assets/theme.css`
    )).toThrow()
    expect(() => buildPublicAssetTarget(
      'http://api:8080/api/v1',
      `/_sforum/assets/themes/demo.theme/${digest}/%2e%2e/secret.css`
    )).toThrow()
    expect(() => buildPrivateAssetTarget(
      'http://api:8080/api/v1',
      `/_sforum/private-assets/extensions/demo.plugin/${digest}/other`
    )).toThrow()
  })

})
