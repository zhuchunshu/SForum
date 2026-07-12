<script setup lang="ts">
import {
  buildForumHomeQuery,
  forumHomeFeedKey,
  hasReachedForumHomeEnd,
  isForumHomeRequestCurrent,
  parseForumHomeQuery,
  type ForumHomeFilters,
  type ForumHomeRequestToken
} from '~/utils/forumHome'
import {
  forumTopicPath,
  type ForumCategoryGroup,
  type ForumTag,
  type ForumTopicList,
  type ForumTopicSummary
} from '~/utils/forumTaxonomy'

const SEARCH_DEBOUNCE_MS = 300

const { t } = useI18n()
const route = useRoute()
const router = useRouter()
const localePath = useLocalePath()
const { seoSettings } = useWebOptions()
const forumApi = useForumApi()
const { can } = usePermissions()
// 主题 extension settings（后台「主题设置」自定义页）；非 core web_options
const {
  homeNotice,
  homeEmptyTitle,
  homeEmptyDescription,
  rightRailEnabled,
  rightRailHotLimit,
  rightRailTagLimit
} = useActiveThemeSettings()

useSForumSeo(computed(() => ({
  type: 'home',
  path: '/',
  description: t('home.metaDescription', { siteName: seoSettings.value.seoSiteName }),
  public: true,
  noindex: Object.keys(route.query).length > 0
})))

const topicUrlMode = computed(() => seoSettings.value.topicUrlMode)
const committedFilters = computed(() => parseForumHomeQuery(route.query))
const activeFeedKey = computed(() => forumHomeFeedKey(committedFilters.value))
const topicDataKey = computed(() => `forum-home-topics:${activeFeedKey.value}`)
const searchDraft = ref(committedFilters.value.query)
const selectedCategorySlug = computed(() => committedFilters.value.categorySlug)
const selectedTagSlug = computed(() => committedFilters.value.tagSlug)
const canCreateTopic = computed(() => can(FORUM_PERMISSIONS.topicCreate))
const renderedAt = useState<number>('forum-home-rendered-at', () => Date.now())
let feedGeneration = 0

const emptyTopicList = (): ForumTopicList => ({
  items: [],
  total: 0,
  page: 1,
  perPage: 20
})

const { data: categoryGroups, pending: categoriesPending } = await useAsyncData(
  'forum-home-category-groups',
  () => forumApi.listCategoryGroups(),
  { default: () => [] as ForumCategoryGroup[] }
)

const { data: activeTags, pending: tagsPending } = await useAsyncData(
  'forum-home-tags',
  async () => (await forumApi.listTags()).filter((tag) => tag.status === 'active'),
  { default: () => [] as ForumTag[] }
)

function loadTopicPage(page: number, filters: ForumHomeFilters = committedFilters.value) {
  if (filters.query) {
    return forumApi.searchTopics({
      query: filters.query,
      categorySlug: filters.categorySlug,
      tagSlug: filters.tagSlug,
      page
    })
  }

  return forumApi.listTopics({
    categorySlug: filters.categorySlug,
    tagSlug: filters.tagSlug,
    page
  })
}

const {
  data: topicList,
  pending: topicsPending,
  error: topicsError,
  refresh: refreshTopics
} = await useAsyncData(
  topicDataKey,
  async () => {
    const filters = { ...committedFilters.value }
    return loadTopicPage(1, filters)
  },
  {
    default: emptyTopicList
  }
)

const loadedTopics = useState<ForumTopicSummary[]>('forum-home-loaded-topics', () => topicList.value.items)
const loadedTopicTotal = useState<number>('forum-home-topic-total', () => topicList.value.total)
const loadedFeedKey = useState<string>('forum-home-loaded-feed-key', () => activeFeedKey.value)
const nextPage = ref(2)
const isLoadingMore = ref(false)
const loadMoreError = ref('')
const loadMoreTrigger = ref<HTMLElement | null>(null)
const hasLoadedAllPages = ref(false)

const categories = computed(() => categoryGroups.value.flatMap((group) => group.categories || []))
const topics = computed(() => loadedTopics.value)
const hasMoreTopics = computed(() => !hasLoadedAllPages.value && topics.value.length < loadedTopicTotal.value)
const hasActiveFilters = computed(() => Boolean(
  committedFilters.value.query
  || committedFilters.value.categorySlug
  || committedFilters.value.tagSlug
))
const selectedCategory = computed(() => categories.value.find(
  category => category.slug === selectedCategorySlug.value
))
const feedTitle = computed(() => selectedCategory.value?.name || t('home.allTopics'))
const emptyTitle = computed(() => {
  if (hasActiveFilters.value) {
    return t('home.emptyState.title')
  }
  return homeEmptyTitle.value || t('home.emptyState.title')
})
const emptyDescription = computed(() => {
  if (hasActiveFilters.value) {
    return t('home.emptyState.filteredDescription')
  }
  return homeEmptyDescription.value || t('home.emptyState.description')
})
const totalTopics = computed(() => {
  const categoryTotal = categories.value.reduce((total, category) => total + category.topicCount, 0)
  if (categoryTotal > 0) {
    return categoryTotal
  }
  return hasActiveFilters.value ? 0 : loadedTopicTotal.value
})
// 右栏热帖：当前已加载主题按回复数排序，条数来自主题设置
const hotTopics = computed(() => {
  return [...topics.value]
    .sort((left, right) => {
      const replyDiff = right.commentCount - left.commentCount
      if (replyDiff !== 0) {
        return replyDiff
      }
      return right.id - left.id
    })
    .slice(0, rightRailHotLimit.value)
})
const totalReplies = computed(() => {
  const fromCategories = categories.value.reduce((sum, category) => sum + (category.commentCount || 0), 0)
  if (fromCategories > 0) {
    return fromCategories
  }
  return topics.value.reduce((sum, topic) => sum + topic.commentCount, 0)
})
const railTags = computed(() => {
  return [...activeTags.value]
    .sort((left, right) => {
      const countDiff = right.topicCount - left.topicCount
      if (countDiff !== 0) {
        return countDiff
      }
      return left.name.localeCompare(right.name)
    })
    .slice(0, rightRailTagLimit.value)
})
function replaceLoadedTopics(list: ForumTopicList) {
  const seen = new Set<number>()
  loadedTopics.value = list.items.filter((topic) => {
    if (seen.has(topic.id)) {
      return false
    }
    seen.add(topic.id)
    return true
  })
  loadedTopicTotal.value = list.total
  loadedFeedKey.value = activeFeedKey.value
  nextPage.value = Math.max(2, list.page + 1)
  loadMoreError.value = ''
  hasLoadedAllPages.value = list.items.length >= list.total || list.items.length < list.perPage
}

function shouldIgnoreClientEmptyHydration(list: ForumTopicList) {
  return import.meta.client
    && activeFeedKey.value === loadedFeedKey.value
    && loadedTopics.value.length > 0
    && loadedTopicTotal.value > 0
    && list.items.length === 0
    && list.total === 0
}

watch(topicList, (list) => {
  if (shouldIgnoreClientEmptyHydration(list)) {
    return
  }
  replaceLoadedTopics(list)
}, { immediate: true })

watch(activeFeedKey, () => {
  feedGeneration += 1
  nextPage.value = 2
  isLoadingMore.value = false
  loadMoreError.value = ''
  hasLoadedAllPages.value = false
}, { flush: 'sync' })

async function loadMoreTopics(forceRetry = false) {
  if (topicsPending.value || isLoadingMore.value || !hasMoreTopics.value) {
    return
  }
  if (loadMoreError.value && !forceRetry) {
    return
  }

  const filters = { ...committedFilters.value }
  const request: ForumHomeRequestToken = {
    generation: feedGeneration,
    feedKey: activeFeedKey.value
  }
  const page = nextPage.value
  isLoadingMore.value = true
  loadMoreError.value = ''

  try {
    const nextList = await loadTopicPage(page, filters)
    if (!isForumHomeRequestCurrent(request, feedGeneration, activeFeedKey.value)) {
      return
    }

    const existingIds = new Set(loadedTopics.value.map((topic) => topic.id))
    const newTopics = nextList.items.filter((topic) => !existingIds.has(topic.id))
    loadedTopics.value = [...loadedTopics.value, ...newTopics]
    loadedTopicTotal.value = Math.max(loadedTopicTotal.value, nextList.total)
    loadedFeedKey.value = activeFeedKey.value
    nextPage.value = nextList.page + 1
    hasLoadedAllPages.value = hasReachedForumHomeEnd({
      requestedPage: page,
      responsePage: nextList.page,
      responseItemCount: nextList.items.length,
      newItemCount: newTopics.length,
      loadedCount: loadedTopics.value.length,
      total: loadedTopicTotal.value,
      perPage: nextList.perPage
    })
  } catch {
    if (isForumHomeRequestCurrent(request, feedGeneration, activeFeedKey.value)) {
      loadMoreError.value = 'home.feed.loadMoreFailed'
    }
  } finally {
    if (isForumHomeRequestCurrent(request, feedGeneration, activeFeedKey.value)) {
      isLoadingMore.value = false
    }
  }
}

let loadMoreObserver: IntersectionObserver | null = null
let stopLoadMoreTriggerWatch: (() => void) | null = null
let searchDebounceTimer: ReturnType<typeof setTimeout> | null = null

function clearSearchDebounce() {
  if (searchDebounceTimer) {
    clearTimeout(searchDebounceTimer)
    searchDebounceTimer = null
  }
}

function commitFilters(nextFilters: ForumHomeFilters) {
  if (forumHomeFeedKey(nextFilters) === activeFeedKey.value) {
    return
  }
  return router.replace({
    path: localePath('/'),
    query: buildForumHomeQuery(nextFilters)
  })
}

watch(() => committedFilters.value.query, (query) => {
  clearSearchDebounce()
  if (searchDraft.value !== query) {
    searchDraft.value = query
  }
})

watch(searchDraft, (query) => {
  clearSearchDebounce()
  if (query.trim() === committedFilters.value.query) {
    return
  }
  searchDebounceTimer = setTimeout(() => {
    searchDebounceTimer = null
    const nextFilters = { ...committedFilters.value, query: query.trim() }
    void commitFilters(nextFilters)
  }, SEARCH_DEBOUNCE_MS)
})

function submitSearch(query: string) {
  clearSearchDebounce()
  searchDraft.value = query
  return commitFilters({ ...committedFilters.value, query: query.trim() })
}

function selectCategory(slug: string) {
  const categorySlug = selectedCategorySlug.value === slug ? '' : slug
  return commitFilters({ ...committedFilters.value, categorySlug })
}

function selectTag(slug: string) {
  const tagSlug = selectedTagSlug.value === slug ? '' : slug
  return commitFilters({ ...committedFilters.value, tagSlug })
}

function resetFilters() {
  clearSearchDebounce()
  searchDraft.value = ''
  return commitFilters({ query: '', categorySlug: '', tagSlug: '' })
}

async function retryFirstPage() {
  await refreshTopics()
}

function topicActivity(topic: ForumTopicSummary) {
  const value = topic.lastActivityAt || topic.createdAt
  const date = new Date(value)
  if (Number.isNaN(date.getTime())) {
    return ''
  }

  const diffMs = renderedAt.value - date.getTime()
  const minute = 60 * 1000
  const hour = 60 * minute
  const day = 24 * hour

  if (diffMs >= 0 && diffMs < hour) {
    return t('home.feed.activityMinutes', { count: Math.max(1, Math.floor(diffMs / minute)) })
  }
  if (diffMs >= 0 && diffMs < day) {
    return t('home.feed.activityHours', { count: Math.max(1, Math.floor(diffMs / hour)) })
  }
  if (diffMs >= 0 && diffMs < 7 * day) {
    return t('home.feed.activityDays', { count: Math.max(1, Math.floor(diffMs / day)) })
  }

  return date.toISOString().slice(0, 10)
}

onMounted(() => {
  if (typeof IntersectionObserver === 'undefined') {
    return
  }

  loadMoreObserver = new IntersectionObserver((entries) => {
    if (entries.some((entry) => entry.isIntersecting)) {
      void loadMoreTopics()
    }
  }, {
    rootMargin: '360px 0px'
  })

  stopLoadMoreTriggerWatch = watch(loadMoreTrigger, (element, previousElement) => {
    if (previousElement) {
      loadMoreObserver?.unobserve(previousElement)
    }
    if (element) {
      loadMoreObserver?.observe(element)
    }
  }, { immediate: true })
})

onBeforeUnmount(() => {
  clearSearchDebounce()
  stopLoadMoreTriggerWatch?.()
  loadMoreObserver?.disconnect()
})
</script>

<template>
  <SFPageOutlet page="forum.home">
  <main class="sforum-home">
    <div
      class="sforum-home__layout"
      :class="{ 'sforum-home__layout--with-right': rightRailEnabled }"
    >
      <div class="sforum-home__sidebar">
        <SFHomeNavigation
          desktop-only
          :categories="categories"
          :selected-category-slug="selectedCategorySlug"
          :total-topics="totalTopics"
          :pending="categoriesPending"
          :can-create-topic="canCreateTopic"
          @select-category="selectCategory"
        />
      </div>

      <section class="sforum-home__main" aria-labelledby="forum-feed-title">
        <h2 id="forum-feed-title" class="sr-only">{{ feedTitle }}</h2>

        <div class="sforum-home__mobile-nav">
          <SFHomeNavigation
            mobile-only
            :categories="categories"
            :selected-category-slug="selectedCategorySlug"
            :total-topics="totalTopics"
            :pending="categoriesPending"
            :can-create-topic="canCreateTopic"
            @select-category="selectCategory"
          />
        </div>

        <div v-if="homeNotice" class="sforum-home__notice" role="note">
          {{ homeNotice }}
        </div>

        <div class="sforum-home__feed-tabs" role="tablist" :aria-label="t('home.filter.latest')">
          <button
            type="button"
            role="tab"
            class="sforum-home__feed-tab"
            :class="{ 'is-active': !selectedCategorySlug && !selectedTagSlug }"
            :aria-selected="!selectedCategorySlug && !selectedTagSlug"
            @click="resetFilters"
          >
            {{ t('home.filter.latest') }}
          </button>
          <button
            v-if="selectedCategory"
            type="button"
            role="tab"
            class="sforum-home__feed-tab is-active"
            aria-selected="true"
          >
            {{ selectedCategory.name }}
          </button>
        </div>

        <p v-if="committedFilters.query" class="sforum-home__search-context">
          {{ t('home.searchResults', { query: committedFilters.query }) }}
        </p>

        <div v-if="activeTags.length || hasActiveFilters" class="sforum-home__filters" :aria-busy="tagsPending">
          <div id="home-tags" class="sforum-home__tag-list">
            <button
              v-for="tag in activeTags"
              :key="tag.slug"
              type="button"
              class="sforum-home__tag"
              :class="{ 'is-active': selectedTagSlug === tag.slug }"
              :aria-pressed="selectedTagSlug === tag.slug"
              @click="selectTag(tag.slug)"
            >
              <span>#{{ tag.name }}</span>
            </button>
          </div>
          <button
            v-if="hasActiveFilters"
            type="button"
            class="sforum-home__clear-filters"
            @click="resetFilters"
          >
            <UIcon name="i-lucide-x" class="size-3.5" aria-hidden="true" />
            {{ t('home.clearFilters') }}
          </button>
        </div>

        <div id="feed-list-container" class="sforum-home__topic-table">
          <div class="sforum-home__topic-head" aria-hidden="true">
            <span>{{ t('home.feed.topicColumn') }}</span>
            <span>{{ t('home.feed.authorColumn') }}</span>
            <span>{{ t('home.feed.repliesColumn') }}</span>
            <span>{{ t('home.feed.activityColumn') }}</span>
          </div>

          <div v-if="topicsError" class="sforum-home__load-error" role="alert">
            <span>{{ t('home.feed.loadFailed') }}</span>
            <button type="button" class="sf-button sf-button--ghost sf-button--sm" @click="retryFirstPage">
              <UIcon name="i-lucide-refresh-cw" class="size-4" aria-hidden="true" />
              {{ t('home.feed.retry') }}
            </button>
          </div>

          <template v-else-if="topicsPending">
            <div v-for="item in 7" :key="item" class="sforum-home__skeleton-row">
              <SFSkeleton avatar :lines="2" />
            </div>
          </template>

          <template v-else-if="topics.length">
            <SFHomeTopicRow
              v-for="topic in topics"
              :key="topic.id"
              :topic="topic"
              :to="localePath(forumTopicPath(topic, topicUrlMode))"
              :activity-label="topicActivity(topic)"
              :extension-list-badges="topicList.extensionListBadges || []"
            />
          </template>

          <div v-else class="sforum-home__empty">
            <SFEmptyState
              :title="emptyTitle"
              :description="emptyDescription"
              :action-label="hasActiveFilters ? t('home.clearFilters') : undefined"
              @action="resetFilters"
            />
          </div>
        </div>

        <div
          v-if="topics.length > 0 && !topicsPending && !topicsError"
          ref="loadMoreTrigger"
          class="sforum-home__infinite-state"
        >
          <span v-if="isLoadingMore">{{ t('home.feed.loadingMore') }}</span>
          <template v-else-if="loadMoreError">
            <span>{{ t(loadMoreError) }}</span>
            <button type="button" class="sf-button sf-button--ghost sf-button--sm" @click="loadMoreTopics(true)">
              <UIcon name="i-lucide-refresh-cw" class="size-4" aria-hidden="true" />
              {{ t('home.feed.retryLoadMore') }}
            </button>
          </template>
          <span v-else-if="!hasMoreTopics">{{ t('home.feed.end') }}</span>
          <span v-else class="sforum-home__sentinel" aria-hidden="true" />
        </div>
      </section>

      <SFHomeRightRail
        v-if="rightRailEnabled"
        :hot-topics="hotTopics"
        :tags="railTags"
        :selected-tag-slug="selectedTagSlug"
        :total-topics="totalTopics"
        :total-replies="totalReplies"
        :category-count="categories.length"
        :tag-count="activeTags.length"
        :topic-url-mode="topicUrlMode"
        :can-create-topic="canCreateTopic"
        @select-tag="selectTag"
      />
    </div>
  </main>
  </SFPageOutlet>
</template>
