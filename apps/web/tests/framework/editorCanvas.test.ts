import { describe, expect, it } from 'bun:test'
import { readFileSync } from 'node:fs'

function source(path: string) {
  return readFileSync(new URL(path, import.meta.url), 'utf8')
}

describe('Forum Canvas editor surface', () => {
  const editor = source('../../app/components/SFEditor.vue')
  const toolbar = source('../../app/components/editor/SFEditorToolbar.vue')
  const styles = source('../../app/assets/css/sforum-components.css')
  const editorExtensions = source('../../app/utils/sfEditor.ts')

  it('keeps write and preview while removing source modes and the Unicode emoji picker', () => {
    expect(editor).toContain('<SFEditorToolbar')
    expect(toolbar).toContain("{ action: 'strike' as const")

    expect(toolbar).toContain("export type SFEditorViewMode = 'write' | 'preview'")
    expect(toolbar).toContain("{ value: 'write', label: '撰写' }")
    expect(toolbar).toContain("{ value: 'preview', label: '预览' }")
    expect(toolbar).not.toContain("value: 'markdown'")
    expect(toolbar).not.toContain("value: 'native'")
    expect(editor).toContain('sf-editor__preview')
    expect(editor).not.toContain('sf-editor__source')
    expect(editor).not.toContain('sf-editor__native')
    expect(editor).not.toContain('showEmojiPanel')
    expect(editor).not.toContain('sforumEditorEmojiItems')
    expect(toolbar).not.toContain('i-lucide-smile')
  })

  it('keeps quiet editor focus and the approved responsive canvas geometry', () => {
    expect(styles).not.toContain('.sf-editor:focus-within')
    expect(styles).toContain('padding: 42px 52px 64px;')
    expect(styles).toContain('padding: 28px 22px 52px;')
    expect(styles).toContain('.sf-editor__toolbar')
    expect(styles).toContain('overflow-x: auto;')
  })

  it('preserves historical sforumEmoji documents and trusted L2 loading', () => {
    expect(editorExtensions).toContain("name: 'sforumEmoji'")
    expect(editorExtensions).toContain('SForumEmoji')
    expect(editor).toContain('useTrustedEditorCatalog()')
    expect(editor).toContain('trustedExtensions: trusted')
  })
})
