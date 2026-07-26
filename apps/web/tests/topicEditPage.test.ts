import { describe, expect, test } from 'bun:test'
import { readFileSync } from 'node:fs'

const editPage = readFileSync(
  new URL('../app/components/SFTopicEditPage.vue', import.meta.url),
  'utf8'
)

describe('standalone topic edit page behavior contract', () => {
  test('resets all topic-scoped editor and feedback state when the route is reused', () => {
    expect(editPage).toContain('editorPayload.value = null')
    expect(editPage).toContain("submitState.value = 'idle'")
    expect(editPage).toContain("errorMessage.value = ''")
    expect(editPage).toContain("conflictMessage.value = ''")
    expect(editPage).toContain('fieldErrors.value = {}')
    expect(editPage).toContain('awaitingEditorBaseline.value = true')
    expect(editPage).toContain(
      "next.id !== prev?.id || next.currentRevision !== prev?.currentRevision"
    )
  })

  test('treats the staff reason as an unsaved edit and blocks no-op revisions', () => {
    expect(editPage).toMatch(
      /const currentSignature = computed\(\(\) => JSON\.stringify\(\{[\s\S]*staffReason: staffReason\.value/
    )
    expect(editPage).toContain('|| !hasUnsavedChanges.value')
    expect(editPage).toContain("return window.confirm(t('composer.editLeaveConfirm'))")
    expect(editPage).toContain('|| !hasUnsavedChanges.value')
  })

  test('keeps revision, author-reason, content, taxonomy and canonical-return contracts', () => {
    expect(editPage).toContain('expectedRevision: topic.value.currentRevision')
    expect(editPage).toContain('editingAnotherAuthor.value && !reason')
    expect(editPage).toContain('runeLength(reason) > editReasonMaxRunes')
    expect(editPage).toContain("t('composer.editReasonTooLong'")
    expect(editPage).toContain('v-if="staffReasonError"')
    expect(editPage).toContain('tagSlugs: tagDraft.value')
    expect(editPage).toContain('content')
    expect(editPage).toContain(':initial-content="editorInitialContent"')
    expect(editPage).toContain(
      ":permission-value-label=\"t('composer.settings.editPermissionValue')\""
    )
    expect(editPage).toContain(
      'forumTopicPath(updated, topicUrlMode.value)'
    )
  })

  test('keeps revision conflicts persistent and separate from field errors', () => {
    expect(editPage).toContain(
      "apiErrorReason(error) === 'forum.revision_conflict'"
    )
    expect(editPage).toContain("conflictMessage.value = t('composer.editConflict')")
    expect(editPage).toContain('fieldErrors.value = apiErrorFields(error)')
    expect(editPage).toContain('v-if="conflictMessage"')
  })
})
