<script setup lang="ts">
import { apiErrorMessage } from '~/composables/useApiClient'
import { useAdminPage } from '~/composables/useAdminPage'
import SFAdminOverviewTrendTrio from '~/components/admin/SFAdminOverviewTrendTrio.vue'
import {
  formatOverviewBytes,
  formatOverviewCount,
  formatOverviewDate,
  formatOverviewUptime,
  overviewActionTone,
  overviewPercent,
  type AdminOverview,
  type AdminOverviewAction,
  type AdminOverviewExtensionWidget
} from '~/utils/adminOverview'

definePageMeta({
  middleware: 'admin',
  layout: 'admin'
})

// 明确声明组件名用于 KeepAlive 匹配
defineOptions({
  name: 'AdminIndex'
})

const { t } = useI18n()
const { request } = useApiClient()
const adminRoutes = useAdminRoutes()
const adminPage = useAdminPage('/')

const {
  data: overview,
  pending,
  error,
  refresh
} = await useAsyncData<AdminOverview | null>(
  'admin-overview',
  () => request<AdminOverview>('/admin/overview'),
  { default: () => null }
)

const kpiCards = computed(() => {
  const data = overview.value
  if (!data) return []
  const actionCount = data.actions.reduce((total, action) => total + action.count, 0)
  return [
    {
      label: t('admin.home.kpi.memory.label'),
      value: formatOverviewBytes(data.runtime.memoryBytes),
      meta: data.runtime.familyMemoryBytes != null
        ? t('admin.home.kpi.memory.metaWithFamily', {
            heap: formatOverviewBytes(data.runtime.heapAllocBytes),
            family: formatOverviewBytes(data.runtime.familyMemoryBytes),
            plugins: formatOverviewCount(data.runtime.pluginChildCount ?? 0)
          })
        : t('admin.home.kpi.memory.meta', {
            heap: formatOverviewBytes(data.runtime.heapAllocBytes)
          }),
      icon: 'i-lucide-memory-stick',
      tone: 'text-[var(--sf-accent)] dark:text-[var(--sf-accent-dark)]'
    },
    {
      label: t('admin.home.kpi.topics.label'),
      value: formatOverviewCount(data.community.postCount),
      meta: t('admin.home.kpi.topics.meta', {
        topics: formatOverviewCount(data.community.topicCount),
        comments: formatOverviewCount(data.community.commentCount)
      }),
      icon: 'i-lucide-message-square-text',
      tone: 'text-blue-600 dark:text-blue-400'
    },
    {
      label: t('admin.home.kpi.users.label'),
      value: formatOverviewCount(data.community.userCount),
      meta: t('admin.home.kpi.users.meta', { active: formatOverviewCount(data.community.activeUserCount) }),
      icon: 'i-lucide-users-round',
      tone: 'text-green-600 dark:text-green-400'
    },
    {
      label: t('admin.home.kpi.actions.label'),
      value: formatOverviewCount(actionCount),
      meta: t('admin.home.kpi.actions.meta', { reports: formatOverviewCount(data.moderation.openCount + data.moderation.reviewingCount) }),
      icon: 'i-lucide-list-checks',
      tone: actionCount > 0 ? 'text-amber-600 dark:text-amber-400' : 'text-emerald-600 dark:text-emerald-400'
    }
  ]
})

const runtimeRows = computed(() => {
  const runtime = overview.value?.runtime
  if (!runtime) return []
  const worker = runtime.worker
  const workerValue = !worker
    ? t('admin.home.runtime.workerUnavailable')
    : worker.status === 'ok'
      ? t('admin.home.runtime.workerOk', { age: formatOverviewCount(worker.ageSeconds ?? 0) })
      : worker.status === 'stale'
        ? t('admin.home.runtime.workerStale', { age: formatOverviewCount(worker.ageSeconds ?? 0) })
        : t('admin.home.runtime.workerUnknown')
  const lag = runtime.queueLag
  const lagValue = !lag
    ? t('admin.home.runtime.queueLagUnavailable')
    : t('admin.home.runtime.queueLagValue', {
        waiting: formatOverviewCount(lag.waiting),
        running: formatOverviewCount(lag.running),
        failed: formatOverviewCount(lag.failed)
      })
  return [
    {
      label: t('admin.home.runtime.uptime'),
      value: formatOverviewUptime(runtime.uptimeSeconds),
      icon: 'i-lucide-clock-3'
    },
    {
      label: t('admin.home.runtime.worker'),
      value: workerValue,
      icon: worker?.stale === false ? 'i-lucide-heart-pulse' : 'i-lucide-heart-off',
      tone: worker?.status === 'ok'
        ? 'text-emerald-600 dark:text-emerald-400'
        : worker?.status === 'stale'
          ? 'text-amber-600 dark:text-amber-400'
          : 'text-slate-500'
    },
    {
      label: t('admin.home.runtime.queueLag'),
      value: lagValue,
      icon: 'i-lucide-layers'
    },
    {
      label: t('admin.home.runtime.goroutines'),
      value: formatOverviewCount(runtime.goroutineCount),
      icon: 'i-lucide-git-branch'
    },
    {
      label: t('admin.home.runtime.gc'),
      value: formatOverviewCount(runtime.gcCount),
      icon: 'i-lucide-repeat-2'
    },
    {
      label: t('admin.home.runtime.goSys'),
      value: formatOverviewBytes(runtime.sysBytes ?? 0),
      icon: 'i-lucide-cpu'
    },
    {
      label: t('admin.home.runtime.database'),
      value: `${formatOverviewCount(runtime.database.acquiredConnections)} / ${formatOverviewCount(runtime.database.maxConnections)}`,
      icon: 'i-lucide-database'
    }
  ]
})

const healthRows = computed(() => {
  const data = overview.value
  if (!data) return []
  return [
    {
      label: t('admin.home.health.activeUsers'),
      value: `${overviewPercent(data.community.activeUserCount, data.community.userCount)}%`,
      sub: t('admin.home.health.activeUsersSub', {
        active: formatOverviewCount(data.community.activeUserCount),
        total: formatOverviewCount(data.community.userCount)
      }),
      percent: overviewPercent(data.community.activeUserCount, data.community.userCount)
    },
    {
      label: t('admin.home.health.activeTopics'),
      value: `${overviewPercent(data.community.activeTopicCount + data.community.lockedTopicCount, data.community.topicCount)}%`,
      sub: t('admin.home.health.activeTopicsSub', {
        active: formatOverviewCount(data.community.activeTopicCount + data.community.lockedTopicCount),
        total: formatOverviewCount(data.community.topicCount)
      }),
      percent: overviewPercent(data.community.activeTopicCount + data.community.lockedTopicCount, data.community.topicCount)
    },
    {
      label: t('admin.home.health.attachments'),
      value: `${overviewPercent(data.attachments.activeCount, data.attachments.totalCount)}%`,
      sub: t('admin.home.health.attachmentsSub', {
        orphan: formatOverviewCount(data.attachments.orphanCount),
        size: formatOverviewBytes(data.attachments.totalBytes)
      }),
      percent: overviewPercent(data.attachments.activeCount, data.attachments.totalCount)
    }
  ]
})

const quickLinks = computed(() => {
  const data = overview.value
  return [
    {
      title: t('admin.home.quick.users.title'),
      description: t('admin.home.quick.users.description', { count: formatOverviewCount(data?.community.userCount || 0) }),
      icon: 'i-lucide-contact',
      to: adminRoutes.path('/users')
    },
    {
      title: t('admin.home.quick.forum.title'),
      description: t('admin.home.quick.forum.description', { count: formatOverviewCount(data?.community.categoryCount || 0) }),
      icon: 'i-lucide-folder-tree',
      to: adminRoutes.path('/forum/categories')
    },
    {
      title: t('admin.home.quick.moderation.title'),
      description: t('admin.home.quick.moderation.description', { count: formatOverviewCount((data?.moderation.openCount || 0) + (data?.moderation.reviewingCount || 0)) }),
      icon: 'i-lucide-shield-alert',
      to: adminRoutes.path('/moderation')
    },
    {
      title: t('admin.home.quick.extensions.title'),
      description: t('admin.home.quick.extensions.description', { count: formatOverviewCount(data?.extensions.totalCount || 0) }),
      icon: 'i-lucide-blocks',
      to: adminRoutes.path('/extensions')
    }
  ]
})

useSeoMeta({
  title: t('admin.home.metaTitle')
})

function actionLabel(action: AdminOverviewAction) {
  return t(`admin.home.actions.${action.key}.title`)
}

function actionDescription(action: AdminOverviewAction) {
  return t(`admin.home.actions.${action.key}.description`, { count: formatOverviewCount(action.count) })
}

function actionRoute(action: AdminOverviewAction) {
  return adminRoutes.path(action.route)
}

function actionColor(action: AdminOverviewAction) {
  const tone = overviewActionTone(action.severity)
  return tone === 'danger' ? 'error' : tone
}

const { locale } = useI18n()

const extensionWidgets = computed(() => overview.value?.extensionWidgets || [])

function extensionWidgetLabel(widget: AdminOverviewExtensionWidget) {
  const labels = widget.label || {}
  return labels[String(locale.value)] || labels['zh-CN'] || labels['en-US'] || Object.values(labels)[0] || widget.id
}

function extensionWidgetRoute(widget: AdminOverviewExtensionWidget) {
  return adminRoutes.path(widget.route)
}

function extensionWidgetColor(widget: AdminOverviewExtensionWidget) {
  const tone = overviewActionTone(widget.severity)
  return tone === 'danger' ? 'error' : tone
}
</script>

<template>
  <div class="mb-4 flex flex-col gap-1">
    <h2 class="text-xl font-bold flex items-center gap-2 text-slate-900 dark:text-zinc-100">
      <UIcon :name="adminPage.icon" class="size-5 text-[var(--sf-accent)] dark:text-[var(--sf-accent-dark)]" />
      {{ t('admin.home.title') }}
    </h2>
    <p class="text-sm text-slate-500 dark:text-zinc-400">
      {{ t('admin.home.intro') }}
    </p>
  </div>

  <UDashboardToolbar class="border border-slate-200 dark:border-zinc-800 bg-white dark:bg-zinc-900 rounded-lg px-4 py-2.5 mb-6 text-slate-500 dark:text-zinc-400">
    <template #left>
      <div class="flex min-w-0 items-center gap-2 text-sm">
        <UIcon name="i-lucide-activity" class="size-4" />
        <span class="truncate">
          {{ overview ? t('admin.home.toolbar.generatedAt', { time: formatOverviewDate(overview.generatedAt), days: overview.windowDays }) : t('admin.home.toolbar.loading') }}
        </span>
      </div>
    </template>
    <template #right>
      <div class="flex items-center gap-3">
        <UButton
          icon="i-lucide-rotate-cw"
          color="neutral"
          variant="subtle"
          class="shrink-0 whitespace-nowrap"
          :loading="pending"
          @click="refresh()"
        >
          {{ t('admin.home.refresh') }}
        </UButton>
        <UBadge color="neutral" variant="soft" class="border border-slate-200 dark:border-zinc-800 font-mono">
          {{ adminRoutes.prefix }}
        </UBadge>
      </div>
    </template>
  </UDashboardToolbar>

  <UAlert
    v-if="error"
    color="error"
    icon="i-lucide-triangle-alert"
    variant="subtle"
    :title="apiErrorMessage(error) || t('admin.home.loadFailed')"
    class="mb-6"
  />

  <div v-if="pending && !overview" class="grid gap-5 lg:grid-cols-4">
    <UCard v-for="index in 4" :key="index" class="elegant-card border-slate-200 dark:border-zinc-800 bg-white dark:bg-zinc-900">
      <SFSkeleton :lines="3" />
    </UCard>
  </div>

  <div v-else-if="overview" class="flex flex-col gap-6">
    <div class="grid gap-5 md:grid-cols-2 xl:grid-cols-4">
      <UCard v-for="card in kpiCards" :key="card.label" class="elegant-card border-slate-200 dark:border-zinc-800 bg-white dark:bg-zinc-900 text-slate-900 dark:text-zinc-100">
        <div class="flex items-start justify-between gap-4">
          <div class="min-w-0">
            <p class="text-xs font-semibold text-slate-500 dark:text-zinc-400 uppercase">
              {{ card.label }}
            </p>
            <p class="mt-2.5 truncate text-3xl font-black text-slate-900 dark:text-white">
              {{ card.value }}
            </p>
            <p class="mt-2 truncate text-xs text-slate-500 dark:text-zinc-400">
              {{ card.meta }}
            </p>
          </div>
          <span class="icon-glass-box shrink-0" :class="card.tone">
            <UIcon :name="card.icon" class="size-5 z-10" />
          </span>
        </div>
      </UCard>
    </div>

    <!-- 社区趋势整行：横向三系列，不与旁侧卡抢高度 -->
    <UCard class="elegant-card border-slate-200 dark:border-zinc-800 bg-white dark:bg-zinc-900 text-slate-900 dark:text-zinc-100">
      <template #header>
        <div class="flex items-center justify-between gap-3">
          <div>
            <h3 class="text-base font-bold text-slate-900 dark:text-white">
              {{ t('admin.home.trend.title') }}
            </h3>
            <p class="mt-1 text-xs text-slate-500 dark:text-zinc-400">
              {{ t('admin.home.trend.description', { days: overview.windowDays }) }}
            </p>
          </div>
          <UIcon name="i-lucide-chart-no-axes-combined" class="size-5 text-slate-400 dark:text-zinc-500" />
        </div>
      </template>

      <SFAdminOverviewTrendTrio
        :days="overview.trends.days"
        :window-days="overview.windowDays"
      />
    </UCard>

    <!-- 运行状态与健康度等同排，避免再占一整行空壳 -->
    <div class="grid gap-6 xl:grid-cols-2 2xl:grid-cols-4">
      <UCard class="elegant-card border-slate-200 dark:border-zinc-800 bg-white dark:bg-zinc-900 text-slate-900 dark:text-zinc-100">
        <template #header>
          <div class="flex items-center justify-between gap-3">
            <div>
              <h3 class="text-base font-bold text-slate-900 dark:text-white">
                {{ t('admin.home.runtime.title') }}
              </h3>
              <p class="mt-1 text-xs text-slate-500 dark:text-zinc-400">
                {{ t('admin.home.runtime.description') }}
              </p>
            </div>
            <UIcon name="i-lucide-server" class="size-5 shrink-0 text-slate-400 dark:text-zinc-500" />
          </div>
        </template>

        <div class="space-y-2.5">
          <div v-for="row in runtimeRows" :key="row.label" class="flex items-center gap-3 rounded-md border border-slate-100 px-3 py-2.5 dark:border-zinc-800">
            <span class="grid size-8 shrink-0 place-items-center rounded-md bg-slate-100 text-slate-600 dark:bg-zinc-800 dark:text-zinc-300">
              <UIcon :name="row.icon" class="size-4" />
            </span>
            <span class="min-w-0 flex-1">
              <span class="block truncate text-xs text-slate-500 dark:text-zinc-400">{{ row.label }}</span>
              <span
                class="block truncate text-sm font-bold"
                :class="row.tone || 'text-slate-900 dark:text-white'"
              >{{ row.value }}</span>
            </span>
          </div>
        </div>
      </UCard>

      <UCard class="elegant-card border-slate-200 dark:border-zinc-800 bg-white dark:bg-zinc-900 text-slate-900 dark:text-zinc-100">
        <template #header>
          <h3 class="text-base font-bold text-slate-900 dark:text-white">
            {{ t('admin.home.health.title') }}
          </h3>
        </template>
        <div class="space-y-4">
          <div v-for="row in healthRows" :key="row.label">
            <div class="mb-2 flex items-center justify-between gap-3">
              <div class="min-w-0">
                <p class="truncate text-sm font-semibold text-slate-900 dark:text-white">
                  {{ row.label }}
                </p>
                <p class="truncate text-xs text-slate-500 dark:text-zinc-400">
                  {{ row.sub }}
                </p>
              </div>
              <span class="shrink-0 font-mono text-sm font-bold text-slate-700 dark:text-zinc-200">{{ row.value }}</span>
            </div>
            <div class="h-2 overflow-hidden rounded-full bg-slate-100 dark:bg-zinc-800">
              <span class="block h-full rounded-full bg-[var(--sf-accent)] dark:bg-[var(--sf-accent-dark)]" :style="{ width: `${row.percent}%` }" />
            </div>
          </div>
        </div>
      </UCard>

      <UCard class="elegant-card border-slate-200 dark:border-zinc-800 bg-white dark:bg-zinc-900 text-slate-900 dark:text-zinc-100">
        <template #header>
          <h3 class="text-base font-bold text-slate-900 dark:text-white">
            {{ t('admin.home.actionsTitle') }}
          </h3>
        </template>
        <div class="divide-y divide-slate-100 dark:divide-zinc-800/60">
          <NuxtLink
            v-for="action in overview.actions"
            :key="action.key"
            :to="actionRoute(action)"
            class="flex items-center gap-3 rounded-md px-2 py-3 transition hover:bg-slate-50 dark:hover:bg-zinc-800/40"
          >
            <UBadge :color="actionColor(action)" variant="soft" class="shrink-0 font-mono">
              {{ formatOverviewCount(action.count) }}
            </UBadge>
            <span class="min-w-0 flex-1">
              <span class="block truncate text-sm font-semibold text-slate-900 dark:text-white">
                {{ actionLabel(action) }}
              </span>
              <span class="block truncate text-xs text-slate-500 dark:text-zinc-400">
                {{ actionDescription(action) }}
              </span>
            </span>
            <UIcon name="i-lucide-arrow-right" class="size-4 shrink-0 text-slate-400 dark:text-zinc-500" />
          </NuxtLink>
        </div>
      </UCard>

      <UCard class="elegant-card border-slate-200 dark:border-zinc-800 bg-white dark:bg-zinc-900 text-slate-900 dark:text-zinc-100">
        <template #header>
          <h3 class="text-base font-bold text-slate-900 dark:text-white">
            {{ t('admin.home.topCategories.title') }}
          </h3>
        </template>
        <div v-if="overview.topCategories.length" class="space-y-3">
          <NuxtLink
            v-for="category in overview.topCategories"
            :key="category.id"
            :to="adminRoutes.path('/forum/categories')"
            class="flex items-center justify-between gap-3 rounded-md border border-slate-100 px-3 py-3 transition hover:bg-slate-50 dark:border-zinc-800 dark:hover:bg-zinc-800/40"
          >
            <span class="min-w-0">
              <span class="block truncate text-sm font-semibold text-slate-900 dark:text-white">{{ category.name }}</span>
              <span class="block truncate text-xs text-slate-500 dark:text-zinc-400">{{ category.slug }}</span>
            </span>
            <span class="shrink-0 text-right text-xs text-slate-500 dark:text-zinc-400">
              <strong class="block text-sm text-slate-900 dark:text-white">{{ formatOverviewCount(category.topicCount) }}</strong>
              {{ t('admin.home.topCategories.topicUnit') }}
            </span>
          </NuxtLink>
        </div>
        <SFEmptyState
          v-else
          icon-label="DB"
          :title="t('admin.home.topCategories.emptyTitle')"
          :description="t('admin.home.topCategories.emptyDescription')"
        />
      </UCard>
    </div>

    <UCard
      v-if="extensionWidgets.length"
      class="elegant-card border-slate-200 dark:border-zinc-800 bg-white dark:bg-zinc-900 text-slate-900 dark:text-zinc-100"
    >
      <template #header>
        <div class="flex items-center justify-between gap-3">
          <h3 class="text-base font-bold text-slate-900 dark:text-white">
            {{ t('admin.home.extensionWidgetsTitle') }}
          </h3>
          <UIcon name="i-lucide-puzzle" class="size-5 text-slate-400 dark:text-zinc-500" />
        </div>
      </template>
      <div class="grid gap-3 md:grid-cols-2 xl:grid-cols-3">
        <NuxtLink
          v-for="widget in extensionWidgets"
          :key="`${widget.extensionId}:${widget.id}`"
          :to="extensionWidgetRoute(widget)"
          class="group flex min-w-0 items-center gap-3 rounded-md border border-slate-100 px-3 py-3 transition hover:bg-slate-50 dark:border-zinc-800 dark:hover:bg-zinc-800/40"
        >
          <span class="icon-glass-box shrink-0 text-slate-600 dark:text-zinc-300">
            <UIcon :name="widget.icon || 'i-lucide-blocks'" class="size-5 z-10" />
          </span>
          <span class="min-w-0 flex-1">
            <span class="block truncate text-sm font-semibold text-slate-900 dark:text-white">
              {{ extensionWidgetLabel(widget) }}
            </span>
            <span class="block truncate text-xs text-slate-500 dark:text-zinc-400">
              {{ widget.extensionId }}
            </span>
          </span>
          <UBadge :color="extensionWidgetColor(widget)" variant="soft" class="shrink-0">
            {{ widget.severity }}
          </UBadge>
        </NuxtLink>
      </div>
    </UCard>

    <UCard class="elegant-card border-slate-200 dark:border-zinc-800 bg-white dark:bg-zinc-900 text-slate-900 dark:text-zinc-100">
      <template #header>
        <div class="flex items-center justify-between gap-3">
          <h3 class="text-base font-bold text-slate-900 dark:text-white">
            {{ t('admin.home.quickTitle') }}
          </h3>
          <UIcon name="i-lucide-route" class="size-5 text-slate-400 dark:text-zinc-500" />
        </div>
      </template>
      <div class="grid gap-3 md:grid-cols-2 xl:grid-cols-4">
        <NuxtLink
          v-for="link in quickLinks"
          :key="link.to"
          :to="link.to"
          class="group flex min-w-0 items-center gap-3 rounded-md border border-slate-100 px-3 py-3 transition hover:border-[var(--sf-accent-soft-border)] hover:bg-slate-50 dark:border-zinc-800 dark:hover:border-[rgb(var(--sf-accent-rgb)/0.35)] dark:hover:bg-zinc-800/40"
        >
          <span class="icon-glass-box shrink-0 text-slate-600 dark:text-zinc-300">
            <UIcon :name="link.icon" class="size-5 z-10" />
          </span>
          <span class="min-w-0 flex-1">
            <span class="block truncate text-sm font-semibold text-slate-900 dark:text-white">
              {{ link.title }}
            </span>
            <span class="block truncate text-xs text-slate-500 dark:text-zinc-400">
              {{ link.description }}
            </span>
          </span>
          <UIcon name="i-lucide-arrow-right" class="size-4 shrink-0 text-slate-400 transition group-hover:text-[var(--sf-accent)] dark:text-zinc-500" />
        </NuxtLink>
      </div>
    </UCard>
  </div>
</template>
