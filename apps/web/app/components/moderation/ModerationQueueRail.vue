<script setup lang="ts">
/**
 * 审核队列右栏：复用首页 sf-home-right-rail 卡片壳，内容为工作台 KPI / 说明。
 */
defineProps<{
  overviewCount: number | null
  overviewCountLabel: string
  headerTitle: string
  typeFilterLabel: string
  pageRangeLabel: string
  loadedTotal: number
  showProcessedToday?: boolean
  processedToday?: number
  /** 抽屉内渲染：去掉 sticky / 边框 */
  drawer?: boolean
}>()

const { t } = useI18n()
</script>

<template>
  <aside
    class="sforum-home__right"
    :class="{ 'sforum-moderation__right--drawer': drawer }"
    :aria-label="t('moderation.workbench.queueOverview')"
  >
    <div class="sf-home-right-rail">
      <section class="sf-home-right-rail__card">
        <header class="sf-home-right-rail__head">
          <h3 class="sf-home-right-rail__title">{{ t('moderation.workbench.queueOverview') }}</h3>
          <span class="sf-home-right-rail__meta">{{ t('moderation.workbench.overviewAuthority') }}</span>
        </header>
        <div v-if="overviewCount !== null" class="sforum-moderation__overview-summary">
          <strong>{{ overviewCount }}</strong>
          <span>{{ overviewCountLabel }}</span>
        </div>
        <p v-else class="sf-home-right-rail__empty" role="alert">
          {{ t('moderation.workbench.countsFailed') }}
        </p>
        <p v-if="overviewCount !== null" class="sforum-moderation__rail-help">
          {{ t('moderation.workbench.overviewSource') }}
        </p>
      </section>

      <section class="sf-home-right-rail__card">
        <header class="sf-home-right-rail__head">
          <h3 class="sf-home-right-rail__title">{{ t('moderation.workbench.pageStatsTitle') }}</h3>
          <span class="sf-home-right-rail__meta">{{ t('moderation.workbench.loadedOnly') }}</span>
        </header>
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
            <dd>{{ loadedTotal }}</dd>
          </div>
          <div v-if="showProcessedToday">
            <dt>{{ t('moderation.workbench.processedToday') }}</dt>
            <dd>{{ processedToday }}</dd>
          </div>
        </dl>
      </section>

      <section class="sf-home-right-rail__card">
        <header class="sf-home-right-rail__head">
          <h3 class="sf-home-right-rail__title">{{ t('moderation.workbench.workflowTitle') }}</h3>
        </header>
        <p class="sforum-moderation__rail-help sforum-moderation__rail-help--card">
          {{ t('moderation.workbench.workflowDescription') }}
        </p>
      </section>

      <section v-if="!drawer" class="sf-home-right-rail__card">
        <header class="sf-home-right-rail__head">
          <h3 class="sf-home-right-rail__title">{{ t('moderation.workbench.stateRestoreTitle') }}</h3>
        </header>
        <p class="sforum-moderation__rail-help sforum-moderation__rail-help--card">
          {{ t('moderation.workbench.stateRestoreDescription') }}
        </p>
      </section>
    </div>
  </aside>
</template>
