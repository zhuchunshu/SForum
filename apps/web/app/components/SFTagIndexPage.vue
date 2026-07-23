<script setup lang="ts">
/**
 * 宿主 body 岛：forum.tag.index。主题 L1 挂载；路由页仅 SEO + fail-closed 回退。
 */

import {
  activeForumTags,
  compareTagsByHeat,
  filterTagIndexTags,
  recentTagIndexTags,
  tagDisplayDescription,
  tagHeatEntries,
  tagIndexOverview,
  type TagIndexFilter
} from '~/utils/forumTagsIndex'
import {
  forumTagPath,
  parseForumTagPublicPagesOption,
  type ForumCategoryGroup,
  type ForumTag
} from '~/utils/forumTaxonomy'

const TAG_HEAT_LIMIT = 6
const RAIL_HOT_LIMIT = 6
const RAIL_RECENT_LIMIT = 4

const { t, locale } = useI18n()
const localePath = useLocalePath()
const { format: formatSiteDateTime } = useSiteDateTime()
const { webOption } = useWebOptions()
const forumApi = useForumApi()
const { can } = usePermissions()

const publicTagPagesEnabled = computed(() => parseForumTagPublicPagesOption(
  webOption('forum.tags.public_pages', 'enabled')
))

if (!publicTagPagesEnabled.value) {
  throw createError({
    statusCode: 404,
    statusMessage: 'Tag pages are disabled'
  })
}

const filter = ref<TagIndexFilter>('all')
const searchQuery = ref('')
const mobileMenuOpen = useState<boolean>('forum-mobile-menu-open', () => false)
const mobileInfoOpen = useState<boolean>('forum-mobile-info-open', () => false)
const canCreateTopic = computed(() => can(FORUM_PERMISSIONS.topicCreate))
const renderedAt = useState<number>('forum-tags-index-rendered-at', () => Date.now())

const {
  data: categoryGroups,
  pending: categoriesPending
} = await useAsyncData(
  'forum-tags-index-category-groups',
  () => forumApi.listCategoryGroups(),
  { default: () => [] as ForumCategoryGroup[] }
)

const {
  data: tags,
  pending: tagsPending,
  error: tagsError,
  refresh: refreshTags
} = await useAsyncData(
  'forum-tags-index-tags',
  () => forumApi.listTags(),
  { default: () => [] as ForumTag[] }
)

const categories = computed(() => categoryGroups.value.flatMap((group) => group.categories || []))
const activeTags = computed(() => activeForumTags(tags.value || []))
const totalTopics = computed(() => categories.value.reduce((sum, category) => sum + category.topicCount, 0))
const overview = computed(() => tagIndexOverview(activeTags.value, renderedAt.value))
const filteredTags = computed(() => filterTagIndexTags(activeTags.value, {
  filter: filter.value,
  query: searchQuery.value,
  locale: locale.value,
  nowMs: renderedAt.value
}))
const heatEntries = computed(() => tagHeatEntries(filteredTags.value, TAG_HEAT_LIMIT, locale.value))
const railHotTags = computed(() =>
  activeTags.value
    .slice()
    .sort((left, right) => compareTagsByHeat(left, right, locale.value))
    .slice(0, RAIL_HOT_LIMIT)
)
const railRecentTags = computed(() =>
  recentTagIndexTags(activeTags.value, RAIL_RECENT_LIMIT, renderedAt.value, locale.value)
)
const hasQuery = computed(() => searchQuery.value.trim().length > 0)
const emptyTitle = computed(() => {
  if (hasQuery.value) return t('taxonomy.tags.emptySearchTitle')
  if (filter.value === 'hot') return t('taxonomy.tags.emptyHotTitle')
  if (filter.value === 'week') return t('taxonomy.tags.emptyWeekTitle')
  return t('taxonomy.tags.emptyTitle')
})
const emptyDescription = computed(() => {
  if (hasQuery.value) return t('taxonomy.tags.emptySearchDescription')
  if (filter.value === 'hot') return t('taxonomy.tags.emptyHotDescription')
  if (filter.value === 'week') return t('taxonomy.tags.emptyWeekDescription')
  return t('taxonomy.tags.emptyDescription')
})

function closeMobileDrawers() {
  mobileMenuOpen.value = false
  mobileInfoOpen.value = false
}

function setFilter(next: TagIndexFilter) {
  filter.value = next
}

function tagPath(slug: string) {
  return localePath(forumTagPath(slug))
}

function formatCount(value: number) {
  return new Intl.NumberFormat(locale.value).format(value)
}

function formatDate(value: string) {
  return value ? formatSiteDateTime(value) : t('taxonomy.tags.noCreatedAt')
}

function descriptionFor(tag: ForumTag) {
  return tagDisplayDescription(tag, t('taxonomy.tags.noDescription'))
}

function retryTags() {
  void refreshTags()
}
</script>

<template>
  <main class="sforum-tags-page">
    <div class="sforum-tags-page__layout sforum-tags-page__layout--with-side">
      <aside class="sforum-tags-page__sidebar" :aria-label="t('home.sidebar.navTitle')">
        <SFHomeNavigation
          desktop-only
          navigation-mode="route"
          :categories="categories"
          selected-category-slug=""
          :total-topics="totalTopics"
          :pending="categoriesPending"
          :can-create-topic="canCreateTopic"
        />
      </aside>

      <section class="sforum-tags-page__main" aria-labelledby="tag-index-title">
        <div class="sforum-tags-page__inner">
          <header class="sforum-tags-page__head">
            <div class="sforum-tags-page__head-copy">
              <h1 id="tag-index-title">{{ t('taxonomy.tags.title') }}</h1>
              <p>{{ t('taxonomy.tags.description') }}</p>
            </div>
            <div class="sforum-tags-page__head-count">
              <strong>{{ formatCount(overview.totalTags) }}</strong>
              <span>{{ t('taxonomy.tags.headerCount') }}</span>
            </div>
          </header>

          <div class="sforum-tags-page__toolbar">
            <label class="sforum-tags-page__search">
              <UIcon name="i-lucide-search" class="size-4" aria-hidden="true" />
              <input
                v-model="searchQuery"
                type="search"
                :placeholder="t('taxonomy.tags.filterPlaceholder')"
                :aria-label="t('taxonomy.tags.filterPlaceholder')"
              >
            </label>

            <div class="sforum-tags-page__filters" role="group" :aria-label="t('taxonomy.tags.filterLabel')">
              <button
                type="button"
                class="sforum-tags-page__filter"
                :class="{ 'is-active': filter === 'all' }"
                :aria-pressed="filter === 'all'"
                @click="setFilter('all')"
              >
                {{ t('taxonomy.tags.filters.all') }}
              </button>
              <button
                type="button"
                class="sforum-tags-page__filter"
                :class="{ 'is-active': filter === 'hot' }"
                :aria-pressed="filter === 'hot'"
                @click="setFilter('hot')"
              >
                {{ t('taxonomy.tags.filters.hot') }}
              </button>
              <button
                type="button"
                class="sforum-tags-page__filter"
                :class="{ 'is-active': filter === 'week' }"
                :aria-pressed="filter === 'week'"
                @click="setFilter('week')"
              >
                {{ t('taxonomy.tags.filters.week') }}
              </button>
              <button
                type="button"
                class="sforum-tags-page__filter"
                :class="{ 'is-active': filter === 'az' }"
                :aria-pressed="filter === 'az'"
                @click="setFilter('az')"
              >
                {{ t('taxonomy.tags.filters.az') }}
              </button>
            </div>
          </div>

          <div v-if="tagsError" class="sforum-tags-page__error" role="alert">
            <span>{{ t('taxonomy.tags.loadFailed') }}</span>
            <button type="button" class="sf-button sf-button--ghost sf-button--sm" @click="retryTags">
              <UIcon name="i-lucide-refresh-cw" class="size-4" aria-hidden="true" />
              {{ t('home.feed.retry') }}
            </button>
          </div>

          <template v-else>
            <section class="sforum-tags-page__section" :aria-busy="tagsPending">
              <div class="sforum-tags-page__section-caption">
                <h2>{{ t('taxonomy.tags.heatTitle') }}</h2>
                <span>{{ t('taxonomy.tags.heatSubtitle') }}</span>
              </div>

              <div v-if="tagsPending" class="sforum-tags-page__pending">
                <SFSkeleton v-for="item in 6" :key="item" :lines="1" />
              </div>

              <div v-else-if="heatEntries.length" class="sforum-tags-page__heat-board">
                <NuxtLink
                  v-for="entry in heatEntries"
                  :key="`heat-${entry.tag.id}`"
                  :to="tagPath(entry.tag.slug)"
                  class="sforum-tags-page__heat-row"
                >
                  <span class="sforum-tags-page__heat-label">
                    <span>#{{ entry.tag.name }}</span>
                  </span>
                  <span class="sforum-tags-page__heat-track" aria-hidden="true">
                    <i :style="{ width: `${entry.widthPercent}%` }" />
                  </span>
                  <span class="sforum-tags-page__heat-count">{{ formatCount(entry.tag.topicCount || 0) }}</span>
                </NuxtLink>
              </div>

              <div v-else class="sforum-tags-page__empty">
                <SFEmptyState
                  :title="emptyTitle"
                  :description="emptyDescription"
                />
              </div>
            </section>

            <section class="sforum-tags-page__section" :aria-busy="tagsPending">
              <div class="sforum-tags-page__section-caption">
                <h2>{{ t('taxonomy.tags.directoryTitle') }}</h2>
                <span>{{ t('taxonomy.tags.directorySummary', { count: formatCount(filteredTags.length) }) }}</span>
              </div>

              <div v-if="tagsPending" class="sforum-tags-page__pending sforum-tags-page__pending--grid">
                <SFSkeleton v-for="item in 8" :key="item" :lines="2" />
              </div>

              <div v-else-if="filteredTags.length" class="sforum-tags-page__directory">
                <NuxtLink
                  v-for="tag in filteredTags"
                  :key="tag.id"
                  :to="tagPath(tag.slug)"
                  class="sforum-tags-page__tag"
                >
                  <span class="sforum-tags-page__tag-copy">
                    <strong>#{{ tag.name }}</strong>
                    <span>{{ descriptionFor(tag) }}</span>
                    <small>{{ tag.slug }} · {{ formatDate(tag.createdAt) }}</small>
                  </span>
                  <b>{{ formatCount(tag.topicCount || 0) }}</b>
                </NuxtLink>
              </div>

              <div v-else class="sforum-tags-page__empty">
                <SFEmptyState
                  :title="emptyTitle"
                  :description="emptyDescription"
                />
              </div>
            </section>
          </template>
        </div>
      </section>

      <aside class="sforum-tags-page__side" :aria-label="t('taxonomy.tags.railLabel')">
        <section class="sforum-tags-rail__section">
          <div class="sforum-tags-rail__heading">
            <h3>{{ t('taxonomy.tags.overviewTitle') }}</h3>
            <span>{{ t('taxonomy.tags.overviewSource') }}</span>
          </div>
          <div class="sforum-tags-rail__overview">
            <div>
              <strong>{{ formatCount(overview.totalTags) }}</strong>
              <span>{{ t('taxonomy.tags.stats.total') }}</span>
            </div>
            <div>
              <strong>{{ formatCount(overview.totalTopicReferences) }}</strong>
              <span>{{ t('taxonomy.tags.stats.topicRefs') }}</span>
            </div>
            <div>
              <strong>{{ formatCount(overview.weekNewTags) }}</strong>
              <span>{{ t('taxonomy.tags.stats.weekNew') }}</span>
            </div>
            <div>
              <strong>{{ formatCount(overview.hotThreshold) }}</strong>
              <span>{{ t('taxonomy.tags.stats.hotThreshold') }}</span>
            </div>
          </div>
        </section>

        <section v-if="railHotTags.length" class="sforum-tags-rail__section">
          <div class="sforum-tags-rail__heading">
            <h3>{{ t('taxonomy.tags.hotRailTitle') }}</h3>
            <span>{{ t('taxonomy.tags.hotRailMeta') }}</span>
          </div>
          <ol class="sforum-tags-rail__hot-list">
            <li v-for="(tag, index) in railHotTags" :key="`rail-hot-${tag.id}`">
              <b>{{ String(index + 1).padStart(2, '0') }}</b>
              <NuxtLink :to="tagPath(tag.slug)">#{{ tag.name }}</NuxtLink>
              <span>{{ formatCount(tag.topicCount || 0) }}</span>
            </li>
          </ol>
        </section>

        <section v-if="railRecentTags.length" class="sforum-tags-rail__section">
          <div class="sforum-tags-rail__heading">
            <h3>{{ t('taxonomy.tags.recentRailTitle') }}</h3>
            <span>{{ t('taxonomy.tags.recentRailMeta') }}</span>
          </div>
          <div class="sforum-tags-rail__recent-list">
            <NuxtLink
              v-for="tag in railRecentTags"
              :key="`rail-recent-${tag.id}`"
              :to="tagPath(tag.slug)"
              class="sforum-tags-rail__recent"
            >
              <i aria-hidden="true" />
              <span>
                <strong>#{{ tag.name }}</strong>
                <small>{{ formatDate(tag.createdAt) }}</small>
              </span>
            </NuxtLink>
          </div>
        </section>

        <section class="sforum-tags-rail__section">
          <p class="sforum-tags-rail__note">
            <UIcon name="i-lucide-circle-help" class="size-4" aria-hidden="true" />
            <span>{{ t('taxonomy.tags.railNote') }}</span>
          </p>
        </section>
      </aside>
    </div>

    <button
      v-if="mobileMenuOpen || mobileInfoOpen"
      type="button"
      class="sforum-mobile-drawer__backdrop"
      :aria-label="t('topicDetail.cancel')"
      @click="closeMobileDrawers"
    />

    <aside v-if="mobileMenuOpen" class="sforum-mobile-drawer sforum-mobile-drawer--left">
      <header class="sforum-mobile-drawer__head">
        <strong>{{ t('home.sidebar.drawerTitle') }}</strong>
        <button type="button" :aria-label="t('topicDetail.cancel')" @click="closeMobileDrawers">
          <UIcon name="i-lucide-x" class="size-5" aria-hidden="true" />
        </button>
      </header>
      <SFHomeNavigation
        desktop-only
        navigation-mode="route"
        :categories="categories"
        selected-category-slug=""
        :total-topics="totalTopics"
        :pending="categoriesPending"
        :can-create-topic="canCreateTopic"
      />
    </aside>

    <aside v-if="mobileInfoOpen" class="sforum-mobile-drawer sforum-mobile-drawer--right">
      <header class="sforum-mobile-drawer__head">
        <strong>{{ t('taxonomy.tags.railLabel') }}</strong>
        <button type="button" :aria-label="t('topicDetail.cancel')" @click="closeMobileDrawers">
          <UIcon name="i-lucide-x" class="size-5" aria-hidden="true" />
        </button>
      </header>
      <div class="sforum-tags-page__mobile-side">
        <section class="sforum-tags-rail__section">
          <div class="sforum-tags-rail__heading">
            <h3>{{ t('taxonomy.tags.overviewTitle') }}</h3>
            <span>{{ t('taxonomy.tags.overviewSource') }}</span>
          </div>
          <div class="sforum-tags-rail__overview">
            <div>
              <strong>{{ formatCount(overview.totalTags) }}</strong>
              <span>{{ t('taxonomy.tags.stats.total') }}</span>
            </div>
            <div>
              <strong>{{ formatCount(overview.totalTopicReferences) }}</strong>
              <span>{{ t('taxonomy.tags.stats.topicRefs') }}</span>
            </div>
            <div>
              <strong>{{ formatCount(overview.weekNewTags) }}</strong>
              <span>{{ t('taxonomy.tags.stats.weekNew') }}</span>
            </div>
            <div>
              <strong>{{ formatCount(overview.hotThreshold) }}</strong>
              <span>{{ t('taxonomy.tags.stats.hotThreshold') }}</span>
            </div>
          </div>
        </section>
        <section v-if="railHotTags.length" class="sforum-tags-rail__section">
          <div class="sforum-tags-rail__heading">
            <h3>{{ t('taxonomy.tags.hotRailTitle') }}</h3>
            <span>{{ t('taxonomy.tags.hotRailMeta') }}</span>
          </div>
          <ol class="sforum-tags-rail__hot-list">
            <li v-for="(tag, index) in railHotTags" :key="`drawer-hot-${tag.id}`">
              <b>{{ String(index + 1).padStart(2, '0') }}</b>
              <NuxtLink :to="tagPath(tag.slug)" @click="closeMobileDrawers">#{{ tag.name }}</NuxtLink>
              <span>{{ formatCount(tag.topicCount || 0) }}</span>
            </li>
          </ol>
        </section>
        <section v-if="railRecentTags.length" class="sforum-tags-rail__section">
          <div class="sforum-tags-rail__heading">
            <h3>{{ t('taxonomy.tags.recentRailTitle') }}</h3>
            <span>{{ t('taxonomy.tags.recentRailMeta') }}</span>
          </div>
          <div class="sforum-tags-rail__recent-list">
            <NuxtLink
              v-for="tag in railRecentTags"
              :key="`drawer-recent-${tag.id}`"
              :to="tagPath(tag.slug)"
              class="sforum-tags-rail__recent"
              @click="closeMobileDrawers"
            >
              <i aria-hidden="true" />
              <span>
                <strong>#{{ tag.name }}</strong>
                <small>{{ formatDate(tag.createdAt) }}</small>
              </span>
            </NuxtLink>
          </div>
        </section>
      </div>
    </aside>
  </main>
</template>
