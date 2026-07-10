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

const ITEMS_PER_PAGE = 10
const SEARCH_DEBOUNCE_MS = 300

const { t } = useI18n()
const route = useRoute()
const router = useRouter()
const localePath = useLocalePath()
const { siteName, seoSettings } = useWebOptions()
const forumApi = useForumApi()
const { can } = usePermissions()

useSForumSeo({
  title: () => t('home.metaTitle', { siteName: siteName.value }),
  description: () => seoSettings.value.metaDescription || t('home.metaDescription', { siteName: siteName.value }),
  type: 'website',
  schema: { type: 'WebPage' }
})

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
  perPage: ITEMS_PER_PAGE
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
      page,
      perPage: ITEMS_PER_PAGE
    })
  }

  return forumApi.listTopics({
    categorySlug: filters.categorySlug,
    tagSlug: filters.tagSlug,
    page,
    perPage: ITEMS_PER_PAGE
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
const emptyDescription = computed(() => hasActiveFilters.value
  ? t('home.emptyState.filteredDescription')
  : t('home.emptyState.description'))
const totalTopics = computed(() => {
  const categoryTotal = categories.value.reduce((total, category) => total + category.topicCount, 0)
  if (categoryTotal > 0) {
    return categoryTotal
  }
  return hasActiveFilters.value ? 0 : loadedTopicTotal.value
})
const dockTopics = computed(() => topics.value.slice(0, 3))
const loadedReplyTotal = computed(() => topics.value.reduce((total, topic) => total + Number(topic.commentCount), 0))

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
  <main class="sforum-home">
    <div class="sforum-home__inner">
      <div class="sforum-home__layout">
        <SFHomeNavigation
          :categories="categories"
          :selected-category-slug="selectedCategorySlug"
          :total-topics="totalTopics"
          :pending="categoriesPending"
          @select-category="selectCategory"
        />

        <section class="sforum-home__feed" aria-labelledby="forum-feed-title">
          <header class="sforum-home__feed-header">
            <h1 id="forum-feed-title" class="sr-only">{{ feedTitle }}</h1>
            <SFSearch
              v-model="searchDraft"
              class="sforum-home__search"
              :placeholder="t('home.searchPlaceholder')"
              :aria-label="t('nav.search')"
              @submit="submitSearch"
            />
            <NuxtLink
              v-if="canCreateTopic"
              :to="localePath('/topics/new')"
              class="sforum-home__new-topic"
            >
              <UIcon name="i-lucide-square-pen" class="size-4" aria-hidden="true" />
              {{ t('nav.newTopic') }}
            </NuxtLink>
            <NuxtLink
              v-else
              :to="localePath('/login')"
              class="sforum-home__new-topic"
            >
              <UIcon name="i-lucide-log-in" class="size-4" aria-hidden="true" />
              {{ t('home.loginToPost') }}
            </NuxtLink>
          </header>

          <p v-if="committedFilters.query" class="sforum-home__search-context">
            {{ t('home.searchResults', { query: committedFilters.query }) }}
          </p>

          <div v-if="categories.length || activeTags.length || hasActiveFilters" class="sforum-home__filters" :aria-busy="tagsPending">
            <div class="sforum-home__tag-list">
              <button
                type="button"
                class="sforum-home__tag"
                :class="{ 'is-active': !selectedCategorySlug && !selectedTagSlug }"
                :aria-pressed="!selectedCategorySlug && !selectedTagSlug"
                @click="selectCategory('')"
              >
                {{ t('home.filter.all') }}
              </button>
              <button
                v-for="category in categories"
                :key="category.slug"
                type="button"
                class="sforum-home__tag"
                :class="{ 'is-active': selectedCategorySlug === category.slug }"
                :aria-pressed="selectedCategorySlug === category.slug"
                @click="selectCategory(category.slug)"
              >
                {{ category.name }}
              </button>
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

          <div id="feed-list-container" class="sforum-home__topics">
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
              />
            </template>

            <div v-else class="sforum-home__empty">
              <SFEmptyState
                :title="t('home.emptyState.title')"
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

        <aside class="sforum-home__dock" :aria-label="t('home.dock.title')">
          <section class="sforum-home__dock-block sforum-home__dock-notice" role="note">
            <UIcon name="i-lucide-shield-check" class="size-5" aria-hidden="true" />
            <p>{{ t('home.notice') }}</p>
          </section>

          <section class="sforum-home__dock-block">
            <h2>{{ t('home.dock.latestActivity') }}</h2>
            <ol class="sforum-home__dock-list">
              <li v-for="(topic, index) in dockTopics" :key="topic.id">
                <span class="sforum-home__dock-rank">{{ Number(index) + 1 }}</span>
                <NuxtLink :to="localePath(forumTopicPath(topic, topicUrlMode))">
                  {{ topic.title }}
                </NuxtLink>
              </li>
            </ol>
          </section>

          <section class="sforum-home__dock-block">
            <h2>{{ t('home.dock.overview') }}</h2>
            <dl class="sforum-home__stats">
              <div>
                <dd>{{ totalTopics }}</dd>
                <dt>{{ t('home.dock.topics') }}</dt>
              </div>
              <div>
                <dd>{{ loadedReplyTotal }}</dd>
                <dt>{{ t('home.dock.loadedReplies') }}</dt>
              </div>
            </dl>
          </section>
        </aside>
      </div>
    </div>
  </main>
</template>
