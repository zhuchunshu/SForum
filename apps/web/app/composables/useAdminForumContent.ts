import type {
  AdminForumCommentDetail,
  AdminForumContentFilters,
  AdminForumContentKind,
  AdminForumContentList,
  AdminForumTopicDetail,
  ForumRevisionList
} from '~/utils/adminForumContent'
import { adminForumContentPath } from '~/utils/adminForumContent'

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

  // M5 仅复用受保护 read model；时间线与操作 UI 保留给 M6。
  function listTopicRevisions(topicId: number) {
    return request<ForumRevisionList>(`/topics/${topicId}/revisions`)
  }

  function listCommentRevisions(commentId: number) {
    return request<ForumRevisionList>(`/comments/${commentId}/revisions`)
  }

  return { list, getTopic, getComment, listTopicRevisions, listCommentRevisions }
}
