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
  <main class="min-h-screen py-8" style="background-color: var(--sf-surface)">
    <div class="max-w-[1376px] mx-auto px-4 sm:px-6">
      <div class="grid grid-cols-1 md:grid-cols-[1fr_290px] lg:grid-cols-[270px_1fr_290px] gap-6">
        
        <!-- ======================================= -->
        <!-- 1. LEFT SIDEBAR: Navigation & Sections  -->
        <!-- ======================================= -->
        <aside class="hidden lg:block space-y-6">
          <!-- Navigation Links -->
          <SFCard flush class="p-4">
            <h2 class="text-xs font-bold text-slate-400 uppercase tracking-widest mb-3 dark:text-zinc-500">
              {{ t('home.sidebar.navTitle') }}
            </h2>
            <nav class="space-y-1" aria-label="首页辅助导航">
              <NuxtLink :to="localePath('/')" class="flex items-center gap-3 px-3 py-2.5 rounded-lg text-[14px] font-bold bg-[#E6F4F1] text-[#0F766E] dark:bg-teal-950/40 dark:text-teal-300">
                <UIcon name="i-lucide-home" class="size-4.5 shrink-0" />
                <span>{{ t('home.sidebar.navHome') }}</span>
              </NuxtLink>
              <a href="#categories" class="flex items-center gap-3 px-3 py-2.5 rounded-lg text-[14px] font-medium text-slate-700 hover:text-slate-900 hover:bg-slate-100 transition dark:text-zinc-300 dark:hover:bg-zinc-800 dark:hover:text-zinc-50">
                <UIcon name="i-lucide-folder-open" class="size-4.5 shrink-0" />
                <span>{{ t('home.sidebar.navCategories') }}</span>
              </a>
              <a href="#tags" class="flex items-center gap-3 px-3 py-2.5 rounded-lg text-[14px] font-medium text-slate-700 hover:text-slate-900 hover:bg-slate-100 transition dark:text-zinc-300 dark:hover:bg-zinc-800 dark:hover:text-zinc-50">
                <UIcon name="i-lucide-tag" class="size-4.5 shrink-0" />
                <span>{{ t('home.sidebar.navTags') }}</span>
              </a>
              <a href="#members" class="flex items-center gap-3 px-3 py-2.5 rounded-lg text-[14px] font-medium text-slate-700 hover:text-slate-900 hover:bg-slate-100 transition dark:text-zinc-300 dark:hover:bg-zinc-800 dark:hover:text-zinc-50">
                <UIcon name="i-lucide-users" class="size-4.5 shrink-0" />
                <span>{{ t('home.sidebar.navMembers') }}</span>
              </a>
            </nav>
          </SFCard>

          <!-- Categories Card -->
          <SFCard flush id="categories" class="p-4">
            <div class="flex justify-between items-center mb-3">
              <h2 class="text-xs font-bold text-slate-400 uppercase tracking-widest dark:text-zinc-500">
                {{ t('home.sidebar.sections') }}
              </h2>
              <SFBadge variant="neutral" class="font-bold">{{ totalCategoryThreads }}</SFBadge>
            </div>
            <ul class="space-y-1.5">
              <li v-for="(cat, idx) in categories" :key="cat.slug">
                <button
                  type="button"
                  class="flex w-full justify-between items-center px-3 py-2 rounded-lg text-[14px] font-medium transition"
                  :class="categoryButtonClass(cat)"
                  @click="selectCategory(cat)"
                >
                  <span class="flex items-center gap-2.5">
                    <span
                      class="w-2 h-2 rounded-full shrink-0"
                      :style="categoryDotStyle(idx)"
                    ></span>
                    <span>{{ cat.name }}</span>
                  </span>
                  <span class="text-xs text-slate-500 font-mono dark:text-zinc-400">{{ cat.topicCount }}</span>
                </button>
              </li>
            </ul>
          </SFCard>
        </aside>

        <!-- ======================================= -->
        <!-- 2. MIDDLE COLUMN: Threads Feed Stream   -->
        <!-- ======================================= -->
        <section class="space-y-4">
          <!-- Search & Filters -->
          <div class="space-y-3">
            <SFSearch
              v-model="searchQuery"
              :placeholder="t('home.searchPlaceholder')"
              id="feed-search"
            />
            
            <div class="flex items-center justify-between">
              <SFTabs
                v-model="currentTab"
                :items="tabItems"
                aria-label="帖子排序切换"
              />
            </div>
          </div>

          <!-- Loader / List / Empty State -->
          <div class="space-y-3" id="feed-list-container">
            <!-- Loading Skeleton -->
            <template v-if="isPending">
              <SFCard v-for="i in 3" :key="i" class="p-5">
                <SFSkeleton width="30%" height="1.25rem" class="mb-3" />
                <SFSkeleton width="100%" class="mb-2" />
                <SFSkeleton width="80%" class="mb-4" />
                <div class="flex items-center gap-3">
                  <SFSkeleton width="24px" height="24px" shape="circle" />
                  <SFSkeleton width="15%" />
                  <SFSkeleton width="10%" class="ml-auto" />
                </div>
              </SFCard>
            </template>

            <!-- Thread Items -->
            <template v-else-if="topics.length > 0">
              <SFCard class="divide-y divide-slate-100 overflow-hidden dark:divide-zinc-800">
                <div v-for="topic in topics" :key="topic.id">
                  <NuxtLink
                    :to="localePath(forumTopicPath(topic))"
                    class="block transition hover:bg-slate-50 dark:hover:bg-zinc-900/60"
                  >
                    <SFFeedRow
                      :title="topic.title"
                      :excerpt="topic.excerpt"
                      :author="topicAuthor(topic)"
                      :meta="topicMeta(topic)"
                      :replies="topic.commentCount"
                      :views="topic.viewCount"
                      :score="0"
                      :badges="topicBadges(topic)"
                    />
                  </NuxtLink>
                </div>
              </SFCard>
            </template>

            <!-- Empty State -->
            <template v-else>
              <SFCard class="p-12 flex justify-center">
                <SFEmptyState
                  :title="t('home.emptyState.title')"
                  :description="t('home.emptyState.description')"
                />
              </SFCard>
            </template>
          </div>

          <!-- Pagination -->
          <div v-if="topics.length > 0 && !isPending" class="flex justify-center pt-4">
            <SFPagination
              v-model:page="currentPage"
              :total-pages="totalPages"
            />
          </div>
        </section>

        <!-- ======================================= -->
        <!-- 3. RIGHT SIDEBAR: Tools & Statistics     -->
        <!-- ======================================= -->
        <aside class="hidden md:block space-y-6">
          <!-- User Status Panel -->
          <SFCard flush class="p-5 text-center">
            <template v-if="user">
              <div class="flex flex-col items-center gap-2">
                <SFAvatar :name="user.displayName" size="lg" status="online" />
                <h2 class="font-bold text-slate-800 text-lg mt-1 dark:text-zinc-100">{{ user.displayName }}</h2>
                <p class="text-sm text-slate-500 dark:text-zinc-400">@{{ user.username }}</p>
                
                <div class="grid grid-cols-2 gap-4 w-full mt-4 pt-4 border-t border-slate-100 dark:border-zinc-800">
                  <div>
                    <span class="block text-base font-bold text-slate-800 dark:text-zinc-100">--</span>
                    <span class="text-xs text-slate-400 uppercase font-semibold dark:text-zinc-500">{{ t('home.sidebar.userPosts') }}</span>
                  </div>
                  <div>
                    <span class="block text-base font-bold text-slate-800 dark:text-zinc-100">--</span>
                    <span class="text-xs text-slate-400 uppercase font-semibold dark:text-zinc-500">{{ t('home.sidebar.userLikes') }}</span>
                  </div>
                </div>
              </div>
            </template>
            <template v-else>
              <div class="space-y-3">
                <div class="w-12 h-12 bg-[#E6F4F1] text-[#0F766E] rounded-full flex items-center justify-center mx-auto dark:bg-teal-950/40 dark:text-teal-300">
                  <UIcon name="i-lucide-message-circle" class="size-5" />
                </div>
                <h2 class="font-bold text-slate-800 text-sm dark:text-zinc-100">{{ t('home.sidebar.welcomeTitle', { siteName }) }}</h2>
                <p class="text-xs text-slate-600 leading-relaxed dark:text-zinc-400">{{ t('home.sidebar.welcomeDesc') }}</p>
                <div class="grid grid-cols-2 gap-2 pt-2">
                  <NuxtLink :to="localePath('/login')" class="sf-button sf-button--ghost sf-button--sm block text-center">
                    {{ t('home.sidebar.loginBtn') }}
                  </NuxtLink>
                  <NuxtLink :to="localePath('/register')" class="sf-button sf-button--primary sf-button--sm block text-center">
                    {{ t('home.sidebar.registerBtn') }}
                  </NuxtLink>
                </div>
              </div>
            </template>
          </SFCard>

          <!-- Check In Card -->
          <SFCard flush v-if="user" class="p-4 flex items-center justify-between gap-3">
            <div class="min-w-0">
              <h3 class="text-sm font-bold text-slate-800 uppercase tracking-wide dark:text-zinc-100">
                {{ t('home.sidebar.checkIn') }}
              </h3>
              <p class="text-xs text-slate-500 mt-1 truncate dark:text-zinc-400">
                {{ checkedIn ? t('home.sidebar.checkedIn', { days: checkInDays }) : t('home.sidebar.checkInDesc') }}
              </p>
            </div>
            <SFButton
              :variant="checkedIn ? 'ghost' : 'primary'"
              size="sm"
              :disabled="checkedIn"
              @click="handleCheckIn"
              class="transition-transform active:scale-95 shrink-0"
            >
              {{ checkedIn ? t('home.sidebar.checkedInBtn') : t('home.sidebar.checkInBtn') }}
            </SFButton>
          </SFCard>

          <!-- Tags Card -->
          <SFCard flush class="p-4" id="tags">
            <h2 class="text-xs font-bold text-slate-400 uppercase tracking-widest mb-3 dark:text-zinc-500">
              {{ t('home.sidebar.navTags') }}
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
            <div v-else class="py-4">
              <SFEmptyState
                icon-label="TAG"
                :title="t('home.emptyState.title')"
                :description="t('home.emptyState.description')"
              />
            </div>
          </SFCard>

          <!-- Hot Discussions Card -->
          <SFCard flush class="p-4">
            <h2 class="text-xs font-bold text-slate-400 uppercase tracking-widest mb-3 dark:text-zinc-500">
              {{ t('home.sidebar.hotThreads') }}
            </h2>
            <ul v-if="hotTopics.length" class="space-y-3">
              <li v-for="(topic, index) in hotTopics" :key="topic.id" class="flex gap-3 items-start">
                <span
                  class="w-[18px] h-[18px] rounded text-[10px] font-bold flex items-center justify-center shrink-0 mt-0.5 px-1"
                  :class="hotTopicRankClass(index)"
                >
                  {{ hotTopicRank(index) }}
                </span>
                <div class="min-w-0 flex-1">
                  <NuxtLink :to="localePath(forumTopicPath(topic))" class="text-sm text-slate-700 hover:text-[#0F766E] hover:underline font-medium block truncate dark:text-zinc-300 dark:hover:text-teal-300">
                    {{ topic.title }}
                  </NuxtLink>
                  <span class="text-xs text-slate-400 font-mono mt-0.5 block dark:text-zinc-500">{{ t('home.sidebar.repliesCount', { count: topic.commentCount }) }}</span>
                </div>
              </li>
            </ul>
            <SFEmptyState
              v-else
              icon-label="HOT"
              :title="t('home.emptyState.title')"
              :description="t('home.emptyState.description')"
            />
          </SFCard>

          <!-- Forum Stats Card -->
          <SFCard flush class="p-4">
            <h2 class="text-xs font-bold text-slate-400 uppercase tracking-widest mb-3 dark:text-zinc-500">
              {{ t('home.sidebar.forumStats') }}
            </h2>
            <ul class="space-y-2.5 text-sm text-slate-700 font-medium dark:text-zinc-300">
              <li class="flex justify-between py-0.5">
                <span class="text-slate-500 font-normal dark:text-zinc-400">{{ t('home.sidebar.statThreads') }}</span>
                <span class="font-semibold font-mono text-slate-800 dark:text-zinc-100">{{ topicList.total || totalCategoryThreads }}</span>
              </li>
              <li class="flex justify-between py-0.5">
                <span class="text-slate-500 font-normal dark:text-zinc-400">{{ t('home.sidebar.statReplies') }}</span>
                <span class="font-semibold font-mono text-slate-800 dark:text-zinc-100">{{ totalCategoryComments }}</span>
              </li>
              <li class="flex justify-between py-0.5">
                <span class="text-slate-500 font-normal dark:text-zinc-400">{{ t('home.sidebar.statMembers') }}</span>
                <span class="font-semibold font-mono text-slate-800 dark:text-zinc-100">--</span>
              </li>
              <li class="flex justify-between py-0.5">
                <span class="text-slate-500 font-normal dark:text-zinc-400">{{ t('home.sidebar.statOnline') }}</span>
                <span class="font-semibold font-mono text-slate-800 flex items-center gap-2 dark:text-zinc-100">
                  <span class="w-2 h-2 rounded-full bg-green-400 pulse-dot"></span>
                  <span>--</span>
                </span>
              </li>
            </ul>
          </SFCard>
        </aside>

      </div>
    </div>
  </main>
</template>

<style scoped>
.pulse-dot {
  animation: pulse 2.4s ease-in-out infinite;
}
@keyframes pulse {
  0%, 100% { opacity: 1; }
  50% { opacity: .4; }
}
</style>
