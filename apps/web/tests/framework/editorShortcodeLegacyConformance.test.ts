import { afterEach, describe, expect, it } from 'bun:test'
import { readFileSync } from 'node:fs'
import { resolve } from 'node:path'
import { Editor, type JSONContent } from '@tiptap/core'
import { createSFEditorExtensions } from '../../app/utils/sfEditor'
import {
  inspectSForumShortcodeDocument,
  inspectSForumShortcodeMarkdown,
  prepareSForumShortcodeMarkdown,
  SF_SHORTCODE_TEXT_VERSION,
  type SFShortcodeResourceKind
} from '../../app/utils/editor/shortcodes'

type FixtureNode = {
  type: 'ref' | 'block'
  id: string
  arguments: Record<string, unknown>
  depth: number
}

type FixtureDeclaration = {
  name: string
  id: string
  contractVersion: string
  kind: 'reference' | 'protected'
  canonicalArgument: string
  bracketArgument: string
  commentOnly: boolean
  fallbackCode: string
  legacySyntax: string
}

type FixtureCase = {
  name: string
  resourceKind: SFShortcodeResourceKind
  legacy: string
  expected: 'converted' | 'literal' | 'invalid' | 'unsupported' | 'over-limit'
  note: string
  canonical?: string
  nodes: FixtureNode[]
}

type FixtureStructuredCase = {
  name: string
  resourceKind: SFShortcodeResourceKind
  authority: 'both' | 'host'
  expected: 'invalid' | 'over-limit'
  note: string
  document: JSONContent
}

type Fixture = {
  version: string
  report: string
  declarations: FixtureDeclaration[]
  legacyAliases: Array<{ name: string, targetDeclaration: string, bracketArgument: string, note: string, legacySyntax: string }>
  recognizedLegacyUnsupported: Array<{ name: string, note: string, legacySyntax: string }>
  cases: FixtureCase[]
  structuredCases: FixtureStructuredCase[]
}

const fixture = JSON.parse(readFileSync(
  resolve(import.meta.dir, '../../../../contracts/fixtures/shortcode-legacy-conversion-v1.json'),
  'utf8'
)) as Fixture

let editor: Editor | undefined

afterEach(() => {
  editor?.destroy()
  editor = undefined
})

describe('Legacy shortcode conversion conformance', () => {
  it('uses the frozen shared grammar version and a classified conversion report', () => {
    expect(fixture.version).toBe(SF_SHORTCODE_TEXT_VERSION)
    expect(fixture.report).toBe('sforum.legacy-shortcode-conversion@1')
  })

  it('declares exactly eight shortcodes with deterministic fallback codes', () => {
    expect(fixture.declarations.map(item => item.name).sort()).toEqual([
      'category', 'comment', 'friend-links', 'login', 'only-author', 'reply', 'topic', 'user'
    ])
    const ids = fixture.declarations.map(item => item.id)
    expect(new Set(ids).size).toBe(ids.length)
    for (const declaration of fixture.declarations) {
      expect(declaration.contractVersion).toBe(`${declaration.id}@1`)
      expect(['reference', 'protected']).toContain(declaration.kind)
      expect(declaration.fallbackCode).toBeTruthy()
    }
    expect(fixture.declarations.find(item => item.name === 'only-author')?.commentOnly).toBe(true)
  })

  it('keeps topic-tag as the only legacy alias and password as the only unsupported legacy name', () => {
    expect(fixture.legacyAliases).toHaveLength(1)
    expect(fixture.legacyAliases[0]).toMatchObject({
      name: 'topic-tag',
      targetDeclaration: 'sforum-shortcodes.category',
      bracketArgument: 'tag_id'
    })
    expect(fixture.recognizedLegacyUnsupported).toHaveLength(1)
    expect(fixture.recognizedLegacyUnsupported[0].name).toBe('password')
    expect(fixture.declarations.map(item => item.name)).not.toContain('password')
    expect(fixture.declarations.map(item => item.name)).not.toContain('topic-tag')
    expect(fixture.declarations.map(item => item.name)).not.toContain('friend_links')
  })

  it('covers every classification at least once', () => {
    for (const classification of ['converted', 'literal', 'invalid', 'unsupported', 'over-limit']) {
      expect(fixture.cases.some(item => item.expected === classification)).toBe(true)
    }
  })

  it('matches the Host classification, canonical output, and fallback codes', () => {
    const codesByID = new Map(fixture.declarations.map(item => [item.id, item.fallbackCode]))
    for (const test of fixture.cases) {
      const inspection = inspectSForumShortcodeMarkdown(test.legacy, test.resourceKind)
      const combinedActivated = inspection.activated && inspection.withinBudgets

      if (test.expected === 'converted') {
        expect(combinedActivated, `${test.name}: expected conversion`).toBe(true)
        expect(test.canonical).toBeTruthy()
        const prepared = prepareSForumShortcodeMarkdown(test.legacy, test.resourceKind)
        editor = new Editor({
          extensions: createSFEditorExtensions({
            placeholder: 'body',
            maxCharacters: 100000,
            resourceKind: test.resourceKind
          }),
          content: prepared,
          contentType: 'markdown'
        })
        expect(summarizeShortcodes(editor.getJSON().content || [], 0)).toEqual(test.nodes)
        expect(editor.getMarkdown()).toBe(test.canonical)
        assertFallbackCodesPresent(editor.getHTML(), test.nodes, codesByID, test.name)
      } else {
        expect(combinedActivated, `${test.name}: must stay literal for ${test.expected}`).toBe(false)
      }
    }
  })

  it('rejects every structured case through the client inspection when the Host and client agree', () => {
    for (const test of fixture.structuredCases) {
      if (test.authority !== 'both') continue
      const inspection = inspectSForumShortcodeDocument(test.document, test.resourceKind)
      expect(inspection.valid, `${test.name}: client inspection must reject`).toBe(false)
    }
    // Host-only cases document that the client inspection is UX-only and never
    // an authority; the Host rejection is asserted by the Go conformance test.
    for (const test of fixture.structuredCases) {
      expect(['both', 'host']).toContain(test.authority)
      expect(['invalid', 'over-limit']).toContain(test.expected)
    }
  })
})

function assertFallbackCodesPresent(
  html: string,
  nodes: FixtureNode[],
  codesByID: Map<string, string>,
  caseName: string
) {
  const expected: string[] = []
  // Only top-level nodes surface a client fallback. Protected descendants are
  // never serialized into client HTML by design (fail-closed render hole).
  for (const node of nodes.filter(item => item.depth === 1)) {
    const code = codesByID.get(node.id)
    expect(code, `${caseName}: missing fallback code for ${node.id}`).toBeTruthy()
    if (code === 'shortcode.friend_links.omitted') continue
    expected.push(code!)
  }
  const actual = Array.from(html.matchAll(/data-fallback="([^"]+)"/g), match => match[1])
  expect(actual, `${caseName}: fallback codes in client HTML`).toEqual(expected)
  if (nodes.some(node => node.depth === 1 && codesByID.get(node.id) === 'shortcode.friend_links.omitted')) {
    // friend links must vanish silently; no fallback wrapper or label may render.
    expect(html).not.toContain('sf-editor-fallback')
  }
}

function summarizeShortcodes(nodes: JSONContent[], depth: number): FixtureNode[] {
  const result: FixtureNode[] = []
  for (const node of nodes) {
    let nextDepth = depth
    if (node.type === 'sforumShortcodeRef' || node.type === 'sforumShortcodeBlock') {
      nextDepth += 1
      result.push({
        type: node.type === 'sforumShortcodeBlock' ? 'block' : 'ref',
        id: String(node.attrs?.id || ''),
        arguments: node.attrs?.arguments || {},
        depth: nextDepth
      })
    }
    result.push(...summarizeShortcodes(node.content || [], nextDepth))
  }
  return result
}
