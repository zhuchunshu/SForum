<script setup lang="ts">
import { ALL_ADMIN_JOBS_FILTER, jobCanCancel, jobCanRetry, jobStateColor } from '~/utils/adminJobs'

definePageMeta({ middleware: 'admin', layout: 'admin' })
defineOptions({ name: 'AdminJobs' })
const { t, locale } = useI18n()
const adminPage = useAdminPage('/jobs')
const { can } = useAuthSession()
const manager = useAdminJobs()
const { contributionsFor } = useAdminExtensionRegistry()
const columnContributions = computed(() => contributionsFor('admin.jobs.table.columns'))
const actionContributions = computed(() => contributionsFor('admin.jobs.row.actions'))
const detailContributions = computed(() => contributionsFor('admin.jobs.detail.sections'))
const canManage = computed(() => can('jobs.manage'))
const counts = computed(() => manager.overview.data.value.counts)
const stateOptions = computed(() => [ALL_ADMIN_JOBS_FILTER, 'available', 'running', 'retryable', 'scheduled', 'completed', 'discarded', 'cancelled'].map(value => ({ label: value === ALL_ADMIN_JOBS_FILTER ? t('admin.jobs.allStates') : value, value })))
const queueOptions = computed(() => [{ label: t('admin.jobs.allQueues'), value: ALL_ADMIN_JOBS_FILTER }, ...manager.overview.data.value.queues.map(item => ({ label: item.name, value: item.name }))])
function contributionLabel(item: { label: Record<string, string>, contributionId: string }) { return item.label[locale.value] || item.label['zh-CN'] || item.label['en-US'] || item.contributionId }
function closeDetail() { manager.selected.value = null }
</script>

<template>
  <div class="mb-5 flex flex-wrap items-start justify-between gap-3">
    <div><h2 class="flex items-center gap-2 text-xl font-bold"><UIcon :name="adminPage.icon" class="size-5 text-[var(--sf-accent)]" />{{ t('admin.jobs.title') }}</h2><p class="mt-1 text-sm text-slate-500 dark:text-zinc-400">{{ t('admin.jobs.intro') }}</p></div>
  </div>
  <UDashboardToolbar class="mb-5 rounded-lg border border-slate-200 bg-white px-4 py-2.5 dark:border-zinc-800 dark:bg-zinc-900"><template #left><span class="text-sm text-slate-500">{{ t('admin.jobs.intro') }}</span></template><template #right><UButton icon="i-lucide-rotate-cw" color="neutral" variant="subtle" :loading="manager.jobs.pending.value" @click="manager.refresh">{{ t('admin.extensions.refresh') }}</UButton></template></UDashboardToolbar>

  <div class="mb-5 grid grid-cols-2 gap-3 lg:grid-cols-4">
    <div v-for="state in ['available', 'running', 'retryable', 'discarded']" :key="state" class="border border-slate-200 bg-white p-4 dark:border-zinc-800 dark:bg-zinc-900"><p class="text-xs uppercase text-slate-500">{{ state }}</p><p class="mt-2 text-2xl font-semibold tabular-nums">{{ counts[state] || 0 }}</p></div>
  </div>

  <section class="mb-5 border border-slate-200 bg-white dark:border-zinc-800 dark:bg-zinc-900">
    <div class="border-b border-slate-200 px-4 py-3 text-sm font-semibold dark:border-zinc-800">{{ t('admin.jobs.queues') }}</div>
    <div class="divide-y divide-slate-200 dark:divide-zinc-800"><div v-for="queue in manager.overview.data.value.queues" :key="queue.name" class="flex flex-wrap items-center justify-between gap-3 px-4 py-3"><div><p class="font-mono text-sm font-medium">{{ queue.name }}</p><p class="mt-1 text-xs text-slate-500">{{ queue.available }} waiting · {{ queue.running }} running · {{ queue.failed }} failed</p></div><UButton v-if="canManage" size="xs" :icon="queue.pausedAt ? 'i-lucide-play' : 'i-lucide-pause'" color="neutral" variant="subtle" :loading="manager.busy.value === `queue:${queue.name}`" @click="manager.queueAction(queue.name, !queue.pausedAt)">{{ queue.pausedAt ? t('admin.jobs.resume') : t('admin.jobs.pause') }}</UButton></div></div>
  </section>

  <div class="mb-3 grid gap-2 sm:grid-cols-3"><UInput v-model="manager.filters.kind" icon="i-lucide-search" :placeholder="t('admin.jobs.filterKind')" /><USelect v-model="manager.filters.state" :items="stateOptions" /><USelect v-model="manager.filters.queue" :items="queueOptions" /></div>
  <div class="overflow-x-auto border border-slate-200 bg-white dark:border-zinc-800 dark:bg-zinc-900"><table class="min-w-full text-left text-sm"><thead class="bg-slate-50 text-xs text-slate-500 dark:bg-zinc-950"><tr><th class="px-3 py-3">ID</th><th class="px-3 py-3">Kind / Queue</th><th class="px-3 py-3">State</th><th class="px-3 py-3">Attempts</th><th v-for="column in columnContributions" :key="`${column.extensionId}:${column.contributionId}`" class="px-3 py-3">{{ contributionLabel(column) }}</th><th class="px-3 py-3 text-right">{{ t('admin.jobs.actions') }}</th></tr></thead><tbody class="divide-y divide-slate-200 dark:divide-zinc-800"><tr v-for="job in manager.jobs.data.value" :key="job.id"><td class="px-3 py-3 font-mono">#{{ job.id }}</td><td class="px-3 py-3"><p class="font-medium">{{ job.kind }}</p><p class="text-xs text-slate-500">{{ job.queue }}</p></td><td class="px-3 py-3"><UBadge :color="jobStateColor(job.state)" variant="subtle">{{ job.state }}</UBadge></td><td class="px-3 py-3 tabular-nums">{{ job.attempt }}/{{ job.maxAttempts }}</td><td v-for="column in columnContributions" :key="`${job.id}:${column.extensionId}:${column.contributionId}`" class="px-3 py-3"><SFAdminExtensionContribution :metadata="column" :context="{ job }" /></td><td class="px-3 py-3"><div class="flex justify-end gap-1"><SFAdminExtensionContribution v-for="action in actionContributions" :key="`${job.id}:${action.extensionId}:${action.contributionId}`" :metadata="action" :context="{ job }" /><UButton size="xs" icon="i-lucide-eye" color="neutral" variant="ghost" @click="manager.detail(job.id)" /><UButton v-if="canManage && jobCanRetry(job.state)" size="xs" icon="i-lucide-refresh-cw" :loading="manager.busy.value === `${job.id}:retry`" @click="manager.jobAction(job.id, 'retry')" /><UButton v-if="canManage && jobCanCancel(job.state)" size="xs" icon="i-lucide-ban" color="error" variant="ghost" :loading="manager.busy.value === `${job.id}:cancel`" @click="manager.jobAction(job.id, 'cancel')" /></div></td></tr></tbody></table><div v-if="!manager.jobs.data.value.length && !manager.jobs.pending.value" class="p-10 text-center text-sm text-slate-500">{{ t('admin.jobs.empty') }}</div></div>

  <UModal :open="Boolean(manager.selected.value)" @update:open="value => { if (!value) closeDetail() }"><template #content><div v-if="manager.selected.value" class="max-h-[85vh] overflow-y-auto p-5"><div class="flex items-center justify-between"><h3 class="font-semibold">Job #{{ manager.selected.value.id }}</h3><UButton icon="i-lucide-x" color="neutral" variant="ghost" @click="closeDetail" /></div><dl class="mt-4 grid gap-3 text-sm sm:grid-cols-2"><div><dt class="text-slate-500">Kind</dt><dd>{{ manager.selected.value.kind }}</dd></div><div><dt class="text-slate-500">Queue</dt><dd>{{ manager.selected.value.queue }}</dd></div></dl><pre class="mt-4 overflow-auto bg-zinc-950 p-3 text-xs text-zinc-200">{{ JSON.stringify(manager.selected.value.args, null, 2) }}</pre><SFAdminExtensionContribution v-for="section in detailContributions" :key="`${section.extensionId}:${section.contributionId}`" class="mt-4" :metadata="section" :context="{ job: manager.selected.value }" /></div></template></UModal>
</template>
