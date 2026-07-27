import { useForumApi } from '~/composables/forum/useForumApi'
import {
  forumTopicPath,
  type ForumCategoryGroup,
  type ForumTag,
  type ForumTopicList,
  type ForumTopicSummary
} from '~/utils/forum/forumTaxonomy'

const emptyTopicList = (): ForumTopicList => ({
  items: [],
  total: 0,
  page: 1,
  perPage: 8,
  hasMore: false
})

const RECOVERY_DATA_TIMEOUT_MS = import.meta.dev ? 700 : 800

export function useSystemErrorRecoveryData() {
  const forumApi = useForumApi()
  const { seoSettings } = useWebOptions()
  const topicUrlMode = computed(() => seoSettings.value.topicUrlMode)
  const readOptions = {
    timeout: RECOVERY_DATA_TIMEOUT_MS,
    serverInternal: import.meta.server
  }

  const { data: categoryGroups, pending: categoriesPending } = useAsyncData(
    'system-error-category-groups',
    async () => {
      try {
        return await forumApi.listCategoryGroups(readOptions)
      } catch {
        return [] as ForumCategoryGroup[]
      }
    },
    { default: () => [] as ForumCategoryGroup[] }
  )

  const { data: tags } = useAsyncData(
    'system-error-tags',
    async () => {
      try {
        return (await forumApi.listTags(readOptions)).filter(tag => tag.status === 'active')
      } catch {
        return [] as ForumTag[]
      }
    },
    { default: () => [] as ForumTag[] }
  )

  const { data: topicList } = useAsyncData(
    'system-error-topic-list',
    async () => {
      try {
        return await forumApi.listTopics({ page: 1 }, readOptions)
      } catch {
        return emptyTopicList()
      }
    },
    { default: emptyTopicList }
  )

  const categories = computed(() => categoryGroups.value.flatMap(group => group.categories || []))
  const totalTopics = computed(() => {
    const categoryTotal = categories.value.reduce((total, category) => total + category.topicCount, 0)
    return categoryTotal > 0 ? categoryTotal : topicList.value.total
  })
  const totalReplies = computed(() => categories.value.reduce((sum, category) => sum + (category.commentCount || 0), 0))
  const activeTags = computed(() => [...tags.value].sort((left, right) => right.topicCount - left.topicCount).slice(0, 12))
  const hotTopics = computed<ForumTopicSummary[]>(() => [...topicList.value.items]
    .sort((left, right) => {
      const replyDiff = right.commentCount - left.commentCount
      return replyDiff || right.id - left.id
    })
    .slice(0, 5))

  function topicTo(topic: ForumTopicSummary) {
    return forumTopicPath(topic, topicUrlMode.value)
  }

  return {
    categories,
    categoryGroups,
    categoriesPending,
    activeTags,
    hotTopics,
    totalTopics,
    totalReplies,
    topicUrlMode,
    topicTo
  }
}
