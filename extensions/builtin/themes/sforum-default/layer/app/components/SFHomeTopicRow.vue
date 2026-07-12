<script setup lang="ts">
import type { ForumTopicExtensionBadge, ForumTopicSummary } from '~/utils/forumTaxonomy'
import { forumAuthorName, forumTopicExtensionLabel } from '~/utils/forumTaxonomy'

const props = defineProps<{
  topic: ForumTopicSummary
  to: string
  activityLabel: string
  /** forum.topic.list.badges；列表级一次解析后挂到每行；空时不渲染 */
  extensionListBadges?: ForumTopicExtensionBadge[]
}>()

const { t, locale } = useI18n()
const localePath = useLocalePath()

const authorName = computed(() => forumAuthorName(props.topic.author, props.topic.authorUserId))
const listBadges = computed(() => props.extensionListBadges || [])

function listBadgeLabel(badge: ForumTopicExtensionBadge) {
  return forumTopicExtensionLabel(badge, String(locale.value || 'zh-CN')) || badge.id
}

function listBadgeHref(badge: ForumTopicExtensionBadge) {
  const href = `${badge.href || ''}`.trim()
  if (!href.startsWith('/') || href.startsWith('//') || href.includes('://') || href.startsWith('/api')) {
    return ''
  }
  return localePath(href)
}
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
        <template v-for="badge in listBadges" :key="`${badge.extensionId}:${badge.id}`">
          <NuxtLink
            v-if="listBadgeHref(badge)"
            :to="listBadgeHref(badge)"
            class="sf-home-topic-row__pill sf-home-topic-row__pill--ext"
            :class="`sf-home-topic-row__pill--ext-${badge.tone || 'neutral'}`"
            data-testid="topic-list-extension-badge"
          >
            {{ listBadgeLabel(badge) }}
          </NuxtLink>
          <span
            v-else
            class="sf-home-topic-row__pill sf-home-topic-row__pill--ext"
            :class="`sf-home-topic-row__pill--ext-${badge.tone || 'neutral'}`"
            data-testid="topic-list-extension-badge"
          >
            {{ listBadgeLabel(badge) }}
          </span>
        </template>
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
