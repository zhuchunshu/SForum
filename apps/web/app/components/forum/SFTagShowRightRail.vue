<script setup lang="ts">
/**
 * 标签详情右栏：当前标签事实 + 相关热门标签。
 * 复用首页 sf-home-right-rail 卡片语言；桌面与抽屉共用。
 */
import { forumTagPath, forumTagsIndexPath, type ForumTag } from '~/utils/forum/forumTaxonomy'

const props = withDefaults(defineProps<{
  tag: ForumTag
  /** 列表 total（真实分页总数，可能与 tag.topicCount 略有差异时优先展示） */
  topicTotal: number
  relatedTags: ForumTag[]
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
  return new Intl.NumberFormat(locale.value).format(Math.max(0, value))
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

const description = computed(() => {
  const text = props.tag.description?.trim() || ''
  return text || t('taxonomy.tags.noDescription')
})

const topicCountDisplay = computed(() =>
  formatCount(props.topicTotal || props.tag.topicCount || 0)
)
const createdDisplay = computed(() => formatDate(props.tag.createdAt))
</script>

<template>
  <div class="sf-home-right-rail sforum-tags-page__rail">
    <section class="sf-home-right-rail__card">
      <header class="sf-home-right-rail__head">
        <h3 class="sf-home-right-rail__title">{{ t('taxonomy.tags.show.currentTitle') }}</h3>
        <span class="sf-home-right-rail__meta">#{{ tag.slug }}</span>
      </header>
      <div class="sforum-tags-page__current">
        <strong class="sforum-tags-page__current-name">#{{ tag.name }}</strong>
        <p class="sforum-tags-page__current-desc">{{ description }}</p>
        <dl class="sforum-tags-page__current-facts">
          <div>
            <dt>{{ t('taxonomy.tags.show.topicCount') }}</dt>
            <dd>{{ topicCountDisplay }}</dd>
          </div>
          <div>
            <dt>{{ t('taxonomy.tags.show.createdAt') }}</dt>
            <dd>{{ createdDisplay }}</dd>
          </div>
        </dl>
      </div>
    </section>

    <section v-if="relatedTags.length" class="sf-home-right-rail__card">
      <header class="sf-home-right-rail__head">
        <h3 class="sf-home-right-rail__title">{{ t('taxonomy.tags.hotRailTitle') }}</h3>
        <span class="sf-home-right-rail__meta">{{ t('taxonomy.tags.hotRailMeta') }}</span>
      </header>
      <ol class="sf-home-right-rail__hot-list">
        <li
          v-for="(item, index) in relatedTags"
          :key="`related-${item.id}`"
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
            :to="tagPath(item.slug)"
            class="sf-home-right-rail__hot-link"
            @click="onNavigate"
          >
            <span class="sf-home-right-rail__hot-title">#{{ item.name }}</span>
          </NuxtLink>
          <span class="sf-home-right-rail__hot-count">
            {{ formatCount(item.topicCount || 0) }}
          </span>
        </li>
      </ol>
    </section>

    <section class="sf-home-right-rail__card">
      <div class="sforum-tags-page__rail-actions">
        <NuxtLink
          :to="localePath(forumTagsIndexPath())"
          class="sf-home-right-rail__action sf-home-right-rail__action--outline sf-home-right-rail__action--block"
          @click="onNavigate"
        >
          <UIcon name="i-lucide-tags" class="size-4" aria-hidden="true" />
          {{ t('taxonomy.tags.show.viewAll') }}
        </NuxtLink>
      </div>
      <p class="sforum-tags-page__rail-note">
        <UIcon name="i-lucide-circle-help" class="size-4" aria-hidden="true" />
        <span>{{ t('taxonomy.tags.railNote') }}</span>
      </p>
    </section>
  </div>
</template>
