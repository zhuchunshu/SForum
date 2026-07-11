<script setup lang="ts">
import type { ForumTopicDetail } from '~/utils/forumTaxonomy'

type TopicHeadingTag = {
  id: number
  name: string
  to: string
}

defineProps<{
  topic: ForumTopicDetail
  authorName: string
  authorTo?: string
  categoryTo: string
  tags: TopicHeadingTag[]
  publishedLabel: string
}>()

const { t } = useI18n()
</script>

<template>
  <header class="sf-topic-heading">
    <h1 class="sf-topic-heading__title">{{ topic.title }}</h1>

    <div class="sf-topic-heading__byline">
      <NuxtLink v-if="authorTo" :to="authorTo" class="sf-topic-heading__author">
        <SFAvatar :name="authorName" :avatar="topic.author?.avatar" size="sm" />
        <span>{{ authorName }}</span>
      </NuxtLink>
      <span v-else class="sf-topic-heading__author">
        <SFAvatar :name="authorName" :avatar="topic.author?.avatar" size="sm" />
        <span>{{ authorName }}</span>
      </span>

      <time :datetime="topic.createdAt">{{ publishedLabel }}</time>
      <span class="sf-topic-heading__metric">
        {{ topic.commentCount }} {{ t('topicDetail.statsComments') }}
      </span>
      <span class="sf-topic-heading__metric">
        {{ topic.viewCount }} {{ t('topicDetail.statsViews') }}
      </span>
      <span v-if="topic.isPinned" class="sf-topic-heading__state">
        <UIcon name="i-lucide-pin" class="size-3.5" aria-hidden="true" />
        {{ t('topicDetail.badge.pinned') }}
      </span>
      <span v-if="topic.status === 'locked'" class="sf-topic-heading__state">
        <UIcon name="i-lucide-lock" class="size-3.5" aria-hidden="true" />
        {{ t('topicDetail.badge.locked') }}
      </span>
      <NuxtLink
        v-for="tag in tags"
        :key="tag.id"
        :to="tag.to"
        class="sf-topic-heading__tag"
      >
        {{ tag.name }}
      </NuxtLink>
    </div>
  </header>
</template>
