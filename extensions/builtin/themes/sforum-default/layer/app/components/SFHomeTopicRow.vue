<script setup lang="ts">
import type { ForumTopicSummary } from '~/utils/forumTaxonomy'

const props = defineProps<{
  topic: ForumTopicSummary
  to: string
  activityLabel: string
}>()

const { t } = useI18n()
</script>

<template>
  <NuxtLink :to="to" class="sf-home-topic-row">
    <div class="sf-home-topic-row__heat" :aria-label="t('home.feed.replyCount', { count: topic.commentCount })">
      {{ topic.commentCount }}
    </div>

    <div class="sf-home-topic-row__body">
      <h2 class="sf-home-topic-row__title">{{ topic.title }}</h2>
      <p class="sf-home-topic-row__excerpt">{{ topic.excerpt }}</p>
    </div>

    <div class="sf-home-topic-row__metric">
      <time :datetime="topic.lastActivityAt || topic.createdAt">{{ activityLabel }}</time>
      <span>{{ t('home.feed.latestActivity') }}</span>
    </div>
  </NuxtLink>
</template>
