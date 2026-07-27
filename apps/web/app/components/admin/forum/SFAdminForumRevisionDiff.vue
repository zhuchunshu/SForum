<script setup lang="ts">
import { diffLines } from 'diff'
import type { ForumRevisionDetail } from '~/utils/admin/adminForumContent'

const props = defineProps<{
  currentRevision: ForumRevisionDetail
  revision: ForumRevisionDetail
}>()

const { t } = useI18n()

const lines = computed(() => {
  const result = diffLines(props.revision.rawContent, props.currentRevision.rawContent, { newlineIsToken: true })
  return result.flatMap(change => change.value.split('\n').filter((line, index, parts) => line || index < parts.length - 1).map(value => ({
    value,
    kind: change.added ? 'added' : change.removed ? 'removed' : 'unchanged'
  })))
})

const metadataRows = computed(() => {
  const historical = props.revision.topicMetadata
  const current = props.currentRevision.topicMetadata
  const currentValues: Record<string, string> = {
    title: current?.title || '',
    category: current?.categorySlug || '',
    tags: current?.tagSlugs.join(', ') || '',
    attachments: props.currentRevision.attachments.ids.join(', ')
  }
  const historicalValues: Record<string, string> = {
    title: historical?.title || '',
    category: historical?.categorySlug || '',
    tags: historical?.tagSlugs.join(', ') || '',
    attachments: props.revision.attachments.ids.join(', ')
  }
  return ['title', 'category', 'tags', 'attachments']
    .filter(field => historical || current || field === 'attachments')
    .map(field => ({ field, before: historicalValues[field], after: currentValues[field], changed: historicalValues[field] !== currentValues[field] }))
})
</script>

<template>
  <section class="space-y-4" data-testid="forum-revision-diff">
    <div>
      <h4 class="text-sm font-semibold text-slate-900 dark:text-zinc-100">{{ t('admin.forum.content.history.diffTitle') }}</h4>
      <p class="mt-1 text-xs text-slate-500 dark:text-zinc-400">{{ t('admin.forum.content.history.diffDescription') }}</p>
    </div>

    <div class="overflow-x-auto rounded-lg border border-slate-200 dark:border-zinc-800">
      <div class="min-w-[36rem] font-mono text-xs leading-5">
        <div v-for="(line, index) in lines" :key="`${index}-${line.kind}-${line.value}`" class="grid grid-cols-[2.5rem_minmax(0,1fr)] border-b border-slate-100 last:border-0 dark:border-zinc-800" :class="{ 'bg-red-50 text-red-950 dark:bg-red-950/30 dark:text-red-100': line.kind === 'removed', 'bg-emerald-50 text-emerald-950 dark:bg-emerald-950/30 dark:text-emerald-100': line.kind === 'added' }">
          <span class="select-none border-r border-inherit px-2 text-right text-slate-400">{{ line.kind === 'removed' ? '-' : line.kind === 'added' ? '+' : ' ' }}</span>
          <code class="whitespace-pre-wrap break-all px-3">{{ line.value }}</code>
        </div>
      </div>
    </div>

    <div v-if="metadataRows.length" class="overflow-x-auto rounded-lg border border-slate-200 dark:border-zinc-800">
      <table class="w-full min-w-[34rem] text-left text-xs">
        <thead class="border-b border-slate-200 bg-slate-50 text-slate-500 dark:border-zinc-800 dark:bg-zinc-950 dark:text-zinc-400">
          <tr><th class="px-3 py-2 font-medium">{{ t('admin.forum.content.history.field') }}</th><th class="px-3 py-2 font-medium">{{ t('admin.forum.content.history.historical') }}</th><th class="px-3 py-2 font-medium">{{ t('admin.forum.content.history.current') }}</th></tr>
        </thead>
        <tbody>
          <tr v-for="row in metadataRows" :key="row.field" :class="row.changed ? 'bg-amber-50/70 dark:bg-amber-950/20' : ''" class="border-b border-slate-100 last:border-0 dark:border-zinc-800">
            <th class="px-3 py-2 font-medium text-slate-700 dark:text-zinc-200">{{ t(`admin.forum.content.history.fields.${row.field}`) }}</th>
            <td class="max-w-56 break-all px-3 py-2 text-slate-600 dark:text-zinc-300">{{ row.before || t('admin.forum.content.history.none') }}</td>
            <td class="max-w-56 break-all px-3 py-2 text-slate-600 dark:text-zinc-300">{{ row.after || t('admin.forum.content.history.none') }}</td>
          </tr>
        </tbody>
      </table>
    </div>
  </section>
</template>
