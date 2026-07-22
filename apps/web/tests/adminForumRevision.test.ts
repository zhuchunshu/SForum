import { describe, expect, test } from 'bun:test'
import { readFileSync } from 'node:fs'

const read = (path: string) => readFileSync(new URL(path, import.meta.url), 'utf8')

describe('admin forum revision workbench', () => {
  test('uses only canonical revision APIs and preserves the expected-revision token', () => {
    const composable = read('../app/composables/useAdminForumContent.ts')
    expect(composable).toContain('/topics/${topicId}/revisions/${revisionNo}/restore')
    expect(composable).toContain('/comments/${commentId}/revisions/${revisionNo}/restore')
    expect(composable).toContain('/topics/${topicId}/revisions/${revisionNo}/redact')
    expect(composable).toContain('/comments/${commentId}/revisions/${revisionNo}/redact')
    expect(composable).not.toContain('/admin/forum/content/topics/${topicId}/revisions')
    expect(composable).not.toContain('/admin/forum/content/comments/${commentId}/revisions')

    const page = read('../app/pages/admin/forum/content.vue')
    expect(page).toContain('expectedRevision: selected.value.currentRevision')
    expect(page).toContain("apiErrorReason(cause) === 'forum.revision_conflict'")
    expect(page).not.toContain('force overwrite')
  })

  test('keeps revision headers lazy and makes unavailable states explicit', () => {
    const timeline = read('../app/components/admin/forum/SFAdminForumRevisionTimeline.vue')
    const page = read('../app/pages/admin/forum/content.vue')
    expect(timeline).toContain("emit('select', revision)")
    expect(page).toContain('getTopicRevision')
    expect(page).toContain('getCommentRevision')
    expect(page).toContain('revision.redacted')
    expect(page).toContain('snapshotComplete')
    expect(page).toContain('selectedRevisionCanRestore')
  })

  test('renders diff through the reviewed dependency and protects redaction behind super_admin', () => {
    const diff = read('../app/components/admin/forum/SFAdminForumRevisionDiff.vue')
    const page = read('../app/pages/admin/forum/content.vue')
    const packageJSON = JSON.parse(read('../package.json'))
    expect(packageJSON.dependencies.diff).toBe('^9')
    expect(diff).toContain("import { diffLines } from 'diff'")
    expect(diff).toContain('break-all')
    expect(page).toContain("user.value?.status === 'active' && user.value.roleKeys.includes('super_admin')")
    expect(page).toContain("redactionConfirmation.value.trim() !== 'REDACT'")
  })

  test('ships both locale sets for timeline, restore, and irreversible redaction', () => {
    const zhCN = JSON.parse(read('../i18n/locales/zh-CN.json'))
    const enUS = JSON.parse(read('../i18n/locales/en-US.json'))
    for (const locale of [zhCN, enUS]) {
      expect(locale.admin.forum.content.history).toMatchObject({
        lazyHint: expect.any(String), restoreTitle: expect.any(String),
        redactionWarning: expect.any(String), redactionConfirmation: expect.any(String)
      })
    }
  })
})
