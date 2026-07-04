<script setup lang="ts">
type FeedBadge = {
  label: string
  variant?: 'neutral' | 'primary' | 'info' | 'success' | 'warning' | 'danger'
}

const props = withDefaults(defineProps<{
  title: string
  excerpt?: string
  author?: string
  meta?: string
  score?: number
  replies?: number
  views?: number
  badges?: FeedBadge[]
}>(), {
  excerpt: undefined,
  author: undefined,
  meta: undefined,
  score: 0,
  replies: 0,
  views: 0,
  badges: () => []
})
</script>

<template>
  <article class="sf-feed-row">
    <div class="sf-feed-row__score">
      <strong>{{ score }}</strong>
      <span>赞同</span>
    </div>
    <div>
      <h3 class="sf-feed-row__title">
        {{ title }}
      </h3>
      <p v-if="excerpt" class="sf-feed-row__excerpt">
        {{ excerpt }}
      </p>
      <div class="sf-feed-row__meta">
        <span v-if="author">{{ author }}</span>
        <span v-if="meta">{{ meta }}</span>
        <span v-if="badges.length" class="sf-feed-row__badges">
          <SFBadge
            v-for="badge in badges"
            :key="badge.label"
            :variant="badge.variant || 'neutral'"
          >
            {{ badge.label }}
          </SFBadge>
        </span>
      </div>
    </div>
    <div class="sf-feed-row__stats">
      <span>{{ replies }} 回复</span>
      <span>{{ views }} 浏览</span>
    </div>
  </article>
</template>
