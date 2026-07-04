<script setup lang="ts">
const { t } = useI18n()
const localePath = useLocalePath()
const { user } = useAuthSession()

useSeoMeta({
  title: () => t('home.metaTitle'),
  description: () => t('home.metaDescription'),
  ogTitle: () => t('home.metaTitle'),
  ogDescription: () => t('home.metaDescription'),
  ogType: 'website'
})

// Mock Categories for Sidebar
interface Category {
  name: string
  key: string
  count: number
}
const categories = computed<Category[]>(() => [
  { name: t('home.sidebar.secTech'), key: 'tech', count: 184 },
  { name: t('home.sidebar.secCreative'), key: 'creative', count: 42 },
  { name: t('home.sidebar.secLife'), key: 'life', count: 96 },
  { name: t('home.sidebar.secNotice'), key: 'notice', count: 8 }
])

// Mock Active Threads for Information Flow
interface Thread {
  id: number
  title: string
  excerpt: string
  category: string
  categoryKey: string
  author: string
  replies: number
  views: number
  score: number
  timeAgo: string
  isPinned?: boolean
  isFeatured?: boolean
}
const threads = computed<Thread[]>(() => [
  {
    id: 1,
    title: 'Vue 3.5 新特性深度解析与最佳实践',
    excerpt: 'Vue 3.5 版本引入了响应式解构性能提升、全新 useTemplateRef 机制以及对 custom elements 的增强支持。本文将逐个分析这些优化点，并结合生产实例探讨如何优化现有的代码仓库结构…',
    category: t('home.sidebar.secTech'),
    categoryKey: 'tech',
    author: '尤雨溪小号',
    replies: 28,
    views: 412,
    score: 45,
    timeAgo: '10分钟前',
    isPinned: true,
    isFeatured: true
  },
  {
    id: 2,
    title: '关于论坛首页三栏布局设计的建议与吐槽收集帖',
    excerpt: 'SForum 首页现已采用经典的三栏式门户布局，大家对目前的整体视觉、各个端上的响应式细节、以及组件排列有什么想法？欢迎在此帖畅所欲言，我们的设计师和开发者会每天跟进优化。',
    category: t('home.sidebar.secNotice'),
    categoryKey: 'notice',
    author: '管理员',
    replies: 142,
    views: 1024,
    score: 99,
    timeAgo: '1小时前',
    isPinned: true
  },
  {
    id: 3,
    title: '用 Go Fiber v3 搭建高性能 API 的踩坑记录与性能分析',
    excerpt: 'Go Fiber v3 目前在底层的 fasthttp 集成上有了更多改动。本文详细梳理了在处理中间件链条、大文件上传流式解析、Redis 连接池生命周期管理等场景下的常见坑点，以及最终的压测对比数据。',
    category: t('home.sidebar.secTech'),
    categoryKey: 'tech',
    author: '蓝猫',
    replies: 12,
    views: 96,
    score: 18,
    timeAgo: '2小时前'
  },
  {
    id: 4,
    title: '我的第一个毛玻璃插画作品，欢迎大家提意见！',
    excerpt: '使用 CSS backdrop-filter 和 SVG 混合发光渐变绘制了一款概念风格的后台卡片插图，大家感觉透明度和背景噪点会不会过重？',
    category: t('home.sidebar.secCreative'),
    categoryKey: 'creative',
    author: '像素艺术家',
    replies: 8,
    views: 64,
    score: 15,
    timeAgo: '3小时前'
  },
  {
    id: 5,
    title: '今天天气不错，分享一下我们这里的日落美景',
    excerpt: '下班路过海滩随手拍的，松石绿色的晚霞和海面真的让人心情平静。周末有空的话大家也可以多出门走走晒晒太阳。',
    category: t('home.sidebar.secLife'),
    categoryKey: 'life',
    author: '追光者',
    replies: 3,
    views: 45,
    score: 7,
    timeAgo: '5小时前'
  }
])

// Mock Hot Discussions
const hotTopics = ref([
  { id: 1, title: '如何评价 SForum 刚刚发布的松石绿设计风格？', replies: 89 },
  { id: 2, title: '2026 年前端开发在大陆找工作现状探讨', replies: 74 },
  { id: 3, title: 'Go 语言新版本的并发控制特性有哪些升级？', replies: 56 },
  { id: 4, title: '写 Markdown 长文时，你更在乎预览同步还是编辑流顺畅？', replies: 38 },
  { id: 5, title: 'ALTCHA 独立部署的真实资源消耗表现怎么样？', replies: 21 }
])

// Search & Filter state
const searchQuery = ref('')
const currentTab = ref('latest')
const currentPage = ref(1)
const isPending = ref(false)

// SFTabs configuration
const tabItems = computed(() => [
  { label: t('home.filter.latest'), value: 'latest' },
  { label: t('home.filter.hot'), value: 'hot' },
  { label: t('home.filter.featured'), value: 'featured' },
  { label: t('home.filter.following'), value: 'following', disabled: !user.value }
])

// Computed filtered threads based on Search and Tab selection
const filteredThreads = computed(() => {
  let result = [...threads.value]
  
  // 1. Filter by Search Query
  if (searchQuery.value.trim()) {
    const query = searchQuery.value.toLowerCase()
    result = result.filter(t => t.title.toLowerCase().includes(query) || t.excerpt.toLowerCase().includes(query))
  }
  
  // 2. Filter / Sort by Tab
  if (currentTab.value === 'featured') {
    result = result.filter(t => t.isFeatured)
  } else if (currentTab.value === 'following') {
    const followedAuthors = ['尤雨溪小号', '蓝猫']
    result = result.filter(t => followedAuthors.includes(t.author))
  } else if (currentTab.value === 'hot') {
    result.sort((a, b) => b.replies - a.replies)
  } else if (currentTab.value === 'latest') {
    // Default mock sequence matches latest
  }
  
  return result
})

const ITEMS_PER_PAGE = 3
const paginatedThreads = computed(() => {
  const start = (currentPage.value - 1) * ITEMS_PER_PAGE
  const end = start + ITEMS_PER_PAGE
  return filteredThreads.value.slice(start, end)
})
const totalPages = computed(() => {
  return Math.ceil(filteredThreads.value.length / ITEMS_PER_PAGE) || 1
})

// Watch tab selection to trigger mock loading skeleton
watch(currentTab, (newVal, oldVal, onCleanup) => {
  currentPage.value = 1
  isPending.value = true
  const timer = setTimeout(() => {
    isPending.value = false
  }, 400)
  onCleanup(() => clearTimeout(timer))
})

// Watch search query to trigger mock loading skeleton
watch(searchQuery, (newVal, oldVal, onCleanup) => {
  currentPage.value = 1
  isPending.value = true
  const timer = setTimeout(() => {
    isPending.value = false
  }, 300)
  onCleanup(() => clearTimeout(timer))
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
const totalCategoryThreads = computed(() => categories.value.reduce((acc, cur) => acc + cur.count, 0))
</script>

<template>
  <main class="min-h-screen bg-[#F7F8FA] py-8">
    <div class="max-w-[1376px] mx-auto px-4 sm:px-6">
      <div class="grid grid-cols-1 md:grid-cols-[1fr_290px] lg:grid-cols-[270px_1fr_290px] gap-6">
        
        <!-- ======================================= -->
        <!-- 1. LEFT SIDEBAR: Navigation & Sections  -->
        <!-- ======================================= -->
        <aside class="hidden lg:block space-y-6">
          <!-- Navigation Links -->
          <SFCard class="p-4">
            <h2 class="text-xs font-bold text-slate-400 uppercase tracking-wider mb-3">
              {{ t('home.sidebar.navTitle') }}
            </h2>
            <nav class="space-y-1" aria-label="首页辅助导航">
              <NuxtLink :to="localePath('/')" class="flex items-center gap-3 px-3 py-2 rounded-lg text-sm font-semibold bg-[#E6F4F1] text-[#0F766E]">
                <span class="text-lg">🏠</span>
                <span>{{ t('home.sidebar.navHome') }}</span>
              </NuxtLink>
              <a href="#categories" class="flex items-center gap-3 px-3 py-2 rounded-lg text-sm text-slate-600 hover:text-slate-900 hover:bg-slate-100 transition">
                <span class="text-lg">📂</span>
                <span>{{ t('home.sidebar.navCategories') }}</span>
              </a>
              <a href="#tags" class="flex items-center gap-3 px-3 py-2 rounded-lg text-sm text-slate-600 hover:text-slate-900 hover:bg-slate-100 transition">
                <span class="text-lg">🏷️</span>
                <span>{{ t('home.sidebar.navTags') }}</span>
              </a>
              <a href="#members" class="flex items-center gap-3 px-3 py-2 rounded-lg text-sm text-slate-600 hover:text-slate-900 hover:bg-slate-100 transition">
                <span class="text-lg">👥</span>
                <span>{{ t('home.sidebar.navMembers') }}</span>
              </a>
            </nav>
          </SFCard>

          <!-- Categories Card -->
          <SFCard id="categories" class="p-4">
            <div class="flex justify-between items-center mb-3">
              <h2 class="text-xs font-bold text-slate-400 uppercase tracking-wider">
                {{ t('home.sidebar.sections') }}
              </h2>
              <SFBadge variant="neutral">{{ totalCategoryThreads }}</SFBadge>
            </div>
            <ul class="space-y-2">
              <li v-for="cat in categories" :key="cat.key">
                <a href="#" class="flex justify-between items-center px-3 py-1.5 rounded-lg text-sm text-slate-600 hover:text-slate-900 hover:bg-slate-100 transition">
                  <span class="flex items-center gap-2">
                    <span class="w-1.5 h-1.5 rounded-full bg-[#0F766E]"></span>
                    <span>{{ cat.name }}</span>
                  </span>
                  <span class="text-xs text-slate-400 font-mono">{{ cat.count }}</span>
                </a>
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
            <template v-else-if="filteredThreads.length > 0">
              <SFCard class="divide-y divide-slate-100 overflow-hidden">
                <div v-for="thread in paginatedThreads" :key="thread.id">
                  <SFFeedRow
                    :title="thread.title"
                    :author="thread.author"
                    :meta="thread.timeAgo"
                    :replies="thread.replies"
                    :views="thread.views"
                    :score="thread.score"
                    :badges="[
                      ...(thread.isPinned ? [{ label: t('home.badge.pinned'), variant: 'danger' as const }] : []),
                      ...(thread.isFeatured ? [{ label: t('home.badge.featured'), variant: 'success' as const }] : []),
                      { label: thread.category, variant: 'primary' as const }
                    ]"
                  />
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
          <div v-if="filteredThreads.length > 0 && !isPending" class="flex justify-center pt-4">
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
          <SFCard class="p-5 text-center">
            <template v-if="user">
              <div class="flex flex-col items-center gap-2">
                <SFAvatar :name="user.displayName" size="lg" status="online" />
                <h2 class="font-bold text-slate-800 text-base mt-1">{{ user.displayName }}</h2>
                <p class="text-xs text-slate-400">@{{ user.username }}</p>
                
                <div class="grid grid-cols-2 gap-4 w-full mt-4 pt-4 border-t border-slate-100">
                  <div>
                    <span class="block text-sm font-bold text-slate-800">12</span>
                    <span class="text-[10px] text-slate-400 uppercase font-semibold">{{ t('home.sidebar.userPosts') }}</span>
                  </div>
                  <div>
                    <span class="block text-sm font-bold text-slate-800">84</span>
                    <span class="text-[10px] text-slate-400 uppercase font-semibold">{{ t('home.sidebar.userLikes') }}</span>
                  </div>
                </div>
              </div>
            </template>
            <template v-else>
              <div class="space-y-3">
                <div class="w-12 h-12 bg-[#E6F4F1] text-[#0F766E] rounded-full flex items-center justify-center text-xl mx-auto">
                  💬
                </div>
                <h2 class="font-bold text-slate-800 text-sm">{{ t('home.sidebar.welcomeTitle') }}</h2>
                <p class="text-xs text-slate-500 leading-relaxed">{{ t('home.sidebar.welcomeDesc') }}</p>
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
          <SFCard v-if="user" class="p-4 flex items-center justify-between gap-3">
            <div class="min-w-0">
              <h3 class="text-xs font-bold text-slate-800 uppercase tracking-wide">
                {{ t('home.sidebar.checkIn') }}
              </h3>
              <p class="text-[10px] text-slate-400 mt-0.5 truncate">
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

          <!-- Hot Discussions Card -->
          <SFCard class="p-4" id="tags">
            <h2 class="text-xs font-bold text-slate-400 uppercase tracking-wider mb-3">
              {{ t('home.sidebar.hotThreads') }}
            </h2>
            <ul class="space-y-2.5">
              <li v-for="(topic, index) in hotTopics" :key="topic.id" class="flex gap-2.5 items-start">
                <span
                  class="w-4 h-4 rounded text-[9px] font-bold flex items-center justify-center shrink-0 mt-0.5"
                  :class="[
                    index === 0 ? 'bg-red-500 text-white' : '',
                    index === 1 ? 'bg-orange-400 text-white' : '',
                    index === 2 ? 'bg-yellow-400 text-slate-800' : '',
                    index > 2 ? 'bg-slate-200 text-slate-600' : ''
                  ]"
                >
                  {{ index + 1 }}
                </span>
                <div class="min-w-0 flex-1">
                  <a href="#" class="text-xs text-slate-700 hover:text-[#0F766E] hover:underline font-medium block truncate">
                    {{ topic.title }}
                  </a>
                  <span class="text-[9px] text-slate-400 font-mono">{{ t('home.sidebar.repliesCount', { count: topic.replies }) }}</span>
                </div>
              </li>
            </ul>
          </SFCard>

          <!-- Forum Stats Card -->
          <SFCard class="p-4">
            <h2 class="text-xs font-bold text-slate-400 uppercase tracking-wider mb-3">
              {{ t('home.sidebar.forumStats') }}
            </h2>
            <ul class="space-y-2 text-xs text-slate-600">
              <li class="flex justify-between py-0.5">
                <span class="text-slate-400">{{ t('home.sidebar.statThreads') }}</span>
                <span class="font-semibold font-mono text-slate-800">4,284</span>
              </li>
              <li class="flex justify-between py-0.5">
                <span class="text-slate-400">{{ t('home.sidebar.statReplies') }}</span>
                <span class="font-semibold font-mono text-slate-800">23,109</span>
              </li>
              <li class="flex justify-between py-0.5">
                <span class="text-slate-400">{{ t('home.sidebar.statMembers') }}</span>
                <span class="font-semibold font-mono text-slate-800">894</span>
              </li>
              <li class="flex justify-between py-0.5">
                <span class="text-slate-400">{{ t('home.sidebar.statOnline') }}</span>
                <span class="font-semibold font-mono text-slate-800 flex items-center gap-1.5">
                  <span class="w-1.5 h-1.5 rounded-full bg-green-400 pulse-dot"></span>
                  <span>1,024</span>
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
