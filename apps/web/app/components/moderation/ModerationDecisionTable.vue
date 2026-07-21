<script setup lang="ts">
import type { ModerationDecision } from '~/composables/useModerationApi'

defineProps<{ items: ModerationDecision[]; loading?: boolean }>()
const { t } = useI18n()
const { format: formatDate } = useSiteDateTime()
</script>

<template>
  <!-- 放在 UCard body 内：不再包一层边框，避免双层描边 -->
  <div class="-mx-4 -mb-4 overflow-x-auto sm:-mx-6 sm:-mb-6">
    <table class="min-w-full text-left text-sm">
      <thead class="border-y border-slate-200 bg-slate-50 text-xs text-slate-500 dark:border-zinc-800 dark:bg-zinc-950 dark:text-zinc-400">
        <tr>
          <th class="px-4 py-3 font-medium sm:px-6">{{ t('admin.moderation.auditTarget') }}</th>
          <th class="px-4 py-3 font-medium sm:px-6">{{ t('admin.moderation.auditAction') }}</th>
          <th class="px-4 py-3 font-medium sm:px-6">{{ t('admin.moderation.auditReviewer') }}</th>
          <th class="px-4 py-3 font-medium sm:px-6">{{ t('admin.moderation.auditNote') }}</th>
          <th class="px-4 py-3 font-medium sm:px-6">{{ t('admin.moderation.auditTime') }}</th>
        </tr>
      </thead>
      <tbody class="divide-y divide-slate-100 dark:divide-zinc-800">
        <tr v-if="loading">
          <td colspan="5" class="px-4 py-8 text-center text-slate-500 sm:px-6 dark:text-zinc-400">
            {{ t('admin.common.loading') }}
          </td>
        </tr>
        <tr v-for="item in items" v-else :key="item.id" class="hover:bg-slate-50/80 dark:hover:bg-zinc-950/50">
          <td class="whitespace-nowrap px-4 py-3 text-slate-900 sm:px-6 dark:text-zinc-100">
            {{ t(`admin.moderation.type.${item.targetType}`) }} #{{ item.targetId }}
          </td>
          <td class="whitespace-nowrap px-4 py-3 sm:px-6">
            <UBadge color="neutral" variant="soft">
              {{ t(`moderation.action.${item.action}`) }}
            </UBadge>
          </td>
          <td class="whitespace-nowrap px-4 py-3 text-slate-700 sm:px-6 dark:text-zinc-300">
            {{ item.reviewerName || `#${item.reviewerUserId}` }}
          </td>
          <td class="max-w-sm px-4 py-3 text-slate-600 sm:px-6 dark:text-zinc-300">
            {{ item.reviewNote || '—' }}
          </td>
          <td class="whitespace-nowrap px-4 py-3 text-slate-500 sm:px-6 dark:text-zinc-400">
            {{ formatDate(item.createdAt) }}
          </td>
        </tr>
        <tr v-if="!loading && !items.length">
          <td colspan="5" class="px-4 py-8 text-center text-slate-500 sm:px-6 dark:text-zinc-400">
            {{ t('admin.moderation.auditEmpty') }}
          </td>
        </tr>
      </tbody>
    </table>
  </div>
</template>
