import {
  buildForumTopicQuery,
  type ForumCategoryGroup,
  type ForumTag,
  type ForumTopicDetail,
  type ForumTopicFilters,
  type ForumTopicList
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

  return {
    listCategoryGroups,
    listTags,
    listTopics,
    getTopic
  }
}

function pathWithQuery(path: string, query: Record<string, string>) {
  const params = new URLSearchParams(query)
  const queryString = params.toString()
  return queryString ? `${path}?${queryString}` : path
}
