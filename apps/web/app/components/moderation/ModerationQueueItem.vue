<script setup lang="ts">
import type { ModerationPendingItem, ModerationReportItem, ModerationSource } from '~/composables/useModerationApi'

defineProps<{ item: ModerationPendingItem | ModerationReportItem; source: ModerationSource }>()
defineEmits<{ open: [] }>()
const { t, locale } = useI18n()
const formatDate = (value: string) => new Intl.DateTimeFormat(locale.value, { dateStyle: 'medium', timeStyle: 'short' }).format(new Date(value))
const isReport = (item: ModerationPendingItem | ModerationReportItem): item is ModerationReportItem => 'reasonCode' in item
</script>

<template>
  <article class="border-l-4 border-amber-400 bg-white px-4 py-4 shadow-sm dark:bg-zinc-900 sm:px-5">
    <div class="flex flex-col gap-3 sm:flex-row sm:items-start sm:justify-between">
      <div class="min-w-0">
        <div class="flex flex-wrap items-center gap-2">
          <SFBadge variant="neutral">{{ t(`admin.moderation.type.${item.targetType}`) }}</SFBadge>
          <h2 class="min-w-0 text-sm font-semibold text-slate-900 dark:text-zinc-100">{{ item.title }}</h2>
          <SFBadge v-if="isReport(item)" variant="danger">{{ t(`admin.moderation.reason.${item.reasonCode}`) }}</SFBadge>
          <SFBadge v-for="trigger in !isReport(item) ? item.triggers : []" :key="trigger" variant="warning">{{ t(`moderation.trigger.${trigger}`) }}</SFBadge>
        </div>
        <p class="mt-2 line-clamp-3 text-sm leading-6 text-slate-600 dark:text-zinc-300">{{ item.excerpt || t('moderation.workbench.noExcerpt') }}</p>
        <p v-if="isReport(item) && item.body" class="mt-2 border-l-2 border-slate-200 pl-3 text-xs text-slate-500 dark:border-zinc-700 dark:text-zinc-400">{{ item.body }}</p>
        <p class="mt-3 text-xs text-slate-500 dark:text-zinc-400">
          {{ isReport(item) ? item.targetAuthorName : item.authorName }} · {{ item.category }} · {{ formatDate(item.createdAt) }}
        </p>
      </div>
      <UButton class="shrink-0 self-start" color="neutral" variant="subtle" icon="i-lucide-panel-right-open" @click="$emit('open')">
        {{ t('moderation.workbench.openReview') }}
      </UButton>
    </div>
  </article>
</template>
