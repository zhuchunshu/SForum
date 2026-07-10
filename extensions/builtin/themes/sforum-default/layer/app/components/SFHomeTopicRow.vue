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
    <SFAvatar
      class="sf-home-topic-row__avatar"
      :name="authorName"
      :avatar="topic.author?.avatar"
      alt=""
      size="sm"
    />

    <div class="sf-home-topic-row__body">
      <h2 class="sf-home-topic-row__title">{{ topic.title }}</h2>
      <div class="sf-home-topic-row__taxonomy">
        <span class="sf-home-topic-row__category">{{ topic.categoryName }}</span>
        <span v-if="topic.isPinned" class="sf-home-topic-row__pinned">
          <UIcon name="i-lucide-pin" class="size-3.5" aria-hidden="true" />
          {{ t('home.badge.pinned') }}
        </span>
        <span v-for="tag in topic.tags || []" :key="tag.slug" class="sf-home-topic-row__tag">
          #{{ tag.name }}
        </span>
      </div>
      <p class="sf-home-topic-row__author">{{ authorName }}</p>
    </div>

    <div class="sf-home-topic-row__replies">
      <strong>{{ topic.commentCount }}</strong>
      <span>{{ t('home.feed.repliesColumn') }}</span>
    </div>
    <time class="sf-home-topic-row__activity" :datetime="topic.lastActivityAt || topic.createdAt">
      {{ activityLabel }}
    </time>
  </NuxtLink>
</template>
