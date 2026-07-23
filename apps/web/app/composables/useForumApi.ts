import {
  buildForumCommentQuery,
  buildForumSearchQuery,
  buildForumTopicQuery,
  forumCommentExtensionActionRequest,
  forumTopicExtensionActionRequest,
  type ForumCategoryGroup,
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

  type ReadOptions = {
    timeout?: number
    serverInternal?: boolean
  }

  function listCategoryGroups(options: ReadOptions = {}) {
    return request<ForumCategoryGroup[]>('/category-groups', options)
  }

  function listTags(options: ReadOptions = {}) {
    return request<ForumTag[]>('/tags', options)
  }

  function listTopics(filters: ForumTopicFilters = {}, options: ReadOptions = {}) {
    return request<ForumTopicList>(pathWithQuery('/topics', buildForumTopicQuery(filters)), options)
  }

  // 关键词检索走选定 search.provider，默认站内引擎支持中英文全文与模糊搜索。
  function searchTopics(filters: ForumTopicSearchFilters) {
    return request<ForumTopicList>(pathWithQuery('/search', buildForumSearchQuery(filters)))
  }

  function getTopic(topicId: number) {
    return request<ForumTopicDetail>(`/topics/${topicId}`)
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

  function updateComment(commentId: number, content: ForumContentInput, expectedRevision: number, reason?: string) {
	return request<ForumComment>(`/comments/${commentId}`, {
	  method: 'PATCH',
	  body: { content, expectedRevision, ...(reason ? { reason } : {}) }
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
      attachmentIds,
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
          editorVersion,
          attachmentIds
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
