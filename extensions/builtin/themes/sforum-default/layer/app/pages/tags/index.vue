<script setup lang="ts">
import {
  forumTagPath,
  forumTagsIndexPath,
  isCreatedWithinDays,
  parseForumTagPublicPagesOption,
  tagCloudSizeBucket,
  tagHotThreshold,
  type ForumTag
} from '~/utils/forumTaxonomy'

type TagFilter = 'all' | 'hot' | 'week' | 'az'

const { t } = useI18n()
const localePath = useLocalePath()
const { seoSettings, webOption, siteName } = useWebOptions()
const forumApi = useForumApi()

const publicTagPagesEnabled = computed(() => parseForumTagPublicPagesOption(
  webOption('forum.tags.public_pages', 'enabled')
))

if (!publicTagPagesEnabled.value) {
  throw createError({
    statusCode: 404,
    statusMessage: 'Tag pages are disabled'
  })
}

const filter = ref<TagFilter>('all')
const searchQuery = ref('')

const { data: tags, pending } = await useAsyncData(
  'forum-tags-index',
  async () => (await forumApi.listTags()).filter((item) => item.status === 'active'),
  { default: () => [] as ForumTag[] }
)

const activeTags = computed(() => tags.value || [])

const countRange = computed(() => {
  if (activeTags.value.length === 0) {
    return { min: 0, max: 0 }
  }
  let min = Number.POSITIVE_INFINITY
  let max = 0
  for (const tag of activeTags.value) {
    const count = tag.topicCount || 0
    if (count < min) min = count
    if (count > max) max = count
  }
  if (!Number.isFinite(min)) min = 0
  return { min, max }
})

const hotThreshold = computed(() => tagHotThreshold(activeTags.value.map((tag) => tag.topicCount || 0)))

const weekNewCount = computed(() =>
  activeTags.value.filter((tag) => isCreatedWithinDays(tag.createdAt, 7)).length
)

const totalReferences = computed(() =>
  activeTags.value.reduce((sum, tag) => sum + (tag.topicCount || 0), 0)
)

const filteredTags = computed(() => {
  const query = searchQuery.value.trim().toLowerCase()
  let list = activeTags.value.slice()

  if (query) {
    list = list.filter((tag) =>
      tag.name.toLowerCase().includes(query)
      || tag.slug.toLowerCase().includes(query)
    )
  }

  switch (filter.value) {
    case 'hot':
      list = list.filter((tag) => (tag.topicCount || 0) >= hotThreshold.value)
      break
    case 'week':
      // 「本周」= 近 7 天新建标签（无「本周活跃」API 时的降级语义）
      list = list.filter((tag) => isCreatedWithinDays(tag.createdAt, 7))
      break
    case 'az':
      list = list.slice().sort((a, b) => a.name.localeCompare(b.name, undefined, { sensitivity: 'base' }))
      break
    default:
      break
  }

  return list
})

// 一期无周环比：降级为热门标签 Top 6（按 topicCount）
const risingTags = computed(() =>
  activeTags.value
    .slice()
    .sort((a, b) => (b.topicCount || 0) - (a.topicCount || 0) || a.name.localeCompare(b.name))
    .slice(0, 6)
)

useSForumSeo(computed(() => ({
  type: 'static' as const,
  path: forumTagsIndexPath(),
  title: t('taxonomy.tags.title'),
  description: t('taxonomy.tags.description'),
  public: true,
  breadcrumbs: [
    { name: seoSettings.value.seoSiteName || siteName.value, path: '/' },
    { name: t('taxonomy.tags.title'), path: forumTagsIndexPath() }
  ]
})))

function sizeClass(tag: ForumTag) {
  const bucket = tagCloudSizeBucket(tag.topicCount || 0, countRange.value.min, countRange.value.max)
  return `sforum-taxonomy__tag--s${bucket}`
}

function setFilter(next: TagFilter) {
  filter.value = next
}

function formatCount(value: number) {
  return new Intl.NumberFormat(undefined).format(value)
}
</script>

<template>
  <main class="sforum-taxonomy">
    <div class="sforum-taxonomy__shell">
      <header class="sforum-taxonomy__head">
        <div>
          <h1>{{ t('taxonomy.tags.title') }}</h1>
          <p>{{ t('taxonomy.tags.description') }}</p>
        </div>
        <div class="sforum-taxonomy__tools">
          <input
            v-model="searchQuery"
            type="search"
            class="sforum-taxonomy__filter"
            :placeholder="t('taxonomy.tags.filterPlaceholder')"
            :aria-label="t('taxonomy.tags.filterPlaceholder')"
          >
          <button
            type="button"
            class="sforum-taxonomy__chip"
            :class="{ 'is-active': filter === 'all' }"
            @click="setFilter('all')"
          >
            {{ t('taxonomy.tags.filters.all') }}
          </button>
          <button
            type="button"
            class="sforum-taxonomy__chip"
            :class="{ 'is-active': filter === 'hot' }"
            @click="setFilter('hot')"
          >
            {{ t('taxonomy.tags.filters.hot') }}
          </button>
          <button
            type="button"
            class="sforum-taxonomy__chip"
            :class="{ 'is-active': filter === 'week' }"
            @click="setFilter('week')"
          >
            {{ t('taxonomy.tags.filters.week') }}
          </button>
          <button
            type="button"
            class="sforum-taxonomy__chip"
            :class="{ 'is-active': filter === 'az' }"
            @click="setFilter('az')"
          >
            {{ t('taxonomy.tags.filters.az') }}
          </button>
        </div>
      </header>

      <section class="sforum-taxonomy__panel" :aria-busy="pending">
        <div v-if="pending" class="sforum-taxonomy__pending">
          <SFSkeleton :lines="4" />
          <SFSkeleton :lines="3" />
        </div>

        <template v-else-if="filteredTags.length">
          <div class="sforum-taxonomy__cloud">
            <NuxtLink
              v-for="tag in filteredTags"
              :key="tag.id"
              :to="localePath(forumTagPath(tag.slug))"
              class="sforum-taxonomy__tag"
              :class="sizeClass(tag)"
            >
              {{ tag.name }}
              <span class="sforum-taxonomy__tag-count">{{ formatCount(tag.topicCount || 0) }}</span>
            </NuxtLink>
          </div>
          <div class="sforum-taxonomy__legend">
            <span>{{ t('taxonomy.tags.legendSize') }}</span>
            <span>
              {{ t('taxonomy.tags.legendSummary', {
                tags: formatCount(activeTags.length),
                refs: formatCount(totalReferences)
              }) }}
            </span>
          </div>
        </template>

        <div v-else class="sforum-taxonomy__empty">
          <SFEmptyState
            :title="t('taxonomy.tags.emptyTitle')"
            :description="t('taxonomy.tags.emptyDescription')"
          />
        </div>
      </section>

      <div class="sforum-taxonomy__stats">
        <div class="sforum-taxonomy__stat">
          <div class="sforum-taxonomy__stat-key">{{ t('taxonomy.tags.stats.total') }}</div>
          <div class="sforum-taxonomy__stat-value">{{ formatCount(activeTags.length) }}</div>
        </div>
        <div class="sforum-taxonomy__stat">
          <div class="sforum-taxonomy__stat-key">{{ t('taxonomy.tags.stats.weekNew') }}</div>
          <div class="sforum-taxonomy__stat-value">{{ formatCount(weekNewCount) }}</div>
        </div>
        <div class="sforum-taxonomy__stat">
          <div class="sforum-taxonomy__stat-key">{{ t('taxonomy.tags.stats.hotThreshold') }}</div>
          <div class="sforum-taxonomy__stat-value">{{ formatCount(hotThreshold) }}</div>
        </div>
      </div>

      <section v-if="risingTags.length" class="sforum-taxonomy__section">
        <!-- 无周环比 API：展示热门标签作为「上升」区降级 -->
        <h2>{{ t('taxonomy.tags.risingTitle') }}</h2>
        <div class="sforum-taxonomy__hot-list">
          <NuxtLink
            v-for="tag in risingTags"
            :key="`rising-${tag.id}`"
            :to="localePath(forumTagPath(tag.slug))"
            class="sforum-taxonomy__hot"
          >
            <div>
              <div class="sforum-taxonomy__hot-name">{{ tag.name }}</div>
              <div class="sforum-taxonomy__hot-meta">{{ t('taxonomy.tags.risingMeta') }}</div>
            </div>
            <div class="sforum-taxonomy__hot-count">{{ formatCount(tag.topicCount || 0) }}</div>
          </NuxtLink>
        </div>
      </section>
    </div>
  </main>
</template>
