<script setup lang="ts">
/**
 * 宿主 body 岛：forum.tag.show。主题 L1 挂载；路由页仅 SEO + fail-closed 回退。
 */

import {
  formatForumTopicListTotal,
  forumTagPath,
  forumTopicPath,
  parseForumTagPublicPagesOption,
  type ForumCategoryGroup,
  type ForumTag,
  type ForumTopicList,
  type ForumTopicSummary
} from '~/utils/forumTaxonomy'

// 使用 composer 闭包，避免模板直接绑定 $setup.t（async setup + 主题岛 SSR 下偶发非函数）
const i18n = useI18n()
const translate = (key: string, params?: Record<string, unknown>) => {
  const value = params ? i18n.t(key, params) : i18n.t(key)
  return typeof value === 'string' ? value : key
}
const localePath = useLocalePath()
const route = useRoute()
const { seoSettings, webOption } = useWebOptions()
const topicUrlMode = computed(() => seoSettings.value.topicUrlMode)

const forumApi = useForumApi()
const { can } = usePermissions()

const tagSlug = computed(() => routeParam(route.params.tagSlug))
const publicTagPagesEnabled = computed(() => parseForumTagPublicPagesOption(
  webOption('forum.tags.public_pages', 'enabled')
))
const currentPage = computed(() => parsePublicPage(route.query.page))
const canCreateTopic = computed(() => can(FORUM_PERMISSIONS.topicCreate))
const renderedAt = useState<number>('forum-taxonomy-rendered-at', () => Date.now())
const emptyTopicList = (): ForumTopicList => ({
  items: [],
  total: 0,
  page: 1,
  perPage: 20
})

if (!publicTagPagesEnabled.value) {
  throw createError({
    statusCode: 404,
    statusMessage: 'Tag pages are disabled'
  })
}

// 三个公开读请求互不依赖，并发启动可避免 SSR 首屏按接口耗时逐个累加。
const [categoryGroupsResult, activeTagsResult, topicListResult] = await Promise.all([
  useAsyncData(
    'forum-tag-page-category-groups',
    () => forumApi.listCategoryGroups(),
    { default: () => [] as ForumCategoryGroup[] }
  ),
  useAsyncData(
    'forum-tag-page-tags',
    async () => (await forumApi.listTags()).filter((item) => item.status === 'active'),
    { default: () => [] as ForumTag[] }
  ),
  useAsyncData(
    () => `forum-tag-page-topics:${tagSlug.value}:${currentPage.value}`,
    () => forumApi.listTopics({
      tagSlug: tagSlug.value,
      page: currentPage.value
    }),
    {
      default: emptyTopicList,
      watch: [currentPage, tagSlug]
    }
  )
] as const)

const { data: categoryGroupsData, pending: categoriesPending } = categoryGroupsResult
const { data: activeTagsData } = activeTagsResult
const { data: topicListData, pending: topicsPending } = topicListResult

// HMR 与响应式 key 切换期间 AsyncData 可能短暂清空；模板始终只读取稳定形状。
const categoryGroups = computed(() => categoryGroupsData.value || [])
const activeTags = computed(() => activeTagsData.value || [])
const topicList = computed<ForumTopicList>(() => {
  const value = topicListData.value
  return value && Array.isArray(value.items) ? value : emptyTopicList()
})
const categories = computed(() => categoryGroups.value.flatMap((group) => group.categories || []))
const totalTopics = computed(() => categories.value.reduce((sum, item) => sum + item.topicCount, 0))
const tag = computed(() => activeTags.value.find((item) => item.slug === tagSlug.value))


if (!tag.value) {
  throw createError({
    statusCode: 404,
    statusMessage: 'Tag not found'
  })
}

const topics = computed(() => topicList.value.items)
const totalPages = computed(() => Math.ceil(topicList.value.total / Math.max(topicList.value.perPage, 1)) || 1)
const hasActiveTags = computed(() => activeTags.value.length > 0)
const hasTopics = computed(() => topics.value.length > 0)
const showPagination = computed(() => hasTopics.value && totalPages.value > 1)
const tagsHeading = computed(() => translate('home.tags'))
const topicListTotalLabel = computed(() => formatForumTopicListTotal(topicList.value, translate))
const emptyStateTitle = computed(() => translate('home.emptyState.title'))
const emptyStateDescription = computed(() => translate('home.emptyState.description'))

useSForumSeo(computed(() => ({
  type: 'tag',
  path: currentPage.value > 1 ? `${forumTagPath(tagSlug.value)}?page=${currentPage.value}` : forumTagPath(tagSlug.value),
  title: tag.value?.name || tagSlug.value,
  description: tag.value?.description,
  public: tag.value?.status === 'active',
  noindex: topicList.value.total === 0,
  variables: { tagName: tag.value?.name || tagSlug.value },
  breadcrumbs: [
    { name: seoSettings.value.seoSiteName, path: '/' },
    { name: tag.value?.name || tagSlug.value, path: forumTagPath(tagSlug.value) }
  ]
})))



function routeParam(value: string | string[] | undefined) {
  return Array.isArray(value) ? value[0] || '' : value || ''
}

function tagPageTo(page: number) {
  return publicPageLocation(localePath(forumTagPath(tagSlug.value)), page)
}

function topicRelativeLabel(iso: string) {
  const date = new Date(iso)
  if (Number.isNaN(date.getTime())) {
    return ''
  }

  const diffMs = renderedAt.value - date.getTime()
  const minute = 60 * 1000
  const hour = 60 * minute
  const day = 24 * hour

  if (diffMs >= 0 && diffMs < hour) {
    return translate('home.feed.activityMinutes', { count: Math.max(1, Math.floor(diffMs / minute)) })
  }
  if (diffMs >= 0 && diffMs < day) {
    return translate('home.feed.activityHours', { count: Math.max(1, Math.floor(diffMs / hour)) })
  }
  if (diffMs >= 0 && diffMs < 7 * day) {
    return translate('home.feed.activityDays', { count: Math.max(1, Math.floor(diffMs / day)) })
  }

  return date.toISOString().slice(0, 10)
}

function topicCreated(topic: ForumTopicSummary) {
  return topicRelativeLabel(topic.createdAt)
}

function topicActivity(topic: ForumTopicSummary) {
  return topicRelativeLabel(topic.lastActivityAt || topic.createdAt)
}

</script>

<template>

<main class="sforum-home">
    <div class="sforum-home__layout">
      <div class="sforum-home__sidebar">
        <SFHomeNavigation
          desktop-only
          navigation-mode="route"
          :categories="categories"
          selected-category-slug=""
          :total-topics="totalTopics"
          :pending="categoriesPending"
          :can-create-topic="canCreateTopic"
        />
      </div>

      <section class="sforum-home__main" aria-labelledby="tag-page-title">
        <div class="sforum-home__mobile-nav">
          <SFHomeNavigation
            mobile-only
            navigation-mode="route"
            :categories="categories"
            selected-category-slug=""
            :total-topics="totalTopics"
            :pending="categoriesPending"
            :can-create-topic="canCreateTopic"
          />
        </div>

        <header class="sforum-home__page-header">
          <p class="sforum-home__page-group">{{ tagsHeading }}</p>
          <h1 id="tag-page-title">#{{ tag?.name }}</h1>
          <p v-if="tag?.description">{{ tag.description }}</p>
          <div class="sforum-home__page-meta">
            {{ topicListTotalLabel }}
          </div>
        </header>

        <div v-if="hasActiveTags" class="sforum-home__filters">
          <div class="sforum-home__tag-list">
            <NuxtLink
              v-for="item in activeTags"
              :key="item.slug"
              :to="localePath(forumTagPath(item.slug))"
              class="sforum-home__tag"
              :class="{ 'is-active': item.slug === tagSlug }"
            >
              <span>#{{ item.name }}</span>
            </NuxtLink>
          </div>
        </div>

        <div
          class="mt-3 overflow-hidden rounded-[var(--sf-public-radius,6px)] border border-[var(--sf-public-border)] bg-[var(--sf-public-surface)] shadow-[var(--sf-public-shadow)]"
          data-sf-region="topic-list"
        >
          <template v-if="topicsPending">
            <div
              v-for="item in 6"
              :key="item"
              class="border-b border-[var(--sf-border-light,#eef0f3)] p-3.5 last:border-b-0"
            >
              <SFSkeleton avatar :lines="2" />
            </div>
          </template>

          <template v-else-if="hasTopics">
            <SFHomeTopicRow
              v-for="topic in topics"
              :key="topic.id"
              :topic="topic"
              :to="localePath(forumTopicPath(topic, topicUrlMode))"
              :created-label="topicCreated(topic)"
              :activity-label="topicActivity(topic)"
              :extension-list-badges="topicList.extensionListBadges || []"
            />
          </template>

          <div v-else class="px-4 py-10 text-center">
            <SFEmptyState
              :title="emptyStateTitle"
              :description="emptyStateDescription"
            />
          </div>
        </div>

        <div v-if="showPagination && !topicsPending" class="mt-3">
          <SFPagination
            :page="currentPage"
            :total-pages="totalPages"
            :page-to="tagPageTo"
          />
        </div>
      </section>
    </div>
  </main>
</template>
