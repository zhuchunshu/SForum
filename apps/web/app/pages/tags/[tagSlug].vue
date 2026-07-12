<script setup lang="ts">
import {
  forumTagPath,
  forumTopicPath,
  parseForumTagPublicPagesOption,
  type ForumCategoryGroup,
  type ForumTag,
  type ForumTopicList,
  type ForumTopicSummary
} from '~/utils/forumTaxonomy'

const { t } = useI18n()
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
const currentPage = ref(1)
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

const { data: categoryGroups, pending: categoriesPending } = await useAsyncData(
  'forum-tag-page-category-groups',
  () => forumApi.listCategoryGroups(),
  { default: () => [] as ForumCategoryGroup[] }
)

const { data: activeTags } = await useAsyncData(
  'forum-tag-page-tags',
  async () => (await forumApi.listTags()).filter((item) => item.status === 'active'),
  { default: () => [] as ForumTag[] }
)

const categories = computed(() => categoryGroups.value.flatMap((group) => group.categories || []))
const totalTopics = computed(() => categories.value.reduce((sum, item) => sum + item.topicCount, 0))
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
    page: currentPage.value
  }),
  {
    default: emptyTopicList,
    watch: [currentPage, tagSlug]
  }
)

const topics = computed(() => topicList.value.items)
const totalPages = computed(() => Math.ceil(topicList.value.total / Math.max(topicList.value.perPage, 1)) || 1)

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

watch(tagSlug, () => {
  currentPage.value = 1
})
</script>

<template>
  <SFPageOutlet page="forum.tag.show">
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
          <p class="sforum-home__page-group">{{ t('home.tags') }}</p>
          <h1 id="tag-page-title">#{{ tag?.name }}</h1>
          <p v-if="tag?.description">{{ tag.description }}</p>
          <div class="sforum-home__page-meta">
            {{ t('home.feed.topicCountMeta', { count: topicList.total }) }}
          </div>
        </header>

        <div v-if="activeTags.length" class="sforum-home__filters">
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

        <div class="sforum-home__topic-table sforum-home__topic-table--standalone">
          <div class="sforum-home__topic-head" aria-hidden="true">
            <span>{{ t('home.feed.topicColumn') }}</span>
            <span>{{ t('home.feed.authorColumn') }}</span>
            <span>{{ t('home.feed.repliesColumn') }}</span>
            <span>{{ t('home.feed.activityColumn') }}</span>
          </div>

          <template v-if="topicsPending">
            <div v-for="item in 6" :key="item" class="sforum-home__skeleton-row">
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
              :title="t('home.emptyState.title')"
              :description="t('home.emptyState.description')"
            />
          </div>
        </div>

        <div v-if="topics.length > 0 && !topicsPending && totalPages > 1" class="sforum-home__pagination">
          <SFPagination
            v-model:page="currentPage"
            :total-pages="totalPages"
          />
        </div>
      </section>
    </div>
  </main>

  </SFPageOutlet>
</template>
