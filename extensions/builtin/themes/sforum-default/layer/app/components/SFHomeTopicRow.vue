<script setup lang="ts">
import type { ForumTopicSummary } from '~/utils/forumTaxonomy'
import { forumAuthorName } from '~/utils/forumTaxonomy'

const props = defineProps<{
  topic: ForumTopicSummary
  to: string
  activityLabel: string
}>()

const { t } = useI18n()
const localePath = useLocalePath()

const authorName = computed(() => forumAuthorName(props.topic.author, props.topic.authorUserId))
</script>

<template>
  <article class="sf-home-topic-row">
    <div class="sf-home-topic-row__topic">
      <div class="sf-home-topic-row__title-line">
        <h2 class="sf-home-topic-row__title">
          <NuxtLink :to="to">{{ topic.title }}</NuxtLink>
        </h2>
        <span v-if="topic.isPinned" class="sf-home-topic-row__pill sf-home-topic-row__pill--pin">
          {{ t('home.badge.pinned') }}
        </span>
        <span v-if="topic.status === 'locked'" class="sf-home-topic-row__pill sf-home-topic-row__pill--locked">
          {{ t('home.badge.locked') }}
        </span>
      </div>
      <div class="sf-home-topic-row__meta">
        <NuxtLink
          :to="localePath(`/c/${topic.categorySlug}`)"
          class="sf-home-topic-row__badge"
        >
          {{ topic.categoryName }}
        </NuxtLink>
        <span>
          <template v-if="topic.author">
            <NuxtLink :to="localePath(`/u/${topic.author.username}`)">{{ authorName }}</NuxtLink>
            ·
          </template>
          {{ t('home.feed.replyCount', { count: topic.commentCount }) }}
        </span>
      </div>
    </div>

    <div class="sf-home-topic-row__author-cell">
      <NuxtLink
        v-if="topic.author?.username"
        :to="localePath(`/u/${topic.author.username}`)"
        :aria-label="authorName"
      >
        <SFAvatar :name="authorName" :avatar="topic.author?.avatar" size="sm" />
      </NuxtLink>
      <SFAvatar v-else :name="authorName" :avatar="topic.author?.avatar" size="sm" />
    </div>

    <div class="sf-home-topic-row__stat">
      {{ topic.commentCount }}
      <small>{{ t('home.feed.repliesColumn') }}</small>
    </div>

    <time
      class="sf-home-topic-row__activity"
      :datetime="topic.lastActivityAt || topic.createdAt"
    >
      {{ activityLabel }}
    </time>
  </article>
</template>
