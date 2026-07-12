<script setup lang="ts">
import { apiErrorMessage } from '~/composables/useApiClient'
import type { ModerationDecision, ModerationPendingItem, ModerationReportItem, ModerationReviewContext, ModerationTargetType } from '~/composables/useModerationApi'

definePageMeta({ requiresAuth: true, middleware: ['moderation-review'] })
type WorkbenchTab = 'pending' | 'reports' | 'history'

const { t } = useI18n()
const route = useRoute()
const router = useRouter()
const moderationApi = useModerationApi()
const tab = computed<WorkbenchTab>(() => ['pending', 'reports', 'history'].includes(String(route.query.tab)) ? route.query.tab as WorkbenchTab : 'pending')
const targetType = ref<ModerationTargetType | undefined>()
const currentPage = ref(1)
const selectedContext = ref<ModerationReviewContext | null>(null)
const selectedReportId = ref<number>()
const contextLoading = ref(false)
const errorMessage = ref('')

const { data: counts, refresh: refreshCounts } = await useAsyncData('moderation-workbench-counts', () => moderationApi.getCounts(), { default: () => ({ pendingContent: 0, openReports: 0, processedToday: 0 }) })
const { data: list, pending, refresh } = await useAsyncData(
  () => `moderation-workbench-${tab.value}-${targetType.value || 'all'}-${currentPage.value}`,
  async () => {
    const filters = { targetType: targetType.value, page: currentPage.value, perPage: 20 }
    if (tab.value === 'reports') return moderationApi.listReportItems(filters)
    if (tab.value === 'history') return moderationApi.listHistory(filters)
    return moderationApi.listPending(filters)
  },
  { watch: [tab, targetType, currentPage], default: () => ({ items: [], total: 0, page: 1, perPage: 20 }) }
)

const queueItems = computed(() => list.value.items as Array<ModerationPendingItem | ModerationReportItem>)
const historyItems = computed(() => list.value.items as ModerationDecision[])
const totalPages = computed(() => Math.max(1, Math.ceil(list.value.total / list.value.perPage)))
const tabs = computed(() => [
  { value: 'pending' as const, label: t('moderation.workbench.pending'), count: counts.value.pendingContent },
  { value: 'reports' as const, label: t('moderation.workbench.reports'), count: counts.value.openReports },
  { value: 'history' as const, label: t('moderation.workbench.history'), count: counts.value.processedToday }
])

async function selectTab(value: WorkbenchTab) {
  selectedContext.value = null
  currentPage.value = 1
  await router.replace({ query: { ...route.query, tab: value === 'pending' ? undefined : value } })
}

async function openItem(item: ModerationPendingItem | ModerationReportItem) {
  contextLoading.value = true
  errorMessage.value = ''
  const source = tab.value === 'reports' ? 'report' : 'pre_publish'
  selectedReportId.value = 'reasonCode' in item ? item.id : undefined
  try {
    selectedContext.value = await moderationApi.getContext(source, item.targetType, item.targetId, selectedReportId.value)
  } catch (error) {
    errorMessage.value = apiErrorMessage(error) || t('moderation.workbench.contextFailed')
  } finally {
    contextLoading.value = false
  }
}

async function decisionCompleted() {
  selectedContext.value = null
  await Promise.all([refresh(), refreshCounts()])
}

async function refreshAll() {
  await Promise.all([refresh(), refreshCounts()])
}
</script>

<template>
  <SFPageOutlet page="moderation.review">
  <main class="mx-auto w-full max-w-6xl px-4 py-6 sm:px-6 lg:py-8">
    <header class="flex flex-wrap items-start justify-between gap-4">
      <div><h1 class="text-2xl font-bold text-slate-950 dark:text-zinc-50">{{ t('moderation.workbench.title') }}</h1><p class="mt-1 text-sm text-slate-500 dark:text-zinc-400">{{ t('moderation.workbench.description') }}</p></div>
      <UButton icon="i-lucide-rotate-cw" color="neutral" variant="subtle" :loading="pending" @click="refreshAll">{{ t('admin.home.refresh') }}</UButton>
    </header>

    <div class="mt-6 grid grid-cols-3 gap-2 sm:gap-4">
      <div class="border border-slate-200 bg-white p-3 dark:border-zinc-800 dark:bg-zinc-900"><span class="text-xs text-slate-500">{{ t('moderation.workbench.pending') }}</span><strong class="mt-1 block text-xl text-amber-600">{{ counts.pendingContent }}</strong></div>
      <div class="border border-slate-200 bg-white p-3 dark:border-zinc-800 dark:bg-zinc-900"><span class="text-xs text-slate-500">{{ t('moderation.workbench.reports') }}</span><strong class="mt-1 block text-xl text-red-700 dark:text-red-300">{{ counts.openReports }}</strong></div>
      <div class="border border-slate-200 bg-white p-3 dark:border-zinc-800 dark:bg-zinc-900"><span class="text-xs text-slate-500">{{ t('moderation.workbench.processedToday') }}</span><strong class="mt-1 block text-xl text-[var(--sf-accent)]">{{ counts.processedToday }}</strong></div>
    </div>

    <nav class="mt-6 flex flex-wrap items-end gap-4 border-b border-slate-200 dark:border-zinc-800" :aria-label="t('moderation.workbench.queueTabs')">
      <button v-for="item in tabs" :key="item.value" type="button" class="border-b-2 px-1 py-3 text-sm font-semibold" :class="tab === item.value ? 'border-[var(--sf-accent)] text-[var(--sf-accent)]' : 'border-transparent text-slate-500'" @click="selectTab(item.value)">{{ item.label }} <span class="ml-1 text-xs">{{ item.count }}</span></button>
      <select v-if="tab !== 'history'" v-model="targetType" class="sf-input mb-2 ml-auto" :aria-label="t('admin.moderation.filterType')"><option :value="undefined">{{ t('admin.moderation.typeAll') }}</option><option value="topic">{{ t('admin.moderation.typeTopic') }}</option><option value="comment">{{ t('admin.moderation.typeComment') }}</option></select>
    </nav>

    <SFAlert v-if="errorMessage" variant="danger" :title="errorMessage" closable class="mt-4" @close="errorMessage = ''" />
    <div v-if="contextLoading" class="mt-4"><SFSkeleton width="100%" height="240px" /></div>
    <ModerationContextPanel v-else-if="selectedContext" class="mt-4" :context="selectedContext" :report-id="selectedReportId" @close="selectedContext = null" @decided="decisionCompleted" />

    <div v-if="tab === 'history'" class="mt-4"><ModerationDecisionTable :items="historyItems" :loading="pending" /></div>
    <div v-else class="mt-4 space-y-3">
      <SFSkeleton v-if="pending" width="100%" height="120px" />
      <ModerationQueueItem v-for="item in queueItems" v-else :key="`${tab}-${item.targetType}-${item.targetId}-${'id' in item ? item.id : ''}`" :item="item" :source="tab === 'reports' ? 'report' : 'pre_publish'" @open="openItem(item)" />
      <SFEmptyState v-if="!pending && !queueItems.length" :title="t('moderation.workbench.emptyTitle')" :description="t('moderation.workbench.emptyDescription')" />
    </div>
    <div v-if="totalPages > 1" class="mt-6 flex justify-center"><SFPagination v-model:page="currentPage" :total-pages="totalPages" /></div>
  </main>

  </SFPageOutlet>
</template>
