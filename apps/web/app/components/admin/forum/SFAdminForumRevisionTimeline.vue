<script setup lang="ts">
import type { ForumRevisionSummary } from '~/utils/adminForumContent'

const props = defineProps<{
  revisions: ForumRevisionSummary[]
  selectedRevisionNo?: number
  loadingRevisionNo?: number
  loading: boolean
  hasMore: boolean
}>()

const emit = defineEmits<{
  select: [revision: ForumRevisionSummary]
  more: []
}>()

const { t } = useI18n()
const { format: formatSiteDateTime } = useSiteDateTime()

function actorName(revision: ForumRevisionSummary) {
  return revision.actor?.displayName || revision.actor?.username || t('admin.forum.content.history.unknownActor')
}
</script>

<template>
  <section data-testid="forum-revision-timeline" class="space-y-3">
    <div class="flex items-center justify-between gap-3"><h4 class="text-sm font-semibold text-slate-900 dark:text-zinc-100">{{ t('admin.forum.content.history.title') }}</h4><span class="text-xs text-slate-500 dark:text-zinc-400">{{ t('admin.forum.content.history.lazyHint') }}</span></div>
    <div v-if="loading && !revisions.length" class="space-y-2"><USkeleton v-for="index in 3" :key="index" class="h-16 w-full" /></div>
    <p v-else-if="!revisions.length" class="py-5 text-center text-sm text-slate-500 dark:text-zinc-400">{{ t('admin.forum.content.history.empty') }}</p>
    <ol v-else class="space-y-2">
      <li v-for="revision in revisions" :key="revision.id">
        <button type="button" class="w-full rounded-lg border p-3 text-left transition-colors" :class="selectedRevisionNo === revision.revisionNo ? 'border-[var(--sf-accent)] bg-[color-mix(in_srgb,var(--sf-accent)_8%,transparent)]' : 'border-slate-200 hover:bg-slate-50 dark:border-zinc-800 dark:hover:bg-zinc-900'" @click="emit('select', revision)">
          <div class="flex flex-wrap items-center gap-2"><span class="font-semibold text-slate-900 dark:text-zinc-100">{{ t('admin.forum.content.history.version', { revision: revision.revisionNo }) }}</span><UBadge v-if="revision.current" color="primary" variant="soft">{{ t('admin.forum.content.history.currentBadge') }}</UBadge><UBadge v-if="revision.redacted" color="error" variant="soft">{{ t('admin.forum.content.history.redactedBadge') }}</UBadge><UBadge v-else-if="!revision.snapshotComplete" color="warning" variant="soft">{{ t('admin.forum.content.history.legacyBadge') }}</UBadge></div>
          <p class="mt-1 text-xs text-slate-500 dark:text-zinc-400">{{ actorName(revision) }} · {{ formatSiteDateTime(revision.committedAt) }}</p>
          <p class="mt-1 break-words text-xs text-slate-600 dark:text-zinc-300">{{ revision.reason || t('admin.forum.content.history.noReason') }}</p>
          <div class="mt-2 flex flex-wrap gap-1"><UBadge v-for="field in revision.changedFields" :key="field" color="neutral" variant="subtle" size="xs">{{ t(`admin.forum.content.history.fields.${field}`) }}</UBadge></div>
        </button>
      </li>
    </ol>
    <UButton v-if="hasMore" color="neutral" variant="outline" block :loading="loading" @click="emit('more')">{{ t('admin.forum.content.history.loadMore') }}</UButton>
  </section>
</template>
