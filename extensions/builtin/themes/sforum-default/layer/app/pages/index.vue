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
const { user } = useAuthSession()
const { siteName, seoSettings } = useWebOptions()
// 帖子 URL 形态：列表卡片链接按当前模式生成。
const topicUrlMode = computed(() => seoSettings.value.topicUrlMode)
const forumApi = useForumApi()

useSForumSeo({
  title: () => t('home.metaTitle', { siteName: siteName.value }),
  description: () => seoSettings.value.metaDescription || t('home.metaDescription', { siteName: siteName.value }),
  type: 'website',
  schema: { type: 'WebPage' }
})

const categoryDotColors = ['#0F766E', '#8B5CF6', '#F59E0B', '#EF4444'] as const
const ITEMS_PER_PAGE = 10

type FeedBadge = {
  label: string
  variant?: 'neutral' | 'primary' | 'info' | 'success' | 'warning' | 'danger'
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

function hotTopicRank(index: string | number) {
  return normalizedIndex(index) + 1
}

function hotTopicRankClass(index: string | number) {
  const rankIndex = normalizedIndex(index)

  if (rankIndex === 0) {
    return 'bg-red-500 text-white'
  }
  if (rankIndex === 1) {
    return 'bg-orange-400 text-white'
  }
  if (rankIndex === 2) {
    return 'bg-yellow-400 text-slate-800'
  }

  return 'bg-slate-200 text-slate-600'
}

// Search & Filter state
const searchQuery = ref('')
const selectedCategorySlug = ref('')
const selectedTagSlug = ref('')
const currentTab = ref('latest')
const currentPage = ref(1)

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
  page: currentPage.value,
  perPage: ITEMS_PER_PAGE
}))

// 搜索关键词非空时走专用搜索端点（Meilisearch），否则走常规主题列表。
// 两条路径返回结构一致（ForumTopicList），下游渲染无需区分。
const { data: topicList, pending: topicsPending } = await useAsyncData(
  'forum-home-topics',
  () => {
    const trimmed = searchQuery.value.trim()
    if (trimmed) {
      return forumApi.searchTopics({
        query: trimmed,
        categorySlug: selectedCategorySlug.value,
        tagSlug: selectedTagSlug.value,
        page: currentPage.value,
        perPage: ITEMS_PER_PAGE
      })
    }
    return forumApi.listTopics(topicFilters.value)
  },
  {
    default: emptyTopicList,
    watch: [topicFilters]
  }
)

// SFTabs configuration
const tabItems = computed(() => [
  { label: t('home.filter.latest'), value: 'latest' },
  { label: t('home.filter.hot'), value: 'hot', disabled: true },
  { label: t('home.filter.featured'), value: 'featured', disabled: true },
  { label: t('home.filter.following'), value: 'following', disabled: true }
])

const categories = computed(() => categoryGroups.value.flatMap((group) => group.categories || []))
const topics = computed(() => topicList.value.items)
const activeCategory = computed(() => {
  return categories.value.find((category) => category.slug === selectedCategorySlug.value)
})
const activeTag = computed(() => {
  return activeTags.value.find((tag) => tag.slug === selectedTagSlug.value)
})
const feedTitle = computed(() => {
  if (activeCategory.value) {
    return activeCategory.value.name
  }
  if (activeTag.value) {
    return `#${activeTag.value.name}`
  }
  return t('home.sidebar.navHome')
})
const isPending = computed(() => categoriesPending.value || tagsPending.value || topicsPending.value)
const totalPages = computed(() => {
  return Math.ceil(topicList.value.total / Math.max(topicList.value.perPage, 1)) || 1
})
const hotTopics = computed(() => {
  return [...topics.value]
    .sort((a, b) => b.commentCount - a.commentCount)
    .slice(0, 5)
})

// Daily check-in status
const checkedIn = ref(false)
const checkInDays = ref(3)
function handleCheckIn() {
  if (!checkedIn.value) {
    checkedIn.value = true
    checkInDays.value += 1
  }
}

// Categories count total
const totalCategoryThreads = computed(() => categories.value.reduce((acc, cur) => acc + cur.topicCount, 0))
const totalCategoryComments = computed(() => categories.value.reduce((acc, cur) => acc + cur.commentCount, 0))

watch([searchQuery, selectedCategorySlug, selectedTagSlug], () => {
  if (currentPage.value !== 1) {
    currentPage.value = 1
  }
})

function selectCategory(category: ForumCategory) {
  selectedCategorySlug.value = selectedCategorySlug.value === category.slug ? '' : category.slug
}

function selectTag(tag: ForumTag) {
  selectedTagSlug.value = selectedTagSlug.value === tag.slug ? '' : tag.slug
}

function categoryButtonClass(category: ForumCategory) {
  return selectedCategorySlug.value === category.slug
    ? 'bg-[#E6F4F1] text-[#0F766E] dark:bg-teal-950/40 dark:text-teal-300'
    : 'text-slate-700 hover:text-slate-900 hover:bg-slate-100 dark:text-zinc-300 dark:hover:bg-zinc-800 dark:hover:text-zinc-50'
}

function tagButtonClass(tag: ForumTag) {
  return selectedTagSlug.value === tag.slug
    ? 'border-[#0F766E] bg-[#E6F4F1] text-[#0F766E] dark:border-teal-700 dark:bg-teal-950/40 dark:text-teal-300'
    : 'border-slate-200 text-slate-700 hover:border-[#0F766E] hover:text-[#0F766E] dark:border-zinc-700 dark:text-zinc-300 dark:hover:border-teal-600 dark:hover:text-teal-300'
}

function topicAuthor(topic: ForumTopicSummary) {
  return topic.author?.displayName || topic.author?.username || `#${topic.authorUserId}`
}

function topicMeta(topic: ForumTopicSummary) {
  return formatShortDate(topic.lastActivityAt || topic.createdAt)
}

function topicBadges(topic: ForumTopicSummary): FeedBadge[] {
  return [
    ...(topic.isPinned ? [{ label: t('home.badge.pinned'), variant: 'danger' as const }] : []),
    { label: topic.categoryName, variant: 'primary' as const },
    ...(topic.tags || []).map((tag) => ({ label: `#${tag.name}`, variant: 'neutral' as const }))
  ]
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
  <main class="sforum-home">
    <div class="sforum-home__inner">
      <div class="sforum-home__layout grid grid-cols-1 gap-4 lg:grid-cols-[240px_minmax(0,1fr)_262px]">
        <aside class="sforum-home__left sforum-home__rail hidden lg:grid lg:sticky lg:top-6">
          <section class="sforum-home__rail-card">
            <h2 class="sforum-home__rail-title">
              <span>{{ t('home.sidebar.navTitle') }}</span>
            </h2>
            <nav class="grid gap-1" aria-label="首页辅助导航">
              <NuxtLink :to="localePath('/')" class="flex items-center gap-2 rounded-lg px-3 py-2 text-sm font-bold bg-[#E6F4F1] text-[#0F766E] dark:bg-teal-950/40 dark:text-teal-300">
                <UIcon name="i-lucide-home" class="size-4 shrink-0" />
                <span>{{ t('home.sidebar.navHome') }}</span>
              </NuxtLink>
              <button type="button" class="flex items-center gap-2 rounded-lg px-3 py-2 text-left text-sm font-semibold text-slate-700 opacity-60 dark:text-zinc-300" disabled>
                <UIcon name="i-lucide-flame" class="size-4 shrink-0" />
                <span>{{ t('home.filter.hot') }}</span>
              </button>
              <button type="button" class="flex items-center gap-2 rounded-lg px-3 py-2 text-left text-sm font-semibold text-slate-700 opacity-60 dark:text-zinc-300" disabled>
                <UIcon name="i-lucide-star" class="size-4 shrink-0" />
                <span>{{ t('home.filter.featured') }}</span>
              </button>
              <button type="button" class="flex items-center gap-2 rounded-lg px-3 py-2 text-left text-sm font-semibold text-slate-700 opacity-60 dark:text-zinc-300" disabled>
                <UIcon name="i-lucide-at-sign" class="size-4 shrink-0" />
                <span>{{ t('home.filter.following') }}</span>
              </button>
            </nav>
          </section>

          <section class="sforum-home__rail-card" id="categories">
            <h2 class="sforum-home__rail-title">
              <span>{{ t('home.sidebar.sections') }}</span>
              <span class="font-mono">{{ totalCategoryThreads }}</span>
            </h2>
            <ul class="grid gap-1">
              <li v-for="(cat, idx) in categories" :key="cat.slug">
                <button
                  type="button"
                  class="flex w-full items-center justify-between gap-2 rounded-lg px-3 py-2 text-left text-sm font-semibold transition"
                  :class="categoryButtonClass(cat)"
                  @click="selectCategory(cat)"
                >
                  <span class="flex min-w-0 items-center gap-2">
                    <span class="size-2 shrink-0 rounded-full" :style="categoryDotStyle(idx)" />
                    <span class="truncate">{{ cat.name }}</span>
                  </span>
                  <span class="font-mono text-xs opacity-70">{{ cat.topicCount }}</span>
                </button>
              </li>
            </ul>
          </section>

          <section class="sforum-home__rail-card" id="tags">
            <h2 class="sforum-home__rail-title">
              <span>{{ t('home.sidebar.navTags') }}</span>
            </h2>
            <div v-if="activeTags.length" class="flex flex-wrap gap-2">
              <button
                v-for="tag in activeTags"
                :key="tag.slug"
                type="button"
                class="inline-flex items-center gap-1.5 rounded-md border px-2.5 py-1.5 text-xs font-semibold transition"
                :class="tagButtonClass(tag)"
                @click="selectTag(tag)"
              >
                <span>#{{ tag.name }}</span>
                <span class="font-mono text-[11px] opacity-70">{{ tag.topicCount }}</span>
              </button>
            </div>
            <SFEmptyState
              v-else
              icon-label="TAG"
              :title="t('home.emptyState.title')"
              :description="t('home.emptyState.description')"
            />
          </section>
        </aside>

        <section class="sforum-home__main grid gap-4">
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

          <div class="sforum-topic-table">
            <header class="sforum-topic-table__toolbar">
              <div class="sforum-topic-table__top">
                <h1 class="sforum-topic-table__title">
                  {{ feedTitle }}
                </h1>
                <SFTabs v-model="currentTab" :items="tabItems" aria-label="帖子排序切换" />
              </div>
              <div class="sforum-topic-table__filters">
                <SFSearch
                  v-model="searchQuery"
                  :placeholder="t('home.searchPlaceholder')"
                  id="feed-search"
                />
                <div v-if="selectedCategorySlug || selectedTagSlug" class="flex flex-wrap gap-2">
                  <SFBadge v-if="activeCategory" variant="primary">
                    {{ activeCategory.name }}
                  </SFBadge>
                  <SFBadge v-if="activeTag" variant="neutral">
                    #{{ activeTag.name }}
                  </SFBadge>
                </div>
              </div>
            </header>

            <div id="feed-list-container" class="sforum-topic-table__rows divide-y divide-slate-100 dark:divide-zinc-800">
              <template v-if="isPending">
                <div v-for="i in 6" :key="i" class="px-4 py-3">
                  <SFSkeleton :lines="2" />
                </div>
              </template>

              <template v-else-if="topics.length > 0">
                <NuxtLink
                  v-for="topic in topics"
                  :key="topic.id"
                  :to="localePath(forumTopicPath(topic, topicUrlMode))"
                  class="block transition hover:bg-slate-50 dark:hover:bg-zinc-900/60"
                >
                  <SFFeedRow
                    layout="table"
                    :show-avatar="false"
                    :title="topic.title"
                    :author="topicAuthor(topic)"
                    :meta="topicMeta(topic)"
                    :replies="topic.commentCount"
                    :views="topic.viewCount"
                    :score="0"
                    :badges="topicBadges(topic)"
                    :last-activity-label="topicMeta(topic)"
                    :last-actor="topicAuthor(topic)"
                  />
                </NuxtLink>
              </template>

              <div v-else class="flex justify-center px-4 py-12">
                <SFEmptyState
                  :title="t('home.emptyState.title')"
                  :description="t('home.emptyState.description')"
                />
              </div>
            </div>
          </div>

          <div v-if="topics.length > 0 && !isPending" class="flex justify-center pt-2">
            <SFPagination v-model:page="currentPage" :total-pages="totalPages" />
          </div>
        </section>

        <aside class="sforum-home__right sforum-home__rail hidden md:grid lg:sticky lg:top-6">
          <section class="sforum-home__rail-card text-center">
            <template v-if="user">
              <div class="flex flex-col items-center gap-2">
                <SFAvatar :name="user.displayName" size="lg" status="online" />
                <h2 class="mt-1 text-base font-bold text-slate-800 dark:text-zinc-100">{{ user.displayName }}</h2>
                <p class="text-sm text-slate-500 dark:text-zinc-400">@{{ user.username }}</p>
              </div>
            </template>
            <template v-else>
              <div class="grid gap-3">
                <div class="mx-auto grid size-11 place-items-center rounded-full bg-[#E6F4F1] text-[#0F766E] dark:bg-teal-950/40 dark:text-teal-300">
                  <UIcon name="i-lucide-message-circle" class="size-5" />
                </div>
                <h2 class="text-sm font-bold text-slate-800 dark:text-zinc-100">{{ t('home.sidebar.welcomeTitle', { siteName }) }}</h2>
                <p class="text-xs leading-relaxed text-slate-600 dark:text-zinc-400">{{ t('home.sidebar.welcomeDesc') }}</p>
                <div class="grid grid-cols-2 gap-2">
                  <NuxtLink :to="localePath('/login')" class="sf-button sf-button--ghost sf-button--sm block text-center">
                    {{ t('home.sidebar.loginBtn') }}
                  </NuxtLink>
                  <NuxtLink :to="localePath('/register')" class="sf-button sf-button--primary sf-button--sm block text-center">
                    {{ t('home.sidebar.registerBtn') }}
                  </NuxtLink>
                </div>
              </div>
            </template>
          </section>

          <section v-if="user" class="sforum-home__rail-card flex items-center justify-between gap-3">
            <div class="min-w-0 text-left">
              <h3 class="text-sm font-bold text-slate-800 dark:text-zinc-100">{{ t('home.sidebar.checkIn') }}</h3>
              <p class="mt-1 truncate text-xs text-slate-500 dark:text-zinc-400">
                {{ checkedIn ? t('home.sidebar.checkedIn', { days: checkInDays }) : t('home.sidebar.checkInDesc') }}
              </p>
            </div>
            <SFButton :variant="checkedIn ? 'ghost' : 'primary'" size="sm" :disabled="checkedIn" @click="handleCheckIn">
              {{ checkedIn ? t('home.sidebar.checkedInBtn') : t('home.sidebar.checkInBtn') }}
            </SFButton>
          </section>

          <section class="sforum-home__rail-card">
            <h2 class="sforum-home__rail-title">
              <span>{{ t('home.sidebar.hotThreads') }}</span>
            </h2>
            <ul v-if="hotTopics.length" class="grid gap-3">
              <li v-for="(topic, index) in hotTopics" :key="topic.id" class="flex items-start gap-3">
                <span class="mt-0.5 flex h-[18px] w-[18px] shrink-0 items-center justify-center rounded px-1 text-[10px] font-bold" :class="hotTopicRankClass(index)">
                  {{ hotTopicRank(index) }}
                </span>
                <div class="min-w-0 flex-1">
                  <NuxtLink :to="localePath(forumTopicPath(topic, topicUrlMode))" class="block truncate text-sm font-semibold text-slate-700 hover:text-[#0F766E] hover:underline dark:text-zinc-300 dark:hover:text-teal-300">
                    {{ topic.title }}
                  </NuxtLink>
                  <span class="mt-0.5 block font-mono text-xs text-slate-400 dark:text-zinc-500">{{ t('home.sidebar.repliesCount', { count: topic.commentCount }) }}</span>
                </div>
              </li>
            </ul>
            <SFEmptyState v-else icon-label="HOT" :title="t('home.emptyState.title')" :description="t('home.emptyState.description')" />
          </section>

          <section class="sforum-home__rail-card">
            <h2 class="sforum-home__rail-title">
              <span>{{ t('home.sidebar.forumStats') }}</span>
            </h2>
            <ul class="grid gap-2.5 text-sm text-slate-700 dark:text-zinc-300">
              <li class="flex justify-between gap-3">
                <span class="text-slate-500 dark:text-zinc-400">{{ t('home.sidebar.statThreads') }}</span>
                <span class="font-mono font-semibold text-slate-800 dark:text-zinc-100">{{ topicList.total || totalCategoryThreads }}</span>
              </li>
              <li class="flex justify-between gap-3">
                <span class="text-slate-500 dark:text-zinc-400">{{ t('home.sidebar.statReplies') }}</span>
                <span class="font-mono font-semibold text-slate-800 dark:text-zinc-100">{{ totalCategoryComments }}</span>
              </li>
              <li class="flex justify-between gap-3">
                <span class="text-slate-500 dark:text-zinc-400">{{ t('home.sidebar.statMembers') }}</span>
                <span class="font-mono font-semibold text-slate-800 dark:text-zinc-100">--</span>
              </li>
            </ul>
          </section>
        </aside>
      </div>
    </div>
  </main>
</template>
