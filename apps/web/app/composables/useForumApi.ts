import {
  buildForumCommentQuery,
  buildForumSearchQuery,
  buildForumTopicQuery,
  forumCommentExtensionActionRequest,
  forumTopicExtensionActionRequest,
  type ForumCategoryGroup,
  type ForumAuthorReviewItem,
  type ForumComment,
  type ForumCommentExtensionAction,
  type ForumCommentList,
  type ForumCommentListQuery,
  type ForumContentInput,
  type ForumTag,
  type ForumTopicAction,
  type ForumTopicActionKey,
  type ForumTopicDetail,
  type ForumTopicExtensionAction,
  type ForumComposerToolbarAction,
  type ForumTopicFilters,
  type ForumTopicCreateInput,
  type ForumTopicList,
  type ForumTopicSearchFilters,
  type ForumTopicUpdateInput,
  forumTopicExtensionActionRequestPath
} from '~/utils/forumTaxonomy'

export function useForumApi() {
  const { request } = useApiClient()

  function listCategoryGroups() {
    return request<ForumCategoryGroup[]>('/category-groups')
  }

  function listTags() {
    return request<ForumTag[]>('/tags')
  }

  function listTopics(filters: ForumTopicFilters = {}) {
    return request<ForumTopicList>(pathWithQuery('/topics', buildForumTopicQuery(filters)))
  }

  // 关键词检索走专用搜索端点（Meilisearch），避免 topics 列表的 ILIKE 全表扫描。
  function searchTopics(filters: ForumTopicSearchFilters) {
    return request<ForumTopicList>(pathWithQuery('/search', buildForumSearchQuery(filters)))
  }

  function getTopic(topicId: number) {
    return request<ForumTopicDetail>(`/topics/${topicId}`)
  }

  function listAuthorReviewItems() {
    return request<{ items: ForumAuthorReviewItem[] }>('/me/content-review')
  }

  // 按 slug 查询主题：仅 "纯 slug" URL 模式使用，对应后端 GET /topics/by-slug/:slug。
  function getTopicBySlug(slug: string) {
    return request<ForumTopicDetail>(`/topics/by-slug/${encodeURIComponent(slug)}`)
  }

  function listTopicComments(topicId: number, query: ForumCommentListQuery = {}) {
    return request<ForumCommentList>(pathWithQuery(`/topics/${topicId}/comments`, buildForumCommentQuery(query)))
  }

  function createTopicComment(topicId: number, content: ForumContentInput, parentId?: number | null) {
    return request<ForumComment>(`/topics/${topicId}/comments`, {
      method: 'POST',
      body: { parentId: parentId ?? null, content }
    })
  }

  function updateComment(commentId: number, content: ForumContentInput) {
    return request<ForumComment>(`/comments/${commentId}`, {
      method: 'PATCH',
      body: { content }
    })
  }

  function deleteComment(commentId: number) {
    return request<ForumComment>(`/comments/${commentId}`, { method: 'DELETE' })
  }

  function listCommentReplies(commentId: number) {
    return request<ForumComment[]>(`/comments/${commentId}/replies`)
  }

  function createTopic(input: ForumTopicCreateInput) {
    const {
      rawContent,
      sourceFormat,
      editorType,
      editorVersion,
      ...topicInput
    } = input

    return request<ForumTopicDetail>('/topics', {
      method: 'POST',
      body: {
        ...topicInput,
        content: {
          rawContent,
          sourceFormat,
          editorType,
          editorVersion
        }
      }
    })
  }

  function updateTopic(topicId: number, input: ForumTopicUpdateInput) {
    return request<ForumTopicDetail>(`/topics/${topicId}`, {
      method: 'PATCH',
      body: input
    })
  }

  function deleteTopic(topicId: number) {
    return request<ForumTopicDetail>(`/topics/${topicId}`, { method: 'DELETE' })
  }

  function applyTopicAction(topicId: number, action: ForumTopicActionKey) {
    return request<ForumTopicAction>(`/topics/${topicId}/${action}`, { method: 'POST' })
  }

  function applyTopicExtensionAction(topicId: number, action: ForumTopicExtensionAction) {
    const input = forumTopicExtensionActionRequest(action, topicId)
    if (!input) {
      throw new Error('Invalid topic extension action')
    }
    return request<unknown>(input.path, {
      method: input.method,
      body: input.body
    })
  }

  function applyCommentExtensionAction(
    topicId: number,
    commentId: number,
    action: ForumCommentExtensionAction
  ) {
    const input = forumCommentExtensionActionRequest(action, topicId, commentId)
    if (!input) {
      throw new Error('Invalid comment extension action')
    }
    return request<unknown>(input.path, {
      method: input.method,
      body: input.body
    })
  }

  function listComposerToolbarActions() {
    return request<ForumComposerToolbarAction[]>('/composer/toolbar')
  }

  function applyComposerToolbarAction(action: ForumComposerToolbarAction) {
    const path = forumTopicExtensionActionRequestPath(action)
    if (!path) {
      throw new Error('Invalid composer toolbar action')
    }
    return request<unknown>(path, {
      method: action.method,
      body: {}
    })
  }

  return {
    listCategoryGroups,
    listTags,
    listTopics,
    searchTopics,
    listAuthorReviewItems,
    getTopic,
    getTopicBySlug,
    listTopicComments,
    createTopicComment,
    updateComment,
    deleteComment,
    listCommentReplies,
    createTopic,
    updateTopic,
    deleteTopic,
    applyTopicAction,
    applyTopicExtensionAction,
    applyCommentExtensionAction,
    listComposerToolbarActions,
    applyComposerToolbarAction
  }
}

function pathWithQuery(path: string, query: Record<string, string>) {
  const params = new URLSearchParams(query)
  const queryString = params.toString()
  return queryString ? `${path}?${queryString}` : path
}
