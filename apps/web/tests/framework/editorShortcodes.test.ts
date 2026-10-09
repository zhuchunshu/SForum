import { afterEach, describe, expect, it } from 'bun:test'
import { readFileSync } from 'node:fs'
import { resolve } from 'node:path'
import { Editor, type JSONContent } from '@tiptap/core'
import {
  createSFEditorExtensions,
  isExternalEditorMarkdownUpdate
} from '../../app/utils/sfEditor'
import {
  buildSForumProtectedTransaction,
  inspectSForumShortcodeDocument,
  inspectSForumShortcodeMarkdown,
  prepareSForumShortcodeMarkdown,
  sforumCommentShortcodeText,
  SF_SHORTCODE_BLOCK_NODE,
  SF_SHORTCODE_REF_NODE,
  SF_SHORTCODE_TEXT_VERSION,
  sforumProtectedNodeAttributes,
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

// 评论区「复制引用 Shortcode」的一键复制文本契约：必须与 referenceTokenizer 的
// canonical 输入格式完全一致，粘贴回原编辑器后能还原为 comment 引用节点。
describe('sforumCommentShortcodeText', () => {
  it('emits the canonical comment reference text', () => {
    expect(sforumCommentShortcodeText(123)).toBe('[comment comment_id="123"][/comment]')
  })

  it('round-trips through the editor tokenizer', () => {
    const text = sforumCommentShortcodeText(123)
    editor = new Editor({
      extensions: createSFEditorExtensions({ placeholder: 'body', maxCharacters: 10000, resourceKind: 'comment' }),
      content: prepareSForumShortcodeMarkdown(text, 'comment'),
      contentType: 'markdown'
    })
    const summary = summarizeShortcodes(editor.getJSON().content || [], 0)
    expect(summary).toEqual([
      { type: 'ref', id: 'sforum-shortcodes.comment', arguments: { commentId: 123 }, depth: 1 }
    ])
    expect(editor.getMarkdown()).toBe('[comment comment_id="123"][/comment]')
  })
})

// 受保护块插入的候选事务契约。Tiptap 对非链式 commands.command 无条件 dispatch，
// 回归边界是「校验前置 + ready 才派发」：任何失败路径都不产出可派发事务，
// 编辑器文档保持逐字节不变。
describe('protected insert transaction builder', () => {
  const replyAttrs = sforumProtectedNodeAttributes('reply', 'topic')

  function makeEditorForBuilder(content: string | JSONContent, resourceKind: SFShortcodeResourceKind = 'topic') {
    editor = new Editor({
      extensions: createSFEditorExtensions({ placeholder: 'body', maxCharacters: 10000, resourceKind }),
      content: content as never
    })
    return editor
  }

  function caretInto(editorInstance: Editor, needle: string) {
    let found = -1
    editorInstance.state.doc.descendants((node, pos) => {
      if (found >= 0) return false
      if (node.isText && node.text && node.text.includes(needle)) {
        found = pos + node.text.indexOf(needle)
        return false
      }
      return true
    })
    editorInstance.commands.setTextSelection(found + 1)
  }

  function build(resourceKind: SFShortcodeResourceKind = 'topic') {
    return buildSForumProtectedTransaction({
      state: editor!.state,
      attrs: replyAttrs!,
      resourceKind
    })
  }

  function expectInvalid(outcome: ReturnType<typeof build>) {
    if (outcome.status !== 'invalid') throw new Error(`expected an invalid outcome, got ${outcome.status}`)
    return outcome
  }

  it('rejects an empty caret paragraph with the empty reason and leaves the document untouched', () => {
    makeEditorForBuilder('<p>第一段</p><p></p>')
    const beforeJson = editor!.getJSON()
    editor!.commands.setTextSelection(editor!.state.doc.content.size - 1)
    const outcome = expectInvalid(build())
    expect(outcome.reasons).toEqual(['empty'])
    expect(editor!.getJSON()).toEqual(beforeJson)
  })

  it('treats a paragraph holding only hard breaks as an empty body', () => {
    makeEditorForBuilder('<p><br></p>')
    const beforeJson = editor!.getJSON()
    editor!.commands.setTextSelection(1)
    const outcome = expectInvalid(build())
    expect(outcome.reasons).toEqual(['empty'])
    expect(editor!.getJSON()).toEqual(beforeJson)
  })

  it('wraps a top-level paragraph caret into a ready transaction without touching the editor', () => {
    makeEditorForBuilder('<p>这是要隐藏的内容</p>')
    const beforeJson = editor!.getJSON()
    caretInto(editor!, '要隐藏')
    const outcome = build()
    expect(editor!.getJSON()).toEqual(beforeJson)
    expect(outcome.status).toBe('ready')
    if (outcome.status === 'ready') {
      editor!.view.dispatch(outcome.tr.scrollIntoView())
      expect(summarizeShortcodes(editor!.getJSON().content || [], 0)).toEqual([
        { type: 'block', id: 'sforum-shortcodes.reply', arguments: {}, depth: 1 }
      ])
    }
  })

  it('flags a blockquote caret as a new placement violation', () => {
    makeEditorForBuilder('<blockquote><p>引用内容</p></blockquote>')
    // 先设选区再快照：StarterKit TrailingNode 会在首次 dispatch 时给非 textblock
    // 结尾的文档补一个尾段落，把它纳入基线，才能断言 build 自身不动文档。
    caretInto(editor!, '引用')
    const beforeJson = editor!.getJSON()
    const outcome = expectInvalid(build())
    expect(outcome.reasons).toEqual(['placement'])
    expect(outcome.hadPlacementViolation).toBe(false)
    expect(editor!.getJSON()).toEqual(beforeJson)
  })

  it('distinguishes a pre-existing placement violation inside a poisoned document', () => {
    makeEditorForBuilder({
      type: 'doc',
      content: [
        { type: 'bulletList', content: [{ type: 'listItem', content: [
          { type: 'paragraph', content: [{ type: 'text', text: '列表' }] },
          { type: SF_SHORTCODE_REF_NODE, attrs: {
            id: 'sforum-shortcodes.topic',
            contractVersion: 'sforum-shortcodes.topic@1',
            arguments: { topicId: 2 }
          } }
        ] }] },
        { type: 'paragraph', content: [{ type: 'text', text: '需要隐藏的内容' }] }
      ]
    })
    caretInto(editor!, '需要隐藏')
    const outcome = expectInvalid(build())
    expect(outcome.reasons).toEqual(['placement'])
    expect(outcome.hadPlacementViolation).toBe(true)
  })

  it('flags crossing the 32-node budget without blaming the body', () => {
    makeEditorForBuilder({
      type: 'doc',
      content: [
        ...Array.from({ length: 32 }, () => ({
          type: SF_SHORTCODE_REF_NODE,
          attrs: {
            id: 'sforum-shortcodes.user',
            contractVersion: 'sforum-shortcodes.user@1',
            arguments: { userId: 7 }
          }
        })),
        { type: 'paragraph', content: [{ type: 'text', text: '需要隐藏的内容' }] }
      ]
    })
    caretInto(editor!, '需要隐藏')
    const outcome = expectInvalid(build())
    expect(outcome.reasons).toEqual(['count'])
  })

  it('flags crossing the 4-level nesting budget', () => {
    const nestedBlock = (inner: JSONContent): JSONContent => ({
      type: SF_SHORTCODE_BLOCK_NODE,
      attrs: { id: 'sforum-shortcodes.login', contractVersion: 'sforum-shortcodes.login@1', arguments: {} },
      content: [inner]
    })
    makeEditorForBuilder({
      type: 'doc',
      content: [nestedBlock(nestedBlock(nestedBlock(nestedBlock({
        type: 'paragraph', content: [{ type: 'text', text: '最深的内容' }]
      }))))]
    })
    caretInto(editor!, '最深')
    const outcome = expectInvalid(build())
    expect(outcome.reasons).toEqual(['depth'])
  })

  it('flags a comment-only block operating in the topic document', () => {
    makeEditorForBuilder({
      type: 'doc',
      content: [
        { type: SF_SHORTCODE_BLOCK_NODE, attrs: {
          id: 'sforum-shortcodes.only-author',
          contractVersion: 'sforum-shortcodes.only-author@1',
          arguments: {}
        }, content: [
          { type: 'paragraph', content: [{ type: 'text', text: '仅作者可见内容' }] }
        ] },
        { type: 'paragraph', content: [{ type: 'text', text: '需要隐藏的内容' }] }
      ]
    })
    caretInto(editor!, '需要隐藏')
    const outcome = expectInvalid(build())
    expect(outcome.reasons).toEqual(['commentOnly'])
  })

  it('builds a ready transaction for changing the kind of an existing block', () => {
    makeEditorForBuilder({
      type: 'doc',
      content: [
        { type: SF_SHORTCODE_BLOCK_NODE, attrs: {
          id: 'sforum-shortcodes.login',
          contractVersion: 'sforum-shortcodes.login@1',
          arguments: {}
        }, content: [
          { type: 'paragraph', content: [{ type: 'text', text: '秘密' }] }
        ] }
      ]
    })
    const outcome = buildSForumProtectedTransaction({
      state: editor!.state,
      attrs: replyAttrs!,
      resourceKind: 'topic',
      selection: {
        id: 'sforum-shortcodes.login',
        contractVersion: 'sforum-shortcodes.login@1',
        arguments: {},
        position: 0,
        protected: true
      }
    })
    expect(outcome.status).toBe('ready')
    if (outcome.status === 'ready') {
      editor!.view.dispatch(outcome.tr)
      expect(summarizeShortcodes(editor!.getJSON().content || [], 0)).toEqual([
        { type: 'block', id: 'sforum-shortcodes.reply', arguments: {}, depth: 1 }
      ])
    }
  })
})
