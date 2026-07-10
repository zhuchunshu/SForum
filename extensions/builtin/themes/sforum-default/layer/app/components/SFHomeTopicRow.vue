<script setup lang="ts">
import { forumAuthorName, type ForumTopicSummary } from '~/utils/forumTaxonomy'

const props = defineProps<{
  topic: ForumTopicSummary
  to: string
  activityLabel: string
}>()

const { t } = useI18n()
const authorName = computed(() => forumAuthorName(props.topic.author, props.topic.authorUserId))
</script>

<template>
  <NuxtLink :to="to" class="sf-home-topic-row">
    <div class="sf-home-topic-row__heat" :aria-label="t('home.feed.replyCount', { count: topic.commentCount })">
      {{ topic.commentCount }}
    </div>

    <div class="sf-home-topic-row__body">
      <h2 class="sf-home-topic-row__title">{{ topic.title }}</h2>
      <div class="sf-home-topic-row__context">
        <SFAvatar
          class="sf-home-topic-row__avatar"
          :name="authorName"
          :avatar="topic.author?.avatar"
          alt=""
          size="xs"
        />
        <span class="sf-home-topic-row__author">{{ authorName }}</span>
        <span class="sf-home-topic-row__category">{{ topic.categoryName }}</span>
        <span v-if="topic.isPinned" class="sf-home-topic-row__pinned">
          <UIcon name="i-lucide-pin" class="size-3.5" aria-hidden="true" />
          {{ t('home.badge.pinned') }}
        </span>
        <span v-for="tag in topic.tags || []" :key="tag.slug" class="sf-home-topic-row__tag">
          #{{ tag.name }}
        </span>
      </div>
    </div>

    <div class="sf-home-topic-row__metric">
      <time :datetime="topic.lastActivityAt || topic.createdAt">{{ activityLabel }}</time>
      <span>{{ t('home.feed.latestActivity') }}</span>
    </div>
  </NuxtLink>
</template>
