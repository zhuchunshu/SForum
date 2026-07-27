import { readFileSync } from 'node:fs'
import { describe, expect, it } from 'vitest'
import {
  PUBLIC_FRONTEND_DOCUMENT_POLICY_SCHEMA_VERSION,
  PUBLIC_FRONTEND_POLICY_SCHEMA_VERSION,
  PUBLIC_PAGE_CSP_HEADER,
  PUBLIC_PAGE_POLICY_PATH,
  collectPublicL2ComponentRefsFromRenderNodes,
  normalizePublicFrontendComponentRefs,
  parsePublicFrontendPolicy,
  publicPagePolicyPath,
  PublicPagePolicyError
} from '../../app/runtime/public-extensions/pagePolicy'

function source(relative: string) {
  return readFileSync(new URL(relative, import.meta.url), 'utf8')
}

describe('public page document CSP policy', () => {
  it('builds deterministic soft-ref page-policy paths', () => {
    expect(publicPagePolicyPath([])).toBe(PUBLIC_PAGE_POLICY_PATH)
    expect(publicPagePolicyPath([
      { extensionId: 'b.plugin', componentId: 'b.card' },
      { extensionId: 'a.plugin', componentId: 'a.card' },
      { extensionId: 'a.plugin', componentId: 'a.card' }
    ])).toBe(
      `${PUBLIC_PAGE_POLICY_PATH}?component=${encodeURIComponent('a.plugin/a.card')}&component=${encodeURIComponent('b.plugin/b.card')}`
    )
    expect(() => normalizePublicFrontendComponentRefs([
      { extensionId: 'Bad', componentId: 'x' }
    ])).toThrow(PublicPagePolicyError)
  })

  it('parses Host document policy and rejects unsafe header values', () => {
    const policy = parsePublicFrontendPolicy({
      schemaVersion: PUBLIC_FRONTEND_POLICY_SCHEMA_VERSION,
      graphDigest: 'a'.repeat(64),
      extensionPolicyDigest: 'b'.repeat(64),
      directives: [{ name: 'img-src', sources: ["'self'", 'https://cdn.example'] }],
      admittedComponents: [{
        extensionId: 'demo.plugin',
        extensionVersion: '1.0.0',
        packageDigest: 'c'.repeat(64),
        impactDigest: 'd'.repeat(64),
        componentId: 'demo.plugin.component.card',
        contractVersion: 'demo.plugin.component.card@1'
      }],
      documentPolicy: {
        schemaVersion: PUBLIC_FRONTEND_DOCUMENT_POLICY_SCHEMA_VERSION,
        digest: 'e'.repeat(64),
        directives: [
          { name: 'default-src', sources: ["'none'"] },
          { name: 'script-src', sources: ["'self'"] }
        ],
        headerValue: "default-src 'none'; script-src 'self'"
      }
    })
    expect(policy.documentPolicy.headerValue).toContain("script-src 'self'")
    expect(() => parsePublicFrontendPolicy({
      ...policy,
      documentPolicy: {
        ...policy.documentPolicy,
        headerValue: "default-src 'none'\nscript-src 'self'"
      }
    })).toThrow(PublicPagePolicyError)
  })

  it('collects L2 soft refs from theme render islands', () => {
    const refs = collectPublicL2ComponentRefsFromRenderNodes([
      {
        kind: 'island',
        descriptor: { componentId: 'core.component.shared.sfextension_widget' },
        props: {
          'extension-id': 'sforum.public-l2-e2e-theme',
          'component-id': 'sforum.public-l2-e2e-theme.component.card'
        },
        children: []
      },
      {
        kind: 'island',
        descriptor: { componentId: 'forum.component.home_page' },
        props: {},
        children: []
      }
    ])
    expect(refs).toEqual([{
      extensionId: 'sforum.public-l2-e2e-theme',
      componentId: 'sforum.public-l2-e2e-theme.component.card'
    }])
  })

  it('wires Nuxt SSR to apply Host DocumentPolicy.HeaderValue', () => {
    const composable = source('../../app/composables/errors/usePublicPageDocumentPolicy.ts')
    const template = source('../../app/components/SFThemeTemplate.vue')
    expect(composable).toContain('setHeader(event, PUBLIC_PAGE_CSP_HEADER, headerValue)')
    expect(composable).toContain(PUBLIC_PAGE_CSP_HEADER)
    expect(composable).toContain('publicPagePolicyPath')
    expect(template).toContain('applyPublicPageDocumentPolicy')
    expect(template).toContain('collectPublicL2ComponentRefsFromRenderNodes')
    expect(template).toContain('data-document-policy-digest')
  })
})
