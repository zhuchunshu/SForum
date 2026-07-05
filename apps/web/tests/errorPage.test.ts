import { describe, expect, test } from 'bun:test'

import {
  normalizeErrorStatus,
  resolveErrorPageContent
} from '../app/utils/errorPage'

describe('error page helpers', () => {
  test('normalizes invalid or non-error status codes to 500', () => {
    expect(normalizeErrorStatus(undefined)).toBe(500)
    expect(normalizeErrorStatus('abc')).toBe(500)
    expect(normalizeErrorStatus(200)).toBe(500)
    expect(normalizeErrorStatus(700)).toBe(500)
    expect(normalizeErrorStatus('404')).toBe(404)
  })

  test('resolves known status-specific content', () => {
    expect(resolveErrorPageContent(404)).toEqual({
      statusCode: 404,
      titleKey: 'errors.page.notFound.title',
      descriptionKey: 'errors.page.notFound.description',
      icon: 'i-lucide-file-question',
      showRetry: false
    })
    expect(resolveErrorPageContent(403).titleKey).toBe('errors.page.forbidden.title')
    expect(resolveErrorPageContent(503).descriptionKey).toBe('errors.page.serviceUnavailable.description')
    expect(resolveErrorPageContent(503).showRetry).toBe(true)
    expect(resolveErrorPageContent(500).icon).toBe('i-lucide-server')
  })

  test('uses generic copy for unknown error statuses', () => {
    expect(resolveErrorPageContent(418)).toEqual({
      statusCode: 418,
      titleKey: 'errors.page.generic.title',
      descriptionKey: 'errors.page.generic.description',
      icon: 'i-lucide-circle-alert',
      showRetry: false
    })
  })
})
