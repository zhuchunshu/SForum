<script setup lang="ts">
import {
  forumTagPath,
  forumTopicPath,
  parseForumTagPublicPagesOption,
  type ForumTag,
  type ForumTopicList,
  type ForumTopicSummary
} from '~/utils/forumTaxonomy'

const { t } = useI18n()
const localePath = useLocalePath()
const route = useRoute()
const { siteName, seoSettings, webOption } = useWebOptions()
const topicUrlMode = computed(() => seoSettings.value.topicUrlMode)
const forumApi = useForumApi()

const ITEMS_PER_PAGE = 10

type FeedBadge = {
  label: string
  variant?: 'neutral' | 'primary' | 'info' | 'success' | 'warning' | 'danger'
}

const tagSlug = computed(() => routeParam(route.params.tagSlug))
const publicTagPagesEnabled = computed(() => parseForumTagPublicPagesOption(
  webOption('forum.tags.public_pages', 'enabled')
))
const currentPage = ref(1)
const emptyTopicList = (): ForumTopicList => ({
  items: [],
  total: 0,
  page: 1,
  perPage: ITEMS_PER_PAGE
})

if (!publicTagPagesEnabled.value) {
  throw createError({
    statusCode: 404,
    statusMessage: 'Tag pages are disabled'
  })
}

const { data: activeTags } = await useAsyncData(
  'forum-tag-page-tags',
  async () => (await forumApi.listTags()).filter((item) => item.status === 'active'),
  { default: () => [] as ForumTag[] }
)

const tag = computed(() => activeTags.value.find((item) => item.slug === tagSlug.value))

if (!tag.value) {
  throw createError({
    statusCode: 404,
    statusMessage: 'Tag not found'
  })
}

const { data: topicList, pending: topicsPending } = await useAsyncData(
  `forum-tag-page-topics:${tagSlug.value}`,
  () => forumApi.listTopics({
    tagSlug: tagSlug.value,
    page: currentPage.value,
    perPage: ITEMS_PER_PAGE
  }),
  {
    default: emptyTopicList,
    watch: [currentPage]
  }
)

const topics = computed(() => topicList.value.items)
const totalPages = computed(() => Math.ceil(topicList.value.total / Math.max(topicList.value.perPage, 1)) || 1)

useSForumSeo({
  title: () => `#${tag.value?.name || tagSlug.value} - ${siteName.value}`,
  description: () => tag.value?.description || seoSettings.value.metaDescription,
  path: () => forumTagPath(tagSlug.value),
  type: 'website',
  schema: { type: 'WebPage' }
})

function routeParam(value: string | string[] | undefined) {
  return Array.isArray(value) ? value[0] || '' : value || ''
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
    ...(topic.tags || []).map((item) => ({ label: `#${item.name}`, variant: 'neutral' as const }))
  ]
}

function tagLinkClass(item: ForumTag) {
  return item.slug === tagSlug.value
    ? 'border-[#0F766E] bg-[#E6F4F1] text-[#0F766E] dark:border-teal-700 dark:bg-teal-950/40 dark:text-teal-300'
    : 'border-slate-200 text-slate-700 hover:border-[#0F766E] hover:text-[#0F766E] dark:border-zinc-700 dark:text-zinc-300 dark:hover:border-teal-600 dark:hover:text-teal-300'
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
      <div class="grid grid-cols-1 lg:grid-cols-[270px_1fr] gap-6">
        <aside class="hidden lg:block space-y-6">
          <SFCard flush class="p-4">
            <h2 class="text-xs font-bold text-slate-400 uppercase tracking-widest mb-3 dark:text-zinc-500">
              {{ t('home.sidebar.navTags') }}
            </h2>
            <nav class="flex flex-wrap gap-2" aria-label="标签导航">
              <NuxtLink
                v-for="item in activeTags"
                :key="item.slug"
                :to="localePath(forumTagPath(item.slug))"
                class="inline-flex items-center gap-1.5 rounded-md border px-2.5 py-1.5 text-xs font-semibold transition"
                :class="tagLinkClass(item)"
              >
                <span>#{{ item.name }}</span>
                <span class="font-mono text-[11px] opacity-70">{{ item.topicCount }}</span>
              </NuxtLink>
            </nav>
          </SFCard>
        </aside>

        <section class="space-y-4">
          <SFCard flush class="p-5">
            <div class="flex flex-col gap-2 sm:flex-row sm:items-start sm:justify-between">
              <div>
                <p class="text-xs font-bold uppercase tracking-widest text-slate-400 dark:text-zinc-500">
                  {{ t('home.sidebar.navTags') }}
                </p>
                <h1 class="mt-1 text-2xl font-bold text-slate-900 dark:text-zinc-50">
                  #{{ tag?.name }}
                </h1>
                <p v-if="tag?.description" class="mt-2 text-sm leading-6 text-slate-600 dark:text-zinc-400">
                  {{ tag.description }}
                </p>
              </div>
              <SFBadge variant="neutral" class="font-bold">
                {{ topicList.total }}
              </SFBadge>
            </div>
          </SFCard>

          <div class="space-y-3">
            <template v-if="topicsPending">
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

            <template v-else-if="topics.length > 0">
              <SFCard class="divide-y divide-slate-100 overflow-hidden dark:divide-zinc-800">
                <div v-for="topic in topics" :key="topic.id">
                  <NuxtLink
                    :to="localePath(forumTopicPath(topic, topicUrlMode))"
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

            <SFCard v-else class="p-12 flex justify-center">
              <SFEmptyState
                :title="t('home.emptyState.title')"
                :description="t('home.emptyState.description')"
              />
            </SFCard>
          </div>

          <div v-if="topics.length > 0 && !topicsPending" class="flex justify-center pt-4">
            <SFPagination
              v-model:page="currentPage"
              :total-pages="totalPages"
            />
          </div>
        </section>
      </div>
    </div>
  </main>
</template>
