<script setup lang="ts">
import type { ForumTopicSummary } from '~/utils/forumTaxonomy'

const props = defineProps<{
  topic: ForumTopicSummary
  to: string
  activityLabel: string
}>()

const { t } = useI18n()
const localePath = useLocalePath()
</script>

<template>
  <article class="sf-home-topic-row">
    <div class="sf-home-topic-row__heat">
      <strong>{{ topic.commentCount }}</strong>
      <span>{{ t('home.feed.repliesColumn') }}</span>
      <small :aria-label="t('home.feed.replyCount', { count: topic.commentCount })">
        <UIcon name="i-lucide-message-circle" aria-hidden="true" />
        {{ topic.commentCount }}
      </small>
    </div>

    <div class="sf-home-topic-row__body">
      <div class="sf-home-topic-row__meta">
        <NuxtLink :to="localePath(`/c/${topic.categorySlug}`)" class="sf-home-topic-row__category">
          <span aria-hidden="true" />
          {{ topic.categoryName }}
        </NuxtLink>
        <template v-if="topic.author">
          <span aria-hidden="true">·</span>
          <NuxtLink :to="localePath(`/u/${topic.author.username}`)">@{{ topic.author.username }}</NuxtLink>
        </template>
        <span aria-hidden="true">·</span>
        <time :datetime="topic.lastActivityAt || topic.createdAt">{{ activityLabel }}</time>
        <span v-if="topic.isPinned" class="sf-home-topic-row__pinned">
          <UIcon name="i-lucide-pin" aria-hidden="true" />
          {{ t('home.badge.pinned') }}
        </span>
      </div>

      <h2 class="sf-home-topic-row__title">
        <NuxtLink :to="to">{{ topic.title }}</NuxtLink>
      </h2>
      <p class="sf-home-topic-row__excerpt">{{ topic.excerpt }}</p>

      <div class="sf-home-topic-row__footer">
        <div v-if="topic.tags?.length" class="sf-home-topic-row__tags">
          <NuxtLink
            v-for="tag in topic.tags"
            :key="tag.slug"
            :to="localePath(`/tags/${tag.slug}`)"
          >
            {{ tag.name }}
          </NuxtLink>
        </div>
        <span class="sf-home-topic-row__views">
          <UIcon name="i-lucide-eye" aria-hidden="true" />
          {{ topic.viewCount }}
        </span>
      </div>
    </div>
  </article>
</template>
