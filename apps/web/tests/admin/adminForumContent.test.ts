import { describe, expect, test } from 'bun:test'
import { readFileSync } from 'node:fs'

import { adminPageDefinitions, adminSidebarNavigation, type AdminNavigationFolderEntry } from '../../app/config/adminModules'
import { adminForumContentPath, buildAdminForumContentQuery } from '../../app/utils/admin/adminForumContent'

describe('admin forum content workbench', () => {
  test('registers the permission-aware Forum navigation entry', () => {
    const page = adminPageDefinitions.find(item => item.id === '/forum/content')
    expect(page?.requiredPermissions).toEqual(['topic.edit_any', 'topic.revision.view_any', 'post.edit_any', 'post.revision.view_any'])
    expect(page?.permissionMode).toBe('any')
    const forum = adminSidebarNavigation.flat().find((entry): entry is AdminNavigationFolderEntry => entry.type === 'folder' && entry.labelKey === 'admin.nav.forum')
    expect(forum?.children.some(entry => entry.type === 'page' && entry.pageId === '/forum/content')).toBe(true)
  })

  test('builds only server-side indexed content filters and opaque cursors', () => {
    expect(buildAdminForumContentQuery({ status: 'hidden', authorUsername: 'moderator', topicID: 42, perPage: 50 }, 'opaque-token')).toEqual({ after: 'opaque-token', status: 'hidden', authorUsername: 'moderator', topicID: '42', perPage: '50' })
    expect(adminForumContentPath('comments', { topicID: 42, perPage: 20 }, 'opaque-token')).toBe('/admin/forum/content/comments?after=opaque-token&topicID=42&perPage=20')
  })

  test('uses a non-empty Select item for all statuses while omitting that filter from the API request', () => {
    const page = readFileSync(new URL('../../app/pages/admin/forum/content.vue', import.meta.url), 'utf8')
    expect(page).toContain("const ALL_STATUS_VALUE = '__all__'")
    expect(page).toContain("status: filters.status === ALL_STATUS_VALUE ? '' : filters.status")
    expect(page).toContain('value-key="value" label-key="label"')
  })

  test('uses admin read models and canonical PATCH editors without admin mutation routes', () => {
    const composable = readFileSync(new URL('../../app/composables/admin/useAdminForumContent.ts', import.meta.url), 'utf8')
    const page = readFileSync(new URL('../../app/pages/admin/forum/content.vue', import.meta.url), 'utf8')
    const commentEditor = readFileSync(new URL('../../app/components/admin/forum/SFAdminForumCommentEditor.vue', import.meta.url), 'utf8')
    const topicEditor = readFileSync(new URL('../../app/components/forum/SFTopicEditor.vue', import.meta.url), 'utf8')
    expect(composable).toContain('/admin/forum/content/')
    expect(composable).not.toContain("method: 'PATCH'")
    expect(page).toContain('<SFTopicEditor')
    expect(commentEditor).toContain('<LazySFEditor')
    expect(commentEditor).toContain('forumApi.updateComment')
    expect(commentEditor).toContain('props.comment.currentRevision')
    expect(commentEditor).toContain('forumEditorInitialContent')
    expect(commentEditor).toContain(':initial-content="editorInitialContent"')
    expect(commentEditor).not.toContain('props.comment.content.rawContent')
    expect(topicEditor).toContain('expectedRevision: props.topic.currentRevision')
    expect(topicEditor).toContain('reason: reason || undefined')
    expect(topicEditor).toContain('forumEditorInitialContent')
    expect(topicEditor).toContain(':initial-content="editorInitialContent"')
    expect(topicEditor).toContain('forumContentFromEditorPayload')
    expect(topicEditor).not.toContain('props.topic.content.rawContent')
  })

  test('keeps cross-author reasons and revision conflicts visible without force overwrite', () => {
    const page = readFileSync(new URL('../../app/pages/admin/forum/content.vue', import.meta.url), 'utf8')
    const zhCN = JSON.parse(readFileSync(new URL('../../i18n/locales/zh-CN.json', import.meta.url), 'utf8'))
    const enUS = JSON.parse(readFileSync(new URL('../../i18n/locales/en-US.json', import.meta.url), 'utf8'))
    expect(page).toContain('requiresReason')
    expect(page).toContain('conflictTitle')
    expect(page).toContain('reloadLatest')
    expect(page).not.toContain('force overwrite')
    expect(zhCN.admin.forum.content).toMatchObject({ reasonRequired: expect.any(String), conflictTitle: expect.any(String) })
    expect(enUS.admin.forum.content).toMatchObject({ reasonRequired: expect.any(String), conflictTitle: expect.any(String) })
  })
})
