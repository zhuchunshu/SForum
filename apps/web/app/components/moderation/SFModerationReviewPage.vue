<script setup lang="ts">
import { useModerationApi } from '~/composables/moderation/useModerationApi'
import { FORUM_PERMISSIONS, usePermissions } from '~/composables/identity/usePermissions'
import { useForumApi } from '~/composables/forum/useForumApi'
import SFHomeNavigation from '~/components/forum/SFHomeNavigation.vue'
import SFContentColumnFooter from '~/components/forum/SFContentColumnFooter.vue'
import SFPublicPageHeader from '~/components/public/SFPublicPageHeader.vue'
import ModerationDecisionRail from '~/components/moderation/ModerationDecisionRail.vue'
import ModerationQueueItem from '~/components/moderation/ModerationQueueItem.vue'
import ModerationQueueRail from '~/components/moderation/ModerationQueueRail.vue'
import ModerationReviewReader from '~/components/moderation/ModerationReviewReader.vue'
import ModerationWorkbenchNav from '~/components/moderation/ModerationWorkbenchNav.vue'
/**
 * 宿主 body 岛：moderation.review。
 * 左右栏对齐首页 + 通知页公共三栏 chrome；队列/审阅业务逻辑不变。
 */

import { apiErrorMessage } from '~/composables/useApiClient'
import type {
  ModerationAction,
  ModerationDecision,
  ModerationPendingItem,
  ModerationQueueCounts,
  ModerationReportItem,
  ModerationReviewContext,
  PagedModerationList
} from '~/composables/moderation/useModerationApi'
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
} from '~/utils/moderation/moderationWorkbench'
import type { ModerationReviewSelection, ModerationWorkbenchTab, ModerationWorkbenchTypeFilter } from '~/utils/moderation/moderationWorkbench'

type QueueRecord = ModerationPendingItem | ModerationReportItem | ModerationDecision

const { t } = useI18n()
const route = useRoute()
const router = useRouter()
const toast = useToast()
const { format: formatDate } = useSiteDateTime()
const moderationApi = useModerationApi()
const forumApi = useForumApi()
const { can } = usePermissions()

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
// 与首页/通知页共用抽屉状态，避免各页自造一套移动端 chrome
const mobileMenuOpen = useState<boolean>('forum-mobile-menu-open', () => false)
const mobileInfoOpen = useState<boolean>('forum-mobile-info-open', () => false)
const openedReviewFromQueue = ref(false)

const emptyCounts = (): ModerationQueueCounts => ({
  pendingContent: 0,
  openReports: 0,
  processedToday: 0,
  historyTotal: 0
})

const { data: counts, error: countsError, refresh: refreshCounts } = await useAsyncData(
  'moderation-workbench-counts',
  async () => {
    const raw = await moderationApi.getCounts()
    // 兼容旧后端缺字段；数字化避免异常类型导致始终显示 0/空
    return {
      pendingContent: Number(raw?.pendingContent ?? 0),
      openReports: Number(raw?.openReports ?? 0),
      processedToday: Number(raw?.processedToday ?? 0),
      historyTotal: Number(raw?.historyTotal ?? 0)
    } satisfies ModerationQueueCounts
  },
  { default: emptyCounts }
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

const { data: categoryGroups, pending: categoriesPending } = await useAsyncData(
  'moderation-workbench-categories',
  () => forumApi.listCategoryGroups(),
  { default: () => [] }
)

const countsAvailable = computed(() => !countsError.value)
const safeCounts = computed<ModerationQueueCounts>(() => counts.value || emptyCounts())
const categories = computed(() => categoryGroups.value.flatMap(group => group.categories || []))
const categoryTopicTotal = computed(() => categories.value.reduce((sum, category) => sum + (category.topicCount || 0), 0))
const canCreateTopic = computed(() => can(FORUM_PERMISSIONS.topicCreate))
const queueErrorMessage = computed(() => listError.value
  ? apiErrorMessage(listError.value) || t('moderation.workbench.queueFailed')
  : '')

/** 来源 tab 徽章：与对应列表 total 对齐；历史用 historyTotal，不用今日 KPI。 */
function sourceCountFor(source: ModerationWorkbenchTab): number | null {
  if (!countsAvailable.value) return null
  const c = safeCounts.value
  if (source === 'pending') {
    return tab.value === 'pending' ? Math.max(c.pendingContent, list.value.total) : c.pendingContent
  }
  if (source === 'reports') {
    return tab.value === 'reports' ? Math.max(c.openReports, list.value.total) : c.openReports
  }
  // history：优先全量 historyTotal；当前 tab 再与列表 total 取大，避免徽章与列表脱节
  const history = c.historyTotal > 0 ? c.historyTotal : c.processedToday
  return tab.value === 'history' ? Math.max(history, list.value.total) : history
}

const sourceTabs = computed(() => [
  { value: 'pending' as const, icon: 'i-lucide-clock-3', label: t('moderation.workbench.pending'), count: sourceCountFor('pending') },
  { value: 'reports' as const, icon: 'i-lucide-flag', label: t('moderation.workbench.reports'), count: sourceCountFor('reports') },
  { value: 'history' as const, icon: 'i-lucide-history', label: t('moderation.workbench.history'), count: sourceCountFor('history') }
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
const typeFilterLabel = computed(() => typeFilters.value.find(item => item.value === typeFilter.value)?.label || t('admin.moderation.typeAll'))
// 右栏大数字：当前来源队列总量（历史用全量，不用今日 KPI）
const overviewCount = computed(() => {
  if (!countsAvailable.value) return null
  return sourceCountFor(tab.value)
})
const overviewCountLabel = computed(() => {
  if (tab.value === 'reports') return t('moderation.workbench.reports')
  if (tab.value === 'history') return t('moderation.workbench.history')
  return t('moderation.workbench.pending')
})
const rightDrawerTitle = computed(() => reviewMode.value
  ? t('moderation.workbench.decisionRail')
  : t('moderation.workbench.queueOverview'))

watch(reviewKey, () => {
  fieldError.value = ''
  loadError.value = ''
  mobileInfoOpen.value = false
  mobileMenuOpen.value = false
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

function closeMobileDrawers() {
  mobileMenuOpen.value = false
  mobileInfoOpen.value = false
}

async function selectTab(value: ModerationWorkbenchTab) {
  closeMobileDrawers()
  await router.replace({ query: queueQuery({ tab: value, page: undefined }) })
}

async function selectType(value: ModerationWorkbenchTypeFilter) {
  closeMobileDrawers()
  await router.replace({ query: queueQuery({ targetType: value, page: undefined }) })
}

async function selectPage(value: number) {
  saveScrollPosition()
  await router.replace({ query: queueQuery({ page: value }) })
}

async function openSelection(selection: ModerationReviewSelection) {
  saveScrollPosition()
  openedReviewFromQueue.value = true
  closeMobileDrawers()
  await router.push({ query: { ...queueQuery(), ...reviewQuery(selection) } })
}

async function openItem(item: QueueRecord) {
  await openSelection(selectionFromQueueItem(tab.value, item))
}

async function returnToQueue() {
  closeMobileDrawers()
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

function isItemActive(item: QueueRecord) {
  return selectionKey(selectionFromQueueItem(tab.value, item)) === reviewKey.value
}
</script>

<template>
  <!-- 三栏 chrome 直接复用首页 sforum-home__* token（host chrome 路径同样生效） -->
  <main
    class="sforum-moderation sforum-home"
    data-sforum-island-body="forum.component.moderation_review"
    data-layout="fullwidth-3col"
    :class="{ 'sforum-moderation--review': reviewMode }"
  >
    <div class="sforum-home__layout sforum-home__layout--with-right">
      <div class="sforum-home__sidebar">
        <SFHomeNavigation
          desktop-only
          navigation-mode="route"
          :categories="categories"
          :total-topics="categoryTopicTotal"
          :pending="categoriesPending"
          :can-create-topic="canCreateTopic"
          :show-categories="false"
        >
          <template #after-navigation>
            <ModerationWorkbenchNav
              :source-tabs="sourceTabs"
              :type-filters="typeFilters"
              :tab="tab"
              :type-filter="typeFilter"
              :review-mode="reviewMode"
              :items="items"
              :active-key="reviewKey"
              @select-tab="selectTab"
              @select-type="selectType"
              @open-item="openItem"
              @back="returnToQueue"
            />
          </template>
        </SFHomeNavigation>
      </div>

      <section
        v-if="!reviewMode"
        class="sforum-home__main sforum-moderation__main sforum-content-column"
        :aria-labelledby="'moderation-page-title'"
      >
        <div class="sforum-moderation__mobile-nav">
          <SFHomeNavigation
            mobile-only
            navigation-mode="route"
            :categories="categories"
            :total-topics="categoryTopicTotal"
            :pending="categoriesPending"
            :can-create-topic="canCreateTopic"
          />
        </div>

        <SFPublicPageHeader
          class="sforum-moderation__head"
          title-id="moderation-page-title"
          :title="headerTitle"
          :subtitle="headerDescription"
        >
          <template #aside>
            <div class="sforum-moderation__head-actions">
              <button
                type="button"
                class="sforum-moderation-icon-button"
                :aria-label="t('admin.home.refresh')"
                @click="refreshAll"
              >
                <UIcon name="i-lucide-refresh-cw" class="size-4" :class="{ 'animate-spin': listPending }" aria-hidden="true" />
              </button>
              <button
                type="button"
                class="sforum-moderation-icon-button sforum-moderation__desktop-hidden"
                :aria-label="t('moderation.workbench.openRightRail')"
                @click="mobileInfoOpen = true"
              >
                <UIcon name="i-lucide-panel-right" class="size-[18px]" aria-hidden="true" />
              </button>
              <button
                type="button"
                class="sforum-moderation-icon-button sforum-moderation__desktop-hidden sforum-moderation__menu-button"
                :aria-label="t('moderation.workbench.openQueueDrawer')"
                @click="mobileMenuOpen = true"
              >
                <UIcon name="i-lucide-menu" class="size-[18px]" aria-hidden="true" />
              </button>
            </div>
          </template>
        </SFPublicPageHeader>

        <div class="sforum-moderation__filter-strip" :aria-label="t('moderation.workbench.queueTabs')">
          <button
            v-for="item in sourceTabs"
            :key="item.value"
            type="button"
            class="sforum-moderation__filter-button"
            :class="{ 'is-active': tab === item.value }"
            :aria-pressed="tab === item.value"
            @click="selectTab(item.value)"
          >
            {{ item.label }}
            <span v-if="item.count !== null">{{ item.count }}</span>
          </button>
        </div>
        <div class="sforum-moderation__filter-strip sforum-moderation__filter-strip--secondary" :aria-label="t('admin.moderation.filterType')">
          <button
            v-for="item in typeFilters"
            :key="item.value"
            type="button"
            class="sforum-moderation__filter-button"
            :class="{ 'is-active': typeFilter === item.value }"
            :aria-pressed="typeFilter === item.value"
            @click="selectType(item.value)"
          >
            {{ item.label }}
          </button>
        </div>
        <p class="sforum-moderation__filter-note">{{ pageRangeLabel }}</p>

        <SFAlert v-if="queueErrorMessage" variant="danger" :title="queueErrorMessage" class="sforum-moderation__alert" />

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
              :active="isItemActive(item)"
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

        <SFContentColumnFooter />
      </section>

      <section
        v-else
        class="sforum-home__main sforum-moderation__main sforum-moderation__main--review"
        :aria-label="t('moderation.workbench.reviewBreadcrumb')"
      >
        <div class="sforum-moderation__mobile-nav">
          <SFHomeNavigation
            mobile-only
            navigation-mode="route"
            :categories="categories"
            :total-topics="categoryTopicTotal"
            :pending="categoriesPending"
            :can-create-topic="canCreateTopic"
          />
        </div>

        <header class="sforum-moderation__head sforum-moderation__head--review">
          <div class="sforum-moderation__head-copy">
            <button type="button" class="sforum-moderation__text-button" @click="returnToQueue">
              <UIcon name="i-lucide-arrow-left" class="size-4" aria-hidden="true" />
              {{ t('moderation.workbench.backToQueue') }}
            </button>
            <p v-if="progressLabel">{{ progressLabel }} · {{ pageRangeLabel }}</p>
          </div>
          <div class="sforum-moderation__head-actions">
            <button
              type="button"
              class="sforum-moderation-icon-button sforum-moderation__desktop-hidden"
              :aria-label="t('moderation.workbench.openDecisionDrawer')"
              @click="mobileInfoOpen = true"
            >
              <UIcon name="i-lucide-panel-right" class="size-[18px]" aria-hidden="true" />
            </button>
            <button
              type="button"
              class="sforum-moderation-icon-button sforum-moderation__desktop-hidden sforum-moderation__menu-button"
              :aria-label="t('moderation.workbench.openQueueDrawer')"
              @click="mobileMenuOpen = true"
            >
              <UIcon name="i-lucide-menu" class="size-[18px]" aria-hidden="true" />
            </button>
          </div>
        </header>

        <ModerationReviewReader
          :context="reviewContext"
          note-id="moderation-review-note"
          :loading="contextPending"
          :tab="tab"
          @back="returnToQueue"
        />
      </section>

      <ModerationQueueRail
        v-if="!reviewMode"
        :overview-count="overviewCount"
        :overview-count-label="overviewCountLabel"
        :header-title="headerTitle"
        :type-filter-label="typeFilterLabel"
        :page-range-label="pageRangeLabel"
        :loaded-total="list.total"
        :show-processed-today="tab === 'history' && countsAvailable"
        :processed-today="safeCounts.processedToday"
      />

      <ModerationDecisionRail
        v-else
        v-model:note="activeNote"
        :context="reviewContext"
        note-id="moderation-review-note-desktop"
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

    <button
      v-if="mobileMenuOpen || mobileInfoOpen"
      type="button"
      class="sforum-mobile-drawer__backdrop"
      :aria-label="t('common.close')"
      @click="closeMobileDrawers"
    />

    <aside v-if="mobileMenuOpen" class="sforum-mobile-drawer sforum-mobile-drawer--left">
      <header class="sforum-mobile-drawer__head">
        <strong>{{ t('home.sidebar.drawerTitle') }}</strong>
        <button type="button" :aria-label="t('common.close')" @click="closeMobileDrawers">
          <UIcon name="i-lucide-x" class="size-5" aria-hidden="true" />
        </button>
      </header>
      <SFHomeNavigation
        desktop-only
        navigation-mode="route"
        :categories="categories"
        :total-topics="categoryTopicTotal"
        :pending="categoriesPending"
        :can-create-topic="canCreateTopic"
        :show-categories="false"
      >
        <template #after-navigation>
          <ModerationWorkbenchNav
            :source-tabs="sourceTabs"
            :type-filters="typeFilters"
            :tab="tab"
            :type-filter="typeFilter"
            :review-mode="reviewMode"
            :items="items"
            :active-key="reviewKey"
            @select-tab="selectTab"
            @select-type="selectType"
            @open-item="openItem"
            @back="returnToQueue"
            @navigate="closeMobileDrawers"
          />
        </template>
      </SFHomeNavigation>
    </aside>

    <aside v-if="mobileInfoOpen" class="sforum-mobile-drawer sforum-mobile-drawer--right">
      <header class="sforum-mobile-drawer__head">
        <strong>{{ rightDrawerTitle }}</strong>
        <button type="button" :aria-label="t('common.close')" @click="closeMobileDrawers">
          <UIcon name="i-lucide-x" class="size-5" aria-hidden="true" />
        </button>
      </header>

      <ModerationQueueRail
        v-if="!reviewMode"
        drawer
        :overview-count="overviewCount"
        :overview-count-label="overviewCountLabel"
        :header-title="headerTitle"
        :type-filter-label="typeFilterLabel"
        :page-range-label="pageRangeLabel"
        :loaded-total="list.total"
        :show-processed-today="tab === 'history' && countsAvailable"
        :processed-today="safeCounts.processedToday"
      />

      <ModerationDecisionRail
        v-else
        v-model:note="activeNote"
        drawer
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
    </aside>
  </main>
</template>
