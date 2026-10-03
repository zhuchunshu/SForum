import { afterEach, describe, expect, it } from 'bun:test'
import { readFileSync } from 'node:fs'
import { resolve } from 'node:path'
import { Editor, type JSONContent } from '@tiptap/core'
import {
  createSFEditorExtensions,
  isExternalEditorMarkdownUpdate
} from '../../app/utils/sfEditor'
import {
  inspectSForumShortcodeMarkdown,
  prepareSForumShortcodeMarkdown,
  SF_SHORTCODE_BLOCK_NODE,
  SF_SHORTCODE_REF_NODE,
  SF_SHORTCODE_TEXT_VERSION,
  type SFShortcodeResourceKind
} from '../../app/utils/editor/shortcodes'

type FixtureSummary = {
  type: 'ref' | 'block'
  id: string
  arguments: Record<string, unknown>
  depth: number
}

type FixtureCase = {
  name: string
  resourceKind: SFShortcodeResourceKind
  markdown: string
  activated: boolean
  canonical?: string
  nodes: FixtureSummary[]
}

const fixture = JSON.parse(readFileSync(
  resolve(import.meta.dir, '../../../../contracts/fixtures/shortcode-text-v1.json'),
  'utf8'
)) as { version: string, cases: FixtureCase[] }

let editor: Editor | undefined

afterEach(() => {
  editor?.destroy()
  editor = undefined
})

describe('Host shortcode text conformance', () => {
  it('uses the frozen shared fixture version', () => {
    expect(fixture.version).toBe(SF_SHORTCODE_TEXT_VERSION)
  })

  for (const test of fixture.cases) {
    it(test.name, () => {
      const inspection = inspectSForumShortcodeMarkdown(test.markdown, test.resourceKind)
      expect(inspection.activated && inspection.withinBudgets).toBe(test.activated)

      const prepared = prepareSForumShortcodeMarkdown(test.markdown, test.resourceKind)
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
      if (test.activated) {
        expect(editor.getMarkdown()).toBe(test.canonical)
      }
    })
  }

  it('does not serialize protected descendants into fallback HTML', () => {
    const secret = 'M2_CLIENT_PROTECTED_SECRET'
    editor = new Editor({
      extensions: createSFEditorExtensions({
        placeholder: 'body',
        maxCharacters: 10000,
        resourceKind: 'comment'
      }),
      content: `[login]\n${secret}\n[/login]`,
      contentType: 'markdown'
    })

    expect(editor.getJSON().content?.[0]?.content?.[0]?.content?.[0]?.text).toBe(secret)
    expect(editor.getHTML()).not.toContain(secret)
    expect(editor.getHTML()).toContain('shortcode.protected.unavailable')
  })

  it('converts canonical paste and the topic-tag typed alias into reference nodes', () => {
    editor = new Editor({
      extensions: createSFEditorExtensions({ placeholder: 'body', maxCharacters: 10000, resourceKind: 'topic' }),
      content: ''
    })
    editor.view.dispatch(editor.state.tr
      .insertText('[user user_id="7"][/user]')
      .setMeta('uiEvent', 'paste'))
    expect(summarizeShortcodes(editor.getJSON().content || [], 0)).toContainEqual({
      type: 'ref', id: 'sforum-shortcodes.user', arguments: { userId: 7 }, depth: 1
    })

    editor.commands.clearContent()
    const typedAlias = '[topic-tag tag_id="9"][/topic-tag]'
    editor.commands.insertContent(typedAlias.slice(0, -1))
    const position = editor.state.selection.from
    const handled = editor.view.someProp('handleTextInput', handler => handler(
      editor!.view, position, position, typedAlias.slice(-1)
    ))
    expect(handled).toBe(true)
    expect(summarizeShortcodes(editor.getJSON().content || [], 0)).toEqual([{
      type: 'ref', id: 'sforum-shortcodes.category', arguments: { categoryId: 9 }, depth: 1
    }])
    expect(editor.getMarkdown().trim()).toBe('[category category_id="9"][/category]')
  })

  it('does not re-import the editor own escaped bracket markdown', () => {
    editor = new Editor({
      extensions: createSFEditorExtensions({ placeholder: 'body', maxCharacters: 10000, resourceKind: 'topic' }),
      content: ''
    })
    editor.commands.insertContent('[reply]内容[/reply]')

    const emittedMarkdown = editor.getMarkdown()
    expect(editor.getText()).toBe('[reply]内容[/reply]')
    expect(emittedMarkdown).toBe('\\[reply\\]内容\\[/reply\\]')
    expect(prepareSForumShortcodeMarkdown(emittedMarkdown, 'topic')).toStartWith('&#91;')
    expect(isExternalEditorMarkdownUpdate(
      emittedMarkdown,
      emittedMarkdown,
      emittedMarkdown
    )).toBe(false)
  })

  it('restores all five reference nodes from native editor-document content for re-edit', () => {
    const markdown = [
      '[user user_id="1"][/user]',
      '[topic topic_id="2"][/topic]',
      '[comment comment_id="3"][/comment]',
      '[category category_id="4"][/category]',
      '[friend-links][/friend-links]'
    ].join('\n\n')
    editor = new Editor({
      extensions: createSFEditorExtensions({ placeholder: 'body', maxCharacters: 10000, resourceKind: 'comment' }),
      content: markdown,
      contentType: 'markdown'
    })
    const native = editor.getJSON()
    editor.destroy()
    editor = new Editor({
      extensions: createSFEditorExtensions({ placeholder: 'body', maxCharacters: 10000, resourceKind: 'comment' }),
      content: native
    })
    expect(summarizeShortcodes(editor.getJSON().content || [], 0).map(item => item.id)).toEqual([
      'sforum-shortcodes.user', 'sforum-shortcodes.topic', 'sforum-shortcodes.comment',
      'sforum-shortcodes.category', 'sforum-shortcodes.friend-links'
    ])
    expect(editor.getMarkdown()).toBe(markdown)
  })
})

function summarizeShortcodes(nodes: JSONContent[], depth: number): FixtureSummary[] {
  const result: FixtureSummary[] = []
  for (const node of nodes) {
    let nextDepth = depth
    if (node.type === SF_SHORTCODE_REF_NODE || node.type === SF_SHORTCODE_BLOCK_NODE) {
      nextDepth += 1
      result.push({
        type: node.type === SF_SHORTCODE_BLOCK_NODE ? 'block' : 'ref',
        id: String(node.attrs?.id || ''),
        arguments: node.attrs?.arguments || {},
        depth: nextDepth
      })
    }
    result.push(...summarizeShortcodes(node.content || [], nextDepth))
  }
  return result
}
