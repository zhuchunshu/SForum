<script setup lang="ts">
/**
 * 宿主 body 岛：moderation.review。
 * 左右栏对齐首页 + 通知页公共三栏 chrome；队列/审阅业务逻辑不变。
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

const { data: categoryGroups, pending: categoriesPending } = await useAsyncData(
  'moderation-workbench-categories',
  () => forumApi.listCategoryGroups(),
  { default: () => [] }
)

const countsAvailable = computed(() => !countsError.value)
const categories = computed(() => categoryGroups.value.flatMap(group => group.categories || []))
const categoryTopicTotal = computed(() => categories.value.reduce((sum, category) => sum + (category.topicCount || 0), 0))
const canCreateTopic = computed(() => can(FORUM_PERMISSIONS.topicCreate))
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
const typeFilterLabel = computed(() => typeFilters.value.find(item => item.value === typeFilter.value)?.label || t('admin.moderation.typeAll'))
// 右栏大数字：跟随当前来源 tab，与通知页未读总数同级
const overviewCount = computed(() => {
  if (!countsAvailable.value) return null
  if (tab.value === 'reports') return counts.value.openReports
  if (tab.value === 'history') return counts.value.processedToday
  return counts.value.pendingContent
})
const overviewCountLabel = computed(() => {
  if (tab.value === 'reports') return t('moderation.workbench.reports')
  if (tab.value === 'history') return t('moderation.workbench.processedToday')
  return t('moderation.workbench.pending')
})
const rightRailAria = computed(() => reviewMode.value
  ? t('moderation.workbench.decisionRail')
  : t('moderation.workbench.queueOverview'))
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

function compactItemIcon(item: QueueRecord) {
  if ('reasonCode' in item) return 'i-lucide-flag'
  return item.targetType === 'topic' ? 'i-lucide-file-text' : 'i-lucide-message-square'
}

function compactItemTitle(item: QueueRecord) {
  if ('title' in item && item.title) return item.title
  return `${t(`admin.moderation.type.${item.targetType}`)} #${item.targetId}`
}

function compactItemMeta(item: QueueRecord) {
  if ('action' in item) return historySummary(item as ModerationDecision)
  if ('reasonCode' in item) return t(`moderation.reason.${(item as ModerationReportItem).reasonCode}`)
  return (item as ModerationPendingItem).triggers.map(trigger => t(`moderation.trigger.${trigger}`)).join(' / ')
}

function isItemActive(item: QueueRecord) {
  return selectionKey(selectionFromQueueItem(tab.value, item)) === reviewKey.value
}
</script>

<template>
  <main
    class="sforum-moderation"
    data-sforum-island-body="forum.component.moderation_review"
    data-layout="fullwidth-3col"
    :class="{ 'sforum-moderation--review': reviewMode }"
  >
    <div class="sforum-moderation__layout">
      <div class="sforum-moderation__sidebar sforum-home__sidebar">
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
            <button
              v-if="reviewMode"
              type="button"
              class="sforum-moderation__back-nav"
              @click="returnToQueue"
            >
              <UIcon name="i-lucide-arrow-left" class="size-4" aria-hidden="true" />
              {{ t('moderation.workbench.backToQueue') }}
            </button>

            <nav class="sforum-moderation__type-nav" :aria-label="t('moderation.workbench.sources')">
              <div class="sf-home-navigation__label">{{ t('moderation.workbench.sources') }}</div>
              <button
                v-for="item in sourceTabs"
                :key="item.value"
                type="button"
                class="sf-home-navigation__link"
                :class="{ 'is-active': tab === item.value }"
                :aria-pressed="tab === item.value"
                @click="selectTab(item.value)"
              >
                <span class="sf-home-navigation__link-main">
                  <UIcon :name="item.icon" class="size-[18px]" aria-hidden="true" />
                  {{ item.label }}
                </span>
                <span v-if="item.count !== null" class="sf-home-navigation__count">{{ item.count }}</span>
              </button>
            </nav>

            <nav class="sforum-moderation__type-nav" :aria-label="t('admin.moderation.filterType')">
              <div class="sf-home-navigation__label">{{ t('admin.moderation.filterType') }}</div>
              <button
                v-for="item in typeFilters"
                :key="item.value"
                type="button"
                class="sf-home-navigation__link"
                :class="{ 'is-active': typeFilter === item.value }"
                :aria-pressed="typeFilter === item.value"
                @click="selectType(item.value)"
              >
                <span class="sf-home-navigation__link-main">
                  <UIcon :name="item.icon" class="size-[18px]" aria-hidden="true" />
                  {{ item.label }}
                </span>
              </button>
              <p class="sforum-moderation__filter-hint">{{ t('moderation.workbench.permissionHint') }}</p>
            </nav>

            <template v-if="reviewMode">
              <div class="sf-home-navigation__label">{{ t('moderation.workbench.currentQueue') }}</div>
              <div class="sforum-moderation-compact-list">
                <button
                  v-for="item in items"
                  :key="queueItemKey(tab, item)"
                  type="button"
                  class="sforum-moderation-compact-item"
                  :class="{ 'is-active': isItemActive(item) }"
                  @click="openItem(item)"
                >
                  <UIcon :name="compactItemIcon(item)" class="size-4" aria-hidden="true" />
                  <span>
                    <strong>{{ compactItemTitle(item) }}</strong>
                    <small>{{ compactItemMeta(item) }}</small>
                  </span>
                </button>
              </div>
            </template>
          </template>
        </SFHomeNavigation>
      </div>

      <section
        v-if="!reviewMode"
        class="sforum-moderation__main"
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

        <header class="sforum-moderation__head">
          <div class="sforum-moderation__head-copy">
            <h1 id="moderation-page-title">{{ headerTitle }}</h1>
            <p>{{ headerDescription }}</p>
          </div>
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
        </header>

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
        class="sforum-moderation__main sforum-moderation__main--review"
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

      <!-- 队列模式右栏：始终展示，含 history，避免三栏跳动 -->
      <aside
        v-if="!reviewMode"
        class="sforum-moderation__right"
        :aria-label="rightRailAria"
      >
        <section class="sforum-moderation__rail-section">
          <div class="sforum-moderation__rail-head">
            <h2>{{ t('moderation.workbench.queueOverview') }}</h2>
            <span>{{ t('moderation.workbench.overviewAuthority') }}</span>
          </div>
          <div v-if="overviewCount !== null" class="sforum-moderation__overview-summary">
            <strong>{{ overviewCount }}</strong>
            <span>{{ overviewCountLabel }}</span>
          </div>
          <p v-else class="sforum-moderation__rail-help" role="alert">
            {{ t('moderation.workbench.countsFailed') }}
          </p>
          <p v-if="overviewCount !== null" class="sforum-moderation__rail-help">
            {{ t('moderation.workbench.overviewSource') }}
          </p>
        </section>

        <section class="sforum-moderation__rail-section">
          <div class="sforum-moderation__rail-head">
            <h2>{{ t('moderation.workbench.pageStatsTitle') }}</h2>
            <span>{{ t('moderation.workbench.loadedOnly') }}</span>
          </div>
          <dl class="sforum-moderation__loaded-stats">
            <div>
              <dt>{{ t('moderation.workbench.sources') }}</dt>
              <dd>{{ headerTitle }}</dd>
            </div>
            <div>
              <dt>{{ t('admin.moderation.filterType') }}</dt>
              <dd>{{ typeFilterLabel }}</dd>
            </div>
            <div>
              <dt>{{ t('moderation.workbench.pageStatsTitle') }}</dt>
              <dd>{{ pageRangeLabel }}</dd>
            </div>
            <div>
              <dt>{{ t('moderation.workbench.loadedTotal') }}</dt>
              <dd>{{ list.total }}</dd>
            </div>
          </dl>
        </section>

        <section class="sforum-moderation__rail-section">
          <div class="sforum-moderation__rail-head">
            <h2>{{ t('moderation.workbench.workflowTitle') }}</h2>
          </div>
          <p class="sforum-moderation__rail-help">{{ t('moderation.workbench.workflowDescription') }}</p>
        </section>

        <section class="sforum-moderation__rail-section">
          <div class="sforum-moderation__rail-head">
            <h2>{{ t('moderation.workbench.stateRestoreTitle') }}</h2>
          </div>
          <p class="sforum-moderation__rail-help">{{ t('moderation.workbench.stateRestoreDescription') }}</p>
        </section>
      </aside>

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
          <nav class="sforum-moderation__type-nav" :aria-label="t('moderation.workbench.sources')">
            <div class="sf-home-navigation__label">{{ t('moderation.workbench.sources') }}</div>
            <button
              v-for="item in sourceTabs"
              :key="item.value"
              type="button"
              class="sf-home-navigation__link"
              :class="{ 'is-active': tab === item.value }"
              @click="selectTab(item.value)"
            >
              <span class="sf-home-navigation__link-main">
                <UIcon :name="item.icon" class="size-[18px]" aria-hidden="true" />
                {{ item.label }}
              </span>
              <span v-if="item.count !== null" class="sf-home-navigation__count">{{ item.count }}</span>
            </button>
          </nav>
          <nav class="sforum-moderation__type-nav" :aria-label="t('admin.moderation.filterType')">
            <div class="sf-home-navigation__label">{{ t('admin.moderation.filterType') }}</div>
            <button
              v-for="item in typeFilters"
              :key="item.value"
              type="button"
              class="sf-home-navigation__link"
              :class="{ 'is-active': typeFilter === item.value }"
              @click="selectType(item.value)"
            >
              <span class="sf-home-navigation__link-main">
                <UIcon :name="item.icon" class="size-[18px]" aria-hidden="true" />
                {{ item.label }}
              </span>
            </button>
          </nav>
          <template v-if="reviewMode && items.length">
            <div class="sf-home-navigation__label">{{ t('moderation.workbench.currentQueue') }}</div>
            <div class="sforum-moderation-compact-list">
              <button
                v-for="item in items"
                :key="queueItemKey(tab, item)"
                type="button"
                class="sforum-moderation-compact-item"
                :class="{ 'is-active': isItemActive(item) }"
                @click="openItem(item)"
              >
                <UIcon :name="compactItemIcon(item)" class="size-4" aria-hidden="true" />
                <span>
                  <strong>{{ compactItemTitle(item) }}</strong>
                  <small>{{ compactItemMeta(item) }}</small>
                </span>
              </button>
            </div>
          </template>
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

      <div v-if="!reviewMode" class="sforum-moderation__right sforum-moderation__right--drawer" :aria-label="rightRailAria">
        <section class="sforum-moderation__rail-section">
          <div class="sforum-moderation__rail-head">
            <h2>{{ t('moderation.workbench.queueOverview') }}</h2>
            <span>{{ t('moderation.workbench.overviewAuthority') }}</span>
          </div>
          <div v-if="overviewCount !== null" class="sforum-moderation__overview-summary">
            <strong>{{ overviewCount }}</strong>
            <span>{{ overviewCountLabel }}</span>
          </div>
          <p v-else class="sforum-moderation__rail-help" role="alert">
            {{ t('moderation.workbench.countsFailed') }}
          </p>
        </section>
        <section class="sforum-moderation__rail-section">
          <div class="sforum-moderation__rail-head">
            <h2>{{ t('moderation.workbench.pageStatsTitle') }}</h2>
          </div>
          <dl class="sforum-moderation__loaded-stats">
            <div>
              <dt>{{ t('moderation.workbench.pageStatsTitle') }}</dt>
              <dd>{{ pageRangeLabel }}</dd>
            </div>
            <div>
              <dt>{{ t('moderation.workbench.loadedTotal') }}</dt>
              <dd>{{ list.total }}</dd>
            </div>
          </dl>
        </section>
        <section class="sforum-moderation__rail-section">
          <div class="sforum-moderation__rail-head">
            <h2>{{ t('moderation.workbench.workflowTitle') }}</h2>
          </div>
          <p class="sforum-moderation__rail-help">{{ t('moderation.workbench.workflowDescription') }}</p>
        </section>
      </div>

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

<style src="~/assets/css/sforum-moderation.css"></style>
