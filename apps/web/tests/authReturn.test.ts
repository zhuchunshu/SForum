import { describe, expect, test } from 'bun:test'

import {
  normalizeAuthReturnPath,
  resolveAuthReturnPath
} from '../app/utils/authReturn'

describe('normalizeAuthReturnPath', () => {
  test('preserves safe local paths with query strings and hashes', () => {
    expect(normalizeAuthReturnPath('/t/42?page=2#reply')).toBe('/t/42?page=2#reply')
    expect(normalizeAuthReturnPath('/en/settings/security')).toBe('/en/settings/security')
  })

  test('rejects external, protocol-relative, and malformed paths', () => {
    expect(normalizeAuthReturnPath('https://evil.example')).toBeNull()
    expect(normalizeAuthReturnPath('//evil.example')).toBeNull()
    expect(normalizeAuthReturnPath('/%E0%A4%A')).toBeNull()
  })

  test('rejects absent and non-string candidates', () => {
    expect(normalizeAuthReturnPath(undefined)).toBeNull()
    expect(normalizeAuthReturnPath(['/settings'])).toBeNull()
  })

  test('rejects authentication pages at any locale depth', () => {
    expect(normalizeAuthReturnPath('/login')).toBeNull()
    expect(normalizeAuthReturnPath('/en/register?redirect=/settings')).toBeNull()
  })
})

describe('resolveAuthReturnPath', () => {
  test('prefers an explicit safe target over the referrer', () => {
    expect(resolveAuthReturnPath('/settings', '/t/42', '/en')).toBe('/settings')
  })

  test('falls back to a safe referrer', () => {
    expect(resolveAuthReturnPath('https://evil.example', '/t/42?page=2#reply', '/en'))
      .toBe('/t/42?page=2#reply')
  })

  test('falls back to localized home and ultimately the root path', () => {
    expect(resolveAuthReturnPath(undefined, '/login', '/en')).toBe('/en')
    expect(resolveAuthReturnPath(undefined, '/login', 'https://evil.example')).toBe('/')
  })
})
