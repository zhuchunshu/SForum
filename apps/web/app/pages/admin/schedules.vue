<script setup lang="ts">
import { formatScheduleDateTime, formatScheduleInterval } from '~/utils/adminJobs'

definePageMeta({ middleware: 'admin', layout: 'admin' })
defineOptions({ name: 'AdminSchedules' })

const { t, locale } = useI18n()
const adminPage = useAdminPage('/schedules')
const { can } = useAuthSession()
const manager = useAdminJobs()
const canManage = computed(() => can('jobs.manage'))

function isBusy(id: string, action: string) {
  return manager.busy.value === `schedule:${id}:${action}`
}
</script>

<template>
  <div class="mb-5 flex flex-wrap items-start justify-between gap-3">
    <div>
      <h2 class="flex items-center gap-2 text-xl font-bold">
        <UIcon :name="adminPage.icon" class="size-5 text-[var(--sf-accent)]" />
        {{ t('admin.schedules.title') }}
      </h2>
      <p class="mt-1 text-sm text-slate-500 dark:text-zinc-400">
        {{ t('admin.schedules.intro') }}
      </p>
    </div>
    <UButton
      icon="i-lucide-rotate-cw"
      color="neutral"
      variant="subtle"
      :loading="manager.schedules.pending.value"
      @click="manager.refreshSchedules"
    >
      {{ t('admin.extensions.refresh') }}
    </UButton>
  </div>

  <UAlert
    class="mb-5"
    color="neutral"
    variant="subtle"
    icon="i-lucide-info"
    :title="t('admin.schedules.hintTitle')"
    :description="t('admin.schedules.hintBody')"
  />

  <section class="border border-slate-200 bg-white dark:border-zinc-800 dark:bg-zinc-900">
    <div class="border-b border-slate-200 px-4 py-3 dark:border-zinc-800">
      <p class="text-sm font-semibold">
        {{ t('admin.schedules.listTitle') }}
      </p>
      <p class="mt-1 text-xs text-slate-500 dark:text-zinc-400">
        {{ t('admin.schedules.listIntro') }}
      </p>
    </div>

    <div v-if="!manager.schedules.data.value.length" class="px-4 py-8 text-sm text-slate-500">
      {{ t('admin.schedules.empty') }}
    </div>

    <div v-else class="overflow-x-auto">
      <table class="min-w-full text-left text-sm">
        <thead class="bg-slate-50 text-xs text-slate-500 dark:bg-zinc-950">
          <tr>
            <th class="px-3 py-3">{{ t('admin.schedules.colId') }}</th>
            <th class="px-3 py-3">{{ t('admin.schedules.colKind') }}</th>
            <th class="px-3 py-3">{{ t('admin.schedules.colInterval') }}</th>
            <th class="px-3 py-3">{{ t('admin.schedules.colLastRun') }}</th>
            <th class="px-3 py-3">{{ t('admin.schedules.colNextRun') }}</th>
            <th class="px-3 py-3">{{ t('admin.schedules.colStatus') }}</th>
            <th class="px-3 py-3 text-right">{{ t('admin.schedules.colActions') }}</th>
          </tr>
        </thead>
        <tbody class="divide-y divide-slate-200 dark:divide-zinc-800">
          <tr v-for="schedule in manager.schedules.data.value" :key="schedule.id">
            <td class="px-3 py-3 align-top">
              <p class="font-mono text-sm font-medium">{{ schedule.id }}</p>
              <p class="mt-1 max-w-xs text-xs text-slate-500">{{ schedule.description }}</p>
              <p class="mt-1 font-mono text-[11px] text-slate-400">{{ schedule.owner }}</p>
            </td>
            <td class="px-3 py-3 align-top">
              <p class="font-mono text-xs">{{ schedule.jobKind }}</p>
              <p class="mt-1 text-xs text-slate-500">{{ schedule.queue }}</p>
            </td>
            <td class="px-3 py-3 align-top tabular-nums">
              {{ formatScheduleInterval(schedule.intervalSeconds) }}
            </td>
            <td class="px-3 py-3 align-top text-xs tabular-nums">
              {{ formatScheduleDateTime(schedule.lastRunAt, locale) }}
            </td>
            <td class="px-3 py-3 align-top text-xs tabular-nums">
              {{ formatScheduleDateTime(schedule.nextRunAt, locale) }}
            </td>
            <td class="px-3 py-3 align-top">
              <UBadge :color="schedule.enabled ? 'success' : 'neutral'" variant="subtle">
                {{ schedule.enabled ? t('admin.schedules.statusOn') : t('admin.schedules.statusOff') }}
              </UBadge>
            </td>
            <td class="px-3 py-3 align-top">
              <div class="flex flex-wrap justify-end gap-1">
                <UButton
                  v-if="canManage"
                  size="xs"
                  color="neutral"
                  variant="subtle"
                  icon="i-lucide-zap"
                  :disabled="!schedule.enabled"
                  :loading="isBusy(schedule.id, 'trigger')"
                  @click="() => { void manager.triggerSchedule(schedule.id) }"
                >
                  {{ t('admin.schedules.trigger') }}
                </UButton>
                <UButton
                  v-if="canManage && schedule.enabled"
                  size="xs"
                  color="neutral"
                  variant="ghost"
                  icon="i-lucide-pause"
                  :loading="isBusy(schedule.id, 'disable')"
                  @click="manager.setScheduleEnabled(schedule.id, false)"
                >
                  {{ t('admin.schedules.disable') }}
                </UButton>
                <UButton
                  v-if="canManage && !schedule.enabled"
                  size="xs"
                  color="neutral"
                  variant="ghost"
                  icon="i-lucide-play"
                  :loading="isBusy(schedule.id, 'enable')"
                  @click="manager.setScheduleEnabled(schedule.id, true)"
                >
                  {{ t('admin.schedules.enable') }}
                </UButton>
              </div>
            </td>
          </tr>
        </tbody>
      </table>
    </div>
  </section>
</template>
