import type {
  AdminForumCommentDetail,
  AdminForumContentFilters,
  AdminForumContentKind,
  AdminForumContentList,
  AdminForumTopicDetail,
  ForumRevisionDetail,
  ForumRevisionList
} from '~/utils/admin/adminForumContent'
import { adminForumContentPath } from '~/utils/admin/adminForumContent'

export function useAdminForumContent() {
  const { request } = useApiClient()

  function list(kind: AdminForumContentKind, filters: AdminForumContentFilters, after?: string) {
    return request<AdminForumContentList>(adminForumContentPath(kind, filters, after))
  }

  function getTopic(topicId: number) {
    return request<AdminForumTopicDetail>(`/admin/forum/content/topics/${topicId}`)
  }

  function getComment(commentId: number) {
    return request<AdminForumCommentDetail>(`/admin/forum/content/comments/${commentId}`)
  }

  function listTopicRevisions(topicId: number, after?: string) {
    return request<ForumRevisionList>(revisionPath('topics', topicId, after))
  }

  function listCommentRevisions(commentId: number, after?: string) {
    return request<ForumRevisionList>(revisionPath('comments', commentId, after))
  }

  function getTopicRevision(topicId: number, revisionNo: number) {
    return request<ForumRevisionDetail>(`/topics/${topicId}/revisions/${revisionNo}`)
  }

  function getCommentRevision(commentId: number, revisionNo: number) {
    return request<ForumRevisionDetail>(`/comments/${commentId}/revisions/${revisionNo}`)
  }

  function restoreTopicRevision(topicId: number, revisionNo: number, payload: { expectedRevision: number, reason: string }) {
    return request<AdminForumTopicDetail>(`/topics/${topicId}/revisions/${revisionNo}/restore`, { method: 'POST', body: payload })
  }

  function restoreCommentRevision(commentId: number, revisionNo: number, payload: { expectedRevision: number, reason: string }) {
    return request<AdminForumCommentDetail>(`/comments/${commentId}/revisions/${revisionNo}/restore`, { method: 'POST', body: payload })
  }

  function redactTopicRevision(topicId: number, revisionNo: number, payload: { expectedRevision: number, reason: string, confirmation: 'REDACT' }) {
    return request<{ redacted: boolean }>(`/topics/${topicId}/revisions/${revisionNo}/redact`, { method: 'POST', body: payload })
  }

  function redactCommentRevision(commentId: number, revisionNo: number, payload: { expectedRevision: number, reason: string, confirmation: 'REDACT' }) {
    return request<{ redacted: boolean }>(`/comments/${commentId}/revisions/${revisionNo}/redact`, { method: 'POST', body: payload })
  }

  return {
    list, getTopic, getComment,
    listTopicRevisions, listCommentRevisions, getTopicRevision, getCommentRevision,
    restoreTopicRevision, restoreCommentRevision, redactTopicRevision, redactCommentRevision
  }
}

function revisionPath(target: 'topics' | 'comments', targetId: number, after?: string) {
  const query = after ? `?after=${encodeURIComponent(after)}` : ''
  return `/${target}/${targetId}/revisions${query}`
}
