import { describe, expect, it } from 'bun:test'
import { readFileSync } from 'node:fs'

import {
  parsePublicContentStyleCatalog,
  PUBLIC_CONTENT_STYLE_SCOPE,
  PUBLIC_CONTENT_STYLES_SCHEMA_VERSION
} from '../../app/runtime/public-extensions/contentStyles'

const digest = 'a'.repeat(64)
const impact = 'b'.repeat(64)

describe('public content extension styles', () => {
  it('admits exact blocking styles for the forum content surface', () => {
    const catalog = parsePublicContentStyleCatalog({
      schemaVersion: PUBLIC_CONTENT_STYLES_SCHEMA_VERSION,
      graphDigest: digest,
      styles: [{
        handle: 'demo.plugin.asset.content-style',
        contractVersion: 'demo.plugin.asset.content-style@1',
        extensionId: 'demo.plugin',
        packageDigest: digest,
        impactDigest: impact,
        type: 'style',
        digest,
        integrity: 'sha256-qqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqo=',
        dependencies: [],
        scope: [PUBLIC_CONTENT_STYLE_SCOPE],
        module: false,
        loading: 'blocking',
        csp: ["style-src 'self'"],
        assetPath: `/_sforum/assets/extensions/demo.plugin/${digest}/frontend/public/content.css`
      }]
    })
    expect(catalog.styles).toHaveLength(1)
    expect(catalog.styles[0]!.scope).toContain(PUBLIC_CONTENT_STYLE_SCOPE)
  })

  it('keeps shortcode selectors in the plugin artifact rather than Core styles or SFCs', () => {
    const pluginCSS = readFileSync(new URL(
      '../../../../extensions/builtin/plugins/sforum-shortcodes/frontend/public/shortcodes.css',
      import.meta.url
    ), 'utf8')
    const coreCSS = readFileSync(new URL('../../app/assets/css/sforum-content-semantics.css', import.meta.url), 'utf8')
    const referenceNode = readFileSync(new URL(
      '../../app/components/editor/shortcodes/SFShortcodeReferenceNodeView.vue',
      import.meta.url
    ), 'utf8')
    const protectedNode = readFileSync(new URL(
      '../../app/components/editor/shortcodes/SFShortcodeProtectedNodeView.vue',
      import.meta.url
    ), 'utf8')

    expect(pluginCSS).toContain('.sf-prose .sf-shortcode--comment')
    expect(pluginCSS).toContain('.sf-shortcode-node__label')
    expect(pluginCSS).toContain('.sf-protected-node__label')
    expect(coreCSS).not.toContain('.sf-shortcode')
    expect(coreCSS).not.toContain('.sf-editor-fallback')
    expect(referenceNode).not.toContain('<style')
    expect(protectedNode).not.toContain('<style')
  })

  it('keeps published protected blocks on the editor-preview geometry without a nested fallback card', () => {
    const pluginCSS = readFileSync(new URL(
      '../../../../extensions/builtin/plugins/sforum-shortcodes/frontend/public/shortcodes.css',
      import.meta.url
    ), 'utf8')
    const protectedSurface = pluginCSS.match(/\.sf-prose \.sf-shortcode--protected \{([\s\S]*?)\n\}/)?.[1] || ''
    const nestedFallback = pluginCSS.match(
      /\.sf-prose \.sf-shortcode--protected > p,\n\.sf-prose \.sf-shortcode--protected > \.sf-editor-fallback \{([\s\S]*?)\n\}/
    )?.[1] || ''

    expect(protectedSurface).toContain('padding: 1.5rem .875rem .875rem')
    expect(protectedSurface).toContain('border-color: color-mix')
    expect(pluginCSS).toContain('.sf-prose .sf-shortcode--protected::before')
    expect(pluginCSS).toContain('.sf-prose .sf-shortcode--login::before')
    expect(pluginCSS).toContain('.sf-prose .sf-shortcode--reply::before')
    expect(pluginCSS).toContain('.sf-prose .sf-shortcode--only-author::before')
    expect(nestedFallback).toContain('border: 0')
    expect(nestedFallback).toContain('background: transparent')
    expect(pluginCSS).toContain('.dark .sf-prose .sf-shortcode--protected > .sf-editor-fallback')
  })
})
