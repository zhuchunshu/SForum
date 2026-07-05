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
    <div class="sf-feed-row__avatar-wrapper">
      <SFAvatar :name="author || '?'" size="sm" />
    </div>
    <div class="sf-feed-row__content">
      <div class="sf-feed-row__header">
        <h3 class="sf-feed-row__title">
          {{ title }}
        </h3>
        <div class="sf-feed-row__actions">
          <div class="sf-feed-row__vote">
            <button class="sf-feed-row__vote-btn" aria-label="赞同">
              <UIcon name="i-lucide-chevron-up" class="size-3.5" />
            </button>
            <span class="sf-feed-row__vote-val">{{ score }}</span>
            <button class="sf-feed-row__vote-btn" aria-label="反对">
              <UIcon name="i-lucide-chevron-down" class="size-3.5" />
            </button>
          </div>
          <div class="sf-feed-row__action-tag">
            <UIcon name="i-lucide-message-circle" class="size-3.5" />
            {{ replies }}
          </div>
        </div>
      </div>
      
      <div class="sf-feed-row__meta-row">
        <span v-if="excerpt" class="sf-feed-row__excerpt">{{ excerpt }}</span>
        <span v-if="badges.length" class="sf-feed-row__badges">
          <SFBadge
            v-for="badge in badges"
            :key="badge.label"
            :variant="badge.variant || 'neutral'"
          >
            {{ badge.label }}
          </SFBadge>
        </span>
        <span v-if="author" class="sf-feed-row__author">{{ author }}</span>
        <span v-if="meta" class="sf-feed-row__time">• {{ meta }}</span>
        <span v-if="views" class="sf-feed-row__views">
          <UIcon name="i-lucide-eye" class="size-3.5" />
          {{ views }} 浏览
        </span>
      </div>
    </div>
  </article>
</template>
