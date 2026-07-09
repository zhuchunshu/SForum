<script setup lang="ts">
import {
  forumTopicPath,
  type ForumCategory,
  type ForumCategoryGroup,
  type ForumTag,
  type ForumTopicList,
  type ForumTopicSummary
} from '~/utils/forumTaxonomy'

const { t } = useI18n()
const localePath = useLocalePath()
const { siteName, seoSettings } = useWebOptions()
// 帖子 URL 形态：列表卡片链接按当前模式生成。
const topicUrlMode = computed(() => seoSettings.value.topicUrlMode)
const forumApi = useForumApi()

definePageMeta({
  layout: false
})

useSForumSeo({
  title: () => t('home.metaTitle', { siteName: siteName.value }),
  description: () => seoSettings.value.metaDescription || t('home.metaDescription', { siteName: siteName.value }),
  type: 'website',
  schema: { type: 'WebPage' }
})

const categoryDotColors = ['var(--sf-accent)', 'var(--sf-violet)', 'var(--sf-warning)', 'var(--sf-danger)'] as const
const ITEMS_PER_PAGE = 10

type FeedTabItem = {
  label: string
  value: string
  disabled?: boolean
}

// Vue 模板中的 v-for index 可能被推断为 string | number，集中转成数字避免模板隐式计算。
function normalizedIndex(index: string | number) {
  return typeof index === 'number' ? index : Number(index)
}

function categoryDotStyle(index: string | number) {
  return {
    background: categoryDotColors[normalizedIndex(index)] || categoryDotColors[0]
  }
}

// Search & Filter state
const searchQuery = ref('')
const selectedCategorySlug = ref('')
const selectedTagSlug = ref('')
const currentTab = ref('latest')

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

const topicFilters = computed(() => ({
  categorySlug: selectedCategorySlug.value,
  tagSlug: selectedTagSlug.value,
  query: searchQuery.value,
  page: 1,
  perPage: ITEMS_PER_PAGE
}))

const activeFeedKey = computed(() => [
  searchQuery.value.trim(),
  selectedCategorySlug.value,
  selectedTagSlug.value
].join('\u001F'))

function loadTopicPage(page: number) {
  const trimmed = searchQuery.value.trim()
  if (trimmed) {
    return forumApi.searchTopics({
      query: trimmed,
      categorySlug: selectedCategorySlug.value,
      tagSlug: selectedTagSlug.value,
      page,
      perPage: ITEMS_PER_PAGE
    })
  }

  return forumApi.listTopics({
    categorySlug: selectedCategorySlug.value,
    tagSlug: selectedTagSlug.value,
    page,
    perPage: ITEMS_PER_PAGE
  })
}

// 搜索关键词非空时走专用搜索端点（Meilisearch），否则走常规主题列表。
// 两条路径返回结构一致（ForumTopicList），下游渲染无需区分。
const { data: topicList, pending: topicsPending } = await useAsyncData(
  'forum-home-topics',
  () => loadTopicPage(1),
  {
    default: emptyTopicList,
    watch: [topicFilters]
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

// SFTabs configuration
const tabItems = computed<FeedTabItem[]>(() => [
  { label: t('home.filter.latest'), value: 'latest' },
  { label: t('home.filter.new'), value: 'new', disabled: true },
  { label: t('home.filter.unread'), value: 'unread', disabled: true },
  { label: t('home.filter.ranking'), value: 'ranking', disabled: true },
  { label: t('home.filter.myTopics'), value: 'my-topics', disabled: true }
])

const categories = computed(() => categoryGroups.value.flatMap((group) => group.categories || []))
const topics = computed(() => loadedTopics.value)
const isPending = computed(() => categoriesPending.value || tagsPending.value || topicsPending.value)
const hasMoreTopics = computed(() => !hasLoadedAllPages.value && loadedTopics.value.length < loadedTopicTotal.value)

// Categories count total
const totalCategoryThreads = computed(() => categories.value.reduce((acc, cur) => acc + cur.topicCount, 0))
const totalCategoryComments = computed(() => categories.value.reduce((acc, cur) => acc + cur.commentCount, 0))

function replaceLoadedTopics(list: ForumTopicList) {
  loadedTopics.value = list.items
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

async function loadMoreTopics(forceRetry = false) {
  if (isPending.value || isLoadingMore.value || !hasMoreTopics.value) {
    return
  }
  if (loadMoreError.value && !forceRetry) {
    return
  }

  const feedKey = activeFeedKey.value
  const page = nextPage.value
  isLoadingMore.value = true
  loadMoreError.value = ''

  try {
    const nextList = await loadTopicPage(page)
    if (feedKey !== activeFeedKey.value) {
      return
    }

    const existingIds = new Set(loadedTopics.value.map((topic) => topic.id))
    const newTopics = nextList.items.filter((topic) => !existingIds.has(topic.id))
    loadedTopics.value = [...loadedTopics.value, ...newTopics]
    loadedTopicTotal.value = Math.max(loadedTopicTotal.value, nextList.total)
    loadedFeedKey.value = activeFeedKey.value
    nextPage.value = Math.max(page + 1, nextList.page + 1)
    hasLoadedAllPages.value = nextList.items.length === 0
      || nextList.items.length < nextList.perPage
      || loadedTopics.value.length >= loadedTopicTotal.value
  } catch {
    if (feedKey === activeFeedKey.value) {
      loadMoreError.value = 'home.feed.loadMoreFailed'
    }
  } finally {
    if (feedKey === activeFeedKey.value) {
      isLoadingMore.value = false
    }
  }
}

let loadMoreObserver: IntersectionObserver | null = null
let stopLoadMoreTriggerWatch: (() => void) | null = null

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
  stopLoadMoreTriggerWatch?.()
  loadMoreObserver?.disconnect()
})

function selectCategory(category: ForumCategory) {
  selectedCategorySlug.value = selectedCategorySlug.value === category.slug ? '' : category.slug
}

function selectTag(tag: ForumTag) {
  selectedTagSlug.value = selectedTagSlug.value === tag.slug ? '' : tag.slug
}

function selectFeedTab(item: FeedTabItem) {
  if (item.disabled) {
    return
  }
  currentTab.value = item.value
}

function categoryButtonClass(category: ForumCategory) {
  return selectedCategorySlug.value === category.slug
    ? 'sforum-home__filter-chip is-active'
    : 'sforum-home__filter-chip'
}

function tagButtonClass(tag: ForumTag) {
  return selectedTagSlug.value === tag.slug
    ? 'sforum-home__filter-tag is-active'
    : 'sforum-home__filter-tag'
}

function topicAuthor(topic: ForumTopicSummary) {
  return topic.author?.displayName || topic.author?.username || `#${topic.authorUserId}`
}

function topicActivity(topic: ForumTopicSummary) {
  const value = topic.lastActivityAt || topic.createdAt
  const date = new Date(value)
  if (Number.isNaN(date.getTime())) {
    return ''
  }

  const diffMs = Date.now() - date.getTime()
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

  return formatShortDate(value)
}

function topicReplyStackLabel(topic: ForumTopicSummary) {
  if (topic.commentCount <= 0) {
    return '0'
  }
  if (topic.commentCount > 99) {
    return '99+'
  }
  return `+${topic.commentCount}`
}

function formatShortDate(value: string) {
  const date = new Date(value)
  if (Number.isNaN(date.getTime())) {
    return ''
  }

  const pad = (part: number) => String(part).padStart(2, '0')
  return `${date.getFullYear()}-${pad(date.getMonth() + 1)}-${pad(date.getDate())} ${pad(date.getHours())}:${pad(date.getMinutes())}`
}
</script>

<template>
  <div class="sforum-home-page">
    <main class="sforum-home">
      <div class="sforum-home__shell">
        <header class="sforum-home__topbar">
          <NuxtLink :to="localePath('/')" class="sforum-home__brand">
            <span class="sforum-home__brand-mark">SF</span>
            <span>{{ siteName }}</span>
          </NuxtLink>

          <nav class="sforum-home__top-links" aria-label="首页主导航">
            <NuxtLink :to="localePath('/')" class="is-active">
              {{ t('home.filter.latest') }}
            </NuxtLink>
            <button type="button" disabled>
              {{ t('home.filter.ranking') }}
            </button>
            <button type="button" disabled>
              {{ t('home.filter.hot') }}
            </button>
            <a href="#categories">
              {{ t('home.filter.categories') }}
            </a>
            <a href="#tags">
              {{ t('home.filter.tags') }}
            </a>
          </nav>

          <div class="sforum-home__top-actions">
            <SFSearch
              v-model="searchQuery"
              class="sforum-home__top-search"
              :placeholder="t('home.searchPlaceholder')"
            />
            <NuxtLink :to="localePath('/topics/new')" class="sf-button sf-button--primary sf-button--sm sforum-home__top-compose">
              <UIcon name="i-lucide-square-pen" class="size-4" />
              <span>{{ t('nav.newTopic') }}</span>
            </NuxtLink>
          </div>
        </header>

      <div class="sforum-home__layout">
        <aside class="sforum-home__left sforum-home__rail hidden lg:block">
          <NuxtLink :to="localePath('/topics/new')" class="sf-button sf-button--primary sf-button--md sf-button--block sforum-home__compose">
            <UIcon name="i-lucide-square-pen" class="size-4" />
            <span>{{ t('home.sidebar.newTopic') }}</span>
          </NuxtLink>

          <div class="sforum-home__side-group">{{ t('home.sidebar.navTitle') }}</div>
          <nav class="sforum-home__side-list" aria-label="首页辅助导航">
            <NuxtLink :to="localePath('/')" class="sforum-home__side-link is-active">
              <span>{{ t('home.sidebar.navHome') }}</span>
              <span class="sforum-home__side-count">{{ loadedTopicTotal || totalCategoryThreads }}</span>
            </NuxtLink>
            <button type="button" class="sforum-home__side-link" disabled>
              <span>{{ t('home.sidebar.about') }}</span>
              <span class="sforum-home__side-count">3</span>
            </button>
            <button type="button" class="sforum-home__side-link" disabled>
              <span>{{ t('home.sidebar.myPosts') }}</span>
              <span class="sforum-home__side-count">--</span>
            </button>
            <button type="button" class="sforum-home__side-link" disabled>
              <span>{{ t('home.sidebar.recentActivity') }}</span>
              <span class="sforum-home__side-count">{{ totalCategoryComments }}</span>
            </button>
          </nav>

          <div id="categories" class="sforum-home__side-group">{{ t('home.sidebar.sections') }}</div>
          <ul class="sforum-home__side-list">
            <li v-for="(cat, idx) in categories" :key="cat.slug">
              <button
                type="button"
                class="sforum-home__side-link"
                :class="{ 'is-active': selectedCategorySlug === cat.slug }"
                @click="selectCategory(cat)"
              >
                <span class="sforum-home__side-name">
                  <span class="sforum-home__side-dot" :style="categoryDotStyle(idx)" />
                  <span class="truncate">{{ cat.name }}</span>
                </span>
                <span class="sforum-home__side-count">{{ cat.topicCount }}</span>
              </button>
            </li>
          </ul>
        </aside>

        <section class="sforum-home__main">
          <div class="sforum-home__mobile-filters lg:hidden">
            <SFCard flush class="p-3">
              <div class="flex gap-2 overflow-x-auto pb-1">
                <button
                  v-for="(cat, idx) in categories"
                  :key="cat.slug"
                  type="button"
                  class="inline-flex shrink-0 items-center gap-2 rounded-lg px-3 py-2 text-sm font-semibold transition"
                  :class="categoryButtonClass(cat)"
                  @click="selectCategory(cat)"
                >
                  <span class="size-2 rounded-full" :style="categoryDotStyle(idx)" />
                  <span>{{ cat.name }}</span>
                </button>
              </div>
              <div v-if="activeTags.length" class="mt-2 flex gap-2 overflow-x-auto pb-1">
                <button
                  v-for="tag in activeTags"
                  :key="tag.slug"
                  type="button"
                  class="inline-flex shrink-0 items-center gap-1.5 rounded-md border px-2.5 py-1.5 text-xs font-semibold transition"
                  :class="tagButtonClass(tag)"
                  @click="selectTag(tag)"
                >
                  <span>#{{ tag.name }}</span>
                  <span class="font-mono text-[11px] opacity-70">{{ tag.topicCount }}</span>
                </button>
              </div>
            </SFCard>
          </div>

          <div class="sforum-home__notice">
            <span>{{ t('home.notice') }}</span>
          </div>

          <div class="sforum-home__feed-tabs" role="tablist" aria-label="帖子排序切换">
            <button
              v-for="item in tabItems"
              :key="item.value"
              type="button"
              :class="{ 'is-active': currentTab === item.value, 'is-disabled': item.disabled }"
              :disabled="item.disabled"
              :aria-pressed="currentTab === item.value"
              @click="selectFeedTab(item)"
            >
              {{ item.label }}
            </button>
          </div>

          <section class="sforum-topic-table" aria-label="主题列表">
            <div class="sforum-topic-table__head">
              <span>{{ t('home.feed.topicColumn') }}</span>
              <span>{{ t('home.feed.repliesColumn') }}</span>
              <span>{{ t('home.feed.activityColumn') }}</span>
            </div>

            <div id="feed-list-container" class="sforum-topic-table__rows">
              <template v-if="isPending">
                <div v-for="i in 6" :key="i" class="sforum-topic-row">
                  <SFSkeleton :lines="2" />
                </div>
              </template>

              <template v-else-if="topics.length > 0">
                <NuxtLink
                  v-for="topic in topics"
                  :key="topic.id"
                  :to="localePath(forumTopicPath(topic, topicUrlMode))"
                  class="sforum-topic-row"
                >
                  <div class="sforum-topic-row__main">
                    <h2 class="sforum-topic-row__title">{{ topic.title }}</h2>
                    <div class="sforum-topic-row__meta">
                      <span class="sforum-topic-row__badge">{{ topic.categoryName }}</span>
                      <span v-if="topic.isPinned" class="sforum-topic-row__badge sforum-topic-row__badge--warn">{{ t('home.badge.pinned') }}</span>
                      <span
                        v-for="tag in topic.tags || []"
                        :key="tag.slug"
                        class="sforum-topic-row__badge sforum-topic-row__badge--neutral"
                      >
                        #{{ tag.name }}
                      </span>
                      <span class="sforum-topic-row__meta-copy">{{ topicAuthor(topic) }} · {{ t('home.sidebar.repliesCount', { count: topic.commentCount }) }} · {{ topic.viewCount }} {{ t('home.feed.views') }}</span>
                    </div>
                  </div>

                  <div class="sforum-topic-row__participants" :aria-label="t('home.feed.repliesColumn')">
                    <SFAvatar :name="topicAuthor(topic)" :avatar="topic.author?.avatar" size="sm" />
                    <span class="sforum-topic-row__reply-count">{{ topicReplyStackLabel(topic) }}</span>
                  </div>

                  <div class="sforum-topic-row__activity">
                    {{ topicActivity(topic) }}
                  </div>
                </NuxtLink>
              </template>

              <div v-else class="flex justify-center px-4 py-12">
                <SFEmptyState
                  :title="t('home.emptyState.title')"
                  :description="t('home.emptyState.description')"
                />
              </div>
            </div>

            <div
              v-if="topics.length > 0 && !isPending"
              ref="loadMoreTrigger"
              class="sforum-topic-table__infinite-state"
            >
              <span v-if="isLoadingMore">{{ t('home.feed.loadingMore') }}</span>
              <template v-else-if="loadMoreError">
                <span>{{ t(loadMoreError) }}</span>
                <button type="button" class="sf-button sf-button--ghost sf-button--sm" @click="loadMoreTopics(true)">
                  {{ t('home.feed.retryLoadMore') }}
                </button>
              </template>
              <span v-else-if="!hasMoreTopics">{{ t('home.feed.end') }}</span>
              <span v-else class="sforum-topic-table__sentinel" aria-hidden="true" />
            </div>
          </section>

          <section id="tags" class="sforum-home__tag-strip">
            <button
              v-for="tag in activeTags"
              :key="tag.slug"
              type="button"
              class="sforum-home__tag"
              :class="{ 'is-active': selectedTagSlug === tag.slug }"
              @click="selectTag(tag)"
            >
              <span>#{{ tag.name }}</span>
              <span>{{ tag.topicCount }}</span>
            </button>
          </section>
        </section>
      </div>
      </div>
    </main>
    <SFFooter />
  </div>
</template>
