<script setup lang="ts">
/**
 * 宿主 body 岛：moderation.review。主题 L1 挂载；路由页仅 outlet + fail-closed 回退。
 */

import { apiErrorMessage } from '~/composables/useApiClient'
import type {
  ModerationAction,
  ModerationDecision,
  ModerationPendingItem,
  ModerationReportItem,
  ModerationReviewContext,
  PagedModerationList
} from '~/composables/useModerationApi'
import {
  REVIEW_REQUIRED_ACTIONS,
  isDecisionReadonly,
  parsePositiveInt,
  parseTargetType,
  parseWorkbenchTab,
  queueQueryForWorkbench,
  queueItemKey,
  reviewQuery,
  selectionFromQueueItem,
  selectionFromQuery,
  selectionKey
} from '~/utils/moderationWorkbench'
import type { ModerationReviewSelection, ModerationWorkbenchTab, ModerationWorkbenchTypeFilter } from '~/utils/moderationWorkbench'

type QueueRecord = ModerationPendingItem | ModerationReportItem | ModerationDecision

const { t } = useI18n()
const route = useRoute()
const router = useRouter()
const toast = useToast()
const { format: formatDate } = useSiteDateTime()
const moderationApi = useModerationApi()

const tab = computed<ModerationWorkbenchTab>(() => parseWorkbenchTab(route.query.tab))
const typeFilter = computed<ModerationWorkbenchTypeFilter>(() => parseTargetType(route.query.targetType))
const currentPage = computed(() => parsePositiveInt(route.query.page, 1))
const apiTypeFilter = computed(() => typeFilter.value === 'all' ? undefined : typeFilter.value)
const reviewSelection = computed(() => selectionFromQuery(route.query as Record<string, unknown>, tab.value))
const reviewMode = computed(() => Boolean(reviewSelection.value))
const reviewKey = computed(() => selectionKey(reviewSelection.value))
const readonlyReview = computed(() => isDecisionReadonly(tab.value))
const queueScrollKey = computed(() => `${tab.value}:${typeFilter.value}:${currentPage.value}`)

const queueScroll = useState<Record<string, number>>('moderation-workbench-scroll', () => ({}))
const noteDrafts = ref<Record<string, string>>({})
const fieldError = ref('')
const loadError = ref('')
const submitting = ref<ModerationAction | null>(null)
const mobileQueueOpen = ref(false)
const mobileActionOpen = ref(false)
const openedReviewFromQueue = ref(false)

const { data: counts, error: countsError, refresh: refreshCounts } = await useAsyncData(
  'moderation-workbench-counts',
  () => moderationApi.getCounts(),
  { default: () => ({ pendingContent: 0, openReports: 0, processedToday: 0 }) }
)

const { data: list, pending: listPending, error: listError, refresh: refreshList } = await useAsyncData(
  () => `moderation-workbench-list:${tab.value}:${typeFilter.value}:${currentPage.value}`,
  async () => {
    const filters = { targetType: apiTypeFilter.value, page: currentPage.value, perPage: 20 }
    if (tab.value === 'reports') return moderationApi.listReportItems(filters) as Promise<PagedModerationList<QueueRecord>>
    if (tab.value === 'history') return moderationApi.listHistory(filters) as Promise<PagedModerationList<QueueRecord>>
    return moderationApi.listPending(filters) as Promise<PagedModerationList<QueueRecord>>
  },
  { watch: [tab, typeFilter, currentPage], default: () => ({ items: [], total: 0, page: 1, perPage: 20 }) }
)

const { data: reviewContext, pending: contextPending, refresh: refreshContext } = await useAsyncData(
  () => `moderation-workbench-context:${reviewKey.value || 'none'}`,
  async () => {
    const selection = reviewSelection.value
    if (!selection) return null
    loadError.value = ''
    try {
      return await moderationApi.getContext(selection.source, selection.targetType, selection.targetId, selection.reportId)
    } catch (error) {
      loadError.value = apiErrorMessage(error) || t('moderation.workbench.contextFailed')
      return null
    }
  },
  { watch: [reviewKey], default: () => null }
)

const countsAvailable = computed(() => !countsError.value)
const queueErrorMessage = computed(() => listError.value
  ? apiErrorMessage(listError.value) || t('moderation.workbench.queueFailed')
  : '')
const sourceTabs = computed(() => [
  { value: 'pending' as const, icon: 'i-lucide-clock-3', label: t('moderation.workbench.pending'), count: countsAvailable.value ? counts.value.pendingContent : null },
  { value: 'reports' as const, icon: 'i-lucide-flag', label: t('moderation.workbench.reports'), count: countsAvailable.value ? counts.value.openReports : null },
  { value: 'history' as const, icon: 'i-lucide-history', label: t('moderation.workbench.history'), count: countsAvailable.value ? counts.value.processedToday : null }
])
const typeFilters = computed(() => [
  { value: 'all' as const, icon: 'i-lucide-layers-3', label: t('admin.moderation.typeAll') },
  { value: 'topic' as const, icon: 'i-lucide-file-text', label: t('admin.moderation.typeTopic') },
  { value: 'comment' as const, icon: 'i-lucide-message-square', label: t('admin.moderation.typeComment') }
])
const items = computed(() => list.value.items)
const historyItems = computed(() => items.value as ModerationDecision[])
const queueItems = computed(() => items.value as Array<ModerationPendingItem | ModerationReportItem>)
const totalPages = computed(() => Math.max(1, Math.ceil(list.value.total / list.value.perPage)))
const activeNote = computed({
  get: () => noteDrafts.value[reviewKey.value] || '',
  set: (value: string) => {
    if (!reviewKey.value) return
    noteDrafts.value = { ...noteDrafts.value, [reviewKey.value]: value }
  }
})
const activeIndex = computed(() => {
  const selection = reviewSelection.value
  if (!selection) return -1
  return items.value.findIndex(item => selectionKey(selectionFromQueueItem(tab.value, item)) === selectionKey(selection))
})
const progressLabel = computed(() => activeIndex.value >= 0 ? t('moderation.workbench.progress', { current: activeIndex.value + 1, total: items.value.length }) : '')
const canPrevious = computed(() => activeIndex.value > 0)
const canNext = computed(() => activeIndex.value >= 0 && activeIndex.value < items.value.length - 1)
const pageRangeLabel = computed(() => {
  if (!list.value.total) return t('moderation.workbench.rangeEmpty')
  const start = (currentPage.value - 1) * list.value.perPage + 1
  const end = Math.min(list.value.total, start + items.value.length - 1)
  return t('moderation.workbench.rangeLabel', { start, end, total: list.value.total })
})
const headerTitle = computed(() => sourceTabs.value.find(item => item.value === tab.value)?.label || t('moderation.workbench.pending'))
const headerDescription = computed(() => tab.value === 'history'
  ? t('moderation.workbench.historyDescription')
  : t('moderation.workbench.queueDescription'))

watch(reviewKey, () => {
  fieldError.value = ''
  loadError.value = ''
  mobileActionOpen.value = false
  mobileQueueOpen.value = false
})

watch(reviewMode, async (active, previous) => {
  if (!active && previous && import.meta.client) {
    await nextTick()
    window.scrollTo({ top: queueScroll.value[queueScrollKey.value] || 0, behavior: 'auto' })
  }
})

function queueQuery(overrides: Record<string, string | number | undefined> = {}) {
  return queueQueryForWorkbench(route.query as Record<string, unknown>, overrides)
}

function saveScrollPosition() {
  if (!import.meta.client) return
  queueScroll.value = { ...queueScroll.value, [queueScrollKey.value]: window.scrollY }
}

async function selectTab(value: ModerationWorkbenchTab) {
  await router.replace({ query: queueQuery({ tab: value, page: undefined }) })
}

async function selectType(value: ModerationWorkbenchTypeFilter) {
  await router.replace({ query: queueQuery({ targetType: value, page: undefined }) })
}

async function onTypeSelect(event: Event) {
  await selectType((event.target as HTMLSelectElement).value as ModerationWorkbenchTypeFilter)
}

async function selectPage(value: number) {
  saveScrollPosition()
  await router.replace({ query: queueQuery({ page: value }) })
}

async function openSelection(selection: ModerationReviewSelection) {
  saveScrollPosition()
  openedReviewFromQueue.value = true
  await router.push({ query: { ...queueQuery(), ...reviewQuery(selection) } })
}

async function openItem(item: QueueRecord) {
  await openSelection(selectionFromQueueItem(tab.value, item))
}

async function returnToQueue() {
  if (openedReviewFromQueue.value && import.meta.client) {
    openedReviewFromQueue.value = false
    router.back()
    return
  }
  await router.replace({ query: queueQuery() })
}

async function navigateWithinQueue(direction: 'previous' | 'next') {
  const offset = direction === 'previous' ? -1 : 1
  const nextItem = items.value[activeIndex.value + offset]
  if (!nextItem) {
    await router.replace({ query: queueQuery() })
    return
  }
  await router.replace({ query: { ...queueQuery(), ...reviewQuery(selectionFromQueueItem(tab.value, nextItem)) } })
}

async function refreshAll() {
  await Promise.all([refreshCounts(), refreshList(), reviewMode.value ? refreshContext() : Promise.resolve()])
}

async function submitDecision(action: ModerationAction) {
  const context = reviewContext.value as ModerationReviewContext | null
  const selection = reviewSelection.value
  if (!context || !selection || readonlyReview.value) return
  fieldError.value = ''
  if (REVIEW_REQUIRED_ACTIONS.has(action) && !activeNote.value.trim()) {
    fieldError.value = t('moderation.workbench.noteRequired')
    return
  }
  submitting.value = action
  const nextItem = items.value[activeIndex.value + 1]
  try {
    await moderationApi.submitDecision({
      source: context.source,
      targetType: context.targetType,
      targetId: context.targetId,
      reportId: selection.reportId,
      action,
      reviewNote: activeNote.value
    })
    const drafts = { ...noteDrafts.value }
    delete drafts[reviewKey.value]
    noteDrafts.value = drafts
    toast.add({ color: 'primary', icon: 'i-lucide-check', title: t('moderation.workbench.decisionSaved'), duration: 10000 })
    await Promise.all([refreshCounts(), refreshList()])
    if (nextItem) {
      await router.replace({ query: { ...queueQuery(), ...reviewQuery(selectionFromQueueItem(tab.value, nextItem)) } })
    } else {
      await router.replace({ query: queueQuery() })
    }
  } catch (error) {
    fieldError.value = apiErrorMessage(error) || t('moderation.workbench.decisionFailed')
  } finally {
    submitting.value = null
  }
}

function historySummary(item: ModerationDecision) {
  return `${t(`moderation.action.${item.action}`)} / ${item.reviewerName || t('moderation.workbench.unknownReviewer')}`
}
</script>

<template>
  <section class="sforum-moderation" :class="{ 'sforum-moderation--review': reviewMode }">
    <div class="sforum-moderation__mobile-bar">
      <button type="button" class="sforum-moderation-mobile-button" :aria-label="t('moderation.workbench.openQueueDrawer')" @click="mobileQueueOpen = true">
        <UIcon name="i-lucide-menu" class="size-4" aria-hidden="true" />
        {{ t('moderation.workbench.queueTabs') }}
      </button>
      <button v-if="reviewMode" type="button" class="sforum-moderation-mobile-button" :aria-label="t('moderation.workbench.openDecisionDrawer')" @click="mobileActionOpen = true">
        <UIcon name="i-lucide-panel-right-open" class="size-4" aria-hidden="true" />
        {{ t('moderation.workbench.decisionRail') }}
      </button>
    </div>

    <div class="sforum-moderation__layout" :class="{ 'sforum-moderation__layout--with-right': reviewMode || tab !== 'history' }">
      <aside class="sforum-moderation__left" :aria-label="t('moderation.workbench.queueTabs')">
        <button v-if="reviewMode" type="button" class="sforum-moderation-sidebar-primary" @click="returnToQueue">
          <UIcon name="i-lucide-arrow-left" class="size-4" aria-hidden="true" />
          {{ t('moderation.workbench.backToQueue') }}
        </button>

        <p class="sforum-moderation__nav-label">{{ t('moderation.workbench.sources') }}</p>
        <nav class="sforum-moderation__side-nav" :aria-label="t('moderation.workbench.sources')">
          <button
            v-for="item in sourceTabs"
            :key="item.value"
            type="button"
            class="sforum-moderation__side-link"
            :class="{ 'is-active': tab === item.value }"
            @click="selectTab(item.value)"
          >
            <span>
              <UIcon :name="item.icon" class="size-4" aria-hidden="true" />
              {{ item.label }}
            </span>
            <span v-if="item.count !== null" class="sforum-moderation__side-count">{{ item.count }}</span>
          </button>
        </nav>

        <p class="sforum-moderation__nav-label">{{ t('admin.moderation.filterType') }}</p>
        <nav class="sforum-moderation__side-nav" :aria-label="t('admin.moderation.filterType')">
          <button
            v-for="item in typeFilters"
            :key="item.value"
            type="button"
            class="sforum-moderation__side-link"
            :class="{ 'is-active': typeFilter === item.value }"
            @click="selectType(item.value)"
          >
            <span>
              <UIcon :name="item.icon" class="size-4" aria-hidden="true" />
              {{ item.label }}
            </span>
          </button>
        </nav>

        <template v-if="reviewMode">
          <p class="sforum-moderation__nav-label">{{ t('moderation.workbench.currentQueue') }}</p>
          <div class="sforum-moderation-compact-list">
            <button
              v-for="item in items"
              :key="queueItemKey(tab, item)"
              type="button"
              class="sforum-moderation-compact-item"
              :class="{ 'is-active': selectionKey(selectionFromQueueItem(tab, item)) === reviewKey }"
              @click="openItem(item)"
            >
              <UIcon :name="'reasonCode' in item ? 'i-lucide-flag' : item.targetType === 'topic' ? 'i-lucide-file-text' : 'i-lucide-message-square'" class="size-4" aria-hidden="true" />
              <span>
                <strong>{{ 'title' in item ? item.title : `${t(`admin.moderation.type.${item.targetType}`)} #${item.targetId}` }}</strong>
                <small>{{ 'action' in item ? historySummary(item) : 'reasonCode' in item ? t(`moderation.reason.${item.reasonCode}`) : item.triggers.map(trigger => t(`moderation.trigger.${trigger}`)).join(' / ') }}</small>
              </span>
            </button>
          </div>
        </template>

        <div class="sforum-moderation__side-foot">
          <span><UIcon name="i-lucide-shield-check" class="size-4" aria-hidden="true" />{{ t('moderation.workbench.permissionHint') }}</span>
          <span><UIcon name="i-lucide-book-open" class="size-4" aria-hidden="true" />{{ pageRangeLabel }}</span>
        </div>
      </aside>

      <main v-if="!reviewMode" class="sforum-moderation__main">
        <header class="sforum-moderation__head">
          <div>
            <h1>{{ headerTitle }}</h1>
            <p>{{ headerDescription }}</p>
          </div>
          <button type="button" class="sforum-moderation-icon-button" :aria-label="t('admin.home.refresh')" @click="refreshAll">
            <UIcon name="i-lucide-refresh-cw" class="size-4" :class="{ 'animate-spin': listPending }" aria-hidden="true" />
          </button>
        </header>

        <div class="sforum-moderation__toolbar">
          <div class="sforum-moderation__segments" :aria-label="t('moderation.workbench.queueTabs')">
            <button v-for="item in sourceTabs" :key="item.value" type="button" :class="{ 'is-active': tab === item.value }" @click="selectTab(item.value)">
              {{ item.label }} <span v-if="item.count !== null">{{ item.count }}</span>
            </button>
          </div>
          <select :value="typeFilter" class="sforum-moderation__select" :aria-label="t('admin.moderation.filterType')" @change="onTypeSelect">
            <option v-for="item in typeFilters" :key="item.value" :value="item.value">{{ item.label }}</option>
          </select>
        </div>

        <SFAlert v-if="queueErrorMessage" variant="danger" :title="queueErrorMessage" class="mb-4" />

        <section class="sforum-moderation__queue" :aria-label="t('moderation.workbench.queueTabs')">
          <template v-if="listPending">
            <SFSkeleton width="100%" height="88px" />
            <SFSkeleton width="100%" height="88px" />
          </template>
          <template v-else-if="tab === 'history'">
            <button
              v-for="item in historyItems"
              :key="queueItemKey(tab, item)"
              type="button"
              class="sforum-moderation-history-row"
              @click="openItem(item)"
            >
              <span>
                <strong>{{ t(`admin.moderation.type.${item.targetType}`) }} #{{ item.targetId }}</strong>
                <small>{{ historySummary(item) }}</small>
              </span>
              <time>{{ formatDate(item.createdAt) }}</time>
            </button>
          </template>
          <template v-else>
            <ModerationQueueItem
              v-for="item in queueItems"
              :key="queueItemKey(tab, item)"
              :item="item"
              :source="tab === 'reports' ? 'report' : 'pre_publish'"
              :active="selectionKey(selectionFromQueueItem(tab, item)) === reviewKey"
              @open="openItem(item)"
            />
          </template>

          <SFEmptyState
            v-if="!listPending && !queueErrorMessage && !items.length"
            icon-label="MOD"
            :title="typeFilter === 'all' ? t('moderation.workbench.emptyTitle') : t('moderation.workbench.filterEmptyTitle')"
            :description="t('moderation.workbench.emptyDescription')"
          />
        </section>

        <div v-if="totalPages > 1" class="sforum-moderation__pagination">
          <SFPagination :page="currentPage" :total-pages="totalPages" @update:page="selectPage" />
        </div>
      </main>

      <ModerationReviewReader
        v-else
        :context="reviewContext"
        note-id="moderation-review-note"
        :loading="contextPending"
        :tab="tab"
        @back="returnToQueue"
      />

      <aside v-if="!reviewMode && tab !== 'history'" class="sforum-moderation__right" :aria-label="t('moderation.workbench.queueOverview')">
        <section class="sforum-moderation-rail-section">
          <header class="sforum-moderation-rail-section__head">
            <h2>{{ t('moderation.workbench.queueOverview') }}</h2>
            <span>{{ pageRangeLabel }}</span>
          </header>
          <div v-if="countsAvailable" class="sforum-moderation-stats">
            <div><strong>{{ counts.pendingContent }}</strong><span>{{ t('moderation.workbench.pending') }}</span></div>
            <div><strong>{{ counts.openReports }}</strong><span>{{ t('moderation.workbench.reports') }}</span></div>
            <div><strong>{{ counts.processedToday }}</strong><span>{{ t('moderation.workbench.processedToday') }}</span></div>
          </div>
          <p v-else class="sforum-moderation-rail-copy" role="alert">
            {{ t('moderation.workbench.countsFailed') }}
          </p>
        </section>
        <section class="sforum-moderation-rail-section">
          <header class="sforum-moderation-rail-section__head">
            <h3>{{ t('moderation.workbench.workflowTitle') }}</h3>
          </header>
          <p class="sforum-moderation-rail-copy">{{ t('moderation.workbench.workflowDescription') }}</p>
        </section>
        <section class="sforum-moderation-rail-section">
          <header class="sforum-moderation-rail-section__head">
            <h3>{{ t('moderation.workbench.stateRestoreTitle') }}</h3>
          </header>
          <p class="sforum-moderation-rail-copy">{{ t('moderation.workbench.stateRestoreDescription') }}</p>
        </section>
      </aside>

      <ModerationDecisionRail
        v-if="reviewMode"
        v-model:note="activeNote"
        :context="reviewContext"
        note-id="moderation-review-note-mobile"
        :readonly="readonlyReview"
        :submitting="submitting"
        :error="fieldError || loadError"
        :progress-label="progressLabel"
        :has-previous="canPrevious"
        :has-next="canNext"
        @decide="submitDecision"
        @previous="navigateWithinQueue('previous')"
        @next="navigateWithinQueue('next')"
      />
    </div>

    <button v-if="mobileQueueOpen || mobileActionOpen" type="button" class="sforum-mobile-drawer__backdrop" :aria-label="t('moderation.close')" @click="mobileQueueOpen = false; mobileActionOpen = false" />
    <aside v-if="mobileQueueOpen" class="sforum-mobile-drawer sforum-mobile-drawer--left">
      <header class="sforum-mobile-drawer__head">
        <strong>{{ t('moderation.workbench.queueTabs') }}</strong>
        <button type="button" class="sforum-moderation-icon-button" :aria-label="t('moderation.close')" @click="mobileQueueOpen = false">
          <UIcon name="i-lucide-x" class="size-4" aria-hidden="true" />
        </button>
      </header>
      <div class="sforum-moderation-compact-list">
        <button
          v-for="item in items"
          :key="queueItemKey(tab, item)"
          type="button"
          class="sforum-moderation-compact-item"
          :class="{ 'is-active': selectionKey(selectionFromQueueItem(tab, item)) === reviewKey }"
          @click="openItem(item); mobileQueueOpen = false"
        >
          <UIcon :name="'reasonCode' in item ? 'i-lucide-flag' : item.targetType === 'topic' ? 'i-lucide-file-text' : 'i-lucide-message-square'" class="size-4" aria-hidden="true" />
          <span>
            <strong>{{ 'title' in item ? item.title : `${t(`admin.moderation.type.${item.targetType}`)} #${item.targetId}` }}</strong>
            <small>{{ 'action' in item ? historySummary(item) : 'reasonCode' in item ? t(`moderation.reason.${item.reasonCode}`) : item.triggers.map(trigger => t(`moderation.trigger.${trigger}`)).join(' / ') }}</small>
          </span>
        </button>
      </div>
    </aside>
    <aside v-if="mobileActionOpen" class="sforum-mobile-drawer sforum-mobile-drawer--right">
      <header class="sforum-mobile-drawer__head">
        <strong>{{ t('moderation.workbench.decisionRail') }}</strong>
        <button type="button" class="sforum-moderation-icon-button" :aria-label="t('moderation.close')" @click="mobileActionOpen = false">
          <UIcon name="i-lucide-x" class="size-4" aria-hidden="true" />
        </button>
      </header>
      <ModerationDecisionRail
        v-model:note="activeNote"
        :context="reviewContext"
        :readonly="readonlyReview"
        :submitting="submitting"
        :error="fieldError || loadError"
        :progress-label="progressLabel"
        :has-previous="canPrevious"
        :has-next="canNext"
        @decide="submitDecision"
        @previous="navigateWithinQueue('previous')"
        @next="navigateWithinQueue('next')"
      />
    </aside>
  </section>
</template>

<style src="~/assets/css/sforum-moderation.css"></style>
