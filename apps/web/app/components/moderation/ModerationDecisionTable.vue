<script setup lang="ts">
import type { ModerationDecision } from '~/composables/useModerationApi'

defineProps<{ items: ModerationDecision[]; loading?: boolean }>()
const { t, locale } = useI18n()
const formatDate = (value: string) => new Intl.DateTimeFormat(locale.value, { dateStyle: 'medium', timeStyle: 'short' }).format(new Date(value))
</script>

<template>
  <div class="overflow-x-auto border border-slate-200 dark:border-zinc-800">
    <table class="min-w-full text-left text-sm">
      <thead class="bg-slate-50 text-xs text-slate-500 dark:bg-zinc-900 dark:text-zinc-400">
        <tr><th class="px-4 py-3">{{ t('admin.moderation.auditTarget') }}</th><th class="px-4 py-3">{{ t('admin.moderation.auditAction') }}</th><th class="px-4 py-3">{{ t('admin.moderation.auditReviewer') }}</th><th class="px-4 py-3">{{ t('admin.moderation.auditNote') }}</th><th class="px-4 py-3">{{ t('admin.moderation.auditTime') }}</th></tr>
      </thead>
      <tbody class="divide-y divide-slate-200 bg-white dark:divide-zinc-800 dark:bg-zinc-950">
        <tr v-if="loading"><td colspan="5" class="px-4 py-8 text-center text-slate-500">{{ t('admin.home.loading') }}</td></tr>
        <tr v-for="item in items" v-else :key="item.id">
          <td class="whitespace-nowrap px-4 py-3">{{ t(`admin.moderation.type.${item.targetType}`) }} #{{ item.targetId }}</td>
          <td class="whitespace-nowrap px-4 py-3"><SFBadge variant="neutral">{{ t(`moderation.action.${item.action}`) }}</SFBadge></td>
          <td class="whitespace-nowrap px-4 py-3">{{ item.reviewerName || `#${item.reviewerUserId}` }}</td>
          <td class="max-w-sm px-4 py-3 text-slate-600 dark:text-zinc-300">{{ item.reviewNote || '—' }}</td>
          <td class="whitespace-nowrap px-4 py-3 text-slate-500">{{ formatDate(item.createdAt) }}</td>
        </tr>
        <tr v-if="!loading && !items.length"><td colspan="5" class="px-4 py-8 text-center text-slate-500">{{ t('admin.moderation.auditEmpty') }}</td></tr>
      </tbody>
    </table>
  </div>
</template>
