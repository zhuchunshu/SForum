import {
  buildForumCommentQuery,
  buildForumTopicQuery,
  type ForumCategoryGroup,
  type ForumComment,
  type ForumCommentList,
  type ForumCommentListQuery,
  type ForumContentInput,
  type ForumTag,
  type ForumTopicAction,
  type ForumTopicActionKey,
  type ForumTopicDetail,
  type ForumTopicFilters,
  type ForumTopicList,
  type ForumTopicUpdateInput
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

  function getTopic(topicId: number) {
    return request<ForumTopicDetail>(`/topics/${topicId}`)
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

  function createTopic(input: ForumContentInput & {
    title: string
    categorySlug?: string
    tagSlugs?: string[]
  }) {
    return request<ForumTopicDetail>('/topics', {
      method: 'POST',
      body: input
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

  return {
    listCategoryGroups,
    listTags,
    listTopics,
    getTopic,
    listTopicComments,
    createTopicComment,
    updateComment,
    deleteComment,
    listCommentReplies,
    createTopic,
    updateTopic,
    deleteTopic,
    applyTopicAction
  }
}

function pathWithQuery(path: string, query: Record<string, string>) {
  const params = new URLSearchParams(query)
  const queryString = params.toString()
  return queryString ? `${path}?${queryString}` : path
}
