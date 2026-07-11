<script setup lang="ts">
import type { ForumTopicDetail } from '~/utils/forumTaxonomy'

const props = defineProps<{
  topic: ForumTopicDetail
  authorName: string
  authorTo?: string
  tags: { id: number, name: string, to: string }[]
  categoryTo: string
}>()

const { t } = useI18n()

const statusLabel = computed(() => {
  if (props.topic.status === 'locked') {
    return t('topicDetail.badge.locked')
  }
  if (props.topic.isPinned) {
    return t('topicDetail.badge.pinned')
  }
  return t('topicDetail.side.statusActive')
})
</script>

<template>
  <aside class="sforum-topic-page__side" :aria-label="t('topicDetail.side.title')">
    <div class="sf-topic-side-card">
      <h3>{{ t('topicDetail.side.title') }}</h3>
      <div class="sf-topic-side-card__row">
        <span>{{ t('topicDetail.side.status') }}</span>
        <strong>{{ statusLabel }}</strong>
      </div>
      <div class="sf-topic-side-card__row">
        <span>{{ t('topicDetail.side.category') }}</span>
        <strong>
          <NuxtLink :to="categoryTo">{{ topic.categoryName }}</NuxtLink>
        </strong>
      </div>
      <div class="sf-topic-side-card__row">
        <span>{{ t('topicDetail.side.replies') }}</span>
        <strong>{{ topic.commentCount }}</strong>
      </div>
      <div class="sf-topic-side-card__row">
        <span>{{ t('topicDetail.side.views') }}</span>
        <strong>{{ topic.viewCount }}</strong>
      </div>
    </div>

    <div class="sf-topic-side-card">
      <h3>{{ t('topicDetail.side.participants') }}</h3>
      <div class="sf-topic-side-card__participants">
        <NuxtLink v-if="authorTo" :to="authorTo" :aria-label="authorName">
          <SFAvatar :name="authorName" :avatar="topic.author?.avatar" size="md" />
        </NuxtLink>
        <SFAvatar v-else :name="authorName" :avatar="topic.author?.avatar" size="md" />
      </div>
    </div>

    <div v-if="tags.length" class="sf-topic-side-card">
      <h3>{{ t('topicDetail.side.tags') }}</h3>
      <div class="sf-topic-side-card__tags">
        <NuxtLink
          v-for="tag in tags"
          :key="tag.id"
          :to="tag.to"
          class="sf-topic-side-card__tag"
        >
          {{ tag.name }}
        </NuxtLink>
      </div>
    </div>
  </aside>
</template>
