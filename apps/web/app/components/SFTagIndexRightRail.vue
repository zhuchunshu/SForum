<script setup lang="ts">
/**
 * 标签索引右栏：复用首页 sf-home-right-rail 卡片语言，内容为标签目录语义。
 * 桌面与移动抽屉共用，避免双份 markup。
 */
import type { TagIndexOverview } from '~/utils/forumTagsIndex'
import { forumTagPath, type ForumTag } from '~/utils/forumTaxonomy'

const props = withDefaults(defineProps<{
  overview: TagIndexOverview
  hotTags: ForumTag[]
  recentTags: ForumTag[]
  /** 抽屉内点击跳转后是否关闭抽屉 */
  closeOnNavigate?: boolean
}>(), {
  closeOnNavigate: false
})

const emit = defineEmits<{
  navigate: []
}>()

const { t, locale } = useI18n()
const localePath = useLocalePath()
const { format: formatSiteDateTime } = useSiteDateTime()

function tagPath(slug: string) {
  return localePath(forumTagPath(slug))
}

function formatCount(value: number) {
  return new Intl.NumberFormat(locale.value).format(value)
}

function formatDate(value: string) {
  return value ? formatSiteDateTime(value) : t('taxonomy.tags.noCreatedAt')
}

function hotRank(index: number) {
  return String(index + 1).padStart(2, '0')
}

function isTopHotRank(index: number) {
  return index < 3
}

function onNavigate() {
  if (props.closeOnNavigate) {
    emit('navigate')
  }
}

const stats = computed(() => [
  { key: 'total', label: t('taxonomy.tags.stats.total'), value: props.overview.totalTags },
  { key: 'refs', label: t('taxonomy.tags.stats.topicRefs'), value: props.overview.totalTopicReferences },
  { key: 'week', label: t('taxonomy.tags.stats.weekNew'), value: props.overview.weekNewTags },
  { key: 'hot', label: t('taxonomy.tags.stats.hotThreshold'), value: props.overview.hotThreshold }
])
</script>

<template>
  <div class="sf-home-right-rail sforum-tags-page__rail">
    <section class="sf-home-right-rail__card">
      <header class="sf-home-right-rail__head">
        <h3 class="sf-home-right-rail__title">{{ t('taxonomy.tags.overviewTitle') }}</h3>
        <span class="sf-home-right-rail__meta">{{ t('taxonomy.tags.overviewSource') }}</span>
      </header>
      <div class="sf-home-right-rail__stats">
        <div v-for="stat in stats" :key="stat.key" class="sf-home-right-rail__stat">
          <strong>{{ formatCount(stat.value) }}</strong>
          <span>{{ stat.label }}</span>
        </div>
      </div>
    </section>

    <section v-if="hotTags.length" class="sf-home-right-rail__card">
      <header class="sf-home-right-rail__head">
        <h3 class="sf-home-right-rail__title">{{ t('taxonomy.tags.hotRailTitle') }}</h3>
        <span class="sf-home-right-rail__meta">{{ t('taxonomy.tags.hotRailMeta') }}</span>
      </header>
      <ol class="sf-home-right-rail__hot-list">
        <li
          v-for="(tag, index) in hotTags"
          :key="`rail-hot-${tag.id}`"
          class="sf-home-right-rail__hot-item"
        >
          <span
            class="sf-home-right-rail__rank"
            :class="{ 'is-top': isTopHotRank(index) }"
            aria-hidden="true"
          >
            {{ hotRank(index) }}
          </span>
          <NuxtLink
            :to="tagPath(tag.slug)"
            class="sf-home-right-rail__hot-link"
            @click="onNavigate"
          >
            <span class="sf-home-right-rail__hot-title">#{{ tag.name }}</span>
          </NuxtLink>
          <span
            class="sf-home-right-rail__hot-count"
            :title="t('taxonomy.tags.stats.topicRefs')"
          >
            {{ formatCount(tag.topicCount || 0) }}
          </span>
        </li>
      </ol>
    </section>

    <section v-if="recentTags.length" class="sf-home-right-rail__card">
      <header class="sf-home-right-rail__head">
        <h3 class="sf-home-right-rail__title">{{ t('taxonomy.tags.recentRailTitle') }}</h3>
        <span class="sf-home-right-rail__meta">{{ t('taxonomy.tags.recentRailMeta') }}</span>
      </header>
      <div class="sforum-tags-page__recent-list">
        <NuxtLink
          v-for="tag in recentTags"
          :key="`rail-recent-${tag.id}`"
          :to="tagPath(tag.slug)"
          class="sforum-tags-page__recent-item"
          @click="onNavigate"
        >
          <span class="sforum-tags-page__recent-dot" aria-hidden="true" />
          <span class="sforum-tags-page__recent-copy">
            <strong>#{{ tag.name }}</strong>
            <small>{{ formatDate(tag.createdAt) }}</small>
          </span>
        </NuxtLink>
      </div>
    </section>

    <section class="sf-home-right-rail__card">
      <p class="sforum-tags-page__rail-note">
        <UIcon name="i-lucide-circle-help" class="size-4" aria-hidden="true" />
        <span>{{ t('taxonomy.tags.railNote') }}</span>
      </p>
    </section>
  </div>
</template>
