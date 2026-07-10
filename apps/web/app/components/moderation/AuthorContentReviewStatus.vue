<script setup lang="ts">
import type { ForumAuthorReviewItem } from '~/utils/forumTaxonomy'

defineProps<{ items: ForumAuthorReviewItem[] }>()
const { t, locale } = useI18n()
const formatDate = (value: string) => new Intl.DateTimeFormat(locale.value, { dateStyle: 'medium', timeStyle: 'short' }).format(new Date(value))
</script>

<template>
  <div class="divide-y divide-slate-200 border-y border-slate-200 dark:divide-zinc-800 dark:border-zinc-800">
    <article v-for="item in items" :key="`${item.targetType}-${item.targetId}`" class="py-5">
      <div class="flex flex-wrap items-center gap-2">
        <SFBadge :variant="item.status === 'pending' ? 'warning' : 'danger'">{{ t(`moderation.authorStatus.${item.status}`) }}</SFBadge>
        <SFBadge variant="neutral">{{ t(`admin.moderation.type.${item.targetType}`) }}</SFBadge>
        <h2 class="text-sm font-semibold text-slate-900 dark:text-zinc-100">{{ item.title }}</h2>
      </div>
      <p class="mt-2 text-sm leading-6 text-slate-600 dark:text-zinc-300">{{ item.excerpt }}</p>
      <p v-if="item.status === 'rejected' && item.reviewNote" class="mt-3 border-l-2 border-red-300 pl-3 text-sm text-red-700 dark:border-red-800 dark:text-red-300">{{ t('moderation.authorStatus.reason') }}：{{ item.reviewNote }}</p>
      <p class="mt-3 text-xs text-slate-500">{{ formatDate(item.createdAt) }}</p>
    </article>
    <SFEmptyState v-if="!items.length" :title="t('moderation.authorStatus.emptyTitle')" :description="t('moderation.authorStatus.emptyDescription')" />
  </div>
</template>
