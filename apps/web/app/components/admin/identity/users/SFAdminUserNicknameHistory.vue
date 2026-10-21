<script setup lang="ts">
import type { AdminNicknameChange } from '~/utils/admin/adminUsers'

/**
 * 后台用户详情的改名历史。
 *
 * 昵称是展示标签，历史回答「谁在什么时候把它从什么改成了什么」；数据由
 * AdminUserDetail.nicknameHistory 提供（最近 20 条，历史表本身永久保留）。
 */
const props = defineProps<{ changes?: AdminNicknameChange[] }>()

const { t } = useI18n()
const { format } = useSiteDateTime()

const items = computed(() => props.changes || [])
</script>

<template>
  <section class="rounded-lg border border-slate-200 bg-slate-50 p-4 dark:border-zinc-800 dark:bg-zinc-900/60">
    <div class="mb-3">
      <h4 class="text-sm font-semibold text-slate-900 dark:text-zinc-100">
        {{ t('admin.users.nicknameHistorySection') }}
      </h4>
      <p class="text-xs text-slate-500 dark:text-zinc-400">
        {{ t('admin.users.nicknameHistoryHelp') }}
      </p>
    </div>
    <p v-if="!items.length" class="text-xs text-slate-500 dark:text-zinc-400">
      {{ t('admin.users.nicknameHistoryEmpty') }}
    </p>
    <ol v-else class="space-y-2">
      <li
        v-for="change in items"
        :key="change.id"
        class="rounded-md border border-slate-200 bg-white px-3 py-2 text-xs dark:border-zinc-800 dark:bg-zinc-950"
      >
        <div class="flex flex-wrap items-center gap-2">
          <span class="font-medium text-slate-700 dark:text-zinc-200">{{ change.oldName }}</span>
          <span class="text-slate-400">→</span>
          <span class="font-semibold text-slate-900 dark:text-zinc-100">{{ change.newName }}</span>
          <UBadge color="neutral" variant="soft" size="sm">
            {{ change.source === 'admin' ? t('admin.users.nicknameHistorySourceAdmin') : t('admin.users.nicknameHistorySourceSelf') }}
          </UBadge>
        </div>
        <p class="mt-1 text-slate-500 dark:text-zinc-400">
          {{ format(change.createdAt) }} · {{ t('admin.users.nicknameHistoryChangedBy', { id: change.changedBy ?? '—' }) }}
        </p>
      </li>
    </ol>
  </section>
</template>
