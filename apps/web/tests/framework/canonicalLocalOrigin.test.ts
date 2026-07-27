import { describe, expect, test } from 'bun:test'
import {
  parseCanonicalLocalOrigin,
  parseLocalRequestHost,
  resolveCanonicalLocalRedirect,
  type CanonicalLocalRequest
} from '../../server/utils/canonicalLocalOrigin'

const canonical = parseCanonicalLocalOrigin('http://127.0.0.1:3000')
const baseRequest: CanonicalLocalRequest = {
  development: true,
  method: 'GET',
  accept: 'text/html,application/xhtml+xml',
  host: 'localhost:3000',
  protocol: 'http',
  pathname: '/categories',
  search: '?group=core'
}

function redirect(overrides: Partial<CanonicalLocalRequest> = {}) {
  return resolveCanonicalLocalRedirect(canonical, { ...baseRequest, ...overrides })
}

describe('canonical local origin', () => {
  test('accepts only origin-only loopback APP_URL values', () => {
    expect(parseCanonicalLocalOrigin('http://localhost')).toEqual({
      origin: 'http://localhost', protocol: 'http:', hostname: 'localhost', port: 80
    })
    expect(parseCanonicalLocalOrigin('https://127.0.0.1')).toEqual({
      origin: 'https://127.0.0.1', protocol: 'https:', hostname: '127.0.0.1', port: 443
    })
    expect(parseCanonicalLocalOrigin('http://[::1]:3000')).toEqual({
      origin: 'http://[::1]:3000', protocol: 'http:', hostname: '[::1]', port: 3000
    })

    for (const value of [
      undefined,
      '',
      'not a url',
      'ftp://127.0.0.1:3000',
      'http://forum.example.com:3000',
      'http://user:secret@127.0.0.1:3000',
      'http://127.0.0.1:3000/base',
      'http://127.0.0.1:3000/?x=1',
      'http://127.0.0.1:3000/#fragment'
    ]) {
      expect(parseCanonicalLocalOrigin(value)).toBeNull()
    }
  })

  test('strictly parses the three supported request Host forms', () => {
    expect(parseLocalRequestHost('localhost:3000')).toEqual({ hostname: 'localhost', explicitPort: 3000 })
    expect(parseLocalRequestHost('127.0.0.1')).toEqual({ hostname: '127.0.0.1', explicitPort: undefined })
    expect(parseLocalRequestHost('[::1]:443')).toEqual({ hostname: '[::1]', explicitPort: 443 })

    for (const value of [
      undefined,
      '',
      'localhost.',
      '127.1:3000',
      '2130706433:3000',
      'user@localhost:3000',
      'localhost:3000,evil.test',
      '::1:3000',
      '[0:0:0:0:0:0:0:1]:3000',
      'localhost:0',
      'localhost:65536'
    ]) {
      expect(parseLocalRequestHost(value)).toBeNull()
    }
  })

  test('redirects aliases to the fixed configured authority with path and query intact', () => {
    expect(redirect()).toBe('http://127.0.0.1:3000/categories?group=core')
    expect(redirect({
      host: '[::1]:3000',
      pathname: '/auth/providers/github/callback',
      search: '?code=a%2Fb&state=one&state=two'
    })).toBe('http://127.0.0.1:3000/auth/providers/github/callback?code=a%2Fb&state=one&state=two')
    expect(redirect({ host: '127.0.0.1:3000' })).toBeNull()
  })

  test('fails closed outside development HTML GET and HEAD documents', () => {
    expect(redirect({ development: false })).toBeNull()
    expect(redirect({ method: 'POST' })).toBeNull()
    expect(redirect({ method: 'OPTIONS' })).toBeNull()
    expect(redirect({ method: 'HEAD' })).toBe('http://127.0.0.1:3000/categories?group=core')
    expect(redirect({ accept: 'application/json' })).toBeNull()
    expect(redirect({ accept: '*/*' })).toBeNull()
    expect(redirect({ accept: 'TEXT/HTML; charset=utf-8' })).toBe('http://127.0.0.1:3000/categories?group=core')
  })

  test('excludes API, Nuxt/HMR, immutable assets, and health probes precisely', () => {
    for (const path of [
      '/api', '/api/v1/health',
      '/_nuxt', '/_nuxt/@vite/client',
      '/_sforum', '/_sforum/assets/digest/app.css',
      '/health', '/health/live'
    ]) {
      expect(redirect({ pathname: path, search: '' })).toBeNull()
    }
    expect(redirect({ pathname: '/apiary', search: '' })).toBe('http://127.0.0.1:3000/apiary')
    expect(redirect({ pathname: '/healthcheck', search: '' })).toBe('http://127.0.0.1:3000/healthcheck')
  })

  test('rejects malformed request targets, ports, protocols, and injected hosts', () => {
    expect(redirect({ host: 'localhost:3001' })).toBeNull()
    expect(redirect({ host: 'evil.test:3000' })).toBeNull()
    expect(redirect({ protocol: 'file' })).toBeNull()
    expect(redirect({ pathname: '//evil.test/path' })).toBeNull()
    expect(redirect({ pathname: '/safe\r\nLocation: https://evil.test' })).toBeNull()
    expect(redirect({ search: 'x=1' })).toBeNull()
    expect(redirect({ search: '?x=1\r\nLocation:https://evil.test' })).toBeNull()
  })

  test('redirects protocol mismatches only when the request uses the canonical port', () => {
    const secureCanonical = parseCanonicalLocalOrigin('https://127.0.0.1:443')
    expect(resolveCanonicalLocalRedirect(secureCanonical, {
      ...baseRequest,
      host: '127.0.0.1:443',
      protocol: 'http'
    })).toBe('https://127.0.0.1/categories?group=core')
    expect(resolveCanonicalLocalRedirect(secureCanonical, {
      ...baseRequest,
      host: '127.0.0.1',
      protocol: 'http'
    })).toBeNull()
  })

  test('middleware uses the trusted helpers, temporary redirect, and no-store', async () => {
    const source = await Bun.file(new URL('../../server/middleware/canonical-local-origin.ts', import.meta.url)).text()
    expect(source).toContain('process.env.APP_URL')
    expect(source).toContain('import.meta.dev')
    expect(source).toContain("getRequestHeader(event, 'host')")
    expect(source).toContain('xForwardedProto: false')
    expect(source).toContain("setHeader(event, 'cache-control', 'no-store')")
    expect(source).toContain('sendRedirect(event, target, 307)')
    expect(source).not.toContain('x-forwarded-host')
    expect(source).not.toContain('NUXT_PUBLIC_I18N_BASE_URL')
  })
})
