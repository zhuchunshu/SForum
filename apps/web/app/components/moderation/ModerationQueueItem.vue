<script setup lang="ts">
import type { ModerationPendingItem, ModerationReportItem, ModerationSource } from '~/composables/useModerationApi'

defineProps<{ item: ModerationPendingItem | ModerationReportItem; source: ModerationSource; active?: boolean }>()
defineEmits<{ open: [] }>()
const { t } = useI18n()
const { format: formatDate } = useSiteDateTime()
const isReport = (item: ModerationPendingItem | ModerationReportItem): item is ModerationReportItem => 'reasonCode' in item
</script>

<template>
  <button
    type="button"
    class="sforum-moderation-row"
    :class="{ 'is-active': active, 'is-report': isReport(item) }"
    :aria-current="active ? 'true' : undefined"
    @click="$emit('open')"
  >
    <span class="sforum-moderation-row__icon" aria-hidden="true">
      <UIcon :name="isReport(item) ? 'i-lucide-flag' : item.targetType === 'topic' ? 'i-lucide-file-text' : 'i-lucide-message-square'" class="size-4" />
    </span>
    <span class="sforum-moderation-row__body">
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
          <span v-if="item.ipAddress" class="ml-1 font-mono text-slate-600 dark:text-zinc-300" :title="t('moderation.workbench.createIp')">· {{ item.ipAddress }}</span>
        </p>
      </div>
    </span>
    <span class="sforum-moderation-row__open">
      <span>{{ t('moderation.workbench.openReview') }}</span>
      <UIcon name="i-lucide-panel-right-open" class="size-4" aria-hidden="true" />
    </span>
  </button>
</template>
