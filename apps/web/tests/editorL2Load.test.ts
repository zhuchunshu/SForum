import { describe, expect, it } from 'bun:test'
import { createHash } from 'node:crypto'
import {
  EDITOR_CATALOG_SCHEMA_VERSION,
  EditorL2ContractError,
  isEditorL2Module,
  parseEditorCatalog
} from '../app/runtime/editor-extensions/types'
import { loadTrustedEditorL2Module, verifyDigest } from '../app/runtime/editor-extensions/load'
import { admitEditorCatalogModules } from '../app/runtime/editor-extensions/admit'
import { createSFEditorExtensions } from '../app/utils/sfEditor'

function sha256Hex(value: string) {
  return createHash('sha256').update(value).digest('hex')
}

describe('trusted editor L2 load path', () => {
  it('parses Host editor catalog and rejects digest-unbound asset paths', () => {
    const packageDigest = 'ab'.repeat(32)
    const moduleDigest = 'cd'.repeat(32)
    const catalog = parseEditorCatalog({
      schemaVersion: EDITOR_CATALOG_SCHEMA_VERSION,
      revision: 2,
      digest: 'ef'.repeat(32),
      modules: [{
        extensionId: 'demo.editor',
        extensionVersion: '1.0.0',
        packageDigest,
        l2Module: 'frontend/editor/vote.mjs',
        l2Digest: moduleDigest,
        assetPath: `/_sforum/assets/extensions/demo.editor/${packageDigest}/frontend/editor/vote.mjs`,
        nodes: [{
          id: 'demo.editor.node.vote',
          contractVersion: 'demo.editor.node.vote@1',
          kind: 'node',
          extensionName: 'demoVote',
          artifact: { extensionId: 'demo.editor', extensionVersion: '1.0.0', packageDigest }
        }],
        marks: [],
        commands: [],
        toolbars: []
      }],
      toolbars: []
    })
    expect(catalog.modules).toHaveLength(1)
    expect(() => parseEditorCatalogModuleWithBadPath(packageDigest, moduleDigest)).toThrow(EditorL2ContractError)
  })

  it('digest-verifies bytes before import and keeps core extensions on failure', async () => {
    const body = 'export default { apiVersion: 1, createExtensions: () => [{ name: "demoVote" }] }'
    const moduleDigest = sha256Hex(body)
    const packageDigest = '11'.repeat(32)
    const assetPath = `/_sforum/assets/extensions/demo.editor/${packageDigest}/frontend/editor/vote.mjs`
    const fakeExtension = { name: 'demoVote' }
    const loaded = await loadTrustedEditorL2Module(
      {
        extensionId: 'demo.editor',
        extensionVersion: '1.0.0',
        packageDigest,
        l2Module: 'frontend/editor/vote.mjs',
        l2Digest: moduleDigest,
        assetPath,
        nodes: [],
        marks: [],
        commands: [],
        toolbars: []
      },
      'http://localhost:8080',
      {
        origin: 'http://localhost:8080',
        fetch: async () => new Response(body, { status: 200 }),
        importModule: async () => ({
          apiVersion: 1,
          createExtensions: () => [fakeExtension]
        }),
        subtle: {
          digest: async (_algo: string, data: ArrayBuffer) => {
            const hex = createHash('sha256').update(Buffer.from(data)).digest()
            return hex.buffer.slice(hex.byteOffset, hex.byteOffset + hex.byteLength)
          }
        } as SubtleCrypto
      }
    )
    expect(isEditorL2Module(loaded.module)).toBe(true)
    expect(loaded.extensions).toEqual([fakeExtension])

    await expect(loadTrustedEditorL2Module(
      {
        extensionId: 'demo.editor',
        extensionVersion: '1.0.0',
        packageDigest,
        l2Module: 'frontend/editor/vote.mjs',
        l2Digest: '00'.repeat(32),
        assetPath,
        nodes: [],
        marks: [],
        commands: [],
        toolbars: []
      },
      'http://localhost:8080',
      {
        origin: 'http://localhost:8080',
        fetch: async () => new Response(body, { status: 200 }),
        importModule: async () => {
          throw new Error('must not import after digest mismatch')
        },
        subtle: {
          digest: async (_algo: string, data: ArrayBuffer) => {
            const hex = createHash('sha256').update(Buffer.from(data)).digest()
            return hex.buffer.slice(hex.byteOffset, hex.byteOffset + hex.byteLength)
          }
        } as SubtleCrypto
      }
    )).rejects.toBeInstanceOf(EditorL2ContractError)

    const coreOnly = createSFEditorExtensions({
      placeholder: 'x',
      maxCharacters: 100,
      trustedExtensions: []
    })
    expect(coreOnly.length).toBeGreaterThan(3)
    const withTrusted = createSFEditorExtensions({
      placeholder: 'x',
      maxCharacters: 100,
      trustedExtensions: [fakeExtension]
    })
    expect(withTrusted.at(-1)).toEqual(fakeExtension)
  })

  it('admits catalog modules and quarantines failures without dropping successes', async () => {
    const packageDigest = '22'.repeat(32)
    const goodBody = 'good-module'
    const goodDigest = sha256Hex(goodBody)
    const catalog = parseEditorCatalog({
      schemaVersion: EDITOR_CATALOG_SCHEMA_VERSION,
      revision: 1,
      digest: '33'.repeat(32),
      modules: [
        {
          extensionId: 'demo.editor',
          extensionVersion: '1.0.0',
          packageDigest,
          l2Module: 'frontend/editor/good.mjs',
          l2Digest: goodDigest,
          assetPath: `/_sforum/assets/extensions/demo.editor/${packageDigest}/frontend/editor/good.mjs`,
          nodes: [],
          marks: [],
          commands: [],
          toolbars: []
        },
        {
          extensionId: 'demo.editor',
          extensionVersion: '1.0.0',
          packageDigest,
          l2Module: 'frontend/editor/bad.mjs',
          l2Digest: '44'.repeat(32),
          assetPath: `/_sforum/assets/extensions/demo.editor/${packageDigest}/frontend/editor/bad.mjs`,
          nodes: [],
          marks: [],
          commands: [],
          toolbars: []
        }
      ],
      toolbars: []
    })
    const admitted = await admitEditorCatalogModules(catalog, 'http://localhost:8080', async (module) => {
      if (module.l2Module.includes('bad')) {
        throw new EditorL2ContractError('forced quarantine')
      }
      return {
        module: { apiVersion: 1 as const, createExtensions: () => [{ name: 'good' }] },
        bridge: {
          apiVersion: 1 as const,
          extensionId: module.extensionId,
          extensionVersion: module.extensionVersion,
          packageDigest: module.packageDigest,
          modulePath: module.l2Module,
          moduleDigest: module.l2Digest
        },
        extensions: [{ name: 'good' }]
      }
    })
    expect(admitted.extensions).toEqual([{ name: 'good' }])
    expect(admitted.quarantined).toHaveLength(1)
    expect(admitted.quarantined[0]).toContain('bad.mjs')
  })

  it('verifyDigest rejects byte drift', async () => {
    const body = new TextEncoder().encode('hello')
    const digest = sha256Hex('hello')
    await verifyDigest(body.buffer, digest, {
      digest: async (_algo: string, data: ArrayBuffer) => {
        const hex = createHash('sha256').update(Buffer.from(data)).digest()
        return hex.buffer.slice(hex.byteOffset, hex.byteOffset + hex.byteLength)
      }
    } as SubtleCrypto)
    await expect(verifyDigest(body.buffer, 'ff'.repeat(32), {
      digest: async (_algo: string, data: ArrayBuffer) => {
        const hex = createHash('sha256').update(Buffer.from(data)).digest()
        return hex.buffer.slice(hex.byteOffset, hex.byteOffset + hex.byteLength)
      }
    } as SubtleCrypto)).rejects.toBeInstanceOf(EditorL2ContractError)
  })
})

function parseEditorCatalogModuleWithBadPath(packageDigest: string, moduleDigest: string) {
  return parseEditorCatalog({
    schemaVersion: EDITOR_CATALOG_SCHEMA_VERSION,
    revision: 1,
    digest: 'aa'.repeat(32),
    modules: [{
      extensionId: 'demo.editor',
      extensionVersion: '1.0.0',
      packageDigest,
      l2Module: 'frontend/editor/vote.mjs',
      l2Digest: moduleDigest,
      assetPath: '/_sforum/assets/extensions/demo.editor/otherdigest/frontend/editor/vote.mjs',
      nodes: [],
      marks: [],
      commands: [],
      toolbars: []
    }],
    toolbars: []
  })
}

describe('forumContentFromEditorPayload', () => {
  it('prefers editor-document when native JSON is present', async () => {
    const { forumContentFromEditorPayload } = await import('../app/utils/forumTaxonomy')
    const content = forumContentFromEditorPayload({
      markdown: 'hello',
      native: { type: 'doc', content: [{ type: 'paragraph', content: [{ type: 'text', text: 'hello' }] }] },
      text: 'hello'
    })
    expect(content.sourceFormat).toBe('editor-document')
    expect(content.editorType).toBe('tiptap')
    expect(JSON.parse(content.rawContent).type).toBe('doc')
  })

  it('falls back to markdown without native payload', async () => {
    const { forumContentFromEditorPayload } = await import('../app/utils/forumTaxonomy')
    const content = forumContentFromEditorPayload({ markdown: 'plain body' })
    expect(content).toEqual({
      rawContent: 'plain body',
      sourceFormat: 'markdown',
      editorType: 'tiptap',
      editorVersion: 'sf-editor-v1'
    })
  })
})
