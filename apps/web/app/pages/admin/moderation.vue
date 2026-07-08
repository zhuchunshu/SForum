<script setup lang="ts">
import { forumTopicPath } from '~/utils/forumTaxonomy'
import { useAdminPage } from '~/composables/useAdminPage'
import { apiErrorMessage } from '~/composables/useApiClient'
import type { ModerationReport, ModerationReportStatus, ModerationTargetType } from '~/composables/useModerationApi'

definePageMeta({
  middleware: 'admin',
  layout: 'admin'
})

defineOptions({
  name: 'AdminModeration'
})

const { t } = useI18n()
const adminPage = useAdminPage('/moderation')

const moderationApi = useModerationApi()

const statusFilter = ref<ModerationReportStatus | ''>('')
const typeFilter = ref<ModerationTargetType | ''>('')
const currentPage = ref(1)
const ITEMS_PER_PAGE = 20

const filterQuery = computed(() => ({
  status: statusFilter.value || undefined,
  targetType: typeFilter.value || undefined,
  page: currentPage.value,
  perPage: ITEMS_PER_PAGE
}))

const { data: reportList, pending, refresh } = await useAsyncData(
  () => `admin-moderation-${statusFilter.value}-${typeFilter.value}-${currentPage.value}`,
  () => moderationApi.listReports(filterQuery.value),
  {
    default: () => ({ items: [], total: 0, page: 1, perPage: ITEMS_PER_PAGE }),
    watch: [filterQuery]
  }
)

const reports = computed(() => reportList.value.items)
const totalPages = computed(() => Math.ceil(reportList.value.total / Math.max(reportList.value.perPage, 1)) || 1)

const statusOptions: { label: string; value: ModerationReportStatus | '' }[] = [
  { label: t('admin.moderation.statusAll'), value: '' },
  { label: t('admin.moderation.statusOpen'), value: 'open' },
  { label: t('admin.moderation.statusReviewing'), value: 'reviewing' },
  { label: t('admin.moderation.statusResolved'), value: 'resolved' },
  { label: t('admin.moderation.statusRejected'), value: 'rejected' }
]

const typeOptions: { label: string; value: ModerationTargetType | '' }[] = [
  { label: t('admin.moderation.typeAll'), value: '' },
  { label: t('admin.moderation.typeTopic'), value: 'topic' },
  { label: t('admin.moderation.typeComment'), value: 'comment' }
]

function statusVariant(status: ModerationReportStatus) {
  switch (status) {
    case 'open': return 'danger'
    case 'reviewing': return 'warning'
    case 'resolved': return 'success'
    case 'rejected': return 'neutral'
    default: return 'neutral'
  }
}

function formatDate(value: string) {
  const date = new Date(value)
  if (Number.isNaN(date.getTime())) return ''
  return date.toLocaleString()
}

function targetPath(report: ModerationReport) {
  // 审核员快速跳转：仅有 targetId，用 id 模式生成稳定链接，
  // 详情页对 id 形态不重定向（即便站点配置为 id_slug / slug，也会被 301 规范化）。
  if (report.targetType === 'topic') {
    return forumTopicPath({ id: report.targetId, slug: '-' }, 'id')
  }
  return ''
}

// 更新举报状态。
const updatingId = ref<number | null>(null)
const errorMessage = ref('')

async function updateStatus(report: ModerationReport, status: ModerationReportStatus) {
  if (updatingId.value) return
  updatingId.value = report.id
  errorMessage.value = ''
  try {
    await moderationApi.updateReport(report.id, { status, reviewNote: report.reviewNote })
    await refresh()
  } catch (error) {
    errorMessage.value = apiErrorMessage(error) || t('admin.moderation.updateFailed')
  } finally {
    updatingId.value = null
  }
}

async function saveReviewNote(report: ModerationReport) {
  if (updatingId.value) return
  updatingId.value = report.id
  errorMessage.value = ''
  try {
    await moderationApi.updateReport(report.id, { status: report.status, reviewNote: report.reviewNote })
    await refresh()
  } catch (error) {
    errorMessage.value = apiErrorMessage(error) || t('admin.moderation.updateFailed')
  } finally {
    updatingId.value = null
  }
}

watch([statusFilter, typeFilter], () => {
  if (currentPage.value !== 1) {
    currentPage.value = 1
  }
})
</script>

<template>
  <div class="space-y-4">
    <header>
      <h1 class="text-xl font-bold text-slate-900 dark:text-zinc-50">
        <UIcon :name="adminPage.icon" class="mr-2 inline-block size-5 text-[var(--sf-accent)] dark:text-[var(--sf-accent-dark)]" />
        {{ t('admin.moderation.title') }}
      </h1>
      <p class="text-sm text-slate-500 mt-1 dark:text-zinc-400">
        {{ t('admin.moderation.description') }}
      </p>
    </header>

    <UDashboardToolbar class="border border-slate-200 dark:border-zinc-800 bg-white dark:bg-zinc-900 rounded-lg px-4 py-2.5 text-slate-500 dark:text-zinc-400">
      <template #left>
        <div class="flex min-w-0 items-center gap-2 text-sm">
          <UIcon name="i-lucide-shield-alert" class="size-4" />
          <span class="truncate">{{ t('admin.moderation.description') }}</span>
        </div>
      </template>
      <template #right>
        <UButton icon="i-lucide-rotate-cw" color="neutral" variant="subtle" :loading="pending" @click="refresh()">
          {{ t('admin.home.refresh') }}
        </UButton>
      </template>
    </UDashboardToolbar>

    <SFAlert v-if="errorMessage" variant="danger" :title="errorMessage" closable @close="errorMessage = ''" />

    <!-- 过滤器 -->
    <SFCard class="p-4 flex flex-wrap gap-4 items-end">
      <div>
        <label class="block text-xs font-semibold text-slate-600 mb-1 dark:text-zinc-400">
          {{ t('admin.moderation.filterStatus') }}
        </label>
        <select v-model="statusFilter" class="sf-input">
          <option v-for="opt in statusOptions" :key="opt.value" :value="opt.value">{{ opt.label }}</option>
        </select>
      </div>
      <div>
        <label class="block text-xs font-semibold text-slate-600 mb-1 dark:text-zinc-400">
          {{ t('admin.moderation.filterType') }}
        </label>
        <select v-model="typeFilter" class="sf-input">
          <option v-for="opt in typeOptions" :key="opt.value" :value="opt.value">{{ opt.label }}</option>
        </select>
      </div>
    </SFCard>

    <!-- 列表 -->
    <SFCard v-if="pending" class="p-6">
      <SFSkeleton width="60%" class="mb-2" />
      <SFSkeleton width="90%" />
    </SFCard>

    <template v-else-if="reports.length">
      <SFCard v-for="report in reports" :key="report.id" class="p-4 space-y-3">
        <div class="flex items-center justify-between">
          <div class="flex items-center gap-2">
            <SFBadge :variant="statusVariant(report.status)">{{ t(`admin.moderation.status.${report.status}`) }}</SFBadge>
            <SFBadge variant="neutral">{{ t(`admin.moderation.type.${report.targetType}`) }} #{{ report.targetId }}</SFBadge>
            <SFBadge variant="info">{{ t(`admin.moderation.reason.${report.reasonCode}`) }}</SFBadge>
          </div>
          <NuxtLink
            v-if="targetPath(report)"
            :to="useLocalePath()(targetPath(report))"
            class="text-sm text-[#0F766E] hover:underline dark:text-teal-300"
          >
            {{ t('admin.moderation.viewTarget') }}
          </NuxtLink>
        </div>
        <p v-if="report.body" class="text-sm text-slate-700 dark:text-zinc-300">
          {{ report.body }}
        </p>
        <p class="text-xs text-slate-400 dark:text-zinc-500">
          {{ report.reporterName || `#${report.reporterUserId}` }} · {{ formatDate(report.createdAt) }}
        </p>
        <!-- 审核备注 -->
        <div class="flex gap-2">
          <input
            v-model="report.reviewNote"
            type="text"
            class="sf-input flex-1"
            :placeholder="t('admin.moderation.reviewNotePlaceholder')"
          >
          <SFButton variant="ghost" size="sm" :disabled="updatingId === report.id" @click="saveReviewNote(report)">
            {{ t('admin.moderation.saveNote') }}
          </SFButton>
        </div>
        <!-- 快速操作 -->
        <div class="flex gap-2">
          <SFButton variant="ghost" size="sm" :disabled="updatingId === report.id" @click="updateStatus(report, 'reviewing')">
            {{ t('admin.moderation.markReviewing') }}
          </SFButton>
          <SFButton variant="ghost" size="sm" :disabled="updatingId === report.id" @click="updateStatus(report, 'resolved')">
            {{ t('admin.moderation.markResolved') }}
          </SFButton>
          <SFButton variant="ghost" size="sm" :disabled="updatingId === report.id" @click="updateStatus(report, 'rejected')">
            {{ t('admin.moderation.markRejected') }}
          </SFButton>
        </div>
      </SFCard>

      <div v-if="totalPages > 1" class="flex justify-center pt-2">
        <SFPagination v-model:page="currentPage" :total-pages="totalPages" />
      </div>
    </template>

    <SFCard v-else class="p-10">
      <SFEmptyState
        :title="t('admin.moderation.empty.title')"
        :description="t('admin.moderation.empty.description')"
      />
    </SFCard>
  </div>
</template>

<style scoped>
.sf-input {
  border: 1px solid #d1d5db;
  border-radius: 0.5rem;
  padding: 0.4rem 0.6rem;
  font-size: 0.9rem;
  background: #ffffff;
  color: #111827;
  outline: none;
  transition: border-color 0.15s;
}
.sf-input:focus {
  border-color: #0f766e;
  box-shadow: 0 0 0 3px rgba(15, 118, 110, 0.12);
}
:global(.dark) .sf-input {
  background: #18181b;
  border-color: #3f3f46;
  color: #f4f4f5;
}
</style>
