<script setup lang="ts">
/**
 * 审核工作台左栏 after-nav：来源 / 类型筛选 + 审阅态紧凑队列。
 * 样式完全复用 sf-home-navigation 链接语言，与首页 / 设置 after-nav 同 token。
 */
import type { ModerationDecision, ModerationPendingItem, ModerationReportItem } from '~/composables/moderation/useModerationApi'
import type { ModerationWorkbenchTab, ModerationWorkbenchTypeFilter } from '~/utils/moderation/moderationWorkbench'
import { queueItemKey, selectionFromQueueItem, selectionKey } from '~/utils/moderation/moderationWorkbench'

type QueueRecord = ModerationPendingItem | ModerationReportItem | ModerationDecision

const props = defineProps<{
  sourceTabs: Array<{ value: ModerationWorkbenchTab; icon: string; label: string; count: number | null }>
  typeFilters: Array<{ value: ModerationWorkbenchTypeFilter; icon: string; label: string }>
  tab: ModerationWorkbenchTab
  typeFilter: ModerationWorkbenchTypeFilter
  reviewMode?: boolean
  items?: QueueRecord[]
  /** selectionKey(...)，与 URL 审阅态一致 */
  activeKey?: string
}>()

const emit = defineEmits<{
  'select-tab': [value: ModerationWorkbenchTab]
  'select-type': [value: ModerationWorkbenchTypeFilter]
  'open-item': [item: QueueRecord]
  back: []
  navigate: []
}>()

const { t } = useI18n()

function isItemActive(item: QueueRecord) {
  if (!props.activeKey) return false
  return selectionKey(selectionFromQueueItem(props.tab, item)) === props.activeKey
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
  if ('action' in item) {
    const decision = item as ModerationDecision
    return `${t(`moderation.action.${decision.action}`)} / ${decision.reviewerName || t('moderation.workbench.unknownReviewer')}`
  }
  if ('reasonCode' in item) return t(`moderation.reason.${(item as ModerationReportItem).reasonCode}`)
  return (item as ModerationPendingItem).triggers.map(trigger => t(`moderation.trigger.${trigger}`)).join(' / ')
}

function onSelectTab(value: ModerationWorkbenchTab) {
  emit('select-tab', value)
  emit('navigate')
}

function onSelectType(value: ModerationWorkbenchTypeFilter) {
  emit('select-type', value)
  emit('navigate')
}

function onOpenItem(item: QueueRecord) {
  emit('open-item', item)
  emit('navigate')
}

function onBack() {
  emit('back')
  emit('navigate')
}
</script>

<template>
  <div class="sforum-moderation-workbench-nav">
    <button
      v-if="reviewMode"
      type="button"
      class="sforum-moderation__back-nav"
      @click="onBack"
    >
      <UIcon name="i-lucide-arrow-left" class="size-4" aria-hidden="true" />
      {{ t('moderation.workbench.backToQueue') }}
    </button>

    <nav :aria-label="t('moderation.workbench.sources')">
      <div class="sf-home-navigation__label">{{ t('moderation.workbench.sources') }}</div>
      <button
        v-for="item in sourceTabs"
        :key="item.value"
        type="button"
        class="sf-home-navigation__link"
        :class="{ 'is-active': tab === item.value }"
        :aria-pressed="tab === item.value"
        @click="onSelectTab(item.value)"
      >
        <span class="sf-home-navigation__link-main">
          <UIcon :name="item.icon" class="size-[18px]" aria-hidden="true" />
          {{ item.label }}
        </span>
        <span v-if="item.count !== null" class="sf-home-navigation__count">{{ item.count }}</span>
      </button>
    </nav>

    <nav :aria-label="t('admin.moderation.filterType')">
      <div class="sf-home-navigation__label">{{ t('admin.moderation.filterType') }}</div>
      <button
        v-for="item in typeFilters"
        :key="item.value"
        type="button"
        class="sf-home-navigation__link"
        :class="{ 'is-active': typeFilter === item.value }"
        :aria-pressed="typeFilter === item.value"
        @click="onSelectType(item.value)"
      >
        <span class="sf-home-navigation__link-main">
          <UIcon :name="item.icon" class="size-[18px]" aria-hidden="true" />
          {{ item.label }}
        </span>
      </button>
      <p class="sforum-moderation__filter-hint">{{ t('moderation.workbench.permissionHint') }}</p>
    </nav>

    <template v-if="reviewMode && items?.length">
      <div class="sf-home-navigation__label">{{ t('moderation.workbench.currentQueue') }}</div>
      <div class="sforum-moderation-compact-list">
        <button
          v-for="item in items"
          :key="queueItemKey(tab, item)"
          type="button"
          class="sf-home-navigation__link sforum-moderation-compact-item"
          :class="{ 'is-active': isItemActive(item) }"
          @click="onOpenItem(item)"
        >
          <UIcon :name="compactItemIcon(item)" class="size-4 shrink-0" aria-hidden="true" />
          <span class="sforum-moderation-compact-item__text">
            <strong>{{ compactItemTitle(item) }}</strong>
            <small>{{ compactItemMeta(item) }}</small>
          </span>
        </button>
      </div>
    </template>
  </div>
</template>
